package cache

import (
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"kinnect-v3/internal/db"
)

// SosNarrative holds the context assembled at SOS trigger time.
type SosNarrative struct {
	TrackGeojson  string  `json:"trackGeojson"`
	MotionSummary string  `json:"motionSummary"`
	BatteryPct    *int    `json:"batteryPct"`
	TriggerRule   string  `json:"triggerRule"`
	LastSignalTs  int64   `json:"lastSignalTs"`
}

// SosAckEntry records a single SOS acknowledgement.
type SosAckEntry struct {
	By string `json:"by"`
}

// SOS holds SOS alert state for an active user.
type SOS struct {
	Active    bool
	At        *int64
	Reason    *string
	Type      *string
	Acks      []SosAckEntry
	Token     *string
	TokenExp  *int64
	Narrative *SosNarrative
}

// Geofence holds geofence config for an active user.
type Geofence struct {
	Enabled   bool
	CenterLat *float64
	CenterLng *float64
	RadiusM   float64
	WasInside *bool
}

// AutoSOS holds auto-SOS config for an active user.
type AutoSOS struct {
	Enabled       bool
	NoMoveMinutes int
	HardStopMin   int
	Geofence      bool
}

// CheckIn holds check-in config for an active user.
type CheckIn struct {
	Enabled            bool
	IntervalMin        int
	OverdueMin         int
	LastCheckInAt      int64
	RequestedAt        int64 // unix ms — last time checkInRequest was sent; prevents duplicate sends
	MissedNotifiedAt   int64 // unix ms — last time checkInMissed was sent; prevents duplicate sends
}

// Retention holds retention mode for an active user.
type Retention struct {
	Mode    string
	ClientID string
}

// ActiveUser represents a connected user with full runtime state.
type ActiveUser struct {
	SocketID         string
	UserID           string
	DisplayName      string
	Role             string
	Latitude         *float64
	Longitude        *float64
	Speed            float64
	LastUpdate       int64
	FormattedTime    string
	Accuracy         *float64
	BatteryPct       *int
	DeviceType       *string
	ConnectionQuality *string
	LastMoveAt       int64
	LastSpeed        float64
	HardStopAt       *int64

	// Meaningful-movement threshold tracking (not persisted)
	LastDBLat      float64 // lat at last DB write; 0 = never written
	LastDBLng      float64 // lng at last DB write
	LastDBAt       int64   // UnixMilli of last DB write; 0 = never written

	// Motion class (derived from speed, drives trip_start/trip_end events)
	MotionClass    string // "still" | "walk" | "run" | "vehicle"

	// Waypoint throttle: last time a waypoint event was written to movement_events
	LastWaypointAt int64 // UnixMilli; 0 = never written

	// Intelligence fields (computed on each position update)
	LastAttestAt  int64   // unix ms — last manual attestation; 0 = never
	MovementPhase string  // "stationary" | "walking" | "driving" | "transit"
	SafetyScore   float64 // 0–100, recomputed on each position update

	// Consumer context fields
	ActivityContext    string        // "At Home" | "Walking" | "In Transit" | "" (computed each position update)
	BatteryAlertSentAt map[int]int64 // threshold (20/10/5) -> UnixMilli of last alert sent

	// Ambient status message (user-set, max 60 chars)
	StatusMessage   string // "" means no active status
	StatusExpiresAt int64  // Unix ms; 0 = no expiry; auto-cleared by cleanup goroutine

	// Share My Ride — ephemeral ride-share state (cleared on endRide or disconnect)
	RideShareActive  bool
	RideShareVehicle string // optional vehicle number, max 15 chars
	RideShareDest    string // optional destination hint, max 30 chars
	RideShareToken   string // live link token used for this ride

	// Crowd Stay Together / Festival Mode
	CrowdModeActive   bool
	CrowdModeRadiusM  int              // alert radius in metres, default 200
	CrowdAlertSentAt  map[string]int64 // peerUserID -> UnixMilli of last crowdAlert sent

	GentleAlertSentAt  int64         // unix ms — last "haven't moved" gentle alert sent to guardians
	SOS              SOS
	Geofence         Geofence
	AutoSOS          AutoSOS
	CheckIn          CheckIn

	// Heartbeat Check — daily "sign of life" wellness monitoring
	HeartbeatEnabled    bool
	HeartbeatDeadline   string // "HH:MM" UTC
	HeartbeatLastSignal int64  // unix ms — updated on any position/checkIn/connect
	HeartbeatNotifiedAt int64  // unix ms — debounce: only notify once per day

	// Journey Shield — auto-trip detection (ephemeral, not persisted)
	TripActive       bool
	TripStartedAt    int64    // unix ms when trip confirmed
	TripVehicleStart int64    // unix ms when speed first exceeded 15km/h
	TripStoppedAt    int64    // unix ms when speed dropped to 0; 0 = still moving
	TripStartLat     float64
	TripStartLng     float64
	TripNotifiedStop bool     // debounce: already sent "stopped" notification

	// Panic Relay — external SMS escalation for SOS
	EmergencyPhone1 string
	EmergencyPhone2 string

	// Walk With Me — virtual escort session (ephemeral)
	WalkActive      bool
	WalkDestLat     float64
	WalkDestLng     float64
	WalkDestName    string
	WalkToken       string // live link token for the watcher
	WalkWatcherID   string // userId of the contact watching
	WalkStartedAt   int64
	WalkStoppedAt   int64  // unix ms when user stopped; 0 = moving
	WalkOfflineAt   int64  // unix ms when last position was received; for offline detection
	WalkDeviationNotifiedAt int64 // debounce deviation alerts

	Retention          *Retention
	PrivacyPausedUntil *int64
	QuietHoursEnabled  bool
	QuietHoursStart    string // "HH:MM" in user's local time — evaluated server-side as UTC window
	QuietHoursEnd      string
	Rooms              []string
	Online             bool
	ForceDelete        bool

	// F5: Speed alert threshold (m/s; 0 = disabled)
	SpeedAlertThresholdMs float64
	LastSpeedAlertAt      int64 // unix ms; rate-limit guard

	// F9: Daily activity — debounce active-minute counting
	LastActiveMinuteAt int64 // unix ms; ephemeral

	// Arrival projection — rate-limit guard (broadcast at most every 30s per user)
	LastArrivalProjectionAt int64 // unix ms; ephemeral
}

// WatchTokenEntry holds watch token state.
type WatchTokenEntry struct {
	SocketID string
	UserID   string
	Exp      int64
}

// OfflineEntry holds an offline user with expiry.
// C-4: RetentionForever preserves the "stored forever" UI label without requiring
// a nil ExpiresAt (which caused OOM by preventing the cleanup goroutine from evicting entries).
type OfflineEntry struct {
	User             *ActiveUser
	ExpiresAt        *int64
	RetentionForever bool
}

// maxOfflineUsers is the maximum number of offline entries retained in RAM.
// When this cap is exceeded, the entry with the earliest ExpiresAt is evicted. (C-4)
const maxOfflineUsers = 500

// PushSubscription holds a Web Push subscription for a user.
type PushSubscription struct {
	Endpoint string
	P256dh   string
	Auth     string
}

// Cache is the thread-safe in-memory cache. All access goes through methods.
type Cache struct {
	mu sync.RWMutex

	// Persistent (from DB)
	UsersCache      map[string]*db.UserCacheEntry
	ShareCodes      map[string]string
	EmailIndex      map[string]string
	MobileIndex     map[string]string
	Rooms           map[string]*db.RoomEntry
	RoomMemberRoles map[string]map[string]*db.RoomMemberRole
	Contacts        map[string]map[string]bool
	LiveTokens      map[string]*db.LiveTokenEntry
	Guardianships   map[string]map[string]*db.GuardianshipEntry

	// F4: Saved places (userID -> places)
	SavedPlaces map[string][]db.SavedPlaceEntry

	// F7: Proximity alerts (targetUserID -> alerts)
	ProximityAlerts map[string][]*db.ProximityAlertEntry

	// Reverse indexes for O(1) lookups
	WardToGuardians  map[string]map[string]bool // wardID -> set of guardianIDs
	OfflineBySocketID map[string]string         // socketID -> userID

	// Ephemeral
	WatchTokens      map[string]*WatchTokenEntry
	ActiveUsers      map[string]*ActiveUser
	OfflineUsers     map[string]*OfflineEntry
	VisibilityCache  map[string]map[string]bool
	LastVisibleSets  map[string]map[string]bool
	LastPositionAt   map[string]int64
	LastDbSaveAt     map[string]int64
	PendingRequests  map[string][]interface{}
	UserIdToSocketId map[string]string
	LiveTokensByUser map[string]map[string]bool
	UserRooms        map[string]map[string]bool
	AdminClientIds   map[string]bool
	PushSubs         map[string][]PushSubscription // userID -> subscriptions
	SharingSchedules map[string][]ScheduleRule      // userID -> rules

	// Secret chat invite tokens (ephemeral, in-memory only)
	SecretChatInvites map[string]*SecretChatInvite

	// M-6: CacheSize TTL fields — avoids holding RLock while iterating all nested maps.
	cachedSize   int64
	cachedSizeAt time.Time

	// Lazy loading
	lazyLoader *LazyLoader

	// M-5: optional cache-miss callback. Set via SetMetricsHook to avoid import cycle.
	onCacheMiss func()
}

// SecretChatInvite is an ephemeral token linking to a secret chat conversation.
type SecretChatInvite struct {
	Token     string
	OwnerID   string // userId of person who generated the link
	PeerID    string // userId of the other person in the conversation
	ExpiresAt int64  // UnixMilli
}

// ScheduleRule defines when a user shares their location with a specific target.
type ScheduleRule struct {
	ID         string
	TargetType string // "all" | "contact" | "room"
	TargetID   string // userID or roomID; empty for "all"
	DayMask    int    // bitmask Mon=1,Tue=2,Wed=4,Thu=8,Fri=16,Sat=32,Sun=64
	StartMin   int    // minutes from midnight
	EndMin     int    // minutes from midnight
	Enabled    bool
}

// New creates a new Cache.
func New() *Cache {
	return &Cache{
		UsersCache:        make(map[string]*db.UserCacheEntry),
		ShareCodes:        make(map[string]string),
		EmailIndex:        make(map[string]string),
		MobileIndex:       make(map[string]string),
		Rooms:             make(map[string]*db.RoomEntry),
		RoomMemberRoles:   make(map[string]map[string]*db.RoomMemberRole),
		Contacts:          make(map[string]map[string]bool),
		LiveTokens:        make(map[string]*db.LiveTokenEntry),
		Guardianships:     make(map[string]map[string]*db.GuardianshipEntry),
		SavedPlaces:       make(map[string][]db.SavedPlaceEntry),
		ProximityAlerts:   make(map[string][]*db.ProximityAlertEntry),
		WardToGuardians:   make(map[string]map[string]bool),
		OfflineBySocketID: make(map[string]string),
		WatchTokens:       make(map[string]*WatchTokenEntry),
		ActiveUsers:       make(map[string]*ActiveUser),
		OfflineUsers:      make(map[string]*OfflineEntry),
		VisibilityCache:   make(map[string]map[string]bool),
		LastVisibleSets:   make(map[string]map[string]bool),
		LastPositionAt:    make(map[string]int64),
		LastDbSaveAt:      make(map[string]int64),
		PendingRequests:   make(map[string][]interface{}),
		UserIdToSocketId:  make(map[string]string),
		LiveTokensByUser:  make(map[string]map[string]bool),
		UserRooms:         make(map[string]map[string]bool),
		AdminClientIds:    make(map[string]bool),
		PushSubs:            make(map[string][]PushSubscription),
		SharingSchedules:    make(map[string][]ScheduleRule),
		SecretChatInvites:   make(map[string]*SecretChatInvite),
		lazyLoader:          nil, // Set via SetLazyLoader
	}
}

// SetMetricsHook wires a callback invoked on every lazy-load cache miss (M-5).
// Use a closure over *monitoring.Metrics to avoid an import cycle.
func (c *Cache) SetMetricsHook(onMiss func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onCacheMiss = onMiss
}

// Init populates the cache from LoadAllResult.
func (c *Cache) Init(result *db.LoadAllResult) {
	if result == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.UsersCache = result.UsersCache
	if c.UsersCache == nil {
		c.UsersCache = make(map[string]*db.UserCacheEntry)
	}
	// Stamp all loaded users with current access time so LRU starts with a fair baseline.
	now := time.Now().UnixMilli()
	for _, e := range c.UsersCache {
		e.LastAccessedAt = now
	}
	c.ShareCodes = result.ShareCodes
	if c.ShareCodes == nil {
		c.ShareCodes = make(map[string]string)
	}
	c.EmailIndex = result.EmailIndex
	if c.EmailIndex == nil {
		c.EmailIndex = make(map[string]string)
	}
	c.MobileIndex = result.MobileIndex
	if c.MobileIndex == nil {
		c.MobileIndex = make(map[string]string)
	}
	c.Rooms = result.Rooms
	if c.Rooms == nil {
		c.Rooms = make(map[string]*db.RoomEntry)
	}
	c.RoomMemberRoles = result.RoomMemberRoles
	if c.RoomMemberRoles == nil {
		c.RoomMemberRoles = make(map[string]map[string]*db.RoomMemberRole)
	}
	c.Contacts = result.Contacts
	if c.Contacts == nil {
		c.Contacts = make(map[string]map[string]bool)
	}
	c.LiveTokens = result.LiveTokens
	if c.LiveTokens == nil {
		c.LiveTokens = make(map[string]*db.LiveTokenEntry)
	}
	c.Guardianships = result.Guardianships
	if c.Guardianships == nil {
		c.Guardianships = make(map[string]map[string]*db.GuardianshipEntry)
	}

	// F4: saved places
	c.SavedPlaces = result.SavedPlaces
	if c.SavedPlaces == nil {
		c.SavedPlaces = make(map[string][]db.SavedPlaceEntry)
	}

	// F7: proximity alerts
	c.ProximityAlerts = result.ProximityAlerts
	if c.ProximityAlerts == nil {
		c.ProximityAlerts = make(map[string][]*db.ProximityAlertEntry)
	}

	// Build WardToGuardians reverse index from loaded guardianships
	c.WardToGuardians = make(map[string]map[string]bool)
	for guardianID, wards := range c.Guardianships {
		for wardID := range wards {
			if c.WardToGuardians[wardID] == nil {
				c.WardToGuardians[wardID] = make(map[string]bool)
			}
			c.WardToGuardians[wardID][guardianID] = true
		}
	}

	for key, reqs := range result.RoomAdminRequests {
		if c.PendingRequests == nil {
			c.PendingRequests = make(map[string][]interface{})
		}
		for _, r := range reqs {
			c.PendingRequests[key] = append(c.PendingRequests[key], r)
		}
	}

	for token, entry := range c.LiveTokens {
		if c.LiveTokensByUser == nil {
			c.LiveTokensByUser = make(map[string]map[string]bool)
		}
		if c.LiveTokensByUser[entry.UserID] == nil {
			c.LiveTokensByUser[entry.UserID] = make(map[string]bool)
		}
		c.LiveTokensByUser[entry.UserID][token] = true
	}

	for code, room := range c.Rooms {
		for memberID := range room.Members {
			if c.UserRooms == nil {
				c.UserRooms = make(map[string]map[string]bool)
			}
			if c.UserRooms[memberID] == nil {
				c.UserRooms[memberID] = make(map[string]bool)
			}
			c.UserRooms[memberID][code] = true
		}
	}

	slog.Info("Cache initialized",
		"users", len(c.UsersCache),
		"rooms", len(c.Rooms),
		"contacts", len(c.Contacts),
		"live_tokens", len(c.LiveTokens),
		"guardianships", len(c.Guardianships),
		"saved_places_users", len(c.SavedPlaces),
		"proximity_alert_targets", len(c.ProximityAlerts))
}

// haversineM returns distance in metres between two lat/lng points.
// Inlined here to avoid importing ws package (would create a cycle).
func haversineM(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// GetSavedPlaces returns a copy of the saved places slice for userID (thread-safe).
func (c *Cache) GetSavedPlaces(userID string) []db.SavedPlaceEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	places := c.SavedPlaces[userID]
	if len(places) == 0 {
		return nil
	}
	out := make([]db.SavedPlaceEntry, len(places))
	copy(out, places)
	return out
}

// inferLocationLabel returns the name of the first saved place the user is currently
// within, or "" if none match. Must be called with c.mu RLock held (or any lock).
func (c *Cache) inferLocationLabel(userID string, lat, lng *float64) string {
	if lat == nil || lng == nil {
		return ""
	}
	places := c.SavedPlaces[userID]
	for _, p := range places {
		if haversineM(*lat, *lng, p.Latitude, p.Longitude) <= p.RadiusM {
			return p.Name
		}
	}
	return ""
}

// GetGuardianSocketIDs returns socket IDs of all active guardians watching wardID. (F5)
func (c *Cache) GetGuardianSocketIDs(wardID string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	guardianIDs := c.WardToGuardians[wardID]
	if len(guardianIDs) == 0 {
		return nil
	}
	var out []string
	for gID := range guardianIDs {
		if sid := c.UserIdToSocketId[gID]; sid != "" {
			out = append(out, sid)
		}
	}
	return out
}

// GetProximityAlertsForTarget returns all enabled proximity alerts targeting a user. (F7)
func (c *Cache) GetProximityAlertsForTarget(targetID string) []*db.ProximityAlertEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ProximityAlerts[targetID]
}

// GetUserIDByEmail returns user ID for an email (case-insensitive).
func (c *Cache) GetUserIDByEmail(email string) string {
	c.mu.RLock()
	userID := c.EmailIndex[strings.ToLower(email)]
	c.mu.RUnlock()

	if userID != "" {
		return userID
	}

	// Not in index: try lazy loading from database
	if c.lazyLoader != nil {
		return c.lazyLoader.FindUserByEmail(strings.ToLower(email))
	}

	return ""
}

// GetUserIDByMobile returns user ID for a mobile number.
func (c *Cache) GetUserIDByMobile(mobile string) string {
	c.mu.RLock()
	userID := c.MobileIndex[mobile]
	c.mu.RUnlock()

	if userID != "" {
		return userID
	}

	// Not in index: try lazy loading from database
	if c.lazyLoader != nil {
		return c.lazyLoader.FindUserByMobile(mobile)
	}

	return ""
}

// GetUserRole returns the role for a user.
func (c *Cache) GetUserRole(userID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if u := c.UsersCache[userID]; u != nil {
		return u.Role
	}
	return ""
}

// RoomCount returns the number of rooms in the cache.
func (c *Cache) RoomCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Rooms)
}

// ShareCodeExists returns true if the share code is already used.
func (c *Cache) ShareCodeExists(code string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.ShareCodes[code]
	return ok
}

// GetLiveToken returns the live token entry, or nil.
func (c *Cache) GetLiveToken(token string) *db.LiveTokenEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LiveTokens[token]
}

// DeleteLiveToken removes a live token from the cache.
func (c *Cache) DeleteLiveToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.LiveTokens[token]; e != nil {
		delete(c.LiveTokens, token)
		if c.LiveTokensByUser[e.UserID] != nil {
			delete(c.LiveTokensByUser[e.UserID], token)
		}
	}
}

// GetWatchToken returns the watch token entry, or nil.
func (c *Cache) GetWatchToken(token string) *WatchTokenEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.WatchTokens[token]
}

// DeleteWatchToken removes a watch token from the cache.
func (c *Cache) DeleteWatchToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.WatchTokens, token)
}

// GetDisplayName returns the display name for a user.
func (c *Cache) GetDisplayName(userID string) string {
	c.mu.RLock()
	u, ok := c.UsersCache[userID]
	c.mu.RUnlock()

	if !ok {
		// M-5: count cache misses for Prometheus.
		c.mu.RLock()
		onMiss := c.onCacheMiss
		c.mu.RUnlock()
		if onMiss != nil {
			onMiss()
		}
		// Try lazy loading if not in cache
		u = c.LoadUserCacheEntry(userID)
		if u == nil {
			return "Unknown"
		}
	}

	name := u.FirstName + " " + u.LastName
	if name == " " || strings.TrimSpace(name) == "" {
		return "Unknown"
	}
	return strings.TrimSpace(name)
}

// GetUsersCache returns a copy of UsersCache (caller must hold lock for direct access patterns).
// For simple reads, use RLock and read.
func (c *Cache) GetUsersCache() map[string]*db.UserCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.UsersCache
}

// GetUserRooms returns room codes the user is a member of.
func (c *Cache) GetUserRooms(userID string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	roomSet, ok := c.UserRooms[userID]
	if !ok || len(roomSet) == 0 {
		return nil
	}
	out := make([]string, 0, len(roomSet))
	for code := range roomSet {
		out = append(out, code)
	}
	return out
}

// GetVisibleSet returns the set of user IDs visible to the given user (rooms, contacts both ways).
// H-2: double-checked locking — hot path uses RLock on cache hit; write lock only on miss.
func (c *Cache) GetVisibleSet(userID string) map[string]bool {
	// Hot path: check under read lock.
	c.mu.RLock()
	if cached, ok := c.VisibilityCache[userID]; ok && len(cached) > 0 {
		result := cached
		c.mu.RUnlock()
		return result
	}
	c.mu.RUnlock()

	// Cache miss: acquire write lock, recheck, then compute.
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getVisibleSetLocked(userID)
}

// InvalidateVisibility clears cached visibility for a user (call after room/contact mutations).
func (c *Cache) InvalidateVisibility(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.VisibilityCache, userID)
}

// HasSharingSchedules returns true if the user has any schedule rules configured.
func (c *Cache) HasSharingSchedules(userID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.SharingSchedules[userID]) > 0
}

// SetSharingSchedules replaces a user's sharing schedule rules in the cache.
func (c *Cache) SetSharingSchedules(userID string, rules []ScheduleRule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SharingSchedules[userID] = rules
}

// IsScheduleVisible returns false if the sender has schedule rules that block the recipient now.
// Returns true (share) by default when no applicable rules exist.
func (c *Cache) IsScheduleVisible(senderID, recipientID string, recipientRooms []string) bool {
	c.mu.RLock()
	rules, ok := c.SharingSchedules[senderID]
	c.mu.RUnlock()
	if !ok || len(rules) == 0 {
		return true // no rules → always share
	}
	now := time.Now().UTC()
	// Go's Weekday: Sun=0, Mon=1...Sat=6. Plan bitmask: Mon=1,Tue=2,...,Sun=64
	wd := int(now.Weekday())
	var dayBit int
	if wd == 0 {
		dayBit = 64 // Sunday
	} else {
		dayBit = 1 << (wd - 1) // Mon=1, Tue=2, ...
	}
	nowMin := now.Hour()*60 + now.Minute()

	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		// Check if this rule applies to the recipient
		switch r.TargetType {
		case "contact":
			if r.TargetID != recipientID {
				continue
			}
		case "room":
			found := false
			for _, code := range recipientRooms {
				if code == r.TargetID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		// "all" matches everyone
		}
		if r.DayMask != 0 && (r.DayMask&dayBit) == 0 {
			continue
		}
		var inWindow bool
		if r.StartMin <= r.EndMin {
			inWindow = nowMin >= r.StartMin && nowMin < r.EndMin
		} else {
			inWindow = nowMin >= r.StartMin || nowMin < r.EndMin
		}
		if inWindow {
			return true // an active sharing window covers this recipient
		}
	}
	// Sender has rules but none match this recipient now → don't share
	return false
}

// IsGuardianOf returns true if guardianID is an active guardian of wardID.
func (c *Cache) IsGuardianOf(guardianID, wardID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if wards, ok := c.Guardianships[guardianID]; ok {
		if e, ok := wards[wardID]; ok && e != nil && e.Status == "active" {
			return true
		}
	}
	return false
}

// GetVisibleSocketIDs returns socket IDs of users who can see targetUser (for emitToVisible).
// Includes admins. Excludes targetUser's own socket.
// H-2: double-checked locking — hot path uses RLock when visibility is already cached.
func (c *Cache) GetVisibleSocketIDs(targetUser *ActiveUser) []string {
	userID := targetUser.UserID

	// Hot path: try read lock first.
	c.mu.RLock()
	if cached, ok := c.VisibilityCache[userID]; ok && len(cached) > 0 {
		var out []string
		seen := make(map[string]bool)
		for uid := range cached {
			if uid == userID {
				continue
			}
			if sid, ok := c.UserIdToSocketId[uid]; ok && sid != targetUser.SocketID {
				out = append(out, sid)
				seen[sid] = true
			}
		}
		for sid := range c.AdminClientIds {
			if sid != targetUser.SocketID && !seen[sid] {
				out = append(out, sid)
			}
		}
		c.mu.RUnlock()
		return out
	}
	c.mu.RUnlock()

	// Cache miss: acquire write lock, compute visibility (getVisibleSetLocked caches it).
	c.mu.Lock()
	defer c.mu.Unlock()

	visibleSet := c.getVisibleSetLocked(userID)
	var out []string
	seen := make(map[string]bool)
	for uid := range visibleSet {
		if uid == userID {
			continue
		}
		if sid, ok := c.UserIdToSocketId[uid]; ok && sid != targetUser.SocketID {
			out = append(out, sid)
			seen[sid] = true
		}
	}
	// KR-008: deduplicate admin socket IDs — an admin who is also a contact/room
	// member would otherwise appear twice and receive every event twice.
	for sid := range c.AdminClientIds {
		if sid != targetUser.SocketID && !seen[sid] {
			out = append(out, sid)
		}
	}
	return out
}

// getVisibleSetLocked assumes c.mu is held. Returns visible set for userID.
// Includes: self, room members, contacts (both directions), guardians/wards.
func (c *Cache) getVisibleSetLocked(userID string) map[string]bool {
	if cached, ok := c.VisibilityCache[userID]; ok && len(cached) > 0 {
		return cached
	}
	visible := make(map[string]bool)
	visible[userID] = true
	if roomSet, ok := c.UserRooms[userID]; ok {
		for code := range roomSet {
			if room, ok := c.Rooms[code]; ok {
				for mid := range room.Members {
					visible[mid] = true
				}
			}
		}
	}
	// Contacts are stored bidirectionally, so Contacts[userID] already covers both directions.
	if contactSet, ok := c.Contacts[userID]; ok {
		for cid := range contactSet {
			visible[cid] = true
		}
	}
	// Guardians: I see my wards (I am guardian) and my guardians (WardToGuardians index).
	// KR-009: only active guardianships grant visibility — pending requests do not, to
	// prevent a malicious actor from discovering someone's location via an unsolicited request.
	if wards, ok := c.Guardianships[userID]; ok {
		for wID, e := range wards {
			if e != nil && e.Status == "active" {
				visible[wID] = true
			}
		}
	}
	for gID := range c.WardToGuardians[userID] {
		if wards, ok := c.Guardianships[gID]; ok {
			if e := wards[userID]; e != nil && e.Status == "active" { // KR-009: active only
				visible[gID] = true
			}
		}
	}
	c.VisibilityCache[userID] = visible
	return visible
}

// SanitizeUser produces a map suitable for JSON emission to clients.
// Always returns a fresh map; callers may safely mutate it.
func (c *Cache) SanitizeUser(user *ActiveUser) map[string]interface{} {
	c.mu.RLock()
	locationLabel := c.inferLocationLabel(user.UserID, user.Latitude, user.Longitude)
	c.mu.RUnlock()

	result := map[string]interface{}{
		"socketId":        user.SocketID,
		"userId":          user.UserID,
		"displayName":     user.DisplayName,
		"role":            user.Role,
		"latitude":        user.Latitude,
		"longitude":       user.Longitude,
		"speed":           user.Speed,
		"lastUpdate":      user.LastUpdate,
		"formattedTime":   user.FormattedTime,
		"batteryPct":      user.BatteryPct,
		"deviceType":      user.DeviceType,
		"connectionQuality": user.ConnectionQuality,
		"online":          user.Online,
		"rooms":           user.Rooms,
		"safetyScore":     user.SafetyScore,
		"movementPhase":   user.MovementPhase,
		"motionClass":     user.MotionClass,
		"lastAttestAt":    user.LastAttestAt,
		"activityContext": user.ActivityContext,
		"statusMessage":   user.StatusMessage,
		"locationLabel":   locationLabel, // F4
		"rideShare": map[string]interface{}{
			"active":  user.RideShareActive,
			"vehicle": user.RideShareVehicle,
			"dest":    user.RideShareDest,
		},
		"crowdMode": map[string]interface{}{
			"active":  user.CrowdModeActive,
			"radiusM": user.CrowdModeRadiusM,
		},
		"sos": map[string]interface{}{
			"active": user.SOS.Active,
			"at":     user.SOS.At,
			"reason": user.SOS.Reason,
			"type":   user.SOS.Type,
		},
		"geofence": map[string]interface{}{
			"enabled":   user.Geofence.Enabled,
			"centerLat": user.Geofence.CenterLat,
			"centerLng": user.Geofence.CenterLng,
			"radiusM":   user.Geofence.RadiusM,
		},
		"autoSos": map[string]interface{}{
			"enabled":   user.AutoSOS.Enabled,
			"noMoveMinutes":  user.AutoSOS.NoMoveMinutes,
			"hardStopMin":    user.AutoSOS.HardStopMin,
			"geofence": user.AutoSOS.Geofence,
		},
		"checkIn": map[string]interface{}{
			"enabled":       user.CheckIn.Enabled,
			"intervalMin":   user.CheckIn.IntervalMin,
			"overdueMin":    user.CheckIn.OverdueMin,
			"lastCheckInAt": user.CheckIn.LastCheckInAt,
		},
		"quietHours": map[string]interface{}{
			"enabled": user.QuietHoursEnabled,
			"start":   user.QuietHoursStart,
			"end":     user.QuietHoursEnd,
		},
		"heartbeat": map[string]interface{}{
			"enabled":  user.HeartbeatEnabled,
			"deadline": user.HeartbeatDeadline,
		},
		"tripActive": user.TripActive,
		"walkWithMe": map[string]interface{}{
			"active":   user.WalkActive,
			"destName": user.WalkDestName,
		},
		"retention": nil,
	}
	if user.Retention != nil {
		result["retention"] = map[string]interface{}{"mode": user.Retention.Mode}
	}
	return result
}

// GetUser returns a user cache entry by ID. Caller must not modify.
// Updates LastAccessedAt under a write lock so EvictLRU can track recency.
func (c *Cache) GetUser(userID string) *db.UserCacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.UsersCache[userID]
	if e != nil {
		e.LastAccessedAt = time.Now().UnixMilli()
	}
	return e
}

// HasShareCode returns true if the share code exists.
func (c *Cache) HasShareCode(code string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.ShareCodes[code]
	return ok
}

// GetUserIDByShareCode returns the user ID for a share code, or "".
func (c *Cache) GetUserIDByShareCode(code string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ShareCodes[code]
}

// WarmShareCode stores a share code → userID mapping that was resolved via DB fallback.
func (c *Cache) WarmShareCode(shareCode, userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ShareCodes[shareCode] = userID
}

// AreContacts returns true if userA has userB in their contact list.
func (c *Cache) AreContacts(userA, userB string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Contacts[userA] != nil && c.Contacts[userA][userB]
}

// SetActiveUser adds or replaces an active user. Also updates UserIdToSocketId and AdminClientIds.
func (c *Cache) SetActiveUser(socketID string, user *ActiveUser) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ActiveUsers[socketID] = user
	c.UserIdToSocketId[user.UserID] = socketID
	if user.Role == "admin" {
		c.AdminClientIds[socketID] = true
	}
}

// DeleteActiveUser removes an active user by socket ID and cleans related maps.
func (c *Cache) DeleteActiveUser(socketID string) *ActiveUser {
	c.mu.Lock()
	defer c.mu.Unlock()
	u := c.ActiveUsers[socketID]
	if u != nil && c.UserIdToSocketId[u.UserID] == socketID {
		delete(c.UserIdToSocketId, u.UserID)
	}
	delete(c.ActiveUsers, socketID)
	delete(c.AdminClientIds, socketID)
	delete(c.LastVisibleSets, socketID)
	return u
}

// GetActiveUser returns the active user by socket ID, or nil.
func (c *Cache) GetActiveUser(socketID string) *ActiveUser {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ActiveUsers[socketID]
}

// SetUserIdToSocketId sets the mapping from user ID to socket ID.
func (c *Cache) SetUserIdToSocketId(userID, socketID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.UserIdToSocketId[userID] = socketID
}

// DeleteUserIdToSocketId removes the mapping for a user ID.
func (c *Cache) DeleteUserIdToSocketId(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.UserIdToSocketId, userID)
}

// GetUserIdToSocketId returns the socket ID for a user, or "".
func (c *Cache) GetUserIdToSocketId(userID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.UserIdToSocketId[userID]
}

// GetOfflineUser returns the offline entry for a user, or nil.
func (c *Cache) GetOfflineUser(userID string) *OfflineEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.OfflineUsers[userID]
}

// SetOfflineUser adds or updates an offline user entry.
// C-4: enforces a cap of maxOfflineUsers (500). When at cap, evicts the entry with
// the earliest ExpiresAt before inserting to prevent OOM on 512 MB hosts.
func (c *Cache) SetOfflineUser(userID string, entry *OfflineEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Remove old socketID index entry if present
	if old := c.OfflineUsers[userID]; old != nil && old.User != nil {
		delete(c.OfflineBySocketID, old.User.SocketID)
	}

	// C-4: evict if cap is reached and this is a new slot (not a replacement).
	_, alreadyExists := c.OfflineUsers[userID]
	if !alreadyExists && len(c.OfflineUsers) >= maxOfflineUsers {
		c.evictEarliestOfflineUserLocked()
	}

	c.OfflineUsers[userID] = entry
	if entry != nil && entry.User != nil {
		c.OfflineBySocketID[entry.User.SocketID] = userID
	}
}

// evictEarliestOfflineUserLocked removes the offline entry with the smallest ExpiresAt.
// Must be called with c.mu held.
func (c *Cache) evictEarliestOfflineUserLocked() {
	var earliestUID string
	var earliestExp int64
	first := true
	for uid, e := range c.OfflineUsers {
		if e.ExpiresAt == nil {
			// Nil ExpiresAt should not occur after the C-4 fix, but guard defensively.
			if first {
				earliestUID = uid
				first = false
			}
			continue
		}
		if first || *e.ExpiresAt < earliestExp {
			earliestUID = uid
			earliestExp = *e.ExpiresAt
			first = false
		}
	}
	if earliestUID != "" {
		if old := c.OfflineUsers[earliestUID]; old != nil && old.User != nil {
			delete(c.OfflineBySocketID, old.User.SocketID)
		}
		delete(c.OfflineUsers, earliestUID)
		slog.Warn("OfflineUsers cap reached, evicted earliest entry",
			"evictedUserID", earliestUID, "cap", maxOfflineUsers)
	}
}

// DeleteOfflineUser removes an offline user.
func (c *Cache) DeleteOfflineUser(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry := c.OfflineUsers[userID]; entry != nil && entry.User != nil {
		delete(c.OfflineBySocketID, entry.User.SocketID)
	}
	delete(c.OfflineUsers, userID)
}

// GetOfflineUserBySocketID returns the ActiveUser for an offline user with the given socket ID, or nil.
func (c *Cache) GetOfflineUserBySocketID(socketID string) *ActiveUser {
	c.mu.RLock()
	defer c.mu.RUnlock()
	uid := c.OfflineBySocketID[socketID]
	if uid == "" {
		return nil
	}
	if entry := c.OfflineUsers[uid]; entry != nil {
		return entry.User
	}
	return nil
}

// SetAdminClientId adds or removes a client ID from admin clients.
func (c *Cache) SetAdminClientId(clientID string, isAdmin bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if isAdmin {
		c.AdminClientIds[clientID] = true
	} else {
		delete(c.AdminClientIds, clientID)
	}
}

// DeleteAdminClientId removes a client from admin clients.
func (c *Cache) DeleteAdminClientId(clientID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.AdminClientIds, clientID)
}

// DeleteLastVisibleSet removes the last visible set for a socket.
func (c *Cache) DeleteLastVisibleSet(socketID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.LastVisibleSets, socketID)
}

// GetLastPositionAt returns last position timestamp for a client (by clientID). Used for 100ms cooldown.
func (c *Cache) GetLastPositionAt(clientID string) int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LastPositionAt[clientID]
}

// SetLastPositionAt sets last position timestamp for a client.
func (c *Cache) SetLastPositionAt(clientID string, ts int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.LastPositionAt == nil {
		c.LastPositionAt = make(map[string]int64)
	}
	c.LastPositionAt[clientID] = ts
}

// DeleteLastPositionAt removes last position timestamp for a client (by clientID).
func (c *Cache) DeleteLastPositionAt(clientID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.LastPositionAt, clientID)
}

// GetLastDbSaveAt returns last DB save timestamp for a user. Used for 30s throttle.
func (c *Cache) GetLastDbSaveAt(userID string) int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LastDbSaveAt[userID]
}

// SetLastDbSaveAt sets last DB save timestamp for a user.
func (c *Cache) SetLastDbSaveAt(userID string, ts int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.LastDbSaveAt == nil {
		c.LastDbSaveAt = make(map[string]int64)
	}
	c.LastDbSaveAt[userID] = ts
}

// DeleteLastDbSaveAt removes last DB save timestamp for a user.
func (c *Cache) DeleteLastDbSaveAt(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.LastDbSaveAt, userID)
}

// BuildExistingUsersPayload builds the existingUsers array for a viewer (under cache lock).
func (c *Cache) BuildExistingUsersPayload(viewerUserID string) []map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	visibleSet := c.getVisibleSetLocked(viewerUserID)
	seen := make(map[string]bool)
	var out []map[string]interface{}

	for _, u := range c.ActiveUsers {
		if u.UserID == viewerUserID {
			continue // Skip self
		}
		if !visibleSet[u.UserID] {
			continue
		}
		if seen[u.UserID] {
			continue
		}
		seen[u.UserID] = true
		m := c.sanitizeUserLocked(u)
		m["online"] = true
		out = append(out, m)
	}

	for _, entry := range c.OfflineUsers {
		if entry.User.UserID == viewerUserID {
			continue // Skip self
		}
		if !visibleSet[entry.User.UserID] || seen[entry.User.UserID] {
			continue
		}
		seen[entry.User.UserID] = true
		m := c.sanitizeUserLocked(entry.User)
		m["online"] = false
		m["offlineExpiresAt"] = entry.ExpiresAt
		out = append(out, m)
	}

	for uid, uc := range c.UsersCache {
		if uid == viewerUserID {
			continue // Skip self
		}
		if seen[uid] || !visibleSet[uid] || uc.LastLat == nil || uc.LastLng == nil {
			continue
		}
		seen[uid] = true
		m := map[string]interface{}{
			"socketId":     "stored-" + uid,
			"userId":       uid,
			"displayName":  c.getDisplayNameLocked(uid),
			"role":         uc.Role,
			"latitude":     uc.LastLat,
			"longitude":    uc.LastLng,
			"speed":        uc.LastSpeed,
			"lastUpdate":   uc.LastUpdate,
			"formattedTime": "",
			"sos":          map[string]interface{}{"active": false},
			"online":       false,
		}
		out = append(out, m)
	}
	return out
}

// getDisplayNameLocked assumes c.mu is held.
func (c *Cache) getDisplayNameLocked(userID string) string {
	u, ok := c.UsersCache[userID]
	if !ok {
		return "Unknown"
	}
	name := u.FirstName + " " + u.LastName
	if name == " " || strings.TrimSpace(name) == "" {
		return "Unknown"
	}
	return strings.TrimSpace(name)
}

// sanitizeUserLocked assumes c.mu is held. Returns a new map.
func (c *Cache) sanitizeUserLocked(user *ActiveUser) map[string]interface{} {
	locationLabel := c.inferLocationLabel(user.UserID, user.Latitude, user.Longitude)
	return map[string]interface{}{
		"socketId": user.SocketID, "userId": user.UserID, "displayName": user.DisplayName,
		"role": user.Role, "latitude": user.Latitude, "longitude": user.Longitude,
		"speed": user.Speed, "lastUpdate": user.LastUpdate, "formattedTime": user.FormattedTime,
		"batteryPct": user.BatteryPct, "deviceType": user.DeviceType, "connectionQuality": user.ConnectionQuality,
		"locationLabel": locationLabel,
		"sos": map[string]interface{}{"active": user.SOS.Active, "at": user.SOS.At, "reason": user.SOS.Reason, "type": user.SOS.Type},
		"geofence": map[string]interface{}{"enabled": user.Geofence.Enabled, "centerLat": user.Geofence.CenterLat, "centerLng": user.Geofence.CenterLng, "radiusM": user.Geofence.RadiusM},
		"autoSos": map[string]interface{}{"enabled": user.AutoSOS.Enabled, "noMoveMinutes": user.AutoSOS.NoMoveMinutes, "hardStopMinutes": user.AutoSOS.HardStopMin, "geofence": user.AutoSOS.Geofence},
		"checkIn": map[string]interface{}{"enabled": user.CheckIn.Enabled, "intervalMinutes": user.CheckIn.IntervalMin, "overdueMinutes": user.CheckIn.OverdueMin, "lastCheckInAt": user.CheckIn.LastCheckInAt},
		"retention": func() interface{} {
			if user.Retention != nil {
				return map[string]interface{}{"mode": user.Retention.Mode}
			}
			return map[string]interface{}{"mode": "default"}
		}(),
		"rooms":           user.Rooms,
		"motionClass":     user.MotionClass,
		"lastAttestAt":    user.LastAttestAt,
		"activityContext": user.ActivityContext,
	}
}

// GetShareCodeInfo returns shareCode, email, mobile for a user.
func (c *Cache) GetShareCodeInfo(userID string) (shareCode, email, mobile string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	u := c.UsersCache[userID]
	if u == nil {
		return "", "", ""
	}
	shareCode = u.ShareCode
	if u.Email != nil {
		email = *u.Email
	}
	if u.Mobile != nil {
		mobile = *u.Mobile
	}
	return shareCode, email, mobile
}

// AddUser adds a new user to the cache. Call after creating user in DB.
func (c *Cache) AddUser(userID string, entry *db.UserCacheEntry, shareCode string, email, mobile *string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry.LastAccessedAt = time.Now().UnixMilli()
	c.UsersCache[userID] = entry
	c.ShareCodes[shareCode] = userID
	if email != nil && *email != "" {
		c.EmailIndex[strings.ToLower(*email)] = userID
	}
	if mobile != nil && *mobile != "" {
		c.MobileIndex[*mobile] = userID
	}
}

// UpdateUserRole updates a user's role in the cache.
func (c *Cache) UpdateUserRole(userID, newRole string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if u := c.UsersCache[userID]; u != nil {
		u.Role = newRole
	}
}

// UpdateUserProfile updates profile fields in the cache after a successful DB write.
func (c *Cache) UpdateUserProfile(userID, firstName, lastName, email, mobile string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	u := c.UsersCache[userID]
	if u == nil {
		return
	}
	// Remove old email/mobile index entries
	if u.Email != nil && *u.Email != "" {
		delete(c.EmailIndex, strings.ToLower(*u.Email))
	}
	if u.Mobile != nil && *u.Mobile != "" {
		delete(c.MobileIndex, *u.Mobile)
	}
	u.FirstName = firstName
	u.LastName = lastName
	if email != "" {
		u.Email = &email
		c.EmailIndex[strings.ToLower(email)] = userID
	} else {
		u.Email = nil
	}
	if mobile != "" {
		u.Mobile = &mobile
		c.MobileIndex[mobile] = userID
	} else {
		u.Mobile = nil
	}
}

// CacheSize returns an estimated size of the cache in bytes.
// M-6: result is TTL-cached for 60 seconds — avoids holding a write lock while
// iterating all nested maps on every Prometheus scrape.
func (c *Cache) CacheSize() int64 {
	const ttl = 60 * time.Second

	// Fast path: return cached value under read lock if still fresh.
	c.mu.RLock()
	if time.Since(c.cachedSizeAt) < ttl {
		v := c.cachedSize
		c.mu.RUnlock()
		return v
	}
	c.mu.RUnlock()

	// Stale: acquire write lock, double-check, then recompute.
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Since(c.cachedSizeAt) < ttl {
		// Another goroutine refreshed while we waited for the write lock.
		return c.cachedSize
	}

	var size int64
	const ptrSize = 8

	mapOverhead := int64(48)
	entryOverhead := int64(24)

	// UsersCache
	size += mapOverhead + int64(len(c.UsersCache))*(entryOverhead+ptrSize*2+256)

	// ShareCodes, EmailIndex, MobileIndex (string maps)
	size += mapOverhead + int64(len(c.ShareCodes))*(entryOverhead+64)
	size += mapOverhead + int64(len(c.EmailIndex))*(entryOverhead+64)
	size += mapOverhead + int64(len(c.MobileIndex))*(entryOverhead+64)

	// Rooms
	size += mapOverhead + int64(len(c.Rooms))*(entryOverhead+ptrSize*2+256)

	// RoomMemberRoles (nested map)
	size += mapOverhead
	for _, roles := range c.RoomMemberRoles {
		size += mapOverhead + int64(len(roles))*(entryOverhead+ptrSize*2+128)
	}

	// Contacts (nested map)
	size += mapOverhead
	for _, contacts := range c.Contacts {
		size += mapOverhead + int64(len(contacts))*(entryOverhead+ptrSize)
	}

	// LiveTokens
	size += mapOverhead + int64(len(c.LiveTokens))*(entryOverhead+ptrSize*2+128)

	// Guardianships (nested map)
	size += mapOverhead
	for _, gships := range c.Guardianships {
		size += mapOverhead + int64(len(gships))*(entryOverhead+ptrSize*2+128)
	}

	// SavedPlaces
	size += mapOverhead + int64(len(c.SavedPlaces))*(entryOverhead+ptrSize*2+256)

	// ProximityAlerts
	size += mapOverhead + int64(len(c.ProximityAlerts))*(entryOverhead+ptrSize*2+128)

	// ActiveUsers (largest contributor)
	size += mapOverhead
	for _, user := range c.ActiveUsers {
		userSize := int64(ptrSize * 20)
		userSize += int64(len(user.SocketID) + len(user.UserID) + len(user.DisplayName) + len(user.Role))
		if user.Retention != nil {
			userSize += int64(len(user.Retention.ClientID))
		}
		if user.Rooms != nil {
			for _, room := range user.Rooms {
				userSize += int64(len(room))
			}
		}
		size += entryOverhead + userSize
	}

	// OfflineUsers
	size += mapOverhead + int64(len(c.OfflineUsers))*(entryOverhead+ptrSize*2+512)

	// Other maps
	size += mapOverhead + int64(len(c.VisibilityCache))*(entryOverhead+ptrSize*2+256)
	size += mapOverhead + int64(len(c.LastVisibleSets))*(entryOverhead+ptrSize*2+256)
	size += mapOverhead + int64(len(c.LastPositionAt))*(entryOverhead+64)
	size += mapOverhead + int64(len(c.LastDbSaveAt))*(entryOverhead+64)
	size += mapOverhead + int64(len(c.PendingRequests))*(entryOverhead+ptrSize*2+256)
	size += mapOverhead + int64(len(c.UserIdToSocketId))*(entryOverhead+64)

	// LiveTokensByUser (nested map)
	size += mapOverhead
	for _, tokens := range c.LiveTokensByUser {
		size += mapOverhead + int64(len(tokens))*(entryOverhead+ptrSize)
	}

	// UserRooms (nested map)
	size += mapOverhead
	for _, rooms := range c.UserRooms {
		size += mapOverhead + int64(len(rooms))*(entryOverhead+ptrSize)
	}

	size += mapOverhead + int64(len(c.AdminClientIds))*(entryOverhead+ptrSize)

	c.cachedSize = size
	c.cachedSizeAt = time.Now()
	return size
}

const maxPushSubsPerUser = 10

// AddPushSubscription stores a Web Push subscription for the user (deduped by endpoint).
// Enforces a cap of maxPushSubsPerUser endpoints per user via FIFO eviction.
func (c *Cache) AddPushSubscription(userID, endpoint, p256dh, auth string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	subs := c.PushSubs[userID]
	for i, s := range subs {
		if s.Endpoint == endpoint {
			subs[i] = PushSubscription{Endpoint: endpoint, P256dh: p256dh, Auth: auth}
			c.PushSubs[userID] = subs
			return
		}
	}
	// Enforce per-user cap: drop the oldest entry (index 0) before appending.
	if len(subs) >= maxPushSubsPerUser {
		subs = subs[1:]
	}
	c.PushSubs[userID] = append(subs, PushSubscription{Endpoint: endpoint, P256dh: p256dh, Auth: auth})
}

// RemovePushSubscription removes a Web Push subscription by endpoint.
func (c *Cache) RemovePushSubscription(userID, endpoint string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	subs := c.PushSubs[userID]
	filtered := subs[:0]
	for _, s := range subs {
		if s.Endpoint != endpoint {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		delete(c.PushSubs, userID)
	} else {
		c.PushSubs[userID] = filtered
	}
}

// GetPushSubscriptions returns all Web Push subscriptions for the user.
func (c *Cache) GetPushSubscriptions(userID string) []PushSubscription {
	c.mu.RLock()
	defer c.mu.RUnlock()
	subs := c.PushSubs[userID]
	if len(subs) == 0 {
		return nil
	}
	out := make([]PushSubscription, len(subs))
	copy(out, subs)
	return out
}

// AddSecretChatInvite stores a new invite token in the ephemeral cache.
func (c *Cache) AddSecretChatInvite(token, ownerID, peerID string, expiresAt int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SecretChatInvites[token] = &SecretChatInvite{
		Token:     token,
		OwnerID:   ownerID,
		PeerID:    peerID,
		ExpiresAt: expiresAt,
	}
}

// GetSecretChatInvite returns the invite for the given token, or nil.
func (c *Cache) GetSecretChatInvite(token string) *SecretChatInvite {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SecretChatInvites[token]
}

// DeleteSecretChatInvite removes an invite token.
func (c *Cache) DeleteSecretChatInvite(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.SecretChatInvites, token)
}

// ApplyCheckInUpdate sets check-in timestamps on an active user under a write lock.
// Parameters with value 0 are skipped ("don't update" convention used by cleanup.go).
func (c *Cache) ApplyCheckInUpdate(socketID string, setLastCheckInAt, setRequestedAt, setMissedNotifyAt int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	user, ok := c.ActiveUsers[socketID]
	if !ok {
		return
	}
	if setLastCheckInAt != 0 {
		user.CheckIn.LastCheckInAt = setLastCheckInAt
	}
	if setRequestedAt != 0 {
		user.CheckIn.RequestedAt = setRequestedAt
	}
	if setMissedNotifyAt != 0 {
		user.CheckIn.MissedNotifiedAt = setMissedNotifyAt
	}
}

// ApplyGentleAlertSentAt sets GentleAlertSentAt on an active user under a write lock.
func (c *Cache) ApplyGentleAlertSentAt(socketID string, sentAt int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	user, ok := c.ActiveUsers[socketID]
	if !ok {
		return
	}
	user.GentleAlertSentAt = sentAt
}

// CollectExpiredSecretChatInvites removes and returns all expired invite tokens.
func (c *Cache) CollectExpiredSecretChatInvites(now int64) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []string
	for token, inv := range c.SecretChatInvites {
		if inv.ExpiresAt != 0 && now > inv.ExpiresAt {
			out = append(out, token)
			delete(c.SecretChatInvites, token)
		}
	}
	return out
}
