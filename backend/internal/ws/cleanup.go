package ws

import (
	"context"
	"log/slog"
	"time"

	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/db"
)

const (
	sevenDaysMs = 7 * 24 * 60 * 60 * 1000
)

// safeLoop runs fn on each ticker tick, recovering from panics and restarting the loop.
// It stops cleanly when ctx is done. Panics are logged and the routine restarts after 5s.
func safeLoop(ctx context.Context, interval time.Duration, name string, fn func()) {
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("cleanup goroutine panicked", "routine", name, "panic", r)
					}
				}()
				ticker := time.NewTicker(interval)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						fn()
					}
				}
			}()
			// Inner func returned — either ctx.Done() or panic recovery.
			select {
			case <-ctx.Done():
				return
			default:
				slog.Info("cleanup goroutine restarting after panic", "routine", name)
				time.Sleep(5 * time.Second)
			}
		}
	}()
}

// StartCleanupRoutines starts 14 periodic goroutines for cache and DB cleanup.
// All routines use safeLoop so a panic in one is recovered and the routine restarts.
func (h *Hub) StartCleanupRoutines(ctx context.Context) {
	safeLoop(ctx, 60*time.Second, "expireOfflineUsers",      h.cleanupExpireOfflineUsers)
	safeLoop(ctx, 30*time.Second, "expireWatchTokens",       h.cleanupExpireWatchTokens)
	safeLoop(ctx, 60*time.Second, "expireLiveTokens",        h.cleanupExpireLiveTokens)
	safeLoop(ctx, 1*time.Hour,    "cleanEmptyOldRooms",      h.cleanupEmptyOldRooms)
	safeLoop(ctx, 60*time.Second, "expireRoomAdmins",        h.cleanupExpireRoomAdmins)
	safeLoop(ctx, 60*time.Second, "expireGuardianships",     h.cleanupExpireGuardianships)
	safeLoop(ctx, 60*time.Second, "checkInOverdue",          h.cleanupCheckInOverdue)
	safeLoop(ctx, 24*time.Hour,   "purgeStaleGuardianships", h.cleanupStaleGuardianships)
	safeLoop(ctx, 5*time.Minute,  "haventMovedAlerts",       h.checkHaventMovedAlerts)
	safeLoop(ctx, 60*time.Second, "expireStatusMessages",    h.cleanupExpiredStatusMessages)
	safeLoop(ctx, 60*time.Second, "heartbeatWellness",       h.cleanupHeartbeatCheck)
	safeLoop(ctx, 30*time.Second, "walkWithMe",              h.EvaluateWalk)
	safeLoop(ctx, 5*time.Minute,  "expireSecretChatInvites", func() {
		h.Cache.CollectExpiredSecretChatInvites(time.Now().UnixMilli())
	})
	safeLoop(ctx, 24*time.Hour,   "purgeSecretMessages",     h.cleanupExpiredSecretMessages)

	slog.Info("Cleanup routines started")
}

func (h *Hub) cleanupExpireOfflineUsers() {
	now := time.Now().UnixMilli()
	list := h.Cache.CollectExpiredOfflineUsers(now)
	for _, e := range list {
		h.SendToClients(e.VisibleSids, "userDisconnect", e.SocketID)
	}
}

func (h *Hub) cleanupExpireWatchTokens() {
	now := time.Now().UnixMilli()
	tokens := h.Cache.CollectExpiredWatchTokens(now)
	payload := map[string]interface{}{"user": nil, "sos": map[string]interface{}{"active": false}}
	for _, token := range tokens {
		h.SendToGroup("watch:"+token, "watchUpdate", payload)
	}
}

func (h *Hub) cleanupExpireLiveTokens() {
	now := time.Now().UnixMilli()
	list := h.Cache.CollectExpiredLiveTokens(now)
	for _, e := range list {
		h.SendToGroup("live:"+e.Token, "liveExpired", map[string]interface{}{"message": "Link expired"})
	}
	if len(list) > 0 {
		if _, err := db.DeleteExpiredLiveTokens(context.Background(), h.pool.DB); err != nil {
			slog.Error("Failed to delete expired live tokens from DB", "error", err)
		}
	}
}

func (h *Hub) cleanupEmptyOldRooms() {
	now := time.Now().UnixMilli()
	_ = h.Cache.CollectEmptyOldRooms(now, sevenDaysMs)
	if err := db.DeleteEmptyOldRooms(context.Background(), h.pool.DB, sevenDaysMs); err != nil {
		slog.Error("Failed to delete empty old rooms from DB", "error", err)
	}
}

func (h *Hub) cleanupExpireRoomAdmins() {
	now := time.Now().UnixMilli()
	list := h.Cache.ExpireRoomAdminsInCache(now)
	payload := func(roomCode, userID string) map[string]interface{} {
		return map[string]interface{}{
			"roomCode":   roomCode,
			"userId":     userID,
			"role":       "member",
			"expiresAt":  nil,
		}
	}
	for _, e := range list {
		p := payload(e.RoomCode, e.UserID)
		h.SendToClients(e.MemberSids, "roomAdminUpdated", p)
	}
	if len(list) > 0 {
		if _, err := db.ExpireRoomAdmins(context.Background(), h.pool.DB, now); err != nil {
			slog.Error("Failed to expire room admins in DB", "error", err)
		}
	}
}

func (h *Hub) cleanupExpireGuardianships() {
	now := time.Now().UnixMilli()
	list := h.Cache.CollectExpiredGuardianships(now)
	updatePayload := func(gID, wID string) map[string]interface{} {
		return map[string]interface{}{
			"guardianId": gID,
			"wardId":     wID,
			"status":     "expired",
			"expiresAt":  nil,
		}
	}
	for _, e := range list {
		p := updatePayload(e.GuardianID, e.WardID)
		if c := h.GetClientByUserID(e.GuardianID); c != nil {
			c.Send("guardianUpdated", p)
		}
		if c := h.GetClientByUserID(e.WardID); c != nil {
			c.Send("guardianUpdated", p)
		}
	}
	if len(list) > 0 {
		if _, err := db.ExpireGuardianships(context.Background(), h.pool.DB, now); err != nil {
			slog.Error("Failed to expire guardianships in DB", "error", err)
		}
	}
}

func (h *Hub) cleanupStaleGuardianships() {
	_, err := h.pool.DB.ExecContext(context.Background(),
		`DELETE FROM guardianships
		 WHERE status IN ('expired', 'revoked')
		   AND created_at < extract(epoch from now() - interval '30 days')::bigint * 1000`)
	if err != nil {
		slog.Error("Failed to purge stale guardianships", "error", err)
	}
}

func (h *Hub) cleanupCheckInOverdue() {
	now := time.Now().UnixMilli()
	h.Cache.ForEachActiveUser(func(socketID string, user *cache.ActiveUser) {
		ch := &user.CheckIn
		if !ch.Enabled {
			return
		}
		lastAt := ch.LastCheckInAt
		if lastAt == 0 {
			// First evaluation: set baseline to now so the first interval starts counting.
			// Persist it so the user doesn't reset every cleanup cycle.
			ch.LastCheckInAt = now
			lastAt = now
		}
		intervalMs := int64(ch.IntervalMin) * 60 * 1000
		overdueMs := int64(ch.OverdueMin) * 60 * 1000
		since := now - lastAt

		// Only send checkInRequest once per interval period (debounce).
		if since >= intervalMs && now-ch.RequestedAt > intervalMs {
			ch.RequestedAt = now
			h.SendToClient(socketID, "checkInRequest", map[string]interface{}{
				"intervalMinutes": ch.IntervalMin,
				"overdueMinutes":  ch.OverdueMin,
			})
		}
		// Only send checkInMissed once per overdue period (debounce).
		if since >= overdueMs && now-ch.MissedNotifiedAt > overdueMs {
			ch.MissedNotifiedAt = now
			missedPayload := map[string]interface{}{
				"socketId":       socketID,
				"userId":         user.UserID,
				"displayName":    user.DisplayName,
				"lastCheckInAt":  lastAt,
				"overdueMinutes": ch.OverdueMin,
			}
			h.emitToVisible(user, "checkInMissed", missedPayload)
		}
	})
}

// cleanupExpiredStatusMessages clears status messages whose expiry time has passed
// and broadcasts the cleared status to visible peers.
func (h *Hub) cleanupExpiredStatusMessages() {
	now := time.Now().UnixMilli()
	h.Cache.ForEachActiveUser(func(_ string, user *cache.ActiveUser) {
		if user.StatusMessage == "" || user.StatusExpiresAt == 0 {
			return
		}
		if now < user.StatusExpiresAt {
			return
		}
		user.StatusMessage = ""
		user.StatusExpiresAt = 0
		// Best-effort DB clear
		go func(uid string) {
			_, _ = h.pool.DB.ExecContext(context.Background(),
				`UPDATE users SET status_message=NULL, status_expires_at=NULL WHERE id=$1`, uid)
		}(user.UserID)
		// Notify visible peers
		sanitized := h.Cache.SanitizeUser(user)
		sanitized["online"] = true
		h.emitToVisibleAndSelf(user, "userUpdate", sanitized)
	})
}

// checkHaventMovedAlerts sends a gentle nudge to guardians when a ward has been
// stationary for 45–240 minutes. Rate-limited to once per 60 min per ward.
func (h *Hub) checkHaventMovedAlerts() {
	now := time.Now().UnixMilli()
	const minStillMs = 45 * 60 * 1000  // 45 min
	const maxStillMs = 4 * 60 * 60 * 1000 // 4 hours (after this assume intentional)
	const cooldownMs = 60 * 60 * 1000  // 1 hour between alerts

	h.Cache.ForEachActiveUser(func(_ string, user *cache.ActiveUser) {
		if user.MotionClass != "still" {
			return
		}
		stillMs := now - user.LastMoveAt
		if stillMs < minStillMs || stillMs > maxStillMs {
			return
		}
		if now-user.GentleAlertSentAt < cooldownMs {
			return
		}
		guardians := h.Cache.GetWardToGuardians(user.UserID)
		if len(guardians) == 0 {
			return
		}
		user.GentleAlertSentAt = now
		payload := map[string]interface{}{
			"userId":       user.UserID,
			"displayName":  user.DisplayName,
			"minutesStill": stillMs / 60000,
		}
		for guardianID := range guardians {
			if sid := h.Cache.GetUserIdToSocketId(guardianID); sid != "" {
				h.SendToClient(sid, "gentleAlert", payload)
			}
		}
	})
}

// cleanupExpiredSecretMessages deletes secret messages older than 7 days.
func (h *Hub) cleanupExpiredSecretMessages() {
	_, err := h.pool.DB.ExecContext(context.Background(),
		`DELETE FROM secret_messages WHERE created_at < NOW() - INTERVAL '7 days'`,
	)
	if err != nil {
		slog.Error("Failed to purge expired secret messages", "error", err)
	}
}
