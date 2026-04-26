package ws

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/db"
	"kinnect-v3/internal/shared"
)

// zoneEntryKey is a composite key for tracking zone entry times.
type zoneEntryKey struct{ userID, placeID string }

// savedPlace mirrors a row from the saved_places table.
type savedPlace struct {
	ID       string
	Name     string
	Lat      float64
	Lng      float64
	RadiusM  float64
}

// arrivalState tracks per-user-per-place arrival state to avoid repeat emissions.
type arrivalState struct {
	mu              sync.Mutex
	lastEtaByUser   map[string]map[string]int64 // userID -> placeID -> lastEmittedEtaMs
	insidePlaces    map[string]map[string]bool  // userID -> placeID -> isInside
	entryTimes      map[zoneEntryKey]time.Time  // zone entry timestamps for visit recording
	placeNames      map[string]string           // placeID -> name (populated from DB during checkArrivals)
	lastColocationAt map[string]int64           // "uid1:uid2" (sorted) -> UnixMilli of last nudge
}

var arrival = &arrivalState{
	lastEtaByUser:   make(map[string]map[string]int64),
	insidePlaces:    make(map[string]map[string]bool),
	entryTimes:      make(map[zoneEntryKey]time.Time),
	placeNames:      make(map[string]string),
	lastColocationAt: make(map[string]int64),
}

// loadUserPlaces queries saved_places for a given userID.
func (h *Hub) loadUserPlaces(ctx context.Context, userID string) []savedPlace {
	rows, err := h.pool.DB.QueryContext(ctx,
		`SELECT id, name, latitude, longitude, radius_m FROM saved_places WHERE user_id = $1`,
		userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []savedPlace
	for rows.Next() {
		var p savedPlace
		if err := rows.Scan(&p.ID, &p.Name, &p.Lat, &p.Lng, &p.RadiusM); err != nil {
			continue
		}
		out = append(out, p)
	}
	return out
}

// StartArrivalMonitor runs a goroutine that polls active users every 30s for ETA projections.
func (h *Hub) StartArrivalMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	slog.Info("Arrival monitor started")
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.checkArrivals(ctx)
		}
	}
}

// userSnap is a lightweight position snapshot used in arrival and colocation checks.
type userSnap struct {
	userID      string
	displayName string
	lat, lng    float64
	speed       float64
}

func (h *Hub) checkArrivals(ctx context.Context) {
	// Collect snapshot of active users with position
	var users []userSnap
	h.Cache.ForEachActiveUser(func(_ string, u *cache.ActiveUser) {
		if u.Latitude == nil || u.Longitude == nil {
			return
		}
		users = append(users, userSnap{
			userID:      u.UserID,
			displayName: u.DisplayName,
			lat:         *u.Latitude,
			lng:         *u.Longitude,
			speed:       u.Speed,
		})
	})

	h.checkColocations(users)

	// Batch-load all saved places for all active users in a single DB query (P2).
	userIDs := make([]string, len(users))
	for i, u := range users {
		userIDs[i] = u.userID
	}
	allPlaces, err := db.GetPlacesForUsers(ctx, h.pool.DB, userIDs)
	if err != nil {
		slog.Warn("checkArrivals: failed to load places", "error", err)
		allPlaces = map[string][]db.SavedPlace{}
	}

	// Rebuild placeNames from scratch each cycle to avoid unbounded accumulation.
	arrival.mu.Lock()
	arrival.placeNames = make(map[string]string)
	for _, places := range allPlaces {
		for _, p := range places {
			arrival.placeNames[p.ID] = p.Name
		}
	}
	arrival.mu.Unlock()

	for _, u := range users {
		// Convert db.SavedPlace slice to local savedPlace slice.
		dbPlaces := allPlaces[u.userID]
		places := make([]savedPlace, len(dbPlaces))
		for i, p := range dbPlaces {
			places[i] = savedPlace{ID: p.ID, Name: p.Name, Lat: p.Lat, Lng: p.Lng, RadiusM: p.RadiusM}
		}
		// placeNames already populated above — no per-user lock needed here.
		for _, p := range places {
			dist := shared.HaversineM(u.lat, u.lng, p.Lat, p.Lng)
			isInside := dist <= p.RadiusM

			arrival.mu.Lock()
			if arrival.insidePlaces[u.userID] == nil {
				arrival.insidePlaces[u.userID] = make(map[string]bool)
			}
			wasInside := arrival.insidePlaces[u.userID][p.ID]

			if isInside && !wasInside {
				// Just arrived — record entry time for zone story
				arrival.insidePlaces[u.userID][p.ID] = true
				key := zoneEntryKey{u.userID, p.ID}
				arrival.entryTimes[key] = time.Now()
				arrival.mu.Unlock()
				// Dismiss outstanding ETA chip
				user := h.Cache.GetActiveUser(h.Cache.GetUserIdToSocketId(u.userID))
				if user != nil {
					h.emitToVisible(user, "arrivalDismiss", map[string]interface{}{
						"userId":  u.userID,
						"placeId": p.ID,
					})
				}
				// Dual-write: arrival event to movement_events
				go func(uid, pid, pname string, lat, lng float64) {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					mc := "still"
					_ = db.InsertMovementEvent(ctx, h.pool.DB, db.MovementEventRow{
						UserID:      uid,
						EventType:   "arrival",
						Lat:         &lat,
						Lng:         &lng,
						PlaceID:     &pid,
						PlaceName:   &pname,
						MotionClass: mc,
					})
				}(u.userID, p.ID, p.Name, u.lat, u.lng)
				continue
			}
			if !isInside && wasInside {
				// Just departed — write zone_visit row
				arrival.insidePlaces[u.userID][p.ID] = false
				key := zoneEntryKey{u.userID, p.ID}
				entryTime, hadEntry := arrival.entryTimes[key]
				if hadEntry {
					delete(arrival.entryTimes, key)
				}
				arrival.mu.Unlock()
				if hadEntry {
					departedAt := time.Now()
					durSec := int(departedAt.Sub(entryTime).Seconds())
					// Existing: zone_visits insert (dual-write phase)
					go func(uid, pid string, arr, dep time.Time, dur int) {
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						_, err := h.pool.DB.ExecContext(ctx,
							`INSERT INTO zone_visits (user_id, place_id, arrived_at, departed_at, duration_seconds)
							 VALUES ($1, $2, $3, $4, $5)`,
							uid, pid, arr, dep, dur)
						if err != nil {
							slog.Warn("Failed to write zone_visit", "error", err)
						}
					}(u.userID, p.ID, entryTime, departedAt, durSec)
					// Dual-write: departure event to movement_events
					go func(uid, pid, pname string, lat, lng float64, arr time.Time, dur int) {
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						mc := "still"
						_ = db.InsertMovementEvent(ctx, h.pool.DB, db.MovementEventRow{
							UserID:      uid,
							EventType:   "departure",
							Lat:         &lat,
							Lng:         &lng,
							PlaceID:     &pid,
							PlaceName:   &pname,
							MotionClass: mc,
							Metadata: map[string]interface{}{
								"dwell_seconds": dur,
								"arrived_at":    arr.UnixMilli(),
							},
						})
					}(u.userID, p.ID, p.Name, u.lat, u.lng, entryTime, durSec)
				}
			} else {
				arrival.mu.Unlock()
			}

			if isInside {
				continue
			}

			// Project ETA: speed in m/s (Haversine returns metres, speed is km/h)
			speedMs := u.speed / 3.6
			if speedMs < 0.3 || dist > 10000 { // not moving or >10km away
				continue
			}
			etaMs := int64((dist / speedMs) * 1000)
			if etaMs > 10*60*1000 { // more than 10 minutes
				continue
			}

			// Throttle emissions: only re-emit if ETA changed by >20s
			arrival.mu.Lock()
			if arrival.lastEtaByUser[u.userID] == nil {
				arrival.lastEtaByUser[u.userID] = make(map[string]int64)
			}
			lastEta := arrival.lastEtaByUser[u.userID][p.ID]
			if abs64(lastEta-etaMs) < 20000 {
				arrival.mu.Unlock()
				continue
			}
			arrival.lastEtaByUser[u.userID][p.ID] = etaMs
			arrival.mu.Unlock()

			user := h.Cache.GetActiveUser(h.Cache.GetUserIdToSocketId(u.userID))
			if user == nil {
				continue
			}
			payload := map[string]interface{}{
				"userId":      u.userID,
				"displayName": u.displayName,
				"placeName":   p.Name,
				"placeId":     p.ID,
				"etaSeconds":  etaMs / 1000,
				"distanceM":   fmt.Sprintf("%.0f", dist),
				"confidence":  "estimated",
			}
			// Emit to guardians and contacts who can see this user
			h.emitToVisible(user, "arrivalProjection", payload)
		}
	}
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// checkColocations scans all pairs of active users and emits a colocationNudge
// when two mutually-visible users are within 100m of each other.
// Rate-limited to once per 60 minutes per pair.
func (h *Hub) checkColocations(users []userSnap) {
	now := time.Now().UnixMilli()
	const colocationRadiusM = 100
	const colocationCooldownMs = 60 * 60 * 1000

	for i := 0; i < len(users); i++ {
		for j := i + 1; j < len(users); j++ {
			u1, u2 := users[i], users[j]
			dist := shared.HaversineM(u1.lat, u1.lng, u2.lat, u2.lng)
			if dist > colocationRadiusM {
				continue
			}
			// Check mutual visibility
			vis1 := h.Cache.GetVisibleSet(u1.userID)
			if !vis1[u2.userID] {
				continue
			}
			vis2 := h.Cache.GetVisibleSet(u2.userID)
			if !vis2[u1.userID] {
				continue
			}
			// Sorted pair key to avoid A:B and B:A duplicates
			key := u1.userID + ":" + u2.userID
			if u1.userID > u2.userID {
				key = u2.userID + ":" + u1.userID
			}
			arrival.mu.Lock()
			lastNudge := arrival.lastColocationAt[key]
			if now-lastNudge < colocationCooldownMs {
				arrival.mu.Unlock()
				continue
			}
			arrival.lastColocationAt[key] = now
			arrival.mu.Unlock()

			payload1 := map[string]interface{}{"userId": u2.userID, "displayName": u2.displayName}
			payload2 := map[string]interface{}{"userId": u1.userID, "displayName": u1.displayName}
			if sid := h.Cache.GetUserIdToSocketId(u1.userID); sid != "" {
				h.SendToClient(sid, "colocationNudge", payload1)
			}
			if sid := h.Cache.GetUserIdToSocketId(u2.userID); sid != "" {
				h.SendToClient(sid, "colocationNudge", payload2)
			}
		}
	}
}

// CloseZoneVisitsForUser writes zone_visit rows for any open entries for a user that just disconnected.
// Called from handleUnregister to prevent orphan open-ended rows.
func (h *Hub) CloseZoneVisitsForUser(userID string) {
	arrival.mu.Lock()
	var toClose []struct {
		placeID   string
		entryTime time.Time
	}
	for key, t := range arrival.entryTimes {
		if key.userID == userID {
			toClose = append(toClose, struct {
				placeID   string
				entryTime time.Time
			}{key.placeID, t})
			delete(arrival.entryTimes, key)
		}
	}
	delete(arrival.insidePlaces, userID)
	delete(arrival.lastEtaByUser, userID)
	// Remove all lastColocationAt entries that involve this user (either side of the pair key).
	for key := range arrival.lastColocationAt {
		// Keys are stored as "smallerID:largerID" — check both positions.
		if strings.HasPrefix(key, userID+":") || strings.HasSuffix(key, ":"+userID) {
			delete(arrival.lastColocationAt, key)
		}
	}
	arrival.mu.Unlock()

	if len(toClose) == 0 {
		return
	}

	departedAt := time.Now()
	for _, entry := range toClose {
		durSec := int(departedAt.Sub(entry.entryTime).Seconds())
		go func(pid string, arr, dep time.Time, dur int) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err := h.pool.DB.ExecContext(ctx,
				`INSERT INTO zone_visits (user_id, place_id, arrived_at, departed_at, duration_seconds)
				 VALUES ($1, $2, $3, $4, $5)`,
				userID, pid, arr, dep, dur)
			if err != nil {
				slog.Warn("Failed to close zone_visit on disconnect", "error", err)
			}
		}(entry.placeID, entry.entryTime, departedAt, durSec)
	}
}

// cleanupArrivalMaps removes lastColocationAt entries that are older than 2 hours.
// Called hourly from StartCleanupRoutines to prevent the map growing unbounded.
func cleanupArrivalMaps() {
	cutoff := time.Now().Add(-2 * time.Hour).UnixMilli()
	arrival.mu.Lock()
	defer arrival.mu.Unlock()
	for key, ts := range arrival.lastColocationAt {
		if ts < cutoff {
			delete(arrival.lastColocationAt, key)
		}
	}
}
