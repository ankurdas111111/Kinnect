/**
 * constellationGeometry — default node/link geometry for Constellation.svelte.
 *
 * This is the Login brand-panel deco-map (Login.svelte:142–173) lifted verbatim:
 * same viewBox (0 0 340 280) and same pin coordinates, so Constellation's
 * adoption diff on Login is geometry-neutral. Kept in a sibling module purely
 * to hold Constellation.svelte under its 200-line budget.
 *
 * Node `hue` is intentionally left undefined here → the component falls back to
 * `var(--primary-400)`. Member hues (Phase 1.4 marker grammar) are injected by
 * the consumer as CSS token strings; this default set stays token-agnostic.
 */

export const VIEWBOX = '0 0 340 280';

/**
 * Default dormant geometry.
 * Index 0 = self/center pin; 1..4 = family-member pins.
 * @type {{ x: number, y: number, hue?: string, state: 'unlit'|'igniting'|'live'|'converging', label?: string }[]}
 */
export const DEFAULT_NODES = [
  { x: 180, y: 130, state: 'live' },              // self (center)
  { x: 90, y: 80, state: 'live' },                // member 1
  { x: 260, y: 90, state: 'live' },               // member 2
  { x: 140, y: 200, state: 'live' },              // member 3
  { x: 290, y: 170, state: 'live' },              // member 4
];

/** Index pairs into DEFAULT_NODES — the dashed connection lines. */
export const DEFAULT_LINKS = [
  [1, 0],
  [0, 2],
  [0, 3],
  [2, 4],
];

/** Decorative background grid dots (non-interactive, non-linked). */
export const GRID_DOTS = [
  { x: 50, y: 160 },
  { x: 120, y: 240 },
  { x: 220, y: 220 },
  { x: 310, y: 50 },
  { x: 40, y: 40 },
];
