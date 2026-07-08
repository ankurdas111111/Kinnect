package db

import (
	"context"
	"database/sql"
	"time"
)

// CreateUser inserts a new user and returns the ID.
func CreateUser(ctx context.Context, db *sql.DB,
	firstName, lastName, passwordHash, role, shareCode string,
	createdAt int64, email, mobile *string) (string, error) {
	var id string
	em, mo := nullStr(email), nullStr(mobile)
	err := db.QueryRowContext(ctx,
		`INSERT INTO users (first_name, last_name, password_hash, role, share_code, email, mobile, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		firstName, lastName, passwordHash, role, shareCode, em, mo, createdAt).Scan(&id)
	return id, err
}

// GetUserPasswordHash returns the password hash for a user. Returns ("", nil) if user not found.
func GetUserPasswordHash(ctx context.Context, db *sql.DB, userID string) (string, error) {
	var h string
	err := db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&h)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return h, err
}

// FindUserByEmail returns user ID for an email.
func FindUserByEmail(ctx context.Context, db *sql.DB, email string) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = $1 LIMIT 1`, email).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

// FindUserByMobile returns user ID for a mobile.
func FindUserByMobile(ctx context.Context, db *sql.DB, mobile string) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE mobile = $1 LIMIT 1`, mobile).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

// UpdateUserLocation updates last position for a user only when the new timestamp is
// strictly newer than what is already stored. This prevents a fire-and-forget disconnect
// goroutine (KR-004) from overwriting a more recent position written by a reconnected session.
func UpdateUserLocation(ctx context.Context, db *sql.DB, userID string, lat, lng float64, speed string, timestamp int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE users SET last_latitude=$1, last_longitude=$2, last_speed=$3, last_update=$4
		 WHERE id=$5 AND (last_update IS NULL OR last_update < $4)`,
		lat, lng, speed, timestamp, userID)
	return err
}

// CreateRoom inserts a room and returns the DB ID.
func CreateRoom(ctx context.Context, db *sql.DB, code, name, createdByUserID string, createdAt int64) (string, error) {
	var id string
	err := db.QueryRowContext(ctx,
		`INSERT INTO rooms (code, name, created_by, created_at) VALUES ($1, $2, $3, $4) RETURNING id`,
		code, name, createdByUserID, createdAt).Scan(&id)
	return id, err
}

// AddRoomMember adds a member to a room.
func AddRoomMember(ctx context.Context, db *sql.DB, roomID, userID, role string) error {
	if role == "" {
		role = "member"
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO room_members (room_id, user_id, role) VALUES ($1, $2, $3) ON CONFLICT (room_id, user_id) DO NOTHING`,
		roomID, userID, role)
	return err
}

// RemoveRoomMember removes a member from a room.
func RemoveRoomMember(ctx context.Context, db *sql.DB, roomID, userID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM room_members WHERE room_id = $1 AND user_id = $2`, roomID, userID)
	return err
}

// DeleteRoom deletes a room by ID.
func DeleteRoom(ctx context.Context, db *sql.DB, roomID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM rooms WHERE id = $1`, roomID)
	return err
}

// GetUserIDByShareCode looks up a user ID by share code directly from DB.
// Used as a cache-miss fallback in handleAddContact.
func GetUserIDByShareCode(ctx context.Context, db *sql.DB, shareCode string) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE share_code = $1`, shareCode).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

// AddContactBidirectional adds A->B and B->A contacts in a transaction.
func AddContactBidirectional(ctx context.Context, db *sql.DB, userA, userB string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO contacts (owner_id, contact_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userA, userB)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO contacts (owner_id, contact_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userB, userA)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// RemoveContactBidirectional removes both directions in a transaction.
func RemoveContactBidirectional(ctx context.Context, db *sql.DB, userA, userB string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `DELETE FROM contacts WHERE owner_id = $1 AND contact_id = $2`, userA, userB)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM contacts WHERE owner_id = $1 AND contact_id = $2`, userB, userA)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// CreateLiveToken inserts a live token.
func CreateLiveToken(ctx context.Context, db *sql.DB, token, userID string, expiresAt *int64, createdAt int64) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO live_tokens (token, user_id, expires_at, created_at) VALUES ($1, $2, $3, $4)`,
		token, userID, expiresAt, createdAt)
	return err
}

// DeleteLiveToken removes a live token.
func DeleteLiveToken(ctx context.Context, db *sql.DB, token string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM live_tokens WHERE token = $1`, token)
	return err
}

// DeleteExpiredLiveTokens removes expired tokens and returns deleted token strings.
func DeleteExpiredLiveTokens(ctx context.Context, db *sql.DB) ([]string, error) {
	now := time.Now().UnixMilli()
	rows, err := db.QueryContext(ctx,
		`DELETE FROM live_tokens WHERE expires_at IS NOT NULL AND expires_at <= $1 RETURNING token`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// CreateGuardianship inserts or updates a guardianship.
func CreateGuardianship(ctx context.Context, db *sql.DB, guardianID, wardID, status string, expiresAt *int64, createdAt int64, initiatedBy string) error {
	if initiatedBy == "" {
		initiatedBy = "guardian"
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO guardianships (guardian_id, ward_id, status, expires_at, created_at, initiated_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (guardian_id, ward_id) DO UPDATE SET status = $3, expires_at = $4, initiated_by = COALESCE($6, guardianships.initiated_by)`,
		guardianID, wardID, status, expiresAt, createdAt, initiatedBy)
	return err
}

// UpdateGuardianshipStatus updates the status of a guardianship.
func UpdateGuardianshipStatus(ctx context.Context, db *sql.DB, guardianID, wardID, status string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE guardianships SET status = $1 WHERE guardian_id = $2 AND ward_id = $3`,
		status, guardianID, wardID)
	return err
}

// ExpireGuardianships expires active guardianships past their expiry.
func ExpireGuardianships(ctx context.Context, db *sql.DB, now int64) ([]struct{ GuardianID, WardID string }, error) {
	rows, err := db.QueryContext(ctx,
		`UPDATE guardianships SET status = 'expired' WHERE status = 'active' AND expires_at IS NOT NULL AND expires_at <= $1
		 RETURNING guardian_id, ward_id`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct{ GuardianID, WardID string }
	for rows.Next() {
		var g, w string
		if err := rows.Scan(&g, &w); err != nil {
			return nil, err
		}
		out = append(out, struct{ GuardianID, WardID string }{g, w})
	}
	return out, rows.Err()
}

// SetRoomMemberRole sets role and expiry for a room member.
func SetRoomMemberRole(ctx context.Context, db *sql.DB, roomDbID, userID, role string, expiresAt *int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE room_members SET role = $1, role_expires_at = $2 WHERE room_id = $3 AND user_id = $4`,
		role, expiresAt, roomDbID, userID)
	return err
}

// ExpireRoomAdmins demotes expired room admins.
func ExpireRoomAdmins(ctx context.Context, db *sql.DB, now int64) ([]struct{ RoomID, UserID string }, error) {
	rows, err := db.QueryContext(ctx,
		`UPDATE room_members SET role = 'member', role_expires_at = NULL WHERE role = 'admin' AND role_expires_at IS NOT NULL AND role_expires_at <= $1
		 RETURNING room_id, user_id`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct{ RoomID, UserID string }
	for rows.Next() {
		var r, u string
		if err := rows.Scan(&r, &u); err != nil {
			return nil, err
		}
		out = append(out, struct{ RoomID, UserID string }{r, u})
	}
	return out, rows.Err()
}

// CreateRoomAdminRequest inserts a room admin request.
func CreateRoomAdminRequest(ctx context.Context, db *sql.DB, roomCode, userID string, expiresIn *string, createdAt int64) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO room_admin_requests (room_code, user_id, expires_in, created_at) VALUES ($1, $2, $3, $4) ON CONFLICT (room_code, user_id) DO NOTHING`,
		roomCode, userID, expiresIn, createdAt)
	return err
}

// DeleteRoomAdminRequest removes a room admin request and its votes.
func DeleteRoomAdminRequest(ctx context.Context, db *sql.DB, roomCode, userID string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `DELETE FROM room_admin_requests WHERE room_code = $1 AND user_id = $2`, roomCode, userID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM room_admin_votes WHERE room_code = $1 AND requester_id = $2`, roomCode, userID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// UpsertRoomAdminVote inserts or updates a vote.
func UpsertRoomAdminVote(ctx context.Context, db *sql.DB, roomCode, requesterID, voterID, vote string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO room_admin_votes (room_code, requester_id, voter_id, vote) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (room_code, requester_id, voter_id) DO UPDATE SET vote = $4`,
		roomCode, requesterID, voterID, vote)
	return err
}

// DeleteEmptyOldRooms deletes empty rooms older than maxAgeMs.
func DeleteEmptyOldRooms(ctx context.Context, db *sql.DB, maxAgeMs int64) error {
	cutoff := time.Now().UnixMilli() - maxAgeMs
	_, err := db.ExecContext(ctx,
		`DELETE FROM rooms WHERE created_at < $1 AND id NOT IN (SELECT DISTINCT room_id FROM room_members)`, cutoff)
	return err
}

// TableSize represents a table's size info.
type TableSize struct {
	Table string
	Size  string
	Rows  int64
}

// GetTableSizes returns table sizes and total DB size.
func GetTableSizes(ctx context.Context, db *sql.DB) (tables []TableSize, totalSize string, err error) {
	rows, err := db.QueryContext(ctx,
		`SELECT relname AS table, pg_size_pretty(pg_total_relation_size(relid)) AS size, n_live_tup AS rows
		 FROM pg_stat_user_tables ORDER BY pg_total_relation_size(relid) DESC`)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var t TableSize
		if err := rows.Scan(&t.Table, &t.Size, &t.Rows); err != nil {
			return nil, "", err
		}
		tables = append(tables, t)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	err = db.QueryRowContext(ctx, `SELECT pg_size_pretty(pg_database_size(current_database())) AS total`).Scan(&totalSize)
	if err != nil {
		return nil, "", err
	}
	return tables, totalSize, nil
}

// UpdateUserRole updates a user's role in the database.
func UpdateUserRole(ctx context.Context, db *sql.DB, userID, newRole string) error {
	_, err := db.ExecContext(ctx, `UPDATE users SET role = $1 WHERE id = $2`, newRole, userID)
	return err
}

func nullStr(p *string) interface{} {
	if p != nil {
		return *p
	}
	return nil
}

// ── F3: Meeting point ────────────────────────────────────────────────────────

// SetRoomMeetingPoint upserts the meeting point on a room identified by code.
func SetRoomMeetingPoint(ctx context.Context, db *sql.DB, roomCode string, lat, lng float64, label, setByUserID string, setAt int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE rooms SET meeting_lat=$1, meeting_lng=$2, meeting_label=$3, meeting_set_by=$4, meeting_set_at=$5
		 WHERE code=$6`,
		lat, lng, label, setByUserID, setAt, roomCode)
	return err
}

// ClearRoomMeetingPoint removes the meeting point from a room.
func ClearRoomMeetingPoint(ctx context.Context, db *sql.DB, roomCode string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE rooms SET meeting_lat=NULL, meeting_lng=NULL, meeting_label=NULL, meeting_set_by=NULL, meeting_set_at=NULL
		 WHERE code=$1`, roomCode)
	return err
}

// ── F6: Geofence event log ───────────────────────────────────────────────────

// InsertGeofenceEvent records a geofence entry or exit event.
func InsertGeofenceEvent(ctx context.Context, db *sql.DB, userID, fenceName, eventType string, lat, lng float64, ts int64) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO geofence_events (user_id, fence_name, event_type, lat, lng, ts)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, fenceName, eventType, lat, lng, ts)
	return err
}

// GetGeofenceEvents returns the most recent geofence events for a user.
func GetGeofenceEvents(ctx context.Context, db *sql.DB, userID string, limit int) ([]GeofenceEventRow, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, fence_name, event_type, lat, lng, ts
		 FROM geofence_events WHERE user_id=$1
		 ORDER BY ts DESC LIMIT $2`,
		userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GeofenceEventRow
	for rows.Next() {
		var r GeofenceEventRow
		if err := rows.Scan(&r.ID, &r.FenceName, &r.EventType, &r.Lat, &r.Lng, &r.Ts); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ── F7: Proximity alerts ─────────────────────────────────────────────────────

// UpsertProximityAlert inserts or updates a proximity alert and returns it.
func UpsertProximityAlert(ctx context.Context, db *sql.DB, ownerID, targetID string, radiusM int, createdAt int64) (*ProximityAlertEntry, error) {
	var id string
	err := db.QueryRowContext(ctx,
		`INSERT INTO proximity_alerts (owner_id, target_id, radius_m, enabled, last_triggered_at, created_at)
		 VALUES ($1, $2, $3, true, 0, $4)
		 ON CONFLICT (owner_id, target_id) DO UPDATE
		   SET radius_m = EXCLUDED.radius_m, enabled = true
		 RETURNING id`,
		ownerID, targetID, radiusM, createdAt).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &ProximityAlertEntry{
		ID:              id,
		OwnerID:         ownerID,
		TargetID:        targetID,
		RadiusM:         radiusM,
		Enabled:         true,
		LastTriggeredAt: 0,
		CreatedAt:       createdAt,
	}, nil
}

// DeleteProximityAlert removes a proximity alert by owner+target.
func DeleteProximityAlert(ctx context.Context, db *sql.DB, ownerID, targetID string) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM proximity_alerts WHERE owner_id=$1 AND target_id=$2`,
		ownerID, targetID)
	return err
}

// GetProximityAlertsForOwner returns all proximity alerts owned by a user.
func GetProximityAlertsForOwner(ctx context.Context, db *sql.DB, ownerID string) ([]*ProximityAlertEntry, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, owner_id, target_id, radius_m, enabled, last_triggered_at, created_at
		 FROM proximity_alerts WHERE owner_id=$1`,
		ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ProximityAlertEntry
	for rows.Next() {
		e := &ProximityAlertEntry{}
		if err := rows.Scan(&e.ID, &e.OwnerID, &e.TargetID, &e.RadiusM, &e.Enabled, &e.LastTriggeredAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpdateProximityAlertTriggered records the last triggered timestamp on an alert.
func UpdateProximityAlertTriggered(ctx context.Context, db *sql.DB, id string, ts int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE proximity_alerts SET last_triggered_at=$1 WHERE id=$2`,
		ts, id)
	return err
}

// ── F8: Room bulletin board ──────────────────────────────────────────────────

// InsertRoomNote inserts a new note and returns its ID.
func InsertRoomNote(ctx context.Context, db *sql.DB, roomID, authorID, body string, createdAt int64) (string, error) {
	var id string
	err := db.QueryRowContext(ctx,
		`INSERT INTO room_notes (room_id, author_id, body, created_at)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		roomID, authorID, body, createdAt).Scan(&id)
	return id, err
}

// EnforceRoomNoteCap deletes rows beyond 20 for a given room (oldest first).
func EnforceRoomNoteCap(ctx context.Context, db *sql.DB, roomID string) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM room_notes WHERE id IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (ORDER BY created_at DESC) AS rn
				FROM room_notes WHERE room_id = $1
			) t WHERE rn > 20
		)`, roomID)
	return err
}

// DeleteRoomNote deletes a note if the requester is the author.
func DeleteRoomNote(ctx context.Context, db *sql.DB, noteID, requesterID string) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM room_notes WHERE id=$1 AND author_id=$2`,
		noteID, requesterID)
	return err
}

// GetRoomNotes returns the most recent 20 notes for a room with author names.
func GetRoomNotes(ctx context.Context, db *sql.DB, roomID string) ([]RoomNoteRow, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT n.id, n.room_id, n.author_id,
		        COALESCE(u.first_name||' '||u.last_name, 'Unknown'),
		        n.body, n.created_at
		 FROM room_notes n
		 LEFT JOIN users u ON u.id = n.author_id
		 WHERE n.room_id = $1
		 ORDER BY n.created_at DESC
		 LIMIT 20`,
		roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RoomNoteRow
	for rows.Next() {
		var r RoomNoteRow
		if err := rows.Scan(&r.ID, &r.RoomID, &r.AuthorID, &r.AuthorName, &r.Body, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ── F9: Daily activity summary ───────────────────────────────────────────────

// UpsertDailyActivity upserts daily distance and active-minute counters for a user.
func UpsertDailyActivity(ctx context.Context, db *sql.DB, userID string, distanceDeltaM int, addActiveMinute bool, now int64) error {
	activeMinDelta := 0
	if addActiveMinute {
		activeMinDelta = 1
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO daily_activity (user_id, date, distance_m, active_minutes, updated_at)
		 VALUES ($1, CURRENT_DATE, $2, $3, $4)
		 ON CONFLICT (user_id, date) DO UPDATE
		   SET distance_m     = daily_activity.distance_m + EXCLUDED.distance_m,
		       active_minutes = daily_activity.active_minutes + EXCLUDED.active_minutes,
		       updated_at     = EXCLUDED.updated_at`,
		userID, distanceDeltaM, activeMinDelta, now)
	return err
}

// PersistPositionBundle writes everything a qualifying position update persists —
// users.last_* snapshot, daily_activity upsert, and the position_history trail row —
// in a single round trip via data-modifying CTEs. The previous 3 sequential
// statements cost 3 RTTs to a remote DB per write on a 10-connection pool.
// Guard semantics match UpdateUserLocation (only newer timestamps win).
func PersistPositionBundle(ctx context.Context, db *sql.DB, userID string, lat, lng float64, speedStr string, speed float64, ts int64, distanceDeltaM int, addActiveMinute bool) error {
	activeMinDelta := 0
	if addActiveMinute {
		activeMinDelta = 1
	}
	_, err := db.ExecContext(ctx, `
		WITH loc AS (
			UPDATE users SET last_latitude=$2, last_longitude=$3, last_speed=$4, last_update=$5
			WHERE id=$1 AND (last_update IS NULL OR last_update < $5)
		),
		act AS (
			INSERT INTO daily_activity (user_id, date, distance_m, active_minutes, updated_at)
			VALUES ($1, CURRENT_DATE, $6, $7, $5)
			ON CONFLICT (user_id, date) DO UPDATE
			  SET distance_m     = daily_activity.distance_m + EXCLUDED.distance_m,
			      active_minutes = daily_activity.active_minutes + EXCLUDED.active_minutes,
			      updated_at     = EXCLUDED.updated_at
		)
		INSERT INTO position_history (user_id, lat, lng, speed, ts)
		VALUES ($1, $2, $3, $8, $5)`,
		userID, lat, lng, speedStr, ts, distanceDeltaM, activeMinDelta, speed)
	return err
}

// GetDailyActivity returns the last N days of activity for a user.
func GetDailyActivity(ctx context.Context, db *sql.DB, userID string, days int) ([]DailyActivityRow, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT date::text, distance_m, active_minutes, updated_at
		 FROM daily_activity
		 WHERE user_id=$1 AND date >= CURRENT_DATE - ($2 - 1) * INTERVAL '1 day'
		 ORDER BY date DESC`,
		userID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DailyActivityRow
	for rows.Next() {
		var r DailyActivityRow
		if err := rows.Scan(&r.Date, &r.DistanceM, &r.ActiveMinutes, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ── F4: Saved places CRUD ────────────────────────────────────────────────────

// InsertSavedPlace inserts a new saved place and returns its ID.
func InsertSavedPlace(ctx context.Context, db *sql.DB, userID, name, icon string, lat, lng, radiusM float64, createdAt int64) (string, error) {
	var id string
	err := db.QueryRowContext(ctx,
		`INSERT INTO saved_places (user_id, name, icon, latitude, longitude, radius_m, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		userID, name, nullIfEmpty(icon), lat, lng, radiusM, createdAt).Scan(&id)
	return id, err
}

// DeleteSavedPlace deletes a saved place by ID restricted to the owner.
func DeleteSavedPlace(ctx context.Context, db *sql.DB, placeID, userID string) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM saved_places WHERE id=$1 AND user_id=$2`,
		placeID, userID)
	return err
}

// GetSavedPlacesForUser returns all saved places for a user.
func GetSavedPlacesForUser(ctx context.Context, db *sql.DB, userID string) ([]SavedPlaceEntry, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, user_id, name, COALESCE(icon,''), latitude, longitude, radius_m, created_at
		 FROM saved_places WHERE user_id=$1 ORDER BY created_at ASC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SavedPlaceEntry
	for rows.Next() {
		var e SavedPlaceEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Name, &e.Icon, &e.Latitude, &e.Longitude, &e.RadiusM, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
