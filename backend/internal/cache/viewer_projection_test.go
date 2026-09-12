package cache

import "testing"

// A share link can be forwarded to anyone. SanitizeUser is the full internal
// record and carries the sharer's home geofence, auto-SOS thresholds, check-in
// schedule and internal ids; SanitizeUserForViewer must carry none of it.
// This test is the guard: adding a field to SanitizeUser must never widen what
// a stranger with a URL receives.
func TestSanitizeUserForViewerWithholdsPrivateFields(t *testing.T) {
	home := 12.9716
	homeLng := 77.5946
	c := New()

	u := &ActiveUser{
		SocketID:    "sock-abc",
		UserID:      "user-123",
		DisplayName: "Meera",
		Role:        "user",
		Latitude:    &home,
		Longitude:   &homeLng,
		Online:      true,
		Rooms:       []string{"room-1"},
	}
	u.Geofence = Geofence{Enabled: true, CenterLat: &home, CenterLng: &homeLng, RadiusM: 150}
	u.CheckIn = CheckIn{Enabled: true, IntervalMin: 30, OverdueMin: 7}
	u.AutoSOS = AutoSOS{Enabled: true, NoMoveMinutes: 20}

	got := c.SanitizeUserForViewer(u)

	// Anything that could locate the sharer's home or expose account internals.
	forbidden := []string{
		"geofence", "autoSos", "checkIn", "quietHours",
		"userId", "socketId", "role", "rooms",
		"safetyScore", "locationLabel", "statusMessage", "retention",
	}
	for _, k := range forbidden {
		if _, present := got[k]; present {
			t.Errorf("viewer payload leaks %q; link recipients are unauthenticated strangers", k)
		}
	}

	// The fields the viewer pages actually render must survive.
	for _, k := range []string{"displayName", "latitude", "longitude", "speed", "online", "sos"} {
		if _, present := got[k]; !present {
			t.Errorf("viewer payload is missing %q, which the live/watch pages render", k)
		}
	}

	if got["displayName"] != "Meera" {
		t.Errorf("displayName = %v, want Meera", got["displayName"])
	}
}

// The full projection is still expected to carry the private fields; if this
// ever stops being true the test above is passing for the wrong reason.
func TestSanitizeUserStillCarriesPrivateFieldsForContacts(t *testing.T) {
	c := New()
	u := &ActiveUser{SocketID: "s", UserID: "u", DisplayName: "Meera"}
	full := c.SanitizeUser(u)
	for _, k := range []string{"geofence", "checkIn", "userId"} {
		if _, present := full[k]; !present {
			t.Errorf("SanitizeUser no longer carries %q; the viewer test is now vacuous", k)
		}
	}
}
