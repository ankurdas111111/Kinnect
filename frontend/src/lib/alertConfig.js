/**
 * alertConfig.js — canonical constants for the saved-places alert system.
 *
 * LOCALSTORAGE_KEYS: same key names as the original 'kinnect_place_alerts' /
 * 'kinnect_speed_alerts' strings — no migration needed; keeping names identical
 * means no silent data loss for existing users.
 *
 * ICON_MAP: maps icon value → { emoji, label } so every emoji get an
 * accessible label (WCAG 2.2 AA, CONTRACTS.md §0).
 *
 * DEFAULT_SPEED_KMH: sensible default speed-alert threshold.
 */

/** localStorage keys — keep consistent with SavedPlacesPanel.svelte reads. */
export const LOCALSTORAGE_KEYS = {
  PLACE_ALERTS: 'kinnect_place_alerts',
  SPEED_ALERTS: 'kinnect_speed_alerts',
};

/**
 * Icon map: value → { emoji, label, textFallback }.
 * textFallback: ultra-short text shown when the emoji does not render (old
 * Android, strict-text contexts). Always color+shape+text, never emoji alone.
 */
export const ICON_MAP = {
  home:   { emoji: '🏠', label: 'Home',   textFallback: 'Home'   },
  work:   { emoji: '💼', label: 'Work',   textFallback: 'Work'   },
  school: { emoji: '🏫', label: 'School', textFallback: 'School' },
  gym:    { emoji: '🏋️', label: 'Gym',    textFallback: 'Gym'    },
  pin:    { emoji: '📍', label: 'Other',  textFallback: 'Place'  },
};

/** Sorted icon options for select menus. */
export const ICON_OPTIONS = Object.entries(ICON_MAP).map(([value, meta]) => ({
  value,
  label: meta.label,
}));

/** Default threshold for new speed alerts (km/h). */
export const DEFAULT_SPEED_KMH = 80;
