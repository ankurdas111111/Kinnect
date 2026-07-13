/**
 * mapPaint.js — instance-scoped MapLibre paint color resolver.
 *
 * MapLibre paint values are plain JS strings, so CSS custom properties never
 * reach them — they silently keep whatever literal hex was hardcoded and would
 * survive a token/palette swap. This module resolves brand tokens to concrete
 * colors ONCE via getComputedStyle, re-resolves on theme change, and asserts a
 * non-empty canary so a missing token fails loud instead of painting invisible.
 *
 * Usage (per map instance — each page that owns its own map subscribes itself):
 *   import { createMapPaint } from '../lib/mapPaint.js';
 *   const paint = createMapPaint(() => repaint());   // callback fires on theme change
 *   paint.colors.route  // resolved string, safe for MapLibre paint
 *   paint.destroy()     // in onDestroy — drops the theme subscription
 */

import { themeStore } from './stores/theme.js';

/**
 * Token → hex fallback map. The fallbacks are the resolved VIGIL brand values;
 * they are the safety net if getComputedStyle returns empty (SSR / detached el).
 * The var() name is the source of truth once the DOM is live.
 */
const PAINT_TOKENS = {
  route: { varName: '--primary-500', fallback: '#5863d3' }, /* raw-color-ok — mapPaint fallback, unreachable once tokens resolve */
  start: { varName: '--success-500', fallback: '#2fc183' }, /* raw-color-ok — mapPaint fallback */
  end:   { varName: '--danger-500',  fallback: '#ea3c3f' }, /* raw-color-ok — mapPaint fallback */
  dot:   { varName: '--primary-400', fallback: '#90a6f7' }, /* raw-color-ok — mapPaint fallback */
  /* stroke ring around start/end markers — reads the surface so it works in both themes */
  stroke: { varName: '--surface-0', fallback: '#ffffff' },  /* raw-color-ok — mapPaint fallback */
};

/** Resolve one token from computed styles on :root, trimmed; fallback if empty. */
function resolveToken(rootStyle, varName, fallback) {
  const raw = rootStyle.getPropertyValue(varName).trim();
  return raw || fallback;
}

/**
 * @param {() => void} [onThemeChange] — called after colors re-resolve on theme swap.
 * @returns {{ colors: Record<string,string>, resolve: () => void, destroy: () => void }}
 */
export function createMapPaint(onThemeChange) {
  const colors = {};

  function resolve() {
    if (typeof window === 'undefined' || typeof document === 'undefined') {
      // No DOM — seed fallbacks so callers never read undefined.
      for (const key in PAINT_TOKENS) colors[key] = PAINT_TOKENS[key].fallback;
      return;
    }
    const rootStyle = getComputedStyle(document.documentElement);
    for (const key in PAINT_TOKENS) {
      const { varName, fallback } = PAINT_TOKENS[key];
      colors[key] = resolveToken(rootStyle, varName, fallback);
    }
    // Canary: the route color must never be empty — an empty paint string paints
    // an invisible line and hides the whole route. Fail loud into the fallback.
    if (!colors.route) colors.route = PAINT_TOKENS.route.fallback;
  }

  resolve();

  // Re-resolve on theme change; skip the initial synchronous emit.
  let primed = false;
  const unsub = themeStore.subscribe(() => {
    if (!primed) { primed = true; return; }
    resolve();
    onThemeChange?.();
  });

  return {
    colors,
    resolve,
    destroy() { unsub(); },
  };
}
