package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"kinnect-v3/internal/auth"
	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/db"
	"kinnect-v3/internal/shared"
)

// PagesHandler handles CSRF, Me, LiveToken, WatchToken, and profile mutations.
type PagesHandler struct {
	cache *cache.Cache
	db    *sql.DB
	store *auth.SessionStore
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

// UpdateProfile handles POST /api/profile/update.
func (h *PagesHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	sess := auth.GetSession(r)
	if sess == nil || sess.User == nil || sess.User.ID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "Not authenticated"})
		return
	}
	var req struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
		Mobile    string `json:"mobile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Invalid request"})
		return
	}
	firstName := shared.SanitizeString(strings.TrimSpace(req.FirstName), 64)
	lastName := shared.SanitizeString(strings.TrimSpace(req.LastName), 64)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	mobile := strings.TrimSpace(req.Mobile)

	if email != "" && !emailRegex.MatchString(email) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Invalid email address"})
		return
	}
	if mobile != "" && !mobileRegex.MatchString(mobile) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Mobile must be in +E.164 format"})
		return
	}

	if err := db.UpdateUserProfile(r.Context(), h.db, sess.User.ID, firstName, lastName, email, mobile); err != nil {
		slog.Error("UpdateProfile: db error", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "Failed to update profile"})
		return
	}
	h.cache.UpdateUserProfile(sess.User.ID, firstName, lastName, email, mobile)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ChangePassword handles POST /api/profile/password.
func (h *PagesHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	sess := auth.GetSession(r)
	if sess == nil || sess.User == nil || sess.User.ID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "Not authenticated"})
		return
	}
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Invalid request"})
		return
	}
	if len(req.NewPassword) < minPasswordBytes || len(req.NewPassword) > maxPasswordBytes {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Password must be 6–72 characters"})
		return
	}
	hash, err := db.GetUserPasswordHash(r.Context(), h.db, sess.User.ID)
	if err != nil || hash == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "Could not verify password"})
		return
	}
	if !auth.ComparePassword(hash, req.CurrentPassword) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "Current password is incorrect"})
		return
	}
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "Failed to hash password"})
		return
	}
	if err := db.UpdateUserPassword(r.Context(), h.db, sess.User.ID, newHash); err != nil {
		slog.Error("ChangePassword: db error", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "Failed to update password"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// DeleteAccount handles POST /api/profile/delete.
func (h *PagesHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	sess := auth.GetSession(r)
	if sess == nil || sess.User == nil || sess.User.ID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "Not authenticated"})
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Invalid request"})
		return
	}
	hash, err := db.GetUserPasswordHash(r.Context(), h.db, sess.User.ID)
	if err != nil || hash == "" || !auth.ComparePassword(hash, req.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "Incorrect password"})
		return
	}
	if err := db.DeleteUser(r.Context(), h.db, sess.User.ID); err != nil {
		slog.Error("DeleteAccount: db error", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "Failed to delete account"})
		return
	}
	// Destroy server-side session and clear cookie
	if sid := auth.GetSessionID(r); sid != "" {
		_ = h.store.Destroy(sid)
	}
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
