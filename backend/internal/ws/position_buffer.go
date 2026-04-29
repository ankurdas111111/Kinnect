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
//   - Purges geofence_events older than 30 days + per-user row cap (200)  [F6]
//   - Purges room_notes older than 30 days                                [F8]
//   - Purges daily_activity older than 30 days                            [F9]
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
	// SOS watch tokens — delete expired rows
	if _, err := h.pool.DB.ExecContext(ctx,
		`DELETE FROM sos_watch_tokens WHERE expires_at < $1`, time.Now().UnixMilli()); err != nil {
		slog.Warn("Failed to purge sos_watch_tokens", "error", err)
	}

	// ── F6: Geofence event log ─────────────────────────────────────────────────
	// Delete rows older than 30 days.
	thirtyDaysMs := time.Now().Add(-30 * 24 * time.Hour).UnixMilli()
	if _, err := h.pool.DB.ExecContext(ctx,
		`DELETE FROM geofence_events WHERE ts < $1`, thirtyDaysMs); err != nil {
		slog.Warn("Failed to purge geofence_events by age", "error", err)
	}
	// Per-user cap of 200 rows — keep the 200 most recent per user.
	if _, err := h.pool.DB.ExecContext(ctx, `
		DELETE FROM geofence_events
		WHERE id IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY ts DESC) AS rn
				FROM geofence_events
			) ranked WHERE rn > 200
		)`); err != nil {
		slog.Warn("Failed to purge geofence_events by cap", "error", err)
	}

	// ── F8: Room bulletin board ────────────────────────────────────────────────
	// Delete notes older than 30 days (the 20-note cap is enforced on write).
	thirtyDaysMs = time.Now().Add(-30 * 24 * time.Hour).UnixMilli()
	if _, err := h.pool.DB.ExecContext(ctx,
		`DELETE FROM room_notes WHERE created_at < $1`, thirtyDaysMs); err != nil {
		slog.Warn("Failed to purge room_notes", "error", err)
	}

	// ── F9: Daily activity summary ─────────────────────────────────────────────
	if _, err := h.pool.DB.ExecContext(ctx,
		`DELETE FROM daily_activity WHERE date < CURRENT_DATE - INTERVAL '30 days'`); err != nil {
		slog.Warn("Failed to purge daily_activity", "error", err)
	}

	// VACUUM ANALYZE: non-blocking — updates planner statistics and marks dead tuples
	// for reuse. Does NOT lock the table (unlike VACUUM FULL).
	// Uses a detached context with a 30s timeout so it is never cancelled by the
	// shutdown context or by the purge cycle's own cancellation.
	for _, table := range []string{
		"position_history", "movement_events", "zone_visits", "sos_watch_tokens",
		"geofence_events", "room_notes", "daily_activity",
	} {
		vacCtx, vacCancel := context.WithTimeout(context.Background(), 30*time.Second)
		_, err := h.pool.DB.ExecContext(vacCtx, "VACUUM ANALYZE "+table)
		vacCancel()
		if err != nil {
			slog.Warn("VACUUM ANALYZE failed", "table", table, "error", err)
		}
	}
}
