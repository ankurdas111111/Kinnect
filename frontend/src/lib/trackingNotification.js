/**
 * Persistent tracking notification for Android.
 *
 * Shows an ongoing (non-dismissible) notification while Kinnect is actively
 * tracking. Provides context-aware action buttons:
 *
 *   Default:     [Share Ride]  [On My Way]  [Pause]
 *   Active ride: [Reached Safely]  [SOS]
 *
 * Action button taps fire callbacks registered via onAction().
 * Updates throttled to every 30s to conserve battery.
 */

const NOTIF_ID = 50;           // Fixed ID so we can update in place
const CHANNEL_ID = 'kinnect_tracking';
const UPDATE_THROTTLE = 30000; // 30 seconds

let _plugin = null;
let _active = false;
let _actionCallbacks = {};
let _listenerRegistered = false;
let _lastUpdateAt = 0;

function isNative() {
  try {
    return typeof window !== 'undefined' &&
      !!window.Capacitor &&
      typeof window.Capacitor.isNativePlatform === 'function' &&
      window.Capacitor.isNativePlatform();
  } catch (_) { return false; }
}

async function getPlugin() {
  if (!isNative()) return null;
  if (_plugin) return _plugin;
  try {
    const { LocalNotifications } = await import('@capacitor/local-notifications');
    _plugin = LocalNotifications;
    return _plugin;
  } catch (_) { return null; }
}

/**
 * Create the tracking notification channel (call once at app startup).
 * Lower importance than alerts — no sound/vibration.
 */
export async function setupTrackingChannel() {
  const p = await getPlugin();
  if (!p) return;
  try {
    await p.createChannel({
      id: CHANNEL_ID,
      name: 'Tracking Status',
      description: 'Persistent notification while Kinnect is sharing your location',
      importance: 2, // IMPORTANCE_LOW — no sound, shows in shade
      visibility: 1, // VISIBILITY_PUBLIC
      sound: '',
      vibration: false,
    });
  } catch (_) {}
}

/**
 * Register action types for the notification buttons.
 * Call once at startup after setupTrackingChannel.
 */
export async function registerActions() {
  const p = await getPlugin();
  if (!p) return;
  try {
    await p.registerActionTypes({
      types: [
        {
          id: 'tracking_default',
          actions: [
            { id: 'share_ride', title: 'Share Ride' },
            { id: 'on_my_way', title: 'On My Way' },
            { id: 'pause', title: 'Pause' },
          ],
        },
        {
          id: 'tracking_ride',
          actions: [
            { id: 'reached_safely', title: 'Reached Safely' },
            { id: 'sos', title: 'SOS' },
          ],
        },
      ],
    });

    // Register action listener once
    if (!_listenerRegistered) {
      _listenerRegistered = true;
      p.addListener('localNotificationActionPerformed', (event) => {
        const actionId = event.actionId;
        if (actionId && _actionCallbacks[actionId]) {
          _actionCallbacks[actionId]();
        }
      });
    }
  } catch (_) {}
}

/**
 * Register a callback for a notification action button tap.
 * @param {string} actionId - 'share_ride' | 'on_my_way' | 'pause' | 'reached_safely' | 'sos'
 * @param {Function} callback
 */
export function onAction(actionId, callback) {
  _actionCallbacks[actionId] = callback;
}

/**
 * Show or update the persistent tracking notification.
 * @param {object} state
 * @param {number} state.visibleCount - Number of people who can see you
 * @param {number} state.accuracy - GPS accuracy in meters
 * @param {boolean} state.rideActive - Whether a ride is in progress
 * @param {string} [state.rideDest] - Ride destination
 * @param {string} [state.rideVehicle] - Vehicle plate/identifier
 * @param {number} [state.rideEtaMins] - Minutes until ETA
 * @param {boolean} [state.force] - Bypass throttle
 */
export async function showOrUpdate(state = {}) {
  const p = await getPlugin();
  if (!p) return;

  // Throttle updates
  const now = Date.now();
  if (!state.force && _active && now - _lastUpdateAt < UPDATE_THROTTLE) return;
  _lastUpdateAt = now;

  const { visibleCount = 0, accuracy, rideActive, rideDest, rideVehicle, rideEtaMins } = state;

  let title, body, actionTypeId;

  if (rideActive) {
    title = 'Ride Active';
    const parts = [];
    if (rideDest) parts.push(`to ${rideDest}`);
    if (rideVehicle) parts.push(rideVehicle);
    if (rideEtaMins != null && rideEtaMins > 0) parts.push(`ETA ${rideEtaMins}m`);
    if (parts.length === 0) parts.push('Sharing live with family');
    body = parts.join(' · ');
    actionTypeId = 'tracking_ride';
  } else {
    title = 'Kinnect · Live';
    const parts = [];
    if (visibleCount > 0) parts.push(`Sharing with ${visibleCount} ${visibleCount === 1 ? 'person' : 'people'}`);
    else parts.push('Location sharing active');
    if (accuracy != null) {
      const accStr = accuracy <= 15 ? `GPS ±${Math.round(accuracy)}m` : accuracy <= 50 ? `GPS ~${Math.round(accuracy)}m` : 'Rough GPS';
      parts.push(accStr);
    }
    body = parts.join(' · ');
    actionTypeId = 'tracking_default';
  }

  try {
    await p.schedule({
      notifications: [{
        id: NOTIF_ID,
        title,
        body,
        ongoing: true,
        autoCancel: false,
        channelId: CHANNEL_ID,
        actionTypeId,
        smallIcon: 'ic_stat_notify',
        extra: { type: 'tracking' },
      }],
    });
    _active = true;
  } catch (_) {}
}

/**
 * Dismiss the persistent tracking notification.
 */
export async function dismiss() {
  if (!_active) return;
  _active = false;
  _lastUpdateAt = 0;
  const p = await getPlugin();
  if (!p) return;
  try {
    await p.cancel({ notifications: [{ id: NOTIF_ID }] });
  } catch (_) {}
}

/**
 * Check if the tracking notification is currently showing.
 */
export function isActive() {
  return _active;
}
