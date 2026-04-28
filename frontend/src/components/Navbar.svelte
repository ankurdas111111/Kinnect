<script>
  import { createEventDispatcher } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { authUser } from '../lib/stores/auth.js';
  import { tracking } from '../lib/stores/map.js';
  import { socket } from '../lib/socket.js';
  import { apiPost } from '../lib/api.js';
  import { privacyPause } from '../lib/stores/places.js';
  import { activeSosUsers } from '../lib/stores/sos.js';
  import { hubBadgeCount, hubBadgeSos } from '../lib/stores/hubBadge.js';

  $: ghostMode = $privacyPause && $privacyPause > Date.now();
  $: hubBadge  = $hubBadgeCount > 0 ? ($hubBadgeCount > 9 ? '9+' : String($hubBadgeCount)) : ($activeSosUsers.size > 0 ? '!' : null);
  $: hubBadgeIsUrgent = $hubBadgeSos || $activeSosUsers.size > 0;

  export let isAdmin = false;
  export let activePanel = null;
  export let isTracking = false;

  const dispatch = createEventDispatcher();

  function toggle(panel) { dispatch('togglePanel', panel); }
  function toggleTracking() { dispatch('toggleTracking'); }

  async function logout() {
    await apiPost('/api/logout');
    window.location.hash = '#/login';
    window.location.reload();
  }

  $: initials = $authUser ? ($authUser.displayName || '').split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2) : '';
</script>

<nav class="navbar navbar-inner" aria-label="Main navigation" data-ghost-mode={ghostMode ? 'true' : 'false'}>
  <div class="navbar-left">
    <!-- Dashboard shortcut -->
    <button
      class="nav-dashboard-btn"
      class:has-badge={hubBadge}
      on:click={() => push('/dashboard')}
      title="Family Dashboard"
      aria-label="Open family dashboard{hubBadge ? ` (${hubBadge} new)` : ''}"
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
      <span class="navbar-title">Kinnect</span>
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
      <button class="nav-btn" class:active={activePanel === 'sharing'} on:click={() => toggle('sharing')} title="Sharing" aria-label="Toggle sharing panel" aria-pressed={activePanel === 'sharing'}>
        <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6"/><line x1="12" y1="2" x2="12" y2="15"/></svg>
      </button>
      <button class="nav-btn" class:active={activePanel === 'users'} on:click={() => toggle('users')} title="People" aria-label="Toggle users panel" aria-pressed={activePanel === 'users'}>
        <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
      </button>
      <button class="nav-btn" class:active={activePanel === 'info'} on:click={() => toggle('info')} title="Signal" aria-label="Toggle info panel" aria-pressed={activePanel === 'info'}>
        <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
      </button>
    </div>

    <div class="nav-divider" aria-hidden="true"></div>

    <!-- Group 2: Safety -->
    <button class="nav-btn nav-btn-safety" class:active={activePanel === 'admin'} on:click={() => toggle('admin')} title="Safety" aria-label="Toggle safety panel" aria-pressed={activePanel === 'admin'}>
      <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
    </button>

    <div class="nav-divider" aria-hidden="true"></div>

    <!-- Group 3: Settings -->
    <div class="nav-group">
      <button class="nav-btn" class:active={activePanel === 'settings'} on:click={() => toggle('settings')} title="Settings" aria-label="Toggle settings panel" aria-pressed={activePanel === 'settings'}>
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
      </button>
      {#if isAdmin}
        <button class="nav-btn" class:active={activePanel === 'superAdmin'} on:click={() => toggle('superAdmin')} title="Super Admin" aria-label="Toggle super admin panel" aria-pressed={activePanel === 'superAdmin'}>
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
        </button>
      {/if}
    </div>

    <!-- Premium track pill -->
    <button class="track-pill" class:live={isTracking} on:click={toggleTracking} aria-label={isTracking ? 'Stop tracking' : 'Start tracking'}>
      {#if isTracking}
        <span class="rec-dot animate-rec-blink" aria-hidden="true"></span>
        Live · Stop
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><polygon points="5 3 19 12 5 21 5 3"/></svg>
        Track
      {/if}
    </button>

    <div class="navbar-avatar" class:avatar-live={isTracking} title={$authUser?.displayName || ''} aria-label="User avatar">{initials}</div>

    <button class="nav-btn" on:click={logout} title="Logout" aria-label="Logout">
      <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
    </button>
  </div>
</nav>

<style>
  .navbar {
    height: var(--navbar-height);
    display: flex;
    align-items: center;
    padding: 0 var(--space-4);
    /* 3D glass surface with depth */
    background: var(--glass-3d);
    backdrop-filter: var(--glass-3d-blur);
    -webkit-backdrop-filter: var(--glass-3d-blur);
    border-bottom: 1px solid var(--glass-3d-border);
    border-top: 1px solid rgba(255, 255, 255, 0.15);
    box-shadow:
      var(--elevation-2),
      inset 0 1px 0 rgba(255, 255, 255, 0.12),
      inset 0 -1px 0 rgba(0, 0, 0, 0.05);
    z-index: var(--z-navbar);
    position: relative;
    flex-shrink: 0;
    gap: var(--space-2);
    transform-style: preserve-3d;
  }

  .navbar-left {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-shrink: 0;
  }

  /* Dashboard / Hub shortcut button */
  .nav-dashboard-btn {
    height: 30px;
    padding: 0 10px 0 8px;
    border-radius: 20px;
    background: rgba(99,102,241,0.13);
    border: 1px solid rgba(99,102,241,0.32);
    color: #a5b4fc;
    display: flex;
    align-items: center;
    gap: 5px;
    cursor: pointer;
    flex-shrink: 0;
    transition: background 0.15s ease, color 0.15s ease, transform 0.12s ease, box-shadow 0.2s ease;
    box-shadow: 0 0 0 0 rgba(99,102,241,0.4);
    animation: hub-pulse 3s ease-in-out infinite;
  }
  .nav-dashboard-btn:hover {
    background: rgba(99,102,241,0.24);
    color: #c7d2fe;
    transform: scale(1.04);
    box-shadow: 0 0 0 3px rgba(99,102,241,0.18);
    animation: none;
  }
  .nav-dashboard-btn:active { transform: scale(0.94); animation: none; }
  .nav-dashboard-label {
    font-family: var(--font-display);
    font-size: 11px;
    font-weight: 800;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }
  @keyframes hub-pulse {
    0%, 100% { box-shadow: 0 0 0 0 rgba(99,102,241,0); border-color: rgba(99,102,241,0.32); }
    50%       { box-shadow: 0 0 0 3px rgba(99,102,241,0.12); border-color: rgba(99,102,241,0.55); }
  }

  /* Stop ambient pulse when there's a live badge — the badge speaks for itself */
  .nav-dashboard-btn.has-badge { animation: none; }

  /* Notification badge */
  .hub-badge {
    min-width: 16px;
    height: 16px;
    border-radius: 8px;
    padding: 0 4px;
    font-size: 9px;
    font-weight: 800;
    color: #fff;
    background: #f59e0b;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    line-height: 1;
    font-family: var(--font-display, system-ui);
    border: 1.5px solid rgba(0, 0, 0, 0.55);
    letter-spacing: 0;
  }
  .hub-badge.hub-badge-urgent {
    background: #ef4444;
    animation: badge-urgent-pulse 1.6s ease-in-out infinite;
  }
  @keyframes badge-urgent-pulse {
    0%, 100% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.4); }
    50%       { box-shadow: 0 0 0 4px rgba(239, 68, 68, 0); }
  }

  /* Circular gradient logo — 3D sphere-like */
  .navbar-logo {
    width: 34px;
    height: 34px;
    border-radius: 50%;
    background: linear-gradient(135deg, var(--primary-400), var(--primary-600), var(--accent-guardian));
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    flex-shrink: 0;
    /* 3D sphere effect with layered shadows */
    box-shadow:
      var(--glow-primary),
      0 4px 12px rgba(99, 102, 241, 0.40),
      inset 0 2px 4px rgba(255, 255, 255, 0.25),
      inset 0 -3px 6px rgba(0, 0, 0, 0.20);
    transform-style: preserve-3d;
    transition: box-shadow 400ms var(--ease-out), transform 300ms var(--ease-3d-spring);
    animation: float-3d 8s ease-in-out infinite;
  }

  .navbar-logo.logo-live {
    animation: logo-celebrate 600ms var(--ease-out) both;
    box-shadow: var(--glow-live), var(--shadow-sm);
  }

  .navbar-brand {
    display: flex;
    flex-direction: column;
    gap: 0;
    line-height: 1;
  }

  /* Brand title — display font for premium feel */
  .navbar-title {
    font-family: var(--font-display);
    font-size: 1.0625rem; /* 17px */
    font-weight: 800;
    color: var(--text-primary);
    letter-spacing: -0.025em;
    line-height: 1.1;
  }

  .navbar-context-live {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-family: var(--font-display);
    font-size: 0.6875rem; /* 11px — slightly larger than 9px */
    font-weight: 700;
    color: var(--success-500);
    letter-spacing: 0.05em;
    text-transform: uppercase;
    line-height: 1;
  }

  .context-dot {
    display: inline-block;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--success-500);
    animation: aurora-pulse 2s ease-in-out infinite;
    flex-shrink: 0;
    box-shadow: 0 0 6px rgba(16, 185, 129, 0.60);
  }

  .navbar-context-ghost {
    font-family: var(--font-display);
    font-size: 0.6875rem;
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

  /* Individual nav buttons — 3D interactive */
  .nav-btn {
    width: 34px;
    height: 34px;
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-tertiary);
    background: transparent;
    border: none;
    cursor: pointer;
    flex-shrink: 0;
    transform-style: preserve-3d;
    transition:
      color 150ms var(--ease-out),
      background 150ms var(--ease-out),
      box-shadow var(--duration-3d) var(--ease-3d-out),
      transform var(--duration-3d) var(--ease-3d-spring);
  }

  .nav-btn:hover {
    color: var(--text-primary);
    background: var(--surface-hover);
    transform: perspective(600px) translateY(-2px) translateZ(4px) scale(1.06);
    box-shadow: var(--elevation-1);
  }

  .nav-btn:active {
    transform: perspective(600px) translateZ(-4px) scale(0.90) !important;
    transition-duration: 60ms !important;
    box-shadow: var(--shadow-3d-active);
  }

  .nav-btn.active {
    color: var(--primary-400);
    background: rgba(99, 102, 241, 0.12);
    box-shadow:
      0 0 0 1px rgba(99, 102, 241, 0.28),
      0 4px 12px rgba(99, 102, 241, 0.18),
      inset 0 1px 0 rgba(255, 255, 255, 0.08);
    transform: perspective(600px) translateZ(2px);
  }

  /* Safety button gets a subtle guardian tint when active */
  .nav-btn-safety.active {
    color: var(--accent-guardian);
    background: rgba(139, 92, 246, 0.10);
    box-shadow: 0 0 0 1px rgba(139, 92, 246, 0.22), inset 0 1px 0 rgba(255, 255, 255, 0.06);
  }

  /* ── Premium track pill — 3D hero action ────────────────────────────── */
  .track-pill {
    border-radius: var(--radius-full);
    padding: var(--space-2) var(--space-4);
    font-family: var(--font-display);
    font-size: 0.8125rem;
    font-weight: 800;
    letter-spacing: -0.005em;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    white-space: nowrap;
    cursor: pointer;
    border: none;
    flex-shrink: 0;
    transform-style: preserve-3d;
    transition:
      transform var(--duration-3d) var(--ease-3d-spring),
      box-shadow var(--duration-3d) var(--ease-3d-out),
      background 180ms var(--ease-out);
  }

  /* Idle: indigo 3D pill with strong depth */
  .track-pill:not(.live) {
    background: linear-gradient(135deg, var(--primary-500) 0%, var(--primary-700) 100%);
    color: white;
    box-shadow:
      0 6px 20px rgba(79, 70, 229, 0.50),
      0 2px 6px rgba(79, 70, 229, 0.30),
      0 0 0 1px rgba(99, 102, 241, 0.35),
      inset 0 1px 0 rgba(255, 255, 255, 0.20),
      inset 0 -2px 4px rgba(0, 0, 0, 0.12);
  }

  .track-pill:not(.live):hover {
    transform: perspective(600px) translateY(-3px) translateZ(8px) scale(1.04);
    box-shadow:
      0 10px 32px rgba(79, 70, 229, 0.55),
      0 4px 10px rgba(79, 70, 229, 0.35),
      var(--glow-primary),
      inset 0 1px 0 rgba(255, 255, 255, 0.22),
      inset 0 -2px 4px rgba(0, 0, 0, 0.10);
  }

  /* Live: aurora green with 3D breathing glow */
  .track-pill.live {
    background: linear-gradient(135deg, var(--success-400) 0%, var(--success-600) 100%);
    color: white;
    box-shadow:
      0 6px 20px rgba(16, 185, 129, 0.50),
      0 2px 6px rgba(16, 185, 129, 0.30),
      0 0 0 1px rgba(16, 185, 129, 0.40),
      inset 0 1px 0 rgba(255, 255, 255, 0.20),
      inset 0 -2px 4px rgba(0, 0, 0, 0.10);
    animation: live-glow-pulse 2.5s ease-in-out infinite;
  }

  .track-pill:active {
    transform: perspective(600px) translateZ(-6px) scale(0.92) !important;
    transition-duration: 55ms !important;
    box-shadow: var(--btn-3d-press) !important;
  }

  /* Recording dot — white with subtle pulse */
  .rec-dot {
    display: inline-block;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.95);
    flex-shrink: 0;
    animation: recording-blink 1.2s ease-in-out infinite;
    box-shadow: 0 0 4px rgba(255, 255, 255, 0.60);
  }

  /* Avatar — bumped to 34px, display font initials, live presence ring */
  .navbar-avatar {
    width: 34px;
    height: 34px;
    border-radius: 50%;
    background: linear-gradient(135deg, rgba(99,102,241,0.25), rgba(79,70,229,0.15));
    color: var(--primary-300);
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 0.8125rem; /* 13px for initials at 34px */
    flex-shrink: 0;
    user-select: none;
    border: 1.5px solid rgba(99, 102, 241, 0.30);
    transition:
      box-shadow 300ms var(--ease-out),
      border-color 300ms var(--ease-out),
      transform 200ms var(--ease-spring);
    cursor: default;
  }

  :global([data-theme="light"]) .navbar-avatar {
    background: linear-gradient(135deg, var(--primary-100), rgba(99,102,241,0.08));
    color: var(--primary-600);
    border-color: rgba(99, 102, 241, 0.20);
  }

  /* Live: aurora green ring with glow */
  .avatar-live {
    border-color: var(--success-500);
    box-shadow:
      0 0 0 2px var(--success-500),
      0 0 0 4px rgba(0, 0, 0, 0.55),
      0 0 14px rgba(16, 185, 129, 0.45);
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
