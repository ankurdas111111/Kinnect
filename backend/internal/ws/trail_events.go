package ws

import (
	"context"
	"encoding/json"
	"time"
)

// handleGetRecentTrail serves historical position points for a target user.
// Payload: { targetUserId: string, windowMinutes: int (15|30|60|120) }
// Response: recentTrailData { points: [{lat,lng,ts,speed}], windowMinutes }
// Response on error: trailError { error: string }
func (h *Hub) handleGetRecentTrail(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("getRecentTrail", 20) {
		return
	}
	m := toMap(data)
	requester := h.Cache.GetActiveUser(c.ID())
	if requester == nil {
		return
	}

	targetUserId, _ := m["targetUserId"].(string)
	if targetUserId == "" {
		c.Send("trailError", map[string]interface{}{"error": "targetUserId required"})
		return
	}

	// Clamp windowMinutes to a safe range.
	window := 60
	if v, ok := m["windowMinutes"].(float64); ok {
		w := int(v)
		if w >= 15 && w <= 120 {
			window = w
		}
	}

	// Visibility check: requester must be able to see the target (or be the target).
	if targetUserId != requester.UserID {
		targetSocketId := h.Cache.GetUserIdToSocketId(targetUserId)
		visible := h.Cache.GetVisibleSocketIDs(requester)
		found := false
		for _, sid := range visible {
			if sid == targetSocketId {
				found = true
				break
			}
		}
		if !found {
			c.Send("trailError", map[string]interface{}{"error": "not authorized"})
			return
		}
	}

	cutoff := time.Now().Add(-time.Duration(window) * time.Minute).UnixMilli()
	rows, err := h.pool.DB.QueryContext(context.Background(),
		`SELECT lat, lng, speed, ts
		 FROM position_history
		 WHERE user_id = $1 AND ts >= $2
		 ORDER BY ts ASC
		 LIMIT 1000`,
		targetUserId, cutoff)
	if err != nil {
		c.Send("trailError", map[string]interface{}{"error": "database error"})
		return
	}
	defer rows.Close()

	type point struct {
		Lat   float64 `json:"lat"`
		Lng   float64 `json:"lng"`
		Speed float64 `json:"speed"`
		Ts    int64   `json:"ts"`
	}
	points := make([]point, 0, 64)
	for rows.Next() {
		var p point
		var speed *float64
		if err := rows.Scan(&p.Lat, &p.Lng, &speed, &p.Ts); err != nil {
			continue
		}
		if speed != nil {
			p.Speed = *speed
		}
		points = append(points, p)
	}

	c.Send("recentTrailData", map[string]interface{}{
		"points":        points,
		"windowMinutes": window,
		"targetUserId":  targetUserId,
	})
}
