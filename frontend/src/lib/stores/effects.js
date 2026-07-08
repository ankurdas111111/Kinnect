import { writable, derived } from 'svelte/store';
import { detectEffectsLevel, supportsWebGL } from '../deviceCapability.js';

/**
 * effects — global "calm mode" / performance-budget store.
 *
 * One source of truth for how heavy the UI may render. Drives the `data-fx`
 * attribute on <html> (CSS reads it to scale blur/animation — see
 * styles/tokens-fx.css) and gates expensive primitives (GlobeCanvas,
 * FamilyOrbit, ParticleField) in JS via the derived helpers below.
 *
 * Levels: 'full' | 'calm' | 'minimal'
 */

export const FX_LEVELS = ['full', 'calm', 'minimal'];
const STORAGE_KEY = 'fx-level';

function readStored() {
  if (typeof localStorage === 'undefined') return null;
  const v = localStorage.getItem(STORAGE_KEY);
  return FX_LEVELS.includes(v) ? v : null;
}

function apply(level) {
  if (typeof document !== 'undefined') {
    document.documentElement.setAttribute('data-fx', level);
  }
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(STORAGE_KEY, level);
  }
}

function createEffectsStore() {
  const { subscribe, set } = writable('full');

  return {
    subscribe,

    /** Hydrate from storage, else auto-detect from device capability. */
    init() {
      const level = readStored() || detectEffectsLevel();
      apply(level);
      set(level);
    },

    /** Explicitly choose a level (Settings toggle). */
    set(level) {
      if (!FX_LEVELS.includes(level)) return;
      apply(level);
      set(level);
    },

    /** Re-run capability detection (e.g. user cleared their preference). */
    reset() {
      if (typeof localStorage !== 'undefined') localStorage.removeItem(STORAGE_KEY);
      const level = detectEffectsLevel();
      apply(level);
      set(level);
    },
  };
}

export const effects = createEffectsStore();

// ── Derived capability gates — subscribe to these in components ──────────────
const webglOk = supportsWebGL();

/** Ambient WebGL scenes (GlobeCanvas, FamilyOrbit) allowed? full only. */
export const allowWebGL = derived(effects, ($fx) => webglOk && $fx === 'full');

/** Ambient particle fields allowed? full only. */
export const allowParticles = derived(effects, ($fx) => $fx === 'full');

/** Heavy backdrop blur allowed? full + calm (minimal falls back to solid). */
export const allowHeavyBlur = derived(effects, ($fx) => $fx !== 'minimal');

/** Decorative (non-essential) motion allowed? disabled in minimal. */
export const allowMotion = derived(effects, ($fx) => $fx !== 'minimal');
