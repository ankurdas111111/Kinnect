package ws

import (
	"context"
	"log/slog"
	"time"

)

const (
	// purgeInterval: 6h instead of 24h so the purge fires multiple times per
	// Render spin-up window (Render free tier restarts every ~15 min of inactivity).
	// A 24h window would allow unbounded row accumulation between restarts.
	purgeInterval         = 6 * time.Hour
	positionRetentionDays = 7
	movementRetentionDays = 7 // was 30 — semantic events only, 7 days is enough
)

// StartPositionPurger runs every 6 hours:
//   - Drains any remaining position_history rows (dual-write stopped; table empties naturally)
//   - Purges movement_events older than 7 days + per-user row cap (500)
//   - Purges zone_visits older than 7 days + per-place row cap (200)
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
	// SOS watch tokens — delete expired rows (no previous purge existed)
	if _, err := h.pool.DB.ExecContext(ctx,
		`DELETE FROM sos_watch_tokens WHERE expires_at < $1`, time.Now().UnixMilli()); err != nil {
		slog.Warn("Failed to purge sos_watch_tokens", "error", err)
	}

	// VACUUM ANALYZE: non-blocking — updates planner statistics and marks dead tuples
	// for reuse. Does NOT lock the table (unlike VACUUM FULL).
	// Uses a detached context with a 30s timeout so it is never cancelled by the
	// shutdown context or by the purge cycle's own cancellation.
	for _, table := range []string{"position_history", "movement_events", "zone_visits", "sos_watch_tokens"} {
		vacCtx, vacCancel := context.WithTimeout(context.Background(), 30*time.Second)
		_, err := h.pool.DB.ExecContext(vacCtx, "VACUUM ANALYZE "+table)
		vacCancel()
		if err != nil {
			slog.Warn("VACUUM ANALYZE failed", "table", table, "error", err)
		}
	}
}
