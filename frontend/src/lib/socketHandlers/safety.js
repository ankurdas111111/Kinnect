import { notifyBatteryLow, notifyHaventMoved } from '../nativeNotifications.js';

/**
 * Safety/journey-domain socket handlers: speed alerts, quiet hours,
 * attestation, on-my-way, co-location, gentle alerts, battery proxy,
 * heartbeat, and Journey Shield trip events.
 */
export function register(socket, ctx) {
  const { setBanner, getLocalMap, scheduleUsersFlush } = ctx;

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
    // Update the user in the local map so the quiet badge renders
    for (const [sid, u] of getLocalMap()) {
      if (u.userId === data.userId) {
        u.quietHoursActive = data.active;
        scheduleUsersFlush();
        break;
      }
    }
  });

  // Attest update — refresh lastAttestAt on the user in otherUsers map
  socket.on('attestUpdate', (data) => {
    if (!data?.userId) return;
    for (const [, u] of getLocalMap()) {
      if (u.userId === data.userId) { u.lastAttestAt = data.at; scheduleUsersFlush(); break; }
    }
  });

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
}
