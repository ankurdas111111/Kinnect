package ws

import "sync"

const rollingBufCap = 120 // 30 min at 15s update rate

// RollingEntry is a single position snapshot stored in the ring buffer.
type RollingEntry struct {
	Lat        float64
	Lng        float64
	SpeedMs    float64
	BatteryPct *int
	Ts         int64 // UnixMilli
}

// rollingBuffer is a fixed-size circular buffer; safe for concurrent use.
type rollingBuffer struct {
	entries [rollingBufCap]RollingEntry
	head    int // next write index
	count   int
	mu      sync.Mutex
}

func (rb *rollingBuffer) push(e RollingEntry) {
	rb.mu.Lock()
	rb.entries[rb.head] = e
	rb.head = (rb.head + 1) % rollingBufCap
	if rb.count < rollingBufCap {
		rb.count++
	}
	rb.mu.Unlock()
}

// snapshot returns entries in chronological order (oldest → newest).
func (rb *rollingBuffer) snapshot() []RollingEntry {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.count == 0 {
		return nil
	}
	out := make([]RollingEntry, rb.count)
	if rb.count < rollingBufCap {
		copy(out, rb.entries[:rb.count])
	} else {
		n := copy(out, rb.entries[rb.head:])
		copy(out[n:], rb.entries[:rb.head])
	}
	return out
}

// Hub-level methods — thin wrappers that own the map lock.

func (h *Hub) pushRollingEntry(userID string, e RollingEntry) {
	h.rollingBufMu.Lock()
	rb := h.rollingBufs[userID]
	if rb == nil {
		rb = &rollingBuffer{}
		h.rollingBufs[userID] = rb
	}
	h.rollingBufMu.Unlock()
	rb.push(e) // rb-level lock, outside hub lock
}

func (h *Hub) rollingSnapshot(userID string) []RollingEntry {
	h.rollingBufMu.RLock()
	rb := h.rollingBufs[userID]
	h.rollingBufMu.RUnlock()
	if rb == nil {
		return nil
	}
	return rb.snapshot()
}

func (h *Hub) deleteRollingBuf(userID string) {
	h.rollingBufMu.Lock()
	delete(h.rollingBufs, userID)
	h.rollingBufMu.Unlock()
}
