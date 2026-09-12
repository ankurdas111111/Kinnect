/**
 * Push subscription, callable from anywhere.
 *
 * Previously this lived only inside SettingsPanel, behind a toggle most people
 * never open. That made SOS push opt-in: the one alert that has to arrive when
 * the phone is in a pocket was the one nobody had turned on. Onboarding now
 * calls this straight after the location grant, while the reason is on screen.
 */
import { socket } from './socket.js';

let pendingVapidResolve = null;
let listenerBound = false;

function bindOnce() {
  if (listenerBound) return;
  listenerBound = true;
  socket.on('vapidKey', (payload) => {
    if (pendingVapidResolve) {
      pendingVapidResolve(payload);
      pendingVapidResolve = null;
    }
  });
}

function fetchVapidKey() {
  bindOnce();
  return new Promise((resolve) => {
    pendingVapidResolve = resolve;
    socket.emit('getVapidKey', {});
    setTimeout(() => {
      if (pendingVapidResolve === resolve) { pendingVapidResolve = null; resolve({ ok: false }); }
    }, 5000);
  });
}

function urlBase64ToUint8Array(base64String) {
  const padding = '='.repeat((4 - base64String.length % 4) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
  const raw = window.atob(base64);
  const arr = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) arr[i] = raw.charCodeAt(i);
  return arr;
}

/**
 * Subscribe this device to push if it is not already subscribed.
 *
 * @param {{ prompt?: boolean }} opts  prompt:false never shows the browser
 *        dialog — it only re-registers a device whose permission is already
 *        granted, so it is safe to call on every app start.
 * @returns {Promise<'subscribed'|'denied'|'unsupported'|'failed'>}
 */
export async function ensurePushSubscription({ prompt = false } = {}) {
  try {
    if (typeof Notification === 'undefined' || !('serviceWorker' in navigator)) return 'unsupported';
    if (Notification.permission === 'denied') return 'denied';
    if (Notification.permission === 'default') {
      if (!prompt) return 'denied';
      const perm = await Notification.requestPermission();
      if (perm !== 'granted') return 'denied';
    }

    const keyPayload = await fetchVapidKey();
    if (!keyPayload?.ok || !keyPayload.key) return 'failed';

    const reg = await Promise.race([
      navigator.serviceWorker.ready,
      new Promise((_, rej) => setTimeout(() => rej(new Error('sw-timeout')), 6000)),
    ]);

    // Reuse an existing subscription; re-emitting it is how a reinstalled
    // server-side record gets restored.
    let sub = await reg.pushManager.getSubscription();
    if (!sub) {
      sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(keyPayload.key),
      });
    }
    const json = sub.toJSON();
    socket.emit('pushSubscribe', { endpoint: json.endpoint, keys: json.keys });
    return 'subscribed';
  } catch {
    return 'failed';
  }
}
