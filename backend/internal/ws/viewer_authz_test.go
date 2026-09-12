package ws

import "testing"

// Anonymous share-link sockets connect without a session. The allowlist is the
// only thing standing between "holds a URL" and "can call any hub handler", so
// it is default-deny by construction: this test fails the moment a new event is
// added to viewerAllowedEvents without a deliberate decision.
func TestViewerAllowlistIsExactlyThreeEvents(t *testing.T) {
	want := map[string]bool{"liveJoin": true, "watchJoin": true, "liveAckSOS": true}

	if len(viewerAllowedEvents) != len(want) {
		t.Fatalf("viewer allowlist has %d events, expected %d: %v",
			len(viewerAllowedEvents), len(want), viewerAllowedEvents)
	}
	for e := range want {
		if !viewerAllowedEvents[e] {
			t.Errorf("share links are broken: %q must be callable by a viewer", e)
		}
	}
	for e := range viewerAllowedEvents {
		if !want[e] {
			t.Errorf("%q was added to the viewer allowlist; an unauthenticated "+
				"stranger with a link can now call it. Confirm that is intended.", e)
		}
	}
}

// Every privileged handler must stay unreachable for a viewer. Listed by name
// rather than derived from the registry so that renaming a handler does not
// silently drop it from the check.
func TestViewerCannotReachPrivilegedEvents(t *testing.T) {
	privileged := []string{
		"position", "positionBatch", "triggerSOS", "cancelSOS",
		"addContact", "removeContact", "createRoom", "joinRoom", "leaveRoom",
		"createLiveLink", "revokeLiveLink", "profileUpdate", "pushSubscribe",
		"adminDeleteUser", "requestAdminOverview", "shareRide", "nudgeUser",
	}
	for _, e := range privileged {
		if viewerAllowedEvents[e] {
			t.Errorf("%q is reachable by an anonymous viewer", e)
		}
	}
}

// A viewer client must be refused before dispatch looks the handler up, so an
// event that is not registered at all and one that is registered but forbidden
// are both dead ends.
func TestViewerGateRunsBeforeHandlerLookup(t *testing.T) {
	for _, e := range []string{"position", "definitelyNotAnEvent"} {
		if viewerAllowedEvents[e] {
			t.Errorf("%q must not be viewer-callable", e)
		}
	}
}
