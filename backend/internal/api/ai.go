package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"kinnect-v3/internal/ai"
	"kinnect-v3/internal/auth"
	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/config"
)

const (
	aiQuestionMaxLen  = 500
	aiAskCooldown     = 8 * time.Second // per-user; protects the free OpenRouter tier
	aiRequestBudget   = 90 * time.Second
	aiMaxGlobalInFlig = 4 // cap concurrent agent runs across all users (free-tier friendly)
)

// AIHandler serves the Ask-the-Map copilot endpoints.
type AIHandler struct {
	db      *sql.DB
	cache   *cache.Cache
	agent   *ai.Agent
	enabled bool
	sem     chan struct{} // global in-flight limiter

	mu       sync.Mutex
	lastAsk  map[string]time.Time
	inFlight map[string]bool // per-user: at most one active run
}

// NewAIHandler wires the copilot. Disabled (returns 503) when no API key is set.
func NewAIHandler(cfg *config.Config, db *sql.DB, c *cache.Cache) *AIHandler {
	h := &AIHandler{
		db:       db,
		cache:    c,
		lastAsk:  make(map[string]time.Time),
		inFlight: make(map[string]bool),
		sem:      make(chan struct{}, aiMaxGlobalInFlig),
	}
	if cfg.OpenRouterAPIKey == "" {
		slog.Info("AI copilot disabled: OPENROUTER_API_KEY not set")
		return h
	}
	h.enabled = true
	client := ai.NewClient(cfg.OpenRouterAPIKey, cfg.AIModels)
	h.agent = &ai.Agent{Client: client, Tools: ai.Registry()}
	slog.Info("AI copilot enabled", "models", strings.Join(client.Models(), ","))
	return h
}

// Status handles GET /api/ai/status.
func (h *AIHandler) Status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "enabled": h.enabled})
}

// Ask handles POST /api/ai/ask — an SSE stream of agent progress.
// Events: status, tool_call, narrative_delta, directives, done, error.
func (h *AIHandler) Ask(w http.ResponseWriter, r *http.Request) {
	if !h.enabled {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "AI copilot is not configured"})
		return
	}
	sess := auth.GetSession(r)
	if sess == nil || sess.User == nil { // RequireAuth already guards; belt and braces
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "Not authenticated"})
		return
	}
	userID := sess.User.ID

	var body struct {
		Question string `json:"question"`
		Timezone string `json:"timezone"` // IANA name from the browser (optional)
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Invalid JSON body"})
		return
	}
	question := strings.TrimSpace(body.Question)
	// Count runes, not bytes, to match the UI's 500-character limit (a
	// non-Latin question is multi-byte and would otherwise be over-rejected).
	if question == "" || utf8.RuneCountInString(question) > aiQuestionMaxLen {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Question must be 1-500 characters"})
		return
	}

	// Cooldown + single-in-flight-per-user, taken together under the lock.
	h.mu.Lock()
	if last, ok := h.lastAsk[userID]; ok && time.Since(last) < aiAskCooldown {
		h.mu.Unlock()
		ai.RecordRejected("rate_limited")
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"ok": false, "error": fmt.Sprintf("Please wait %d seconds between questions", int(aiAskCooldown.Seconds())),
		})
		return
	}
	if h.inFlight[userID] {
		h.mu.Unlock()
		ai.RecordRejected("in_flight")
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"ok": false, "error": "You already have a question in progress",
		})
		return
	}
	h.lastAsk[userID] = time.Now()
	h.inFlight[userID] = true
	// Opportunistic cleanup so lastAsk cannot grow unbounded.
	if len(h.lastAsk) > 1000 {
		cutoff := time.Now().Add(-time.Hour)
		for id, t := range h.lastAsk {
			if t.Before(cutoff) {
				delete(h.lastAsk, id)
			}
		}
	}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.inFlight, userID)
		h.mu.Unlock()
	}()

	// Global concurrency cap — bounds total OpenRouter load and goroutines.
	select {
	case h.sem <- struct{}{}:
		defer func() { <-h.sem }()
	default:
		ai.RecordRejected("busy")
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"ok": false, "error": "The copilot is busy right now — please try again in a moment",
		})
		return
	}

	// Resolve the requester's timezone (falls back to UTC), so "this morning"
	// and rendered times match the user, not the server's zone.
	loc := time.UTC
	if body.Timezone != "" {
		if l, err := time.LoadLocation(body.Timezone); err == nil {
			loc = l
		}
	}

	// Build the family-scoped tool context — authorization happens HERE,
	// below the model. The visible set comes from the session user, then is
	// further gated by each target's sharing schedule and quiet hours to match
	// the live broadcast path (ws.flushPositionBroadcasts).
	visible := h.cache.GetVisibleSet(userID)
	requesterRooms := h.cache.GetUserRooms(userID)
	toolCtx := &ai.ToolContext{
		RequesterID:   userID,
		RequesterName: h.cache.GetDisplayName(userID),
		Visible:       make(map[string]string, len(visible)),
		Coarsen:       make(map[string]bool),
		DB:            h.db,
		Loc:           loc,
	}
	for id := range visible {
		if id != userID {
			// Sharing schedule: drop targets outside their current window for this requester.
			if h.cache.HasSharingSchedules(id) && !h.cache.IsScheduleVisible(id, userID, requesterRooms) {
				continue
			}
		}
		toolCtx.Visible[id] = h.cache.GetDisplayName(id)
	}
	h.applyQuietHoursCoarsening(r.Context(), toolCtx)

	// ── SSE setup ────────────────────────────────────────────────────────────
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	send := func(event string, data any) {
		payload, err := json.Marshal(data)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload)
		if err := rc.Flush(); err != nil {
			slog.Debug("ai: SSE flush failed (client gone?)", "error", err)
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), aiRequestBudget)
	defer cancel()

	started := time.Now()
	slog.Info("ai: ask", "user", userID, "question_len", utf8.RuneCountInString(question))

	result, err := h.agent.Run(ctx, toolCtx, question, func(ev ai.Event) {
		send(ev.Type, ev.Data)
	})
	if err != nil {
		// A client-initiated Stop cancels the request context — that's normal
		// user behavior, not an error to alert on.
		if errors.Is(err, context.Canceled) {
			ai.RecordRequest("canceled", time.Since(started).Seconds(), ai.Usage{})
			slog.Info("ai: run canceled by client", "user", userID)
			return
		}
		status := "error"
		msg := "The copilot hit a problem answering that. Please try again."
		switch {
		case strings.Contains(err.Error(), "rate limited"):
			status = "rate_limited"
			msg = "The AI provider is rate-limiting requests right now — try again in a minute."
		case errors.Is(err, context.DeadlineExceeded):
			status = "timeout"
			msg = "That took too long to answer. Please try a narrower question."
		}
		ai.RecordRequest(status, time.Since(started).Seconds(), ai.Usage{})
		slog.Error("ai: run failed", "user", userID, "status", status, "error", err)
		send("error", map[string]any{"message": msg})
		return
	}

	ai.RecordRequest("ok", time.Since(started).Seconds(), result.Usage)
	send("done", map[string]any{
		"model":        result.Model,
		"steps":        result.Steps,
		"duration_ms":  result.DurationMS,
		"tool_summary": ai.FormatToolSummary(result.Audit),
		"audit":        result.Audit,
		"narrative":    result.Narrative, // authoritative final text (deltas may drop chunk-split runes)
		"usage": map[string]int{
			"prompt_tokens":     result.Usage.PromptTokens,
			"completion_tokens": result.Usage.CompletionTokens,
		},
	})
}

// applyQuietHoursCoarsening marks visible targets whose coordinates must be
// coarsened: the target is in quiet hours AND the requester is not their
// guardian — mirroring ws.flushPositionBroadcasts. Quiet-hours settings live in
// the users table (also served for offline members), so read them from the DB.
func (h *AIHandler) applyQuietHoursCoarsening(ctx context.Context, tc *ai.ToolContext) {
	ids := make([]string, 0, len(tc.Visible))
	for id := range tc.Visible {
		if id != tc.RequesterID {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	qctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := h.db.QueryContext(qctx, `
		SELECT id, COALESCE(quiet_hours_start::text, ''), COALESCE(quiet_hours_end::text, '')
		FROM users
		WHERE id = ANY($1) AND quiet_hours_enabled = true`, pqUUIDArray(ids))
	if err != nil {
		slog.Warn("ai: quiet-hours lookup failed; proceeding without coarsening", "error", err)
		return
	}
	defer rows.Close()
	now := time.Now()
	for rows.Next() {
		var id, start, end string
		if err := rows.Scan(&id, &start, &end); err != nil {
			continue
		}
		if ai.QuietWindowActive(start, end, now) && !h.cache.IsGuardianOf(tc.RequesterID, id) {
			tc.Coarsen[id] = true
		}
	}
}

// pqUUIDArray renders UUID strings as a PostgreSQL text[] literal for ANY($1).
func pqUUIDArray(ids []string) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(id))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}
