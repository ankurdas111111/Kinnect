package ws

import (
	"encoding/json"
	"time"

	"kinnect-v3/internal/intelligence"
	"kinnect-v3/internal/shared"
)

// handleAttest handles the "attest" event — user confirms their location signal is genuine.
// Rate-limited to 12 per hour (one per 5 minutes).
func (h *Hub) handleAttest(c *Client, _ json.RawMessage) {
	if !c.CheckRateLimit("attest", 12) {
		return
	}

	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}

	now := time.Now().UnixMilli()
	user.LastAttestAt = now

	// Recompute safety score with fresh attestation
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

	payload := map[string]interface{}{
		"socketId": user.SocketID,
		"userId":   user.UserID,
		"name":     user.DisplayName,
		"at":       now,
	}

	h.emitToVisibleAndSelf(user, "attestUpdate", payload)
}
