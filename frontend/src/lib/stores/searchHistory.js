/**
 * searchHistory — writable store for the last 10 selected destinations.
 *
 * Each entry: { name, sub, lat, lng, mode, ts }
 * Persisted to localStorage key 'kinnect_search_history'.
 * Deduped by name + rounded coords (1m precision). Newest first.
 */
import { writable } from 'svelte/store';

const LS_KEY = 'kinnect_search_history';
const MAX_ENTRIES = 10;

function loadHistory() {
  try {
    const raw = typeof localStorage !== 'undefined' ? localStorage.getItem(LS_KEY) : null;
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

function saveHistory(entries) {
  try {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(LS_KEY, JSON.stringify(entries));
    }
  } catch {}
}

/** Round coord to ~1m precision for dedup */
function roundCoord(n) {
  return Math.round(n * 10000) / 10000;
}

function createSearchHistory() {
  const { subscribe, set, update } = writable(loadHistory());

  return {
    subscribe,
    /**
     * Record a destination selection. Dedupes by name + rounded coords.
     * @param {{ name: string, sub?: string, lat: number, lng: number, mode?: string }} entry
     */
    add(entry) {
      update(hist => {
        const fresh = {
          name: entry.name || '',
          sub: entry.sub || '',
          lat: entry.lat,
          lng: entry.lng,
          mode: entry.mode || 'car',
          ts: Date.now(),
        };
        const deduped = hist.filter(h =>
          !(h.name === fresh.name &&
            roundCoord(h.lat) === roundCoord(fresh.lat) &&
            roundCoord(h.lng) === roundCoord(fresh.lng))
        );
        const next = [fresh, ...deduped].slice(0, MAX_ENTRIES);
        saveHistory(next);
        return next;
      });
    },
    clear() {
      set([]);
      try {
        if (typeof localStorage !== 'undefined') localStorage.removeItem(LS_KEY);
      } catch {}
    },
  };
}

export const searchHistory = createSearchHistory();
