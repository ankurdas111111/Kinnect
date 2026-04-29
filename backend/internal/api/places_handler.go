package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"kinnect-v3/internal/auth"
	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/db"
	"kinnect-v3/internal/shared"
)

// PlacesHandler handles HTTP CRUD for saved places. (F4)
type PlacesHandler struct {
	db    *sql.DB
	cache *cache.Cache
}

// List handles GET /api/places — returns all saved places for the authenticated user.
func (h *PlacesHandler) List(w http.ResponseWriter, r *http.Request) {
	sess := auth.GetSession(r)
	if sess == nil || sess.User == nil || sess.User.ID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "Not authenticated"})
		return
	}
	userID := sess.User.ID

	places, err := db.GetSavedPlacesForUser(context.Background(), h.db, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}
	if places == nil {
		places = []db.SavedPlaceEntry{}
	}
	writeJSON(w, http.StatusOK, places)
}

// Create handles POST /api/places — creates a new saved place.
func (h *PlacesHandler) Create(w http.ResponseWriter, r *http.Request) {
	sess := auth.GetSession(r)
	if sess == nil || sess.User == nil || sess.User.ID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "Not authenticated"})
		return
	}
	userID := sess.User.ID

	var req struct {
		Name      string  `json:"name"`
		Icon      string  `json:"icon"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		RadiusM   float64 `json:"radiusM"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Invalid request"})
		return
	}

	req.Name = shared.SanitizeString(strings.TrimSpace(req.Name), 100)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "name is required"})
		return
	}
	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid coordinates"})
		return
	}
	if req.RadiusM <= 0 || req.RadiusM > 50000 {
		req.RadiusM = 200 // sensible default
	}

	now := time.Now().UnixMilli()
	id, err := db.InsertSavedPlace(context.Background(), h.db, userID, req.Name, req.Icon, req.Latitude, req.Longitude, req.RadiusM, now)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}

	entry := db.SavedPlaceEntry{
		ID:        id,
		UserID:    userID,
		Name:      req.Name,
		Icon:      req.Icon,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		RadiusM:   req.RadiusM,
		CreatedAt: now,
	}
	h.cache.AddSavedPlace(userID, entry)

	writeJSON(w, http.StatusCreated, entry)
}

// Delete handles DELETE /api/places/{id} — deletes a saved place owned by the caller.
func (h *PlacesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	sess := auth.GetSession(r)
	if sess == nil || sess.User == nil || sess.User.ID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "Not authenticated"})
		return
	}
	userID := sess.User.ID

	placeID := r.PathValue("id")
	if placeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing id"})
		return
	}

	if err := db.DeleteSavedPlace(context.Background(), h.db, placeID, userID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}
	h.cache.RemoveSavedPlace(userID, placeID)

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
