<script>
  import { createEventDispatcher } from 'svelte';
  import { push } from 'svelte-spa-router';
  import StatusPills from './StatusPills.svelte';

  /**
   * @typedef {Object} Props
   * @property {string} [activeTab]
   * @property {boolean} [trackingActive]
   * @property {boolean} [hasNotification]
   * @property {any} [lastAccuracy]
   * @property {any} [latencyMs]
   * @property {boolean} [isOnline]
   * @property {boolean} [socketConnected]
   * @property {number} [bufferedCount]
   */

  /** @type {Props} */
  let {
    activeTab = 'track',
    trackingActive = false,
    hasNotification = false,
    lastAccuracy = null,
    latencyMs = null,
    isOnline = true,
    socketConnected = false,
    bufferedCount = 0
  } = $props();

  const dispatch = createEventDispatcher();

  const tabTitles = {
    track: 'Track',
    people: 'People',
    share: 'Share',
    safety: 'Safety',
    me: 'Profile'
  };

  let title = $derived(tabTitles[activeTab] || 'Track');
</script>

<header class="mobile-top-bar">
  <div class="bar-main">
    <div class="title-wrap">
      <div class="title-row">
        <h1>{title}</h1>
        <!-- Connection warning — surfaces the existing isOnline prop, calm amber -->
        {#if !isOnline}
          <span class="conn-warning" role="status" aria-live="polite">
            <span class="conn-dot" aria-hidden="true"></span>
            Offline
          </span>
        {/if}
      </div>
      <p>{trackingActive ? 'Sharing your location' : 'Location paused'}</p>
    </div>
    <div class="top-actions">
      <button class="icon-btn dashboard-btn" aria-label="Family Dashboard" onclick={() => push('/dashboard')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
      </button>
      <button class="icon-btn" aria-label="Open profile tab" onclick={() => dispatch('openMe')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M20 21a8 8 0 0 0-16 0"></path>
          <circle cx="12" cy="7" r="4"></circle>
        </svg>
        {#if hasNotification}
          <span class="dot" aria-hidden="true"></span>
        {/if}
      </button>
    </div>
  </div>
  <StatusPills
    {trackingActive}
    {lastAccuracy}
    {latencyMs}
    {isOnline}
    {socketConnected}
    {bufferedCount}
  />
</header>

<style>
  .mobile-top-bar {
    position: fixed;
    left: 0;
    right: 0;
    top: 0;
    z-index: calc(var(--z-navbar) + 1);
    padding: calc(var(--safe-top, 0px) + 8px) 12px 8px;
    /* Liquid-glass surface — theme-aware translucent + blur */
    background: var(--glass-bg);
    backdrop-filter: var(--glass-blur);
    -webkit-backdrop-filter: var(--glass-blur);
    border-bottom: 1px solid var(--glass-border);
    box-shadow: var(--glass-shadow);
  }

  :global([data-theme="dark"]) .mobile-top-bar {
    border-bottom-color: var(--glass-border);
  }

  .bar-main {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--space-2);
    gap: var(--space-2);
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .title-wrap h1 {
    margin: 0;
    font-size: var(--text-xl);
    line-height: 1.2;
    letter-spacing: 0.01em;
  }

  .title-wrap p {
    margin: 2px 0 0;
    font-size: 12px;
    color: var(--text-secondary, #64748b);
  }

  /* Offline / connection warning — calm amber, not alarming */
  .conn-warning {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px var(--space-2);
    border-radius: var(--radius-full);
    background: color-mix(in oklch, var(--warning-500) 12%, transparent);
    color: var(--warning-500);
    border: 1px solid color-mix(in oklch, var(--warning-500) 24%, transparent);
    font-family: var(--font-display);
    font-size: var(--text-2xs, 10px);
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    line-height: 1;
  }

  .conn-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--warning-500);
    flex-shrink: 0;
    animation: conn-warn-pulse 2.4s ease-in-out infinite;
  }

  @keyframes conn-warn-pulse {
    0%, 100% { opacity: 1; }
    50%      { opacity: 0.4; }
  }

  .top-actions {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-shrink: 0;
  }

  .icon-btn {
    width: 44px;
    height: 44px;
    border-radius: 14px;
    border: 1px solid var(--border-default);
    background: var(--surface-2);
    color: var(--text-primary);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    position: relative;
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
    box-shadow: var(--elevation-1, 0 2px 8px rgba(0,0,0,0.08));
    transition: background var(--duration-fast) var(--ease-out), transform 100ms ease;
  }

  .icon-btn:active {
    background: var(--surface-active);
    transform: scale(0.93);
  }
  @media (hover: none) {
    .icon-btn:hover {
      background: var(--surface-2);
    }
  }

  /* Dashboard shortcut — teal tint to stand out */
  .dashboard-btn {
    background: rgba(20, 184, 166, 0.10);
    border-color: rgba(20, 184, 166, 0.20);
    color: var(--primary-500);
  }
  .dashboard-btn:active {
    background: rgba(20, 184, 166, 0.18);
  }

  .dot {
    position: absolute;
    top: 8px;
    right: 8px;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--danger-500, #ef4444);
    border: 2px solid var(--surface-2);
  }

  @media (prefers-reduced-motion: reduce) {
    .conn-dot { animation: none; }
  }

  @media (min-width: 768px) {
    .mobile-top-bar {
      display: none;
    }
  }
</style>
