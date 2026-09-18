<script>
  import { createEventDispatcher } from 'svelte';
  import { haptics } from '../../lib/haptics.js';

  /**
   * @typedef {Object} Props
   * @property {boolean} [isTracking]
   * @property {boolean} [followMode]
   */

  /** @type {Props} */
  let { isTracking = false, followMode = false } = $props();

  const dispatch = createEventDispatcher();

  // Brief scale-pulse confirmation when center-on-me is tapped
  let centerPulse = $state(false);
  let pulseTimer;
  function centerOnMe() {
    haptics.tap();
    dispatch('centerOnMe');
    centerPulse = false;
    // re-trigger on rapid taps
    requestAnimationFrame(() => { centerPulse = true; });
    clearTimeout(pulseTimer);
    pulseTimer = setTimeout(() => { centerPulse = false; }, 420);
  }
</script>

<div class="fab-cluster" role="group" aria-label="Map controls">
  <!-- Secondary: center-on-me -->
  <button
    class="fab fab--secondary tactile"
    class:center-pulse={centerPulse}
    onclick={centerOnMe}
    title="Center map on me"
    aria-label="Center map on me"
  >
    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M12 21s-6-4.3-6-9a6 6 0 1 1 12 0c0 4.7-6 9-6 9z"/>
      <circle cx="12" cy="12" r="2.5"/>
    </svg>
  </button>

  <!-- Secondary: follow-mode -->
  <button
    class="fab fab--secondary tactile"
    class:follow-active={followMode}
    onclick={() => { haptics.tap(); dispatch('toggleFollow'); }}
    title={followMode ? 'Stop following me' : 'Follow me automatically'}
    aria-label={followMode ? 'Stop following me' : 'Follow me automatically'}
    aria-pressed={followMode}
  >
    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <circle cx="12" cy="8" r="3"/>
      <path d="M6.5 19a5.5 5.5 0 0 1 11 0"/>
      {#if followMode}
        <path d="m17 6 1.7 1.7L22 4.4"/>
      {/if}
    </svg>
  </button>

  <!-- Primary: tracking toggle -->
  <button
    class="fab fab--primary tactile"
    class:tracking={isTracking}
    onclick={() => dispatch('toggleTracking')}
    title={isTracking ? 'Stop sharing location' : 'Share my location'}
    aria-label={isTracking ? 'Stop sharing location' : 'Share my location'}
    aria-pressed={isTracking}
  >
    {#if isTracking}
      <!-- Stop icon -->
      <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
        <rect x="6" y="6" width="12" height="12" rx="2"/>
      </svg>
    {:else}
      <!-- Location pin icon -->
      <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/>
        <circle cx="12" cy="10" r="3"/>
      </svg>
    {/if}
  </button>
</div>

<style>
  .fab-cluster {
    display: flex;
    flex-direction: column;
    align-items: center;
    /* Slightly more breathing room between stacked FABs */
    gap: var(--space-3);
  }

  .fab {
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    cursor: pointer;
    border-radius: 50%;
    transition:
      transform var(--dur-fast, 150ms) var(--ease-spring, cubic-bezier(0.34,1.56,0.64,1)),
      box-shadow var(--dur-normal, 250ms) var(--ease-out, ease-out),
      background var(--dur-fast, 150ms);
    -webkit-tap-highlight-color: transparent;
  }

  .fab:active {
    transform: scale(0.93) !important;
  }

  /* Brief scale-pulse confirmation on center-on-me (transform-only) */
  .fab--secondary.center-pulse {
    animation: center-confirm 420ms var(--ease-spring, cubic-bezier(0.34,1.56,0.64,1));
  }

  @keyframes center-confirm {
    0%   { transform: scale(1); }
    35%  { transform: scale(1.16); }
    70%  { transform: scale(0.97); }
    100% { transform: scale(1); }
  }

  /* Primary FAB — 56px ember pebble. Sharing yourself is presence, not
     alarm: the active state settles to sage ("live"), never red. */
  .fab--primary {
    width: 56px;
    height: 56px;
    background: var(--primary-500);
    color: var(--text-on-primary);
    box-shadow: var(--shadow-primary);
  }

  .fab--primary:hover {
    transform: scale(1.05);
    background: var(--primary-600);
  }

  /* Tracking active — sage, quietly live. No breathing shadow loop. */
  .fab--primary.tracking {
    background: var(--success-600);
    color: var(--text-inverse);
    box-shadow: var(--shadow-md), 0 0 0 4px color-mix(in oklch, var(--success-500) 18%, transparent);
  }

  /* Secondary FABs — 44px paper chips */
  .fab--secondary {
    width: 44px;
    height: 44px;
    background: var(--surface-1);
    color: var(--text-secondary);
    box-shadow: var(--shadow-sm), 0 0 0 1px var(--border-default);
  }

  .fab--secondary:hover {
    transform: scale(1.08);
    color: var(--primary-700);
    background: var(--surface-2);
  }

  .fab--secondary.follow-active {
    color: var(--primary-700);
    background: var(--primary-100);
    box-shadow: var(--shadow-sm), 0 0 0 1px color-mix(in oklch, var(--primary-500) 35%, transparent);
  }

  @media (prefers-reduced-motion: reduce) {
    .fab,
    .fab--primary.tracking,
    .fab--secondary.center-pulse {
      animation: none;
      transition: none;
    }
  }
</style>
