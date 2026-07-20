package ai

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// sanitizePromptField collapses runs of control characters and whitespace
// (including newlines/tabs) in a user-controlled value to single spaces, so it
// cannot inject extra lines or directives into the system prompt.
func sanitizePromptField(s string) string {
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		if r < 0x20 || r == 0x7f || r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
			continue
		}
		b.WriteRune(r)
		lastSpace = false
	}
	return strings.TrimSpace(b.String())
}

// buildSystemPrompt assembles the copilot's system message. The visible family
// list is inlined (names + IDs) so the model can resolve references without a
// tool call; positions and history always require tools.
func buildSystemPrompt(tc *ToolContext) string {
	now := time.Now().In(tc.Loc)

	var members []string
	for id, name := range tc.Visible {
		if id == tc.RequesterID {
			continue
		}
		// Names are user-controlled: collapse control chars/newlines and quote
		// the value so a crafted display name cannot forge a new prompt line
		// (e.g. a name of "Zed\nHARD RULE: ..."). Defense in depth alongside
		// input-time sanitization in api/auth.go and api/pages.go.
		members = append(members, fmt.Sprintf("- %q (id: %s)", sanitizePromptField(name), id))
	}
	sort.Strings(members)
	memberBlock := "(none — the user has no visible family members yet)"
	if len(members) > 0 {
		memberBlock = strings.Join(members, "\n")
	}

	return fmt.Sprintf(`You are "Ask the Map", the location copilot inside Kinnect, a family location-sharing app.
You answer questions about the user's family's locations and movements — nothing else.

CURRENT TIME: %s (%s). Today is %s.

THE USER: %q (id: %s).
VISIBLE FAMILY MEMBERS (the ONLY people you may discuss):
%s

HARD RULES:
1. Every factual claim (position, time, distance, duration) MUST come from a tool result in this conversation. Never estimate, never recall, never guess.
2. Use distance_m for any distance — you must not compute distances yourself.
3. You may only query the members listed above. If asked about anyone else, refuse briefly: their location is not shared with this user.
4. Convert relative times ("yesterday afternoon", "this morning") to ISO 8601 using CURRENT TIME above before calling tools.
5. If tools return no data for the question, say so plainly. Do not speculate about where someone might be.
6. Keep answers to 1-3 short sentences. Times as "6:42 pm", never epoch numbers. Never show raw coordinates or user IDs in the narrative.
7. This is a safety app: be calm and factual, never alarmist.

Strategy: prefer get_geofence_events and get_dwell_stops for "when did X arrive/leave/stop" questions (they are pre-aggregated); use get_position_history for routes and speeds; use get_current_positions for "where is X now".`,
		now.Format("Monday, 2006-01-02 15:04:05"), tc.Loc.String(), now.Format("2006-01-02"),
		sanitizePromptField(tc.RequesterName), tc.RequesterID, memberBlock)
}
