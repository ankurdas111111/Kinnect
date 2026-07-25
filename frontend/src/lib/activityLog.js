/**
 * activityLog.js — device-persistent family activity feed.
 *
 * Fixes the amnesiac Activity Feed (it used to reset to empty on every refresh).
 * Events are held in a localStorage-backed ring buffer keyed to this device, so
 * the feed and the Hub "recent" peek survive reloads and can say "earlier today".
 *
 * DB load: ZERO — pure device persistence. Nothing here touches the server.
 *
 * @module activityLog
 */
import { writable } from 'svelte/store';

const KEY = 'kinnect_activity_log';
const CAP = 80;                    // keep the most recent 80 events
const TTL_MS = 48 * 3_600_000;    // drop anything older than 48h

function load() {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return [];
    const arr = JSON.parse(raw);
    if (!Array.isArray(arr)) return [];
    const cutoff = Date.now() - TTL_MS;
    return arr.filter((e) => e && typeof e.ts === 'number' && e.ts >= cutoff).slice(0, CAP);
  } catch {
    return [];
  }
}

function persist(arr) {
  try {
    localStorage.setItem(KEY, JSON.stringify(arr.slice(0, CAP)));
  } catch {
    /* private mode / quota — feed simply degrades to in-memory for the session */
  }
}

/** Reactive event list, newest first. Hydrated synchronously from localStorage. */
export const activityEvents = writable(load());

let _seq = 0;

/**
 * Append one event. Deduped against the newest entry to avoid double-logging
 * when the same socket event fires twice in quick succession.
 * @param {{ type: string, userId?: string|null, userName?: string|null, message: string, severity?: string }} evt
 */
export function pushActivity(evt) {
  if (!evt || !evt.message) return;
  activityEvents.update((list) => {
    const head = list[0];
    if (head && head.type === evt.type && head.message === evt.message && Date.now() - head.ts < 2_000) {
      return list; // debounce identical back-to-back events
    }
    const next = [
      { id: `${Date.now()}_${_seq++}`, ts: Date.now(), severity: 'normal', ...evt },
      ...list,
    ].slice(0, CAP);
    persist(next);
    return next;
  });
}

/** Clear the entire persisted feed. */
export function clearActivity() {
  activityEvents.set([]);
  try {
    localStorage.removeItem(KEY);
  } catch {
    /* ignore */
  }
}
