import { get } from 'svelte/store';
import { pushActivity } from '../activityLog.js';
import { recordTransition } from '../presenceRhythm.js';
import { authUser } from '../stores/auth.js';
import { announceSosStart, announceSosEnd } from '../voiceAnnouncer.js';

/**
 * Global, always-on activity recorder.
 *
 * Previously the Activity Feed only captured events while it was open, so the
 * feed + Hub "Recent" peek were blank until you visited them. This module
 * registers ONCE at socket init and persists notable family events to
 * activityLog (device-local) regardless of what screen you're on. It also feeds
 * on-device presence rhythm from the same online/offline transitions.
 *
 * DB load: ZERO — reads the live WS stream, writes only to localStorage.
 */
export function register(socket, ctx) {
  const getLocalMap = ctx?.getLocalMap || (() => new Map());

  // Track which userIds currently have an active SOS so start/end log once each.
  const sosActiveSet = new Set();

  socket.on('userConnected', (d) => {
    if (!d) return;
    pushActivity({ type: 'position', userId: d.userId, userName: d.displayName, message: `${d.displayName || 'Someone'} came online` });
    recordTransition(d.userId, true);
  });

  socket.on('userOffline', (d) => {
    if (!d) return;
    pushActivity({ type: 'offline', userId: d.userId, userName: d.displayName, message: `${d.displayName || 'Someone'} went offline` });
    recordTransition(d.userId, false);
  });

  // userDisconnect sends a raw socketId — resolve the name/userId from the map.
  socket.on('userDisconnect', (socketId) => {
    const uid = typeof socketId === 'string' ? socketId : socketId?.socketId;
    const user = getLocalMap()?.get?.(uid);
    const name = user?.displayName || 'Someone';
    pushActivity({ type: 'offline', userId: user?.userId || null, userName: name, message: `${name} left` });
    if (user?.userId) recordTransition(user.userId, false);
  });

  // Backend emits sosUpdate for both SOS start AND end; diff against the set.
  socket.on('sosUpdate', (d) => {
    if (!d || !d.userId) return;
    const sos = d.sos || {};
    const name = d.displayName || 'Someone';
    const isMe = d.userId === get(authUser)?.userId;
    if (sos.active && !sosActiveSet.has(d.userId)) {
      sosActiveSet.add(d.userId);
      pushActivity({ type: 'sos_start', userId: d.userId, userName: name, message: `SOS — ${sos.reason || 'Emergency'}`, severity: 'danger' });
      announceSosStart(name, sos.reason, isMe, /geofence|safe zone/i.test(sos.reason || ''));
    } else if (!sos.active && sosActiveSet.has(d.userId)) {
      sosActiveSet.delete(d.userId);
      pushActivity({ type: 'sos_end', userId: d.userId, userName: name, message: `${name} cancelled SOS` });
      announceSosEnd(name, isMe);
    }
  });

  socket.on('contactAdded', () => {
    pushActivity({ type: 'contact', userId: null, userName: null, message: 'New contact added to your network' });
  });

  socket.on('roomJoined', (d) => {
    pushActivity({ type: 'self', userId: null, userName: null, message: `Joined room: ${d?.name || d?.roomId || 'Unknown'}` });
  });
}
