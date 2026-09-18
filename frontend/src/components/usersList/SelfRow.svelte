<script>
  import { createEventDispatcher } from 'svelte';
  import { myLocation } from '../../lib/stores/map.js';
  import { authUser } from '../../lib/stores/auth.js';

  const dispatch = createEventDispatcher();
</script>

<!-- Self entry — always shown when tracking. Ember is YOU: the one accent. -->
{#if $myLocation}
  <button
    class="user-item user-item-btn me"
    onclick={() => dispatch('locate', '__self__')}
    aria-label="Locate yourself on map"
  >
    <div class="user-avatar self-avatar">
      {($authUser?.displayName || 'Y')[0].toUpperCase()}
      <span class="presence-ring-self" aria-hidden="true"></span>
    </div>
    <div class="user-meta">
      <div class="user-name-row">
        <strong class="user-name">{$authUser?.displayName || 'You'}</strong>
        <span class="you-badge">
          <span class="you-badge-dot" aria-hidden="true"></span>
          Live
        </span>
      </div>
      <div class="user-sub">
        {#if $myLocation.speed != null && $myLocation.speed > 0.5}
          <span class="font-tabular">{parseFloat($myLocation.speed).toFixed(0)} km/h</span>
          <span class="sep" aria-hidden="true">·</span>
        {/if}
        <span class="sub-live-label">Sharing your location right now</span>
      </div>
    </div>
    <span class="locate-icon" aria-hidden="true">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="10" r="3"/><path d="M12 2a8 8 0 0 0-8 8c0 1.892.402 3.13 1.5 4.5L12 22l6.5-7.5c1.098-1.37 1.5-2.608 1.5-4.5a8 8 0 0 0-8-8z"/></svg>
    </span>
  </button>
{/if}

<style>
  .user-item {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-4);
    min-height: 76px;
    border-bottom: 1px solid var(--border-subtle);
  }

  .user-item-btn {
    width: 100%;
    background: none;
    border-top: none;
    border-left: none;
    border-right: none;
    cursor: pointer;
    text-align: left;
    color: inherit;
    font: inherit;
    border-radius: 0;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    position: relative;
  }

  .user-item-btn:hover {
    background: var(--surface-hover);
  }

  .user-item-btn:active {
    background: var(--surface-active, var(--surface-hover));
  }

  .user-item-btn:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: -2px;
  }

  /* Self row — the faintest ember wash + steady ember accent bar */
  .me {
    background: color-mix(in oklch, var(--primary-500) 4%, transparent);
    box-shadow: inset 3px 0 0 var(--primary-500);
  }
  .me:hover { background: color-mix(in oklch, var(--primary-500) 8%, transparent); }

  /* Ember pebble — warm tint face, ember initial, single ember ring */
  .user-avatar {
    width: 44px;
    height: 44px;
    border-radius: var(--radius-full);
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-sans);
    font-weight: 700;
    font-size: var(--text-lg);
    flex-shrink: 0;
    text-transform: uppercase;
    line-height: 1;
    position: relative;
  }

  .self-avatar {
    background: color-mix(in oklch, var(--primary-500) 16%, var(--card));
    color: var(--primary-700);
    box-shadow:
      0 0 0 2px var(--primary-500),
      0 0 0 5px var(--card);
  }

  /* Gentle breathing halo — you are live. GPU-only (opacity/transform inside
     the global aurora-pulse keyframes). */
  .presence-ring-self {
    position: absolute;
    inset: calc(-1 * var(--space-1));
    border-radius: var(--radius-full);
    pointer-events: none;
    box-shadow: 0 0 0 1px color-mix(in oklch, var(--primary-500) 30%, transparent);
    animation: aurora-pulse 2.8s ease-in-out infinite;
  }

  .user-meta {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-0-5);
  }

  .user-name-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
    overflow: hidden;
  }

  .user-name {
    font-family: var(--font-sans);
    font-size: var(--text-base);    /* 16px — legible primary label */
    font-weight: 600;
    letter-spacing: -0.01em;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--text-primary);
    line-height: var(--leading-tight);
  }

  /* Quiet "Live" pill — ember register: you, present, accounted for */
  .you-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    font-family: var(--font-sans);
    font-size: var(--text-2xs);
    font-weight: 600;
    color: var(--primary-700);
    background: color-mix(in oklch, var(--primary-500) 12%, transparent);
    border-radius: var(--radius-full);
    padding: var(--space-0-5) var(--space-2);
    line-height: 1.3;
    flex-shrink: 0;
    letter-spacing: 0.02em;
  }

  .you-badge-dot {
    width: 5px;
    height: 5px;
    border-radius: var(--radius-full);
    background: var(--primary-500);
    flex-shrink: 0;
    animation: aurora-pulse 2s ease-in-out infinite;
  }

  /* One plain sentence, 16px functional register */
  .user-sub {
    display: flex;
    align-items: baseline;
    gap: var(--space-1);
    font-size: var(--text-base);
    line-height: var(--leading-normal);
    color: var(--text-secondary);
    flex-wrap: nowrap;
    overflow: hidden;
    white-space: nowrap;
  }

  .sub-live-label {
    color: var(--text-secondary);
  }

  .sep { color: var(--text-tertiary); opacity: 0.7; flex-shrink: 0; }

  .locate-icon {
    color: var(--text-tertiary);
    opacity: 0.6;
    transition: color var(--duration-fast) var(--ease-out), opacity var(--duration-fast) var(--ease-out);
    display: flex;
    align-items: center;
  }
  .user-item-btn:hover .locate-icon {
    color: var(--primary-700);
    opacity: 1;
  }

  @media (prefers-reduced-motion: reduce) {
    .presence-ring-self,
    .you-badge-dot { animation: none; }
  }
</style>
