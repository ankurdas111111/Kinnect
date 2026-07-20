package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	DatabaseURL         string
	SessionSecret       string
	Port                string
	NodeEnv             string
	AdminEmail          string
	LogLevel            string
	CORSAllowedOrigins  []string
	RedisURL            string // Optional: Redis connection URL from Aiven
	// Web Push (VAPID) — optional; push features disabled if empty.
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string // mailto: or https: URL

	// Twilio SMS — optional; panic relay SMS disabled if empty.
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string // E.164 format, e.g. +15551234567

	// Ola Maps — optional; enables Ola place search + routing. Falls back to
	// Photon / OSRM when empty.
	OlaMapsAPIKey string

	// OpenRouter — optional; enables the Ask-the-Map AI copilot when set.
	OpenRouterAPIKey string
	// AIModels is the preference-ordered model list (primary first). Set via
	// AI_MODELS as a comma-separated list; sensible free-tier defaults otherwise.
	AIModels []string
}

const (
	defaultPort    = "3000"
	defaultNodeEnv = "development"
	defaultLogLevel = "info"
	minSessionSecretLen = 32
)

var defaultCORSOrigins = []string{
	"http://localhost",
	"https://localhost",
	"http://localhost:5173",
	"https://localhost:5173",
	"capacitor://localhost",
	"ionic://localhost",
}

// Load reads configuration from environment variables and returns a Config.
// Returns an error if required vars are missing or invalid.
func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("SESSION_SECRET is required")
	}
	// Pad if shorter than 32 chars
	if len(secret) < minSessionSecretLen {
		pad := strings.Repeat("0", minSessionSecretLen-len(secret))
		secret = secret + pad
		slog.Warn("SESSION_SECRET was shorter than 32 chars; padded", "original_len", len(secret)-len(pad))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	nodeEnv := os.Getenv("NODE_ENV")
	if nodeEnv == "" {
		nodeEnv = os.Getenv("GO_ENV")
	}
	if nodeEnv == "" {
		nodeEnv = defaultNodeEnv
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = defaultLogLevel
	}

	// Always start with the default origins (includes capacitor://localhost for
	// native mobile apps). If CORS_ALLOWED_ORIGINS is set (e.g. on Render), merge
	// those in rather than replacing — so the Capacitor app never gets locked out.
	corsOrigins := defaultCORSOrigins
	if extra := parseCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS")); len(extra) > 0 {
		seen := make(map[string]bool, len(corsOrigins))
		for _, o := range corsOrigins {
			seen[o] = true
		}
		for _, o := range extra {
			if !seen[o] {
				corsOrigins = append(corsOrigins, o)
			}
		}
	}

	return &Config{
		DatabaseURL:        dbURL,
		SessionSecret:      secret,
		Port:               port,
		NodeEnv:            nodeEnv,
		AdminEmail:         os.Getenv("ADMIN_EMAIL"),
		LogLevel:           logLevel,
		CORSAllowedOrigins: corsOrigins,
		RedisURL:           os.Getenv("REDIS_URL"), // Optional
		VAPIDPublicKey:     os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:    os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDSubject:       os.Getenv("VAPID_SUBJECT"),
		TwilioAccountSID:   os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:    os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioFromNumber:   os.Getenv("TWILIO_FROM_NUMBER"),
		OlaMapsAPIKey:      os.Getenv("OLA_MAPS_API_KEY"),
		OpenRouterAPIKey:   os.Getenv("OPENROUTER_API_KEY"),
		AIModels:           ParseAIModels(os.Getenv("AI_MODELS")),
	}, nil
}

// DefaultAIModels returns the free tool-calling models used when AI_MODELS is
// unset (preference order; OpenRouter falls back server-side).
func DefaultAIModels() []string {
	return []string{
		"openai/gpt-oss-20b:free",
		"nvidia/nemotron-3-super-120b-a12b:free",
	}
}

// ParseAIModels parses AI_MODELS (comma-separated, primary first), trimming and
// deduping, and falling back to DefaultAIModels when empty. Exported so the
// verification CLIs (cmd/ai-smoke, cmd/ai-eval) parse identically to the server.
func ParseAIModels(raw string) []string {
	models := parseCORSOrigins(raw) // same trim/dedupe semantics
	if len(models) == 0 {
		return DefaultAIModels()
	}
	return models
}

func parseCORSOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]bool)
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
