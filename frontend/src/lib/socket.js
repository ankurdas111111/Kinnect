import { get } from 'svelte/store';
import { otherUsers, mySocketId, mySafetyStatus } from './stores/map.js';
import { myRooms } from './stores/rooms.js';
import { myContacts } from './stores/contacts.js';
import { banner, myLiveLinks, mySosActive, sosNarratives, activeSosUsers } from './stores/sos.js';
import { pulseMap } from './stores/pulses.js';
import { cancelAll as cancelVoiceQueue } from './voice.js';
import { networkGraph } from './stores/network.js';
import { recordLatency } from './stores/latency.js';
import { drainBuffer, hasBuffered } from './offlineBuffer.js';
import { createRealtimeSocket } from './realtimeClient.js';
import { register as registerRoomHandlers } from './socketHandlers/rooms.js';
import { register as registerContactHandlers } from './socketHandlers/contacts.js';
import { register as registerGuardianHandlers } from './socketHandlers/guardians.js';
import { register as registerSosHandlers } from './socketHandlers/sos.js';
import { register as registerChatHandlers } from './socketHandlers/chat.js';
import { register as registerPlaceHandlers } from './socketHandlers/places.js';
import { register as registerSafetyHandlers } from './socketHandlers/safety.js';
import { register as registerSocialHandlers } from './socketHandlers/social.js';
import { register as registerMiscHandlers } from './socketHandlers/misc.js';
import { register as registerActivityRecorder } from './socketHandlers/activityRecorder.js';
import { initVoiceAnnouncer } from './voiceAnnouncer.js';

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
  cancelVoiceQueue(); // logout must not keep talking
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

  function applyUserUpdate(user) {
    const mySid = get(mySocketId);
    if (!mySid || user.socketId === mySid) {
      if (mySid && user.socketId === mySid) extractSafety(user);
      return;
    }
    if (user.timestamp) recordLatency(user.timestamp, user.serverTs);
    _localMap.set(user.socketId, user);
    _scheduleUsersFlush();
  }

  socket.on('userUpdate', applyUserUpdate);

  // Batched position frame: the server groups every update visible to this
  // client in a 40ms tick into one array — one frame per tick instead of one
  // frame per moving user.
  socket.on('userUpdates', (users) => {
    (users || []).forEach(applyUserUpdate);
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

  // ── Domain handler modules (see ./socketHandlers/) ──────────────────────────
  // Shared context: banner helper + access to the microtask-batched user map.
  // _localMap is reassigned on existingUsers/visibilityRefresh/reset, so modules
  // must go through getLocalMap() instead of holding a direct reference.
  const ctx = {
    setBanner,
    getLocalMap: () => _localMap,
    scheduleUsersFlush: _scheduleUsersFlush
  };

  registerRoomHandlers(socket, ctx);
  registerContactHandlers(socket, ctx);
  registerGuardianHandlers(socket, ctx);
  registerSosHandlers(socket, ctx);
  registerChatHandlers(socket, ctx);
  registerPlaceHandlers(socket, ctx);
  registerSafetyHandlers(socket, ctx);
  registerSocialHandlers(socket, ctx);
  registerMiscHandlers(socket, ctx);
  registerActivityRecorder(socket, ctx);
  initVoiceAnnouncer(socket); // spoken updates: arrival + verdict watchers

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
