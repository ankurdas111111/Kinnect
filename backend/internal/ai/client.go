// Package ai implements the "Ask the Map" agentic copilot: a hand-rolled
// agent loop over OpenRouter's OpenAI-compatible API, with family-scoped
// read-only tools, grounding validation, and SSE streaming to the client.
//
// Design rules (see docs/ai-copilot.md):
//   - The LLM never writes SQL and never invents coordinates: all geometry
//     comes from parameterized queries; directive coordinates are validated
//     against tool outputs before reaching the client (grounding).
//   - Authorization lives BELOW the model: tools resolve the requester's
//     visible set from the session, never from model arguments.
//   - Zero new Go dependencies: net/http + encoding/json only.
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	openRouterURL = "https://openrouter.ai/api/v1/chat/completions"
	// Attribution headers recommended by OpenRouter; harmless if the app is private.
	refererHeader = "https://github.com/ankurdas111111/Kinnect"
	titleHeader   = "Kinnect Ask-the-Map"
)

// Client is a minimal OpenRouter chat-completions client.
type Client struct {
	apiKey string
	// models is a preference-ordered list; OpenRouter falls back server-side
	// via the "models" request field when the primary is unavailable.
	models []string
	http   *http.Client
}

// NewClient creates a Client. models must be non-empty.
//
// The http.Client has NO overall Timeout on purpose: that would also bound the
// streaming response-body read and could sever a slow synthesis mid-answer.
// Total duration is instead governed by the per-request context (the handler's
// 90s budget); Transport-level timeouts still fail fast on connect/header stalls.
func NewClient(apiKey string, models []string) *Client {
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 45 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
	return &Client{
		apiKey: apiKey,
		models: models,
		http:   &http.Client{Transport: transport},
	}
}

// Models returns the configured model list (primary first).
func (c *Client) Models() []string { return c.models }

// ─── Wire types (OpenAI chat-completions format) ─────────────────────────────

// Message is one chat message. Content is a plain string (no multimodal parts).
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall is a model-requested function invocation.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ToolDef declares a callable function to the model.
type ToolDef struct {
	Type     string `json:"type"` // always "function"
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

// ResponseFormat requests strict structured output.
type ResponseFormat struct {
	Type       string `json:"type"` // "json_schema"
	JSONSchema struct {
		Name   string          `json:"name"`
		Strict bool            `json:"strict"`
		Schema json.RawMessage `json:"schema"`
	} `json:"json_schema"`
}

// Provider controls OpenRouter provider routing.
type Provider struct {
	// RequireParameters routes only to providers that support every request
	// parameter (critical for response_format on free models).
	RequireParameters bool `json:"require_parameters,omitempty"`
}

// Reasoning bounds a reasoning model's hidden thinking budget. Without it,
// models like gpt-oss can spend the whole max_tokens budget on reasoning and
// emit zero content.
type Reasoning struct {
	Effort string `json:"effort,omitempty"` // "low" | "medium" | "high"
}

// Request is a chat-completions request body.
type Request struct {
	Model          string          `json:"model,omitempty"`
	Models         []string        `json:"models,omitempty"` // fallback array (OpenRouter)
	Messages       []Message       `json:"messages"`
	Tools          []ToolDef       `json:"tools,omitempty"`
	ToolChoice     string          `json:"tool_choice,omitempty"` // "auto" | "none"
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	Provider       *Provider       `json:"provider,omitempty"`
	Reasoning      *Reasoning      `json:"reasoning,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	StreamOptions  *struct {
		IncludeUsage bool `json:"include_usage"`
	} `json:"stream_options,omitempty"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
}

// Usage reports token consumption for one LLM call.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Response is a non-streaming chat-completions response.
type Response struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

// ─── Non-streaming call (agent-loop steps) ───────────────────────────────────

// Chat performs one non-streaming completion. The request's Models field is
// filled from the client's configured fallback list.
func (c *Client) Chat(ctx context.Context, req Request) (*Response, error) {
	req.Stream = false
	c.applyModels(&req)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ai: marshal request: %w", err)
	}
	httpResp, err := c.post(ctx, body)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("ai: read response: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, apiError(httpResp.StatusCode, raw)
	}
	var resp Response
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("ai: decode response: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("ai: provider error: %s", resp.Error.Message)
	}
	if len(resp.Choices) == 0 {
		return nil, errors.New("ai: empty choices in response")
	}
	return &resp, nil
}

// ─── Streaming call (final synthesis) ────────────────────────────────────────

// StreamEvent is one delta from a streaming completion.
type StreamEvent struct {
	ContentDelta string // token text, may be ""
	Done         bool
	FinishReason string
	Usage        *Usage // non-nil on the final usage chunk
	Model        string
}

// ChatStream performs a streaming completion, invoking onEvent for each delta.
// onEvent errors abort the stream. Handles OpenRouter SSE comment payloads
// (": OPENROUTER PROCESSING") and mid-stream error events.
func (c *Client) ChatStream(ctx context.Context, req Request, onEvent func(StreamEvent) error) error {
	req.Stream = true
	req.StreamOptions = &struct {
		IncludeUsage bool `json:"include_usage"`
	}{IncludeUsage: true}
	c.applyModels(&req)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("ai: marshal request: %w", err)
	}
	httpResp, err := c.post(ctx, body)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(httpResp.Body, 64<<10))
		return apiError(httpResp.StatusCode, raw)
	}

	scanner := bufio.NewScanner(httpResp.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		// SSE comments (keep-alive) start with ':'; blank lines separate events.
		if len(line) == 0 || line[0] == ':' {
			continue
		}
		payload, ok := bytes.CutPrefix(line, []byte("data: "))
		if !ok {
			continue
		}
		if bytes.Equal(payload, []byte("[DONE]")) {
			return onEvent(StreamEvent{Done: true})
		}

		var chunk struct {
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *Usage `json:"usage"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(payload, &chunk); err != nil {
			continue // tolerate malformed keep-alives
		}
		if chunk.Error != nil {
			return fmt.Errorf("ai: mid-stream provider error: %s", chunk.Error.Message)
		}

		ev := StreamEvent{Model: chunk.Model, Usage: chunk.Usage}
		if len(chunk.Choices) > 0 {
			ev.ContentDelta = chunk.Choices[0].Delta.Content
			ev.FinishReason = chunk.Choices[0].FinishReason
			if ev.FinishReason == "error" {
				return errors.New("ai: stream finished with provider error")
			}
		}
		if err := onEvent(ev); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ai: stream read: %w", err)
	}
	return nil
}

// ─── Internals ───────────────────────────────────────────────────────────────

func (c *Client) applyModels(req *Request) {
	if len(c.models) == 1 {
		req.Model = c.models[0]
		return
	}
	// "models" (plural) enables OpenRouter's server-side fallback routing.
	req.Models = c.models
}

func (c *Client) post(ctx context.Context, body []byte) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ai: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("HTTP-Referer", refererHeader)
	httpReq.Header.Set("X-Title", titleHeader)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ai: http: %w", err)
	}
	return resp, nil
}

func apiError(status int, raw []byte) error {
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	msg := strings.TrimSpace(string(raw))
	if json.Unmarshal(raw, &e) == nil && e.Error.Message != "" {
		msg = e.Error.Message
	}
	if len(msg) > 300 {
		msg = msg[:300]
	}
	if status == http.StatusTooManyRequests {
		return fmt.Errorf("ai: rate limited by OpenRouter (HTTP 429): %s", msg)
	}
	return fmt.Errorf("ai: OpenRouter HTTP %d: %s", status, msg)
}
