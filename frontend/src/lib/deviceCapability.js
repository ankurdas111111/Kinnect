/**
 * deviceCapability — one-shot probe that recommends a starting effects level.
 *
 * The redesign leans on backdrop-filter blur, WebGL canvases (GlobeCanvas,
 * FamilyOrbit) and particle fields. Those cost 15–30% FPS on mid-tier Android
 * (the daily Capacitor target), so we auto-pick a sane default the user can
 * still override in Settings via the effects store.
 *
 * Levels: 'full' | 'calm' | 'minimal'
 *   full    — everything on (capable desktop / high-end phone)
 *   calm    — reduced blur, no ambient particles, WebGL still allowed
 *   minimal — no blur, no WebGL, no ambient motion (low-end / data-saver / RM)
 */

/** @returns {boolean} true when the OS-level reduce-motion switch is on. */
export function prefersReducedMotion() {
  return typeof matchMedia === 'function'
    && matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/** @returns {boolean} true when the browser reports a data-saver preference. */
function prefersReducedData() {
  const c = typeof navigator !== 'undefined' && navigator.connection;
  if (!c) return false;
  return c.saveData === true || /^(slow-2g|2g|3g)$/.test(c.effectiveType || '');
}

/** Heuristic: is this a resource-constrained device? */
function isLowEndDevice() {
  if (typeof navigator === 'undefined') return false;
  const mem = navigator.deviceMemory;          // GB, Chrome/Android only
  const cores = navigator.hardwareConcurrency;  // logical cores
  if (typeof mem === 'number' && mem <= 4) return true;
  if (typeof cores === 'number' && cores <= 4) return true;
  return false;
}

/** @returns {boolean} running inside a Capacitor native shell. */
function isNativeShell() {
  try {
    // Capacitor injects a global; our stub returns false via isNativePlatform.
    return !!(globalThis.Capacitor && globalThis.Capacitor.isNativePlatform
      && globalThis.Capacitor.isNativePlatform());
  } catch {
    return false;
  }
}

/**
 * Probe once and return the recommended level. Cheap and synchronous —
 * call at startup before the store hydrates from localStorage.
 * @returns {'full'|'calm'|'minimal'}
 */
export function detectEffectsLevel() {
  if (prefersReducedMotion() || prefersReducedData()) return 'minimal';

  const native = isNativeShell();
  const lowEnd = isLowEndDevice();
  const coarse = typeof matchMedia === 'function'
    && matchMedia('(pointer: coarse)').matches;

  // Low-end anything → minimal. Native/coarse (phones) default to calm so the
  // daily map/list surfaces stay smooth; desktop pointers get the full show.
  if (lowEnd) return 'minimal';
  if (native || coarse) return 'calm';
  return 'full';
}

/**
 * True when the primary pointer is coarse (finger/stylus).
 * Use to gate hover-only decorations (cursor glow) — absent, not just paused.
 * Same matchMedia pattern as detectEffectsLevel above.
 * @returns {boolean}
 */
export function prefersCoarsePointer() {
  return typeof matchMedia === 'function'
    && matchMedia('(pointer: coarse)').matches;
}

/** WebGL is only worth attempting above 'minimal' and if the context exists. */
export function supportsWebGL() {
  if (typeof document === 'undefined') return false;
  try {
    const c = document.createElement('canvas');
    return !!(c.getContext('webgl') || c.getContext('experimental-webgl'));
  } catch {
    return false;
  }
}
