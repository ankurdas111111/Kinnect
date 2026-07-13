/**
 * presence.js — Shared freshness + connection-state language.
 *
 * Pure module: no DOM, no stores, no side effects. Unit-testable.
 * Zero imports.
 *
 * DELETES-IN-WAITING (adoption phase — do not act now):
 *   LiveViewer.svelte:46–52  — inline connState derivation → replace with deriveConnState()
 *   LiveViewer.svelte:115    — hardcoded 8000ms timeout → replace with CONNECTION_TIMEOUT_MS
 *   WatchViewer.svelte:27–32 — inline connState derivation → replace with deriveConnState()
 *   WatchViewer.svelte:48    — hardcoded 8000ms timeout → replace with CONNECTION_TIMEOUT_MS
 *   ActivityFeed relTime     — inline relative-time formatting → replace with formatAge()
 *   CheckinSchedule formatRelative — → replace with formatAge()
 *   Monitoring last-update stamps  — → replace with formatAge()
 *
 * @module presence
 */

// ── Constants ────────────────────────────────────────────────────────────────

/**
 * Init-timeout both viewers currently hardcode (LiveViewer.svelte:115,
 * WatchViewer.svelte:48). Consumers pass this to setTimeout and surface
 * connState 'issue' when it fires before first init.
 *
 * @type {number}
 */
export const CONNECTION_TIMEOUT_MS = 8000;

/**
 * FreshnessChip thresholds (FreshnessChip.svelte:22–29):
 *   live  → age < LIVE_MS  (< 5 s)
 *   stale → age < STALE_MS (< 30 s)
 *   old   → age ≥ STALE_MS (≥ 30 s)
 *
 * @type {{ LIVE_MS: number, STALE_MS: number }}
 */
export const FRESHNESS = { LIVE_MS: 5_000, STALE_MS: 30_000 };

// ── deriveConnState ──────────────────────────────────────────────────────────

/**
 * Derive a canonical connection state from the four boolean signals that both
 * viewers track. Precedence is fixed and documented below — do not reorder.
 *
 * SOS is NOT a connection state (see §2 of CONTRACTS.md). WatchViewer's
 * sosActive is an orthogonal safety state; deriveConnState never returns 'sos'.
 *
 * Precedence:
 *   1. issue        → 'issue'      (init timeout, invalid/expired link, banner error)
 *   2. online && initialized → 'live'
 *   3. connecting || !initialized  → 'connecting'
 *   4. else         → 'offline'
 *
 * Mapping to existing consumers (unification contract):
 *
 * | Consumer                       | initialized | online   | issue                    | connecting                                |
 * |--------------------------------|-------------|----------|--------------------------|-------------------------------------------|
 * | LiveViewer (was :46–52)        | hasInit     | online   | !!connectionIssue        | statusText ∋ connect/reconnect/error/retry|
 * | WatchViewer (was :27–32)       | hasInit     | hasInit  | bannerSos && !sosActive  | !hasInit                                  |
 *
 * @param {{ initialized?: boolean, online?: boolean, issue?: boolean, connecting?: boolean }} flags
 * @returns {'live' | 'connecting' | 'issue' | 'offline'}
 */
export function deriveConnState({
  initialized = false,
  online = false,
  issue = false,
  connecting = false,
} = {}) {
  if (issue) return 'issue';
  if (online && initialized) return 'live';
  if (connecting || !initialized) return 'connecting';
  return 'offline';
}

// ── formatAge ────────────────────────────────────────────────────────────────

/**
 * Format an age (caller computes `now - lastSeenMs`) as a human-readable
 * string. Stays pure and testable — no Date.now() inside.
 *
 * Vocabulary matches FreshnessChip's label logic (FreshnessChip.svelte:31–38)
 * exactly at the shared boundaries, extended with an h and date tier:
 *
 * | age ms                  | output              |
 * |-------------------------|---------------------|
 * | null / undefined / < 0  | ''                  |
 * | < 5 000                 | 'just now'          |
 * | < 60 000                | '32s ago'           |
 * | < 3 600 000             | '5m ago'            |
 * | < 86 400 000            | '2h ago'            |
 * | ≥ 86 400 000            | '9 Jul'             |
 *
 * @param {number | null | undefined} ms — age in milliseconds (must be ≥ 0 for a real age)
 * @returns {string}
 */
export function formatAge(ms) {
  if (ms == null || ms < 0) return '';

  if (ms < 5_000) return 'just now';

  if (ms < 60_000) {
    return Math.floor(ms / 1_000) + 's ago';
  }

  if (ms < 3_600_000) {
    return Math.floor(ms / 60_000) + 'm ago';
  }

  if (ms < 86_400_000) {
    return Math.floor(ms / 3_600_000) + 'h ago';
  }

  // Date tier: needs an absolute wall-clock reference, so this one branch
  // calls Date.now(). All sub-day tiers (the testable hot paths) stay pure.
  const d = new Date(Date.now() - ms);
  return d.toLocaleDateString(undefined, { day: 'numeric', month: 'short' });
}
