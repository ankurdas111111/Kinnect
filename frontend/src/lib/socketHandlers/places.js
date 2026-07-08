import { geofenceLog, proximityAlerts } from '../stores/places.js';
import { geofenceShake } from '../stores/sos.js';
import { arrivalProjections } from '../stores/arrivals.js';
import { haptics } from '../haptics.js';
import { notifyProximitySOS } from '../nativeNotifications.js';

/**
 * Places-domain socket handlers: place arrive/leave alerts, geofence log,
 * proximity alerts, and arrival ETA projections.
 */
export function register(socket, ctx) {
  const { setBanner } = ctx;

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
}
