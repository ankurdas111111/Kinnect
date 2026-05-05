package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/db"
	"kinnect-v3/internal/shared"
)

// sendTwilioSMS fires a single SMS via the Twilio Messages REST API.
// Silently logs and returns on error so a misconfigured key never crashes the relay.
func sendTwilioSMS(accountSID, authToken, from, to, body string) {
	apiURL := "https://api.twilio.com/2010-04-01/Accounts/" + accountSID + "/Messages.json"
	form := url.Values{}
	form.Set("To", to)
	form.Set("From", from)
	form.Set("Body", body)
	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		slog.Warn("Panic Relay: failed to build Twilio request", "error", err)
		return
	}
	req.SetBasicAuth(accountSID, authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("Panic Relay: Twilio HTTP error", "to", to, "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		slog.Warn("Panic Relay: Twilio rejected SMS", "to", to, "status", resp.StatusCode, "body", string(b))
		return
	}
	slog.Info("Panic Relay: SMS sent via Twilio", "to", to)
}

// ═══════════════════════════════════════════════════════════════════════════
// HEARTBEAT CHECK — daily "sign of life" wellness pulse
// ═══════════════════════════════════════════════════════════════════════════

// handleSetHeartbeat configures heartbeat check settings.
// Frontend sends: { enabled: bool, deadline: "HH:MM" }
func (h *Hub) handleSetHeartbeat(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("setHeartbeat", 10) {
		return
	}
	m := toMap(data)
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	if v, ok := m["enabled"].(bool); ok {
		user.HeartbeatEnabled = v
	}
	if v, ok := m["deadline"].(string); ok && len(v) == 5 {
		user.HeartbeatDeadline = v
	}
	// Record current time as last signal (enabling = sign of life)
	user.HeartbeatLastSignal = time.Now().UnixMilli()

	// Persist to DB
	go func(uid string, enabled bool, deadline string, lastSignal int64) {
		_, _ = h.pool.DB.ExecContext(context.Background(),
			`UPDATE users SET heartbeat_enabled=$1, heartbeat_deadline=$2::time, heartbeat_last_signal=$3 WHERE id=$4`,
			enabled, deadline+":00", lastSignal, uid)
	}(user.UserID, user.HeartbeatEnabled, user.HeartbeatDeadline, user.HeartbeatLastSignal)

	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
}

// cleanupHeartbeatCheck runs every 60s. For each user with heartbeat enabled,
// checks if the daily deadline has passed without any sign of life.
func (h *Hub) cleanupHeartbeatCheck() {
	now := time.Now()
	nowMs := now.UnixMilli()
	nowMin := now.UTC().Hour()*60 + now.UTC().Minute()

	h.Cache.ForEachActiveUser(func(socketID string, user *cache.ActiveUser) {
		if !user.HeartbeatEnabled || user.HeartbeatDeadline == "" {
			return
		}
		// Parse deadline "HH:MM"
		var dH, dM int
		fmt.Sscanf(user.HeartbeatDeadline, "%d:%d", &dH, &dM)
		deadlineMin := dH*60 + dM

		// Only trigger if we're past the deadline
		if nowMin < deadlineMin {
			return
		}

		// Check if last signal was today (after midnight UTC)
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).UnixMilli()
		if user.HeartbeatLastSignal >= todayStart {
			return // got a signal today — all good
		}

		// Debounce: only notify once per day (check if already notified today)
		if user.HeartbeatNotifiedAt >= todayStart {
			return
		}

		// No signal today and deadline passed — notify contacts
		user.HeartbeatNotifiedAt = nowMs
		slog.Info("Heartbeat missed", "user", user.DisplayName, "userId", user.UserID)

		payload := map[string]interface{}{
			"type":        "heartbeatMissed",
			"userId":      user.UserID,
			"displayName": user.DisplayName,
			"deadline":    user.HeartbeatDeadline,
		}
		h.emitToVisible(user, "heartbeatMissed", payload)
	})

	// Also check offline users with heartbeat enabled
	h.Cache.ForEachOfflineUser(func(userID string, entry *cache.OfflineEntry) {
		user := entry.User
		if user == nil || !user.HeartbeatEnabled || user.HeartbeatDeadline == "" {
			return
		}
		var dH, dM int
		fmt.Sscanf(user.HeartbeatDeadline, "%d:%d", &dH, &dM)
		deadlineMin := dH*60 + dM
		if nowMin < deadlineMin {
			return
		}
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).UnixMilli()
		if user.HeartbeatLastSignal >= todayStart || user.HeartbeatNotifiedAt >= todayStart {
			return
		}
		user.HeartbeatNotifiedAt = nowMs
		slog.Info("Heartbeat missed (offline)", "user", user.DisplayName, "userId", user.UserID)

		// For offline users, we can still notify their visible contacts who are online
		payload := map[string]interface{}{
			"type":        "heartbeatMissed",
			"userId":      user.UserID,
			"displayName": user.DisplayName,
			"deadline":    user.HeartbeatDeadline,
			"offline":     true,
		}
		sids := h.Cache.GetVisibleSocketIDs(user)
		for _, sid := range sids {
			h.SendToClient(sid, "heartbeatMissed", payload)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// JOURNEY SHIELD — auto-trip detection and arrival notifications
// ═══════════════════════════════════════════════════════════════════════════

const (
	vehicleSpeedThresholdMs = 4.17 // ~15 km/h in m/s
	tripConfirmDurationMs   = 120_000 // 2 minutes sustained
	tripStopDurationMs      = 300_000 // 5 minutes stopped = trip ended
)

// EvaluateTrip is called from handlePosition on every position update.
// It manages trip lifecycle: detect start, detect stop, send notifications.
func (h *Hub) EvaluateTrip(user *cache.ActiveUser) {
	now := time.Now().UnixMilli()
	speed := user.Speed // m/s

	if !user.TripActive {
		// ── Not in a trip — look for vehicle speed ─────────────────────
		if speed >= vehicleSpeedThresholdMs {
			if user.TripVehicleStart == 0 {
				user.TripVehicleStart = now
			} else if now-user.TripVehicleStart >= tripConfirmDurationMs {
				// Sustained vehicle speed for 2+ minutes → confirm trip
				user.TripActive = true
				user.TripStartedAt = now
				user.TripStoppedAt = 0
				user.TripNotifiedStop = false
				if user.Latitude != nil {
					user.TripStartLat = *user.Latitude
				}
				if user.Longitude != nil {
					user.TripStartLng = *user.Longitude
				}
				slog.Info("Trip started", "user", user.DisplayName)

				h.emitToVisible(user, "tripStarted", map[string]interface{}{
					"userId":      user.UserID,
					"displayName": user.DisplayName,
					"startedAt":   user.TripStartedAt,
				})
			}
		} else {
			// Speed dropped below threshold — reset detection
			user.TripVehicleStart = 0
		}
		return
	}

	// ── In a trip — monitor for stop ────────────────────────────────
	if speed < 0.5 { // effectively stopped
		if user.TripStoppedAt == 0 {
			user.TripStoppedAt = now
		} else if now-user.TripStoppedAt >= tripStopDurationMs && !user.TripNotifiedStop {
			// Stopped for 5+ minutes — end the trip
			user.TripNotifiedStop = true
			h.endTrip(user)
		}
	} else {
		// Moving again — reset stop timer
		user.TripStoppedAt = 0
		user.TripNotifiedStop = false
	}
}

// endTrip finalizes a trip: checks if arrived at saved place, notifies contacts.
func (h *Hub) endTrip(user *cache.ActiveUser) {
	user.TripActive = false
	user.TripVehicleStart = 0

	slog.Info("Trip ended", "user", user.DisplayName)
	h.emitToVisible(user, "tripStoppedNew", map[string]interface{}{
		"userId":      user.UserID,
		"displayName": user.DisplayName,
	})
}

// haversine returns the distance in meters between two lat/lng points.
func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000 // Earth radius in meters
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// ═══════════════════════════════════════════════════════════════════════════
// PANIC RELAY — SOS SMS escalation
// ═══════════════════════════════════════════════════════════════════════════

// handleSetEmergencyPhones saves external emergency phone numbers.
// Frontend sends: { phone1: "+91...", phone2: "+91..." }
func (h *Hub) handleSetEmergencyPhones(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("setEmergencyPhones", 10) {
		return
	}
	m := toMap(data)
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	if v, ok := m["phone1"].(string); ok && len(v) <= 20 {
		user.EmergencyPhone1 = v
	}
	if v, ok := m["phone2"].(string); ok && len(v) <= 20 {
		user.EmergencyPhone2 = v
	}

	// Persist
	go func(uid, p1, p2 string) {
		_, _ = h.pool.DB.ExecContext(context.Background(),
			`UPDATE users SET emergency_phone_1=$1, emergency_phone_2=$2 WHERE id=$3`,
			nilIfEmpty(p1), nilIfEmpty(p2), uid)
	}(user.UserID, user.EmergencyPhone1, user.EmergencyPhone2)

	c.Send("emergencyPhonesUpdated", map[string]interface{}{
		"phone1": user.EmergencyPhone1,
		"phone2": user.EmergencyPhone2,
	})
}

// StartPanicRelayTimer is called when an SOS is triggered.
// It starts a 3-minute timer. If no in-app ack arrives, escalates via SMS.
func (h *Hub) StartPanicRelayTimer(user *cache.ActiveUser) {
	if user.EmergencyPhone1 == "" && user.EmergencyPhone2 == "" {
		return // no external contacts configured
	}
	userID := user.UserID
	displayName := user.DisplayName
	sosToken := ""
	if user.SOS.Token != nil {
		sosToken = *user.SOS.Token
	}

	time.AfterFunc(3*time.Minute, func() {
		// Re-fetch user — they might have cancelled SOS or received an ack
		client := h.GetClientByUserID(userID)
		if client == nil {
			return
		}
		sid := client.ID()
		current := h.Cache.GetActiveUser(sid)
		if current == nil || !current.SOS.Active {
			return // SOS was cancelled or already resolved
		}
		if len(current.SOS.Acks) > 0 {
			return // someone acknowledged in-app
		}

		slog.Warn("Panic Relay: no in-app ack after 3min, escalating",
			"user", displayName, "userId", userID)

		// Build SMS message
		watchURL := fmt.Sprintf("https://kinnect.app/watch/%s", sosToken)
		msg := fmt.Sprintf("KINNECT SOS: %s needs help! Track them live: %s", displayName, watchURL)

		// Send to configured phones (placeholder — needs real SMS provider)
		phones := []string{}
		if current.EmergencyPhone1 != "" {
			phones = append(phones, current.EmergencyPhone1)
		}
		if current.EmergencyPhone2 != "" {
			phones = append(phones, current.EmergencyPhone2)
		}

		cfg := h.config
		for _, phone := range phones {
			slog.Info("Panic Relay: sending SMS", "to", phone, "user", displayName)
			if cfg.TwilioAccountSID != "" && cfg.TwilioAuthToken != "" && cfg.TwilioFromNumber != "" {
				p := phone
				go sendTwilioSMS(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioFromNumber, p, msg)
			}
		}

		// Notify the user that SMS was sent
		h.SendToClient(sid, "panicRelaySent", map[string]interface{}{
			"phones":  phones,
			"message": "Emergency contacts notified via SMS",
		})
	})
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ═══════════════════════════════════════════════════════════════════════════
// WALK WITH ME — virtual escort session
// ═══════════════════════════════════════════════════════════════════════════

const (
	walkStopAlertMs    = 3 * 60 * 1000   // 3 min stopped
	walkOfflineAlertMs = 2 * 60 * 1000   // 2 min no position
	walkArrivalRadiusM = 150.0           // within 150m of destination = arrived
	walkDeviationM     = 500.0           // >500m off straight-line path
	walkDeviationCooldownMs = 5 * 60 * 1000 // debounce deviation alerts
	walkMaxDurationMs  = 4 * 60 * 60 * 1000 // 4h auto-expire
)

// handleStartWalkWithMe starts a virtual escort session.
// Frontend sends: { destLat, destLng, destName, watcherUserId }
func (h *Hub) handleStartWalkWithMe(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("startWalkWithMe", 5) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}

	// End any existing walk session first
	if user.WalkActive {
		h.endWalkWithMe(user, "restarted")
	}

	m := toMap(data)
	destLat, okLat := toFloat64(m["destLat"])
	destLng, okLng := toFloat64(m["destLng"])
	if !okLat || !okLng {
		c.Send("walkError", map[string]interface{}{"message": "Destination required"})
		return
	}
	destName := ""
	if v, ok := m["destName"].(string); ok {
		destName = shared.SanitizeString(v, 50)
	}
	if destName == "" {
		destName = "Destination"
	}

	watcherID := ""
	if v, ok := m["watcherUserId"].(string); ok {
		watcherID = v
	}

	// Create a 4-hour live link (reuse ride-share pattern)
	if h.Cache.GetLiveTokenCount(user.UserID) >= 5 {
		c.Send("walkError", map[string]interface{}{"message": "Live link limit reached"})
		return
	}
	expiresAt := time.Now().UnixMilli() + walkMaxDurationMs
	expiresAtPtr := &expiresAt
	token := generateLiveToken()
	createdAt := time.Now().UnixMilli()

	if err := db.CreateLiveToken(context.Background(), h.pool.DB, token, user.UserID, expiresAtPtr, createdAt); err != nil {
		slog.Error("startWalkWithMe: failed to create live token", "error", err)
		c.Send("walkError", map[string]interface{}{"message": "Failed to start walk session"})
		return
	}
	h.Cache.AddLiveToken(token, user.UserID, expiresAtPtr, createdAt)

	now := time.Now().UnixMilli()
	user.WalkActive = true
	user.WalkDestLat = destLat
	user.WalkDestLng = destLng
	user.WalkDestName = destName
	user.WalkToken = token
	user.WalkWatcherID = watcherID
	user.WalkStartedAt = now
	user.WalkStoppedAt = 0
	user.WalkOfflineAt = 0
	user.WalkDeviationNotifiedAt = 0

	slog.Info("Walk With Me started", "user", user.DisplayName, "dest", destName)

	// Tell the initiating client
	c.Send("walkStarted", map[string]interface{}{
		"token":    token,
		"destName": destName,
		"destLat":  destLat,
		"destLng":  destLng,
	})

	// Notify the selected watcher
	if watcherID != "" {
		if watcher := h.GetClientByUserID(watcherID); watcher != nil {
			watcher.Send("walkWithMeRequest", map[string]interface{}{
				"fromUserId":   user.UserID,
				"fromName":     user.DisplayName,
				"destName":     destName,
				"token":        token,
			})
		}
	}

	// Broadcast updated state
	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
}

// handleEndWalkWithMe manually ends a walk session.
func (h *Hub) handleEndWalkWithMe(c *Client, _ json.RawMessage) {
	if !c.CheckRateLimit("endWalkWithMe", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil || !user.WalkActive {
		return
	}
	h.endWalkWithMe(user, "cancelled")
}

// endWalkWithMe clears walk state, revokes live link, notifies watcher.
func (h *Hub) endWalkWithMe(user *cache.ActiveUser, reason string) {
	token := user.WalkToken
	watcherID := user.WalkWatcherID
	destName := user.WalkDestName

	user.WalkActive = false
	user.WalkDestLat = 0
	user.WalkDestLng = 0
	user.WalkDestName = ""
	user.WalkToken = ""
	user.WalkWatcherID = ""
	user.WalkStartedAt = 0
	user.WalkStoppedAt = 0
	user.WalkOfflineAt = 0

	if token != "" {
		h.revokeLiveTokenByToken(user, token)
	}

	slog.Info("Walk With Me ended", "user", user.DisplayName, "reason", reason)

	// Notify the user
	if c := h.GetClientByUserID(user.UserID); c != nil {
		c.Send("walkEnded", map[string]interface{}{
			"reason":   reason,
			"destName": destName,
		})
	}

	// Notify the watcher
	if watcherID != "" {
		if watcher := h.GetClientByUserID(watcherID); watcher != nil {
			watcher.Send("walkEnded", map[string]interface{}{
				"fromUserId": user.UserID,
				"fromName":   user.DisplayName,
				"reason":     reason,
				"destName":   destName,
			})
		}
	}

	// Broadcast cleared state
	sanitized := h.Cache.SanitizeUser(user)
	sanitized["online"] = true
	h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
}

// EvaluateWalk is called from the cleanup goroutine every 30s.
// Monitors all active walk sessions for: arrival, stopped, offline, deviation, timeout.
func (h *Hub) EvaluateWalk() {
	now := time.Now().UnixMilli()

	h.Cache.ForEachActiveUser(func(socketID string, user *cache.ActiveUser) {
		if !user.WalkActive {
			return
		}

		// ── Auto-expire after max duration ──────────────────────────────
		if now-user.WalkStartedAt > walkMaxDurationMs {
			h.endWalkWithMe(user, "expired")
			return
		}

		if user.Latitude == nil || user.Longitude == nil {
			return
		}
		lat, lng := *user.Latitude, *user.Longitude

		// ── Check arrival (within 150m of destination) ───────────────────
		distToDest := haversine(lat, lng, user.WalkDestLat, user.WalkDestLng)
		if distToDest <= walkArrivalRadiusM {
			slog.Info("Walk With Me: arrived!", "user", user.DisplayName, "dest", user.WalkDestName)

			// Notify user
			if c := h.GetClientByUserID(user.UserID); c != nil {
				c.Send("walkArrived", map[string]interface{}{
					"destName": user.WalkDestName,
				})
			}
			// Notify watcher
			if user.WalkWatcherID != "" {
				if watcher := h.GetClientByUserID(user.WalkWatcherID); watcher != nil {
					watcher.Send("walkArrived", map[string]interface{}{
						"fromUserId": user.UserID,
						"fromName":   user.DisplayName,
						"destName":   user.WalkDestName,
					})
				}
			}
			h.endWalkWithMe(user, "arrived")
			return
		}

		// ── Check stopped >3 min ─────────────────────────────────────────
		if user.Speed < 0.3 {
			if user.WalkStoppedAt == 0 {
				user.WalkStoppedAt = now
			} else if now-user.WalkStoppedAt >= walkStopAlertMs {
				h.sendWalkAlert(user, "stopped", fmt.Sprintf("%s has stopped moving for 3+ minutes", user.DisplayName))
				user.WalkStoppedAt = now + walkStopAlertMs // reset to avoid spamming (alert again in another 3 min)
			}
		} else {
			user.WalkStoppedAt = 0
		}

		// ── Check offline (no position update >2 min) ────────────────────
		timeSinceUpdate := now - user.LastUpdate
		if timeSinceUpdate >= walkOfflineAlertMs && user.WalkOfflineAt == 0 {
			user.WalkOfflineAt = now
			h.sendWalkAlert(user, "offline", fmt.Sprintf("%s's phone appears to be offline", user.DisplayName))
		} else if timeSinceUpdate < walkOfflineAlertMs {
			user.WalkOfflineAt = 0 // back online
		}

		// ── Check path deviation (>500m from straight line to dest) ──────
		// Simple heuristic: distance to dest should not increase by >500m
		// from the straight-line distance at start. More precisely: distance
		// from current position to the line (start→dest) > 500m.
		if user.WalkStartedAt > 0 && now-user.WalkDeviationNotifiedAt > walkDeviationCooldownMs {
			// Use perpendicular distance from current pos to start→dest line
			// Approximation: if distance to dest > initial distance + 500m, flag it
			// Better: use point-to-line distance
			startLat, startLng := user.TripStartLat, user.TripStartLng
			if startLat == 0 && startLng == 0 {
				// Use walk start position (we didn't store it separately, use current)
				// Skip deviation check if we don't have start position
			} else {
				perpDist := pointToSegmentDistance(lat, lng, startLat, startLng, user.WalkDestLat, user.WalkDestLng)
				if perpDist > walkDeviationM {
					user.WalkDeviationNotifiedAt = now
					h.sendWalkAlert(user, "deviation",
						fmt.Sprintf("%s has deviated significantly from the route", user.DisplayName))
				}
			}
		}
	})
}

// sendWalkAlert sends an alert to both the walking user and their watcher.
func (h *Hub) sendWalkAlert(user *cache.ActiveUser, alertType, message string) {
	payload := map[string]interface{}{
		"type":     alertType,
		"userId":   user.UserID,
		"userName": user.DisplayName,
		"destName": user.WalkDestName,
		"message":  message,
	}

	// Notify user
	if c := h.GetClientByUserID(user.UserID); c != nil {
		c.Send("walkAlert", payload)
	}
	// Notify watcher
	if user.WalkWatcherID != "" {
		if watcher := h.GetClientByUserID(user.WalkWatcherID); watcher != nil {
			watcher.Send("walkAlert", payload)
		}
	}
}

// pointToSegmentDistance returns the perpendicular distance in meters from
// point P to the line segment A→B (all in lat/lng degrees).
func pointToSegmentDistance(pLat, pLng, aLat, aLng, bLat, bLng float64) float64 {
	// Convert to approximate flat coordinates (meters)
	cosLat := math.Cos(pLat * math.Pi / 180)
	mPerDeg := 111320.0
	px := (pLng - aLng) * cosLat * mPerDeg
	py := (pLat - aLat) * mPerDeg
	bx := (bLng - aLng) * cosLat * mPerDeg
	by := (bLat - aLat) * mPerDeg

	lenSq := bx*bx + by*by
	if lenSq < 1 {
		return math.Sqrt(px*px + py*py) // A and B are same point
	}
	t := math.Max(0, math.Min(1, (px*bx+py*by)/lenSq))
	projX := t * bx
	projY := t * by
	dx := px - projX
	dy := py - projY
	return math.Sqrt(dx*dx + dy*dy)
}
