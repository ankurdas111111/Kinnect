// Hearth pebble palette — members ride the member wheel (--member-1..4),
// constant L/C so no member outshines another. Ember stays reserved for YOU.
// Faces are warm paper tinted with the member hue; the initial carries the hue.
export const AVATAR_PALETTES = [
  { bg: 'color-mix(in oklch, var(--member-1) 14%, var(--card))', color: 'var(--member-1)', solid: 'var(--member-1)' },
  { bg: 'color-mix(in oklch, var(--member-2) 14%, var(--card))', color: 'var(--member-2)', solid: 'var(--member-2)' },
  { bg: 'color-mix(in oklch, var(--member-3) 14%, var(--card))', color: 'var(--member-3)', solid: 'var(--member-3)' },
  { bg: 'color-mix(in oklch, var(--member-4) 14%, var(--card))', color: 'var(--member-4)', solid: 'var(--member-4)' },
];

export function getAvatarPalette(name) {
  const code = (name || '?').toUpperCase().charCodeAt(0);
  return AVATAR_PALETTES[code % AVATAR_PALETTES.length];
}

export function getAvatarStyle(name) {
  const p = getAvatarPalette(name);
  return `background: ${p.bg}; color: ${p.color};`;
}

export function getPresenceRingStyle(user) {
  if (user.sos?.active) return `box-shadow: 0 0 0 2.5px var(--danger-500), 0 0 12px color-mix(in oklch, var(--danger-500) 35%, transparent);`;
  if (user.online !== false) {
    const p = getAvatarPalette(user.displayName);
    return `box-shadow: 0 0 0 2.5px ${p.solid}, 0 0 0 5px var(--card), 0 0 12px color-mix(in oklch, ${p.solid} 27%, transparent);`;
  }
  return `box-shadow: 0 0 0 2px var(--status-offline); opacity: 0.6;`;
}
