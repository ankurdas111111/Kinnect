/**
 * viewTransition.js — Gated navigation transitions.
 *
 * Wraps push() / replace() from svelte-spa-router so that programmatic
 * navigations participate in the View Transitions API — exactly like the
 * anchor-click handler in App.svelte. Without this, store-driven redirects
 * (Login.svelte:117–119 and others) bypass startViewTransition entirely,
 * killing the brand-beacon handoff animation.
 *
 * The gate is computed at CALL TIME (not module load) so that changes to
 * the effects store or OS preference are always respected:
 *   1. Browser supports document.startViewTransition
 *   2. OS reduce-motion switch is OFF  (live matchMedia check via deviceCapability.js)
 *   3. Effects store is NOT 'minimal'  (allowMotion from lib/stores/effects.js)
 *
 * If any gate fails, navigate() is IDENTICAL to push() / replace() — no added
 * latency, no extra frames, no ::view-transition pseudo-elements created.
 *
 * Error-safety: any exception inside the transition machinery is caught and
 * the navigation is retried instantly so auth-critical paths never dead-end
 * on a decoration failure.
 *
 * IMPORTANT: This is the single wrapping point for startViewTransition in
 * the codebase. No other new file should call document.startViewTransition
 * for routing purposes.
 *
 * Exports:
 *   navigate(path, opts?) → Promise<void>
 *     opts.replace — use replace() instead of push() (default: false)
 *
 * @module viewTransition
 */

import { push, replace } from 'svelte-spa-router';
import { get } from 'svelte/store';
import { prefersReducedMotion } from './deviceCapability.js';
import { allowMotion } from './stores/effects.js';

/**
 * Navigate to `path`, wrapping the router call in startViewTransition when
 * the environment supports it and the user has not opted out of motion.
 *
 * Fire-and-forget safe: await is optional.
 *
 * @param {string} path — SPA route, e.g. '/' or '/add-contact/...'
 * @param {{ replace?: boolean }} [opts]
 * @returns {Promise<void>}
 */
export async function navigate(path, opts = {}) {
  /** The router function to delegate to (push or replace). */
  const routerNav = opts.replace ? replace : push;

  // Evaluate all three gates at call time — never at module load.
  const useTransition =
    typeof document !== 'undefined' &&
    typeof document.startViewTransition === 'function' &&
    !prefersReducedMotion() &&
    get(allowMotion);

  if (!useTransition) {
    // Instant fallback — behaviorally identical to calling push()/replace() directly.
    return routerNav(path);
  }

  try {
    // Start the transition; resolve as soon as the navigation callback finishes.
    // We intentionally do NOT await transition.finished — decoration must never
    // block routing (a slow cross-fade should not delay auth redirects).
    const transition = document.startViewTransition(() => routerNav(path));
    // Await only the navigation callback completion, not the animation.
    await transition.ready;
  } catch (_) {
    // Safety net: if startViewTransition or the router call throws, retry the
    // navigation instantly without transitions so the user is never stranded.
    return routerNav(path);
  }
}
