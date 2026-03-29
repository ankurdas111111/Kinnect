import { get } from 'svelte/store';
import { otherUsers, mySocketId, myLocation, mySafetyStatus } from './stores/map.js';
import { myRooms, myShareCode, myContactInfo } from './stores/rooms.js';
import { myContacts } from './stores/contacts.js';
import { myGuardianData, canManage, pendingIncomingRequests } from './stores/guardians.js';
import { banner, alertState, myLiveLinks, mySosActive, sosNarratives, activeSosUsers } from './stores/sos.js';
import { adminOverview } from './stores/admin.js';
import { privacyPause } from './stores/places.js';
import { authUser } from './stores/auth.js';
import { drainBuffer, hasBuffered } from './offlineBuffer.js';
import { pulseMap } from './stores/pulses.js';
import { arrivalProjections } from './stores/arrivals.js';
import { networkGraph } from './stores/network.js';
import { trailData } from './stores/trail.js';
import { recordLatency } from './stores/latency.js';
import { createRealtimeSocket } from './realtimeClient.js';
import { notifySOS, notifyGuardianRequest, notifyBatteryLow, notifyHaventMoved } from './nativeNotifications.js';

const storedClientId = localStorage.getItem('clientId');
const clientId = storedClientId || (crypto && crypto.randomUUID ? crypto.randomUUID() : String(Date.now()) + '-' + Math.random().toString(16).slice(2));
if (!storedClientId) localStorage.setItem('clientId', clientId);

export const socket = createRealtimeSocket({
  auth: { clientId },
  autoConnect: false,
  reconnection: true,
  reconnectionAttempts: 50,
  reconnectionDelay: 200,
  reconnectionDelayMax: 3000,
  randomizationFactor: 0.3,
  timeout: 8000
});

let connected = false;
let handlersRegistered = false;

// ── Fix F: single cancellable banner timer to prevent timer pile-up ──────────
let _bannerClearTimer = null;
function setBanner(b, autoClearMs) {
  if (_bannerClearTimer) { clearTimeout(_bannerClearTimer); _bannerClearTimer = null; }
  banner.set(b);
  if (autoClearMs) {
    _bannerClearTimer = setTimeout(() => {
      _bannerClearTimer = null;
      banner.set({ type: null, text: null, actions: [] });
    }, autoClearMs);
  }
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
  // Only show reconnecting banner if disconnected for more than 4 seconds.
  // Brief drops (network hiccups, Render cold starts) should be invisible.
  let _reconnectBannerTimer = null;

  socket.on('connect', () => {
    connected = true;
    if (_reconnectBannerTimer) { clearTimeout(_reconnectBannerTimer); _reconnectBannerTimer = null; }
    setBanner({ type: null, text: null, actions: [] });
  });

  // Server tells us our assigned socket ID — use this for self-filtering
  socket.on('welcome', (data) => {
    if (data?.socketId) mySocketId.set(data.socketId);
  });

  socket.on('disconnect', (reason) => {
    connected = false;
    // Only show banner after 10s — brief drops on network/background should be invisible.
    // Android WebView suspends WS when the app is backgrounded; the appStateChange
    // listener reconnects in <1s on foreground, so 10s covers all normal cases.
    _reconnectBannerTimer = setTimeout(() => {
      _reconnectBannerTimer = null;
      if (!connected) setBanner({ type: 'info', text: 'Reconnecting...', actions: [] });
    }, 10000);
    // Auth errors need immediate feedback
    if (reason === 'io server disconnect') {
      clearTimeout(_reconnectBannerTimer);
      _reconnectBannerTimer = null;
    }
  });

  socket.on('connect_error', (err) => {
    const msg = err && err.message ? err.message : '';
    if (msg.includes('Authentication') || msg.includes('session') || msg.includes('401') || msg.includes('403')) {
      if (_reconnectBannerTimer) { clearTimeout(_reconnectBannerTimer); _reconnectBannerTimer = null; }
      setBanner({ type: 'sos', text: 'Session expired. Redirecting to login...', actions: [] });
      setTimeout(() => { window.location.hash = '#/login'; }, 2000);
    }
    // Other errors: silently retry, banner already scheduled from disconnect
  });

  socket.io.on('reconnect', () => {
    // Drain offline buffer after a short delay
    if (hasBuffered()) {
      setTimeout(() => {
        const batch = drainBuffer();
        if (batch.length > 0 && socket.connected) socket.emit('positionBatch', batch);
      }, 500);
    }
  });

  // No per-attempt banner — too noisy
  socket.io.on('reconnect_attempt', () => {});

  socket.io.on('reconnect_failed', () => {
    if (_reconnectBannerTimer) { clearTimeout(_reconnectBannerTimer); _reconnectBannerTimer = null; }
    setBanner({ type: 'sos', text: 'Unable to reconnect. Please refresh the page.', actions: [
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
    const statusMsg = data.status === 'active' ? 'approved' : data.status === 'denied' ? 'denied' : data.status === 'revoked' ? 'revoked' : data.status === 'expired' ? 'expired' : data.status;
    setBanner({ type: 'info', text: 'Guardian relationship ' + statusMsg, actions: [] }, 2000);
  });

  // Room/contact action results
  socket.on('roomError', (data) => {
    setBanner({ type: 'info', text: data?.message || 'Room error', actions: [] }, 2500);
  });
  socket.on('contactError', (data) => {
    setBanner({ type: 'info', text: data?.message || 'Contact error', actions: [] }, 2500);
  });
  socket.on('roomCreated', (data) => {
    setBanner({ type: 'info', text: `Room "${data.name}" created! Code: ${data.code}`, actions: [] }, 3000);
  });
  socket.on('roomJoined', (data) => {
    setBanner({ type: 'info', text: `Joined room "${data.name}"`, actions: [] }, 2000);
  });
  socket.on('roomLeft', (data) => {
    setBanner({ type: 'info', text: `Left room "${data?.name || ''}"`, actions: [] }, 2000);
  });
  socket.on('contactAdded', (data) => {
    setBanner({ type: 'info', text: `Added ${data?.displayName || 'contact'} to contacts`, actions: [] }, 2000);
  });
  socket.on('contactRemoved', () => {
    setBanner({ type: 'info', text: 'Contact removed', actions: [] }, 2000);
  });
  socket.on('liveLinkCreated', (data) => {
    const url = window.location.origin + '/#/live/' + data.token;
    navigator.clipboard.writeText(url).catch(() => {
      // Clipboard write failed (e.g. permissions denied) — show the URL so user can copy manually.
      setBanner({ type: 'info', text: 'Live link: ' + url, actions: [] }, 10000);
      return;
    });
    setBanner({ type: 'info', text: 'Live link created and copied!', actions: [] }, 2500);
  });

  // SOS events (persistent banners — no auto-clear)
  // Pulse Check-In — ephemeral heartbeat from a contact
  socket.on('pulseReceived', (data) => {
    if (!data || !data.userId) return;
    pulseMap.update(m => {
      m.set(data.userId, data);
      return m;
    });
    // Auto-expire after 30s
    setTimeout(() => {
      pulseMap.update(m => { m.delete(data.userId); return m; });
    }, 30000);
    if (data.type === 'ok') {
      setBanner({ type: 'info', text: `${data.displayName || 'Someone'} sent an I'm OK pulse`, actions: [] }, 5000);
    } else if (data.type === 'callme') {
      setBanner({ type: 'sos', text: `${data.displayName || 'Someone'} needs you to call them`, actions: [
        { label: 'Dismiss', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
      ] }, 8000);
    }
  });

  // SOS Narrative — builds crisis card for AlertOverlay / WatchViewer
  socket.on('sosNarrative', (data) => {
    if (!data || !data.userId) return;
    sosNarratives.update(m => { m.set(data.userId, data); return m; });
  });

  socket.on('sosUpdate', (s) => {
    if (!s) return;
    const sos = s.sos || {};
    const isMe = s.socketId === get(mySocketId);
    if (isMe) mySosActive.set(!!sos.active);
    // Track active SOS users for AlertOverlay narrative display
    if (s.userId) {
      if (sos.active) {
        activeSosUsers.update(m => { m.set(s.userId, s); return m; });
      } else {
        activeSosUsers.update(m => { m.delete(s.userId); return m; });
      }
    }
    if (sos.active) {
      const from = isMe ? 'You' : ((_localMap.get(s.socketId) || {}).displayName || s.socketId);
      const reason = sos.reason || 'SOS';
      const ackCount = typeof sos.ackCount === 'number' ? sos.ackCount : (sos.acks ? sos.acks.length : 0);
      const ackText = ackCount ? `Acknowledged (${ackCount})` : 'Not acknowledged';

      if (isMe) {
        const ackNames = Array.isArray(sos.acks) && sos.acks.length > 0
          ? sos.acks.map(a => a.by || 'Someone').join(', ')
          : null;
        const myText = ackNames
          ? `Your SOS is active: ${reason} — Acknowledged by: ${ackNames}`
          : `Your SOS is active: ${reason} — Not yet acknowledged`;
        // SOS banners persist — no auto-clear
        setBanner({ type: 'sos', text: myText, actions: [
          { label: 'Copy watch link', kind: 'btn-secondary', onClick: () => { if (sos.token) navigator.clipboard.writeText(window.location.origin + '/#/watch/' + sos.token).catch(() => {}); } }
        ] });
      } else {
        const isGeofence = sos.type === 'geofence';
        const msg = `${isGeofence ? 'GEOFENCE BREACH' : 'SOS'} from ${from}: ${reason} - ${ackText}`;
        // Fire local notification when app is backgrounded
        notifySOS(from, reason).catch(() => {});
        setBanner({ type: 'sos', text: msg, actions: [
          { label: 'Acknowledge', kind: 'btn-primary', onClick: () => { socket.emit('ackSOS', { socketId: s.socketId }); } },
          { label: 'Dismiss', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
        ] });
        if (ackCount === 0) {
          alertState.set({
            visible: true,
            title: isGeofence ? 'GEOFENCE BREACH' : 'SOS ALERT',
            body: msg,
            actions: [
              { label: 'Acknowledge', kind: 'btn-primary', onClick: () => socket.emit('ackSOS', { socketId: s.socketId }) }
            ],
            alarmMs: isGeofence ? 7000 : 10000
          });
        }
      }
    } else if (isMe) {
      setBanner({ type: null, text: null, actions: [] });
    } else {
      // Non-self SOS resolved — clear any active SOS banner for this user.
      setBanner({ type: null, text: null, actions: [] });
    }
  });

  socket.on('checkInRequest', () => {
    alertState.set({
      visible: true,
      title: 'CHECK-IN REQUIRED',
      body: "Tap \"I'm OK\" to confirm you are safe.",
      actions: [
        { label: "I'm OK", kind: 'btn-primary', onClick: () => socket.emit('checkInAck') }
      ],
      alarmMs: 5000
    });
    setBanner({ type: 'info', text: "Check-in requested. Tap \"I'm OK\".", actions: [
      { label: "I'm OK", kind: 'btn-primary', onClick: () => socket.emit('checkInAck') }
    ] });
  });

  socket.on('checkInMissed', (p) => {
    if (!p) return;
    setBanner({ type: 'sos', text: 'Missed check-in: ' + (p.displayName || p.socketId), actions: [
      { label: 'Acknowledge SOS', kind: 'btn-primary', onClick: () => socket.emit('ackSOS', { socketId: p.socketId }) }
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
  });

  // Speed alerts
  socket.on('speedAlert', (data) => {
    if (!data) return;
    setBanner({ type: 'sos', text: `${data.targetName || 'Someone'} is going ${Math.round(data.speed)} km/h (limit: ${Math.round(data.threshold)})`, actions: [
      { label: 'Dismiss', kind: 'btn-secondary', onClick: () => setBanner({ type: null, text: null, actions: [] }) }
    ] }, 8000);
  });

  // Arrival Intelligence — ETA projections to named saved places
  socket.on('arrivalProjection', (data) => {
    if (!data || !data.userId) return;
    arrivalProjections.update(m => { m.set(data.userId, data); return m; });
  });

  socket.on('arrivalDismiss', (data) => {
    if (!data || !data.userId) return;
    arrivalProjections.update(m => { m.delete(data.userId); return m; });
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

  // Privacy pause
  socket.on('privacyPauseUpdate', (data) => {
    if (data) privacyPause.set(data.pausedUntil || null);
  });

  // ── Consumer feature events ──────────────────────────────────────────────────

  // Attest update — refresh lastAttestAt on the user in otherUsers map
  socket.on('attestUpdate', (data) => {
    if (!data?.userId) return;
    for (const [, u] of _localMap) {
      if (u.userId === data.userId) { u.lastAttestAt = data.at; _scheduleUsersFlush(); break; }
    }
  });

  // Trail data — store for Map.svelte to render polyline
  socket.on('recentTrail', (data) => {
    if (!data?.userId || !Array.isArray(data.points)) return;
    trailData.update(m => { m.set(data.userId, data); return m; });
  });
  socket.on('trailError', (data) => {
    setBanner({ type: 'info', text: data?.error || 'Trail unavailable', actions: [] }, 2000);
  });

  // Network graph
  socket.on('networkGraph', (data) => { if (data) networkGraph.set(data); });

  // On My Way
  socket.on('onMyWayBroadcast', (data) => {
    if (!data?.displayName) return;
    const place = data.placeName ? ` → ${data.placeName}` : '';
    setBanner({ type: 'info', text: `${data.displayName} is on their way${place}`, actions: [] }, 6000);
  });
  socket.on('onMyWayCancel', () => {});

  // Co-location nudge
  socket.on('colocationNudge', (data) => {
    if (!data?.displayName) return;
    setBanner({ type: 'info', text: `You're near ${data.displayName}!`, actions: [] }, 8000);
  });

  // Gentle "haven't moved" alert (received by guardian)
  socket.on('gentleAlert', (data) => {
    if (!data?.displayName) return;
    const min = data.minutesStill ?? '?';
    notifyHaventMoved(data.displayName, min).catch(() => {});
    setBanner({ type: 'info', text: `${data.displayName} hasn't moved in ${min} min`, actions: [] }, 8000);
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

  // Network online/offline detection for immediate UX feedback
  if (typeof window !== 'undefined') {
    window.addEventListener('offline', () => {
      setBanner({ type: 'info', text: "You're offline. Positions will be buffered.", actions: [] });
    });
    window.addEventListener('online', () => {
      setBanner({ type: 'info', text: 'Back online!', actions: [] }, 2000);
      if (!socket.connected) socket.connect();
    });
  }

  // All handlers registered -- now connect
  socket.connect();
}

export function isConnected() { return connected; }
