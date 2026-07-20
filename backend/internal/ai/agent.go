package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"kinnect-v3/internal/shared"
)

const (
	maxAgentSteps      = 5
	maxToolResultBytes = 16 << 10 // cap on a single tool result fed back to the model
	maxDirectivePins   = 10
	maxDirectiveTrails = 4 // each trail is a DB query + up to 300 wire points
	maxPinLabelBytes   = 80
	// groundingToleranceM: a directive coordinate must lie within this distance
	// of some coordinate that actually appeared in a tool result.
	groundingToleranceM = 300.0
)

// ─── Events streamed to the HTTP layer ───────────────────────────────────────

// Event is one progress update emitted during an agent run.
type Event struct {
	Type string // "status" | "tool_call" | "narrative_delta" | "directives" | "done" | "error"
	Data map[string]any
}

// MapDirective is a typed map command for the frontend.
type MapDirective struct {
	Type  string  `json:"type"` // fly_to | add_pin | draw_trail
	Lat   float64 `json:"lat,omitempty"`
	Lng   float64 `json:"lng,omitempty"`
	Zoom  float64 `json:"zoom,omitempty"`
	Label string  `json:"label,omitempty"`
	// draw_trail: server-materialized points [[lng,lat],...] — never from the LLM.
	User   string       `json:"user,omitempty"`
	From   string       `json:"from,omitempty"`
	To     string       `json:"to,omitempty"`
	Points [][2]float64 `json:"points,omitempty"`
}

// ToolCallRecord is one audited tool invocation.
type ToolCallRecord struct {
	Tool       string `json:"tool"`
	Args       string `json:"args"`
	DurationMS int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
	ResultSize int    `json:"result_bytes"`
}

// Result is the final outcome of an agent run.
type Result struct {
	Narrative  string           `json:"narrative"`
	Directives []MapDirective   `json:"map_directives"`
	Audit      []ToolCallRecord `json:"audit"`
	Model      string           `json:"model"`
	Steps      int              `json:"steps"`
	Usage      Usage            `json:"usage"`
	DurationMS int64            `json:"duration_ms"`
}

// finalSchema is the strict structured-output contract for the synthesis call.
const finalSchema = `{
	"type": "object",
	"properties": {
		"narrative": {
			"type": "string",
			"description": "The answer for the user: 1-3 short sentences, concrete times and places, no coordinates."
		},
		"map_directives": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"type":  {"type": "string", "enum": ["fly_to", "add_pin", "draw_trail"]},
					"lat":   {"type": "number"},
					"lng":   {"type": "number"},
					"zoom":  {"type": "number"},
					"label": {"type": "string"},
					"user":  {"type": "string"},
					"from":  {"type": "string"},
					"to":    {"type": "string"}
				},
				"required": ["type"],
				"additionalProperties": false
			}
		}
	},
	"required": ["narrative", "map_directives"],
	"additionalProperties": false
}`

// Agent executes one question end-to-end.
type Agent struct {
	Client *Client
	Tools  []Tool
}

// Run drives the agent loop: tool-calling steps, then a streaming structured
// synthesis. emit is called for every progress event (already goroutine-safe
// for our single-writer usage).
func (a *Agent) Run(ctx context.Context, tc *ToolContext, question string, emit func(Event)) (*Result, error) {
	started := time.Now()
	res := &Result{}

	messages := []Message{
		{Role: "system", Content: buildSystemPrompt(tc)},
		{Role: "user", Content: question},
	}
	defs := Defs(a.Tools)
	byName := make(map[string]Tool, len(a.Tools))
	for _, t := range a.Tools {
		byName[t.Name] = t
	}

	// grounding accumulates every coordinate seen in tool results.
	var grounding [][2]float64 // [lat, lng]

	// ── Phase A: tool-calling loop ──────────────────────────────────────────
	for step := 1; step <= maxAgentSteps; step++ {
		res.Steps = step
		emit(Event{Type: "status", Data: map[string]any{"stage": "thinking", "step": step}})

		llmStart := time.Now()
		resp, err := a.Client.Chat(ctx, Request{
			Messages:    messages,
			Tools:       defs,
			ToolChoice:  "auto",
			Temperature: 0.1,
			MaxTokens:   1500,
			Reasoning:   &Reasoning{Effort: "low"},
		})
		if err != nil {
			metricLLMCalls.WithLabelValues("step", "error").Inc()
			return nil, err
		}
		metricLLMCalls.WithLabelValues("step", "ok").Inc()
		metricLLMLatency.Observe(time.Since(llmStart).Seconds())
		res.Usage.PromptTokens += resp.Usage.PromptTokens
		res.Usage.CompletionTokens += resp.Usage.CompletionTokens
		res.Usage.TotalTokens += resp.Usage.TotalTokens
		res.Model = resp.Model

		choice := resp.Choices[0]
		if len(choice.Message.ToolCalls) == 0 {
			// Model is ready to answer. Preserve any content as context and
			// move to synthesis.
			if strings.TrimSpace(choice.Message.Content) != "" {
				messages = append(messages, Message{Role: "assistant", Content: choice.Message.Content})
			}
			break
		}

		// Echo the assistant tool-call message back verbatim, then execute.
		messages = append(messages, choice.Message)
		for _, call := range choice.Message.ToolCalls {
			rec := ToolCallRecord{Tool: call.Function.Name, Args: call.Function.Arguments}
			tool, ok := byName[call.Function.Name]

			var resultJSON []byte
			if !ok {
				rec.Error = "unknown tool"
				resultJSON = []byte(`{"error":"unknown tool"}`)
			} else {
				tStart := time.Now()
				out, execErr := tool.Exec(ctx, tc, json.RawMessage(call.Function.Arguments))
				rec.DurationMS = time.Since(tStart).Milliseconds()
				if execErr != nil {
					// rec.Error is client-facing (SSE + audit) — keep it safe.
					// The model sees the fuller message via resultJSON so it can
					// self-correct bad arguments.
					rec.Error = clientSafeError(execErr.Error())
					resultJSON, _ = json.Marshal(map[string]string{"error": execErr.Error()})
					metricToolCalls.WithLabelValues(tool.Name, "error").Inc()
				} else {
					var mErr error
					resultJSON, mErr = json.Marshal(out)
					if mErr != nil {
						rec.Error = "tool result could not be encoded"
						resultJSON = []byte(`{"error":"internal marshal error"}`)
					}
					metricToolCalls.WithLabelValues(tool.Name, "ok").Inc()
					grounding = append(grounding, extractCoords(out)...)
				}
			}
			rec.ResultSize = len(resultJSON)
			res.Audit = append(res.Audit, rec)

			emit(Event{Type: "tool_call", Data: map[string]any{
				"tool":        rec.Tool,
				"duration_ms": rec.DurationMS,
				"ok":          rec.Error == "",
				"error":       rec.Error,
			}})

			// Never feed the model a truncated (invalid) JSON blob; our tools are
			// already row/point-bounded, so overflow means "ask narrower".
			if len(resultJSON) > maxToolResultBytes {
				resultJSON = []byte(`{"error":"result too large — ask a narrower question or a smaller time range","truncated":true}`)
			}
			messages = append(messages, Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Name:       call.Function.Name,
				Content:    string(resultJSON),
			})
		}
	}

	// ── Phase B: streaming structured synthesis ─────────────────────────────
	emit(Event{Type: "status", Data: map[string]any{"stage": "answering"}})

	rf := &ResponseFormat{Type: "json_schema"}
	rf.JSONSchema.Name = "map_answer"
	rf.JSONSchema.Strict = true
	rf.JSONSchema.Schema = json.RawMessage(finalSchema)
	// require_parameters routes only to providers honoring response_format —
	// without it, some free-tier providers silently return prose.
	prov := &Provider{RequireParameters: true}

	// NOTE: role "user", not "system" — several free-tier providers (notably
	// gpt-oss serving stacks) silently drop system messages that appear after
	// tool results, which made the model answer in prose.
	messages = append(messages, Message{
		Role: "user",
		Content: "Now respond with ONLY a JSON object (no prose, no markdown fences) of this exact shape:\n" +
			`{"narrative": "...", "map_directives": [{"type": "fly_to|add_pin|draw_trail", ...}]}` + "\n" +
			"narrative: 1-3 short sentences answering the question with concrete times/places — NEVER raw coordinates in the text. " +
			"map_directives: one fly_to {type,lat,lng,zoom} for the most relevant location; one add_pin {type,lat,lng,label} per key spot (label like 'Zara — stopped 14 min, 6:42-6:56pm'); " +
			"one draw_trail {type,user,from,to} (ISO 8601 times) when the question involves a route or movement. " +
			"Coordinates MUST be copied exactly from tool results. If you found no data, say so plainly and use an empty map_directives array.",
	})

	extractor := newNarrativeExtractor(func(delta string) {
		emit(Event{Type: "narrative_delta", Data: map[string]any{"text": delta}})
	})
	var rawJSON strings.Builder
	firstToken := time.Time{}
	var finishReason string

	llmStart := time.Now()
	err := a.Client.ChatStream(ctx, Request{
		Messages:       messages,
		ResponseFormat: rf,
		Provider:       prov,
		Temperature:    0.1,
		MaxTokens:      2000, // reasoning models spend hidden tokens before content
		Reasoning:      &Reasoning{Effort: "low"},
	}, func(ev StreamEvent) error {
		if ev.Usage != nil {
			res.Usage.PromptTokens += ev.Usage.PromptTokens
			res.Usage.CompletionTokens += ev.Usage.CompletionTokens
			res.Usage.TotalTokens += ev.Usage.TotalTokens
		}
		if ev.Model != "" {
			res.Model = ev.Model
		}
		if ev.FinishReason != "" {
			finishReason = ev.FinishReason
		}
		if ev.ContentDelta != "" {
			if firstToken.IsZero() {
				firstToken = time.Now()
				metricTTFT.Observe(time.Since(llmStart).Seconds())
			}
			rawJSON.WriteString(ev.ContentDelta)
			extractor.Feed(ev.ContentDelta)
		}
		return nil
	})
	if err != nil {
		metricLLMCalls.WithLabelValues("synthesis", "error").Inc()
		return nil, err
	}
	metricLLMCalls.WithLabelValues("synthesis", "ok").Inc()
	// A truncated answer (hit max_tokens) yields unparseable JSON; fail loudly
	// rather than surfacing half an object as the narrative.
	if finishReason == "length" {
		return nil, fmt.Errorf("ai: answer exceeded the token budget (finish_reason=length)")
	}

	// Parse the full structured output.
	var parsed struct {
		Narrative     string         `json:"narrative"`
		MapDirectives []MapDirective `json:"map_directives"`
	}
	jsonText := extractJSONObject(rawJSON.String())
	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		// Graceful degradation: treat the raw text as narrative — but only if it
		// reads as prose. Raw JSON leaking to the user is worse than a generic
		// apology, so suppress it when the stream still looks like an object.
		slog.Warn("ai: structured output parse failed; falling back to plain narrative", "error", err)
		fallback := strings.TrimSpace(rawJSON.String())
		if strings.HasPrefix(fallback, "{") || strings.HasPrefix(fallback, "[") {
			fallback = "I found the data but couldn't compose a clean answer — please ask again."
		}
		parsed.Narrative = fallback
		parsed.MapDirectives = nil
		if extractor.emitted == 0 && parsed.Narrative != "" {
			emit(Event{Type: "narrative_delta", Data: map[string]any{"text": parsed.Narrative}})
		}
	} else if extractor.emitted == 0 && parsed.Narrative != "" {
		// Extractor missed (e.g. key order surprises); emit the whole narrative once.
		emit(Event{Type: "narrative_delta", Data: map[string]any{"text": parsed.Narrative}})
	}

	res.Narrative = parsed.Narrative
	res.Directives = a.validateDirectives(ctx, tc, parsed.MapDirectives, grounding)
	res.DurationMS = time.Since(started).Milliseconds()

	emit(Event{Type: "directives", Data: map[string]any{"map_directives": res.Directives}})
	return res, nil
}

// ─── Grounding validation ────────────────────────────────────────────────────

// validateDirectives enforces that every coordinate the model emitted is
// traceable to a tool result, materializes draw_trail from the DB, and caps
// volume. Ungrounded directives are dropped, not fixed.
func (a *Agent) validateDirectives(ctx context.Context, tc *ToolContext, in []MapDirective, grounding [][2]float64) []MapDirective {
	grounded := func(lat, lng float64, tolM float64) bool {
		for _, g := range grounding {
			if shared.HaversineM(lat, lng, g[0], g[1]) <= tolM {
				return true
			}
		}
		return false
	}

	var out []MapDirective
	pins, trails := 0, 0
	for _, d := range in {
		switch d.Type {
		case "fly_to":
			if !grounded(d.Lat, d.Lng, 1000) { // relaxed: fly target may be a region center
				slog.Warn("ai: dropped ungrounded fly_to", "lat", d.Lat, "lng", d.Lng)
				metricDirectivesDropped.Inc()
				continue
			}
			if d.Zoom == 0 {
				d.Zoom = 15
			}
			d.Zoom = math.Max(3, math.Min(18, d.Zoom))
			out = append(out, d)
		case "add_pin":
			if pins >= maxDirectivePins {
				continue
			}
			if !grounded(d.Lat, d.Lng, groundingToleranceM) {
				slog.Warn("ai: dropped ungrounded pin", "lat", d.Lat, "lng", d.Lng, "label", d.Label)
				metricDirectivesDropped.Inc()
				continue
			}
			d.Label = truncateRunes(d.Label, maxPinLabelBytes)
			pins++
			out = append(out, d)
		case "draw_trail":
			if trails >= maxDirectiveTrails {
				continue // don't issue the DB query for excess trails
			}
			// The LLM supplies only (user, from, to); points come from the DB.
			trail, err := a.materializeTrail(ctx, tc, d)
			if err != nil {
				slog.Warn("ai: draw_trail materialization failed", "error", err)
				metricDirectivesDropped.Inc()
				continue
			}
			if len(trail.Points) >= 2 {
				trails++
				out = append(out, trail)
			}
		}
	}
	return out
}

// truncateRunes caps s at maxBytes without splitting a UTF-8 rune.
func truncateRunes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// materializeTrail loads real trail points for a draw_trail directive.
func (a *Agent) materializeTrail(ctx context.Context, tc *ToolContext, d MapDirective) (MapDirective, error) {
	if tc.DB == nil {
		return d, fmt.Errorf("draw_trail requires a database (none configured)")
	}
	uid, name, err := tc.resolveUser(d.User)
	if err != nil {
		return d, err
	}
	from, err := tc.parseWhen(d.From)
	if err != nil {
		return d, err
	}
	to, err := tc.parseWhen(d.To)
	if err != nil {
		return d, err
	}
	qctx, cancel := context.WithTimeout(ctx, toolQueryTimeout)
	defer cancel()
	rows, err := tc.DB.QueryContext(qctx, `
		SELECT lat, lng FROM position_history
		WHERE user_id = $1 AND ts >= $2 AND ts <= $3
		ORDER BY ts ASC
		LIMIT 2000`, uid, from.UnixMilli(), to.UnixMilli())
	if err != nil {
		return d, err
	}
	defer rows.Close()

	coarse := tc.coarsened(uid)
	var pts [][2]float64
	for rows.Next() {
		var lat, lng float64
		if err := rows.Scan(&lat, &lng); err != nil {
			return d, fmt.Errorf("scan trail point: %w", err)
		}
		if coarse {
			lat, lng = tc.coord(uid, lat, lng)
		}
		pts = append(pts, [2]float64{lng, lat}) // GeoJSON order
	}
	if err := rows.Err(); err != nil {
		return d, fmt.Errorf("iterate trail: %w", err)
	}
	// Downsample to ≤300 points for the wire.
	if len(pts) > 300 {
		stride := float64(len(pts)-1) / 299.0
		ds := make([][2]float64, 0, 300)
		for i := 0; i < 300; i++ {
			ds = append(ds, pts[int(float64(i)*stride+0.5)])
		}
		ds[len(ds)-1] = pts[len(pts)-1]
		pts = ds
	}
	d.User = name
	d.Points = pts
	return d, nil
}

// extractCoords walks a decoded tool result and collects plausible lat/lng
// pairs (fields named lat/lng or latitude/longitude at the same level).
func extractCoords(v any) [][2]float64 {
	var out [][2]float64
	var walk func(any)
	walk = func(node any) {
		switch n := node.(type) {
		case map[string]any:
			lat, latOK := floatField(n, "lat", "latitude")
			lng, lngOK := floatField(n, "lng", "longitude")
			if latOK && lngOK {
				out = append(out, [2]float64{lat, lng})
			}
			for _, child := range n {
				walk(child)
			}
		case []any:
			for _, child := range n {
				walk(child)
			}
		}
	}
	// Tool results are Go values (maps/slices/structs). Round-trip through
	// JSON so struct fields become map keys walkable above.
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	walk(decoded)
	return out
}

func floatField(m map[string]any, names ...string) (float64, bool) {
	for _, name := range names {
		if v, ok := m[name]; ok {
			if f, ok := v.(float64); ok {
				return f, true
			}
		}
	}
	return 0, false
}

// ─── Streaming narrative extraction ──────────────────────────────────────────
//
// The synthesis call streams a JSON object. To give the user real streaming
// UX, we extract the value of the "narrative" key from the raw token stream
// as it arrives, handling string escapes across chunk boundaries.

type narrativeExtractor struct {
	onDelta func(string)
	buf     strings.Builder // accumulated raw stream until narrative found
	state   int             // 0=searching key, 1=inside narrative string, 2=done
	pending string          // partial escape sequence carried across chunks
	emitted int
}

func newNarrativeExtractor(onDelta func(string)) *narrativeExtractor {
	return &narrativeExtractor{onDelta: onDelta}
}

func (e *narrativeExtractor) Feed(chunk string) {
	switch e.state {
	case 0:
		e.buf.WriteString(chunk)
		s := e.buf.String()
		idx := strings.Index(s, `"narrative"`)
		if idx < 0 {
			return
		}
		rest := s[idx+len(`"narrative"`):]
		colon := strings.IndexByte(rest, ':')
		if colon < 0 {
			return
		}
		afterColon := rest[colon+1:]
		quote := strings.IndexByte(afterColon, '"')
		if quote < 0 {
			return
		}
		e.state = 1
		e.buf.Reset()
		e.consume(afterColon[quote+1:])
	case 1:
		e.consume(chunk)
	}
}

// consume processes string-content bytes, emitting unescaped text until the
// closing quote.
func (e *narrativeExtractor) consume(s string) {
	s = e.pending + s
	e.pending = ""
	var out strings.Builder
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '"' {
			e.state = 2
			break
		}
		if c != '\\' {
			out.WriteByte(c)
			i++
			continue
		}
		// Escape sequence; may be split across chunks.
		if i+1 >= len(s) {
			e.pending = s[i:]
			break
		}
		switch s[i+1] {
		case 'n':
			out.WriteByte('\n')
			i += 2
		case 't':
			out.WriteByte('\t')
			i += 2
		case '"', '\\', '/':
			out.WriteByte(s[i+1])
			i += 2
		case 'r', 'b', 'f':
			i += 2 // drop
		case 'u':
			if i+6 > len(s) {
				e.pending = s[i:]
				i = len(s)
				break
			}
			if cp, err := strconv.ParseUint(s[i+2:i+6], 16, 32); err == nil {
				r := rune(cp)
				// High surrogate: needs the following \uXXXX low surrogate. If
				// the pair is split across chunk boundaries, stash and wait for
				// the next chunk rather than emitting a lone U+FFFD.
				if utf16.IsSurrogate(r) {
					if i+12 > len(s) {
						e.pending = s[i:]
						i = len(s)
						break
					}
					if s[i+6] == '\\' && s[i+7] == 'u' {
						if cp2, err2 := strconv.ParseUint(s[i+8:i+12], 16, 32); err2 == nil {
							out.WriteRune(utf16.DecodeRune(r, rune(cp2)))
							i += 12
							continue
						}
					}
				}
				out.WriteRune(r)
			}
			i += 6
		default:
			i += 2 // unknown escape; drop
		}
	}
	if out.Len() > 0 {
		e.emitted += out.Len()
		e.onDelta(out.String())
	}
}

// extractJSONObject returns the first balanced top-level {...} block in s
// (models sometimes wrap JSON in fences or prose despite response_format),
// or s trimmed when no complete object is found.
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return strings.TrimSpace(s)
	}
	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if esc {
			esc = false
			continue
		}
		switch {
		case c == '\\' && inStr:
			esc = true
		case c == '"':
			inStr = !inStr
		case !inStr && c == '{':
			depth++
		case !inStr && c == '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return strings.TrimSpace(s)
}

// clientSafeError maps a tool error to text safe to show the user. Validation
// and authorization messages (which guide the user) pass through; anything else
// (DB/driver/query internals) collapses to a generic message so table names and
// SQL state never reach the browser.
func clientSafeError(msg string) string {
	safePrefixes := []string{
		"bad arguments", "no visible family member", "cannot parse time",
		"user reference is empty", "unknown tool", "draw_trail requires",
	}
	for _, p := range safePrefixes {
		if strings.HasPrefix(msg, p) {
			return msg
		}
	}
	if strings.Contains(msg, "ambiguous") {
		return msg
	}
	return "the tool could not fetch data"
}

// FormatToolSummary renders a compact human-readable audit line ("queried 3 tools").
func FormatToolSummary(audit []ToolCallRecord) string {
	if len(audit) == 0 {
		return "no tools used"
	}
	names := make([]string, 0, len(audit))
	for _, r := range audit {
		names = append(names, r.Tool)
	}
	return fmt.Sprintf("%d tool call(s): %s", len(audit), strings.Join(names, ", "))
}
