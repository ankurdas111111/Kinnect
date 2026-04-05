package db

import (
	"database/sql"
	"fmt"
)

// InitDB creates the schema if it does not exist. Matches backend-v2 schema.
func InitDB(db *sql.DB) error {
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,

		`CREATE TABLE IF NOT EXISTS users (
			id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			first_name    VARCHAR(50) NOT NULL,
			last_name     VARCHAR(50) NOT NULL,
			password_hash TEXT NOT NULL,
			role          VARCHAR(10) NOT NULL DEFAULT 'user',
			share_code    VARCHAR(6) UNIQUE NOT NULL,
			email         VARCHAR(255) DEFAULT NULL,
			mobile        VARCHAR(20) DEFAULT NULL,
			created_at    BIGINT NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_mobile ON users(mobile) WHERE mobile IS NOT NULL`,

		`CREATE TABLE IF NOT EXISTS "session" (
			"sid"    VARCHAR NOT NULL PRIMARY KEY,
			"sess"   JSON NOT NULL,
			"expire" TIMESTAMP(6) NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS "IDX_session_expire" ON "session" ("expire")`,

	// Drop legacy active_sessions table — mirrors in-memory cache with no reader; pure write waste
	`DROP TABLE IF EXISTS active_sessions`,

		`CREATE TABLE IF NOT EXISTS rooms (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			code       VARCHAR(6) UNIQUE NOT NULL,
			name       VARCHAR(50) NOT NULL,
			created_by UUID REFERENCES users(id),
			created_at BIGINT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS room_members (
			id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(room_id, user_id)
		)`,

		`CREATE TABLE IF NOT EXISTS contacts (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			owner_id   UUID REFERENCES users(id) ON DELETE CASCADE,
			contact_id UUID REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(owner_id, contact_id)
		)`,

		`CREATE TABLE IF NOT EXISTS live_tokens (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			token      VARCHAR(64) UNIQUE NOT NULL,
			user_id    UUID REFERENCES users(id) ON DELETE CASCADE,
			expires_at BIGINT,
			created_at BIGINT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS guardianships (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			guardian_id UUID REFERENCES users(id) ON DELETE CASCADE,
			ward_id     UUID REFERENCES users(id) ON DELETE CASCADE,
			status      VARCHAR(10) NOT NULL DEFAULT 'pending',
			expires_at  BIGINT,
			created_at  BIGINT NOT NULL,
			UNIQUE(guardian_id, ward_id)
		)`,

		`CREATE TABLE IF NOT EXISTS position_history (
			id          BIGSERIAL PRIMARY KEY,
			user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			latitude    DOUBLE PRECISION NOT NULL,
			longitude   DOUBLE PRECISION NOT NULL,
			speed       REAL,
			accuracy    REAL,
			recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pos_history_user_time ON position_history (user_id, recorded_at DESC)`,
		// Drop redundant single-column index — the purge query uses a seq scan (runs once/day); saves ~120 MB
		`DROP INDEX IF EXISTS idx_pos_history_recorded_at`,

		`CREATE INDEX IF NOT EXISTS idx_contacts_contact_id ON contacts(contact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_live_tokens_expires ON live_tokens(expires_at) WHERE expires_at IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_guardianships_ward ON guardianships(ward_id)`,

		// Additional columns (ALTER for compatibility with V2)
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_latitude DOUBLE PRECISION`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_longitude DOUBLE PRECISION`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_speed TEXT`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_update BIGINT`,

		`ALTER TABLE room_members ADD COLUMN IF NOT EXISTS role VARCHAR(10) DEFAULT 'member'`,
		`ALTER TABLE room_members ADD COLUMN IF NOT EXISTS role_expires_at BIGINT`,

		`ALTER TABLE guardianships ADD COLUMN IF NOT EXISTS initiated_by VARCHAR(10) DEFAULT 'guardian'`,

		`CREATE TABLE IF NOT EXISTS room_admin_requests (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			room_code   VARCHAR(6) NOT NULL,
			user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
			expires_in  VARCHAR(10),
			created_at  BIGINT NOT NULL,
			UNIQUE(room_code, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS room_admin_votes (
			id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			room_code    VARCHAR(6) NOT NULL,
			requester_id UUID REFERENCES users(id) ON DELETE CASCADE,
			voter_id     UUID REFERENCES users(id) ON DELETE CASCADE,
			vote         VARCHAR(10) NOT NULL,
			UNIQUE(room_code, requester_id, voter_id)
		)`,

		// Saved places (server-backed for arrival intelligence & zone stories)
		`CREATE TABLE IF NOT EXISTS saved_places (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id    UUID REFERENCES users(id) ON DELETE CASCADE,
			name       VARCHAR(100) NOT NULL,
			icon       VARCHAR(20) DEFAULT 'pin',
			latitude   DOUBLE PRECISION NOT NULL,
			longitude  DOUBLE PRECISION NOT NULL,
			radius_m   INTEGER DEFAULT 100,
			created_at BIGINT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_saved_places_user ON saved_places(user_id)`,

		// Zone visits — who visited which saved place and for how long
		`CREATE TABLE IF NOT EXISTS zone_visits (
			id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id        UUID REFERENCES users(id) ON DELETE CASCADE,
			place_id       UUID REFERENCES saved_places(id) ON DELETE CASCADE,
			arrived_at     TIMESTAMPTZ NOT NULL,
			departed_at    TIMESTAMPTZ,
			duration_seconds INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_zone_visits_place_time ON zone_visits(place_id, arrived_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_zone_visits_user ON zone_visits(user_id, arrived_at DESC)`,

		// Quiet Hours — server-enforced location coarsening for non-guardians
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS quiet_hours_enabled BOOLEAN DEFAULT false`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS quiet_hours_start TIME`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS quiet_hours_end TIME`,

		// Heartbeat Check — daily wellness pulse
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS heartbeat_enabled BOOLEAN DEFAULT false`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS heartbeat_deadline TIME`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS heartbeat_last_signal BIGINT`,

		// Panic Relay — external emergency SMS contacts
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS emergency_phone_1 VARCHAR(20)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS emergency_phone_2 VARCHAR(20)`,

		// Ambient Status Messages — user-set short status visible to family
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS status_message VARCHAR(80)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS status_expires_at BIGINT`,

		// Sharing Schedules — granular per-recipient / per-time-window visibility control
		`CREATE TABLE IF NOT EXISTS sharing_schedules (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
			target_type VARCHAR(10) NOT NULL DEFAULT 'all',
			target_id   UUID,
			day_mask    SMALLINT NOT NULL DEFAULT 127,
			start_time  TIME NOT NULL DEFAULT '00:00',
			end_time    TIME NOT NULL DEFAULT '23:59',
			enabled     BOOLEAN NOT NULL DEFAULT true,
			created_at  BIGINT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sharing_schedules_user ON sharing_schedules(user_id)`,

		// movement_events — semantic location event log (replaces position_history long-term)
		`CREATE TABLE IF NOT EXISTS movement_events (
			id           BIGSERIAL PRIMARY KEY,
			user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			event_type   VARCHAR(25) NOT NULL,
			lat          DOUBLE PRECISION,
			lng          DOUBLE PRECISION,
			speed_ms     REAL,
			accuracy_m   REAL,
			place_id     UUID REFERENCES saved_places(id) ON DELETE SET NULL,
			place_name   VARCHAR(100),
			motion_class VARCHAR(10),
			metadata     JSONB DEFAULT '{}',
			recorded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_movement_events_user_time
			ON movement_events (user_id, recorded_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_movement_events_place_time
			ON movement_events (place_id, recorded_at DESC)
			WHERE place_id IS NOT NULL`,

		// SOS watch tokens — persisted so they survive server restarts and can be queried for audit.
		`CREATE TABLE IF NOT EXISTS sos_watch_tokens (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			token      VARCHAR(64) UNIQUE NOT NULL,
			user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			socket_id  VARCHAR(64) NOT NULL,
			expires_at BIGINT NOT NULL,
			created_at BIGINT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sos_watch_tokens_user ON sos_watch_tokens(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sos_watch_tokens_exp ON sos_watch_tokens(expires_at)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("schema init: %w", err)
		}
	}
	return nil
}
