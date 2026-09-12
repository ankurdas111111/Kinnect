package ws

import (
	"sort"
	"testing"

	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/config"
	"kinnect-v3/internal/db"
)

func testHub(c *cache.Cache) *Hub {
	// Pool is only touched by DB-writing handlers; recipient derivation reads
	// the cache alone.
	return NewHub(c, nil, &config.Config{})
}

// The point of SOS push: it must reach contacts whose phone is asleep. A
// socket-based fan-out reaches only people with the app open, which at 11pm is
// nobody. Recipients are therefore derived by USER id, and this test pins that.
func TestSosPushReachesOfflineContacts(t *testing.T) {
	c := cache.New()
	c.Contacts["meera"] = map[string]bool{"arjun": true, "nani": true}

	// Only Arjun holds a live socket. Nani's phone is locked.
	c.SetActiveUser("sock-arjun", &cache.ActiveUser{SocketID: "sock-arjun", UserID: "arjun", Online: true})

	h := testHub(c)
	got := h.sosPushRecipients(&cache.ActiveUser{SocketID: "sock-meera", UserID: "meera", DisplayName: "Meera"})
	sort.Strings(got)

	want := []string{"arjun", "nani"}
	if len(got) != len(want) {
		t.Fatalf("recipients = %v, want %v (an offline contact was dropped)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("recipients = %v, want %v", got, want)
		}
	}
}

// Pushing the alert back to the person who raised it is noise at the worst
// possible moment.
func TestSosPushExcludesSender(t *testing.T) {
	c := cache.New()
	c.Contacts["meera"] = map[string]bool{"arjun": true}

	h := testHub(c)
	for _, uid := range h.sosPushRecipients(&cache.ActiveUser{UserID: "meera"}) {
		if uid == "meera" {
			t.Error("SOS push includes the sender")
		}
	}
}

// Family membership grants visibility, so it must also grant the alert.
func TestSosPushIncludesRoomMembers(t *testing.T) {
	c := cache.New()
	c.Rooms["FAM1"] = &db.RoomEntry{
		Name:    "The Nairs",
		Members: map[string]bool{"meera": true, "arjun": true, "nani": true},
	}
	c.UserRooms["meera"] = map[string]bool{"FAM1": true}

	h := testHub(c)
	got := h.sosPushRecipients(&cache.ActiveUser{UserID: "meera"})
	if len(got) != 2 {
		t.Fatalf("recipients = %v, want the 2 other family members", got)
	}
}

// A solo account raising an SOS must not panic or fan out to strangers.
func TestSosPushWithNoContactsIsEmpty(t *testing.T) {
	h := testHub(cache.New())
	if got := h.sosPushRecipients(&cache.ActiveUser{UserID: "alone"}); len(got) != 0 {
		t.Fatalf("recipients = %v, want none", got)
	}
}
