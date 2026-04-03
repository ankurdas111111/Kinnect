import { writable } from 'svelte/store';

/** Number of notable events since Hub/Activity was last visited (max 9) */
export const hubBadgeCount = writable(0);

/** True when badge is driven by an active SOS (red, urgent) */
export const hubBadgeSos = writable(false);

export function bumpHubBadge(isSos = false) {
  hubBadgeCount.update(n => Math.min(n + 1, 9));
  if (isSos) hubBadgeSos.set(true);
}

export function clearHubBadge() {
  hubBadgeCount.set(0);
  hubBadgeSos.set(false);
}
