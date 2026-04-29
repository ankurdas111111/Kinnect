package ws

import (
	"context"
	"encoding/json"

	"kinnect-v3/internal/db"
)

// handleGetDailyActivity returns up to 30 days of daily activity rows for a user. (F9)
// Payload in (all optional): { userId: string }
//   - If userId is omitted or equals the caller's own ID, returns self data.
//   - If userId is another user, the caller must be an active guardian of that user.
//
// Server→Client: dailyActivity → self
func (h *Hub) handleGetDailyActivity(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("getDailyActivity", 20) {
		return
	}
	caller := h.Cache.GetActiveUser(c.ID())
	if caller == nil {
		return
	}

	// Determine whose activity to query.
	targetUserID := caller.UserID
	m := toMap(data)
	if m != nil {
		if uid, ok := m["userId"].(string); ok && uid != "" && uid != caller.UserID {
			// Authorise: caller must be an active guardian of the target.
			gs := h.Cache.GetGuardianship(caller.UserID, uid)
			if gs == nil || gs.Status != "active" {
				c.Send("dailyActivityError", map[string]interface{}{
					"error": "not_authorized",
					"msg":   "You must be an active guardian to view this user's activity",
				})
				return
			}
			targetUserID = uid
		}
	}

	rows, err := db.GetDailyActivity(context.Background(), h.pool.DB, targetUserID, 30)
	if err != nil {
		c.Send("dailyActivity", map[string]interface{}{
			"userId": targetUserID,
			"days":   []interface{}{},
		})
		return
	}

	days := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		days = append(days, map[string]interface{}{
			"date":          r.Date,
			"distanceM":     r.DistanceM,
			"activeMinutes": r.ActiveMinutes,
			"updatedAt":     r.UpdatedAt,
		})
	}
	c.Send("dailyActivity", map[string]interface{}{
		"userId": targetUserID,
		"days":   days,
	})
}
