/**
 * verdict.js — the shared family-verdict domain store.
 *
 * One store per domain (CLAUDE.md): this file owns "how is the family doing
 * right now" plus the nav badge grammar derived from it. The pure compute
 * stays in lib/hubStatus.js — this is only the reactive wiring, so Navbar,
 * BottomTabBar, Sidebar, MobileTopBar, FamilyDashboard, and the voice layer
 * all read ONE source of truth instead of re-deriving locally.
 *
 * DB load: ZERO — derives exclusively from in-memory WS-fed stores.
 */
import { readable, derived } from 'svelte/store';
import { otherUsers } from './map.js';
import { mySosActive, activeSosUsers } from './sos.js';
import { arrivalProjections } from './arrivals.js';
import { hubBadgeCount } from './hubBadge.js';
import { normMember, computeVerdict } from '../hubStatus.js';

/**
 * Shared 10s heartbeat for freshness/verdict recompute. readable() start/stop
 * means the interval exists ONLY while someone subscribes — zero timers on
 * public viewer routes and the login page.
 */
export const verdictNow = readable(Date.now(), (set) => {
  const id = setInterval(() => set(Date.now()), 10_000);
  return () => clearInterval(id);
});

/** Normalized member list — shared by FamilyDashboard, FamilyPanel, rosters. */
export const familyMembers = derived(otherUsers, ($u) =>
  [...$u.values()].map(normMember).filter(Boolean)
);

/**
 * The live verdict. Equality-gated: otherUsers notifies on every GPS fix, but
 * nav chrome only re-renders when the verdict MEANINGFULLY changes (tone /
 * sentence / word / detail) — state transitions, not coordinates.
 */
let _prev = null;
export const familyVerdict = derived(
  [familyMembers, mySosActive, arrivalProjections, verdictNow],
  ([$members, $sos, $arrivals, $now], set) => {
    const next = computeVerdict({ members: $members, mySosActive: $sos, arrivals: $arrivals, now: $now });
    if (
      !_prev ||
      _prev.tone !== next.tone ||
      _prev.sentence !== next.sentence ||
      _prev.word !== next.word ||
      _prev.detail !== next.detail
    ) {
      _prev = next;
      set(next);
    }
  },
  { tone: 'safe', word: 'quiet', sentence: '', detail: '' }
);

/**
 * Badge grammar for the Family entry (nav tabs, Navbar Hub button, sidebar):
 *   tone  = ambient state (tint/dot) — safe shows nothing (calm by default)
 *   count = unread events bubble ('1'..'9+' or null)
 *   urgent = pulse gate — alert tone only
 */
export const familyBadge = derived([familyVerdict, hubBadgeCount], ([$v, $n]) => ({
  tone: $v.tone,
  count: $n > 0 ? ($n > 9 ? '9+' : String($n)) : null,
  urgent: $v.tone === 'alert',
}));

/** Badge for the Help entry: red dot while any SOS is live. */
export const helpBadge = derived([mySosActive, activeSosUsers], ([$mine, $m]) => ({
  active: $mine || $m.size > 0,
  mine: $mine,
}));
