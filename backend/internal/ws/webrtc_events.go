package ws

import "encoding/json"

// handleWebRTCOffer relays an SDP offer from caller to callee.
// The offer doubles as the "incoming call" signal — callee shows UI on receipt.
func (h *Hub) handleWebRTCOffer(c *Client, data json.RawMessage) {
	m := toMap(data)
	targetUserID, _ := m["targetUserID"].(string)
	sdp, _ := m["sdp"].(string)
	if targetUserID == "" || sdp == "" {
		return
	}
	socketID := h.Cache.GetUserIdToSocketId(targetUserID)
	if socketID == "" {
		return
	}
	fromName := c.UserID()
	if au := h.Cache.GetActiveUser(c.UserID()); au != nil {
		fromName = au.DisplayName
	}
	h.SendToClient(socketID, "webrtc:offer", map[string]any{
		"fromUserID": c.UserID(),
		"fromName":   fromName,
		"sdp":        sdp,
	})
}

// handleWebRTCAnswer relays the SDP answer from callee back to caller.
func (h *Hub) handleWebRTCAnswer(c *Client, data json.RawMessage) {
	m := toMap(data)
	targetUserID, _ := m["targetUserID"].(string)
	sdp, _ := m["sdp"].(string)
	if targetUserID == "" || sdp == "" {
		return
	}
	socketID := h.Cache.GetUserIdToSocketId(targetUserID)
	if socketID == "" {
		return
	}
	h.SendToClient(socketID, "webrtc:answer", map[string]any{
		"fromUserID": c.UserID(),
		"sdp":        sdp,
	})
}

// handleWebRTCIce relays an ICE candidate between peers for NAT traversal.
func (h *Hub) handleWebRTCIce(c *Client, data json.RawMessage) {
	m := toMap(data)
	targetUserID, _ := m["targetUserID"].(string)
	candidate := m["candidate"]
	if targetUserID == "" || candidate == nil {
		return
	}
	socketID := h.Cache.GetUserIdToSocketId(targetUserID)
	if socketID == "" {
		return
	}
	h.SendToClient(socketID, "webrtc:ice", map[string]any{
		"fromUserID": c.UserID(),
		"candidate":  candidate,
	})
}

// handleWebRTCHangup notifies the remote peer that the call has ended.
func (h *Hub) handleWebRTCHangup(c *Client, data json.RawMessage) {
	m := toMap(data)
	targetUserID, _ := m["targetUserID"].(string)
	if targetUserID == "" {
		return
	}
	socketID := h.Cache.GetUserIdToSocketId(targetUserID)
	if socketID == "" {
		return
	}
	h.SendToClient(socketID, "webrtc:hangup", map[string]any{
		"fromUserID": c.UserID(),
	})
}
