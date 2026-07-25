/**
 * daypart.js — time-of-day ambient temperature (emotionally-aware modes).
 *
 * Sets <html data-daypart="dawn|day|dusk|night">; styles/tokens-daypart.css
 * bends ONLY --nav-tint / --amb-warmth off it (≤8% mixes — text contrast can
 * never regress). Mirrors the effects.js store shape: attribute on <html> +
 * localStorage + init(). index.html pre-paints the attribute in the theme
 * IIFE so there is no tint flash on cold start.
 *
 * Off switch: SettingsPanel "Time-of-day tint" toggle → setDaypartEnabled().
 * Disabled = attribute removed = neutral base tokens.
 */
import { writable } from 'svelte/store';

const ENABLED_KEY = 'daypart-enabled'; // '1' (default) | '0'
const CHECK_MS = 60_000;

/** @param {number} h 0-23 → daypart bucket (keep in sync with index.html pre-paint) */
export function daypartFor(h) {
  if (h < 5) return 'night';
  if (h < 8) return 'dawn';
  if (h < 17) return 'day';
  if (h < 20) return 'dusk';
  return 'night';
}

function loadEnabled() {
  try { return localStorage.getItem(ENABLED_KEY) !== '0'; } catch { return true; }
}

const _daypart = writable('day');
/** Current daypart bucket (reactive) — for JS consumers like greetings. */
export const daypart = { subscribe: _daypart.subscribe };
/** Whether the tint layer is on (reactive) — drives the Settings toggle. */
export const daypartEnabled = writable(true);

let _enabled = true;
let _interval = null;

function apply() {
  if (typeof document === 'undefined') return;
  const dp = daypartFor(new Date().getHours());
  _daypart.set(dp);
  if (_enabled) {
    document.documentElement.setAttribute('data-daypart', dp);
  } else {
    document.documentElement.removeAttribute('data-daypart');
  }
}

/** Call once at startup (main.js, beside effects.init()). */
export function initDaypart() {
  _enabled = loadEnabled();
  daypartEnabled.set(_enabled);
  apply();
  if (typeof window === 'undefined') return;
  if (!_interval) _interval = setInterval(apply, CHECK_MS);
  // Boundary crossings usually happen while backgrounded — re-check on return.
  document.addEventListener('visibilitychange', () => {
    if (!document.hidden) apply();
  });
}

/** @param {boolean} on */
export function setDaypartEnabled(on) {
  _enabled = !!on;
  daypartEnabled.set(_enabled);
  try { localStorage.setItem(ENABLED_KEY, on ? '1' : '0'); } catch { /* private mode */ }
  apply();
}
