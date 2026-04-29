package db

// UserCacheEntry holds cached user profile data.
type UserCacheEntry struct {
	FirstName      string
	LastName       string
	Role           string
	ShareCode      string
	Email          *string
	Mobile         *string
	CreatedAt      int64
	LastLat        *float64
	LastLng        *float64
	LastSpeed      *string
	LastUpdate     *int64
	LastAccessedAt int64 // unix ms — updated on every cache read; used by EvictLRU
}

// RoomEntry holds cached room data.
type RoomEntry struct {
	DbID      string
	Name      string
	Members   map[string]bool // userId -> true
	CreatedBy string
	CreatedAt int64

	// F3: meeting point
	MeetingLat   *float64
	MeetingLng   *float64
	MeetingLabel string
	MeetingSetBy string // userID
	MeetingSetAt int64
}

// RoomMemberRole holds role and expiry for a room member.
type RoomMemberRole struct {
	Role     string
	ExpiresAt *int64
}

// LiveTokenEntry holds live token data.
type LiveTokenEntry struct {
	UserID    string
	ExpiresAt *int64
	CreatedAt int64
}

// GuardianshipEntry holds guardianship data.
type GuardianshipEntry struct {
	Status      string
	InitiatedBy string
	ExpiresAt   *int64
	CreatedAt   int64
}

// RoomAdminRequestEntry holds room admin request with votes.
type RoomAdminRequestEntry struct {
	Type      string // "roomAdmin"
	From      string
	RoomCode  string
	ExpiresIn *string
	CreatedAt int64
	Approvals map[string]bool // voterId -> true
	Denials   map[string]bool // voterId -> true
}

// UserSettings holds persisted per-user settings loaded on connection.
type UserSettings struct {
	QuietHoursEnabled  bool
	QuietHoursStart    string
	QuietHoursEnd      string
	HeartbeatEnabled   bool
	HeartbeatDeadline  string
	HeartbeatLastSignal int64
	EmergencyPhone1    string
	EmergencyPhone2    string
	SpeedAlertThresholdMs float64 // F5: 0 = disabled
}

// SavedPlaceEntry holds a saved place for a user (F4).
type SavedPlaceEntry struct {
	ID        string
	UserID    string
	Name      string
	Icon      string
	Latitude  float64
	Longitude float64
	RadiusM   float64
	CreatedAt int64
}

// ProximityAlertEntry holds a proximity alert configuration (F7).
type ProximityAlertEntry struct {
	ID              string
	OwnerID         string
	TargetID        string
	RadiusM         int
	Enabled         bool
	LastTriggeredAt int64  // unix ms; mutated at runtime, not reloaded from DB
	CreatedAt       int64
}

// GeofenceEventRow holds a single geofence entry/exit event (F6).
type GeofenceEventRow struct {
	ID        string
	FenceName string
	EventType string // "entry" | "exit"
	Lat       float64
	Lng       float64
	Ts        int64
}

// RoomNoteRow holds a room bulletin board note (F8).
type RoomNoteRow struct {
	ID         string
	RoomID     string
	AuthorID   string
	AuthorName string // JOIN with users
	Body       string
	CreatedAt  int64
}

// DailyActivityRow holds daily activity summary data (F9).
type DailyActivityRow struct {
	Date          string // "YYYY-MM-DD"
	DistanceM     int
	ActiveMinutes int
	UpdatedAt     int64
}

// LoadAllResult contains all data loaded from DB for cache init.
type LoadAllResult struct {
	UsersCache        map[string]*UserCacheEntry
	ShareCodes        map[string]string
	EmailIndex        map[string]string
	MobileIndex       map[string]string
	Rooms             map[string]*RoomEntry
	RoomMemberRoles   map[string]map[string]*RoomMemberRole // roomCode -> userId -> role
	Contacts          map[string]map[string]bool            // ownerId -> contactId -> true
	LiveTokens        map[string]*LiveTokenEntry
	Guardianships     map[string]map[string]*GuardianshipEntry // guardianId -> wardId -> entry
	RoomAdminRequests map[string][]*RoomAdminRequestEntry      // key (roomCode:roomAdmin) -> requests
	SavedPlaces       map[string][]SavedPlaceEntry             // userID -> []entry (F4)
	ProximityAlerts   map[string][]*ProximityAlertEntry        // targetUserID -> []alert (F7)
}
