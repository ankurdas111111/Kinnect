<script>
  import { createEventDispatcher } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { authUser } from '../lib/stores/auth.js';
  import { tracking } from '../lib/stores/map.js';
  import { socket } from '../lib/socket.js';
  import { apiPost } from '../lib/api.js';
  import { privacyPause } from '../lib/stores/places.js';
  import { activeSosUsers } from '../lib/stores/sos.js';
  import { familyBadge } from '../lib/stores/verdict.js';
  import ThemeToggle from './ThemeToggle.svelte';

  let ghostMode = $derived($privacyPause && $privacyPause > Date.now());
  // One badge grammar: count = unread bubble, tone = tint, pulse = urgent only
  let hubBadge  = $derived($familyBadge.count || ($activeSosUsers.size > 0 ? '!' : null));
  let hubBadgeIsUrgent = $derived($familyBadge.urgent);
  let hubTone = $derived($familyBadge.tone);

  /**
   * @typedef {Object} Props
   * @property {boolean} [isAdmin]
   * @property {any} [activePanel]
   * @property {boolean} [isTracking]
   */

  /** @type {Props} */
  let { isAdmin = false, activePanel = null, isTracking = false } = $props();

  const dispatch = createEventDispatcher();

  function toggle(panel) { dispatch('togglePanel', panel); }
  function toggleTracking() { dispatch('toggleTracking'); }

  async function logout() {
    await apiPost('/api/logout');
    window.location.hash = '#/login';
    window.location.reload();
  }

  let initials = $derived($authUser ? ($authUser.displayName || '').split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2) : '');
</script>

<nav class="navbar navbar-inner" aria-label="Main navigation" data-ghost-mode={ghostMode ? 'true' : 'false'}>
  <div class="navbar-left">
    <!-- Dashboard shortcut -->
    <button
      class="nav-dashboard-btn"
      class:has-badge={hubBadge}
      class:tone-caution={hubTone === 'caution'}
      class:tone-alert={hubTone === 'alert'}
      onclick={() => push('/dashboard')}
      title="Family Hub"
      aria-label="Open family hub{hubTone === 'alert' ? ' — needs attention' : hubTone === 'caution' ? ' — check in' : ''}{hubBadge ? ` (${hubBadge} new)` : ''}"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
      <span class="nav-dashboard-label">Hub</span>
      {#if hubBadge}
        <span class="hub-badge" class:hub-badge-urgent={hubBadgeIsUrgent} aria-label="{hubBadge} new events">{hubBadge}</span>
      {/if}
    </button>
    <!-- Circular gradient logo — glows on tracking -->
    <div class="navbar-logo" class:logo-live={isTracking} aria-hidden="true">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="10" r="3"/><path d="M12 2a8 8 0 0 0-8 8c0 1.892.402 3.13 1.5 4.5L12 22l6.5-7.5c1.098-1.37 1.5-2.608 1.5-4.5a8 8 0 0 0-8-8z"/></svg>
    </div>
    <div class="navbar-brand">
      <span class="navbar-title verdict-voice">Kinnect</span>
      {#if isTracking}
        <span class="navbar-context-live"><span class="context-dot"></span>Live</span>
      {:else if ghostMode}
        <span class="navbar-context-ghost">Ghost</span>
      {/if}
    </div>
  </div>

  <div class="navbar-right">
    <!-- Group 1: Social/Info -->
    <div class="nav-group">
      <button class="nav-btn tactile" class:active={activePanel === 'sharing'} onclick={() => toggle('sharing')} title="Share" aria-label="Toggle share panel" aria-pressed={activePanel === 'sharing'}>
        <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6"/><line x1="12" y1="2" x2="12" y2="15"/></svg>
      </button>
      <button class="nav-btn tactile" onclick={() => toggle('family')} title="Family" aria-label="Open family panel">
        <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
      </button>
    </div>

    <div class="nav-divider" aria-hidden="true"></div>

    <!-- Group 2: Safety -->
    <button class="nav-btn nav-btn-safety tactile" class:active={activePanel === 'admin'} onclick={() => toggle('admin')} title="Help" aria-label="Toggle help and safety panel" aria-pressed={activePanel === 'admin'}>
      <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
    </button>

    <div class="nav-divider" aria-hidden="true"></div>

    <!-- Group 3: Settings -->
    <div class="nav-group">
      <button class="nav-btn tactile" class:active={activePanel === 'settings'} onclick={() => toggle('settings')} title="Me" aria-label="Toggle profile and settings panel" aria-pressed={activePanel === 'settings'}>
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
      </button>
      {#if isAdmin}
        <button class="nav-btn tactile" class:active={activePanel === 'superAdmin'} onclick={() => toggle('superAdmin')} title="Super Admin" aria-label="Toggle super admin panel" aria-pressed={activePanel === 'superAdmin'}>
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
        </button>
      {/if}
      <!-- Theme picker entry — was orphaned in the intent-first nav rebuild -->
      <ThemeToggle />
    </div>

    <!-- Premium track pill -->
    <button class="track-pill" class:live={isTracking} onclick={toggleTracking} aria-label={isTracking ? 'Stop tracking' : 'Start tracking'}>
      {#if isTracking}
        <span class="rec-dot animate-rec-blink" aria-hidden="true"></span>
        Live · Stop
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><polygon points="5 3 19 12 5 21 5 3"/></svg>
        Track
      {/if}
    </button>

    <div class="navbar-avatar" class:avatar-live={isTracking} title={$authUser?.displayName || ''} aria-label="User avatar">{initials}</div>

    <!-- Footer: sign-out, visually separated + danger-tinted -->
    <div class="nav-divider" aria-hidden="true"></div>
    <div class="navbar-footer">
      <button class="nav-btn nav-btn-logout tactile" onclick={logout} title="Logout" aria-label="Logout">
        <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
      </button>
    </div>
  </div>
</nav>

<style>
  /* ── Top strip — warm paper anchor, quiet round controls (Hearth) ─────── */
  .navbar {
    /* Bar keeps its 56px content row; a notch inset grows the bar, never
       squeezes the controls (safe-top is 0 in desktop browsers). */
    height: calc(var(--navbar-height) + var(--safe-top, 0px));
    display: flex;
    align-items: center;
    padding: 0 var(--space-4);
    padding-top: var(--safe-top, 0px);
    background: var(--paper-warm);
    border-bottom: 1px solid var(--border-subtle);
    box-shadow: var(--shadow-xs);
    z-index: var(--z-navbar);
    position: relative;
    flex-shrink: 0;
    gap: var(--space-2);
  }

  .navbar-left {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-shrink: 0;
  }

  /* Dashboard / Hub shortcut — quiet ember capsule */
  .nav-dashboard-btn {
    min-height: 44px;
    padding: 0 var(--space-3) 0 var(--space-2-5);
    border-radius: var(--radius-full);
    background: color-mix(in oklch, var(--primary-500) 10%, transparent);
    border: 1px solid color-mix(in oklch, var(--primary-500) 25%, transparent);
    color: var(--primary-700);
    display: flex;
    align-items: center;
    gap: var(--space-1);
    cursor: pointer;
    flex-shrink: 0;
    transition: background var(--duration-fast) var(--ease-out), color var(--duration-fast) var(--ease-out), transform var(--duration-fast) var(--ease-out);
  }
  .nav-dashboard-btn:hover {
    background: color-mix(in oklch, var(--primary-500) 18%, transparent);
  }
  .nav-dashboard-btn:active { transform: scale(0.96); animation: none; }
  .nav-dashboard-label {
    font-family: var(--font-sans);
    font-size: var(--text-2xs);
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }
  /* Verdict tone tint — border/ink follow the family state (badge grammar:
     tone = tint, count = bubble, pulse = urgent only). Static by design. */
  .nav-dashboard-btn.tone-caution {
    animation: none;
    background: color-mix(in oklch, var(--warning-500) 12%, transparent);
    border-color: color-mix(in oklch, var(--warning-500) 40%, transparent);
    color: var(--warning-700);
  }
  .nav-dashboard-btn.tone-alert {
    animation: none;
    background: color-mix(in oklch, var(--danger-500) 12%, transparent);
    border-color: color-mix(in oklch, var(--danger-500) 45%, transparent);
    color: var(--danger-600);
  }

  /* Notification badge — ochre "worth a look"; vermilion only when the count
     is riding an active SOS (urgent). */
  .hub-badge {
    min-width: 18px;
    height: 18px;
    border-radius: var(--radius-full);
    padding: 0 var(--space-1);
    font-size: var(--text-2xs);
    font-weight: 700;
    color: var(--text-inverse);
    background: var(--status-stale);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    line-height: 1;
    font-family: var(--font-sans);
    border: 1.5px solid var(--paper-warm);
    letter-spacing: 0;
  }
  .hub-badge.hub-badge-urgent {
    background: var(--danger-500);
    animation: badge-urgent-pulse 1.6s ease-in-out infinite;
  }
  @keyframes badge-urgent-pulse {
    0%, 100% { box-shadow: 0 0 0 0 color-mix(in oklch, var(--danger-500) 40%, transparent); }
    50%       { box-shadow: 0 0 0 4px color-mix(in oklch, var(--danger-500) 0%, transparent); }
  }

  /* Ember pebble logo — one warm circle, no gradients */
  .navbar-logo {
    width: 34px;
    height: 34px;
    border-radius: var(--radius-full);
    background: var(--primary-500);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-on-primary);
    flex-shrink: 0;
    box-shadow: var(--shadow-sm);
    transition: box-shadow var(--duration-normal) var(--ease-out);
  }

  .navbar-logo.logo-live {
    box-shadow: var(--shadow-primary);
  }

  .navbar-brand {
    display: flex;
    flex-direction: column;
    gap: 0;
    line-height: 1;
  }

  /* Brand — the verdict voice speaks the name (serif italic, ember ink) */
  .navbar-title {
    font-size: var(--text-xl);
    font-weight: 700;
    color: var(--primary-700);
    letter-spacing: -0.01em;
    line-height: 1.1;
  }

  .navbar-context-live {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    font-family: var(--font-sans);
    font-size: var(--text-2xs);
    font-weight: 700;
    color: var(--success-700);
    letter-spacing: 0.05em;
    text-transform: uppercase;
    line-height: 1;
  }

  .context-dot {
    display: inline-block;
    width: 6px;
    height: 6px;
    border-radius: var(--radius-full);
    background: var(--status-live);
    animation: aurora-pulse 2s ease-in-out infinite;
    flex-shrink: 0;
  }

  .navbar-context-ghost {
    font-family: var(--font-sans);
    font-size: var(--text-2xs);
    font-weight: 700;
    color: var(--text-tertiary);
    letter-spacing: 0.05em;
    text-transform: uppercase;
    line-height: 1;
  }

  .navbar-right {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    flex: 1;
    overflow-x: auto;
    overflow-y: hidden;
    -webkit-overflow-scrolling: touch;
    min-width: 0;
    padding-right: var(--space-1);
    scrollbar-width: none;
  }

  .navbar-right::-webkit-scrollbar { display: none; }

  /* Nav groups — visually cluster related buttons */
  .nav-group {
    display: flex;
    align-items: center;
    gap: 2px;
    flex-shrink: 0;
  }

  .nav-divider {
    width: 1px;
    height: 18px;
    background: var(--border-subtle);
    margin: 0 var(--space-1);
    flex-shrink: 0;
  }

  /* Footer cluster — trailing sign-out, pushed to the far end */
  .navbar-footer {
    display: flex;
    align-items: center;
    flex-shrink: 0;
    margin-left: auto;
  }

  /* Individual nav buttons — 44px round paper targets */
  .nav-btn {
    position: relative;
    width: 44px;
    height: 44px;
    border-radius: var(--radius-full);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
    background: transparent;
    border: none;
    cursor: pointer;
    flex-shrink: 0;
    transition:
      color var(--duration-fast) var(--ease-out),
      background var(--duration-fast) var(--ease-out),
      transform var(--duration-fast) var(--ease-out);
  }

  .nav-btn:hover {
    color: var(--text-primary);
    background: var(--surface-3);
  }

  .nav-btn:active {
    transform: scale(0.94);
    transition-duration: var(--duration-fast);
  }

  .nav-btn:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 2px;
  }

  /* Active destination — filled ember-light pill, the reference's grammar */
  .nav-btn.active {
    color: var(--primary-700);
    background: var(--primary-100);
  }

  /* Safety shares the single-accent grammar; no second hue competes */
  .nav-btn-safety.active {
    color: var(--primary-700);
    background: var(--primary-100);
  }

  /* Sign-out — ochre "pause" tint on intent; never vermilion */
  .nav-btn-logout:hover {
    color: var(--warning-700);
    background: color-mix(in oklch, var(--warning-500) 12%, transparent);
  }

  /* ── Track pill — the one filled ember action on the strip ───────────── */
  .track-pill {
    border-radius: var(--radius-full);
    min-height: 44px;
    padding: var(--space-2) var(--space-4);
    font-family: var(--font-sans);
    font-size: var(--text-sm);
    font-weight: 700;
    letter-spacing: -0.005em;
    display: inline-flex;
    align-items: center;
    gap: var(--space-1-5);
    white-space: nowrap;
    cursor: pointer;
    border: none;
    flex-shrink: 0;
    transition:
      transform var(--duration-fast) var(--ease-out),
      box-shadow var(--duration-normal) var(--ease-out),
      background var(--duration-normal) var(--ease-out);
  }
  .track-pill:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 2px;
  }

  /* Idle: filled ember — begin accompanying */
  .track-pill:not(.live) {
    background: var(--primary-500);
    color: var(--text-on-primary);
    box-shadow: var(--shadow-primary);
  }

  .track-pill:not(.live):hover {
    background: var(--primary-600);
    transform: translateY(-1px);
  }

  /* Live: settled sage — quietly present */
  .track-pill.live {
    background: var(--success-500);
    color: var(--text-inverse);
    box-shadow: 0 4px 16px color-mix(in oklch, var(--success-500) 28%, transparent);
  }

  .track-pill:active {
    transform: scale(0.96);
    transition-duration: var(--duration-fast);
  }

  /* Recording dot — paper-bright breathing point */
  .rec-dot {
    display: inline-block;
    width: 7px;
    height: 7px;
    border-radius: var(--radius-full);
    background: color-mix(in oklch, var(--text-inverse) 95%, transparent);
    flex-shrink: 0;
    animation: recording-blink 1.2s ease-in-out infinite;
  }

  /* Avatar — warm ember-tinted pebble, sage ring when live */
  .navbar-avatar {
    width: 34px;
    height: 34px;
    border-radius: var(--radius-full);
    background: var(--primary-100);
    color: var(--primary-700);
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-sans);
    font-weight: 700;
    font-size: var(--text-sm);
    flex-shrink: 0;
    user-select: none;
    border: 1.5px solid color-mix(in oklch, var(--primary-500) 30%, transparent);
    transition:
      box-shadow var(--duration-normal) var(--ease-out),
      border-color var(--duration-normal) var(--ease-out);
    cursor: default;
  }

  /* Live: sage presence ring */
  .avatar-live {
    border-color: var(--success-500);
    box-shadow:
      0 0 0 2px var(--success-500),
      0 0 0 4px var(--paper-warm);
    animation: aurora-pulse 2.5s ease-in-out infinite;
  }

  /* Ghost mode: desaturate + violet tint the navbar */
  [data-ghost-mode="true"] {
    filter: saturate(0.25) hue-rotate(20deg) brightness(0.80);
    transition: filter 600ms var(--ease-out);
  }
  [data-ghost-mode="false"] {
    filter: none;
    transition: filter 600ms var(--ease-out);
  }

  @media (prefers-reduced-motion: reduce) {
    .context-dot,
    .rec-dot,
    .avatar-live,
    .hub-badge.hub-badge-urgent { animation: none; }
    .nav-btn:active,
    .track-pill:not(.live):hover,
    .track-pill:active,
    .nav-dashboard-btn:active { transform: none; }
  }

  /* Mobile: hide desktop navbar */
  @media (max-width: 767px) {
    .navbar {
      display: none;
    }
  }

  /* Tablet: hide title */
  @media (min-width: 768px) and (max-width: 1023px) {
    .navbar-title {
      display: none;
    }
  }
</style>
