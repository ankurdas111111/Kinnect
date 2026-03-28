package ws

import (
	"encoding/json"
)

// handleGetNetworkGraph handles the "getNetworkGraph" event.
// Builds a graph of nodes and edges from cache only (no DB queries).
func (h *Hub) handleGetNetworkGraph(c *Client, _ json.RawMessage) {
	callerUserID := c.UserID()
	callerSid := c.ID()

	me := h.Cache.GetActiveUser(callerSid)
	displayName := h.Cache.GetDisplayName(callerUserID)

	type nodeEntry struct {
		ID      string
		Name    string
		Role    string
		Online  bool
		LastLat *float64
		LastLng *float64
	}

	nodeMap := make(map[string]*nodeEntry)

	// Self node
	var selfLat, selfLng *float64
	if me != nil {
		selfLat = me.Latitude
		selfLng = me.Longitude
	}
	nodeMap[callerUserID] = &nodeEntry{
		ID:      callerUserID,
		Name:    displayName,
		Role:    "self",
		Online:  true,
		LastLat: selfLat,
		LastLng: selfLng,
	}

	// Helper to get online status and last position
	enrichNode := func(userID string) (online bool, lat, lng *float64) {
		if sid := h.Cache.GetUserIdToSocketId(userID); sid != "" {
			if u := h.Cache.GetActiveUser(sid); u != nil {
				return true, u.Latitude, u.Longitude
			}
		}
		return false, nil, nil
	}

	ensureNode := func(userID, role string) {
		if _, ok := nodeMap[userID]; ok {
			return // already added with a more specific role
		}
		name := h.Cache.GetDisplayName(userID)
		online, lat, lng := enrichNode(userID)
		nodeMap[userID] = &nodeEntry{
			ID:      userID,
			Name:    name,
			Role:    role,
			Online:  online,
			LastLat: lat,
			LastLng: lng,
		}
	}

	type edge struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Kind   string `json:"kind"` // "contact" | "guardian" | "ward" | "room"
	}
	var edges []edge

	// Contacts
	contacts := h.Cache.GetContactsForUser(callerUserID)
	for _, uid := range contacts {
		ensureNode(uid, "contact")
		edges = append(edges, edge{Source: callerUserID, Target: uid, Kind: "contact"})
	}

	// Guardianships: I am guardian of wards
	guardianships := h.Cache.GetGuardianshipsAsGuardian(callerUserID)
	for _, g := range guardianships {
		wardID, _ := g["wardId"].(string)
		if wardID != "" {
			ensureNode(wardID, "ward")
			edges = append(edges, edge{Source: callerUserID, Target: wardID, Kind: "ward"})
		}
	}

	// WardToGuardians: who is my guardian
	guardians := h.Cache.GetGuardianshipsAsWard(callerUserID)
	for _, g := range guardians {
		guardianID, _ := g["guardianId"].(string)
		if guardianID != "" {
			ensureNode(guardianID, "guardian")
			edges = append(edges, edge{Source: guardianID, Target: callerUserID, Kind: "guardian"})
		}
	}

	// Room members
	rooms := h.Cache.GetUserRooms(callerUserID)
	for _, code := range rooms {
		room := h.Cache.GetRoom(code)
		if room == nil {
			continue
		}
		for memberID := range room.Members {
			if memberID == callerUserID {
				continue
			}
			ensureNode(memberID, "roommate")
			edges = append(edges, edge{Source: callerUserID, Target: memberID, Kind: "room"})
		}
	}

	// Serialize nodes
	nodes := make([]map[string]interface{}, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, map[string]interface{}{
			"id":      n.ID,
			"name":    n.Name,
			"role":    n.Role,
			"online":  n.Online,
			"lastLat": n.LastLat,
			"lastLng": n.LastLng,
		})
	}

	c.Send("networkGraph", map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	})
}
