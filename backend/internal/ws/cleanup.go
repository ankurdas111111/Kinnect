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

// safeLoop runs fn(ctx) on each ticker tick, recovering from panics and restarting
// the loop with exponential backoff (5s → 10s → 20s … capped at 60s).
// It stops cleanly when ctx is done.
func safeLoop(ctx context.Context, interval time.Duration, name string, fn func(ctx context.Context)) {
	go func() {
		panicCount := 0
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicCount++
						if panicCount > 3 {
							slog.Error("cleanup goroutine panicking repeatedly",
								"routine", name, "panic", r, "panicCount", panicCount)
						} else {
							slog.Error("cleanup goroutine panicked",
								"routine", name, "panic", r)
						}
					}
				}()
				ticker := time.NewTicker(interval)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						fn(ctx)
					}
				}
			}()
			// Inner func returned — either ctx.Done() or panic recovery.
			select {
			case <-ctx.Done():
				return
			default:
				// Exponential backoff: 5 * 2^panicCount, capped at 60s.
				backoff := time.Duration(5*(1<<uint(panicCount-1))) * time.Second
				if backoff > 60*time.Second {
					backoff = 60 * time.Second
				}
				if backoff < 5*time.Second {
					backoff = 5 * time.Second
				}
				slog.Info("cleanup goroutine restarting after panic",
					"routine", name, "backoff", backoff)
				time.Sleep(backoff)
			}
		}
	}()
}

// StartCleanupRoutines starts periodic goroutines for cache and DB cleanup.
// All routines use safeLoop so a panic in one is recovered and the routine restarts.
func (h *Hub) StartCleanupRoutines(ctx context.Context) {
	safeLoop(ctx, 60*time.Second, "expireOfflineUsers", func(ctx context.Context) {
		h.cleanupExpireOfflineUsers()
	})
	safeLoop(ctx, 30*time.Second, "expireWatchTokens", func(ctx context.Context) {
		h.cleanupExpireWatchTokens()
	})
	safeLoop(ctx, 60*time.Second, "expireLiveTokens", func(ctx context.Context) {
		h.cleanupExpireLiveTokens()
	})
	safeLoop(ctx, 1*time.Hour, "cleanEmptyOldRooms", func(ctx context.Context) {
		h.cleanupEmptyOldRooms()
	})
	safeLoop(ctx, 60*time.Second, "expireRoomAdmins", func(ctx context.Context) {
		h.cleanupExpireRoomAdmins()
	})
	safeLoop(ctx, 60*time.Second, "expireGuardianships", func(ctx context.Context) {
		h.cleanupExpireGuardianships()
	})
	safeLoop(ctx, 60*time.Second, "checkInOverdue", func(ctx context.Context) {
		h.cleanupCheckInOverdue()
	})
	safeLoop(ctx, 24*time.Hour, "purgeStaleGuardianships", func(ctx context.Context) {
		h.cleanupStaleGuardianships(ctx)
	})
	safeLoop(ctx, 5*time.Minute, "haventMovedAlerts", func(ctx context.Context) {
		h.checkHaventMovedAlerts()
	})
	safeLoop(ctx, 60*time.Second, "expireStatusMessages", func(ctx context.Context) {
		h.cleanupExpiredStatusMessages()
	})
	safeLoop(ctx, 60*time.Second, "heartbeatWellness", func(ctx context.Context) {
		h.cleanupHeartbeatCheck()
	})
	safeLoop(ctx, 30*time.Second, "walkWithMe", func(ctx context.Context) {
		h.EvaluateWalk()
	})
	safeLoop(ctx, 5*time.Minute, "expireSecretChatInvites", func(ctx context.Context) {
		h.Cache.CollectExpiredSecretChatInvites(time.Now().UnixMilli())
	})
	safeLoop(ctx, 24*time.Hour, "purgeSecretMessages", func(ctx context.Context) {
		h.cleanupExpiredSecretMessages(ctx)
	})

	// M-3: purge expired HTTP sessions every 6 hours so the session table does
	// not accumulate indefinitely (connect-pg-simple / gorilla/sessions rows).
	safeLoop(ctx, 6*time.Hour, "cleanupExpiredSessions", func(ctx context.Context) {
		if _, err := h.pool.DB.ExecContext(ctx, `DELETE FROM "session" WHERE expire < NOW()`); err != nil {
			slog.Warn("cleanupExpiredSessions failed", "err", err)
		}
	})

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
			"roomCode":  roomCode,
			"userId":    userID,
			"role":      "member",
			"expiresAt": nil,
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

// cleanupStaleGuardianships purges old expired/revoked guardianship rows.
// ctx is threaded so the query can be cancelled on shutdown.
// L-6: retention window reduced from 30 days to 7 days.
func (h *Hub) cleanupStaleGuardianships(ctx context.Context) {
	_, err := h.pool.DB.ExecContext(ctx,
		`DELETE FROM guardianships
		 WHERE status IN ('expired', 'revoked')
		   AND created_at < extract(epoch from now() - interval '7 days')::bigint * 1000`)
	if err != nil {
		slog.Error("Failed to purge stale guardianships", "error", err)
	}
}

// checkInPendingUpdate holds a deferred write to ActiveUser.CheckIn fields,
// collected during the read-only ForEachActiveUser pass and applied afterwards
// under a write lock to avoid the data race (H-5).
type checkInPendingUpdate struct {
	socketID         string
	setRequestedAt   int64 // 0 = no change
	setMissedNotifyAt int64 // 0 = no change
	setLastCheckInAt int64 // 0 = no change
}

func (h *Hub) cleanupCheckInOverdue() {
	now := time.Now().UnixMilli()

	// Phase 1: read-only scan under RLock — collect what needs sending and what
	// fields need updating.  Do NOT mutate anything here (data race fix H-5).
	type pendingNotify struct {
		socketID    string
		user        *cache.ActiveUser // snapshot of pointer; values read under RLock
		sendRequest bool
		sendMissed  bool
	}

	var notifies []pendingNotify
	var updates []checkInPendingUpdate

	h.Cache.ForEachActiveUser(func(socketID string, user *cache.ActiveUser) {
		ch := &user.CheckIn
		if !ch.Enabled {
			return
		}
		lastAt := ch.LastCheckInAt
		needsBaselineSet := lastAt == 0

		var upd checkInPendingUpdate
		upd.socketID = socketID

		if needsBaselineSet {
			upd.setLastCheckInAt = now
			lastAt = now
		}

		intervalMs := int64(ch.IntervalMin) * 60 * 1000
		overdueMs := int64(ch.OverdueMin) * 60 * 1000
		since := now - lastAt

		pn := pendingNotify{socketID: socketID, user: user}

		if since >= intervalMs && now-ch.RequestedAt > intervalMs {
			upd.setRequestedAt = now
			pn.sendRequest = true
		}
		if since >= overdueMs && now-ch.MissedNotifiedAt > overdueMs {
			upd.setMissedNotifyAt = now
			pn.sendMissed = true
		}

		if upd.setRequestedAt != 0 || upd.setMissedNotifyAt != 0 || upd.setLastCheckInAt != 0 {
			updates = append(updates, upd)
		}
		if pn.sendRequest || pn.sendMissed {
			notifies = append(notifies, pn)
		}
	})

	// Phase 2: apply mutations under write lock.  GetActiveUser acquires its own
	// RLock, so we call it in a loop and apply changes via a targeted write block
	// using the exported Cache.UpdateCheckInTimestamps helper.
	// To avoid adding a new cache export, we inline the write via the
	// Cache.mu write-lock path that ForEachActiveUser comment says is safe to do
	// AFTER ForEachActiveUser returns (lock is released by then).
	for _, upd := range updates {
		h.Cache.ApplyCheckInUpdate(upd.socketID, upd.setLastCheckInAt, upd.setRequestedAt, upd.setMissedNotifyAt)
	}

	// Phase 3: send notifications (no lock held).
	for _, pn := range notifies {
		if pn.sendRequest {
			h.SendToClient(pn.socketID, "checkInRequest", map[string]interface{}{
				"intervalMinutes": pn.user.CheckIn.IntervalMin,
				"overdueMinutes":  pn.user.CheckIn.OverdueMin,
			})
		}
		if pn.sendMissed {
			missedPayload := map[string]interface{}{
				"socketId":       pn.socketID,
				"userId":         pn.user.UserID,
				"displayName":    pn.user.DisplayName,
				"lastCheckInAt":  pn.user.CheckIn.LastCheckInAt,
				"overdueMinutes": pn.user.CheckIn.OverdueMin,
			}
			h.emitToVisible(pn.user, "checkInMissed", missedPayload)
		}
	}
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

// gentleAlertPendingUpdate holds a deferred write to ActiveUser.GentleAlertSentAt,
// collected during the read-only ForEachActiveUser pass (H-5 data race fix).
type gentleAlertPendingUpdate struct {
	socketID string
	sentAt   int64
}

// checkHaventMovedAlerts sends a gentle nudge to guardians when a ward has been
// stationary for 45–240 minutes. Rate-limited to once per 60 min per ward.
func (h *Hub) checkHaventMovedAlerts() {
	now := time.Now().UnixMilli()
	const minStillMs = 45 * 60 * 1000     // 45 min
	const maxStillMs = 4 * 60 * 60 * 1000 // 4 hours (after this assume intentional)
	const cooldownMs = 60 * 60 * 1000     // 1 hour between alerts

	type pendingAlert struct {
		socketID    string
		userID      string
		displayName string
		minutesStill int64
		guardians   map[string]bool
	}

	var alerts []pendingAlert
	var updates []gentleAlertPendingUpdate

	// Phase 1: read-only scan — collect alerts and field update intentions.
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
		updates = append(updates, gentleAlertPendingUpdate{socketID: user.SocketID, sentAt: now})
		alerts = append(alerts, pendingAlert{
			socketID:     user.SocketID,
			userID:       user.UserID,
			displayName:  user.DisplayName,
			minutesStill: stillMs / 60000,
			guardians:    guardians,
		})
	})

	// Phase 2: apply GentleAlertSentAt mutations under write lock.
	for _, upd := range updates {
		h.Cache.ApplyGentleAlertSentAt(upd.socketID, upd.sentAt)
	}

	// Phase 3: send alerts (no lock held).
	for _, a := range alerts {
		payload := map[string]interface{}{
			"userId":       a.userID,
			"displayName":  a.displayName,
			"minutesStill": a.minutesStill,
		}
		for guardianID := range a.guardians {
			if sid := h.Cache.GetUserIdToSocketId(guardianID); sid != "" {
				h.SendToClient(sid, "gentleAlert", payload)
			}
		}
	}
}

// cleanupExpiredSecretMessages deletes secret messages older than 7 days,
// then enforces a per-conversation-pair cap of 200 messages (C-3).
// ctx is threaded so queries can be cancelled on shutdown.
func (h *Hub) cleanupExpiredSecretMessages(ctx context.Context) {
	// Step 1: TTL purge — remove messages older than 7 days.
	_, err := h.pool.DB.ExecContext(ctx,
		`DELETE FROM secret_messages WHERE created_at < NOW() - INTERVAL '7 days'`,
	)
	if err != nil {
		slog.Error("Failed to purge expired secret messages", "error", err)
		return
	}

	// Step 2: per-pair row cap — keep at most 200 messages per ordered pair
	// (A→B and B→A are treated as the same conversation via LEAST/GREATEST).
	_, err = h.pool.DB.ExecContext(ctx,
		`DELETE FROM secret_messages
		 WHERE id IN (
		     SELECT id FROM (
		         SELECT id,
		                ROW_NUMBER() OVER (
		                    PARTITION BY LEAST(sender_id, receiver_id),
		                                 GREATEST(sender_id, receiver_id)
		                    ORDER BY created_at DESC
		                ) AS rn
		         FROM secret_messages
		     ) ranked
		     WHERE rn > 200
		 )`,
	)
	if err != nil {
		slog.Error("Failed to enforce per-pair secret message cap", "error", err)
	}
}
