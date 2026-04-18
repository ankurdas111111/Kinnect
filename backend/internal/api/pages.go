package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"kinnect-v3/internal/auth"
	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/db"
)

// PagesHandler handles CSRF, Me, LiveToken, WatchToken.
type PagesHandler struct {
	cache *cache.Cache
	db    *sql.DB
}

// Csrf handles GET /api/csrf.
func (h *PagesHandler) Csrf(w http.ResponseWriter, r *http.Request) {
	sess := auth.GetSession(r)
	token := ""
	if sess != nil {
		token = sess.CsrfToken
	}
	writeJSON(w, http.StatusOK, map[string]any{"csrfToken": token})
}

// Me handles GET /api/me.
func (h *PagesHandler) Me(w http.ResponseWriter, r *http.Request) {
	sess := auth.GetSession(r)
	if sess == nil || sess.User == nil || sess.User.ID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "Not authenticated"})
		return
	}
	userID := sess.User.ID
	role := sess.User.Role
	if role == "" {
		role = "user"
	}
	ud := h.cache.GetUser(userID)
	displayName := h.cache.GetDisplayName(userID)
	shareCode := ""
	var email, mobile string
	if ud != nil {
		shareCode = ud.ShareCode
		if ud.Email != nil {
			email = *ud.Email
		}
		if ud.Mobile != nil {
			mobile = *ud.Mobile
		}
		if role == "" {
			role = ud.Role
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"userId":      userID,
		"displayName": displayName,
		"role":        role,
		"shareCode":   shareCode,
		"email":       email,
		"mobile":      mobile,
	})
}

// LiveToken handles GET /api/live/{token}.
func (h *PagesHandler) LiveToken(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	entry := h.cache.GetLiveToken(token)
	if entry == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "expired": true})
		return
	}
	now := time.Now().UnixMilli()
	if entry.ExpiresAt != nil && *entry.ExpiresAt <= now {
		h.cache.DeleteLiveToken(token)
		_ = db.DeleteLiveToken(context.Background(), h.db, token)
		sharedBy := h.cache.GetDisplayName(entry.UserID)
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": false, "expired": true, "sharedBy": sharedBy,
		})
		return
	}
	sharedBy := h.cache.GetDisplayName(entry.UserID)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "token": token, "sharedBy": sharedBy, "expired": false,
	})
}

// WatchToken handles GET /api/watch/{token}.
func (h *PagesHandler) WatchToken(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	entry := h.cache.GetWatchToken(token)
	if entry == nil || entry.Exp < time.Now().UnixMilli() {
		if entry != nil {
			h.cache.DeleteWatchToken(token)
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "expired": true})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "token": token})
}

// SecretChatMessage is the shape returned by GET /api/m/{token}.
// Only encrypted fields are included — the server never has plaintext.
type SecretChatMessage struct {
	Ciphertext  string    `json:"ciphertext"`
	IV          string    `json:"iv"`
	Salt        string    `json:"salt"`
	FromOwner   bool      `json:"fromOwner"` // true = sent by the link creator
	CreatedAt   time.Time `json:"createdAt"`
}

// MInvite handles GET /api/m/{token}.
// No authentication required — ciphertext is worthless without the PIN,
// so serving it publicly is safe. The PIN is the only security layer.
func (h *PagesHandler) MInvite(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	invite := h.cache.GetSecretChatInvite(token)
	if invite == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "expired": true})
		return
	}
	if time.Now().UnixMilli() > invite.ExpiresAt {
		h.cache.DeleteSecretChatInvite(token)
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "expired": true})
		return
	}

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT sender_id, ciphertext, iv, salt, created_at
		 FROM secret_messages
		 WHERE (sender_id = $1 AND receiver_id = $2)
		    OR (sender_id = $2 AND receiver_id = $1)
		 ORDER BY created_at ASC LIMIT 50`,
		invite.OwnerID, invite.PeerID,
	)
	if err != nil {
		slog.Error("MInvite: failed to query secret messages", "error", err)
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      false,
			"expired": false,
			"error":   "temporarily_unavailable",
		})
		return
	}
	defer rows.Close()

	var msgs []SecretChatMessage
	for rows.Next() {
		var senderID string
		var m SecretChatMessage
		if err := rows.Scan(&senderID, &m.Ciphertext, &m.IV, &m.Salt, &m.CreatedAt); err != nil {
			continue
		}
		m.FromOwner = senderID == invite.OwnerID
		msgs = append(msgs, m)
	}
	if msgs == nil {
		msgs = []SecretChatMessage{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"messages": msgs,
	})
}

// MInviteReply handles POST /api/m/{token}.
// Allows the invite recipient (peer) to send an encrypted reply without a session.
// The token proves access; the PIN on the message is the security layer.
func (h *PagesHandler) MInviteReply(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	invite := h.cache.GetSecretChatInvite(token)
	if invite == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "expired": true})
		return
	}
	if time.Now().UnixMilli() > invite.ExpiresAt {
		h.cache.DeleteSecretChatInvite(token)
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "expired": true})
		return
	}

	var body struct {
		Ciphertext string `json:"ciphertext"`
		IV         string `json:"iv"`
		Salt       string `json:"salt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		len(body.Ciphertext) == 0 || len(body.Ciphertext) > 10000 ||
		body.IV == "" || body.Salt == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid"})
		return
	}

	var msgID int64
	err := h.db.QueryRowContext(r.Context(),
		`INSERT INTO secret_messages (sender_id, receiver_id, ciphertext, iv, salt)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		invite.PeerID, invite.OwnerID, body.Ciphertext, body.IV, body.Salt,
	).Scan(&msgID)
	if err != nil {
		slog.Error("MInviteReply: insert failed", "error", err)
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": "temporarily_unavailable"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": msgID})
}
