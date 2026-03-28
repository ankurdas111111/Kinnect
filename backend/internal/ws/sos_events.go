package ws

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/shared"
)

const (
	sosTokenBytes   = 16
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
		h.emitToVisibleAndSelf(user, "sosNarrative", narrativePayload)
		// Also send to live links
		tokens := h.Cache.GetLiveTokensForUser(user.UserID)
		for lt := range tokens {
			h.SendToGroup("live:"+lt, "sosNarrative", narrativePayload)
		}
	}
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
	token := user.SOS.Token
	h.setSos(user, false, "", "", "")
	if token != nil {
		h.Cache.DeleteWatchToken(*token)
	}
	h.emitWatch(user)
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
