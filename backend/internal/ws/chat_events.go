package ws

import (
	"context"
	"encoding/json"
	"time"
)

type sendSecretMsgPayload struct {
	ReceiverID string `json:"receiverId"`
	Ciphertext string `json:"ciphertext"`
	IV         string `json:"iv"`
	Salt       string `json:"salt"`
}

// handleSendSecretMsg stores an E2E-encrypted message and delivers it to the receiver if online.
// The server never sees plaintext — only ciphertext, IV, and salt are stored.
func (h *Hub) handleSendSecretMsg(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("sendSecretMsg", 20) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}

	var p sendSecretMsgPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return
	}

	// Authorization: receiver must be visible to sender (contact, room-mate, or guardian).
	visible := h.Cache.GetVisibleSet(user.UserID)
	if !visible[p.ReceiverID] {
		return
	}

	// Validate encrypted fields — non-empty, reasonable size.
	if len(p.Ciphertext) == 0 || len(p.Ciphertext) > 10000 {
		return
	}
	if len(p.IV) == 0 || len(p.Salt) == 0 {
		return
	}

	// Persist to DB.
	var msgID int64
	err := h.pool.DB.QueryRowContext(context.Background(),
		`INSERT INTO secret_messages (sender_id, receiver_id, ciphertext, iv, salt)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		user.UserID, p.ReceiverID, p.Ciphertext, p.IV, p.Salt,
	).Scan(&msgID)
	if err != nil {
		return
	}

	now := time.Now().UTC()

	// Ack to sender.
	c.Send("secretMsgSent", map[string]interface{}{
		"id":         msgID,
		"receiverId": p.ReceiverID,
		"ciphertext": p.Ciphertext,
		"iv":         p.IV,
		"salt":       p.Salt,
		"createdAt":  now,
	})

	// Deliver to receiver if online.
	if receiverSocketID := h.Cache.GetUserIdToSocketId(p.ReceiverID); receiverSocketID != "" {
		h.SendToClient(receiverSocketID, "secretMsgReceived", map[string]interface{}{
			"id":         msgID,
			"senderId":   user.UserID,
			"ciphertext": p.Ciphertext,
			"iv":         p.IV,
			"salt":       p.Salt,
			"createdAt":  now,
		})
	}
}

type getSecretMsgsPayload struct {
	PeerID string `json:"peerId"`
	Limit  int    `json:"limit"`
}

// handleGetSecretMsgs fetches the conversation history between the caller and a peer.
// Returns only ciphertext — decryption happens client-side.
func (h *Hub) handleGetSecretMsgs(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("getSecretMsgs", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}

	var p getSecretMsgsPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return
	}
	if p.Limit <= 0 || p.Limit > 50 {
		p.Limit = 20
	}

	// Authorization: peer must be visible to requester.
	visible := h.Cache.GetVisibleSet(user.UserID)
	if !visible[p.PeerID] {
		return
	}

	rows, err := h.pool.DB.QueryContext(context.Background(),
		`SELECT id, sender_id, receiver_id, ciphertext, iv, salt, seen_at, created_at
		 FROM secret_messages
		 WHERE (sender_id = $1 AND receiver_id = $2)
		    OR (sender_id = $2 AND receiver_id = $1)
		 ORDER BY created_at DESC LIMIT $3`,
		user.UserID, p.PeerID, p.Limit,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	type msgRow struct {
		ID         int64      `json:"id"`
		SenderID   string     `json:"senderId"`
		ReceiverID string     `json:"receiverId"`
		Ciphertext string     `json:"ciphertext"`
		IV         string     `json:"iv"`
		Salt       string     `json:"salt"`
		SeenAt     *time.Time `json:"seenAt"`
		CreatedAt  time.Time  `json:"createdAt"`
	}
	var msgs []msgRow
	for rows.Next() {
		var m msgRow
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Ciphertext, &m.IV, &m.Salt, &m.SeenAt, &m.CreatedAt); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	if msgs == nil {
		msgs = []msgRow{} // send empty array not null
	}

	c.Send("secretMsgsHistory", map[string]interface{}{
		"peerId":   p.PeerID,
		"messages": msgs,
	})
}

// handleDeleteSecretMsg lets a sender delete one of their own secret messages.
func (h *Hub) handleDeleteSecretMsg(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("deleteSecretMsg", 30) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}

	var p struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return
	}

	// Only allow deleting own messages (WHERE sender_id = user).
	h.pool.DB.ExecContext(context.Background(),
		`DELETE FROM secret_messages WHERE id = $1 AND sender_id = $2`,
		p.ID, user.UserID,
	)
	c.Send("secretMsgDeleted", map[string]interface{}{"id": p.ID})
}

// handleMarkSecretMsgSeen marks a message as seen when the receiver successfully decrypts it.
// Client sends: { "msgId": 123 }
// Emits "secretMsgSeen" to the original sender: { "id": 123, "seenAt": "2026-04-19T..." }
func (h *Hub) handleMarkSecretMsgSeen(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("markSecretMsgSeen", 30) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}

	var p struct {
		MsgID int64 `json:"msgId"`
	}
	if err := json.Unmarshal(data, &p); err != nil || p.MsgID <= 0 {
		return
	}

	// Only the receiver can mark as seen; seen_at IS NULL prevents double-marking.
	var senderID string
	var seenAt time.Time
	err := h.pool.DB.QueryRowContext(context.Background(),
		`UPDATE secret_messages
		 SET seen_at = NOW()
		 WHERE id = $1 AND receiver_id = $2 AND seen_at IS NULL
		 RETURNING sender_id, seen_at`,
		p.MsgID, user.UserID,
	).Scan(&senderID, &seenAt)
	if err != nil {
		return // not found, caller is not receiver, or already seen
	}

	// Notify the sender if they are online.
	if senderSocketID := h.Cache.GetUserIdToSocketId(senderID); senderSocketID != "" {
		h.SendToClient(senderSocketID, "secretMsgSeen", map[string]interface{}{
			"id":     p.MsgID,
			"seenAt": seenAt.UTC().Format(time.RFC3339),
		})
	}
}

// handleCreateSecretChatInvite generates a short-lived token that lets anyone fetch
// the ciphertext of a secret conversation via GET /api/m/{token}. No auth is required
// to redeem — the PIN is the security layer (ciphertext is worthless without it).
func (h *Hub) handleCreateSecretChatInvite(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("createSecretChatInvite", 5) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}

	var p struct {
		PeerID string `json:"peerId"`
		Nonce  string `json:"nonce"`
	}
	if err := json.Unmarshal(data, &p); err != nil || p.PeerID == "" {
		return
	}

	token := generateLiveToken() // reuses same 22-char base64url impl from auth_events.go
	expiresAt := time.Now().Add(24 * time.Hour).UnixMilli()
	h.Cache.AddSecretChatInvite(token, user.UserID, p.PeerID, expiresAt)

	c.Send("secretChatInviteCreated", map[string]interface{}{
		"token": token,
		"nonce": p.Nonce,
	})
}

