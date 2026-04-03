/**
 * Activity Status Layer — Feature 6
 * Derives a soft presence status from position data: Riding, Walking, At Rest,
 * Phone Inactive. No new sensors — pure math on the existing position stream.
 *
 * Speed is in m/s (as emitted by the Kalman-filtered position stream).
 * lastSeen is a Unix timestamp in milliseconds.
 */

// Speed thresholds (m/s)
const RIDE_MS  = 2.78;  // ≥ 10 km/h  — car, bus, train, bike
const WALK_MS  = 0.5;   // ≥ 1.8 km/h — on foot

// Time threshold — no update in 30 min → "Phone Inactive"
const INACTIVE_MS = 30 * 60 * 1000;

/**
 * @param {{ speed?: number, lastSeen?: number, online?: boolean }} user
 * @returns {{ label: string, color: string, dotColor: string } | null}
 */
export function computeActivityStatus(user) {
  if (!user) return null;

  const now     = Date.now();
  const lastSeen = user.lastSeen || 0;
  const ageMs   = lastSeen ? now - lastSeen : Infinity;
  const speed   = user.speed ?? 0; // m/s

  // Phone inactive — no signal for > 30 min regardless of online flag
  if (ageMs > INACTIVE_MS) {
    return { label: 'Phone Inactive', color: '#94a3b8', dotColor: '#cbd5e1' };
  }

  if (!user.online) {
    return { label: 'Offline', color: '#94a3b8', dotColor: '#cbd5e1' };
  }

  if (speed >= RIDE_MS) {
    return { label: 'In Transit', color: '#6366f1', dotColor: '#818cf8' };
  }

  if (speed >= WALK_MS) {
    return { label: 'Walking', color: '#10b981', dotColor: '#34d399' };
  }

  return { label: 'At Rest', color: '#f59e0b', dotColor: '#fbbf24' };
}

/**
 * Returns a human-readable age string: "just now", "5 min ago", "2h ago".
 * @param {number} lastSeen - Unix ms timestamp
 * @returns {string}
 */
export function formatActivityAge(lastSeen) {
  if (!lastSeen) return '';
  const ageSec = Math.floor((Date.now() - lastSeen) / 1000);
  if (ageSec < 10)  return 'just now';
  if (ageSec < 60)  return `${ageSec}s ago`;
  const ageMin = Math.floor(ageSec / 60);
  if (ageMin < 60)  return `${ageMin} min ago`;
  const ageHr = Math.floor(ageMin / 60);
  if (ageHr < 24)   return `${ageHr}h ago`;
  return `${Math.floor(ageHr / 24)}d ago`;
}
