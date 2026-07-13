/**
 * Member marker color grammar — VIGIL fixed wheel (Phase 1.4).
 *
 * Every family member hashes to one of 4 wheel slots. The wheel lives in
 * styles/tokens-oklch.css (--member-1..4: harbor 205° / cobalt 235° /
 * orchid 300° / plum 320° at constant L/C) and deliberately excludes:
 *   - brand 272±20°  (self marker stays var(--primary-500) — never a member hue)
 *   - success 160±20°, warning 80±15°, danger 25±25° (safety semantics)
 * H180 (retiring teal) and H110 (deuteranopia-confusable with success green)
 * were rejected by adversarial review — do not re-add them.
 *
 * Families with >4 members cycle the wheel; initials + ring shape are the
 * mandatory disambiguators. Hue is NEVER the only encoding.
 *
 * DOM consumers (markers, cards, badges) receive `var(--member-N)` strings —
 * live-resolved by CSS, so theme changes need no cache invalidation.
 * CANVAS consumers (GlobeCanvas, FamilyOrbit ctx.fillStyle / addColorStop)
 * cannot resolve var(): use resolveMemberColor()/CANVAS_DANGER, which return
 * canvas-safe hex twins of the tokens. The wheel is defined once in :root
 * (theme-invariant), so the twins are static constants.
 */

const WHEEL_SIZE = 4;

const MEMBER_HEX = ['#00a7ba', '#009bdd', '#9e77dc', '#b76ec7']; // raw-color-ok — canvas-safe hex twins; keep in sync with tokens-oklch.css
export const CANVAS_DANGER = '#ea3c3f'; // raw-color-ok — hex twin of dark --danger-500 for canvas SOS pins

/** Deterministic wheel slot (0-based) from a userId string. */
function memberIndex(userId) {
  let hash = 0;
  for (let i = 0; i < userId.length; i++) {
    hash = userId.charCodeAt(i) + ((hash << 5) - hash);
    hash |= 0; // 32-bit
  }
  return Math.abs(hash) % WHEEL_SIZE;
}

/**
 * Member color as a CSS token reference — for DOM styles only.
 * @param {string} userId
 * @returns {string} e.g. 'var(--member-2)'
 */
export function getUserColor(userId) {
  if (!userId) return 'var(--primary-500)';
  return `var(--member-${memberIndex(userId) + 1})`;
}

/**
 * Soft tint of the member color (was hsla 15% alpha) — DOM styles only.
 * @param {string} userId
 */
export function getUserColorLight(userId) {
  if (!userId) return 'color-mix(in oklch, var(--primary-500) 15%, transparent)';
  return `color-mix(in oklch, var(--member-${memberIndex(userId) + 1}) 15%, transparent)`;
}

/**
 * Luminous halo variant (rings, glows) — DOM styles only.
 * @param {string} userId
 */
export function getUserColorHalo(userId) {
  if (!userId) return 'var(--primary-400)';
  return `var(--member-${memberIndex(userId) + 1}-halo)`;
}

/**
 * Concrete hex for canvas/WebGL consumers (ctx.fillStyle, addColorStop) —
 * var() strings silently fail or throw there.
 * @param {string} userId
 */
export function resolveMemberColor(userId) {
  if (!userId) return MEMBER_HEX[0];
  return MEMBER_HEX[memberIndex(userId)];
}
