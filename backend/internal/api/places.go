package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"kinnect-v3/internal/auth"
	"kinnect-v3/internal/shared"
)

// PlacesHandler handles saved places CRUD and zone story queries.
type PlacesHandler struct {
	db *sql.DB
}

type savedPlaceRow struct {
	ID         string  `json:"id"`
	UserID     string  `json:"userId"`
	Name       string  `json:"name"`
	Icon       string  `json:"icon"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	RadiusM    int     `json:"radiusM"`
	Visibility string  `json:"visibility"`
	RoomCode   string  `json:"roomCode,omitempty"`
	CreatedAt  int64   `json:"createdAt"`
}

// ListPlaces returns all saved places for the authenticated user.
func (h *PlacesHandler) ListPlaces(w http.ResponseWriter, r *http.Request) {
	sess := auth.GetSession(r)
	if sess == nil || sess.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, user_id, name, icon, latitude, longitude, radius_m, COALESCE(visibility,'personal'), COALESCE(room_code,''), created_at
		 FROM saved_places WHERE user_id = $1 ORDER BY created_at DESC`,
		sess.User.ID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var places []savedPlaceRow
	for rows.Next() {
		var p savedPlaceRow
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Icon, &p.Latitude, &p.Longitude, &p.RadiusM, &p.Visibility, &p.RoomCode, &p.CreatedAt); err != nil {
			continue
		}
		places = append(places, p)
	}
	if places == nil {
		places = []savedPlaceRow{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(places)
}

// CreatePlace inserts a new saved place.
func (h *PlacesHandler) CreatePlace(w http.ResponseWriter, r *http.Request) {
	sess := auth.GetSession(r)
	if sess == nil || sess.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Name       string  `json:"name"`
		Icon       string  `json:"icon"`
		Latitude   float64 `json:"latitude"`
		Longitude  float64 `json:"longitude"`
		RadiusM    int     `json:"radiusM"`
		Visibility string  `json:"visibility"`
		RoomCode   string  `json:"roomCode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	name := shared.SanitizeString(body.Name, 100)
	if name == "" {
		http.Error(w, "Name required", http.StatusBadRequest)
		return
	}
	icon := body.Icon
	if icon == "" {
		icon = "pin"
	}
	radiusM := body.RadiusM
	if radiusM < 0 {
		radiusM = 0
	}
	visibility := body.Visibility
	roomCode := ""
	switch visibility {
	case "universal":
		// no extra validation
	case "room":
		roomCode = body.RoomCode
		if len(roomCode) == 0 || len(roomCode) > 6 {
			http.Error(w, "Valid roomCode required for room visibility", http.StatusBadRequest)
			return
		}
		// Validate caller is a member of that room
		var dummy string
		err := h.db.QueryRowContext(r.Context(),
			`SELECT r.code FROM rooms r JOIN room_members rm ON rm.room_id = r.id
			 WHERE r.code = $1 AND rm.user_id = $2`,
			roomCode, sess.User.ID).Scan(&dummy)
		if err == sql.ErrNoRows {
			http.Error(w, "Not a member of that room", http.StatusForbidden)
			return
		}
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
	default:
		visibility = "personal"
	}
	createdAt := time.Now().UnixMilli()
	var id string
	var roomCodeArg interface{}
	if roomCode != "" {
		roomCodeArg = roomCode
	}
	err := h.db.QueryRowContext(r.Context(),
		`INSERT INTO saved_places (user_id, name, icon, latitude, longitude, radius_m, visibility, room_code, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		sess.User.ID, name, icon, body.Latitude, body.Longitude, radiusM, visibility, roomCodeArg, createdAt).Scan(&id)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(savedPlaceRow{
		ID: id, UserID: sess.User.ID, Name: name, Icon: icon,
		Latitude: body.Latitude, Longitude: body.Longitude,
		RadiusM: radiusM, Visibility: visibility, RoomCode: roomCode, CreatedAt: createdAt,
	})
}

// DeletePlace removes a saved place owned by the authenticated user.
func (h *PlacesHandler) DeletePlace(w http.ResponseWriter, r *http.Request) {
	sess := auth.GetSession(r)
	if sess == nil || sess.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	placeID := r.PathValue("placeId")
	if placeID == "" {
		http.Error(w, "Missing placeId", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`DELETE FROM saved_places WHERE id = $1 AND user_id = $2`,
		placeID, sess.User.ID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetZoneStory returns the visit history for a place (last 7 days).
func (h *PlacesHandler) GetZoneStory(w http.ResponseWriter, r *http.Request) {
	sess := auth.GetSession(r)
	if sess == nil || sess.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	placeID := r.PathValue("placeId")
	if placeID == "" {
		http.Error(w, "Missing placeId", http.StatusBadRequest)
		return
	}
	// Verify ownership
	var ownerID string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT user_id FROM saved_places WHERE id = $1`, placeID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err != nil || ownerID != sess.User.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT zv.id, u.first_name || ' ' || u.last_name as display_name,
		        zv.arrived_at, zv.departed_at, zv.duration_seconds
		 FROM zone_visits zv
		 JOIN users u ON zv.user_id = u.id
		 WHERE zv.place_id = $1 AND zv.arrived_at >= NOW() - INTERVAL '7 days'
		 ORDER BY zv.arrived_at DESC
		 LIMIT 100`,
		placeID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type visitRow struct {
		ID              string  `json:"id"`
		DisplayName     string  `json:"displayName"`
		ArrivedAt       string  `json:"arrivedAt"`
		DepartedAt      *string `json:"departedAt"`
		DurationSeconds *int    `json:"durationSeconds"`
	}
	var visits []visitRow
	for rows.Next() {
		var v visitRow
		var departed sql.NullTime
		var dur sql.NullInt32
		var arrived time.Time
		if err := rows.Scan(&v.ID, &v.DisplayName, &arrived, &departed, &dur); err != nil {
			continue
		}
		v.ArrivedAt = arrived.Format(time.RFC3339)
		if departed.Valid {
			s := departed.Time.Format(time.RFC3339)
			v.DepartedAt = &s
		}
		if dur.Valid {
			n := int(dur.Int32)
			v.DurationSeconds = &n
		}
		visits = append(visits, v)
	}
	if visits == nil {
		visits = []visitRow{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(visits)
}
