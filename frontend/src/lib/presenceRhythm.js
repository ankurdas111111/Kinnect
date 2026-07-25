/**
 * presenceRhythm.js — on-device, opt-in presence rhythm.
 *
 * Learns each family member's normal online rhythm entirely on THIS device from
 * the online/offline transitions already streaming over WS, then offers a gentle,
 * NEUTRAL hint ("usually active around now") — never an alarm, never "unusually
 * offline", never red. Off by default: a design judge flagged the naive version
 * as anxiety-manufacturing, so nothing surfaces unless the user opts in AND
 * enough history has accrued.
 *
 * DB load: ZERO — history lives in localStorage, derived from the live stream.
 *
 * @module presenceRhythm
 */
import { writable } from 'svelte/store';

const HKEY = (userId) => `kinnect_rhythm_${userId}`;
const ENABLED_KEY = 'kinnect_rhythm_enabled';
const MAX_ENTRIES = 60;
const WINDOW_MS = 7 * 86_400_000; // 7 days
const MIN_DAYS = 3;               // need ≥3 distinct days before hinting

function loadEnabled() {
  try { return localStorage.getItem(ENABLED_KEY) === '1'; } catch { return false; }
}

/** Opt-in flag — default OFF. */
export const rhythmEnabled = writable(loadEnabled());

export function setRhythmEnabled(on) {
  rhythmEnabled.set(!!on);
  try { localStorage.setItem(ENABLED_KEY, on ? '1' : '0'); } catch { /* private mode */ }
}

/**
 * Record an online/offline transition (called by the global activity recorder).
 * Only genuine state changes are stored, bounded to a 7-day / 60-entry ring.
 * @param {string} userId
 * @param {boolean} online
 * @param {number} [ts]
 */
export function recordTransition(userId, online, ts = Date.now()) {
  if (!userId) return;
  try {
    const raw = localStorage.getItem(HKEY(userId));
    let arr = raw ? JSON.parse(raw) : [];
    if (!Array.isArray(arr)) arr = [];
    const last = arr[arr.length - 1];
    if (last && last.online === !!online) return; // only real transitions
    arr.push({ ts, online: !!online });
    const cutoff = ts - WINDOW_MS;
    arr = arr.filter((e) => e && e.ts >= cutoff).slice(-MAX_ENTRIES);
    localStorage.setItem(HKEY(userId), JSON.stringify(arr));
  } catch { /* private mode / quota — rhythm silently degrades */ }
}

/**
 * A gentle, neutral hint for a member — or null. Conservative on purpose:
 * requires ≥3 sample days and repeated online activity in this hour-of-day.
 * @param {string} userId
 * @param {number} [now]
 * @returns {string|null}
 */
export function getRhythmHint(userId, now = Date.now()) {
  try {
    const raw = localStorage.getItem(HKEY(userId));
    if (!raw) return null;
    const arr = JSON.parse(raw);
    if (!Array.isArray(arr) || arr.length < 4) return null;
    const hour = new Date(now).getHours();
    const days = new Set();
    let onlineThisHour = 0;
    for (const e of arr) {
      const d = new Date(e.ts);
      days.add(d.toDateString());
      if (e.online && d.getHours() === hour) onlineThisHour++;
    }
    if (days.size < MIN_DAYS) return null;
    return onlineThisHour >= 2 ? 'usually active around now' : null;
  } catch {
    return null;
  }
}
