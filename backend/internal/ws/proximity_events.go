package ws

import (
	"context"
	"encoding/json"
	"time"

	"kinnect-v3/internal/db"
)

// handleSetProximityAlert creates or updates a proximity alert for the caller targeting
// another user. The caller must be a contact or guardian/ward of the target. (F7)
// Payload in: { targetUserId: string, radiusM: int }
// Server→Client: proximityAlertSet → self
func (h *Hub) handleSetProximityAlert(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("setProximityAlert", 10) {
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
	targetUserID, _ := m["targetUserId"].(string)
	if targetUserID == "" {
		return
	}
	if targetUserID == user.UserID {
		return
	}

	radiusM := 500 // default 500 m
	if v, ok := toInt(m["radiusM"]); ok && v > 0 && v <= 50000 {
		radiusM = v
	}

	// Authorization: caller must be a contact of or in a guardian/ward relationship with target.
	authorized := false
	contacts := h.Cache.GetContactsForUser(user.UserID)
	for _, cid := range contacts {
		if cid == targetUserID {
			authorized = true
			break
		}
	}
	if !authorized {
		// Check guardian relationship (either direction)
		if gs := h.Cache.GetGuardianship(user.UserID, targetUserID); gs != nil && gs.Status == "active" {
			authorized = true
		} else if gs := h.Cache.GetGuardianship(targetUserID, user.UserID); gs != nil && gs.Status == "active" {
			authorized = true
		}
	}
	if !authorized {
		c.Send("proximityAlertError", map[string]interface{}{
			"error": "not_authorized",
			"msg":   "You must be a contact or in a guardian relationship with this user",
		})
		return
	}

	now := time.Now().UnixMilli()
	entry, err := db.UpsertProximityAlert(context.Background(), h.pool.DB, user.UserID, targetUserID, radiusM, now)
	if err != nil {
		c.Send("proximityAlertError", map[string]interface{}{"error": "db_error"})
		return
	}

	h.Cache.UpsertProximityAlert(entry)

	c.Send("proximityAlertSet", map[string]interface{}{
		"id":           entry.ID,
		"targetUserId": entry.TargetID,
		"radiusM":      entry.RadiusM,
		"enabled":      entry.Enabled,
		"createdAt":    now,
	})
}

// handleRemoveProximityAlert deletes the caller's proximity alert targeting a user. (F7)
// Payload in: { targetUserId: string }
// Server→Client: proximityAlertRemoved → self
func (h *Hub) handleRemoveProximityAlert(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("removeProximityAlert", 20) {
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
	targetUserID, _ := m["targetUserId"].(string)
	if targetUserID == "" {
		return
	}

	_ = db.DeleteProximityAlert(context.Background(), h.pool.DB, user.UserID, targetUserID)
	h.Cache.RemoveProximityAlert(user.UserID, targetUserID)

	c.Send("proximityAlertRemoved", map[string]interface{}{
		"targetUserId": targetUserID,
	})
}

// handleListProximityAlerts returns all proximity alerts owned by the caller. (F7)
// Payload in: {} (empty)
// Server→Client: proximityAlerts → self
func (h *Hub) handleListProximityAlerts(c *Client, _ json.RawMessage) {
	if !c.CheckRateLimit("listProximityAlerts", 20) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}

	rows, err := db.GetProximityAlertsForOwner(context.Background(), h.pool.DB, user.UserID)
	if err != nil {
		c.Send("proximityAlerts", map[string]interface{}{"alerts": []interface{}{}})
		return
	}

	alerts := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		alerts = append(alerts, map[string]interface{}{
			"id":              r.ID,
			"targetUserId":    r.TargetID,
			"targetName":      h.Cache.GetDisplayName(r.TargetID),
			"radiusM":         r.RadiusM,
			"enabled":         r.Enabled,
			"lastTriggeredAt": r.LastTriggeredAt,
		})
	}
	c.Send("proximityAlerts", map[string]interface{}{"alerts": alerts})
}
