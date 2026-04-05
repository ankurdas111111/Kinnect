package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"kinnect-v3/internal/db"
)

type getRecentTrailRequest struct {
	TargetUserID  string `json:"targetUserId"`
	WindowMinutes int    `json:"windowMinutes"`
}

// handleGetRecentTrail handles the "getRecentTrail" event.
// Returns position history for a visible user within a time window.
func (h *Hub) handleGetRecentTrail(c *Client, data json.RawMessage) {
	var req getRecentTrailRequest
	if err := json.Unmarshal(data, &req); err != nil || req.TargetUserID == "" {
		c.Send("trailError", map[string]interface{}{"error": "invalid request"})
		return
	}

	// Default and clamp window
	if req.WindowMinutes <= 0 {
		req.WindowMinutes = 30
	}
	if req.WindowMinutes > 60 {
		req.WindowMinutes = 60
	}

	// Auth: targetUserId must be in the caller's visible set
	callerUserID := c.UserID()
	visible := h.Cache.GetVisibleSet(callerUserID)
	if !visible[req.TargetUserID] && req.TargetUserID != callerUserID {
		c.Send("trailError", map[string]interface{}{"error": "not authorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	points, err := db.GetRecentTrail(ctx, h.pool.DB, req.TargetUserID, req.WindowMinutes)
	if err != nil {
		slog.Warn("getRecentTrail DB error", "error", err, "targetUserId", req.TargetUserID)
		c.Send("trailError", map[string]interface{}{"error": "query failed"})
		return
	}

	// Convert to JSON-friendly slice
	out := make([]map[string]interface{}, 0, len(points))
	for _, p := range points {
		out = append(out, map[string]interface{}{
			"lat": p.Lat,
			"lng": p.Lng,
			"ts":  p.Ts,
		})
	}

	c.Send("recentTrail", map[string]interface{}{
		"userId": req.TargetUserID,
		"points": out,
	})
}

type getHistoryRequest struct {
	UserID string `json:"userId"`
	Start  int64  `json:"start"`
	End    int64  `json:"end"`
}

// handleGetHistory handles the "getHistory" event.
// Returns position history for a visible user within an absolute time range.
func (h *Hub) handleGetHistory(c *Client, data json.RawMessage) {
	var req getHistoryRequest
	if err := json.Unmarshal(data, &req); err != nil || req.UserID == "" {
		c.Send("historyData", map[string]interface{}{"ok": false, "error": "invalid request"})
		return
	}

	callerUserID := c.UserID()
	visible := h.Cache.GetVisibleSet(callerUserID)
	if !visible[req.UserID] && req.UserID != callerUserID {
		c.Send("historyData", map[string]interface{}{"ok": false, "error": "not authorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	points, err := db.GetHistory(ctx, h.pool.DB, req.UserID, req.Start, req.End)
	if err != nil {
		slog.Warn("getHistory DB error", "error", err, "userId", req.UserID)
		c.Send("historyData", map[string]interface{}{"ok": false, "error": "query failed"})
		return
	}

	out := make([]map[string]interface{}, 0, len(points))
	for _, p := range points {
		out = append(out, map[string]interface{}{
			"lat": p.Lat,
			"lng": p.Lng,
			"t":   p.Ts,
		})
	}

	c.Send("historyData", map[string]interface{}{
		"ok":     true,
		"points": out,
	})
}
