<script>
  import { createEventDispatcher } from 'svelte';
  import { push } from 'svelte-spa-router';
  import StatusPills from './StatusPills.svelte';

  export let activeTab = 'track';
  export let trackingActive = false;
  export let hasNotification = false;
  export let lastAccuracy = null;
  export let latencyMs = null;
  export let isOnline = true;
  export let socketConnected = false;
  export let bufferedCount = 0;

  const dispatch = createEventDispatcher();

  const tabTitles = {
    track: 'Track',
    people: 'People',
    share: 'Share',
    safety: 'Safety',
    me: 'Profile'
  };

  $: title = tabTitles[activeTab] || 'Track';
</script>

<header class="mobile-top-bar">
  <div class="bar-main">
    <div class="title-wrap">
      <h1>{title}</h1>
      <p>{trackingActive ? 'Live location active' : 'Tracking paused'}</p>
    </div>
    <div class="top-actions">
      <button class="icon-btn dashboard-btn" aria-label="Family Dashboard" on:click={() => push('/dashboard')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
      </button>
      <button class="icon-btn" aria-label="Open profile tab" on:click={() => dispatch('openMe')}>
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
    /* Solid surface — avoids GPU stacking with panels + bottom bar */
    background: rgba(252, 252, 255, 0.97);
    border-bottom: 1px solid rgba(15, 23, 42, 0.08);
    box-shadow: 0 1px 0 rgba(0, 0, 0, 0.05), 0 2px 10px rgba(0, 0, 0, 0.06);
  }

  :global([data-theme="dark"]) .mobile-top-bar {
    background: rgba(9, 9, 20, 0.97);
    border-bottom-color: rgba(255, 255, 255, 0.07);
    box-shadow: 0 1px 0 rgba(255, 255, 255, 0.05), 0 2px 12px rgba(0, 0, 0, 0.40);
  }

  .bar-main {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
    gap: 8px;
  }

  .title-wrap h1 {
    margin: 0;
    font-size: 19px;
    line-height: 1.2;
    letter-spacing: 0.01em;
  }

  .title-wrap p {
    margin: 2px 0 0;
    font-size: 12px;
    color: var(--text-secondary, #64748b);
  }

  .top-actions {
    display: flex;
    align-items: center;
    gap: 6px;
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
    box-shadow: var(--elevation-1, 0 2px 8px rgba(0,0,0,0.08));
    transition: background var(--duration-fast) var(--ease-out), transform 100ms ease;
  }

  .icon-btn:active {
    background: var(--surface-active);
    transform: scale(0.93);
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

  @media (min-width: 768px) {
    .mobile-top-bar {
      display: none;
    }
  }
</style>
