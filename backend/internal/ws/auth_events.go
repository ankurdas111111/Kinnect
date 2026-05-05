package ws

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	mathrand "math/rand"
	"strconv"
	"strings"
	"time"

	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/db"
	"kinnect-v3/internal/intelligence"
	"kinnect-v3/internal/shared"
)

const (
	positionCooldownMs   = 100
	positionRateMin      = 360
	positionBatchRateMin = 5
	dbSaveThrottleMs     = 30000
	maxRoomsPerUser      = 20
	maxContactsPerUser   = 50
	maxLiveLinksPerUser  = 10
)

// dbWriteSem bounds the number of concurrent goroutines performing DB writes
// spawned by position update handlers. Without this limit, a spike of 200-item
// batches from many clients could spawn thousands of goroutines simultaneously.
var dbWriteSem = make(chan struct{}, 64)

// positionPayload is a lean struct for the userMoved broadcast.
// It holds only the ~12 fields that change on each position update, avoiding the
// ~25 nested-map allocations that SanitizeUser produces on every ~20 Hz broadcast.
// Other events (existingUsers, userUpdate, userOffline) continue to use SanitizeUser.
type positionPayload struct {
	SocketID    string   `json:"socketId"`
	UserID      string   `json:"userId"`
	Lat         *float64 `json:"lat"`
	Lng         *float64 `json:"lng"`
	Speed       float64  `json:"speed"`
	LastUpdate  int64    `json:"lastUpdate"`
	BatteryPct  *int     `json:"batteryPct"`
	Online      bool     `json:"online"`
	MotionClass string   `json:"motionClass"`
	SafetyScore float64  `json:"safetyScore"`
	ActivityCtx string   `json:"activityContext"`
	SOSActive   bool     `json:"sosActive"`
}

// motionClass derives a motion class string from speed in km/h.
// Frontend Kalman filter emits speed in km/h (browser raw m/s × 3.6).
func motionClass(speedKmh float64) string {
	switch {
	case speedKmh < 1:
		return "still"
	case speedKmh < 7:
		return "walk"
	case speedKmh < 18:
		return "run"
	default:
		return "vehicle"
	}
}

// toMovementPhase maps speed in km/h to intelligence movement phase label.
func toMovementPhase(speedKmh float64) string {
	switch {
	case speedKmh < 1:
		return "stationary"
	case speedKmh < 10:
		return "walking"
	case speedKmh < 50:
		return "driving"
	default:
		return "transit"
	}
}

// shouldWritePositionToDB returns true when a position update is meaningful enough to persist.
// speedKmh is the filtered speed in km/h (same unit used by motionClass).
func shouldWritePositionToDB(user *cache.ActiveUser, lat, lng, speedKmh float64) bool {
	if user.LastDBAt == 0 {
		return true // first position ever
	}
	if shared.HaversineM(user.LastDBLat, user.LastDBLng, lat, lng) > 100 {
		return true // moved 100m
	}
	if motionClass(speedKmh) != user.MotionClass {
		return true // class transition
	}
	if user.MotionClass != "still" &&
		time.Now().UnixMilli()-user.LastDBAt > 5*60*1000 {
		return true // 5-min heartbeat for moving users
	}
	return false
}

func toMap(data json.RawMessage) map[string]interface{} {
	if len(data) == 0 {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}

// handlePosition handles position updates with rate limit, cooldown, validation.
func (h *Hub) handlePosition(c *Client, data json.RawMessage) {
	clientID := c.ID()
	now := time.Now().UnixMilli()

	if !c.CheckRateLimit("position", positionRateMin) {
		return
	}
	if now-h.Cache.GetLastPositionAt(clientID) < positionCooldownMs {
		return
	}
	h.Cache.SetLastPositionAt(clientID, now)

	m := toMap(data)

	// ── F1: Battery level ─────────────────────────────────────────────────────
	// Read battery from the same position payload — avoids a round-trip profileUpdate.
	// handleProfileUpdate still works for dedicated battery-only updates.
	user := h.Cache.GetActiveUser(clientID)
	if user != nil {
		if v, ok := m["batteryPct"].(float64); ok && v >= 0 && v <= 100 {
			pct := int(v)
			user.BatteryPct = &pct
		}
	}

	pos := shared.ValidatePosition(m)
	if pos == nil {
		return
	}

	if user == nil {
		return
	}

	// Update user fields
	user.Latitude = &pos.Latitude
	user.Longitude = &pos.Longitude
	user.Speed = pos.Speed
	user.LastUpdate = now
	user.FormattedTime = pos.FormattedTime
	user.Accuracy = pos.Accuracy
	prevSpeed := user.LastSpeed
	user.LastSpeed = pos.Speed
	if pos.Speed > 0.8 {
		user.LastMoveAt = now
	}
	if prevSpeed > 25 && pos.Speed < 2 {
		t := now
		user.HardStopAt = &t
	}

	// ── Rolling buffer (always, every update — feeds SOS narrative) ──────────
	h.pushRollingEntry(user.UserID, RollingEntry{
		Lat:        pos.Latitude,
		Lng:        pos.Longitude,
		SpeedMs:    pos.Speed,
		BatteryPct: user.BatteryPct,
		Ts:         now,
	})

	// ── Motion class ─────────────────────────────────────────────────────────
	newClass := motionClass(pos.Speed)
	user.MotionClass = newClass

	// ── Movement phase (intelligence label) ──────────────────────────────────
	user.MovementPhase = toMovementPhase(pos.Speed)

	// ── Heartbeat signal (any position update = sign of life) ────────────────
	if user.HeartbeatEnabled {
		user.HeartbeatLastSignal = now
	}

	// ── Journey Shield — trip detection ──────────────────────────────────────
	h.EvaluateTrip(user)

	// ── F5: Speed alert — notify guardians when ward exceeds threshold ────────
	// Threshold is in m/s (stored as speed_alert_threshold_ms in DB/cache).
	// pos.Speed is the Kalman-filtered speed in km/h from the frontend.
	// Convert pos.Speed (km/h) → m/s for comparison.
	if user.SpeedAlertThresholdMs > 0 && user.Latitude != nil && user.Longitude != nil {
		speedMs := pos.Speed / 3.6
		const speedAlertCooldownMs = 60_000 // rate-limit to once per minute
		if speedMs > user.SpeedAlertThresholdMs && now-user.LastSpeedAlertAt > speedAlertCooldownMs {
			user.LastSpeedAlertAt = now
			guardianSIDs := h.Cache.GetGuardianSocketIDs(user.UserID)
			if len(guardianSIDs) > 0 {
				payload := map[string]interface{}{
					"userId":      user.UserID,
					"displayName": user.DisplayName,
					"speedMs":     speedMs,
					"thresholdMs": user.SpeedAlertThresholdMs,
					"lat":         user.Latitude,
					"lng":         user.Longitude,
					"at":          now,
				}
				h.SendToClients(guardianSIDs, "speedAlert", payload)
			}
		}
	}

	// ── Safety score (recomputed on every position update) ───────────────────
	{
		checkInOverdueAt := int64(0)
		if user.CheckIn.Enabled && user.CheckIn.IntervalMin > 0 && user.CheckIn.LastCheckInAt > 0 {
			checkInOverdueAt = user.CheckIn.LastCheckInAt +
				int64(user.CheckIn.IntervalMin+user.CheckIn.OverdueMin)*60*1000
		}
		geofenceBreached := user.Geofence.Enabled &&
			user.Geofence.CenterLat != nil && user.Geofence.CenterLng != nil &&
			user.Latitude != nil && user.Longitude != nil &&
			shared.HaversineM(*user.Geofence.CenterLat, *user.Geofence.CenterLng,
				*user.Latitude, *user.Longitude) > user.Geofence.RadiusM
		sc := intelligence.ComputeSafetyScore(
			user.Accuracy,
			user.LastUpdate,
			user.LastAttestAt,
			user.CheckIn.Enabled,
			checkInOverdueAt,
			user.Geofence.Enabled,
			geofenceBreached,
			user.MotionClass,
		)
		user.SafetyScore = sc.Total
	}

	if shouldWritePositionToDB(user, pos.Latitude, pos.Longitude, pos.Speed) {
		// F9: capture old DB position BEFORE updating, so distDelta is meaningful.
		oldLat, oldLng := user.LastDBLat, user.LastDBLng
		user.LastDBLat = pos.Latitude
		user.LastDBLng = pos.Longitude
		user.LastDBAt = now

		lat, lng, spd := pos.Latitude, pos.Longitude, pos.Speed
		uid := user.UserID

		// ── F9: daily activity counters ───────────────────────────────────────
		// Compute distance delta and whether this minute is new (for active-minutes).
		// Guard oldLat==0: first-ever DB write has no previous position, so distance
		// must be 0 rather than a spurious haversine from (0,0) to current position.
		distDelta := 0
		if oldLat != 0 {
			distDelta = int(shared.HaversineM(oldLat, oldLng, lat, lng))
		}
		addMinute := false
		nowMin := now / 60000
		if user.LastActiveMinuteAt == 0 || nowMin != user.LastActiveMinuteAt/60000 {
			addMinute = true
			user.LastActiveMinuteAt = now
		}

		// Non-blocking acquire: drop the write under extreme load rather than
		// blocking the hub's dispatch goroutine or spawning unbounded goroutines.
		select {
		case dbWriteSem <- struct{}{}:
			go func(distM int, addMin bool) {
				defer func() { <-dbWriteSem }()
				bCtx := context.Background()
				// users.last_* snapshot
				speedStr := fmt.Sprintf("%.2f", spd)
				_ = db.UpdateUserLocation(bCtx, h.pool.DB, uid, lat, lng, speedStr, now)
				// F9: upsert daily activity
				if err := db.UpsertDailyActivity(bCtx, h.pool.DB, uid, distM, addMin, now); err != nil {
					slog.Warn("UpsertDailyActivity failed", "userId", uid, "error", err)
				}
				// F10: append position_history row for trail replay
				if _, err := h.pool.DB.ExecContext(bCtx,
					`INSERT INTO position_history(user_id,lat,lng,speed,ts) VALUES($1,$2,$3,$4,$5)`,
					uid, lat, lng, spd, now); err != nil {
					slog.Warn("position_history insert failed", "userId", uid, "error", err)
				}
			}(distDelta, addMinute)
		default:
			slog.Warn("dbWriteSem full, skipping position DB write", "userId", uid)
		}
	}

	// ── F7: Proximity alerts — check if any owner set an alert for this user ──
	// Iterate over alerts that target user.UserID; alert ownerIDs when within radius.
	if user.Latitude != nil && user.Longitude != nil {
		alerts := h.Cache.GetProximityAlertsForTarget(user.UserID)
		for _, alert := range alerts {
			if !alert.Enabled {
				continue
			}
			const proxCooldownMs = 60_000 // alert at most once per minute per pair
			if now-alert.LastTriggeredAt < proxCooldownMs {
				continue
			}
			ownerSID := h.Cache.GetUserIdToSocketId(alert.OwnerID)
			if ownerSID == "" {
				continue
			}
			ownerUser := h.Cache.GetActiveUser(ownerSID)
			if ownerUser == nil || ownerUser.Latitude == nil || ownerUser.Longitude == nil {
				continue
			}
			dist := shared.HaversineM(*ownerUser.Latitude, *ownerUser.Longitude, *user.Latitude, *user.Longitude)
			if dist > float64(alert.RadiusM) {
				continue
			}
			// Within radius — fire alert
			alert.LastTriggeredAt = now
			h.SendToClient(ownerSID, "proximityAlert", map[string]interface{}{
				"targetUserId":   user.UserID,
				"targetName":     user.DisplayName,
				"distanceM":      int(math.Round(dist)),
				"radiusM":        alert.RadiusM,
				"lat":            user.Latitude,
				"lng":            user.Longitude,
				"at":             now,
			})
			// Persist last triggered async
			alertID := alert.ID
			go func() {
				_ = db.UpdateProximityAlertTriggered(context.Background(), h.pool.DB, alertID, now)
			}()
		}
	}

	// ── Arrival projection — ETA to nearest saved place within 5 km ─────────
	// Emits arrivalProjection to visible users at most every 30 s.
	// Sends etaSeconds=null when user enters the place radius (arrival).
	if user.Latitude != nil && user.Longitude != nil {
		const arrivalCooldownMs = 30_000
		if now-user.LastArrivalProjectionAt >= arrivalCooldownMs {
			places := h.Cache.GetSavedPlaces(user.UserID)
			if len(places) > 0 {
				var bestIdx int = -1
				var bestDist float64
				for i, p := range places {
					d := shared.HaversineM(*user.Latitude, *user.Longitude, p.Latitude, p.Longitude)
					if d < 5000 && (bestIdx < 0 || d < bestDist) {
						bestIdx = i
						bestDist = d
					}
				}
				if bestIdx >= 0 {
					p := places[bestIdx]
					user.LastArrivalProjectionAt = now
					visibleSIDs := h.Cache.GetVisibleSocketIDs(user)
					if len(visibleSIDs) > 0 {
						if bestDist <= p.RadiusM {
							// User has arrived — clear projection
							h.SendToClients(visibleSIDs, "arrivalProjection", map[string]interface{}{
								"userId": user.UserID,
							})
						} else if user.Speed > 1.5 {
							// User is moving toward place — compute ETA
							speedMs := user.Speed / 3.6
							etaSec := int(bestDist / speedMs)
							if etaSec > 0 && etaSec < 7200 {
								h.SendToClients(visibleSIDs, "arrivalProjection", map[string]interface{}{
									"userId":      user.UserID,
									"displayName": user.DisplayName,
									"placeId":     p.ID,
									"placeName":   p.Name,
									"distanceM":   int(math.Round(bestDist)),
									"etaSeconds":  etaSec,
									"at":          now,
								})
							}
						}
					}
				}
			}
		}
	}

	// ── Activity context (derived from motion class + arrival state) ─────────
	user.ActivityContext = computeActivityContext(user.UserID, user.MotionClass)

	// ── Battery proxy alerts ──────────────────────────────────────────────────
	checkBatteryAlerts(h, user)

	// ── M-5: count every accepted position update ─────────────────────────────
	if h.metrics != nil {
		h.metrics.PositionUpdatesTotal.Inc()
	}

	// ── M-1: lean payload for userMoved — avoids ~25 nested-map allocations ──
	// queuePositionBroadcast (and the downstream flushPositionBroadcasts) emits
	// "userMoved"; only position-volatile fields are needed there.
	lean := map[string]interface{}{
		"socketId":        user.SocketID,
		"userId":          user.UserID,
		"lat":             user.Latitude,
		"lng":             user.Longitude,
		"speed":           user.Speed,
		"lastUpdate":      user.LastUpdate,
		"batteryPct":      user.BatteryPct,
		"online":          true,
		"motionClass":     user.MotionClass,
		"safetyScore":     user.SafetyScore,
		"activityContext": user.ActivityContext,
		"sosActive":       user.SOS.Active,
	}
	h.queuePositionBroadcast(user, lean)

	// liveUpdate viewers need the full snapshot (displayName, rooms, geofence, etc.)
	tokens := h.Cache.GetLiveTokensForUser(user.UserID)
	if len(tokens) > 0 {
		sanitized := h.Cache.SanitizeUser(user)
		sanitized["online"] = true
		for token := range tokens {
			h.SendToGroup("live:"+token, "liveUpdate", map[string]interface{}{"user": sanitized})
		}
	}

	h.runAutoRules(user)
	h.checkCrowdMode(user)
}

// handlePositionBatch handles batched position updates.
// Applies the same motion/safety/auto-rules logic as handlePosition to each item.
func (h *Hub) handlePositionBatch(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("positionBatch", positionBatchRateMin) {
		return
	}

	var batch []map[string]interface{}
	if err := json.Unmarshal(data, &batch); err != nil || len(batch) == 0 {
		return
	}
	// M-2: cap at 20 to prevent blocking the hub dispatch goroutine for >100ms.
	// Keep the most-recent 20 items (tail) so the final broadcast reflects the
	// latest GPS fixes from the offline period.
	if len(batch) > 20 {
		slog.Debug("positionBatch truncated", "original", len(batch), "kept", 20)
		batch = batch[len(batch)-20:]
	}

	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}

	// KR-014: per-user rate limit so reconnect cycles can't bypass the per-client limit.
	// One drain per 12 s ≈ 5 / min, same as positionBatchRateMin.
	const batchUserCooldownMs = 12000
	nowMs := time.Now().UnixMilli()
	if nowMs-h.batchUserLast[user.UserID] < batchUserCooldownMs {
		return
	}
	h.batchUserLast[user.UserID] = nowMs

	for _, m := range batch {
		pos := shared.ValidatePosition(m)
		if pos == nil {
			continue
		}
		// KR-013: use the original GPS timestamp when available so position_history
		// records reflect when the fix was taken, not when it was replayed.
		now := time.Now().UnixMilli()
		if pos.Timestamp != nil {
			ts := *pos.Timestamp
			// Accept timestamps in the past and within the last 24 hours so that
			// positions buffered during flights / long offline periods are replayed
			// with their original GPS timestamps (L-3).
			if ts > 0 && ts <= now && now-ts <= 86400000 {
				now = ts
			}
		}
		user.Latitude = &pos.Latitude
		user.Longitude = &pos.Longitude
		user.Speed = pos.Speed
		user.LastUpdate = now
		user.FormattedTime = pos.FormattedTime
		user.Accuracy = pos.Accuracy

		// Rolling buffer feeds SOS narrative
		h.pushRollingEntry(user.UserID, RollingEntry{
			Lat:        pos.Latitude,
			Lng:        pos.Longitude,
			SpeedMs:    pos.Speed,
			BatteryPct: user.BatteryPct,
			Ts:         now,
		})

		// Motion tracking
		user.MotionClass = motionClass(pos.Speed)
		user.MovementPhase = toMovementPhase(pos.Speed)
		if pos.Speed > 0.8 {
			user.LastMoveAt = now
		}
		prevSpeed := user.LastSpeed
		user.LastSpeed = pos.Speed
		if prevSpeed > 25 && pos.Speed < 2 {
			t := now
			user.HardStopAt = &t
		}

		// position_history dual-write removed — movement_events covers semantic history
	}

	// Safety score on final position
	{
		checkInOverdueAt := int64(0)
		if user.CheckIn.Enabled && user.CheckIn.IntervalMin > 0 && user.CheckIn.LastCheckInAt > 0 {
			checkInOverdueAt = user.CheckIn.LastCheckInAt +
				int64(user.CheckIn.IntervalMin+user.CheckIn.OverdueMin)*60*1000
		}
		geofenceBreached := user.Geofence.Enabled &&
			user.Geofence.CenterLat != nil && user.Geofence.CenterLng != nil &&
			user.Latitude != nil && user.Longitude != nil &&
			shared.HaversineM(*user.Geofence.CenterLat, *user.Geofence.CenterLng,
				*user.Latitude, *user.Longitude) > user.Geofence.RadiusM
		sc := intelligence.ComputeSafetyScore(
			user.Accuracy,
			user.LastUpdate,
			user.LastAttestAt,
			user.CheckIn.Enabled,
			checkInOverdueAt,
			user.Geofence.Enabled,
			geofenceBreached,
			user.MotionClass,
		)
		user.SafetyScore = sc.Total
	}

	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.queuePositionBroadcast(user, sanitized)
	tokens := h.Cache.GetLiveTokensForUser(user.UserID)
	for token := range tokens {
		h.SendToGroup("live:"+token, "liveUpdate", map[string]interface{}{"user": sanitized})
	}
	h.runAutoRules(user)
	h.checkCrowdMode(user) // KR-005: crowd mode must re-evaluate after batch replay, same as single position
}

// handleProfileUpdate updates battery, deviceType, connectionQuality.
func (h *Hub) handleProfileUpdate(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("profileUpdate", 20) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}
	if v, ok := m["batteryPct"].(float64); ok && v >= 0 && v <= 100 {
		pct := int(v)
		user.BatteryPct = &pct
	}
	if s, ok := m["deviceType"].(string); ok {
		sanitized := shared.SanitizeString(s, 20)
		user.DeviceType = &sanitized
	}
	if s, ok := m["connectionQuality"].(string); ok {
		sanitized := shared.SanitizeString(s, 20)
		user.ConnectionQuality = &sanitized
	}
	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
}

// handleSetRetention sets retention mode (48h/default).
func (h *Hub) handleSetRetention(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("setRetention", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}
	if user.Retention == nil {
		user.Retention = &cache.Retention{Mode: "default", ClientID: c.ID()}
	}
	validModes := map[string]bool{"default": true, "48h": true, "5d": true, "10d": true, "30d": true}
	if mode, ok := m["mode"].(string); ok && validModes[mode] {
		user.Retention.Mode = mode
	}
	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
}

// handleSetPrivacyPause pauses or resumes the user's location sharing.
// duration: "1h" | "4h" | "8h" → sets a pausedUntil timestamp.
// duration: "resume" → clears the pause.
func (h *Hub) handleSetPrivacyPause(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("setPrivacyPause", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}
	duration, _ := m["duration"].(string)
	now := time.Now()
	var pausedUntil *int64
	switch duration {
	case "1h":
		t := now.Add(1 * time.Hour).UnixMilli()
		pausedUntil = &t
	case "4h":
		t := now.Add(4 * time.Hour).UnixMilli()
		pausedUntil = &t
	case "8h":
		t := now.Add(8 * time.Hour).UnixMilli()
		pausedUntil = &t
	case "resume":
		pausedUntil = nil
	default:
		return
	}
	// Collect currently-visible sockets before invalidating, so we can tombstone them.
	prevVisibleSids := h.Cache.GetVisibleSocketIDs(user)
	user.PrivacyPausedUntil = pausedUntil
	h.invalidateVisibility(user.UserID)
	c.Send("privacyPauseUpdate", map[string]interface{}{"ok": true, "pausedUntil": pausedUntil})
	// Notify previously-visible users that this user is no longer sharing.
	if pausedUntil != nil {
		tombstone := map[string]interface{}{"socketId": user.SocketID, "privacyPaused": true}
		h.SendToClients(prevVisibleSids, "userPrivacyPaused", tombstone)
	}
}

// handleSetRetentionForever sets retention to forever (admin only).
func (h *Hub) handleSetRetentionForever(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("setRetentionForever", 10) {
		return
	}
	if c.Role() != "admin" {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}
	targetSocketID, _ := m["socketId"].(string)
	if targetSocketID == "" {
		return
	}

	target := h.Cache.GetActiveUser(targetSocketID)
	if target == nil {
		target = h.Cache.GetOfflineUserBySocketID(targetSocketID)
	}
	if target == nil {
		return
	}

	forever, _ := m["forever"].(bool)
	if target.Retention == nil {
		target.Retention = &cache.Retention{Mode: "default", ClientID: targetSocketID}
	}
	if forever {
		target.Retention.Mode = "forever"
	} else if target.Retention.Mode == "forever" {
		target.Retention.Mode = "default"
	}

	sanitized := h.Cache.SanitizeUser(target)
	sanitized["online"] = h.Cache.GetUserIdToSocketId(target.UserID) != ""
	h.emitToVisibleAndSelf(target, "userUpdate", sanitized)
}

// handleAdminDeleteUser force-deletes a user (admin only).
func (h *Hub) handleAdminDeleteUser(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("adminDeleteUser", 5) {
		return
	}
	if c.Role() != "admin" {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}
	targetSocketID, _ := m["socketId"].(string)
	if targetSocketID == "" {
		return
	}

	targetUser := h.Cache.GetActiveUser(targetSocketID)
	if targetUser != nil {
		targetUser.ForceDelete = true
		h.Cache.DeleteOfflineUser(targetUser.UserID)
		h.DisconnectClient(targetSocketID)
		visibleSids := h.Cache.GetVisibleSocketIDs(targetUser)
		for _, sid := range visibleSids {
			h.SendToClient(sid, "userDisconnect", targetSocketID)
		}
		slog.Info("Admin deleted user", "targetUserId", targetUser.UserID, "by", c.UserID())
		return
	}
	// Offline user with socketId - would need to search offline; skip for simplicity
}

// handleCreateRoom creates a new room.
func (h *Hub) handleCreateRoom(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("createRoom", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	if h.Cache.GetUserRoomCount(user.UserID) >= maxRoomsPerUser {
		c.Send("roomError", map[string]interface{}{"message": "Room limit reached (" + strconv.Itoa(maxRoomsPerUser) + ")"})
		return
	}
	m := toMap(data)
	name := ""
	if m != nil {
		if s, ok := m["name"].(string); ok {
			name = shared.SanitizeString(s, 50)
		}
	}
	code := h.generateUniqueRoomCode()
	roomName := name
	if roomName == "" {
		roomName = "Room " + code
	}
	createdAt := time.Now().UnixMilli()

	roomDbID, err := db.CreateRoom(context.Background(), h.pool.DB, code, roomName, user.UserID, createdAt)
	if err != nil {
		slog.Error("Failed to create room", "error", err)
		c.Send("roomError", map[string]interface{}{"message": "Failed to create room"})
		return
	}

	// Persist creator as admin member — must match cache.AddRoom which sets Members[createdBy]=admin.
	// Without this, the room_members table has no rows and the room appears empty after a restart.
	if err := db.AddRoomMember(context.Background(), h.pool.DB, roomDbID, user.UserID, "admin"); err != nil {
		slog.Error("Failed to add room creator as member", "error", err)
		_ = db.DeleteRoom(context.Background(), h.pool.DB, roomDbID)
		c.Send("roomError", map[string]interface{}{"message": "Failed to create room"})
		return
	}

	h.Cache.AddRoom(code, roomDbID, roomName, user.UserID, createdAt)
	user.Rooms = h.Cache.GetUserRooms(user.UserID)
	h.invalidateVisibility(user.UserID)

	c.Send("roomCreated", map[string]interface{}{"code": code, "name": roomName})
	h.emitMyRooms(c, user.UserID)
}

// generateUniqueRoomCode generates a unique 6-char room code.
func (h *Hub) generateUniqueRoomCode() string {
	for {
		code := shared.GenerateCode()
		if !h.Cache.ShareCodeExists(code) && h.Cache.GetRoom(code) == nil {
			return code
		}
	}
}

// handleJoinRoom joins a room by code.
func (h *Hub) handleJoinRoom(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("joinRoom", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	code := ""
	if m != nil {
		if s, ok := m["code"].(string); ok {
			code = strings.TrimSpace(strings.ToUpper(s))
		}
	}
	if code == "" {
		c.Send("roomError", map[string]interface{}{"message": "Invalid code"})
		return
	}

	room := h.Cache.GetRoom(code)
	if room == nil {
		c.Send("roomError", map[string]interface{}{"message": "Room not found"})
		return
	}

	if err := db.AddRoomMember(context.Background(), h.pool.DB, room.DbID, user.UserID, "member"); err != nil {
		slog.Error("Failed to add room member", "error", err)
		c.Send("roomError", map[string]interface{}{"message": "Failed to join"})
		return
	}

	h.Cache.AddRoomMember(code, user.UserID, "member")
	user.Rooms = h.Cache.GetUserRooms(user.UserID)

	memberIDs := make([]string, 0, len(room.Members))
	for mid := range room.Members {
		memberIDs = append(memberIDs, mid)
	}
	h.invalidateVisibilityForUsers(memberIDs)

	c.Send("roomJoined", map[string]interface{}{"code": code, "name": room.Name})
	h.emitMyRooms(c, user.UserID)
	h.scheduleVisibilityRefresh(c, user)
}

// handleLeaveRoom leaves a room.
func (h *Hub) handleLeaveRoom(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("leaveRoom", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	code := ""
	if m != nil {
		if s, ok := m["code"].(string); ok {
			code = strings.TrimSpace(strings.ToUpper(s))
		}
	}
	if code == "" {
		return
	}

	room := h.Cache.GetRoom(code)
	if room == nil {
		return
	}

	if err := db.RemoveRoomMember(context.Background(), h.pool.DB, room.DbID, user.UserID); err != nil {
		slog.Error("Failed to remove room member", "error", err)
		return
	}

	h.Cache.RemoveRoomMember(code, user.UserID)
	if len(room.Members) <= 1 {
		_ = db.DeleteRoom(context.Background(), h.pool.DB, room.DbID)
		h.Cache.DeleteRoom(code)
	}

	user.Rooms = h.Cache.GetUserRooms(user.UserID)
	memberIDs := make([]string, 0, len(room.Members))
	for mid := range room.Members {
		memberIDs = append(memberIDs, mid)
	}
	memberIDs = append(memberIDs, user.UserID)
	h.invalidateVisibilityForUsers(memberIDs)

	c.Send("roomLeft", map[string]interface{}{"code": code})
	h.emitMyRooms(c, user.UserID)
	h.scheduleVisibilityRefresh(c, user)
}

// handleAddContact adds a contact by share code.
func (h *Hub) handleAddContact(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("addContact", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	if h.Cache.GetContactCount(user.UserID) >= maxContactsPerUser {
		c.Send("contactError", map[string]interface{}{"message": "Contact limit reached"})
		return
	}
	m := toMap(data)
	shareCode := ""
	if m != nil {
		if s, ok := m["shareCode"].(string); ok {
			shareCode = strings.TrimSpace(strings.ToUpper(s))
		}
	}
	if shareCode == "" {
		c.Send("contactError", map[string]interface{}{"message": "Invalid share code"})
		return
	}

	targetID := h.Cache.GetUserIDByShareCode(shareCode)
	if targetID == "" {
		// Cache miss: fall back to DB (handles users registered after last cache load)
		var err error
		targetID, err = db.GetUserIDByShareCode(context.Background(), h.pool.DB, shareCode)
		if err != nil || targetID == "" {
			c.Send("contactError", map[string]interface{}{"message": "No user found with that code"})
			return
		}
		// Warm the cache so future lookups hit memory
		h.Cache.WarmShareCode(shareCode, targetID)
	}
	if targetID == user.UserID {
		c.Send("contactError", map[string]interface{}{"message": "That's your own code — share it with someone else to connect"})
		return
	}

	// Already contacts? Tell them explicitly instead of silently doing nothing.
	if h.Cache.AreContacts(user.UserID, targetID) {
		c.Send("contactError", map[string]interface{}{"message": "You're already connected with this person"})
		return
	}

	if err := db.AddContactBidirectional(context.Background(), h.pool.DB, user.UserID, targetID); err != nil {
		slog.Error("Failed to add contact", "error", err)
		c.Send("contactError", map[string]interface{}{"message": "Failed to add contact"})
		return
	}

	h.Cache.AddContactBidirectional(user.UserID, targetID)
	h.invalidateVisibilityForUsers([]string{user.UserID, targetID})

	c.Send("contactAdded", map[string]interface{}{"userId": targetID, "displayName": h.Cache.GetDisplayName(targetID)})
	h.emitMyContacts(c, user.UserID)
	h.scheduleVisibilityRefresh(c, user)

	if other := h.GetClientByUserID(targetID); other != nil {
		ou := h.Cache.GetActiveUser(other.ID())
		if ou != nil {
			h.emitMyContacts(other, targetID)
			h.scheduleVisibilityRefresh(other, ou)
		}
	}
}

// handleRemoveContact removes a contact.
func (h *Hub) handleRemoveContact(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("removeContact", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	targetID := ""
	if m != nil {
		if s, ok := m["userId"].(string); ok {
			targetID = strings.TrimSpace(s)
		}
	}
	if targetID == "" {
		return
	}

	if err := db.RemoveContactBidirectional(context.Background(), h.pool.DB, user.UserID, targetID); err != nil {
		slog.Error("Failed to remove contact", "error", err)
		return
	}

	h.Cache.RemoveContactBidirectional(user.UserID, targetID)
	h.invalidateVisibilityForUsers([]string{user.UserID, targetID})

	c.Send("contactRemoved", map[string]interface{}{"userId": targetID})
	h.emitMyContacts(c, user.UserID)
	h.scheduleVisibilityRefresh(c, user)

	if other := h.GetClientByUserID(targetID); other != nil {
		ou := h.Cache.GetActiveUser(other.ID())
		if ou != nil {
			h.emitMyContacts(other, targetID)
			h.scheduleVisibilityRefresh(other, ou)
		}
	}
}

// handleCreateLiveLink creates a live sharing link.
func (h *Hub) handleCreateLiveLink(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("createLiveLink", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	if h.Cache.GetLiveTokenCount(user.UserID) >= maxLiveLinksPerUser {
		c.Send("liveLinkError", map[string]interface{}{"message": "Live link limit reached"})
		return
	}
	m := toMap(data)
	expStr := "24h"
	if m != nil {
		// v1/v2-style payload
		if v, ok := m["expiresIn"]; ok {
			if s, ok := v.(string); ok {
				expStr = strings.TrimSpace(s)
			} else {
				// explicit null means "forever"
				expStr = ""
			}
		}
		// frontend currently sends "duration"
		if v, ok := m["duration"]; ok {
			if s, ok := v.(string); ok {
				expStr = strings.TrimSpace(s)
			} else {
				// explicit null means "forever"
				expStr = ""
			}
		}
	}

	expiresAt := h.parseExpiresIn(expStr)
	token := generateLiveToken()
	createdAt := time.Now().UnixMilli()

	if err := db.CreateLiveToken(context.Background(), h.pool.DB, token, user.UserID, expiresAt, createdAt); err != nil {
		slog.Error("Failed to create live token", "error", err)
		c.Send("liveLinkError", map[string]interface{}{"message": "Failed to create link"})
		return
	}

	h.Cache.AddLiveToken(token, user.UserID, expiresAt, createdAt)
	c.Send("liveLinkCreated", map[string]interface{}{"token": token, "expiresAt": expiresAt})
	h.emitMyLiveLinks(c, user.UserID)
}

func generateLiveToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// handleRevokeLiveLink revokes a live link.
func (h *Hub) handleRevokeLiveLink(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("revokeLiveLink", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	token := ""
	if m != nil {
		if s, ok := m["token"].(string); ok {
			token = strings.TrimSpace(s)
		}
	}
	if token == "" {
		return
	}

	entry := h.Cache.GetLiveToken(token)
	if entry == nil || entry.UserID != user.UserID {
		return
	}

	_ = db.DeleteLiveToken(context.Background(), h.pool.DB, token)
	h.Cache.DeleteLiveToken(token)
	h.SendToGroup("live:"+token, "liveExpired", map[string]interface{}{"message": "Link revoked"})
	c.Send("liveLinkRevoked", map[string]interface{}{"token": token})
	h.emitMyLiveLinks(c, user.UserID)
}

// handleSendPulse broadcasts a "I'm OK" or "Need help, call me" pulse to visible users.
// Ephemeral — no DB write.
func (h *Hub) handleSendPulse(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("sendPulse", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	pulseType := "ok"
	if t, ok := m["type"].(string); ok && (t == "ok" || t == "callme") {
		pulseType = t
	}
	expiresAt := time.Now().UnixMilli() + 30000 // 30s
	payload := map[string]interface{}{
		"userId":      user.UserID,
		"displayName": user.DisplayName,
		"type":        pulseType,
		"lat":         user.Latitude,
		"lng":         user.Longitude,
		"expiresAt":   expiresAt,
	}
	h.emitToVisibleAndSelf(user, "pulseReceived", payload)
}

// handleIAmSafe broadcasts an "I'm safe" signal to all visible users and guardians. (F2)
// Ephemeral — no DB write. Rate-limited to 10/min.
func (h *Hub) handleIAmSafe(c *Client, _ json.RawMessage) {
	if !c.CheckRateLimit("iAmSafe", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	payload := map[string]interface{}{
		"userId":      user.UserID,
		"displayName": user.DisplayName,
		"lat":         user.Latitude,
		"lng":         user.Longitude,
		"at":          time.Now().UnixMilli(),
	}
	h.emitToVisibleAndSelf(user, "iAmSafe", payload)
}

// handleSetMeetingPoint sets a meeting point pin for all members of a room. (F3)
// Payload: { roomCode, lat, lng, label }
func (h *Hub) handleSetMeetingPoint(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("setMeetingPoint", 20) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}
	roomCode, _ := m["roomCode"].(string)
	roomCode = strings.TrimSpace(strings.ToUpper(roomCode))
	if roomCode == "" {
		return
	}
	room := h.Cache.GetRoom(roomCode)
	if room == nil || !room.Members[user.UserID] {
		c.Send("roomError", map[string]interface{}{"message": "Not a member of that room"})
		return
	}
	lat, latOK := m["lat"].(float64)
	lng, lngOK := m["lng"].(float64)
	if !latOK || !lngOK {
		return
	}
	label := ""
	if v, ok := m["label"].(string); ok {
		label = shared.SanitizeString(v, 80)
	}
	now := time.Now().UnixMilli()

	// Update in-memory cache
	room.MeetingLat = &lat
	room.MeetingLng = &lng
	room.MeetingLabel = label
	room.MeetingSetBy = user.UserID
	room.MeetingSetAt = now

	// Persist to DB async
	go func() {
		if err := db.SetRoomMeetingPoint(context.Background(), h.pool.DB, roomCode, lat, lng, label, user.UserID, now); err != nil {
			slog.Warn("SetRoomMeetingPoint failed", "roomCode", roomCode, "error", err)
		}
	}()

	// Broadcast to all online room members
	payload := map[string]interface{}{
		"roomCode":  roomCode,
		"lat":       lat,
		"lng":       lng,
		"label":     label,
		"setBy":     user.UserID,
		"setByName": user.DisplayName,
		"setAt":     now,
	}
	for mid := range room.Members {
		if cli := h.GetClientByUserID(mid); cli != nil {
			h.SendToClient(cli.ID(), "meetingPointUpdated", payload)
		}
	}
}

// handleClearMeetingPoint removes the meeting point pin from a room. (F3)
// Payload: { roomCode }
func (h *Hub) handleClearMeetingPoint(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("clearMeetingPoint", 20) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}
	roomCode, _ := m["roomCode"].(string)
	roomCode = strings.TrimSpace(strings.ToUpper(roomCode))
	if roomCode == "" {
		return
	}
	room := h.Cache.GetRoom(roomCode)
	if room == nil || !room.Members[user.UserID] {
		return
	}

	// Clear in-memory cache
	room.MeetingLat = nil
	room.MeetingLng = nil
	room.MeetingLabel = ""
	room.MeetingSetBy = ""
	room.MeetingSetAt = 0

	// Persist to DB async
	go func() {
		if err := db.ClearRoomMeetingPoint(context.Background(), h.pool.DB, roomCode); err != nil {
			slog.Warn("ClearRoomMeetingPoint failed", "roomCode", roomCode, "error", err)
		}
	}()

	payload := map[string]interface{}{"roomCode": roomCode, "cleared": true}
	for mid := range room.Members {
		if cli := h.GetClientByUserID(mid); cli != nil {
			h.SendToClient(cli.ID(), "meetingPointUpdated", payload)
		}
	}
}

// handleSetSpeedAlert sets or clears the speed alert threshold for the caller. (F5)
// Payload: { thresholdKmh: float64 } — value in km/h from the frontend; 0 or absent = disable.
// The threshold is converted to m/s internally and stored as speed_alert_threshold_ms.
func (h *Hub) handleSetSpeedAlert(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("setSpeedAlert", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	threshold := float64(0)
	if v, ok := m["thresholdKmh"].(float64); ok && v > 0 {
		threshold = v / 3.6 // convert km/h → m/s for internal storage
	}
	user.SpeedAlertThresholdMs = threshold

	// Persist to DB async
	go func() {
		_, err := h.pool.DB.ExecContext(context.Background(),
			`UPDATE users SET speed_alert_threshold_ms=$1 WHERE id=$2`,
			threshold, user.UserID)
		if err != nil {
			slog.Warn("handleSetSpeedAlert: DB persist failed", "userId", user.UserID, "error", err)
		}
	}()

	c.Send("speedAlertSet", map[string]interface{}{
		"thresholdMs": threshold,
		"enabled":     threshold > 0,
	})
}

// handleQuietHoursUpdate sets or clears the user's quiet hours window.
// During quiet hours, position broadcasts to non-guardian contacts are jittered ±500m.
func (h *Hub) handleQuietHoursUpdate(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("quietHoursUpdate", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}
	enabled, _ := m["enabled"].(bool)
	start, _ := m["startTime"].(string) // "HH:MM"
	end, _ := m["endTime"].(string)     // "HH:MM"

	user.QuietHoursEnabled = enabled
	user.QuietHoursStart = start
	user.QuietHoursEnd = end

	// Persist to DB (best-effort)
	go func() {
		var startVal, endVal interface{}
		if enabled && start != "" {
			startVal = start
		}
		if enabled && end != "" {
			endVal = end
		}
		_, err := h.pool.DB.ExecContext(context.Background(),
			`UPDATE users SET quiet_hours_enabled=$1, quiet_hours_start=$2::time, quiet_hours_end=$3::time WHERE id=$4`,
			enabled, startVal, endVal, user.UserID)
		if err != nil {
			slog.Warn("Failed to persist quiet hours", "error", err)
		}
	}()

	c.Send("quietHoursUpdated", map[string]interface{}{
		"userId":  user.UserID,
		"enabled": enabled,
		"active":  enabled && isQuietHoursNow(start, end),
	})
}

// isQuietHoursNow returns true if the current UTC time falls in [startHHMM, endHHMM).
// Start and end are "HH:MM" strings in 24h format (user is expected to send UTC equivalents).
func isQuietHoursNow(start, end string) bool {
	if start == "" || end == "" {
		return false
	}
	now := time.Now().UTC()
	nowMin := now.Hour()*60 + now.Minute()
	parse := func(s string) int {
		var h, m int
		fmt.Sscanf(s, "%d:%d", &h, &m)
		return h*60 + m
	}
	s := parse(start)
	e := parse(end)
	if s <= e {
		return nowMin >= s && nowMin < e
	}
	// Overnight window (e.g. 22:00 – 07:00)
	return nowMin >= s || nowMin < e
}

// applyPrivacyJitter adds a random offset of up to ±444m to lat/lng.
func applyPrivacyJitter(lat, lng float64) (float64, float64) {
	// 0.004 deg ≈ 444m at equator
	jitterDeg := 0.004
	jLat := (mathrand.Float64()*2 - 1) * jitterDeg
	cosLat := math.Cos(lat * math.Pi / 180)
	if cosLat < 0.1 {
		cosLat = 0.1 // avoid division by near-zero at poles
	}
	jLng := (mathrand.Float64()*2 - 1) * jitterDeg / cosLat
	return lat + jLat, lng + jLng
}

// handleUpdateSchedule sets sharing schedule rules for the authenticated user.
func (h *Hub) handleUpdateSchedule(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("updateSchedule", 5) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	type ruleIn struct {
		TargetType string `json:"targetType"`
		TargetID   string `json:"targetId"`
		DayMask    int    `json:"dayMask"`
		StartTime  string `json:"startTime"` // "HH:MM"
		EndTime    string `json:"endTime"`   // "HH:MM"
		Enabled    bool   `json:"enabled"`
	}
	var payload struct {
		Rules []ruleIn `json:"rules"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}
	parseMin := func(s string) int {
		var hr, min int
		fmt.Sscanf(s, "%d:%d", &hr, &min)
		return hr*60 + min
	}

	rules := make([]cache.ScheduleRule, 0, len(payload.Rules))
	for _, r := range payload.Rules {
		rules = append(rules, cache.ScheduleRule{
			TargetType: r.TargetType,
			TargetID:   r.TargetID,
			DayMask:    r.DayMask,
			StartMin:   parseMin(r.StartTime),
			EndMin:     parseMin(r.EndTime),
			Enabled:    r.Enabled,
		})
	}

	h.Cache.SetSharingSchedules(user.UserID, rules)

	// Persist to DB in background
	go func(uid string, ruleList []cache.ScheduleRule) {
		ctx := context.Background()
		if _, err := h.pool.DB.ExecContext(ctx, `DELETE FROM sharing_schedules WHERE user_id = $1`, uid); err != nil {
			slog.Error("Failed to delete sharing schedules", "error", err, "userId", uid)
		}
		for _, r := range ruleList {
			if _, err := h.pool.DB.ExecContext(ctx,
				`INSERT INTO sharing_schedules (user_id, target_type, target_id, day_mask, start_time, end_time, enabled, created_at)
				 VALUES ($1, $2, NULLIF($3,'')::uuid, $4, $5::time, $6::time, $7, $8)`,
				uid, r.TargetType, r.TargetID, r.DayMask,
				fmt.Sprintf("%02d:%02d", r.StartMin/60, r.StartMin%60),
				fmt.Sprintf("%02d:%02d", r.EndMin/60, r.EndMin%60),
				r.Enabled, time.Now().UnixMilli()); err != nil {
				slog.Error("Failed to persist sharing schedule", "error", err, "userId", uid)
			}
		}
	}(user.UserID, rules)

	c.Send("scheduleUpdated", map[string]interface{}{"ok": true, "count": len(rules)})
}

// handleWatchJoin joins watch:token group and sends watchInit.
func (h *Hub) handleWatchJoin(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("watchJoin", 10) {
		return
	}
	m := toMap(data)
	token := ""
	if m != nil {
		if s, ok := m["token"].(string); ok {
			token = strings.TrimSpace(s)
		}
	}
	if token == "" {
		return
	}

	entry := h.Cache.GetWatchToken(token)
	if entry == nil {
		c.Send("watchError", map[string]interface{}{"message": "Invalid or expired token"})
		return
	}

	h.JoinGroup(c.ID(), "watch:"+token)
	target := h.Cache.GetActiveUser(entry.SocketID)
	if target == nil {
		// Target offline - send minimal init
		c.Send("watchInit", map[string]interface{}{"userId": entry.UserID})
		return
	}
	c.Send("watchInit", h.Cache.SanitizeUser(target))
}

// handleLiveJoin joins live:token group and sends liveInit.
func (h *Hub) handleLiveJoin(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("liveJoin", 10) {
		return
	}
	m := toMap(data)
	token := ""
	viewerName := "Viewer"
	if m != nil {
		if s, ok := m["token"].(string); ok {
			token = strings.TrimSpace(s)
		}
		if s, ok := m["viewerName"].(string); ok {
			viewerName = shared.SanitizeString(s, 50)
		}
	}
	if token == "" {
		return
	}

	entry := h.Cache.GetLiveToken(token)
	if entry == nil {
		c.Send("liveError", map[string]interface{}{"message": "Invalid or expired link"})
		return
	}

	h.JoinGroup(c.ID(), "live:"+token)
	c.liveToken = token
	c.liveViewerName = viewerName

	target := h.Cache.GetActiveUser(h.Cache.GetUserIdToSocketId(entry.UserID))
	if target == nil {
		c.Send("liveInit", map[string]interface{}{"userId": entry.UserID})
		return
	}
	c.Send("liveInit", map[string]interface{}{"user": h.Cache.SanitizeUser(target)})
}

// handleRequestAdminOverview sends full admin overview (admin only).
func (h *Hub) handleRequestAdminOverview(c *Client, data json.RawMessage) {
	if c.Role() != "admin" {
		return
	}
	h.emitAdminOverview(c)
}

// handleRequestRoomAdmin requests room admin role.
func (h *Hub) handleRequestRoomAdmin(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("requestRoomAdmin", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	code := ""
	expiresIn := "7d"
	if m != nil {
		if s, ok := m["roomCode"].(string); ok {
			code = strings.TrimSpace(strings.ToUpper(s))
		}
		if s, ok := m["expiresIn"].(string); ok {
			expiresIn = s
		}
	}
	if code == "" {
		return
	}

	room := h.Cache.GetRoom(code)
	if room == nil || !room.Members[user.UserID] {
		return
	}

	expPtr := &expiresIn
	createdAt := time.Now().UnixMilli()
	entry := &db.RoomAdminRequestEntry{Type: "roomAdmin", From: user.UserID, RoomCode: code, ExpiresIn: expPtr, CreatedAt: createdAt, Approvals: make(map[string]bool), Denials: make(map[string]bool)}

	if err := db.CreateRoomAdminRequest(context.Background(), h.pool.DB, code, user.UserID, expPtr, createdAt); err != nil {
		return
	}
	h.Cache.AddRoomAdminRequest(entry)

	for mid := range room.Members {
		if cli := h.GetClientByUserID(mid); cli != nil {
			h.emitMyRooms(cli, mid)
		}
	}
}

// handleVoteRoomAdmin votes on room admin request.
func (h *Hub) handleVoteRoomAdmin(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("voteRoomAdmin", 20) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	roomCode, targetUserID, vote := "", "", ""
	if m != nil {
		if s, ok := m["roomCode"].(string); ok {
			roomCode = strings.TrimSpace(strings.ToUpper(s))
		}
		if s, ok := m["userId"].(string); ok {
			targetUserID = strings.TrimSpace(s)
		}
		if s, ok := m["vote"].(string); ok {
			vote = strings.TrimSpace(s)
		}
	}
	if roomCode == "" || targetUserID == "" || (vote != "approve" && vote != "deny") {
		return
	}

	room := h.Cache.GetRoom(roomCode)
	if room == nil || !room.Members[user.UserID] {
		return
	}

	_ = db.UpsertRoomAdminVote(context.Background(), h.pool.DB, roomCode, targetUserID, user.UserID, vote)
	h.Cache.AddRoomAdminVote(roomCode, targetUserID, user.UserID, vote)

	reqs := h.Cache.GetRoomAdminRequests(roomCode)
	var targetReq *db.RoomAdminRequestEntry
	for _, r := range reqs {
		if r.From == targetUserID {
			targetReq = r
			break
		}
	}
	if targetReq == nil {
		return
	}

	totalEligible := len(room.Members) - 1
	if totalEligible <= 0 {
		return
	}
	majority := totalEligible/2 + 1

	if len(targetReq.Approvals) >= majority {
		expStr := ""
		if targetReq.ExpiresIn != nil {
			expStr = *targetReq.ExpiresIn
		}
		expiresAt := h.parseExpiresIn(expStr)
		h.Cache.SetRoomMemberRole(roomCode, targetUserID, "admin", expiresAt)
		_ = db.SetRoomMemberRole(context.Background(), h.pool.DB, room.DbID, targetUserID, "admin", expiresAt)
		_ = db.DeleteRoomAdminRequest(context.Background(), h.pool.DB, roomCode, targetUserID)
		h.Cache.RemoveRoomAdminRequest(roomCode, targetUserID)
		for mid := range room.Members {
			if cli := h.GetClientByUserID(mid); cli != nil {
				h.SendToClient(cli.ID(), "roomAdminUpdated", map[string]interface{}{"roomCode": roomCode, "userId": targetUserID, "role": "admin", "expiresAt": expiresAt})
				h.emitMyRooms(cli, mid)
			}
		}
	} else if len(targetReq.Denials) >= majority {
		_ = db.DeleteRoomAdminRequest(context.Background(), h.pool.DB, roomCode, targetUserID)
		h.Cache.RemoveRoomAdminRequest(roomCode, targetUserID)
		for mid := range room.Members {
			if cli := h.GetClientByUserID(mid); cli != nil {
				h.SendToClient(cli.ID(), "roomAdminUpdated", map[string]interface{}{"roomCode": roomCode, "userId": targetUserID, "role": "denied", "expiresAt": nil})
				h.emitMyRooms(cli, mid)
			}
		}
	} else {
		for mid := range room.Members {
			if cli := h.GetClientByUserID(mid); cli != nil {
				h.emitMyRooms(cli, mid)
			}
		}
	}
}

// handleRevokeRoomAdmin revokes room admin role.
func (h *Hub) handleRevokeRoomAdmin(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("revokeRoomAdmin", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	roomCode, targetUserID := "", ""
	if m != nil {
		if s, ok := m["roomCode"].(string); ok {
			roomCode = strings.TrimSpace(strings.ToUpper(s))
		}
		if s, ok := m["userId"].(string); ok {
			targetUserID = strings.TrimSpace(s)
		}
	}
	if roomCode == "" || targetUserID == "" {
		return
	}

	room := h.Cache.GetRoom(roomCode)
	if room == nil || !room.Members[targetUserID] {
		return
	}

	actorRole := h.Cache.GetRoomMemberRole(roomCode, user.UserID)
	isSelf := user.UserID == targetUserID
	if !isSelf && (actorRole == nil || actorRole.Role != "admin") {
		return
	}

	h.Cache.SetRoomMemberRole(roomCode, targetUserID, "member", nil)
	_ = db.SetRoomMemberRole(context.Background(), h.pool.DB, room.DbID, targetUserID, "member", nil)

	for mid := range room.Members {
		if cli := h.GetClientByUserID(mid); cli != nil {
			h.SendToClient(cli.ID(), "roomAdminUpdated", map[string]interface{}{"roomCode": roomCode, "userId": targetUserID, "role": "member", "expiresAt": nil})
		}
	}
}

// handleRequestGuardian requests guardian role (guardian initiates).
func (h *Hub) handleRequestGuardian(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("requestGuardian", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	wardID := ""
	expiresIn := ""
	if m != nil {
		if s, ok := m["contactUserId"].(string); ok {
			wardID = strings.TrimSpace(s)
		}
		if s, ok := m["expiresIn"].(string); ok {
			expiresIn = s
		}
	}
	if wardID == "" || wardID == user.UserID {
		return
	}

	myContacts := h.Cache.GetContactsForUser(user.UserID)
	theirContacts := h.Cache.GetContactsForUser(wardID)
	hasMutual := false
	for _, cid := range myContacts {
		if cid == wardID {
			for _, tid := range theirContacts {
				if tid == user.UserID {
					hasMutual = true
					break
				}
			}
			break
		}
	}
	if !hasMutual {
		c.Send("contactError", map[string]interface{}{"message": "Both must be mutual contacts to request guardian role"})
		return
	}

	existing := h.Cache.GetGuardianship(user.UserID, wardID)
	if existing != nil && (existing.Status == "active" || existing.Status == "pending") {
		c.Send("contactError", map[string]interface{}{"message": "Request already pending"})
		return
	}

	// KR-012: store the real expiry timestamp so it can be enforced, not just displayed.
	pendingExpiresAt := h.parseExpiresIn(expiresIn)
	entry := &db.GuardianshipEntry{Status: "pending", InitiatedBy: "guardian", ExpiresAt: pendingExpiresAt, CreatedAt: time.Now().UnixMilli()}
	h.Cache.SetGuardianship(user.UserID, wardID, entry)
	_ = db.CreateGuardianship(context.Background(), h.pool.DB, user.UserID, wardID, "pending", pendingExpiresAt, entry.CreatedAt, "guardian")

	h.Cache.AddPendingRequest(wardID+":guardian", map[string]interface{}{"type": "guardian", "from": user.UserID, "expiresIn": expiresIn})

	if wardCli := h.GetClientByUserID(wardID); wardCli != nil {
		wardCli.Send("guardianRequest", map[string]interface{}{"fromUserId": user.UserID, "fromName": user.DisplayName, "expiresIn": expiresIn, "initiatedBy": "guardian"})
		h.emitMyGuardians(wardCli, wardID)
		h.emitPendingRequests(wardCli, wardID)
	}
	// KR-002: use guardianInfo (not contactError) so success isn't rendered as an error banner.
	c.Send("guardianInfo", map[string]interface{}{"message": "Guardian request sent"})
	h.emitMyGuardians(c, user.UserID)
}

// handleInviteGuardian invites someone to be guardian (ward initiates).
func (h *Hub) handleInviteGuardian(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("inviteGuardian", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	guardianID := ""
	expiresIn := ""
	if m != nil {
		if s, ok := m["contactUserId"].(string); ok {
			guardianID = strings.TrimSpace(s)
		}
		if s, ok := m["expiresIn"].(string); ok {
			expiresIn = s
		}
	}
	if guardianID == "" || guardianID == user.UserID {
		return
	}

	myContacts := h.Cache.GetContactsForUser(user.UserID)
	theirContacts := h.Cache.GetContactsForUser(guardianID)
	hasMutual := false
	for _, cid := range myContacts {
		if cid == guardianID {
			for _, tid := range theirContacts {
				if tid == user.UserID {
					hasMutual = true
					break
				}
			}
			break
		}
	}
	if !hasMutual {
		c.Send("contactError", map[string]interface{}{"message": "Both must be mutual contacts"})
		return
	}

	existing := h.Cache.GetGuardianship(guardianID, user.UserID)
	if existing != nil && (existing.Status == "active" || existing.Status == "pending") {
		c.Send("contactError", map[string]interface{}{"message": "Request already pending"})
		return
	}

	// KR-012: store the real expiry timestamp so it can be enforced, not just displayed.
	pendingExpiresAt := h.parseExpiresIn(expiresIn)
	entry := &db.GuardianshipEntry{Status: "pending", InitiatedBy: "ward", ExpiresAt: pendingExpiresAt, CreatedAt: time.Now().UnixMilli()}
	h.Cache.SetGuardianship(guardianID, user.UserID, entry)
	_ = db.CreateGuardianship(context.Background(), h.pool.DB, guardianID, user.UserID, "pending", pendingExpiresAt, entry.CreatedAt, "ward")

	h.Cache.AddPendingRequest(guardianID+":guardianInvite", map[string]interface{}{"type": "guardianInvite", "from": user.UserID, "expiresIn": expiresIn})

	if gCli := h.GetClientByUserID(guardianID); gCli != nil {
		gCli.Send("guardianInvite", map[string]interface{}{"fromUserId": user.UserID, "fromName": user.DisplayName, "expiresIn": expiresIn, "initiatedBy": "ward"})
		h.emitMyGuardians(gCli, guardianID)
		h.emitPendingRequests(gCli, guardianID)
	}
	// KR-002: use guardianInfo (not contactError) so success isn't rendered as an error banner.
	c.Send("guardianInfo", map[string]interface{}{"message": "Guardian invite sent"})
	h.emitMyGuardians(c, user.UserID)
}

// handleApproveGuardian approves a guardianship.
func (h *Hub) handleApproveGuardian(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("approveGuardian", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	guardianID, _ := m["guardianUserId"].(string)
	wardID, _ := m["wardUserId"].(string)
	if guardianID != "" {
		guardianID = strings.TrimSpace(guardianID)
	}
	if wardID != "" {
		wardID = strings.TrimSpace(wardID)
	}

	var gID, wID, pendingKey string
	if guardianID != "" {
		gID = guardianID
		wID = user.UserID
		pendingKey = user.UserID + ":guardian"
	} else if wardID != "" {
		gID = user.UserID
		wID = wardID
		pendingKey = user.UserID + ":guardianInvite"
	} else {
		return
	}

	entry := h.Cache.GetGuardianship(gID, wID)
	if entry == nil || entry.Status != "pending" {
		return
	}

	expiresAt := h.parseExpiresIn("")
	if payloadExp, ok := m["expiresIn"].(string); ok {
		expiresAt = h.parseExpiresIn(payloadExp)
	}
	reqs := h.Cache.GetPendingRequests(pendingKey)
	fromID := gID
	if guardianID != "" {
		fromID = guardianID
	} else {
		fromID = wardID
	}
	for _, r := range reqs {
		if m2, ok := r.(map[string]interface{}); ok {
			if f, _ := m2["from"].(string); f == fromID {
				if e, _ := m2["expiresIn"].(string); e != "" {
					expiresAt = h.parseExpiresIn(e)
				}
				break
			}
		}
	}
	h.Cache.RemovePendingRequestByFrom(pendingKey, fromID)

	entry.Status = "active"
	entry.ExpiresAt = expiresAt
	h.Cache.SetGuardianship(gID, wID, &db.GuardianshipEntry{Status: "active", InitiatedBy: entry.InitiatedBy, ExpiresAt: expiresAt, CreatedAt: entry.CreatedAt})
	_ = db.CreateGuardianship(context.Background(), h.pool.DB, gID, wID, "active", expiresAt, entry.CreatedAt, entry.InitiatedBy)

	h.invalidateVisibilityForUsers([]string{gID, wID})

	payload := map[string]interface{}{"guardianId": gID, "wardId": wID, "status": "active", "expiresAt": expiresAt}
	c.Send("guardianUpdated", payload)
	h.emitMyGuardians(c, user.UserID)
	h.emitPendingRequests(c, user.UserID)

	otherID := wID
	if user.UserID == wID {
		otherID = gID
	}
	if otherCli := h.GetClientByUserID(otherID); otherCli != nil {
		otherCli.Send("guardianUpdated", payload)
		h.emitMyGuardians(otherCli, otherID)
		h.emitPendingRequests(otherCli, otherID)
	}
}

// handleDenyGuardian denies a guardianship.
func (h *Hub) handleDenyGuardian(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("denyGuardian", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	guardianID, _ := m["guardianUserId"].(string)
	wardID, _ := m["wardUserId"].(string)
	if guardianID != "" {
		guardianID = strings.TrimSpace(guardianID)
	}
	if wardID != "" {
		wardID = strings.TrimSpace(wardID)
	}

	var gID, wID, pendingKey string
	if guardianID != "" {
		gID = guardianID
		wID = user.UserID
		pendingKey = user.UserID + ":guardian"
	} else if wardID != "" {
		gID = user.UserID
		wID = wardID
		pendingKey = user.UserID + ":guardianInvite"
	} else {
		return
	}

	entry := h.Cache.GetGuardianship(gID, wID)
	if entry == nil || entry.Status != "pending" {
		return
	}

	h.Cache.DeleteGuardianship(gID, wID)
	fromID := guardianID
	if fromID == "" {
		fromID = wardID
	}
	h.Cache.RemovePendingRequestByFrom(pendingKey, fromID)
	if guardianID != "" {
		h.Cache.RemovePendingRequestByFrom(wID+":guardianInvite", user.UserID)
	} else {
		h.Cache.RemovePendingRequestByFrom(gID+":guardian", user.UserID)
	}
	_ = db.UpdateGuardianshipStatus(context.Background(), h.pool.DB, gID, wID, "revoked")

	h.invalidateVisibilityForUsers([]string{gID, wID})

	payload := map[string]interface{}{"guardianId": gID, "wardId": wID, "status": "denied", "expiresAt": nil}
	c.Send("guardianUpdated", payload)
	h.emitMyGuardians(c, user.UserID)
	h.emitPendingRequests(c, user.UserID)

	otherID := wID
	if user.UserID == wID {
		otherID = gID
	}
	if otherCli := h.GetClientByUserID(otherID); otherCli != nil {
		otherCli.Send("guardianUpdated", payload)
		h.emitMyGuardians(otherCli, otherID)
		h.emitPendingRequests(otherCli, otherID)
	}
}

// handleRevokeGuardian revokes a guardianship.
func (h *Hub) handleRevokeGuardian(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("revokeGuardian", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	guardianID, _ := m["guardianUserId"].(string)
	wardID, _ := m["wardUserId"].(string)
	if guardianID != "" {
		guardianID = strings.TrimSpace(guardianID)
	}
	if wardID != "" {
		wardID = strings.TrimSpace(wardID)
	}

	var gID, wID string
	if guardianID != "" {
		gID = guardianID
		wID = user.UserID
	} else if wardID != "" {
		gID = user.UserID
		wID = wardID
	} else {
		return
	}

	entry := h.Cache.GetGuardianship(gID, wID)
	if entry == nil || (entry.Status != "active" && entry.Status != "pending") {
		return
	}

	if entry.Status == "active" && user.UserID != gID {
		c.Send("error", map[string]interface{}{"message": "Only the guardian can revoke an active guardianship."})
		return
	}
	if entry.Status == "pending" {
		isInitiator := (entry.InitiatedBy == "guardian" && user.UserID == gID) || (entry.InitiatedBy == "ward" && user.UserID == wID)
		if !isInitiator {
			c.Send("error", map[string]interface{}{"message": "Only the requester can cancel a pending guardianship request."})
			return
		}
	}

	h.Cache.DeleteGuardianship(gID, wID)
	h.Cache.RemovePendingRequestByFrom(wID+":guardian", gID)
	h.Cache.RemovePendingRequestByFrom(gID+":guardianInvite", wID)
	_ = db.UpdateGuardianshipStatus(context.Background(), h.pool.DB, gID, wID, "revoked")

	h.invalidateVisibilityForUsers([]string{gID, wID})

	payload := map[string]interface{}{"guardianId": gID, "wardId": wID, "status": "revoked", "expiresAt": nil}
	c.Send("guardianUpdated", payload)
	h.emitMyGuardians(c, user.UserID)
	h.emitPendingRequests(c, user.UserID)

	otherID := wID
	if user.UserID == wID {
		otherID = gID
	}
	if otherCli := h.GetClientByUserID(otherID); otherCli != nil {
		otherCli.Send("guardianUpdated", payload)
		h.emitMyGuardians(otherCli, otherID)
		h.emitPendingRequests(otherCli, otherID)
	}
}

// ── Consumer product features ──────────────────────────────────────────────────

// computeActivityContext derives a human-readable activity string from motion class.
func computeActivityContext(userID, mClass string) string {
	switch mClass {
	case "walk":
		return "Walking"
	case "run":
		return "Running"
	case "vehicle":
		return "In Transit"
	default:
		return ""
	}
}

// checkBatteryAlerts sends batteryProxyAlert to visible users when battery crosses
// thresholds 20/10/5%. Rate-limited to once per threshold per hour.
func checkBatteryAlerts(h *Hub, user *cache.ActiveUser) {
	if user.BatteryPct == nil {
		return
	}
	pct := *user.BatteryPct
	thresholds := []int{20, 10, 5}
	now := time.Now().UnixMilli()
	for _, t := range thresholds {
		if pct <= t {
			if now-user.BatteryAlertSentAt[t] < 60*60*1000 {
				continue
			}
			user.BatteryAlertSentAt[t] = now
			payload := map[string]interface{}{
				"userId":      user.UserID,
				"displayName": user.DisplayName,
				"batteryPct":  pct,
				"threshold":   t,
			}
			h.emitToVisible(user, "batteryProxyAlert", payload)
			break // only lowest threshold fires per position update
		}
	}
}

// handleOnMyWay broadcasts an "on my way" signal to all visible users.
func (h *Hub) handleOnMyWay(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("onMyWay", 20) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	placeName := ""
	if v, ok := m["placeName"].(string); ok {
		placeName = shared.SanitizeString(v, 50)
	}
	payload := map[string]interface{}{
		"userId":      user.UserID,
		"displayName": user.DisplayName,
		"placeName":   placeName,
		"at":          time.Now().UnixMilli(),
	}
	h.emitToVisible(user, "onMyWayBroadcast", payload)
}

// handleCancelOnMyWay cancels an "on my way" broadcast.
func (h *Hub) handleCancelOnMyWay(c *Client, _ json.RawMessage) {
	if !c.CheckRateLimit("cancelOnMyWay", 20) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	h.emitToVisible(user, "onMyWayCancel", map[string]interface{}{"userId": user.UserID})
}

// handleShareRide starts a ride-share session: creates a 4-hour live link and
// marks the user as actively sharing a ride with optional vehicle/destination info.
func (h *Hub) handleShareRide(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("shareRide", 5) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	// If already sharing a ride, silently revoke the old token first
	if user.RideShareActive && user.RideShareToken != "" {
		h.revokeLiveTokenByToken(user, user.RideShareToken)
	}

	m := toMap(data)
	vehicle, dest := "", ""
	if m != nil {
		if v, ok := m["vehicle"].(string); ok {
			vehicle = shared.SanitizeString(v, 15)
		}
		if v, ok := m["dest"].(string); ok {
			dest = shared.SanitizeString(v, 30)
		}
	}

	// Create a 4-hour live link
	if h.Cache.GetLiveTokenCount(user.UserID) >= maxLiveLinksPerUser {
		c.Send("rideShareError", map[string]interface{}{"message": "Live link limit reached"})
		return
	}
	fourHoursMs := int64(4 * 60 * 60 * 1000)
	expiresAt := time.Now().UnixMilli() + fourHoursMs
	expiresAtPtr := &expiresAt
	token := generateLiveToken()
	createdAt := time.Now().UnixMilli()

	if err := db.CreateLiveToken(context.Background(), h.pool.DB, token, user.UserID, expiresAtPtr, createdAt); err != nil {
		slog.Error("handleShareRide: failed to create live token", "error", err)
		c.Send("rideShareError", map[string]interface{}{"message": "Failed to create ride link"})
		return
	}

	h.Cache.AddLiveToken(token, user.UserID, expiresAtPtr, createdAt)

	user.RideShareActive = true
	user.RideShareVehicle = vehicle
	user.RideShareDest = dest
	user.RideShareToken = token

	// Tell the initiating client the token (for URL construction + WhatsApp share)
	c.Send("rideShareStarted", map[string]interface{}{
		"token":   token,
		"vehicle": vehicle,
		"dest":    dest,
	})

	// Broadcast updated state to visible peers via userUpdate
	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
}

// handleEndRide ends a ride-share session, revoking the live link.
func (h *Hub) handleEndRide(c *Client, _ json.RawMessage) {
	if !c.CheckRateLimit("endRide", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil || !user.RideShareActive {
		return
	}

	token := user.RideShareToken
	user.RideShareActive = false
	user.RideShareVehicle = ""
	user.RideShareDest = ""
	user.RideShareToken = ""

	if token != "" {
		h.revokeLiveTokenByToken(user, token)
	}

	// Broadcast cleared state
	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
}

// revokeLiveTokenByToken is a shared helper for revoking a specific live token.
// Callers are responsible for re-emitting myLiveLinks if needed.
func (h *Hub) revokeLiveTokenByToken(user *cache.ActiveUser, token string) {
	entry := h.Cache.GetLiveToken(token)
	if entry == nil || entry.UserID != user.UserID {
		return
	}
	h.SendToGroup("live:"+token, "liveExpired", map[string]interface{}{"message": "Ride ended"})
	h.Cache.DeleteLiveToken(token)
	go func() {
		if _, err := h.pool.DB.ExecContext(context.Background(),
			`DELETE FROM live_tokens WHERE token=$1`, token); err != nil {
			slog.Warn("revokeLiveTokenByToken: failed to delete live token from DB", "error", err)
		}
	}()
}

// handleSetStatusMessage sets or clears the user's ambient status message.
// Max 60 chars; empty string clears. Expiry in minutes: 0 = no expiry, else auto-clear.
func (h *Hub) handleSetStatusMessage(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("setStatusMessage", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}

	msg, _ := m["message"].(string)
	msg = shared.SanitizeString(msg, 60)

	var expiresAt int64
	if expMin, ok := m["expiryMinutes"].(float64); ok && expMin > 0 {
		expiresAt = time.Now().Add(time.Duration(expMin) * time.Minute).UnixMilli()
	}

	user.StatusMessage = msg
	user.StatusExpiresAt = expiresAt

	// Persist to DB (best-effort, consistent with quiet hours pattern)
	go func() {
		var expiresVal interface{}
		if expiresAt > 0 {
			expiresVal = expiresAt
		}
		var msgVal interface{}
		if msg != "" {
			msgVal = msg
		}
		_, err := h.pool.DB.ExecContext(context.Background(),
			`UPDATE users SET status_message=$1, status_expires_at=$2 WHERE id=$3`,
			msgVal, expiresVal, user.UserID)
		if err != nil {
			slog.Warn("Failed to persist status message", "error", err)
		}
	}()

	// Broadcast the updated user to all visible peers
	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
}

// handleToggleCrowdMode enables or disables Festival / Crowd-Stay-Together mode for the caller.
// When active the server checks on each position update if any crowd-mode peers have
// drifted more than radiusM metres from the group centre and emits crowdAlert to them.
func (h *Hub) handleToggleCrowdMode(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("toggleCrowdMode", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}
	enabled, _ := m["enabled"].(bool)
	radiusM := 200
	if r, ok := m["radiusM"].(float64); ok && r > 0 {
		if r > 5000 {
			r = 5000 // max 5 km
		}
		radiusM = int(r)
	}

	user.CrowdModeActive = enabled
	user.CrowdModeRadiusM = radiusM
	if enabled && user.CrowdAlertSentAt == nil {
		user.CrowdAlertSentAt = make(map[string]int64)
	}

	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
}

// checkCrowdMode runs after each position update for a user who has CrowdModeActive.
// Computes the centre of all crowd-mode-enabled visible users (including the mover),
// then alerts any peer whose distance from that centre exceeds their radiusM threshold.
// Alerts are rate-limited to once per 30 s per user pair.
func (h *Hub) checkCrowdMode(mover *cache.ActiveUser) {
	if !mover.CrowdModeActive || mover.Latitude == nil || mover.Longitude == nil {
		return
	}

	// Collect crowd-mode peers visible to the mover who also have a known position.
	visibleUIDs := h.Cache.GetVisibleSet(mover.UserID)
	type crowdPeer struct {
		user *cache.ActiveUser
		lat  float64
		lng  float64
	}
	peers := make([]crowdPeer, 0, 4)
	// Always include self
	peers = append(peers, crowdPeer{user: mover, lat: *mover.Latitude, lng: *mover.Longitude})
	for uid := range visibleUIDs {
		sid := h.Cache.GetUserIdToSocketId(uid)
		if sid == "" {
			continue
		}
		u := h.Cache.GetActiveUser(sid)
		if u == nil || !u.CrowdModeActive || u.Latitude == nil || u.Longitude == nil {
			continue
		}
		peers = append(peers, crowdPeer{user: u, lat: *u.Latitude, lng: *u.Longitude})
	}
	if len(peers) < 2 {
		return // nothing to compare against
	}

	// Compute group centre (simple average lat/lng — fine for radii < 10 km)
	var sumLat, sumLng float64
	for _, p := range peers {
		sumLat += p.lat
		sumLng += p.lng
	}
	n := float64(len(peers))
	cLat := sumLat / n
	cLng := sumLng / n

	now := time.Now().UnixMilli()
	const alertCooldownMs = 30_000

	// Alert any peer who is farther than their radiusM from the group centre.
	for _, p := range peers {
		if p.user.UserID == mover.UserID {
			continue // don't alert ourselves
		}
		radius := float64(p.user.CrowdModeRadiusM)
		if radius <= 0 {
			radius = 200
		}
		dist := shared.HaversineM(cLat, cLng, p.lat, p.lng)
		if dist <= radius {
			continue
		}
		// Rate-limit: skip if we already alerted this peer recently
		if mover.CrowdAlertSentAt == nil {
			mover.CrowdAlertSentAt = make(map[string]int64)
		}
		if now-mover.CrowdAlertSentAt[p.user.UserID] < alertCooldownMs {
			continue
		}
		mover.CrowdAlertSentAt[p.user.UserID] = now

		distM := int(math.Round(dist))
		h.SendToClient(p.user.SocketID, "crowdAlert", map[string]interface{}{
			"fromName":        mover.DisplayName,
			"distanceM":       distM,
			"groupSizePeople": len(peers),
		})
	}
}
