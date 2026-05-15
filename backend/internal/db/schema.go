package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// InitDB creates the schema if it does not exist. Matches backend-v2 schema.
// ctx is used for all DDL statements — a hang (e.g. Aiven DDL lock) surfaces
// as a context-deadline error instead of blocking forever.
// gen_random_uuid() is built-in since PG 13; no pgcrypto/uuid-ossp needed.
func InitDB(ctx context.Context, db *sql.DB) error {
	statements := []string{

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

		`CREATE INDEX IF NOT EXISTS idx_contacts_contact_id ON contacts(contact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_live_tokens_expires ON live_tokens(expires_at) WHERE expires_at IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_guardianships_ward ON guardianships(ward_id)`,
		`CREATE INDEX IF NOT EXISTS idx_room_members_room_id ON room_members(room_id)`,
		`CREATE INDEX IF NOT EXISTS idx_room_members_user_id ON room_members(user_id)`,

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

		// Secret Messages — end-to-end encrypted ephemeral messages between contacts
		`CREATE TABLE IF NOT EXISTS secret_messages (
			id           BIGSERIAL PRIMARY KEY,
			sender_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			receiver_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			ciphertext   TEXT NOT NULL,
			iv           TEXT NOT NULL,
			salt         TEXT NOT NULL,
			seen_at      TIMESTAMPTZ,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_secret_messages_receiver ON secret_messages(receiver_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_secret_messages_sender ON secret_messages(sender_id, created_at DESC)`,
		// Migration guard: add seen_at to existing deployments that predate this column
		`ALTER TABLE secret_messages ADD COLUMN IF NOT EXISTS seen_at TIMESTAMPTZ`,

		// Secret chat invite tokens — persisted so permanent links survive server restarts
		`CREATE TABLE IF NOT EXISTS secret_chat_invites (
			token      TEXT PRIMARY KEY,
			owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			peer_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		// Dedup guard: remove duplicate (owner_id, peer_id) rows before creating
		// the unique index. Keeps the most-recently-created token per pair.
		// Safe to re-run: no-op if there are no duplicates.
		`DELETE FROM secret_chat_invites
			WHERE token NOT IN (
				SELECT DISTINCT ON (owner_id, peer_id) token
				FROM secret_chat_invites
				ORDER BY owner_id, peer_id, created_at DESC
			)`,
		// One stable link per (owner, peer) pair — upsert-safe migration guard
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_secret_chat_invites_pair ON secret_chat_invites(owner_id, peer_id)`,

		// F3: Meeting point columns on rooms
		`ALTER TABLE rooms ADD COLUMN IF NOT EXISTS meeting_lat    DOUBLE PRECISION`,
		`ALTER TABLE rooms ADD COLUMN IF NOT EXISTS meeting_lng    DOUBLE PRECISION`,
		`ALTER TABLE rooms ADD COLUMN IF NOT EXISTS meeting_label  VARCHAR(80)`,
		`ALTER TABLE rooms ADD COLUMN IF NOT EXISTS meeting_set_by UUID REFERENCES users(id) ON DELETE SET NULL`,
		`ALTER TABLE rooms ADD COLUMN IF NOT EXISTS meeting_set_at BIGINT`,

		// F5: Speed alert threshold on users (m/s; NULL = disabled)
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS speed_alert_threshold_ms DOUBLE PRECISION`,

		// F4: Saved places
		`CREATE TABLE IF NOT EXISTS saved_places (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name       VARCHAR(80) NOT NULL,
			icon       VARCHAR(20),
			latitude   DOUBLE PRECISION NOT NULL,
			longitude  DOUBLE PRECISION NOT NULL,
			radius_m   DOUBLE PRECISION NOT NULL DEFAULT 100,
			created_at BIGINT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_saved_places_user ON saved_places(user_id)`,

		// F6: Geofence event log
		`CREATE TABLE IF NOT EXISTS geofence_events (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			fence_name VARCHAR(80) NOT NULL DEFAULT '',
			event_type VARCHAR(10) NOT NULL,
			lat        DOUBLE PRECISION NOT NULL,
			lng        DOUBLE PRECISION NOT NULL,
			ts         BIGINT NOT NULL
		)`,
		// Migration guard: ts column added after initial deployment
		`ALTER TABLE geofence_events ADD COLUMN IF NOT EXISTS ts BIGINT NOT NULL DEFAULT 0`,
		`CREATE INDEX IF NOT EXISTS idx_geofence_events_user_ts ON geofence_events(user_id, ts DESC)`,

		// F7: Proximity alerts
		`CREATE TABLE IF NOT EXISTS proximity_alerts (
			id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			owner_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			target_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			radius_m          INT NOT NULL DEFAULT 500,
			enabled           BOOLEAN NOT NULL DEFAULT true,
			last_triggered_at BIGINT NOT NULL DEFAULT 0,
			created_at        BIGINT NOT NULL,
			UNIQUE(owner_id, target_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_proximity_alerts_owner  ON proximity_alerts(owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_proximity_alerts_target ON proximity_alerts(target_id)`,

		// F8: Room bulletin board
		`CREATE TABLE IF NOT EXISTS room_notes (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			room_id    UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
			author_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			body       VARCHAR(200) NOT NULL,
			created_at BIGINT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_room_notes_room ON room_notes(room_id, created_at DESC)`,

		// F9: Daily activity summary
		`CREATE TABLE IF NOT EXISTS daily_activity (
			user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			date           DATE NOT NULL,
			distance_m     INT NOT NULL DEFAULT 0,
			active_minutes INT NOT NULL DEFAULT 0,
			updated_at     BIGINT NOT NULL,
			PRIMARY KEY (user_id, date)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_daily_activity_user ON daily_activity(user_id, date DESC)`,

		// F10: Position history — throttled trail points for route replay and arrival ETA
		`CREATE TABLE IF NOT EXISTS position_history (
			id      BIGSERIAL PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			lat     DOUBLE PRECISION NOT NULL,
			lng     DOUBLE PRECISION NOT NULL,
			speed   DOUBLE PRECISION,
			ts      BIGINT NOT NULL
		)`,
		// Migration guard: ts column added after initial deployment
		`ALTER TABLE position_history ADD COLUMN IF NOT EXISTS ts BIGINT NOT NULL DEFAULT 0`,
		`CREATE INDEX IF NOT EXISTS idx_position_history_user_ts ON position_history(user_id, ts DESC)`,
	}

	for i, stmt := range statements {
		// Log BEFORE executing — if we hang, the last printed index tells us which
		// statement blocked. Use os.Stderr (unbuffered) so it flushes before exit.
		preview := strings.TrimSpace(stmt)
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		fmt.Fprintf(os.Stderr, "InitDB: stmt %d: %s\n", i, preview)

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			slog.Error("schema init: statement failed", "stmt_index", i, "stmt_preview", preview, "error", err)
			return fmt.Errorf("schema init stmt %d (%s): %w", i, preview, err)
		}
	}
	return nil
}
