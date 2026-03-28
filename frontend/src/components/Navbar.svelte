<script>
  import { createEventDispatcher } from 'svelte';
  import { authUser } from '../lib/stores/auth.js';
  import { tracking } from '../lib/stores/map.js';
  import { socket } from '../lib/socket.js';
  import { apiPost } from '../lib/api.js';
  import ThemeToggle from './ThemeToggle.svelte';
  import { privacyPause } from '../lib/stores/places.js';

  $: ghostMode = $privacyPause && $privacyPause > Date.now();

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

    <!-- Group 3: Places + Settings -->
    <div class="nav-group">
      <button class="nav-btn" class:active={activePanel === 'places'} on:click={() => toggle('places')} title="Your Spots" aria-label="Toggle places panel" aria-pressed={activePanel === 'places'}>
        <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
      </button>
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

    <ThemeToggle />

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
    background: var(--glass-bg-strong);
    backdrop-filter: var(--glass-blur);
    -webkit-backdrop-filter: var(--glass-blur);
    border-bottom: 1px solid var(--border-default);
    box-shadow: var(--shadow-navbar);
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

  /* Circular gradient logo — MERIDIAN */
  .navbar-logo {
    width: 34px;
    height: 34px;
    border-radius: 50%;
    background: linear-gradient(135deg, var(--primary-500), var(--accent-guardian));
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    flex-shrink: 0;
    box-shadow: var(--glow-primary), var(--shadow-sm);
    transition: box-shadow 400ms var(--ease-out), transform 300ms var(--ease-spring);
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

  .navbar-title {
    font-size: var(--text-lg);
    font-weight: 800;
    color: var(--text-primary);
    letter-spacing: -0.02em;
    line-height: 1.1;
  }

  .navbar-context-live {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 9px;
    font-weight: 700;
    color: var(--success-500);
    letter-spacing: 0.08em;
    text-transform: uppercase;
    line-height: 1;
  }

  .context-dot {
    display: inline-block;
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--success-500);
    animation: animate-rec-blink 1.4s ease-in-out infinite;
    flex-shrink: 0;
  }

  .navbar-context-ghost {
    font-size: 9px;
    font-weight: 700;
    color: var(--text-tertiary);
    letter-spacing: 0.08em;
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

  /* Individual nav buttons */
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
    transition:
      color 150ms var(--ease-out),
      background 150ms var(--ease-out),
      box-shadow 200ms var(--ease-out),
      transform 120ms var(--ease-spring);
  }

  .nav-btn:hover {
    color: var(--text-primary);
    background: var(--surface-hover);
    transform: translateY(-1px);
  }

  .nav-btn:active {
    transform: scale(0.92) !important;
    transition-duration: 60ms !important;
  }

  .nav-btn.active {
    color: var(--primary-400);
    background: rgba(99, 102, 241, 0.10);
    box-shadow: 0 0 0 1px rgba(99, 102, 241, 0.22), inset 0 1px 0 rgba(255, 255, 255, 0.06);
  }

  /* Safety button gets a subtle guardian tint when active */
  .nav-btn-safety.active {
    color: var(--accent-guardian);
    background: rgba(139, 92, 246, 0.10);
    box-shadow: 0 0 0 1px rgba(139, 92, 246, 0.22), inset 0 1px 0 rgba(255, 255, 255, 0.06);
  }

  /* Premium track pill */
  .track-pill {
    border-radius: var(--radius-full);
    padding: var(--space-1-5) var(--space-4);
    font-family: var(--font-sans);
    font-size: var(--text-sm);
    font-weight: 700;
    letter-spacing: -0.01em;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    white-space: nowrap;
    cursor: pointer;
    border: none;
    flex-shrink: 0;
    transition: all 220ms var(--ease-spring);
  }

  .track-pill:not(.live) {
    background: var(--primary-600);
    color: white;
    box-shadow: var(--shadow-primary), 0 0 16px rgba(99, 102, 241, 0.20);
  }

  .track-pill:not(.live):hover {
    background: var(--primary-500);
    box-shadow: var(--shadow-primary), var(--glow-primary);
    transform: translateY(-1px);
  }

  .track-pill.live {
    background: linear-gradient(135deg, var(--success-500), var(--success-600));
    color: white;
    animation: live-glow-pulse 2.5s ease-in-out infinite;
  }

  .track-pill:active {
    transform: scale(0.95) !important;
    transition-duration: 60ms !important;
  }

  .rec-dot {
    display: inline-block;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: white;
    flex-shrink: 0;
  }

  .navbar-avatar {
    width: 30px;
    height: 30px;
    border-radius: 50%;
    background: var(--primary-100);
    color: var(--primary-700);
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    font-size: var(--text-xs);
    flex-shrink: 0;
    user-select: none;
    transition: box-shadow 300ms var(--ease-out);
  }

  :global([data-theme="dark"]) .navbar-avatar {
    background: rgba(99, 102, 241, 0.18);
    color: var(--primary-300);
  }

  .avatar-live {
    box-shadow: var(--ring-live);
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
