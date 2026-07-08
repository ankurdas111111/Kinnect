// Premium avatar palette — richer, more saturated for social feel.
// `color`/`solid` reference design tokens (original hexes kept as var() fallbacks)
// so avatars follow the token system. Entries without a matching token keep
// their original hex value.
export const AVATAR_PALETTES = [
  { bg: 'linear-gradient(135deg, rgba(99,102,241,0.22) 0%, rgba(79,70,229,0.14) 100%)',   color: 'var(--indigo-400, #818cf8)',  solid: 'var(--indigo-500, #6366f1)' },   // indigo
  { bg: 'linear-gradient(135deg, rgba(16,185,129,0.22) 0%, rgba(5,150,105,0.14) 100%)',   color: 'var(--success-400, #34d399)', solid: 'var(--success-500, #10b981)' },  // emerald
  { bg: 'linear-gradient(135deg, rgba(245,158,11,0.22) 0%, rgba(217,119,6,0.14) 100%)',   color: 'var(--warning-400, #fbbf24)', solid: 'var(--warning-500, #f59e0b)' },  // amber
  { bg: 'linear-gradient(135deg, rgba(239,68,68,0.20) 0%, rgba(220,38,38,0.12) 100%)',    color: 'var(--danger-400, #f87171)',  solid: 'var(--danger-500, #ef4444)' },   // red
  { bg: 'linear-gradient(135deg, rgba(139,92,246,0.22) 0%, rgba(124,58,237,0.14) 100%)',  color: 'var(--violet-400, #a78bfa)',  solid: '#8b5cf6' },                      // violet
  { bg: 'linear-gradient(135deg, rgba(6,182,212,0.20) 0%, rgba(8,145,178,0.12) 100%)',    color: 'var(--cyan-400, #22d3ee)',    solid: '#06b6d4' },                      // cyan
  { bg: 'linear-gradient(135deg, rgba(251,113,133,0.20) 0%, rgba(225,29,72,0.12) 100%)',  color: 'var(--rose-400, #fb7185)',    solid: '#e11d48' },                      // rose
  { bg: 'linear-gradient(135deg, rgba(52,211,153,0.22) 0%, rgba(16,185,129,0.14) 100%)',  color: '#6ee7b7',                     solid: 'var(--success-400, #34d399)' },  // teal
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
  if (user.sos?.active) return `box-shadow: 0 0 0 2.5px var(--danger-500, #ef4444), 0 0 0 5px rgba(239,68,68,0.15), 0 0 16px rgba(239,68,68,0.35);`;
  if (user.online !== false) {
    const p = getAvatarPalette(user.displayName);
    // Translucent glow derived from the token color (was `${p.solid}44` hex-alpha concat)
    return `box-shadow: 0 0 0 2.5px ${p.solid}, 0 0 0 5px rgba(0,0,0,0.5), 0 0 12px color-mix(in srgb, ${p.solid} 27%, transparent);`;
  }
  return `box-shadow: 0 0 0 2px rgba(107,114,128,0.3); opacity: 0.6;`;
}
