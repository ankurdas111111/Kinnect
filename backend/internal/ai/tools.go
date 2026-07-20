package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"kinnect-v3/internal/shared"
)

// ─── Tool context: the authorization boundary ────────────────────────────────

// ToolContext carries the requester's identity and pre-resolved visibility.
// Every tool is scoped by Visible — computed from the session server-side.
// The model can neither widen it nor query users outside it.
type ToolContext struct {
	RequesterID   string
	RequesterName string
	// Visible maps userID → displayName for everyone the requester may see
	// (self, room members, contacts, active guardians/wards) AFTER sharing-
	// schedule filtering. A schedule-blocked target is absent entirely.
	Visible map[string]string
	// Coarsen holds target userIDs whose coordinates must be rounded to ~1.1 km
	// because the target is in quiet hours and the requester is not their
	// guardian — mirroring the live broadcast path (ws.flushPositionBroadcasts).
	Coarsen map[string]bool
	DB      *sql.DB
	Loc     *time.Location
}

// coord returns the coordinate to expose for a target, coarsening it to ~1.1 km
// when the target is under quiet-hours privacy for this requester.
func (tc *ToolContext) coord(userID string, lat, lng float64) (float64, float64) {
	if tc.Coarsen[userID] {
		return math.Round(lat*100) / 100, math.Round(lng*100) / 100
	}
	return lat, lng
}

// coarsened reports whether the target's location is under quiet-hours privacy.
func (tc *ToolContext) coarsened(userID string) bool { return tc.Coarsen[userID] }

// resolveUser matches a model-supplied name or ID against the visible set.
// Case-insensitive; accepts exact ID, exact name, or unique name prefix.
func (tc *ToolContext) resolveUser(ref string) (string, string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", fmt.Errorf("user reference is empty")
	}
	if name, ok := tc.Visible[ref]; ok {
		return ref, name, nil
	}
	lower := strings.ToLower(ref)
	if lower == "me" || lower == "self" || lower == strings.ToLower(tc.RequesterName) {
		return tc.RequesterID, tc.RequesterName, nil
	}
	var matchID, matchName string
	matches := 0
	for id, name := range tc.Visible {
		ln := strings.ToLower(name)
		if ln == lower {
			return id, name, nil // exact name match wins immediately
		}
		if strings.HasPrefix(ln, lower) {
			matchID, matchName = id, name
			matches++
		}
	}
	switch matches {
	case 1:
		return matchID, matchName, nil
	case 0:
		return "", "", fmt.Errorf("no visible family member matches %q — you can only query members of this family", ref)
	default:
		return "", "", fmt.Errorf("%q is ambiguous — ask the user which member they mean", ref)
	}
}

// visibleIDs returns the visible set as a slice for SQL ANY() binding.
func (tc *ToolContext) visibleIDs() []string {
	ids := make([]string, 0, len(tc.Visible))
	for id := range tc.Visible {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// parseWhen accepts ISO 8601 (with or without zone) or "now"; naive times are
// interpreted in the server's local zone. The LLM never does epoch math.
func (tc *ToolContext) parseWhen(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "now") {
		return time.Now().In(tc.Loc), nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, tc.Loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time %q — use ISO 8601 like 2026-07-19T15:00", s)
}

func (tc *ToolContext) fmtTime(ms int64) string {
	t := time.UnixMilli(ms).In(tc.Loc)
	age := time.Since(t)
	var rel string
	switch {
	case age < time.Minute:
		rel = "just now"
	case age < time.Hour:
		rel = fmt.Sprintf("%dm ago", int(age.Minutes()))
	case age < 48*time.Hour:
		rel = fmt.Sprintf("%.1fh ago", age.Hours())
	default:
		rel = fmt.Sprintf("%dd ago", int(age.Hours()/24))
	}
	return t.Format("Mon 2006-01-02 15:04") + " (" + rel + ")"
}

// QuietWindowActive reports whether the current UTC time falls in [start, end),
// mirroring ws.isQuietHoursNow. Accepts "HH:MM" or "HH:MM:SS" (DB ::text form).
func QuietWindowActive(start, end string, now time.Time) bool {
	if start == "" || end == "" {
		return false
	}
	nowMin := now.UTC().Hour()*60 + now.UTC().Minute()
	parse := func(s string) int {
		var h, m int
		fmt.Sscanf(s, "%d:%d", &h, &m)
		return h*60 + m
	}
	s, e := parse(start), parse(end)
	if s <= e {
		return nowMin >= s && nowMin < e
	}
	return nowMin >= s || nowMin < e // overnight window (e.g. 22:00–07:00)
}

// ─── Tool registry ───────────────────────────────────────────────────────────

// Tool couples an OpenAI function definition with its executor.
type Tool struct {
	Name        string
	Description string
	Params      string // JSON schema for arguments
	Exec        func(ctx context.Context, tc *ToolContext, args json.RawMessage) (any, error)
}

const toolQueryTimeout = 5 * time.Second

// Registry returns the copilot's read-only tool set.
func Registry() []Tool {
	return []Tool{
		{
			Name:        "get_current_positions",
			Description: "Latest known position of every visible family member: lat, lng, speed (m/s), and when it was recorded. Use this for any 'where is X now' question.",
			Params:      `{"type":"object","properties":{},"additionalProperties":false}`,
			Exec:        toolCurrentPositions,
		},
		{
			Name:        "get_position_history",
			Description: "Movement trail for ONE family member between two times: ordered GPS points with speed. Use for 'where was X', 'which route did X take', 'how fast'. Times are ISO 8601 local.",
			Params: `{"type":"object","properties":{
				"user":{"type":"string","description":"family member name or id"},
				"from":{"type":"string","description":"ISO 8601 start, e.g. 2026-07-19T08:00"},
				"to":{"type":"string","description":"ISO 8601 end, or 'now'"},
				"max_points":{"type":"integer","description":"downsample cap, default 60, max 200"}
			},"required":["user","from","to"],"additionalProperties":false}`,
			Exec: toolPositionHistory,
		},
		{
			Name:        "get_dwell_stops",
			Description: "Places where ONE family member STOPPED (dwelled) between two times: center point, arrival, departure, duration, and the nearest saved place if any. Use for 'where did X stop', 'how long was X at school'.",
			Params: `{"type":"object","properties":{
				"user":{"type":"string"},
				"from":{"type":"string"},
				"to":{"type":"string"},
				"min_minutes":{"type":"integer","description":"minimum dwell length, default 5"}
			},"required":["user","from","to"],"additionalProperties":false}`,
			Exec: toolDwellStops,
		},
		{
			Name:        "get_geofence_events",
			Description: "Geofence enter/exit events for one member or the whole family between two times (fence name, event type, location, time). Use for 'when did X arrive at / leave Y'.",
			Params: `{"type":"object","properties":{
				"user":{"type":"string","description":"optional: one member; omit for everyone visible"},
				"from":{"type":"string"},
				"to":{"type":"string"}
			},"required":["from","to"],"additionalProperties":false}`,
			Exec: toolGeofenceEvents,
		},
		{
			Name:        "get_saved_places",
			Description: "The asking user's saved places (Home, School, ...): name, lat, lng, radius meters. Use to relate positions to named places.",
			Params:      `{"type":"object","properties":{},"additionalProperties":false}`,
			Exec:        toolSavedPlaces,
		},
		{
			Name:        "distance_m",
			Description: "Great-circle distance in meters between two coordinates. ALWAYS use this instead of estimating distances yourself.",
			Params: `{"type":"object","properties":{
				"lat1":{"type":"number"},"lng1":{"type":"number"},
				"lat2":{"type":"number"},"lng2":{"type":"number"}
			},"required":["lat1","lng1","lat2","lng2"],"additionalProperties":false}`,
			Exec: toolDistance,
		},
		{
			Name:        "get_daily_activity",
			Description: "Per-day movement summary for ONE member over a date range: distance meters and active minutes per day. Use for 'how active was X this week'.",
			Params: `{"type":"object","properties":{
				"user":{"type":"string"},
				"from":{"type":"string","description":"ISO date, e.g. 2026-07-13"},
				"to":{"type":"string","description":"ISO date"}
			},"required":["user","from","to"],"additionalProperties":false}`,
			Exec: toolDailyActivity,
		},
	}
}

// Defs converts the registry to wire-format tool definitions.
func Defs(tools []Tool) []ToolDef {
	defs := make([]ToolDef, len(tools))
	for i, t := range tools {
		defs[i].Type = "function"
		defs[i].Function.Name = t.Name
		defs[i].Function.Description = t.Description
		defs[i].Function.Parameters = json.RawMessage(t.Params)
	}
	return defs
}

// ─── Executors ───────────────────────────────────────────────────────────────

func toolCurrentPositions(ctx context.Context, tc *ToolContext, _ json.RawMessage) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, toolQueryTimeout)
	defer cancel()

	rows, err := tc.DB.QueryContext(ctx, `
		SELECT DISTINCT ON (user_id) user_id, lat, lng, COALESCE(speed, 0), ts
		FROM position_history
		WHERE user_id = ANY($1)
		ORDER BY user_id, ts DESC`, pqStringArray(tc.visibleIDs()))
	if err != nil {
		return nil, fmt.Errorf("query current positions: %w", err)
	}
	defer rows.Close()

	type pos struct {
		User    string  `json:"user"`
		UserID  string  `json:"user_id"`
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		SpeedMS float64 `json:"speed_ms"`
		Seen    string  `json:"seen"`
	}
	var out []pos
	seen := map[string]bool{}
	for rows.Next() {
		var p pos
		var ts int64
		if err := rows.Scan(&p.UserID, &p.Lat, &p.Lng, &p.SpeedMS, &ts); err != nil {
			return nil, fmt.Errorf("scan current position: %w", err)
		}
		p.User = tc.Visible[p.UserID]
		p.Lat, p.Lng = tc.coord(p.UserID, p.Lat, p.Lng)
		if tc.coarsened(p.UserID) {
			p.SpeedMS = 0 // quiet hours: don't reveal movement, mirroring the live path
		}
		p.Seen = tc.fmtTime(ts)
		seen[p.UserID] = true
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate current positions: %w", err)
	}
	var noData []string
	for id, name := range tc.Visible {
		if !seen[id] {
			noData = append(noData, name+" ("+id+")")
		}
	}
	sort.Strings(noData)
	return map[string]any{"positions": out, "no_position_data": noData}, nil
}

func toolPositionHistory(ctx context.Context, tc *ToolContext, args json.RawMessage) (any, error) {
	var a struct {
		User      string `json:"user"`
		From      string `json:"from"`
		To        string `json:"to"`
		MaxPoints int    `json:"max_points"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("bad arguments: %w", err)
	}
	uid, name, err := tc.resolveUser(a.User)
	if err != nil {
		return nil, err
	}
	from, err := tc.parseWhen(a.From)
	if err != nil {
		return nil, err
	}
	to, err := tc.parseWhen(a.To)
	if err != nil {
		return nil, err
	}
	// Clamp to [2, 200]: a max of 1 would make the downsample stride divide by
	// zero (total-1)/(max-1), producing a NaN index and a panic.
	if a.MaxPoints < 2 {
		a.MaxPoints = 60
	}
	if a.MaxPoints > 200 {
		a.MaxPoints = 200
	}
	coarse := tc.coarsened(uid)

	ctx, cancel := context.WithTimeout(ctx, toolQueryTimeout)
	defer cancel()
	rows, err := tc.DB.QueryContext(ctx, `
		SELECT lat, lng, COALESCE(speed, 0), ts
		FROM position_history
		WHERE user_id = $1 AND ts >= $2 AND ts <= $3
		ORDER BY ts ASC
		LIMIT 2000`, uid, from.UnixMilli(), to.UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	type pt struct {
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		SpeedMS float64 `json:"speed_ms"`
		At      string  `json:"at"`
		ts      int64
	}
	var pts []pt
	for rows.Next() {
		var p pt
		if err := rows.Scan(&p.Lat, &p.Lng, &p.SpeedMS, &p.ts); err != nil {
			return nil, fmt.Errorf("scan history point: %w", err)
		}
		if coarse {
			p.Lat, p.Lng = tc.coord(uid, p.Lat, p.Lng)
			p.SpeedMS = 0
		}
		p.At = time.UnixMilli(p.ts).In(tc.Loc).Format("15:04:05")
		pts = append(pts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate history: %w", err)
	}
	total := len(pts)
	// Stride-downsample to max_points, always keeping first and last.
	if total > a.MaxPoints {
		stride := float64(total-1) / float64(a.MaxPoints-1)
		ds := make([]pt, 0, a.MaxPoints)
		for i := 0; i < a.MaxPoints; i++ {
			ds = append(ds, pts[int(float64(i)*stride+0.5)])
		}
		ds[len(ds)-1] = pts[total-1]
		pts = ds
	}
	var distM float64
	for i := 1; i < len(pts); i++ {
		distM += shared.HaversineM(pts[i-1].Lat, pts[i-1].Lng, pts[i].Lat, pts[i].Lng)
	}
	return map[string]any{
		"user": name, "user_id": uid,
		"from": from.Format("2006-01-02 15:04"), "to": to.Format("2006-01-02 15:04"),
		"total_points_recorded": total, "points_returned": len(pts),
		"approx_distance_m": math.Round(distM),
		"points":            pts,
	}, nil
}

func toolDwellStops(ctx context.Context, tc *ToolContext, args json.RawMessage) (any, error) {
	var a struct {
		User       string `json:"user"`
		From       string `json:"from"`
		To         string `json:"to"`
		MinMinutes int    `json:"min_minutes"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("bad arguments: %w", err)
	}
	uid, name, err := tc.resolveUser(a.User)
	if err != nil {
		return nil, err
	}
	from, err := tc.parseWhen(a.From)
	if err != nil {
		return nil, err
	}
	to, err := tc.parseWhen(a.To)
	if err != nil {
		return nil, err
	}
	if a.MinMinutes <= 0 {
		a.MinMinutes = 5
	}

	ctx, cancel := context.WithTimeout(ctx, toolQueryTimeout)
	defer cancel()
	rows, err := tc.DB.QueryContext(ctx, `
		SELECT lat, lng, ts
		FROM position_history
		WHERE user_id = $1 AND ts >= $2 AND ts <= $3
		ORDER BY ts ASC
		LIMIT 5000`, uid, from.UnixMilli(), to.UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	type raw struct {
		lat, lng float64
		ts       int64
	}
	var pts []raw
	for rows.Next() {
		var p raw
		if err := rows.Scan(&p.lat, &p.lng, &p.ts); err != nil {
			return nil, fmt.Errorf("scan dwell point: %w", err)
		}
		pts = append(pts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dwell history: %w", err)
	}
	coarse := tc.coarsened(uid)

	// Stay-point detection: greedy clustering — extend the current cluster
	// while points remain within dwellRadiusM of its running centroid.
	const dwellRadiusM = 80.0
	minDur := time.Duration(a.MinMinutes) * time.Minute

	type dwell struct {
		Lat         float64 `json:"lat"`
		Lng         float64 `json:"lng"`
		ArrivedAt   string  `json:"arrived_at"`
		LeftAt      string  `json:"left_at"`
		DurationMin float64 `json:"duration_min"`
		NearPlace   string  `json:"near_saved_place,omitempty"`
		NearDistM   float64 `json:"near_place_dist_m,omitempty"`
	}
	var dwells []dwell

	places, _ := loadSavedPlaces(ctx, tc)

	flush := func(cLat, cLng float64, startTs, endTs int64, n int) {
		if n == 0 {
			return
		}
		dur := time.UnixMilli(endTs).Sub(time.UnixMilli(startTs))
		if dur < minDur {
			return
		}
		dLat, dLng := tc.coord(uid, cLat, cLng)
		d := dwell{
			Lat: dLat, Lng: dLng,
			ArrivedAt:   tc.fmtTime(startTs),
			LeftAt:      tc.fmtTime(endTs),
			DurationMin: math.Round(dur.Minutes()*10) / 10,
		}
		// Place annotation uses the true centroid; skip it under quiet-hours
		// privacy so a coarsened stop isn't re-identified via a nearby place name.
		if !coarse {
			bestDist := 300.0 // only annotate if within 300 m
			for _, pl := range places {
				if dm := shared.HaversineM(cLat, cLng, pl.Lat, pl.Lng); dm < bestDist {
					bestDist = dm
					d.NearPlace = pl.Name
					d.NearDistM = math.Round(dm)
				}
			}
		}
		dwells = append(dwells, d)
	}

	var cLat, cLng float64
	var startTs, endTs int64
	n := 0
	for _, p := range pts {
		if n == 0 {
			cLat, cLng, startTs, endTs, n = p.lat, p.lng, p.ts, p.ts, 1
			continue
		}
		if shared.HaversineM(cLat, cLng, p.lat, p.lng) <= dwellRadiusM {
			// extend cluster; update running centroid
			cLat = (cLat*float64(n) + p.lat) / float64(n+1)
			cLng = (cLng*float64(n) + p.lng) / float64(n+1)
			endTs = p.ts
			n++
			continue
		}
		flush(cLat, cLng, startTs, endTs, n)
		cLat, cLng, startTs, endTs, n = p.lat, p.lng, p.ts, p.ts, 1
	}
	flush(cLat, cLng, startTs, endTs, n)

	return map[string]any{
		"user": name, "user_id": uid,
		"from": from.Format("2006-01-02 15:04"), "to": to.Format("2006-01-02 15:04"),
		"points_analyzed": len(pts),
		"dwells":          dwells,
	}, nil
}

func toolGeofenceEvents(ctx context.Context, tc *ToolContext, args json.RawMessage) (any, error) {
	var a struct {
		User string `json:"user"`
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("bad arguments: %w", err)
	}
	from, err := tc.parseWhen(a.From)
	if err != nil {
		return nil, err
	}
	to, err := tc.parseWhen(a.To)
	if err != nil {
		return nil, err
	}

	ids := tc.visibleIDs()
	if a.User != "" {
		uid, _, err := tc.resolveUser(a.User)
		if err != nil {
			return nil, err
		}
		ids = []string{uid}
	}

	ctx, cancel := context.WithTimeout(ctx, toolQueryTimeout)
	defer cancel()
	rows, err := tc.DB.QueryContext(ctx, `
		SELECT user_id, fence_name, event_type, lat, lng, ts
		FROM geofence_events
		WHERE user_id = ANY($1) AND ts >= $2 AND ts <= $3
		ORDER BY ts DESC
		LIMIT 100`, pqStringArray(ids), from.UnixMilli(), to.UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("query geofence events: %w", err)
	}
	defer rows.Close()

	type ev struct {
		User  string  `json:"user"`
		Fence string  `json:"fence"`
		Event string  `json:"event"` // enter | exit
		Lat   float64 `json:"lat"`
		Lng   float64 `json:"lng"`
		At    string  `json:"at"`
	}
	var out []ev
	for rows.Next() {
		var e ev
		var uid string
		var ts int64
		if err := rows.Scan(&uid, &e.Fence, &e.Event, &e.Lat, &e.Lng, &ts); err != nil {
			return nil, fmt.Errorf("scan geofence event: %w", err)
		}
		e.User = tc.Visible[uid]
		e.Lat, e.Lng = tc.coord(uid, e.Lat, e.Lng)
		e.At = tc.fmtTime(ts)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate geofence events: %w", err)
	}
	return map[string]any{"events": out, "count": len(out)}, nil
}

type savedPlace struct {
	Name    string  `json:"name"`
	Icon    string  `json:"icon,omitempty"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	RadiusM float64 `json:"radius_m"`
}

func loadSavedPlaces(ctx context.Context, tc *ToolContext) ([]savedPlace, error) {
	rows, err := tc.DB.QueryContext(ctx, `
		SELECT name, COALESCE(icon, ''), latitude, longitude, radius_m
		FROM saved_places
		WHERE user_id = $1
		ORDER BY name
		LIMIT 50`, tc.RequesterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []savedPlace
	for rows.Next() {
		var p savedPlace
		if err := rows.Scan(&p.Name, &p.Icon, &p.Lat, &p.Lng, &p.RadiusM); err != nil {
			return nil, fmt.Errorf("scan saved place: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate saved places: %w", err)
	}
	return out, nil
}

func toolSavedPlaces(ctx context.Context, tc *ToolContext, _ json.RawMessage) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, toolQueryTimeout)
	defer cancel()
	places, err := loadSavedPlaces(ctx, tc)
	if err != nil {
		return nil, fmt.Errorf("query saved places: %w", err)
	}
	return map[string]any{"places": places, "count": len(places)}, nil
}

func toolDistance(_ context.Context, _ *ToolContext, args json.RawMessage) (any, error) {
	var a struct {
		Lat1 float64 `json:"lat1"`
		Lng1 float64 `json:"lng1"`
		Lat2 float64 `json:"lat2"`
		Lng2 float64 `json:"lng2"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("bad arguments: %w", err)
	}
	m := shared.HaversineM(a.Lat1, a.Lng1, a.Lat2, a.Lng2)
	return map[string]any{"meters": math.Round(m), "km": math.Round(m/100) / 10}, nil
}

func toolDailyActivity(ctx context.Context, tc *ToolContext, args json.RawMessage) (any, error) {
	var a struct {
		User string `json:"user"`
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("bad arguments: %w", err)
	}
	uid, name, err := tc.resolveUser(a.User)
	if err != nil {
		return nil, err
	}
	from, err := tc.parseWhen(a.From)
	if err != nil {
		return nil, err
	}
	to, err := tc.parseWhen(a.To)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, toolQueryTimeout)
	defer cancel()
	rows, err := tc.DB.QueryContext(ctx, `
		SELECT date, distance_m, active_minutes
		FROM daily_activity
		WHERE user_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date ASC
		LIMIT 60`, uid, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("query daily activity: %w", err)
	}
	defer rows.Close()

	type day struct {
		Date          string `json:"date"`
		DistanceM     int    `json:"distance_m"`
		ActiveMinutes int    `json:"active_minutes"`
	}
	var out []day
	for rows.Next() {
		var d day
		var date time.Time
		if err := rows.Scan(&date, &d.DistanceM, &d.ActiveMinutes); err != nil {
			return nil, fmt.Errorf("scan daily activity: %w", err)
		}
		d.Date = date.Format("2006-01-02")
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daily activity: %w", err)
	}
	return map[string]any{"user": name, "days": out}, nil
}

// pqStringArray renders a []string as a PostgreSQL text[] literal for ANY($1)
// binding without importing pq. IDs are UUIDs from our own DB (no injection
// surface), but values are still quoted defensively.
func pqStringArray(ids []string) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(id))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}
