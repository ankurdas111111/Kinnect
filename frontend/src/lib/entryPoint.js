/**
 * Where a logged-out visitor goes.
 *
 * The landing page is marketing: it exists to explain Kinnect to someone who
 * has never seen it. Two groups must never be shown it.
 *
 *   Native (Android/iOS) — they already installed the app. Pitching it to them
 *   is nonsense, and an app-store reviewer opening a marketing page instead of
 *   a login screen is a rejection risk.
 *
 *   Anyone who has signed in on this browser before — an expired session is not
 *   a reason to re-pitch the product. They want the login form.
 *
 * Everyone else is a genuine first-time web visitor, and gets the landing.
 */
import { isNativePlatform } from './geoProvider.js';

const RETURNING_KEY = 'kinnect_returning';

/** Call after any successful sign-in or registration. */
export function markReturningVisitor() {
  try { localStorage.setItem(RETURNING_KEY, '1'); } catch { /* private mode */ }
}

export function isReturningVisitor() {
  try { return localStorage.getItem(RETURNING_KEY) === '1'; } catch { return false; }
}

/** Route for a visitor with no live session: '/landing' or '/login'. */
export function loggedOutRoute() {
  if (isNativePlatform()) return '/login';
  return isReturningVisitor() ? '/login' : '/landing';
}

/** True when the landing page must not be shown at all on this platform. */
export function landingIsBlocked() {
  return isNativePlatform();
}
