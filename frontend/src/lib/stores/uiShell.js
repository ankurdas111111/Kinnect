import { writable } from 'svelte/store';

/**
 * Intent-first tab ids (2026 nav): what the user is trying to DO, not the
 * feature name. family = "Is everyone OK?", map = "Where is everyone?",
 * share = "Share me", help = "Get help", me = "Me".
 */
export const TAB_IDS = ['family', 'map', 'share', 'help', 'me'];

/** Pre-rename ids still normalize — migration safety net for stray callers. */
const LEGACY_TAB_MAP = { track: 'map', people: 'family', safety: 'help' };

const initialState = {
  mobileTab: 'map',
  sheetOpen: false
};

export const uiShellStore = writable(initialState);

export function setMobileTab(tab) {
  const t = LEGACY_TAB_MAP[tab] || tab;
  uiShellStore.update((state) => ({ ...state, mobileTab: TAB_IDS.includes(t) ? t : 'map' }));
}

export function setSheetOpen(open) {
  uiShellStore.update((state) => ({ ...state, sheetOpen: !!open }));
}

export function resetUiShell() {
  uiShellStore.set(initialState);
}
