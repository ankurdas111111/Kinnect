/**
 * Haptic feedback — uses @capacitor/haptics on Android native for crisp
 * tactile response, falls back to navigator.vibrate on web/desktop.
 *
 * iOS is intentionally excluded (Apple blocks WebView vibration and Capacitor
 * haptics on iOS requires different entitlements).
 */

// Lazily-loaded Capacitor Haptics — avoids import cost on web
let _hapticPlugin = null;
let _hapticReady = false;

async function getHaptics() {
  if (_hapticReady) return _hapticPlugin;
  _hapticReady = true;
  try {
    if (typeof window !== 'undefined' && window.Capacitor?.isNativePlatform?.()) {
      const { Haptics, ImpactStyle, NotificationType } = await import('@capacitor/haptics');
      _hapticPlugin = { Haptics, ImpactStyle, NotificationType };
    }
  } catch (_) {}
  return _hapticPlugin;
}

// Warm up import on first call so subsequent calls are instant
getHaptics().catch(() => {});

/** Light impact — button presses, selections */
async function light() {
  const h = await getHaptics();
  if (h) { await h.Haptics.impact({ style: h.ImpactStyle.Light }).catch(() => {}); return; }
  try { navigator.vibrate?.(10); } catch (_) {}
}

/** Medium impact — confirmations, toggle on */
async function medium() {
  const h = await getHaptics();
  if (h) { await h.Haptics.impact({ style: h.ImpactStyle.Medium }).catch(() => {}); return; }
  try { navigator.vibrate?.([10, 40, 10]); } catch (_) {}
}

/** Heavy impact — destructive actions, errors */
async function heavy() {
  const h = await getHaptics();
  if (h) { await h.Haptics.impact({ style: h.ImpactStyle.Heavy }).catch(() => {}); return; }
  try { navigator.vibrate?.([50, 80, 30]); } catch (_) {}
}

/** Success notification pattern */
async function success() {
  const h = await getHaptics();
  if (h) { await h.Haptics.notification({ type: h.NotificationType.Success }).catch(() => {}); return; }
  try { navigator.vibrate?.([10, 30, 10, 30, 20]); } catch (_) {}
}

/** Warning notification pattern */
async function warning() {
  const h = await getHaptics();
  if (h) { await h.Haptics.notification({ type: h.NotificationType.Warning }).catch(() => {}); return; }
  try { navigator.vibrate?.([50, 80, 30]); } catch (_) {}
}

/** Error notification pattern */
async function error() {
  const h = await getHaptics();
  if (h) { await h.Haptics.notification({ type: h.NotificationType.Error }).catch(() => {}); return; }
  try { navigator.vibrate?.([30, 20, 30, 20, 30]); } catch (_) {}
}

/** SOS — urgent, unmistakable long vibration */
async function sos() {
  const h = await getHaptics();
  if (h) {
    // Three heavy impacts with gaps — unmistakable
    for (let i = 0; i < 3; i++) {
      await h.Haptics.impact({ style: h.ImpactStyle.Heavy }).catch(() => {});
      if (i < 2) await new Promise(r => setTimeout(r, 120));
    }
    return;
  }
  try { navigator.vibrate?.([200, 100, 200, 100, 200]); } catch (_) {}
}

/** SOS cancelled — relief pattern */
async function sosCancelled() {
  const h = await getHaptics();
  if (h) { await h.Haptics.notification({ type: h.NotificationType.Success }).catch(() => {}); return; }
  try { navigator.vibrate?.([80, 40, 40, 40, 20]); } catch (_) {}
}

/** Room notification */
async function notify() {
  const h = await getHaptics();
  if (h) { await h.Haptics.notification({ type: h.NotificationType.Success }).catch(() => {}); return; }
  try { navigator.vibrate?.([20, 60, 20]); } catch (_) {}
}

export const haptics = {
  tap:          () => light().catch(() => {}),
  confirm:      () => medium().catch(() => {}),
  success:      () => success().catch(() => {}),
  warning:      () => warning().catch(() => {}),
  notify:       () => notify().catch(() => {}),
  sos:          () => sos().catch(() => {}),
  sosCancelled: () => sosCancelled().catch(() => {}),
  error:        () => error().catch(() => {}),
};
