package ws

import (
	"encoding/json"
	"log/slog"
)

// EncodeMessageOptimized encodes an event+data pair to the wire format.
func EncodeMessageOptimized(event string, data interface{}) ([]byte, error) {
	msg := map[string]interface{}{
		"event": event,
		"data":  data,
	}
	return json.Marshal(msg)
}

// DecodeMessageOptimized decodes a raw wire message into event name and data map.
func DecodeMessageOptimized(raw []byte) (string, map[string]interface{}, error) {
	var msg map[string]interface{}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return "", nil, err
	}

	event := ""
	if e, ok := msg["event"].(string); ok {
		event = e
	}

	data := make(map[string]interface{})
	if d, ok := msg["data"].(map[string]interface{}); ok {
		data = d
	}

	return event, data, nil
}

// LogPayloadSize logs the byte size of a payload for monitoring/debugging.
func LogPayloadSize(event string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	slog.Debug("Payload size", "event", event, "bytes", len(data))
}
