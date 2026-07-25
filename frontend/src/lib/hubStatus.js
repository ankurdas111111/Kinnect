/**
 * hubStatus.js — Pure, side-effect-free derivations for the Family Hub.
 *
 * No stores, no DOM. Callers pass `now` (ms) so the hot paths stay pure and
 * unit-testable (the one Date-tier branch lives in presence.formatAge).
 *
 * FIELD-NAME CONTRACT (verified against backend cache.SanitizeUser):
 *   The live `otherUsers` entries carry `latitude` / `longitude` / `lastUpdate`
 *   (unix ms) and a nested `sos: { active }`. The lean `userMoved` payload uses
 *   `lat` / `lng` / `sosActive`. normMember() below reconciles BOTH shapes so the
 *   Hub never silently reads an undefined coordinate again.
 *
 * @module hubStatus
 */

import { formatAge } from './presence.js';

// ── Freshness tiers (ms) ─────────────────────────────────────────────────────
// Aligns with presence.FRESHNESS at the shared boundaries, extended for the Hub.
export const FRESH_MS = 30_000;       // < 30s  → fresh   (live green)
export const AGING_MS = 5 * 60_000;   // < 5m   → aging   (amber)
export const STALE_MS = 30 * 60_000;  // < 30m  → stale   (muted)
                                      // ≥ 30m  → silent  (needs attention if online)

/**
 * Normalize a raw otherUsers entry to one stable shape, tolerant of both the
 * SanitizeUser (`latitude`/`longitude`/`sos.active`) and lean-userMoved
 * (`lat`/`lng`/`sosActive`) field conventions.
 *
 * @param {Record<string, any>} u
 * @returns {null | {
 *   userId: string, socketId: string, displayName: string, online: boolean,
 *   speed: number, lat: number|null, lng: number|null,
 *   lastUpdate: number|null, sosActive: boolean, batteryPct: number|null
 * }}
 */
export function normMember(u) {
  if (!u) return null;
  const lat = u.latitude ?? u.lat ?? null;
  const lng = u.longitude ?? u.lng ?? null;
  const lastUpdate = u.lastUpdate ?? u.lastSeen ?? u.timestamp ?? null;
  const sosActive = u?.sos?.active ?? u.sosActive ?? false;
  return {
    userId: u.userId,
    socketId: u.socketId,
    displayName: u.displayName || 'Unknown',
    online: u.online !== false, // default online unless explicitly false
    speed: u.speed || 0,        // m/s
    lat, lng, lastUpdate, sosActive,
    batteryPct: u.batteryPct ?? null,
  };
}

/**
 * Freshness tier for a member's last update.
 * @param {number|null} lastUpdate unix ms
 * @param {number} now unix ms
 * @returns {'fresh'|'aging'|'stale'|'silent'|'unknown'}
 */
export function freshness(lastUpdate, now) {
  if (!lastUpdate) return 'unknown';
  const age = now - lastUpdate;
  if (age < 0) return 'fresh';
  if (age < FRESH_MS) return 'fresh';
  if (age < AGING_MS) return 'aging';
  if (age < STALE_MS) return 'stale';
  return 'silent';
}

/**
 * Presence class for a normalized member.
 * @param {ReturnType<normMember>} m
 * @param {number} now
 * @returns {'sos'|'offline'|'silent'|'moving'|'settled'}
 */
export function presenceOf(m, now) {
  if (!m) return 'offline';
  if (m.sosActive) return 'sos';
  if (!m.online) return 'offline';
  if (freshness(m.lastUpdate, now) === 'silent') return 'silent';
  if (m.speed > 1) return 'moving'; // > 1 m/s ≈ 3.6 km/h → actually moving
  return 'settled';
}

/** Human ETA phrasing. @param {number|null} sec @returns {string} */
export function fmtEta(sec) {
  if (sec == null || sec < 0) return '';
  if (sec < 60) return `${Math.max(1, Math.round(sec))} sec`;
  const m = Math.round(sec / 60);
  if (m < 60) return `${m} min`;
  const h = Math.floor(m / 60);
  const r = m % 60;
  return r ? `${h}h ${r}m` : `${h}h`;
}

/**
 * The Verdict — one honest sentence about the whole family, via a fixed rules
 * ladder: SOS › silent-but-online › arriving › moving › settled › empty.
 * Pure compute over already-in-memory state. No DB, no side effects.
 *
 * @param {{
 *   members: Array<ReturnType<normMember>>,
 *   mySosActive?: boolean,
 *   arrivals?: Map<string, {displayName?:string, placeName?:string, etaSeconds?:number}>,
 *   now: number
 * }} input
 * @returns {{ tone:'safe'|'caution'|'alert', word:string, sentence:string, detail:string }}
 */
export function computeVerdict({ members = [], mySosActive = false, arrivals = null, now }) {
  const list = members.filter(Boolean);

  // 1 ── SOS wins over everything
  const sos = list.filter((m) => m.sosActive);
  if (mySosActive || sos.length) {
    const who = mySosActive ? 'You' : sos[0].displayName;
    return {
      tone: 'alert',
      word: 'SOS',
      sentence: mySosActive ? 'Your SOS is active' : `${who} needs help`,
      detail: sos.length + (mySosActive ? 1 : 0) > 1 ? `${sos.length + (mySosActive ? 1 : 0)} active alerts` : 'Open the map',
    };
  }

  // 2 ── Someone marked online but gone silent (phone asleep / lost signal)
  const silent = list.filter((m) => m.online && presenceOf(m, now) === 'silent');
  if (silent.length) {
    const m = silent[0];
    return {
      tone: 'caution',
      word: silent.length > 1 ? `${silent.length} quiet` : m.displayName,
      sentence:
        silent.length > 1
          ? `${silent.length} people haven't updated in a while`
          : `${m.displayName} hasn't updated in a while`,
      detail: m.lastUpdate ? `last seen ${formatAge(now - m.lastUpdate)}` : 'no recent signal',
    };
  }

  // 3 ── Someone converging on a saved place — the reassuring "almost home"
  const arr = [...(arrivals?.values?.() || [])]
    .filter((a) => a && a.etaSeconds != null)
    .sort((a, b) => a.etaSeconds - b.etaSeconds);
  if (arr.length) {
    const a = arr[0];
    const name = a.displayName || 'Someone';
    return {
      tone: 'safe',
      word: name,
      sentence: `${name} is heading to ${a.placeName || 'a saved place'}`,
      detail: `about ${fmtEta(a.etaSeconds)} away`,
    };
  }

  // 4 ── People on the move
  const moving = list.filter((m) => presenceOf(m, now) === 'moving');
  const online = list.filter((m) => m.online);
  if (moving.length) {
    return {
      tone: 'safe',
      word: moving.length === 1 ? moving[0].displayName : String(moving.length),
      sentence:
        moving.length === 1 ? `${moving[0].displayName} is on the move` : `${moving.length} people are on the move`,
      detail: `${online.length} online`,
    };
  }

  // 5 ── Empty circle
  if (list.length === 0) {
    return { tone: 'safe', word: 'quiet', sentence: 'Your circle is quiet', detail: 'Invite family to see them here' };
  }

  // 6 ── Everyone settled
  return {
    tone: 'safe',
    word: 'settled',
    sentence: online.length ? "Everyone's settled" : 'All quiet right now',
    detail: online.length ? `${online.length} online · all accounted for` : 'No one is sharing right now',
  };
}
