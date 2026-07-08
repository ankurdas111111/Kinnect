package ws

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/db"
	"kinnect-v3/internal/shared"
)

const (
	sosTokenBytes    = 16
	sosWatchExpiryMs = 24 * 60 * 60 * 1000
)

// generateSosToken returns a base64url random token for SOS watch links.
func generateSosToken() string {
	b := make([]byte, sosTokenBytes)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:22]
}

// assembleSosNarrative builds a narrative from the rolling in-memory buffer (no DB call).
func (h *Hub) assembleSosNarrative(user *cache.ActiveUser, triggerRule string) *cache.SosNarrative {
	entries := h.rollingSnapshot(user.UserID)

	narrative := &cache.SosNarrative{
		TriggerRule:  triggerRule,
		LastSignalTs: user.LastUpdate,
		BatteryPct:   user.BatteryPct,
	}

	if len(entries) == 0 {
		narrative.MotionSummary = "No recent position data"
		narrative.TrackGeojson = `{"type":"LineString","coordinates":[]}`
		return narrative
	}

	// Build GeoJSON LineString
	coords := make([]string, 0, len(entries))
	var stationaryStart int64
	longestStationary := int64(0)

	for i, e := range entries {
		coords = append(coords, fmt.Sprintf("[%f,%f]", e.Lng, e.Lat))
		if e.SpeedMs <= 0.5 {
			if stationaryStart == 0 {
				stationaryStart = e.Ts
			}
		} else {
			if stationaryStart > 0 && i > 0 {
				dur := entries[i-1].Ts - stationaryStart
				if dur > longestStationary {
					longestStationary = dur
				}
				stationaryStart = 0
			}
		}
	}
	// Close any open stationary window
	if stationaryStart > 0 {
		dur := entries[len(entries)-1].Ts - stationaryStart
		if dur > longestStationary {
			longestStationary = dur
		}
	}

	narrative.TrackGeojson = fmt.Sprintf(
		`{"type":"LineString","coordinates":[%s]}`,
		strings.Join(coords, ","))

	if longestStationary > 0 {
		narrative.MotionSummary = fmt.Sprintf(
			"Stationary for %d min before SOS", longestStationary/60000)
	} else {
		narrative.MotionSummary = fmt.Sprintf("Active movement in last 30 min (%d fixes)", len(entries))
	}

	return narrative
}

// handleTriggerSOS sets SOS active, creates watch token, emits to contacts/visible/live.
func (h *Hub) handleTriggerSOS(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("triggerSOS", 5) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	reason := ""
	if r, ok := m["reason"].(string); ok {
		reason = shared.SanitizeString(r, 100)
	}
	sosType := "manual"
	if t, ok := m["type"].(string); ok {
		sosType = shared.SanitizeString(t, 20)
	}
	if reason == "" {
		reason = "SOS triggered"
	}

	// Optional medical card snapshot — user-consented, device-local data
	// sent only when the triggering user has filled out their emergency profile.
	var medicalCard map[string]interface{}
	if mc, ok := m["medicalCard"].(map[string]interface{}); ok && len(mc) > 0 {
		medicalCard = sanitizeMedicalCard(mc)
	}

	token := generateSosToken()
	exp := time.Now().UnixMilli() + sosWatchExpiryMs
	h.setSos(user, true, reason, "", sosType)
	user.SOS.Token = &token
	user.SOS.TokenExp = &exp

	// Assemble and store narrative
	narrative := h.assembleSosNarrative(user, sosType)
	user.SOS.Narrative = narrative

	// Store watch token for public /watch/:token page
	h.Cache.SetWatchToken(token, user.SocketID, user.UserID, exp)

	// DB: live_tokens-style entry for watch - reuses watch token cache
	// No separate DB table for watch tokens in schema; watch uses in-memory only
	h.emitSosUpdate(user)
	h.emitWatch(user)

	// Emit narrative separately so receivers can render crisis card
	if narrative != nil {
		narrativePayload := map[string]interface{}{
			"sosToken": token,
			"userId":   user.UserID,
			"narrative": map[string]interface{}{
				"trackGeojson":  narrative.TrackGeojson,
				"motionSummary": narrative.MotionSummary,
				"batteryPct":    narrative.BatteryPct,
				"triggerRule":   narrative.TriggerRule,
				"lastSignalTs":  narrative.LastSignalTs,
			},
		}
		// Include medical card when provided — shown in family AlertOverlay
		if len(medicalCard) > 0 {
			narrativePayload["medicalCard"] = medicalCard
		}
		h.emitToVisibleAndSelf(user, "sosNarrative", narrativePayload)
		// Also send to live links
		tokens := h.Cache.GetLiveTokensForUser(user.UserID)
		for lt := range tokens {
			h.SendToGroup("live:"+lt, "sosNarrative", narrativePayload)
		}
	}

	// ── Proximity SOS broadcast (5 km radius) ──────────────────────────────
	// Notify nearby active users who are NOT already in the SOS user's visible
	// set (family/contacts already receive the full sosUpdate above).
	if user.Latitude != nil && user.Longitude != nil {
		sosLat, sosLng := *user.Latitude, *user.Longitude
		watchToken := token // always non-empty at this point

		// Collect socket IDs that already received the full SOS alert.
		alreadyNotified := make(map[string]bool)
		for _, sid := range h.Cache.GetVisibleSocketIDs(user) {
			alreadyNotified[sid] = true
		}
		alreadyNotified[user.SocketID] = true

		h.Cache.ForEachActiveUser(func(sid string, u *cache.ActiveUser) {
			if alreadyNotified[sid] {
				return
			}
			if u.Latitude == nil || u.Longitude == nil {
				return
			}
			distKm := haversineKm(sosLat, sosLng, *u.Latitude, *u.Longitude)
			if distKm > 5.0 {
				return
			}
			distRounded := math.Round(distKm*10) / 10
			h.SendToClient(sid, "proximitySosAlert", map[string]interface{}{
				"distanceKm": distRounded,
				"watchToken": watchToken,
			})
		})
	}

	// ── Panic Relay: start 3-minute SMS escalation timer ─────────────────
	h.StartPanicRelayTimer(user)
}

// sanitizeMedicalCard keeps only known string fields and truncates each to 500 chars.
// Also sanitizes the emergencyContacts array (max 5 entries, 4 fields each).
// This prevents injection of unexpected keys and limits payload size.
func sanitizeMedicalCard(mc map[string]interface{}) map[string]interface{} {
	allowed := []string{
		"fullName", "dob", "bloodType",
		"emergencyName", "emergencyPhone",
		"conditions", "allergies", "medications",
		"doctorName", "doctorPhone",
		"language", "responderNotes",
	}
	out := make(map[string]interface{}, len(allowed)+1)
	for _, k := range allowed {
		if v, ok := mc[k].(string); ok && v != "" {
			out[k] = shared.SanitizeString(v, 500)
		}
	}
	// Sanitize emergencyContacts array — up to 5 contacts, 4 string fields each.
	if contacts, ok := mc["emergencyContacts"].([]interface{}); ok {
		allowedContactFields := []string{"name", "relation", "phone", "address"}
		var sanitized []map[string]interface{}
		for _, raw := range contacts {
			cm, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			sc := make(map[string]interface{}, 4)
			for _, f := range allowedContactFields {
				if v, ok := cm[f].(string); ok && v != "" {
					sc[f] = shared.SanitizeString(v, 200)
				}
			}
			if len(sc) > 0 {
				sanitized = append(sanitized, sc)
			}
			if len(sanitized) >= 5 {
				break
			}
		}
		if len(sanitized) > 0 {
			out["emergencyContacts"] = sanitized
		}
	}
	return out
}

// haversineKm returns the great-circle distance in kilometres between two lat/lng points.
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// handleCancelSOS clears SOS, deletes watch token, emits watchUpdate and sosUpdate.
func (h *Hub) handleCancelSOS(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("cancelSOS", 5) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	// KR-015: emit watchUpdate with active:false BEFORE clearing the token.
	// emitWatch exits early if user.SOS.Token is nil, so send the final update first.
	h.emitWatch(user)
	token := user.SOS.Token
	h.setSos(user, false, "", "", "")
	if token != nil {
		h.Cache.DeleteWatchToken(*token)
	}
	payload := h.publicSos(user)
	h.emitToVisibleAndSelf(user, "sosUpdate", payload)
	h.emitLiveSos(user)
}

// handleAckSOS finds target by userId, adds ack (deduplicated by name), emits sosUpdate.
func (h *Hub) handleAckSOS(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("ackSOS", 10) {
		return
	}
	m := toMap(data)
	// Accept userId (stable across reconnects) or fall back to socketId for compatibility.
	targetUserID, _ := m["userId"].(string)
	var target *cache.ActiveUser
	if targetUserID != "" {
		if sid := h.Cache.GetUserIdToSocketId(targetUserID); sid != "" {
			target = h.Cache.GetActiveUser(sid)
		}
	} else {
		socketId, _ := m["socketId"].(string)
		if socketId == "" {
			return
		}
		target = h.Cache.GetActiveUser(socketId)
	}
	if target == nil || !target.SOS.Active {
		return
	}
	ackerName := h.Cache.GetDisplayName(c.UserID())
	// Deduplicate: do not append if same name already present.
	for _, a := range target.SOS.Acks {
		if a.By == ackerName {
			return
		}
	}
	target.SOS.Acks = append(target.SOS.Acks, cache.SosAckEntry{By: ackerName})
	h.emitSosUpdate(target)
	h.emitWatch(target)
}

// handleCheckInAck updates lastCheckInAt, emits to visible+self and live links.
func (h *Hub) handleCheckInAck(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("checkInAck", 20) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	user.CheckIn.LastCheckInAt = time.Now().UnixMilli()
	user.CheckIn.RequestedAt = 0
	user.CheckIn.MissedNotifiedAt = 0
	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	ci := map[string]interface{}{
		"userId": user.UserID, "lastCheckInAt": user.CheckIn.LastCheckInAt,
	}
	h.emitToVisibleAndSelf(user, "checkInUpdate", ci)
	h.emitLiveCheckIn(user)
}

// handleSetCheckInRules updates check-in config on user, broadcasts userUpdate.
func (h *Hub) handleSetCheckInRules(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("setCheckInRules", 10) {
		return
	}
	m := toMap(data)
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	if v, ok := m["enabled"].(bool); ok {
		user.CheckIn.Enabled = v
	}
	if v, ok := toInt(m["intervalMin"]); ok && v >= 0 {
		user.CheckIn.IntervalMin = v
	}
	if v, ok := toInt(m["overdueMin"]); ok && v >= 0 {
		user.CheckIn.OverdueMin = v
	}
	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
}

// handleSetGeofence updates geofence config, broadcasts userUpdate.
func (h *Hub) handleSetGeofence(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("setGeofence", 10) {
		return
	}
	m := toMap(data)
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	if v, ok := m["enabled"].(bool); ok {
		user.Geofence.Enabled = v
	}
	if v, ok := toFloat64(m["centerLat"]); ok {
		user.Geofence.CenterLat = &v
	}
	if v, ok := toFloat64(m["centerLng"]); ok {
		user.Geofence.CenterLng = &v
	}
	if v, ok := toFloat64(m["radiusM"]); ok && v >= 0 {
		user.Geofence.RadiusM = v
	}
	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
}

// handleSetAutoSos updates auto-SOS config, broadcasts userUpdate.
func (h *Hub) handleSetAutoSos(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("setAutoSos", 10) {
		return
	}
	m := toMap(data)
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	if v, ok := m["enabled"].(bool); ok {
		user.AutoSOS.Enabled = v
	}
	if v, ok := toInt(m["noMoveMinutes"]); ok && v >= 0 {
		user.AutoSOS.NoMoveMinutes = v
	}
	if v, ok := toInt(m["hardStopMin"]); ok && v >= 0 {
		user.AutoSOS.HardStopMin = v
	}
	if v, ok := m["geofence"].(bool); ok {
		user.AutoSOS.Geofence = v
	}
	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
}

// handleLiveAckSOS: live viewer acks SOS; uses c.liveToken, c.liveViewerName.
func (h *Hub) handleLiveAckSOS(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("liveAckSOS", 10) {
		return
	}
	token := c.liveToken
	if token == "" {
		return
	}
	entry := h.Cache.GetLiveToken(token)
	if entry == nil {
		return
	}
	targetSid := h.Cache.GetUserIdToSocketId(entry.UserID)
	target := h.Cache.GetActiveUser(targetSid)
	if target == nil {
		// Target offline - could use offlineUsers
		return
	}
	viewerName := c.liveViewerName
	if viewerName == "" {
		viewerName = "Viewer"
	}
	ackLabel := viewerName + " (via link)"
	for _, a := range target.SOS.Acks {
		if a.By == ackLabel {
			return
		}
	}
	target.SOS.Acks = append(target.SOS.Acks, cache.SosAckEntry{By: ackLabel})
	h.emitSosUpdate(target)
	h.emitWatch(target)
}

// handleGetGeofenceLog returns the last 50 geofence entry/exit events for the caller. (F6)
func (h *Hub) handleGetGeofenceLog(c *Client, _ json.RawMessage) {
	if !c.CheckRateLimit("getGeofenceLog", 20) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	userID := user.UserID
	h.offloadDB(func(ctx context.Context) {
		rows, err := db.GetGeofenceEvents(ctx, h.pool.DB, userID, 50)
		if err != nil {
			c.Send("geofenceLog", map[string]interface{}{"events": []interface{}{}})
			return
		}
		events := make([]map[string]interface{}, 0, len(rows))
		for _, r := range rows {
			events = append(events, map[string]interface{}{
				"id":        r.ID,
				"fenceName": r.FenceName,
				"eventType": r.EventType,
				"lat":       r.Lat,
				"lng":       r.Lng,
				"ts":        r.Ts,
			})
		}
		c.Send("geofenceLog", map[string]interface{}{"events": events})
	})
}

func toInt(v interface{}) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case int64:
		return int(x), true
	default:
		return 0, false
	}
}

func toFloat64(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}
