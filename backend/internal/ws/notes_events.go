package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"kinnect-v3/internal/db"
	"kinnect-v3/internal/shared"
)

// handlePostRoomNote posts a note to a room's bulletin board. (F8)
// Payload in: { roomCode: string, body: string }
// Server→Client: roomNoteAdded → all online room members
func (h *Hub) handlePostRoomNote(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("postRoomNote", 10) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}

	roomCode, _ := m["roomCode"].(string)
	roomCode = strings.TrimSpace(strings.ToUpper(roomCode))
	if roomCode == "" {
		return
	}

	body, _ := m["body"].(string)
	body = shared.SanitizeString(strings.TrimSpace(body), 200)
	if body == "" {
		return
	}

	room := h.Cache.GetRoom(roomCode)
	if room == nil || !room.Members[user.UserID] {
		return
	}

	now := time.Now().UnixMilli()
	authorID := user.UserID
	authorName := user.DisplayName
	dbID := room.DbID

	h.offloadDB(func(ctx context.Context) {
		noteID, err := db.InsertRoomNote(ctx, h.pool.DB, dbID, authorID, body, now)
		if err != nil {
			slog.Warn("InsertRoomNote failed", "roomCode", roomCode, "userId", authorID, "error", err)
			return
		}

		// Enforce 20-note cap per room (async — non-critical, already offloaded pattern).
		go func() {
			if err := db.EnforceRoomNoteCap(context.Background(), h.pool.DB, dbID); err != nil {
				slog.Warn("EnforceRoomNoteCap failed", "roomCode", roomCode, "error", err)
			}
		}()

		payload := map[string]interface{}{
			"id":         noteID,
			"roomCode":   roomCode,
			"authorId":   authorID,
			"authorName": authorName,
			"body":       body,
			"createdAt":  now,
		}

		// room.Members iteration must happen on the hub loop.
		h.RunOnLoop(func() {
			for mid := range room.Members {
				if cli := h.GetClientByUserID(mid); cli != nil {
					h.SendToClient(cli.ID(), "roomNoteAdded", payload)
				}
			}
		})
	})
}

// handleDeleteRoomNote deletes a note from a room's bulletin board. (F8)
// Only the note's author can delete it (enforced by the DB query).
// Payload in: { noteId: string, roomCode: string }
// Server→Client: roomNoteDeleted → all online room members
func (h *Hub) handleDeleteRoomNote(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("deleteRoomNote", 20) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}

	noteID, _ := m["noteId"].(string)
	if noteID == "" {
		return
	}
	roomCode, _ := m["roomCode"].(string)
	roomCode = strings.TrimSpace(strings.ToUpper(roomCode))
	if roomCode == "" {
		return
	}

	room := h.Cache.GetRoom(roomCode)
	if room == nil || !room.Members[user.UserID] {
		return
	}

	authorID := user.UserID
	payload := map[string]interface{}{
		"noteId":   noteID,
		"roomCode": roomCode,
	}

	h.offloadDB(func(ctx context.Context) {
		if err := db.DeleteRoomNote(ctx, h.pool.DB, noteID, authorID); err != nil {
			slog.Warn("DeleteRoomNote failed", "noteId", noteID, "userId", authorID, "error", err)
			return
		}
		// room.Members iteration must happen on the hub loop.
		h.RunOnLoop(func() {
			for mid := range room.Members {
				if cli := h.GetClientByUserID(mid); cli != nil {
					h.SendToClient(cli.ID(), "roomNoteDeleted", payload)
				}
			}
		})
	})
}

// handleGetRoomNotes returns the last 20 notes for a room. (F8)
// Payload in: { roomCode: string }
// Server→Client: roomNotes → self
func (h *Hub) handleGetRoomNotes(c *Client, data json.RawMessage) {
	if !c.CheckRateLimit("getRoomNotes", 20) {
		return
	}
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}
	m := toMap(data)
	if m == nil {
		return
	}

	roomCode, _ := m["roomCode"].(string)
	roomCode = strings.TrimSpace(strings.ToUpper(roomCode))
	if roomCode == "" {
		return
	}

	room := h.Cache.GetRoom(roomCode)
	if room == nil || !room.Members[user.UserID] {
		return
	}

	dbID := room.DbID
	h.offloadDB(func(ctx context.Context) {
		rows, err := db.GetRoomNotes(ctx, h.pool.DB, dbID)
		if err != nil {
			c.Send("roomNotes", map[string]interface{}{"roomCode": roomCode, "notes": []interface{}{}})
			return
		}

		notes := make([]map[string]interface{}, 0, len(rows))
		for _, r := range rows {
			notes = append(notes, map[string]interface{}{
				"id":         r.ID,
				"authorId":   r.AuthorID,
				"authorName": r.AuthorName,
				"body":       r.Body,
				"createdAt":  r.CreatedAt,
			})
		}
		c.Send("roomNotes", map[string]interface{}{"roomCode": roomCode, "notes": notes})
	})
}
