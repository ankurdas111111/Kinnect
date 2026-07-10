package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"kinnect-v3/internal/shared"
)

// ─── Shared types ─────────────────────────────────────────────────────────────

// SearchResult is a single normalized place entry returned by /api/search.
type SearchResult struct {
	Name string  `json:"name"`
	Sub  string  `json:"sub"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	Type string  `json:"type"`
	// DistanceM from the bias point (user's location); 0 when no bias given.
	DistanceM float64 `json:"distanceM,omitempty"`
}

// geoJSONLineString is a minimal GeoJSON LineString geometry.
type geoJSONLineString struct {
	Type        string       `json:"type"`
	Coordinates [][2]float64 `json:"coordinates"`
}

// routeStep is one navigation step in a route response.
type routeStep struct {
	Instruction string  `json:"instruction"`
	DistanceM   float64 `json:"distanceM"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

// RouteResult is the normalized routing response returned by /api/route.
type RouteResult struct {
	DistanceM float64           `json:"distanceM"`
	DurationS float64           `json:"durationS"`
	Geometry  geoJSONLineString `json:"geometry"`
	Steps     []routeStep       `json:"steps"`
}

// ─── SearchHandler — GET /api/search ──────────────────────────────────────────

const (
	searchCacheTTL = 15 * time.Minute
	searchCacheMax = 1000
	searchTimeout  = 5 * time.Second
)

type searchCacheEntry struct {
	results   []SearchResult
	createdAt time.Time
}

// SearchHandler proxies place-search to Ola Maps (when configured) with a
// Photon fallback. Results are LRU-cached for 15 minutes.
type SearchHandler struct {
	mu         sync.Mutex
	cache      map[string]*searchCacheEntry
	client     *http.Client
	olaMapsKey string
}

// NewSearchHandler creates a SearchHandler with an optional Ola Maps API key.
func NewSearchHandler(olaMapsKey string) *SearchHandler {
	return &SearchHandler{
		cache:      make(map[string]*searchCacheEntry),
		client:     &http.Client{Timeout: searchTimeout},
		olaMapsKey: olaMapsKey,
	}
}

// Search handles GET /api/search?q=<query>[&lat=<f>&lng=<f>].
// Returns {"results":[...]} always; empty array when no results.
// Rejects queries shorter than 2 characters (returns empty immediately).
func (s *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		writeJSON(w, http.StatusOK, map[string][]SearchResult{"results": {}})
		return
	}

	// Optional bias coordinates
	var hasBias bool
	var biasLat, biasLng float64
	if ls, lo := r.URL.Query().Get("lat"), r.URL.Query().Get("lng"); ls != "" && lo != "" {
		var e1, e2 error
		biasLat, e1 = strconv.ParseFloat(ls, 64)
		biasLng, e2 = strconv.ParseFloat(lo, 64)
		hasBias = e1 == nil && e2 == nil
	}

	// Cache key: lowercase query + bias rounded to 2 decimal places (~1.1 km)
	cacheKey := strings.ToLower(q)
	if hasBias {
		cacheKey += fmt.Sprintf(",%.2f,%.2f",
			math.Round(biasLat*100)/100,
			math.Round(biasLng*100)/100)
	}

	s.mu.Lock()
	if e, ok := s.cache[cacheKey]; ok && time.Since(e.createdAt) < searchCacheTTL {
		results := e.results
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string][]SearchResult{"results": results})
		return
	}
	s.mu.Unlock()

	var results []SearchResult
	var providerErr error

	// Primary: Ola Maps (when API key is configured)
	if s.olaMapsKey != "" {
		results, providerErr = s.searchOla(r, q, hasBias, biasLat, biasLng)
		if providerErr != nil {
			slog.Warn("ola maps search failed, falling back to photon", "err", providerErr)
			results = nil // ensure fallback triggers
		}
	}

	// Fallback: Photon (komoot) — used when no key or Ola error/timeout
	if results == nil {
		results, providerErr = s.searchPhoton(r, q, hasBias, biasLat, biasLng)
		if providerErr != nil {
			http.Error(w, `{"error":"search service unavailable"}`, http.StatusBadGateway)
			return
		}
	}

	if results == nil {
		results = []SearchResult{}
	}

	// Proximity ranking: when we know where the user is, annotate every result
	// with its distance and order same-name results nearest-first (the common
	// "which Domino's?" case). Provider text-relevance order is preserved
	// between distinct names: groups keep the position of their first
	// occurrence; only members within a name-group are re-ordered.
	if hasBias {
		for i := range results {
			results[i].DistanceM = math.Round(shared.HaversineM(biasLat, biasLng, results[i].Lat, results[i].Lng))
		}
		groupOrder := make(map[string]int)
		for i, res := range results {
			key := strings.ToLower(res.Name)
			if _, seen := groupOrder[key]; !seen {
				groupOrder[key] = i
			}
		}
		sort.SliceStable(results, func(a, b int) bool {
			ka, kb := strings.ToLower(results[a].Name), strings.ToLower(results[b].Name)
			if ka == kb {
				return results[a].DistanceM < results[b].DistanceM
			}
			return groupOrder[ka] < groupOrder[kb]
		})
	}

	s.mu.Lock()
	if len(s.cache) >= searchCacheMax {
		s.evictOldest()
	}
	s.cache[cacheKey] = &searchCacheEntry{results: results, createdAt: time.Now()}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string][]SearchResult{"results": results})
}

// searchOla calls Ola Maps Autocomplete and normalizes to []SearchResult.
// Predictions missing geometry.location are dropped (no Place Details call).
func (s *SearchHandler) searchOla(r *http.Request, q string, hasBias bool, biasLat, biasLng float64) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("input", q)
	params.Set("api_key", s.olaMapsKey)
	if hasBias {
		params.Set("location", fmt.Sprintf("%.6f,%.6f", biasLat, biasLng))
	}
	reqURL := "https://api.olamaps.io/places/v1/autocomplete?" + params.Encode()

	req, err := http.NewRequestWithContext(r.Context(), "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", nominatimUA)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if err != nil {
		return nil, fmt.Errorf("ola read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ola autocomplete http %d", resp.StatusCode)
	}

	// Defensive struct: geometry is a pointer so we can detect its absence.
	var olaResp struct {
		Status      string `json:"status"`
		Predictions []struct {
			Description          string `json:"description"`
			StructuredFormatting struct {
				MainText      string `json:"main_text"`
				SecondaryText string `json:"secondary_text"`
			} `json:"structured_formatting"`
			Geometry *struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
			Types []string `json:"types"`
		} `json:"predictions"`
	}
	if err := json.Unmarshal(body, &olaResp); err != nil {
		return nil, fmt.Errorf("ola parse: %w", err)
	}
	switch olaResp.Status {
	case "OK", "ZERO_RESULTS", "": // empty means field was absent — treat as OK
	default:
		return nil, fmt.Errorf("ola status %q", olaResp.Status)
	}

	results := make([]SearchResult, 0, len(olaResp.Predictions))
	for _, p := range olaResp.Predictions {
		// Drop predictions without coordinates — calling Place Details is non-trivial.
		if p.Geometry == nil {
			continue
		}
		lat := p.Geometry.Location.Lat
		lng := p.Geometry.Location.Lng
		if lat == 0 && lng == 0 {
			continue
		}

		name := p.StructuredFormatting.MainText
		if name == "" {
			// Derive from description: text before the first comma.
			if idx := strings.IndexByte(p.Description, ','); idx > 0 {
				name = strings.TrimSpace(p.Description[:idx])
			} else {
				name = p.Description
			}
		}
		sub := p.StructuredFormatting.SecondaryText
		if sub == "" {
			if idx := strings.IndexByte(p.Description, ','); idx >= 0 {
				sub = strings.TrimSpace(p.Description[idx+1:])
			}
		}
		typ := ""
		if len(p.Types) > 0 {
			typ = p.Types[0]
		}

		results = append(results, SearchResult{
			Name: name,
			Sub:  sub,
			Lat:  lat,
			Lng:  lng,
			Type: typ,
		})
	}
	return results, nil
}

// searchPhoton calls Photon (komoot) and normalizes GeoJSON features.
func (s *SearchHandler) searchPhoton(r *http.Request, q string, hasBias bool, biasLat, biasLng float64) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("q", q)
	params.Set("limit", "6")
	params.Set("lang", "en")
	if hasBias {
		params.Set("lat", fmt.Sprintf("%.6f", biasLat))
		params.Set("lon", fmt.Sprintf("%.6f", biasLng))
	}
	reqURL := "https://photon.komoot.io/api?" + params.Encode()

	req, err := http.NewRequestWithContext(r.Context(), "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", nominatimUA)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if err != nil {
		return nil, fmt.Errorf("photon read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("photon http %d", resp.StatusCode)
	}

	// Photon returns GeoJSON FeatureCollection; coordinates are [lng, lat].
	var photonResp struct {
		Features []struct {
			Geometry struct {
				Coordinates [2]float64 `json:"coordinates"` // [lng, lat]
			} `json:"geometry"`
			Properties struct {
				Name    string `json:"name"`
				Street  string `json:"street"`
				City    string `json:"city"`
				State   string `json:"state"`
				Country string `json:"country"`
				Type    string `json:"type"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(body, &photonResp); err != nil {
		return nil, fmt.Errorf("photon parse: %w", err)
	}

	results := make([]SearchResult, 0, len(photonResp.Features))
	for _, f := range photonResp.Features {
		lng := f.Geometry.Coordinates[0] // GeoJSON order
		lat := f.Geometry.Coordinates[1]
		if lat == 0 && lng == 0 {
			continue
		}

		name := f.Properties.Name
		if name == "" {
			name = f.Properties.Street
		}
		if name == "" {
			continue // no usable label
		}

		// Build sub from street, city, state — omit whichever equal name.
		subParts := make([]string, 0, 3)
		if f.Properties.Street != "" && f.Properties.Street != name {
			subParts = append(subParts, f.Properties.Street)
		}
		if f.Properties.City != "" {
			subParts = append(subParts, f.Properties.City)
		}
		if f.Properties.State != "" {
			subParts = append(subParts, f.Properties.State)
		}

		results = append(results, SearchResult{
			Name: name,
			Sub:  strings.Join(subParts, ", "),
			Lat:  lat,
			Lng:  lng,
			Type: f.Properties.Type,
		})
	}
	return results, nil
}

func (s *SearchHandler) evictOldest() {
	oldest := ""
	var oldestTime time.Time
	for k, v := range s.cache {
		if oldest == "" || v.createdAt.Before(oldestTime) {
			oldest = k
			oldestTime = v.createdAt
		}
	}
	if oldest != "" {
		delete(s.cache, oldest)
	}
}

// ─── RouteHandler — GET /api/route ────────────────────────────────────────────

const (
	routeCacheTTL = 5 * time.Minute
	routeCacheMax = 500
	routeTimeout  = 10 * time.Second // routing upstream can be slower
)

type routeCacheEntry struct {
	result    *RouteResult
	createdAt time.Time
}

// RouteHandler proxies routing requests to Ola Maps Directions (car/scooter
// with API key) or FOSSGIS OSRM (all other cases). Results are LRU-cached for
// 5 minutes. OSRM access is rate-limited to 1 req/s.
type RouteHandler struct {
	mu         sync.Mutex
	cache      map[string]*routeCacheEntry
	client     *http.Client
	olaMapsKey string
	lastOSRM   time.Time // guards the 1 req/s limit to routing.openstreetmap.de
}

// NewRouteHandler creates a RouteHandler with an optional Ola Maps API key.
func NewRouteHandler(olaMapsKey string) *RouteHandler {
	return &RouteHandler{
		cache:      make(map[string]*routeCacheEntry),
		client:     &http.Client{Timeout: routeTimeout},
		olaMapsKey: olaMapsKey,
	}
}

// validModes is the whitelist of accepted mode values.
var validModes = map[string]bool{
	"car": true, "foot": true, "bike": true, "scooter": true,
}

// Route handles GET /api/route?mode=<m>&from=<lat,lng>&to=<lat,lng>.
func (rh *RouteHandler) Route(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if !validModes[mode] {
		http.Error(w, `{"error":"mode must be one of: car, foot, bike, scooter"}`, http.StatusBadRequest)
		return
	}

	fromLat, fromLng, err := parseLatLng(r.URL.Query().Get("from"))
	if err != nil {
		http.Error(w, `{"error":"invalid from parameter (expected lat,lng)"}`, http.StatusBadRequest)
		return
	}
	toLat, toLng, err := parseLatLng(r.URL.Query().Get("to"))
	if err != nil {
		http.Error(w, `{"error":"invalid to parameter (expected lat,lng)"}`, http.StatusBadRequest)
		return
	}

	// Cache key: mode + coords rounded to 4 decimal places (~11 m precision)
	cacheKey := fmt.Sprintf("%s,%.4f,%.4f,%.4f,%.4f", mode,
		math.Round(fromLat*10000)/10000, math.Round(fromLng*10000)/10000,
		math.Round(toLat*10000)/10000, math.Round(toLng*10000)/10000)

	rh.mu.Lock()
	if e, ok := rh.cache[cacheKey]; ok && time.Since(e.createdAt) < routeCacheTTL {
		result := e.result
		rh.mu.Unlock()
		writeJSON(w, http.StatusOK, result)
		return
	}
	rh.mu.Unlock()

	var result *RouteResult

	// Use Ola Maps Directions for car/scooter when API key is configured.
	if rh.olaMapsKey != "" && (mode == "car" || mode == "scooter") {
		result, err = rh.routeOla(r, fromLat, fromLng, toLat, toLng)
		if err != nil {
			slog.Warn("ola maps routing failed, falling back to osrm", "err", err)
			result = nil
		}
	}

	// OSRM fallback (always for foot/bike; fallback for car/scooter on Ola failure).
	if result == nil {
		result, err = rh.routeOSRM(r, mode, fromLat, fromLng, toLat, toLng)
		if err != nil {
			http.Error(w, `{"error":"routing service unavailable"}`, http.StatusBadGateway)
			return
		}
	}

	rh.mu.Lock()
	if len(rh.cache) >= routeCacheMax {
		rh.evictOldestRoute()
	}
	rh.cache[cacheKey] = &routeCacheEntry{result: result, createdAt: time.Now()}
	rh.mu.Unlock()

	writeJSON(w, http.StatusOK, result)
}

// routeOSRM calls the FOSSGIS OSRM instance and normalizes the response.
// Rate-limited to 1 req/s (mirrors geocode.go's Nominatim limiter pattern).
func (rh *RouteHandler) routeOSRM(r *http.Request, mode string, fromLat, fromLng, toLat, toLng float64) (*RouteResult, error) {
	profile := "car"
	switch mode {
	case "foot":
		profile = "foot"
	case "bike":
		profile = "bike"
	// "car" and "scooter" both map to routed-car
	}

	// 1 req/s rate limit to routing.openstreetmap.de (same pattern as geocode.go)
	rh.mu.Lock()
	if elapsed := time.Since(rh.lastOSRM); elapsed < time.Second {
		rh.mu.Unlock()
		time.Sleep(time.Second - elapsed)
		rh.mu.Lock()
	}
	rh.lastOSRM = time.Now()
	rh.mu.Unlock()

	// OSRM coordinate order is lng,lat (GeoJSON convention).
	osrmURL := fmt.Sprintf(
		"https://routing.openstreetmap.de/routed-%s/route/v1/driving/%.6f,%.6f;%.6f,%.6f?overview=full&geometries=geojson&steps=true",
		profile,
		fromLng, fromLat,
		toLng, toLat,
	)

	req, err := http.NewRequestWithContext(r.Context(), "GET", osrmURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", nominatimUA)

	resp, err := rh.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("osrm read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osrm http %d", resp.StatusCode)
	}

	var osrmResp struct {
		Code   string `json:"code"`
		Routes []struct {
			Distance float64 `json:"distance"`
			Duration float64 `json:"duration"`
			Geometry struct {
				Type        string       `json:"type"`
				Coordinates [][2]float64 `json:"coordinates"`
			} `json:"geometry"`
			Legs []struct {
				Steps []struct {
					Distance float64 `json:"distance"`
					Name     string  `json:"name"`
					Maneuver struct {
						Type     string     `json:"type"`
						Modifier string     `json:"modifier"`
						Location [2]float64 `json:"location"` // [lng, lat]
					} `json:"maneuver"`
				} `json:"steps"`
			} `json:"legs"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(body, &osrmResp); err != nil {
		return nil, fmt.Errorf("osrm parse: %w", err)
	}
	if osrmResp.Code != "Ok" || len(osrmResp.Routes) == 0 {
		return nil, fmt.Errorf("osrm no route (code=%s)", osrmResp.Code)
	}

	route := osrmResp.Routes[0]
	steps := make([]routeStep, 0)
	if len(route.Legs) > 0 {
		for _, st := range route.Legs[0].Steps {
			steps = append(steps, routeStep{
				Instruction: osrmInstruction(st.Maneuver.Type, st.Maneuver.Modifier, st.Name),
				DistanceM:   st.Distance,
				Lat:         st.Maneuver.Location[1], // location is [lng, lat]
				Lng:         st.Maneuver.Location[0],
			})
		}
	}

	return &RouteResult{
		DistanceM: route.Distance,
		DurationS: route.Duration,
		Geometry: geoJSONLineString{
			Type:        "LineString",
			Coordinates: route.Geometry.Coordinates,
		},
		Steps: steps,
	}, nil
}

// routeOla calls Ola Maps Directions API and normalizes the response.
// Falls back to OSRM silently on error (caller handles the fallback).
// Verified live response shape (2026-07): POST with query params; status
// "SUCCESS"; overview_polyline is a plain encoded string; legs[0].distance
// and .duration are bare numbers (metres/seconds); steps use "instructions".
func (rh *RouteHandler) routeOla(r *http.Request, fromLat, fromLng, toLat, toLng float64) (*RouteResult, error) {
	params := url.Values{}
	params.Set("origin", fmt.Sprintf("%.6f,%.6f", fromLat, fromLng))
	params.Set("destination", fmt.Sprintf("%.6f,%.6f", toLat, toLng))
	params.Set("mode", "driving")
	params.Set("api_key", rh.olaMapsKey)
	reqURL := "https://api.olamaps.io/routing/v1/directions?" + params.Encode()

	req, err := http.NewRequestWithContext(r.Context(), "POST", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", nominatimUA)

	resp, err := rh.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("ola directions read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ola directions http %d", resp.StatusCode)
	}

	var olaResp struct {
		Status string `json:"status"`
		Routes []struct {
			OverviewPolyline string `json:"overview_polyline"`
			Legs             []struct {
				Distance float64 `json:"distance"` // metres
				Duration float64 `json:"duration"` // seconds
				Steps    []struct {
					Instructions  string  `json:"instructions"`
					Distance      float64 `json:"distance"`
					StartLocation struct {
						Lat float64 `json:"lat"`
						Lng float64 `json:"lng"`
					} `json:"start_location"`
				} `json:"steps"`
			} `json:"legs"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(body, &olaResp); err != nil {
		return nil, fmt.Errorf("ola directions parse: %w", err)
	}
	switch olaResp.Status {
	case "SUCCESS", "OK", "": // Ola uses "SUCCESS"; empty means field absent
	default:
		return nil, fmt.Errorf("ola directions status %q", olaResp.Status)
	}
	if len(olaResp.Routes) == 0 || len(olaResp.Routes[0].Legs) == 0 {
		return nil, fmt.Errorf("ola directions: no routes in response")
	}

	route := olaResp.Routes[0]
	leg := route.Legs[0]

	// Decode the overview polyline into GeoJSON coordinates ([lng, lat] pairs).
	coords := decodePolyline5(route.OverviewPolyline)

	steps := make([]routeStep, 0, len(leg.Steps))
	for _, st := range leg.Steps {
		steps = append(steps, routeStep{
			Instruction: stripHTML(st.Instructions),
			DistanceM:   st.Distance,
			Lat:         st.StartLocation.Lat,
			Lng:         st.StartLocation.Lng,
		})
	}

	return &RouteResult{
		DistanceM: leg.Distance,
		DurationS: leg.Duration,
		Geometry: geoJSONLineString{
			Type:        "LineString",
			Coordinates: coords,
		},
		Steps: steps,
	}, nil
}

func (rh *RouteHandler) evictOldestRoute() {
	oldest := ""
	var oldestTime time.Time
	for k, v := range rh.cache {
		if oldest == "" || v.createdAt.Before(oldestTime) {
			oldest = k
			oldestTime = v.createdAt
		}
	}
	if oldest != "" {
		delete(rh.cache, oldest)
	}
}

// ─── Pure helpers ─────────────────────────────────────────────────────────────

// decodePolyline5 decodes a Google Encoded Polyline Algorithm Format string
// (precision 5) into GeoJSON coordinate pairs ([lng, lat] order).
// Implemented in ~30 lines with no external dependencies.
func decodePolyline5(encoded string) [][2]float64 {
	coords := make([][2]float64, 0, len(encoded)/5)
	lat, lng := 0, 0
	i, n := 0, len(encoded)

	for i < n {
		// Decode latitude delta
		result, shift := 0, 0
		for i < n {
			b := int(encoded[i]) - 63
			i++
			result |= (b & 0x1f) << shift
			shift += 5
			if b < 0x20 { // high bit clear → last chunk
				break
			}
		}
		if result&1 != 0 {
			lat += ^(result >> 1)
		} else {
			lat += result >> 1
		}

		// Decode longitude delta
		result, shift = 0, 0
		for i < n {
			b := int(encoded[i]) - 63
			i++
			result |= (b & 0x1f) << shift
			shift += 5
			if b < 0x20 {
				break
			}
		}
		if result&1 != 0 {
			lng += ^(result >> 1)
		} else {
			lng += result >> 1
		}

		// GeoJSON order: coordinates are [lng, lat]
		coords = append(coords, [2]float64{float64(lng) / 1e5, float64(lat) / 1e5})
	}
	return coords
}

// stripHTML removes HTML tags from a string (used for Ola step instructions).
func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, c := range s {
		switch {
		case c == '<':
			inTag = true
		case c == '>':
			inTag = false
		case !inTag:
			b.WriteRune(c)
		}
	}
	return strings.TrimSpace(b.String())
}

// osrmInstruction synthesizes a human-readable instruction from OSRM maneuver
// fields. OSRM provides type + modifier + street name but no instruction string.
func osrmInstruction(mType, mMod, street string) string {
	switch mType {
	case "depart":
		if street != "" {
			return "Head onto " + street
		}
		return "Depart"
	case "arrive":
		return "Arrive at destination"
	case "continue", "new name":
		if street != "" {
			return "Continue onto " + street
		}
		return "Continue straight"
	case "merge":
		if street != "" {
			return "Merge onto " + street
		}
		return "Merge"
	case "on ramp":
		if street != "" {
			return "Take the ramp onto " + street
		}
		return "Take the on-ramp"
	case "off ramp":
		if street != "" {
			return "Take the exit onto " + street
		}
		return "Take the off-ramp"
	case "fork":
		d := dirPhrase(mMod)
		if street != "" {
			return "Keep " + strings.ToLower(d) + " at fork onto " + street
		}
		return "Keep " + strings.ToLower(d) + " at fork"
	case "end of road":
		return dirPhrase(mMod) + " at end of road"
	case "roundabout", "rotary":
		return "Enter the roundabout"
	case "exit roundabout", "exit rotary":
		if street != "" {
			return "Exit roundabout onto " + street
		}
		return "Exit roundabout"
	default: // "turn" and unknowns
		d := dirPhrase(mMod)
		if street != "" {
			return d + " onto " + street
		}
		return d
	}
}

// dirPhrase converts an OSRM maneuver modifier to a human-readable direction.
func dirPhrase(mod string) string {
	switch mod {
	case "left":
		return "Turn left"
	case "right":
		return "Turn right"
	case "sharp left":
		return "Turn sharp left"
	case "sharp right":
		return "Turn sharp right"
	case "slight left":
		return "Bear left"
	case "slight right":
		return "Bear right"
	case "uturn":
		return "Make a U-turn"
	case "straight", "":
		return "Continue straight"
	default:
		return strings.ToUpper(mod[:1]) + mod[1:]
	}
}

// parseLatLng parses a "lat,lng" string into two float64 values.
func parseLatLng(s string) (lat, lng float64, err error) {
	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected lat,lng pair, got %q", s)
	}
	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lat: %w", err)
	}
	lng, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lng: %w", err)
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, fmt.Errorf("coordinates out of range (lat=%.6f lng=%.6f)", lat, lng)
	}
	return lat, lng, nil
}
