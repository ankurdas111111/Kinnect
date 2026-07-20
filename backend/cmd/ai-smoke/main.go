// ai-smoke exercises the Ask-the-Map agent loop against the live OpenRouter
// API using canned in-memory tools — no database required. Use it to verify
// model behavior (tool calling, structured outputs, streaming) before a demo:
//
//	OPENROUTER_API_KEY=sk-... go run ./cmd/ai-smoke "Where is everyone right now?"
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"kinnect-v3/internal/ai"
	"kinnect-v3/internal/config"
)

func main() {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		fmt.Fprintln(os.Stderr, "OPENROUTER_API_KEY is not set")
		os.Exit(1)
	}
	question := "Where is everyone right now?"
	if len(os.Args) > 1 {
		question = strings.Join(os.Args[1:], " ")
	}

	models := config.ParseAIModels(os.Getenv("AI_MODELS"))
	fmt.Printf("models: %v\nquestion: %q\n\n", models, question)

	// Canned tools: a fake family around Bengaluru with one member moving.
	now := time.Now()
	tools := []ai.Tool{
		{
			Name:        "get_current_positions",
			Description: "Latest known position of every visible family member.",
			Params:      `{"type":"object","properties":{},"additionalProperties":false}`,
			Exec: func(_ context.Context, _ *ai.ToolContext, _ json.RawMessage) (any, error) {
				return map[string]any{"positions": []map[string]any{
					{"user": "Dad", "user_id": "u-dad", "lat": 12.9716, "lng": 77.5946, "speed_ms": 8.4, "seen": now.Format("Mon 2006-01-02 15:04") + " (just now)"},
					{"user": "Zara", "user_id": "u-zara", "lat": 12.9352, "lng": 77.6245, "speed_ms": 0.0, "seen": now.Add(-4*time.Minute).Format("Mon 2006-01-02 15:04") + " (4m ago)"},
				}}, nil
			},
		},
		{
			Name:        "get_saved_places",
			Description: "The asking user's saved places.",
			Params:      `{"type":"object","properties":{},"additionalProperties":false}`,
			Exec: func(_ context.Context, _ *ai.ToolContext, _ json.RawMessage) (any, error) {
				return map[string]any{"places": []map[string]any{
					{"name": "Home", "lat": 12.9352, "lng": 77.6245, "radius_m": 150},
					{"name": "School", "lat": 12.9863, "lng": 77.6034, "radius_m": 200},
				}}, nil
			},
		},
	}

	agent := &ai.Agent{Client: ai.NewClient(key, models), Tools: tools}
	tc := &ai.ToolContext{
		RequesterID:   "u-me",
		RequesterName: "Ankur",
		Visible:       map[string]string{"u-me": "Ankur", "u-dad": "Dad", "u-zara": "Zara"},
		Loc:           time.Local,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	started := time.Now()
	result, err := agent.Run(ctx, tc, question, func(ev ai.Event) {
		switch ev.Type {
		case "status":
			fmt.Printf("[status] %v\n", ev.Data)
		case "tool_call":
			fmt.Printf("[tool]   %s ok=%v (%vms)\n", ev.Data["tool"], ev.Data["ok"], ev.Data["duration_ms"])
		case "narrative_delta":
			fmt.Print(ev.Data["text"])
		case "directives":
			fmt.Printf("\n[directives] ")
			b, _ := json.MarshalIndent(ev.Data["map_directives"], "", "  ")
			fmt.Println(string(b))
		}
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nFAILED after %s: %v\n", time.Since(started).Round(time.Millisecond), err)
		os.Exit(1)
	}

	fmt.Printf("\n\nOK in %s — model=%s steps=%d tokens=%d (prompt %d / completion %d)\n",
		time.Since(started).Round(time.Millisecond), result.Model, result.Steps,
		result.Usage.TotalTokens, result.Usage.PromptTokens, result.Usage.CompletionTokens)
	fmt.Printf("audit: %s\n", ai.FormatToolSummary(result.Audit))
}
