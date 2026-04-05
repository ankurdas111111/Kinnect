package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// GeoHandler handles reverse geocoding requests, proxying to Nominatim
// with an in-memory LRU cache and 1-request-per-second rate limiting.
type GeoHandler struct {
	mu        sync.Mutex
	cache     map[string]*geoCacheEntry
	lastReq   time.Time
	client    *http.Client
}

type geoCacheEntry struct {
	result    *GeoResult
	createdAt time.Time
}

// GeoResult is the JSON shape returned to the frontend.
type GeoResult struct {
	DisplayName string `json:"displayName"`
	Road        string `json:"road,omitempty"`
	Suburb      string `json:"suburb,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	Country     string `json:"country,omitempty"`
}

const (
	geoCacheTTL     = 24 * time.Hour
	geoCacheMaxSize = 2000
	nominatimUA     = "Kinnect/1.0 (family-tracker; contact ankur@kinnect.app)"
)

// NewGeoHandler creates a new geocode handler.
func NewGeoHandler() *GeoHandler {
	return &GeoHandler{
		cache:  make(map[string]*geoCacheEntry),
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// ReverseGeocode handles GET /api/geocode?lat=X&lng=Y
func (g *GeoHandler) ReverseGeocode(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")
	if latStr == "" || lngStr == "" {
		http.Error(w, `{"error":"lat and lng required"}`, http.StatusBadRequest)
		return
	}
	lat, err1 := strconv.ParseFloat(latStr, 64)
	lng, err2 := strconv.ParseFloat(lngStr, 64)
	if err1 != nil || err2 != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		http.Error(w, `{"error":"invalid coordinates"}`, http.StatusBadRequest)
		return
	}

	// Round to 4 decimal places (~11m) for cache key
	cacheKey := fmt.Sprintf("%.4f,%.4f", math.Round(lat*10000)/10000, math.Round(lng*10000)/10000)

	g.mu.Lock()
	if entry, ok := g.cache[cacheKey]; ok && time.Since(entry.createdAt) < geoCacheTTL {
		g.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		json.NewEncoder(w).Encode(entry.result)
		return
	}
	g.mu.Unlock()

	// Rate limit: 1 request per second to Nominatim
	g.mu.Lock()
	elapsed := time.Since(g.lastReq)
	if elapsed < time.Second {
		g.mu.Unlock()
		time.Sleep(time.Second - elapsed)
		g.mu.Lock()
	}
	g.lastReq = time.Now()
	g.mu.Unlock()

	// Call Nominatim
	url := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/reverse?format=json&lat=%.6f&lon=%.6f&zoom=17&addressdetails=1&accept-language=en",
		lat, lng,
	)
	req, _ := http.NewRequestWithContext(r.Context(), "GET", url, nil)
	req.Header.Set("User-Agent", nominatimUA)

	resp, err := g.client.Do(req)
	if err != nil {
		http.Error(w, `{"error":"geocoding service unavailable"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, `{"error":"geocoding failed"}`, http.StatusBadGateway)
		return
	}

	// Parse Nominatim response
	var nomResp struct {
		DisplayName string `json:"display_name"`
		Address     struct {
			Road         string `json:"road"`
			Suburb       string `json:"suburb"`
			Neighbourhood string `json:"neighbourhood"`
			City         string `json:"city"`
			Town         string `json:"town"`
			Village      string `json:"village"`
			State        string `json:"state"`
			Country      string `json:"country"`
		} `json:"address"`
	}
	if err := json.Unmarshal(body, &nomResp); err != nil {
		http.Error(w, `{"error":"failed to parse response"}`, http.StatusBadGateway)
		return
	}

	// Build simplified result
	city := nomResp.Address.City
	if city == "" {
		city = nomResp.Address.Town
	}
	if city == "" {
		city = nomResp.Address.Village
	}
	suburb := nomResp.Address.Suburb
	if suburb == "" {
		suburb = nomResp.Address.Neighbourhood
	}

	result := &GeoResult{
		DisplayName: nomResp.DisplayName,
		Road:        nomResp.Address.Road,
		Suburb:      suburb,
		City:        city,
		State:       nomResp.Address.State,
		Country:     nomResp.Address.Country,
	}

	// Cache the result
	g.mu.Lock()
	// Evict old entries if cache is full
	if len(g.cache) >= geoCacheMaxSize {
		oldest := ""
		var oldestTime time.Time
		for k, v := range g.cache {
			if oldest == "" || v.createdAt.Before(oldestTime) {
				oldest = k
				oldestTime = v.createdAt
			}
		}
		if oldest != "" {
			delete(g.cache, oldest)
		}
	}
	g.cache[cacheKey] = &geoCacheEntry{result: result, createdAt: time.Now()}
	g.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	json.NewEncoder(w).Encode(result)
}
