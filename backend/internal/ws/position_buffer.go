package ws

import (
	"context"
	"log/slog"
	"time"

	"kinnect-v3/internal/db"
)

const (
	purgeInterval         = 24 * time.Hour
	positionRetentionDays = 7
	movementRetentionDays = 7 // was 30 — semantic events only, 7 days is enough
)

// StartPositionPurger runs daily:
//   - Drains any remaining position_history rows (dual-write stopped; table empties naturally)
//   - Purges movement_events older than 7 days
//   - Purges zone_visits older than 7 days
//   - Purges expired sos_watch_tokens
//   - Runs VACUUM ANALYZE on the heavy time-series tables (non-blocking, updates planner stats)
func (h *Hub) StartPositionPurger(ctx context.Context) {
	// Run once immediately at startup so the first deployment cleans up right away.
	h.runPurge(ctx)

	ticker := time.NewTicker(purgeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.runPurge(ctx)
		}
	}
}

func (h *Hub) runPurge(ctx context.Context) {
	// Drain legacy position_history (dual-write removed; rows expire naturally over 7 days)
	if err := db.PurgePositionHistory(ctx, h.pool.DB, positionRetentionDays); err != nil {
		slog.Error("Failed to purge position_history", "error", err)
	}

	// Movement events — keep 7 days of semantic transitions
	if err := db.PurgeMovementEvents(ctx, h.pool.DB, movementRetentionDays); err != nil {
		slog.Warn("Failed to purge movement_events", "error", err)
	}

	// Zone visits — 7-day retention
	if _, err := h.pool.DB.ExecContext(ctx,
		`DELETE FROM zone_visits WHERE arrived_at < NOW() - INTERVAL '7 days'`); err != nil {
		slog.Warn("Failed to purge zone_visits", "error", err)
	}

	// SOS watch tokens — delete expired rows (no previous purge existed)
	if _, err := h.pool.DB.ExecContext(ctx,
		`DELETE FROM sos_watch_tokens WHERE expires_at < $1`, time.Now().UnixMilli()); err != nil {
		slog.Warn("Failed to purge sos_watch_tokens", "error", err)
	}

	// VACUUM ANALYZE: non-blocking — updates planner statistics and marks dead tuples
	// for reuse. Does NOT lock the table (unlike VACUUM FULL).
	// Runs after deletes so the planner immediately benefits from updated row counts.
	for _, table := range []string{"position_history", "movement_events", "zone_visits", "sos_watch_tokens"} {
		if _, err := h.pool.DB.ExecContext(ctx, "VACUUM ANALYZE "+table); err != nil {
			slog.Warn("VACUUM ANALYZE failed", "table", table, "error", err)
		}
	}
}
