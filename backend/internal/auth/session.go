package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"time"
)

const sessionCacheTTL = 5 * time.Minute

type sessionCacheEntry struct {
	data      *SessionData
	expiresAt time.Time
}

// SessionData holds session payload.
type SessionData struct {
	User      *SessionUser `json:"user"`
	CsrfToken string      `json:"csrfToken"`
}

// SessionUser holds user info in session.
type SessionUser struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

// sessionRow matches the connect-pg-simple sess JSON shape.
type sessionRow struct {
	Sess json.RawMessage `json:"-"` // stored as JSON in DB
}

// sessPayload matches the JSON structure in the session table.
type sessPayload struct {
	User      *SessionUser `json:"user"`
	CsrfToken string      `json:"csrfToken"`
}

// redisClient is the minimal Redis interface needed for session storage.
// *cache.RedisCache satisfies this interface.
type redisClient interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Del(ctx context.Context, key string) error
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

// SessionStore reads/writes the session table with an in-memory cache to avoid DB hits per request.
type SessionStore struct {
	db    *sql.DB
	cache sync.Map // map[string]*sessionCacheEntry
	redis redisClient
}

// NewSessionStore creates a SessionStore.
func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

// SetRedis wires an optional Redis backend for session storage.
// When set, new sessions are written to Redis (not PostgreSQL), eliminating
// session INSERT queries from the DB. Existing DB sessions remain accessible
// as a read fallback during migration.
func (s *SessionStore) SetRedis(rc redisClient) {
	s.redis = rc
}

// Get loads session by sid. Checks in-memory cache → Redis → PostgreSQL.
func (s *SessionStore) Get(sid string) (*SessionData, error) {
	// 1. In-memory cache (same-process hot path)
	if v, ok := s.cache.Load(sid); ok {
		entry := v.(*sessionCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.data, nil
		}
		s.cache.Delete(sid)
	}

	// 2. Redis (avoids a DB round-trip; sessions created after Redis was wired live here)
	if s.redis != nil {
		raw, err := s.redis.Get(context.Background(), "session:"+sid)
		if err == nil && raw != nil {
			var data SessionData
			if jsonErr := json.Unmarshal(raw, &data); jsonErr == nil {
				s.cache.Store(sid, &sessionCacheEntry{data: &data, expiresAt: time.Now().Add(sessionCacheTTL)})
				return &data, nil
			}
		}
		// Redis miss — fall through to DB for sessions predating Redis wiring
	}

	// 3. PostgreSQL fallback (legacy sessions created before Redis was wired)
	var raw []byte
	var expire time.Time
	err := s.db.QueryRowContext(context.Background(),
		`SELECT sess, expire FROM session WHERE sid = $1`, sid).Scan(&raw, &expire)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if time.Now().After(expire) {
		_, _ = s.db.ExecContext(context.Background(), `DELETE FROM session WHERE sid = $1`, sid)
		return nil, nil
	}
	var payload sessPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	data := &SessionData{User: payload.User, CsrfToken: payload.CsrfToken}
	s.cache.Store(sid, &sessionCacheEntry{data: data, expiresAt: time.Now().Add(sessionCacheTTL)})
	return data, nil
}

// Create stores a session. When Redis is available, writes only to Redis (not PostgreSQL),
// eliminating session INSERT queries from the DB load.
func (s *SessionStore) Create(sid string, data *SessionData, maxAge time.Duration) error {
	// Always populate in-memory cache first for same-process CSRF consistency.
	s.cache.Store(sid, &sessionCacheEntry{data: data, expiresAt: time.Now().Add(sessionCacheTTL)})

	if s.redis != nil {
		return s.redis.Set(context.Background(), "session:"+sid, data, maxAge)
	}

	// Fallback: PostgreSQL
	expire := time.Now().Add(maxAge)
	payload := sessPayload{User: data.User, CsrfToken: data.CsrfToken}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(context.Background(),
		`INSERT INTO session (sid, sess, expire) VALUES ($1, $2, $3)
		 ON CONFLICT (sid) DO UPDATE SET sess = $2, expire = $3`,
		sid, raw, expire)
	return err
}

// Destroy deletes the session by sid from cache, Redis (if wired), and DB.
func (s *SessionStore) Destroy(sid string) error {
	s.cache.Delete(sid)
	if s.redis != nil {
		_ = s.redis.Del(context.Background(), "session:"+sid)
	}
	// Always attempt DB delete to clean up any legacy session that predates Redis wiring.
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM session WHERE sid = $1`, sid)
	return err
}

// Touch extends the TTL of an existing session.
func (s *SessionStore) Touch(sid string, maxAge time.Duration) error {
	if v, ok := s.cache.Load(sid); ok {
		entry := v.(*sessionCacheEntry)
		s.cache.Store(sid, &sessionCacheEntry{data: entry.data, expiresAt: time.Now().Add(sessionCacheTTL)})
	}
	if s.redis != nil {
		return s.redis.Expire(context.Background(), "session:"+sid, maxAge)
	}
	expire := time.Now().Add(maxAge)
	_, err := s.db.ExecContext(context.Background(), `UPDATE session SET expire = $1 WHERE sid = $2`, expire, sid)
	return err
}

// ParseSessionID extracts session ID from connect.sid cookie value.
// Format: s:SESSION_ID.HMAC_SIGNATURE (URL-decoded). Returns the part between s: and . (exclusive).
func ParseSessionID(cookie string) string {
	if cookie == "" {
		return ""
	}
	decoded, err := url.QueryUnescape(cookie)
	if err != nil {
		decoded = cookie
	}
	if !strings.HasPrefix(decoded, "s:") {
		return ""
	}
	rest := decoded[2:]
	dot := strings.Index(rest, ".")
	if dot < 0 {
		return rest
	}
	return rest[:dot]
}

// GenerateSessionID returns 32 random bytes as hex.
func GenerateSessionID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateCsrfToken returns 16 random bytes as hex.
func GenerateCsrfToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SignCookie creates the full cookie value "s:sessionId.base64(hmac-sha256(sessionId, secret))".
func SignCookie(sessionID, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionID))
	sig := mac.Sum(nil)
	return "s:" + sessionID + "." + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(sig)
}
