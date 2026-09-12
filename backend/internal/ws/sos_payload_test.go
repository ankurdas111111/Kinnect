package ws

import (
	"testing"

	"kinnect-v3/internal/cache"
)

func sosUser() *cache.ActiveUser {
	lat, lng := 12.9716, 77.5946
	acc := 18.0
	u := &cache.ActiveUser{
		SocketID: "sock-meera", UserID: "meera", DisplayName: "Meera",
		Latitude: &lat, Longitude: &lng, Accuracy: &acc, LastUpdate: 1789168018000,
	}
	reason, sosType, at := "SOS", "manual", int64(1789168018000)
	u.SOS = cache.SOS{Active: true, Reason: &reason, Type: &sosType, At: &at}
	return u
}

// "Where is she" is the first question a contact asks. The alert has to answer
// it without depending on state the app may not have — the reader is often
// opening cold from a push notification.
func TestPublicSosCarriesLocationForContacts(t *testing.T) {
	h := testHub(cache.New())
	got := h.publicSos(sosUser())

	for _, k := range []string{"latitude", "longitude", "locationLabel", "lastUpdate", "accuracy"} {
		if _, present := got[k]; !present {
			t.Errorf("SOS payload to contacts is missing %q; the alert cannot say where they are", k)
		}
	}
	if got["latitude"] == nil {
		t.Error("latitude is nil on a user who has a fix")
	}
}

// Contacts exchange no phone numbers today. An SOS must not be the event that
// permanently widens that, when the in-app call path already reaches them.
func TestPublicSosDoesNotCarryPhoneNumber(t *testing.T) {
	h := testHub(cache.New())
	got := h.publicSos(sosUser())
	for _, k := range []string{"mobile", "phone", "email"} {
		if _, present := got[k]; present {
			t.Errorf("SOS payload leaks %q to every contact", k)
		}
	}
}

// Anonymous watch-link holders get the sos block only. Adding a field to
// publicSos must never reach them through this path.
func TestViewerSosStaysNarrowWhenPublicSosGrows(t *testing.T) {
	h := testHub(cache.New())
	got := h.viewerSos(sosUser())

	for _, k := range []string{"socketId", "userId", "latitude", "longitude", "locationLabel"} {
		if _, present := got[k]; present {
			t.Errorf("viewer SOS block leaks %q", k)
		}
	}
	if got["active"] != true {
		t.Error("viewer SOS block lost its active flag")
	}
}
