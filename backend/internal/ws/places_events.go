package ws

import (
	"context"
	"encoding/json"

	"github.com/lib/pq"
)

type syncPlacePayload struct {
	Action     string  `json:"action"` // "add" | "remove"
	PlaceID    string  `json:"placeId"`
	Name       string  `json:"name"`
	Icon       string  `json:"icon"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Visibility string  `json:"visibility"` // "personal" | "universal" | "room"
	RoomCode   string  `json:"roomCode"`
}

// handleSyncPlace broadcasts a non-personal pin add/remove to the correct recipients:
//   - "universal": all visible contacts
//   - "room": only members of the specified room
//   - "personal": never broadcast (client should not emit for personal pins)
//
// The DB write happens via REST before the client emits this WS event.
func (h *Hub) handleSyncPlace(c *Client, data json.RawMessage) {
	user := h.Cache.GetActiveUser(c.ID())
	if user == nil {
		return
	}

	var p syncPlacePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return
	}

	if p.Action != "add" && p.Action != "remove" {
		return
	}
	if p.PlaceID == "" {
		return
	}
	// Never broadcast personal pins.
	if p.Visibility == "personal" || p.Visibility == "" {
		return
	}

	payload := map[string]interface{}{
		"action":     p.Action,
		"placeId":    p.PlaceID,
		"userId":     user.UserID,
		"name":       p.Name,
		"icon":       p.Icon,
		"latitude":   p.Latitude,
		"longitude":  p.Longitude,
		"visibility": p.Visibility,
		"roomCode":   p.RoomCode,
	}

	var targetSids []string
	if p.Visibility == "room" && p.RoomCode != "" {
		// Broadcast only to online members of that room (excluding sender).
		all := h.Cache.GetRoomMemberSocketIDs(p.RoomCode)
		for _, sid := range all {
			if sid != c.ID() {
				targetSids = append(targetSids, sid)
			}
		}
	} else if p.Visibility == "universal" {
		targetSids = h.Cache.GetVisibleSocketIDs(user)
	}

	for _, sid := range targetSids {
		h.SendToClient(sid, "syncPlace", payload)
	}
}

// emitExistingPlaces sends all pins visible to a newly connected client:
//   - Universal pins owned by any visible contact
//   - Room-scoped pins in any room the user is a member of (from any member, excluding self)
//
// Called from handleRegister so the client immediately sees shared pins on the map.
func (h *Hub) emitExistingPlaces(c *Client, userID string) {
	visibleSet := h.Cache.GetVisibleSet(userID)
	userRoomCodes := h.Cache.GetUserRooms(userID)

	// Collect visible contactIDs (exclude self) for universal-pin lookup.
	contactIDs := make([]string, 0, len(visibleSet))
	for uid := range visibleSet {
		if uid != userID {
			contactIDs = append(contactIDs, uid)
		}
	}

	if len(contactIDs) == 0 && len(userRoomCodes) == 0 {
		return
	}

	if userRoomCodes == nil {
		userRoomCodes = []string{}
	}

	rows, err := h.pool.DB.QueryContext(context.Background(),
		`SELECT id, user_id, name, icon, latitude, longitude,
		        COALESCE(visibility,'personal'), COALESCE(room_code,'')
		 FROM saved_places
		 WHERE user_id != $3
		   AND (
		     (COALESCE(visibility,'personal') = 'universal' AND user_id = ANY($1::uuid[]))
		     OR (COALESCE(visibility,'personal') = 'room' AND room_code = ANY($2::varchar[]))
		   )
		 ORDER BY created_at DESC
		 LIMIT 200`,
		pq.Array(contactIDs), pq.Array(userRoomCodes), userID)
	if err != nil {
		return
	}
	defer rows.Close()

	type placeItem struct {
		PlaceID    string  `json:"placeId"`
		UserID     string  `json:"userId"`
		Name       string  `json:"name"`
		Icon       string  `json:"icon"`
		Latitude   float64 `json:"latitude"`
		Longitude  float64 `json:"longitude"`
		Visibility string  `json:"visibility"`
		RoomCode   string  `json:"roomCode,omitempty"`
	}

	var places []placeItem
	for rows.Next() {
		var p placeItem
		if err := rows.Scan(&p.PlaceID, &p.UserID, &p.Name, &p.Icon, &p.Latitude, &p.Longitude, &p.Visibility, &p.RoomCode); err != nil {
			continue
		}
		places = append(places, p)
	}
	if len(places) == 0 {
		return
	}
	c.Send("existingPlaces", places)
}
