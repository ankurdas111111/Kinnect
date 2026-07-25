import { get } from 'svelte/store';
import { mySocketId } from '../stores/map.js';
import { alertState, mySosActive, sosNarratives, activeSosUsers } from '../stores/sos.js';
import { authUser } from '../stores/auth.js';
import { bumpHubBadge } from '../stores/hubBadge.js';
import { pulseMap } from '../stores/pulses.js';
import { pushActivity } from '../activityLog.js';
import { announceCheckin } from '../voiceAnnouncer.js';
import { notifySOS, notifyProximitySOS } from '../nativeNotifications.js';
import { getShareOrigin } from '../env.js';

/**
 * SOS-domain socket handlers: SOS lifecycle, narratives, proximity SOS,
 * check-ins, panic relay, and emergency profile acks.
 */
export function register(socket, ctx) {
  const { setBanner, getLocalMap, scheduleUsersFlush } = ctx;

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
      // back to the local users map so we never show a raw socket ID UUID in the banner.
      const from = isMe ? 'You' : (s.displayName || (getLocalMap().get(s.socketId) || {}).displayName || 'Unknown');
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
    const u = getLocalMap().get(sid);
    if (u) {
      if (!u.checkIn) u.checkIn = {};
      u.checkIn.lastCheckInAt = data.lastCheckInAt;
      scheduleUsersFlush();
    }
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
    // Surface as a Hub "all good" chip on that member's row for 5 min (zero DB —
    // pulseMap is an in-memory TTL store). Skip our own echo.
    if (data.userId && data.userId !== get(authUser)?.userId) {
      const expiresAt = Date.now() + 5 * 60_000;
      pulseMap.update((m) => {
        const nm = new Map(m);
        nm.set(data.userId, { userId: data.userId, displayName: data.displayName, type: 'ok', at: data.at || Date.now(), expiresAt });
        return nm;
      });
      setTimeout(() => {
        pulseMap.update((m) => { const nm = new Map(m); if (nm.get(data.userId)?.expiresAt <= Date.now()) nm.delete(data.userId); return nm; });
      }, 5 * 60_000);
      pushActivity({ type: 'contact', userId: data.userId, userName: data.displayName, message: `${data.displayName} checked in — safe` });
      announceCheckin(data.displayName);
    }
  });
}
