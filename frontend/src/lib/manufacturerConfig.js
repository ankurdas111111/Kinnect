// Android manufacturer battery-optimization guidance.
//
// Aggressive OEM battery savers (MIUI, ColorOS, Funtouch, One UI) kill
// background location unless the app is whitelisted. We detect the maker from
// the WebView UA string and surface the exact settings path for that brand.
//
// Pure config + a pure detector — no side effects, no DOM, no stores.

/**
 * Detect the Android device manufacturer from the WebView UA string.
 * @param {string} [ua] Optional UA override (defaults to navigator.userAgent).
 * @returns {'miui' | 'coloros' | 'funtouch' | 'samsung' | 'generic'}
 */
export function detectAndroidManufacturer(ua) {
  const s = (ua ?? (typeof navigator !== 'undefined' ? navigator.userAgent : '')).toLowerCase();
  if (s.includes('xiaomi') || s.includes('redmi') || s.includes('miui')) return 'miui';
  if (s.includes('oppo') || s.includes('realme')) return 'coloros';
  if (s.includes('vivo')) return 'funtouch';
  if (s.includes('samsung')) return 'samsung';
  return 'generic';
}

/**
 * Per-manufacturer, step-by-step instructions for allowing unrestricted
 * background activity. `brand`/`steps` are null for the generic fallback,
 * which shows a plain explanatory paragraph instead of a numbered list.
 */
export const BATTERY_INSTRUCTIONS = {
  miui: {
    brand: 'Xiaomi / Redmi',
    steps: [
      'Open Settings → Apps',
      'Find and tap Kinnect',
      'Tap Battery Saver',
      'Select "No restrictions"',
    ],
  },
  coloros: {
    brand: 'Oppo / Realme',
    steps: [
      'Open Settings → App Management',
      'Find and tap Kinnect',
      'Tap Battery',
      'Enable "Allow background activity"',
    ],
  },
  funtouch: {
    brand: 'Vivo',
    steps: [
      'Open Settings → Battery',
      'Tap "Background power consumption"',
      'Find Kinnect → set to "Unrestricted"',
    ],
  },
  samsung: {
    brand: 'Samsung',
    steps: [
      'Open Settings → Apps → Kinnect',
      'Tap Battery → select "Unrestricted"',
      'Then tap "Allow background activity"',
    ],
  },
  generic: {
    brand: null,
    steps: null,
  },
};
