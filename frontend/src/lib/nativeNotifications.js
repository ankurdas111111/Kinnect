/**
 * Local notifications for Android.
 * Fires system tray notifications when the app is backgrounded so users
 * don't miss critical events (SOS, guardian requests, battery alerts).
 *
 * Uses @capacitor/local-notifications — no internet required, works offline.
 * Silently no-ops on web / iOS.
 */

let _appActive = true; // toggled by MainApp via setAppActive()
let _permissionGranted = null; // null = unknown, true/false after first check
let _plugin = null;
let _notifId = 100; // start above 0 to avoid conflicts

function nextId() {
  return _notifId++;
}

/** Called from MainApp's appStateChange listener */
export function setAppActive(active) {
  _appActive = !!active;
}

function isNative() {
  try {
    return typeof window !== 'undefined' &&
      !!window.Capacitor &&
      typeof window.Capacitor.isNativePlatform === 'function' &&
      window.Capacitor.isNativePlatform();
  } catch (_) { return false; }
}

async function plugin() {
  if (!isNative()) return null;
  if (_plugin) return _plugin;
  try {
    const { LocalNotifications } = await import('@capacitor/local-notifications');
    _plugin = LocalNotifications;
    return _plugin;
  } catch (_) { return null; }
}

async function ensurePermission() {
  if (_permissionGranted === true) return true;
  const p = await plugin();
  if (!p) return false;
  try {
    const status = await p.checkPermissions();
    if (status.display === 'granted') { _permissionGranted = true; return true; }
    const req = await p.requestPermissions();
    _permissionGranted = req.display === 'granted';
    return _permissionGranted;
  } catch (_) { return false; }
}

async function schedule(title, body, extra = {}) {
  const p = await plugin();
  if (!p) return;
  const ok = await ensurePermission();
  if (!ok) return;
  try {
    await p.schedule({
      notifications: [{
        id: nextId(),
        title,
        body,
        sound: 'default',
        smallIcon: 'ic_stat_notify',
        channelId: 'kinnect_alerts',
        extra,
      }]
    });
  } catch (_) {}
}

/** Create Android notification channels (call once at app start) */
export async function setupNotificationChannels() {
  const p = await plugin();
  if (!p) return;
  try {
    await p.createChannel({
      id: 'kinnect_alerts',
      name: 'Kinnect Alerts',
      description: 'SOS alerts, guardian requests, and safety notifications',
      importance: 5, // IMPORTANCE_HIGH
      visibility: 1, // VISIBILITY_PUBLIC
      sound: 'default',
      vibration: true,
      lights: true,
      lightColor: '#ef4444',
    });
  } catch (_) {}
}

/**
 * Notify a nearby (non-contact) user that someone within 5km triggered SOS.
 * Gentler than family SOS — no alarm, just an informational notification.
 */
export async function notifyProximitySOS(distanceKm) {
  if (_appActive) return;
  const dist = distanceKm < 1
    ? `${Math.round(distanceKm * 1000)} m`
    : `${distanceKm.toFixed(1)} km`;
  await schedule(
    'Nearby SOS Alert',
    `Someone ${dist} away has triggered an SOS. Tap to see if you can help.`,
    { type: 'proximity_sos' }
  );
}

/**
 * Notify that a contact triggered SOS.
 * Only fires when the app is backgrounded — in-app banner handles foreground.
 */
export async function notifySOS(displayName, reason) {
  if (_appActive) return;
  const r = reason ? ` — ${reason}` : '';
  await schedule(
    '🆘 SOS Alert',
    `${displayName} has triggered an SOS${r}. Tap to open Kinnect.`,
    { type: 'sos' }
  );
}

/**
 * Notify that someone sent a guardian request.
 */
export async function notifyGuardianRequest(displayName) {
  if (_appActive) return;
  await schedule(
    'Guardian Request',
    `${displayName} wants to be your guardian`,
    { type: 'guardian_request' }
  );
}

/**
 * Notify battery low proxy alert for a ward.
 */
export async function notifyBatteryLow(displayName, pct) {
  if (_appActive) return;
  await schedule(
    'Low Battery',
    `${displayName}'s battery is at ${pct}%`,
    { type: 'battery' }
  );
}

/**
 * Notify that a contact hasn't moved for a while (gentle alert from backend).
 */
export async function notifyHaventMoved(displayName, minutes) {
  if (_appActive) return;
  await schedule(
    'Check In',
    `${displayName} hasn't moved in ${minutes} minutes`,
    { type: 'gentle_alert' }
  );
}
