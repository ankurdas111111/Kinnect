import { writable } from 'svelte/store';

/**
 * Number of notable events since Hub/Activity was last visited (max 9).
 *
 * Badge grammar note: this store carries ONLY the unread count. Urgency and
 * tone now live in lib/stores/verdict.js (familyBadge.tone/.urgent, derived
 * from the live family verdict) — the old hubBadgeSos flag was retired when
 * nav chrome became verdict-aware.
 */
export const hubBadgeCount = writable(0);

/** @param {boolean} [isSos] kept for caller compatibility — urgency is verdict-driven now */
export function bumpHubBadge(isSos = false) {
  hubBadgeCount.update(n => Math.min(n + 1, 9));
}

export function clearHubBadge() {
  hubBadgeCount.set(0);
}
