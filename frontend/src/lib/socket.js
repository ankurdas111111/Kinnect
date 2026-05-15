import { get } from 'svelte/store';
import { otherUsers, mySocketId, myLocation, mySafetyStatus } from './stores/map.js';
import { myRooms, myShareCode, myContactInfo, roomNotes } from './stores/rooms.js';
import { myContacts } from './stores/contacts.js';
import { myGuardianData, canManage, pendingIncomingRequests } from './stores/guardians.js';
import { banner, alertState, myLiveLinks, mySosActive, sosNarratives, activeSosUsers, geofenceShake } from './stores/sos.js';
import { haptics } from './haptics.js';
import { adminOverview } from './stores/admin.js';
import { authUser } from './stores/auth.js';
import { drainBuffer, hasBuffered } from './offlineBuffer.js';
import { pulseMap } from './stores/pulses.js';
import { networkGraph } from './stores/network.js';
import { recordLatency } from './stores/latency.js';
import { createRealtimeSocket } from './realtimeClient.js';
import { notifySOS, notifyProximitySOS, notifyGuardianRequest, notifyBatteryLow, notifyHaventMoved } from './nativeNotifications.js';
import { rideShare } from './stores/rideShare.js';
import { crowdMode } from './stores/crowdMode.js';
import { bumpHubBadge } from './stores/hubBadge.js';
import { addSecretMessage, setSecretMessages, removeSecretMessage, updateSecretMessageSeen, secretChatPresence } from './stores/secretChat.js';
import { getShareOrigin } from './env.js';
import { geofenceLog, proximityAlerts } from './stores/places.js';
import { incomingOffer } from './stores/webrtc.js';
import { dailyActivity } from './stores/activity.js';
import { recentTrailResult } from './stores/trail.js';
import { arrivalProjections } from './stores/arrivals.js';

const storedClientId = localStorage.getItem('clientId');
const clientId = storedClientId || (crypto && crypto.randomUUID ? crypto.randomUUID() : String(Date.now()) + '-' + Math.random().toString(16).slice(2));
if (!storedClientId) localStorage.setItem('clientId', clientId);

export const socket = createRealtimeSocket({
  auth: { clientId },
  autoConnect: false,
  reconnection: true,
  reconnectionAttempts: 50,
  reconnectionDelay: 1000,
  reconnectionDelayMax: 15000,
  randomizationFactor: 0.5,
  timeout: 8000
});

let connected = false;
let handlersRegistered = false;

// ── Fix F: single cancellable banner timer to prevent timer pile-up ──────────
let _bannerClearTimer = null;
// KR-003: exported so MainApp.svelte (and any other module) can route all banner
// mutations through the same timer-managed path instead of calling banner.set() directly.
export function setBanner(b, autoClearMs) {
  if (_bannerClearTimer) { clearTimeout(_bannerClearTimer); _bannerClearTimer = null; }
  banner.set(b);
  if (autoClearMs) {
    _bannerClearTimer = setTimeout(() => {
      _bannerClearTimer = null;
      banner.set({ type: null, text: null, actions: [] });
    }, autoClearMs);
  }
}

// ── Module-scope reconnect banner timer — must be accessible from appStateChange ─
let _reconnectBannerTimer = null;

/**
 * Cancel the pending "Reconnecting…" banner timer and clear the banner if it
 * is currently showing the reconnecting message. Called from:
 *  - appStateChange (isActive=true) so foreground resumes are always silent
 *  - window.online handler so network-back events don't race with reconnect
 */
export function cancelReconnectBanner() {
  if (_reconnectBannerTimer) { clearTimeout(_reconnectBannerTimer); _reconnectBannerTimer = null; }
  // Use type field (not text string) to identify the reconnect banner so this
  // check survives any future copy/i18n changes to the displayed message.
  banner.update(b => (b && b.type === 'reconnecting') ? { type: null, text: null, actions: [] } : b);
}

/**
 * Reset client-side user state. Call this on explicit logout so that a
 * re-login as a different user doesn't briefly display the previous user's
 * contacts before the server sends a fresh existingUsers event.
 */
export function resetSocketState() {
  _localMap = new Map();
  otherUsers.set(_localMap);
  pulseMap.set(new Map());
  sosNarratives.set(new Map());
  activeSosUsers.set(new Map());
  networkGraph.set(new Map());
  myRooms.set([]);
  myContacts.set([]);
  myLiveLinks.set([]);
  mySosActive.set(false);
  banner.set({ type: null, text: null, actions: [] });
  drainBuffer(); // flush stale buffered positions
}

// ── Fix J: module-level Map + microtask batching for otherUsers store ─────────
// Batches all same-tick socket events into a single Svelte store notification,
// preventing O(n) store updates when many users report positions simultaneously.
let _localMap = new Map();
let _dirtyUsers = false;

function _scheduleUsersFlush() {
  if (_dirtyUsers) return;
  _dirtyUsers = true;
  Promise.resolve().then(() => {
    _dirtyUsers = false;
    otherUsers.set(_localMap);
  });
}

export function setupSocketHandlers() {
  if (handlersRegistered) {
    if (!socket.connected) socket.connect();
    return;
  }
  handlersRegistered = true;

  socket.on('connect', async () => {
    connected = true;
    cancelReconnectBanner();
    setBanner({ type: null, text: null, actions: [] });
  });

  // Server tells us our assigned socket ID — use this for self-filtering
  socket.on('welcome', (data) => {
    if (data?.socketId) mySocketId.set(data.socketId);
  });

  socket.on('disconnect', (reason) => {
    connected = false;
    // Explicit client-initiated disconnect (e.g. logout) — clear stale user data
    // immediately so a re-login as a different user doesn't briefly see the previous
    // user's contacts before the server sends a fresh existingUsers event.
    if (reason === 'io client disconnect') {
      _localMap = new Map();
      otherUsers.set(_localMap);
    }
    // Start the 10s countdown. Each reconnect_attempt resets it so the banner only
    // appears if a full 10s elapses without a new attempt — i.e. we're genuinely stuck.
    // appStateChange (foreground) and window.online both call cancelReconnectBanner()
    // before calling socket.connect(), so normal background/resume cycles are silent.
    if (_reconnectBannerTimer) { clearTimeout(_reconnectBannerTimer); _reconnectBannerTimer = null; }
    _reconnectBannerTimer = setTimeout(() => {
      _reconnectBannerTimer = null;
      if (!connected) setBanner({ type: 'reconnecting', text: 'Trying to reconnect...', actions: [] });
    }, 10000);
    // Auth errors need immediate feedback — skip the timer
    if (reason === 'io server disconnect') {
      cancelReconnectBanner();
    }
  });

  socket.on('connect_error', (err) => {
    const msg = err && err.message ? err.message : '';
    if (msg.includes('Authentication') || msg.includes('session') || msg.includes('401') || msg.includes('403')) {
      cancelReconnectBanner();
      setBanner({ type: 'sos', text: 'Your session has ended. Signing you in again...', actions: [] });
      setTimeout(() => { window.location.hash = '#/login'; }, 2000);
    }
    // Other errors: silently retry — banner is already scheduled from disconnect
  });

  socket.io.on('reconnect', () => {
    // Drain offline buffer IMMEDIATELY before any new position emits
    if (hasBuffered()) {
      const batch = drainBuffer();
      if (batch.length > 0 && socket.connected) socket.emit('positionBatch', batch);
    }
  });

  // Reset the 10s banner timer on each reconnect attempt so "Reconnecting…" only
  // appears when we've been genuinely stuck without any new attempt for 10 seconds.
  socket.io.on('reconnect_attempt', () => {
    if (_reconnectBannerTimer) { clearTimeout(_reconnectBannerTimer); _reconnectBannerTimer = null; }
    _reconnectBannerTimer = setTimeout(() => {
      _reconnectBannerTimer = null;
      if (!connected) setBanner({ type: 'reconnecting', text: 'Trying to reconnect...', actions: [] });
    }, 10000);
  });

  socket.io.on('reconnect_failed', () => {
    cancelReconnectBanner();
    setBanner({ type: 'sos', text: "Can't reach Kinnect right now. Try refreshing.", actions: [
      { label: 'Refresh', kind: 'btn-primary', onClick: () => window.location.reload() }
    ] });
  });

  // ── User data events ────────────────────────────────────────────────────────
  function extractSafety(u) {
    if (!u) return;
    mySafetyStatus.set({
      geofence: u.geofence || { enabled: false },
      autoSos: u.autoSos || { enabled: false },
      checkIn: u.checkIn || { enabled: false }
    });
  }

  socket.on('existingUsers', (users) => {
    _localMap = new Map();
    const sid = get(mySocketId);
    (users || []).forEach(u => {
      if (u.socketId === sid) { extractSafety(u); return; }
      _localMap.set(u.socketId, u);
    });
    otherUsers.set(_localMap); // full replacement — notify immediately
  });

  socket.on('userConnected', (user) => {
    const mySid = get(mySocketId);
    if (!mySid || user.socketId === mySid) return;
    _localMap.set(user.socketId, user);
    _scheduleUsersFlush();
  });

  socket.on('userUpdate', (user) => {
    const mySid = get(mySocketId);
    if (!mySid || user.socketId === mySid) {
      if (mySid && user.socketId === mySid) extractSafety(user);
      return;
    }
    if (user.timestamp) recordLatency(user.timestamp, user.serverTs);
    _localMap.set(user.socketId, user);
    _scheduleUsersFlush();
  });

  socket.on('userDisconnect', (socketId) => {
    _localMap.delete(socketId);
    _scheduleUsersFlush();
  });

  socket.on('userOffline', (user) => {
    if (!user || user.socketId === get(mySocketId)) return;
    _localMap.set(user.socketId, user);
    _scheduleUsersFlush();
  });

  socket.on('visibilityRefresh', (users) => {
    _localMap = new Map();
    const sid = get(mySocketId);
    (users || []).forEach(u => {
      if (u.socketId === sid) { extractSafety(u); return; }
      _localMap.set(u.socketId, u);
    });
    otherUsers.set(_localMap); // full replacement — notify immediately
  });

  // Share code + personal info
  socket.on('myShareCode', (data) => {
    if (data && data.shareCode) myShareCode.set(data.shareCode);
    if (data) myContactInfo.set({ email: data.email || '', mobile: data.mobile || '' });
  });

  // Rooms
  socket.on('myRooms', (data) => myRooms.set(data || []));

  // Contacts
  socket.on('myContacts', (data) => myContacts.set(data || []));

  // Live links
  socket.on('myLiveLinks', (links) => myLiveLinks.set(links || []));

  // Guardians
  socket.on('myGuardians', (data) => {
    if (!data) return;
    myGuardianData.set({
      asGuardian: data.asGuardian ?? [],
      asWard: data.asWard ?? [],
      manageable: data.manageable ?? []
    });
    const cm = new Map();
    (data.manageable || []).forEach(m => cm.set(m.userId, m.displayName));
    canManage.set(cm);
  });

  // Persistent pending requests (sent on connect, survives reconnects)
  socket.on('pendingRequests', (data) => {
    if (!Array.isArray(data)) return;
    pendingIncomingRequests.set(data);
  });

  // Guardian/admin request events (real-time notifications)
  socket.on('roomAdminRequest', (data) => {
    if (!data) return;
    pendingIncomingRequests.update(arr => {
      if (arr.some(r => r.type === 'roomAdmin' && r.from === data.fromUserId && r.roomCode === data.roomCode)) return arr;
      return [...arr, { type: 'roomAdmin', from: data.fromUserId, fromName: data.fromName, roomCode: data.roomCode, expiresIn: data.expiresIn, approvals: data.approvals || 0, denials: data.denials || 0, totalEligible: data.totalEligible || 0, myVote: null }];
    });
    setBanner({ type: 'info', text: data.fromName + ' requested Room Admin in ' + data.roomCode + ' — Vote now!', actions: [] }, 3000);
  });

  socket.on('roomAdminVoteUpdate', (data) => {
    if (!data) return;
    if (data.denied) {
      pendingIncomingRequests.update(arr => arr.filter(r => !(r.type === 'roomAdmin' && r.from === data.userId && r.roomCode === data.roomCode)));
      return;
    }
    pendingIncomingRequests.update(arr => arr.map(r => {
      if (r.type === 'roomAdmin' && r.from === data.userId && r.roomCode === data.roomCode) {
        return { ...r, approvals: data.approvals, denials: data.denials, totalEligible: data.totalEligible, myVote: data.myVote };
      }
      return r;
    }));
  });

  // KR-002: guardianInfo is the success notification for requestGuardian / inviteGuardian.
  // Previously these used contactError which showed as a red error banner.
  socket.on('guardianInfo', (data) => {
    if (!data) return;
    setBanner({ type: 'info', text: data.message || 'Guardian request sent', actions: [] }, 3000);
  });

  socket.on('guardianRequest', (data) => {
    if (!data) return;
    pendingIncomingRequests.update(arr => {
      if (arr.some(r => r.type === 'guardian' && r.from === data.fromUserId)) return arr;
      return [...arr, { type: 'guardian', from: data.fromUserId, fromName: data.fromName, expiresIn: data.expiresIn }];
    });
    notifyGuardianRequest(data.fromName).catch(() => {});
    setBanner({ type: 'info', text: data.fromName + ' wants to be your guardian', actions: [] }, 3000);
  });

  socket.on('guardianInvite', (data) => {
    if (!data) return;
    pendingIncomingRequests.update(arr => {
      if (arr.some(r => r.type === 'guardianInvite' && r.from === data.fromUserId)) return arr;
      return [...arr, { type: 'guardianInvite', from: data.fromUserId, fromName: data.fromName, expiresIn: data.expiresIn }];
    });
    setBanner({ type: 'info', text: data.fromName + ' wants you to be their guardian', actions: [] }, 3000);
  });

  socket.on('roomAdminUpdated', (data) => {
    if (!data) return;
    if (data.role === 'admin' || data.role === 'denied') {
      pendingIncomingRequests.update(arr => arr.filter(r => !(r.type === 'roomAdmin' && r.from === data.userId && r.roomCode === data.roomCode)));
    }
    setBanner({ type: 'info', text: 'Room admin role updated in ' + data.roomCode, actions: [] }, 2000);
  });

  socket.on('guardianUpdated', (data) => {
    if (!data) return;
    if (data.status === 'active' || data.status === 'denied' || data.status === 'revoked' || data.status === 'expired') {
      pendingIncomingRequests.update(arr => arr.filter(r => {
        if (r.type === 'guardian' && r.from === data.guardianId) return false;
        if (r.type === 'guardianInvite' && r.from === data.wardId) return false;
        return true;
      }));
    }
    const statusMsg = data.status === 'active' ? 'accepted' : data.status === 'denied' ? 'declined' : data.status === 'revoked' ? 'ended' : data.status === 'expired' ? 'expired' : data.status;
    setBanner({ type: 'info', text: 'Guardian request ' + statusMsg, actions: [] }, 2000);
  });

  // Room/contact action results
  socket.on('roomError', (data) => {
    setBanner({ type: 'info', text: data?.message || 'Something went wrong with this room', actions: [] }, 2500);
  });
  socket.on('contactError', (data) => {
    setBanner({ type: 'info', text: data?.message || 'Could not update this contact', actions: [] }, 2500);
  });
  socket.on('roomCreated', (data) => {
    setBanner({ type: 'info', text: `"${data.name}" created — share code ${data.code} with family`, actions: [] }, 4000);
  });
  socket.on('roomJoined', (data) => {
    setBanner({ type: 'info', text: `You joined "${data.name}"`, actions: [] }, 2000);
    bumpHubBadge(false);
  });
  socket.on('roomLeft', (data) => {
    setBanner({ type: 'info', text: `You left "${data?.name || 'the room'}"`, actions: [] }, 2000);
  });
  socket.on('contactAdded', (data) => {
    setBanner({ type: 'info', text: `${data?.displayName || 'Contact'} added to your family`, actions: [] }, 2000);
    bumpHubBadge(false);
  });
  socket.on('contactRemoved', () => {
    setBanner({ type: 'info', text: 'Contact removed from your list', actions: [] }, 2000);
  });
  socket.on('liveLinkCreated', (data) => {
    const url = getShareOrigin() + '/#/live/' + data.token;
    navigator.clipboard.writeText(url).catch(() => {
      setBanner({ type: 'info', text: 'Share this link: ' + url, actions: [] }, 10000);
      return;
    });
    setBanner({ type: 'info', text: 'Live link copied — share it with anyone', actions: [] }, 2500);
  });

  // SOS events (persistent banners — no auto-clear)
  // Pulse Check-In — ephemeral heartbeat from a contact
  socket.on('pulseReceived', (data) => {
    if (!data || !data.userId) return;
    pulseMap.update(m => {
      const nm = new Map(m);
      nm.set(data.userId, data);
      return nm;
    });
    // Auto-expire after 30s
    setTimeout(() => {
      pulseMap.update(m => { const nm = new Map(m); nm.delete(data.userId); return nm; });
    }, 30000);
    if (data.type === 'ok') {
      setBanner({ type: 'info', text: `${data.displayName || 'Someone'} says they're okay`, actions: [] }, 5000);
    } else if (data.type === 'callme') {
      setBanner({ type: 'sos', text: `${data.displayName || 'Someone'} is asking you to call them`, actions: [
        { label: 'Got it', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
      ] }, 8000);
    }
  });

  // Secret chat — ciphertext-only delivery; decryption is client-side
  socket.on('secretMsgReceived', (msg) => {
    if (!msg || !msg.senderId) return;
    addSecretMessage(msg.senderId, msg);
  });
  socket.on('secretMsgSent', (msg) => {
    if (!msg || !msg.receiverId) return;
    const myId = get(authUser)?.userId;
    addSecretMessage(msg.receiverId, { ...msg, senderId: myId });
  });
  socket.on('secretMsgsHistory', (data) => {
    if (!data || !data.peerId) return;
    setSecretMessages(data.peerId, data.messages ?? []);
  });
  socket.on('secretMsgDeleted', (data) => {
    if (!data || !data.id) return;
    removeSecretMessage(data.id);
  });
  socket.on('secretMsgSeen', (data) => {
    if (!data || !data.id) return;
    updateSecretMessageSeen(data.id, data.seenAt);
  });
  socket.on('secretChatPresence', (data) => {
    if (!data || !data.userId) return;
    secretChatPresence.update(m => {
      const copy = new Map(m);
      copy.set(data.userId, { open: !!data.open, at: data.at ?? Date.now() });
      return copy;
    });
  });

  // SOS Narrative — builds crisis card for AlertOverlay / WatchViewer
  socket.on('sosNarrative', (data) => {
    if (!data || !data.userId) return;
    sosNarratives.update(m => { const nm = new Map(m); nm.set(data.userId, data); return nm; });
  });

  // Proximity SOS — someone within 5 km (not a contact) triggered SOS
  socket.on('proximitySosAlert', (data) => {
    if (!data) return;
    const dist = data.distanceKm != null
      ? (data.distanceKm < 1
          ? `${Math.round(data.distanceKm * 1000)} m`
          : `${Number(data.distanceKm).toFixed(1)} km`)
      : 'nearby';
    const watchUrl = data.watchToken
      ? getShareOrigin() + '/#/watch/' + data.watchToken
      : null;
    setBanner({
      type: 'info',
      text: `Someone ${dist} away has triggered an SOS — can you help?`,
      actions: [
        ...(watchUrl ? [{ label: 'View', kind: 'btn-primary', onClick: () => window.open(watchUrl, '_blank') }] : []),
        { label: 'OK', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
      ]
    }, 30000);
    notifyProximitySOS(data.distanceKm ?? 5).catch(() => {});
  });

  socket.on('sosUpdate', (s) => {
    if (!s) return;
    const sos = s.sos || {};
    const isMe = s.socketId === get(mySocketId);
    if (isMe) mySosActive.set(!!sos.active);
    // Track active SOS users for AlertOverlay narrative display
    if (s.userId) {
      if (sos.active) {
        // Bump badge only when this is a new SOS from someone else
        if (!isMe && !get(activeSosUsers).has(s.userId)) bumpHubBadge(true);
        activeSosUsers.update(m => { const nm = new Map(m); nm.set(s.userId, s); return nm; });
      } else {
        activeSosUsers.update(m => { const nm = new Map(m); nm.delete(s.userId); return nm; });
        sosNarratives.update(m => { const nm = new Map(m); nm.delete(s.userId); return nm; });
      }
    }
    if (sos.active) {
      // KR-010: s.displayName is always present in the payload — use it before falling
      // back to _localMap so we never show a raw socket ID UUID in the banner.
      const from = isMe ? 'You' : (s.displayName || (_localMap.get(s.socketId) || {}).displayName || 'Unknown');
      const reason = sos.reason || 'SOS';
      const ackCount = typeof sos.ackCount === 'number' ? sos.ackCount : (sos.acks ? sos.acks.length : 0);
      const ackText = ackCount ? `${ackCount} responded` : 'No one has responded yet';

      if (isMe) {
        const ackNames = Array.isArray(sos.acks) && sos.acks.length > 0
          ? sos.acks.map(a => a.by || 'Someone').join(', ')
          : null;
        const myText = ackNames
          ? `Your SOS is active — ${ackNames} responded`
          : `Your SOS is active — waiting for someone to respond`;
        // SOS banners persist — no auto-clear
        setBanner({ type: 'sos', text: myText, actions: [
          { label: 'Share watch link', kind: 'btn-secondary', onClick: () => { if (sos.token) navigator.clipboard.writeText(getShareOrigin() + '/#/watch/' + sos.token).catch(() => {}); } }
        ] });
      } else {
        const isGeofence = sos.type === 'geofence';
        const alertTitle = isGeofence ? `${from} left their safe zone` : `${from} needs help`;
        const msg = `${alertTitle} — ${reason}. ${ackText}`;
        // Fire local notification when app is backgrounded
        notifySOS(from, reason).catch(() => {});
        setBanner({ type: 'sos', text: msg, actions: [
          { label: "I'm here", kind: 'btn-primary', onClick: () => { socket.emit('ackSOS', { socketId: s.socketId }); } },
          { label: 'OK', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
        ] });
        if (ackCount === 0) {
          alertState.set({
            visible: true,
            title: isGeofence ? `${from} left their safe zone` : `${from} needs help`,
            body: reason || 'They triggered an SOS alert.',
            actions: [
              { label: "I'm here — I can help", kind: 'btn-primary', onClick: () => socket.emit('ackSOS', { socketId: s.socketId }) }
            ],
            alarmMs: isGeofence ? 7000 : 10000
          });
        }
      }
    } else if (isMe) {
      setBanner({ type: null, text: null, actions: [] });
    } else {
      // KR-001: only clear the banner when NO other SOS is still active.
      // The activeSosUsers update runs before this check; size reflects the post-delete state.
      const remainingSos = get(activeSosUsers);
      if (remainingSos.size === 0) {
        setBanner({ type: null, text: null, actions: [] });
      } else {
        // Re-raise banner for the first remaining active SOS so it isn't silently lost.
        const [, firstSos] = [...remainingSos.entries()][0];
        const firstSosData = firstSos.sos || {};
        const firstName = firstSos.displayName || 'Someone';
        const firstReason = firstSosData.reason || 'SOS';
        const firstAckCount = typeof firstSosData.ackCount === 'number' ? firstSosData.ackCount : (firstSosData.acks ? firstSosData.acks.length : 0);
        const firstAckText = firstAckCount ? `${firstAckCount} responded` : 'No one has responded yet';
        setBanner({ type: 'sos', text: `${firstName} needs help — ${firstReason}. ${firstAckText}`, actions: [
          { label: "I'm here", kind: 'btn-primary', onClick: () => { socket.emit('ackSOS', { socketId: firstSos.socketId }); } },
          { label: 'OK', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
        ] });
      }
    }
  });

  socket.on('checkInRequest', () => {
    alertState.set({
      visible: true,
      title: 'Time to check in',
      body: "Let your family know you're safe.",
      actions: [
        { label: "I'm OK", kind: 'btn-primary', onClick: () => socket.emit('checkInAck') }
      ],
      alarmMs: 5000
    });
    setBanner({ type: 'info', text: "Your family is waiting — tap to check in", actions: [
      { label: "I'm OK", kind: 'btn-primary', onClick: () => socket.emit('checkInAck') }
    ] });
  });

  socket.on('checkInMissed', (p) => {
    if (!p) return;
    setBanner({ type: 'sos', text: `${p.displayName || 'Someone'} missed their check-in`, actions: [
      { label: "I'm here", kind: 'btn-primary', onClick: () => socket.emit('ackSOS', { socketId: p.socketId }) }
    ] });
  });

  socket.on('checkInUpdate', (data) => {
    if (!data) return;
    const sid = data.socketId;
    if (sid === get(mySocketId)) return;
    const u = _localMap.get(sid);
    if (u) {
      if (!u.checkIn) u.checkIn = {};
      u.checkIn.lastCheckInAt = data.lastCheckInAt;
      _scheduleUsersFlush();
    }
  });

  // Admin overview
  socket.on('adminOverview', (data) => { if (data) adminOverview.set(data); });

  // Place alerts
  socket.on('placeAlert', (data) => {
    if (!data) return;
    const action = data.type === 'arrive' ? 'arrived at' : 'left';
    setBanner({ type: 'info', text: `${data.targetName || 'Someone'} ${action} ${data.placeName || 'a place'}`, actions: [] }, 5000);
    // Feature 8: geofence breach (leave) → haptic warning + camera shake in AlertOverlay
    if (data.type === 'leave') {
      haptics.warning?.();
      geofenceShake.update(n => n + 1);
    }
  });

  // ── F5: Speed alert — guardian receives when ward exceeds threshold ────────────
  // Go sends: { userId, displayName, speedMs, thresholdMs, lat, lng, at }
  socket.on('speedAlert', (data) => {
    if (!data) return;
    const speedKmh = data.speedMs != null ? Math.round(data.speedMs * 3.6) : '?';
    const limitKmh = data.thresholdMs != null ? Math.round(data.thresholdMs * 3.6) : '?';
    setBanner({ type: 'sos', text: `${data.displayName || 'Someone'} is going ${speedKmh} km/h (limit: ${limitKmh} km/h)`, actions: [
      { label: 'OK', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
    ] }, 8000);
  });

  // F5: Speed alert configuration ack from server
  socket.on('speedAlertSet', (data) => {
    if (!data) return;
    const kmh = data.thresholdMs ? Math.round(data.thresholdMs * 3.6) : 0;
    setBanner({ type: 'info', text: kmh > 0 ? `Speed alert set at ${kmh} km/h` : 'Speed alert disabled', actions: [] }, 3000);
  });

  // Quiet Hours update
  socket.on('quietHoursUpdated', (data) => {
    if (!data) return;
    // Update the user in _localMap so the quiet badge renders
    for (const [sid, u] of _localMap) {
      if (u.userId === data.userId) {
        u.quietHoursActive = data.active;
        _scheduleUsersFlush();
        break;
      }
    }
  });

  // ── Consumer feature events ──────────────────────────────────────────────────

  // Attest update — refresh lastAttestAt on the user in otherUsers map
  socket.on('attestUpdate', (data) => {
    if (!data?.userId) return;
    for (const [, u] of _localMap) {
      if (u.userId === data.userId) { u.lastAttestAt = data.at; _scheduleUsersFlush(); break; }
    }
  });

  // Network graph
  socket.on('networkGraph', (data) => { if (data) networkGraph.set(data); });

  // On My Way
  socket.on('onMyWayBroadcast', (data) => {
    if (!data?.displayName) return;
    const place = data.placeName ? ` to ${data.placeName}` : '';
    setBanner({ type: 'info', text: `${data.displayName} is heading out${place}`, actions: [] }, 6000);
  });
  socket.on('onMyWayCancel', () => {});

  // Co-location nudge
  socket.on('colocationNudge', (data) => {
    if (!data?.displayName) return;
    setBanner({ type: 'info', text: `${data.displayName} is nearby!`, actions: [] }, 8000);
  });

  // Gentle "haven't moved" alert (received by guardian)
  socket.on('gentleAlert', (data) => {
    if (!data?.displayName) return;
    const min = data.minutesStill ?? '?';
    notifyHaventMoved(data.displayName, min).catch(() => {});
    setBanner({ type: 'info', text: `${data.displayName} hasn't moved for ${min} minutes — you might want to check in`, actions: [] }, 8000);
  });

  // Battery proxy alert
  socket.on('batteryProxyAlert', (data) => {
    if (!data?.displayName || data.batteryPct == null) return;
    notifyBatteryLow(data.displayName, data.batteryPct).catch(() => {});
    setBanner({
      type: data.batteryPct <= 5 ? 'sos' : 'info',
      text: `${data.displayName}'s battery is at ${data.batteryPct}%`,
      actions: []
    }, 10000);
  });

  // Share My Ride — server confirms ride link was created
  socket.on('rideShareStarted', (data) => {
    if (!data?.token) return;
    // Use update() to preserve locally-set vehicleType and eta from the component
    rideShare.update(current => ({
      ...current,
      active: true,
      token: data.token,
      vehicle: data.vehicle || '',
      dest: data.dest || '',
      startedAt: Date.now(),
    }));
  });

  // Ride ended by server (token expired or endRide called from another device)
  socket.on('rideShareError', (data) => {
    setBanner({ type: 'info', text: data?.message || 'Could not share your ride right now', actions: [] }, 3000);
  });

  // Crowd Mode — someone in your festival group has drifted too far
  socket.on('crowdAlert', (data) => {
    if (!data?.fromName) return;
    const dist = data.distanceM != null ? `${data.distanceM}m` : 'far';
    setBanner({
      type: 'info',
      text: `${data.fromName} has drifted ${dist} away from your group`,
      actions: [
        { label: 'OK', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
      ]
    }, 12000);
  });

  // ── Heartbeat missed ────────────────────────────────────────────────
  socket.on('heartbeatMissed', (data) => {
    if (!data?.displayName) return;
    const offline = data.offline ? ' (offline)' : '';
    setBanner({
      type: 'info',
      text: `Haven't heard from ${data.displayName} today${offline}`,
      actions: [
        { label: 'OK', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
      ]
    }, 30000);
  });

  // ── Journey Shield events ──────────────────────────────────────────
  socket.on('tripStarted', (data) => {
    if (!data?.displayName) return;
    setBanner({ type: 'info', text: `${data.displayName} is on the move`, actions: [] }, 8000);
  });
  socket.on('tripArrived', (data) => {
    if (!data?.displayName) return;
    setBanner({ type: 'info', text: `${data.displayName} arrived safely at ${data.placeName || 'their destination'}`, actions: [] }, 10000);
  });
  socket.on('tripStoppedNew', (data) => {
    if (!data?.displayName) return;
    setBanner({
      type: 'info',
      text: `${data.displayName} stopped somewhere new — check in with them?`,
      actions: [
        { label: 'OK', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
      ]
    }, 20000);
  });

  // ── Panic Relay SMS sent ───────────────────────────────────────────
  socket.on('panicRelaySent', (data) => {
    if (!data?.phones) return;
    setBanner({
      type: 'sos',
      text: `Emergency text sent to ${data.phones.length} ${data.phones.length === 1 ? 'contact' : 'contacts'} outside the app`,
      actions: []
    }, 15000);
  });

  // ── Emergency phones updated ───────────────────────────────────────
  socket.on('emergencyPhonesUpdated', () => {
    // Handled by EmergencyProfile page directly
  });

  // ── F2: I'm Safe broadcast ─────────────────────────────────────────
  socket.on('iAmSafe', (data) => {
    if (!data?.displayName) return;
    setBanner({ type: 'info', text: `${data.displayName} is safe`, actions: [] }, 5000);
  });

  // ── F3: Meeting point updated ──────────────────────────────────────
  socket.on('meetingPointUpdated', (data) => {
    if (!data?.roomCode) return;
    myRooms.update(rooms => rooms.map(r => {
      if (r.code !== data.roomCode) return r;
      if (data.lat != null && data.lng != null) {
        return { ...r, meetingPoint: { lat: data.lat, lng: data.lng, label: data.label || '', setBy: data.setBy || '', setAt: data.setAt || 0 } };
      }
      // cleared
      const { meetingPoint: _mp, ...rest } = r;
      return rest;
    }));
  });

  // ── F6: Geofence event log ─────────────────────────────────────────
  socket.on('geofenceLog', (data) => {
    if (!data?.events) return;
    geofenceLog.set(data.events);
  });

  // ── F7: Proximity alerts ───────────────────────────────────────────
  socket.on('proximityAlert', (data) => {
    if (!data?.targetName) return;
    const dist = data.distanceM != null ? `${Math.round(data.distanceM)}m` : 'nearby';
    setBanner({
      type: 'info',
      text: `${data.targetName} is ${dist} away`,
      actions: [
        { label: 'OK', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
      ]
    }, 10000);
    notifyProximitySOS(data.distanceM != null ? data.distanceM / 1000 : 1).catch(() => {});
  });

  socket.on('proximityAlerts', (data) => {
    if (!Array.isArray(data?.alerts)) return;
    proximityAlerts.set(data.alerts);
  });

  socket.on('proximityAlertSet', (data) => {
    if (!data) return;
    proximityAlerts.update(list => {
      const idx = list.findIndex(a => a.targetUserId === data.targetUserId);
      if (idx >= 0) {
        const next = [...list];
        next[idx] = data;
        return next;
      }
      return [...list, data];
    });
    setBanner({ type: 'info', text: 'Proximity alert saved', actions: [] }, 2000);
  });

  socket.on('proximityAlertRemoved', (data) => {
    if (!data?.targetUserId) return;
    proximityAlerts.update(list => list.filter(a => a.targetUserId !== data.targetUserId));
    setBanner({ type: 'info', text: 'Proximity alert removed', actions: [] }, 2000);
  });

  // ── F8: Room bulletin board ────────────────────────────────────────
  socket.on('roomNotes', (data) => {
    if (!data?.roomCode) return;
    roomNotes.update(m => {
      const nm = new Map(m);
      nm.set(data.roomCode, data.notes || []);
      return nm;
    });
  });

  // Go sends note fields flat on the payload (id, roomCode, authorId, authorName, body, createdAt).
  // Extract the note object without roomCode and prepend it to the room's list.
  socket.on('roomNoteAdded', (data) => {
    if (!data?.roomCode || !data?.id) return;
    const { roomCode, ...note } = data;
    roomNotes.update(m => {
      const nm = new Map(m);
      const existing = nm.get(roomCode) || [];
      nm.set(roomCode, [note, ...existing].slice(0, 20));
      return nm;
    });
  });

  socket.on('roomNoteDeleted', (data) => {
    if (!data?.noteId || !data?.roomCode) return;
    roomNotes.update(m => {
      const nm = new Map(m);
      const existing = nm.get(data.roomCode) || [];
      nm.set(data.roomCode, existing.filter(n => n.id !== data.noteId));
      return nm;
    });
  });

  // ── F9: Daily activity summary ─────────────────────────────────────
  socket.on('dailyActivity', (data) => {
    if (!data?.userId) return;
    dailyActivity.update(m => {
      const nm = new Map(m);
      nm.set(data.userId, data.days || []);
      return nm;
    });
  });

  // ── F10: Trail playback ────────────────────────────────────────────
  socket.on('recentTrailData', (data) => {
    if (!data) return;
    recentTrailResult.set({ ok: true, ...data });
  });
  socket.on('trailError', (data) => {
    recentTrailResult.set({ ok: false, error: data?.error || 'Could not load trail' });
  });

  // ── Arrival ETA projections ────────────────────────────────────────
  socket.on('arrivalProjection', (data) => {
    if (!data?.userId) return;
    arrivalProjections.update(m => {
      const nm = new Map(m);
      if (data.etaSeconds != null) {
        nm.set(data.userId, data);
      } else {
        nm.delete(data.userId); // cleared — user is no longer moving toward a place
      }
      return nm;
    });
  });

  // Network online/offline detection for immediate UX feedback
  if (typeof window !== 'undefined') {
    window.addEventListener('offline', () => {
      setBanner({ type: 'info', text: "You're offline. Positions will be buffered.", actions: [] });
    });
    window.addEventListener('online', () => {
      // Cancel the pending "Reconnecting…" timer — the upcoming socket.connect() will
      // reconnect quickly and the 'connect' handler will clear the banner naturally.
      // We do NOT show "Back online!" here because the socket may not be connected yet.
      cancelReconnectBanner();
      if (!socket.connected) socket.connect();
    });
  }

  // ── WebRTC signaling relay ─────────────────────────────────────────────────
  // Only the offer sets the store (triggers IncomingCallOverlay).
  // answer/ice/hangup are handled by webrtc.js initWebRTCSocketHandlers().
  socket.on('webrtc:offer', (data) => { incomingOffer.set(data); });

  // All handlers registered -- now connect
  socket.connect();
}

export function isConnected() { return connected; }

/** Notify the backend that the receiver has decrypted (read) a secret message. */
export function markSecretMsgSeen(msgId) {
  socket.emit('markSecretMsgSeen', { msgId });
}

/**
 * Ask the backend to generate a 24h invite token for the given secret chat conversation.
 * Returns a Promise that resolves to the token string.
 */
export function createSecretChatInvite(peerId) {
  return new Promise((resolve, reject) => {
    const nonce = (typeof crypto !== 'undefined' && crypto.randomUUID)
      ? crypto.randomUUID()
      : String(Date.now()) + '-' + Math.random().toString(36).slice(2);

    const handler = (data) => {
      if (data?.nonce !== nonce) return; // not our response
      socket.off('secretChatInviteCreated', handler);
      clearTimeout(timer);
      resolve(data.token);
    };

    const timer = setTimeout(() => {
      socket.off('secretChatInviteCreated', handler);
      reject(new Error('createSecretChatInvite timed out'));
    }, 10000);

    socket.on('secretChatInviteCreated', handler);
    socket.emit('createSecretChatInvite', { peerId, nonce });
  });
}

// ── F2: I'm Safe emit ──────────────────────────────────────────────────────
export function emitIAmSafe() {
  socket.emit('iAmSafe', {});
}

// ── F3: Meeting point emits ────────────────────────────────────────────────
export function emitSetMeetingPoint(roomCode, lat, lng, label) {
  socket.emit('setMeetingPoint', { roomCode, lat, lng, label: label || '' });
}

export function emitClearMeetingPoint(roomCode) {
  socket.emit('clearMeetingPoint', { roomCode });
}

// ── F5: Speed alert emit ───────────────────────────────────────────────────
export function emitSetSpeedAlert(thresholdKmh) {
  socket.emit('setSpeedAlert', { thresholdKmh });
}

// ── F6: Geofence log emit ──────────────────────────────────────────────────
export function emitGetGeofenceLog() {
  socket.emit('getGeofenceLog', {});
}

// ── F7: Proximity alert emits ──────────────────────────────────────────────
export function emitSetProximityAlert(targetUserId, radiusM) {
  socket.emit('setProximityAlert', { targetUserId, radiusM });
}

export function emitRemoveProximityAlert(targetUserId) {
  socket.emit('removeProximityAlert', { targetUserId });
}

export function emitListProximityAlerts() {
  socket.emit('listProximityAlerts', {});
}

// ── F8: Room bulletin board emits ──────────────────────────────────────────
export function emitPostRoomNote(roomCode, body) {
  socket.emit('postRoomNote', { roomCode, body });
}

export function emitDeleteRoomNote(noteId, roomCode) {
  socket.emit('deleteRoomNote', { noteId, roomCode });
}

export function emitGetRoomNotes(roomCode) {
  socket.emit('getRoomNotes', { roomCode });
}

// ── F9: Daily activity emit ────────────────────────────────────────────────
export function emitGetDailyActivity(userId) {
  socket.emit('getDailyActivity', userId ? { userId } : {});
}

// ── F10: Trail emit ────────────────────────────────────────────────────────
export function emitGetRecentTrail(targetUserId, windowMinutes) {
  socket.emit('getRecentTrail', { targetUserId, windowMinutes: windowMinutes || 60 });
}

// ── WebRTC signaling emit helpers ─────────────────────────────────────────
export function emitWebRTCOffer(targetUserID, sdp) {
  socket.emit('webrtc:offer', { targetUserID, sdp });
}

export function emitWebRTCAnswer(targetUserID, sdp) {
  socket.emit('webrtc:answer', { targetUserID, sdp });
}

export function emitWebRTCIce(targetUserID, candidate) {
  socket.emit('webrtc:ice', { targetUserID, candidate });
}

export function emitWebRTCHangup(targetUserID) {
  socket.emit('webrtc:hangup', { targetUserID });
}
