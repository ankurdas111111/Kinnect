<script>
  import { createEventDispatcher } from 'svelte';
  import { myLocation } from '../../lib/stores/map.js';
  import { authUser } from '../../lib/stores/auth.js';

  const dispatch = createEventDispatcher();
</script>

<!-- Self entry — always shown when tracking -->
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
        <span class="you-badge animate-live-badge">
          <span class="you-badge-dot" aria-hidden="true"></span>
          Live
        </span>
      </div>
      <div class="user-sub">
        {#if $myLocation.speed != null && $myLocation.speed > 0.5}
          <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="5 12 12 5 19 12"/></svg>
          <span>{parseFloat($myLocation.speed).toFixed(0)} km/h</span>
          <span class="sep">·</span>
        {/if}
        <span class="sub-live-label">Sharing now</span>
      </div>
    </div>
    <span class="locate-icon" aria-hidden="true">
      <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="10" r="3"/><path d="M12 2a8 8 0 0 0-8 8c0 1.892.402 3.13 1.5 4.5L12 22l6.5-7.5c1.098-1.37 1.5-2.608 1.5-4.5a8 8 0 0 0-8-8z"/></svg>
    </span>
  </button>
{/if}

<style>
  .user-item {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2-5) var(--space-4);
    min-height: 76px;
  }

  .user-item-btn {
    width: 100%;
    background: none;
    border: none;
    border-bottom: none;
    cursor: pointer;
    text-align: left;
    color: inherit;
    font: inherit;
    border-radius: 0;
    transition:
      background var(--duration-fast) var(--ease-out),
      transform var(--duration-fast) var(--ease-spring);
    -webkit-tap-highlight-color: transparent;
    position: relative;
  }

  .user-item-btn:hover {
    background: var(--surface-hover);
  }

  .user-item-btn:active {
    background: var(--surface-active);
    transform: scale(0.97) translateZ(-2px);
    transition-duration: 60ms;
  }

  /* Self row — subtle primary left accent */
  .me {
    background: rgba(99, 102, 241, 0.04);
    box-shadow: inset 3px 0 0 rgba(99, 102, 241, 0.50);
  }
  .me:hover { background: rgba(99, 102, 241, 0.08); }

  .user-avatar {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 1.0625rem; /* 17px — legible initials */
    flex-shrink: 0;
    text-transform: uppercase;
    line-height: 1;
    position: relative;
    transition: transform var(--duration-normal) var(--ease-spring);
  }

  .user-item-btn:hover .user-avatar {
    transform: scale(1.08) translateZ(4px);
    filter: brightness(1.12);
  }

  /* Self avatar — gradient + strong glow */
  .self-avatar {
    background: linear-gradient(135deg, var(--primary-400) 0%, var(--primary-700) 100%);
    color: white;
    font-size: 1.125rem;
    box-shadow:
      0 0 0 2.5px var(--primary-500),
      0 0 0 5px rgba(0, 0, 0, 0.55),
      0 0 16px rgba(99, 102, 241, 0.40);
  }

  /* Self: pulsing live ring */
  .presence-ring-self {
    position: absolute;
    inset: -3px;
    border-radius: 50%;
    pointer-events: none;
    animation: aurora-pulse 2.8s ease-in-out infinite;
  }

  .user-meta {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .user-name-row {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    min-width: 0;
    overflow: hidden;
  }

  .user-name {
    font-family: var(--font-display);
    font-size: var(--text-base);    /* 16px — legible primary label */
    font-weight: 600;
    letter-spacing: -0.01em;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--text-primary);
    line-height: 1.25;
  }

  /* Premium "Live" badge for self — pulsing dot */
  .you-badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-family: var(--font-display);
    font-size: 0.6875rem; /* 11px */
    font-weight: 700;
    color: var(--success-500);
    background: rgba(16, 185, 129, 0.12);
    border: 1px solid rgba(16, 185, 129, 0.30);
    border-radius: var(--radius-full);
    padding: 2px 7px 2px 5px;
    line-height: 1.3;
    flex-shrink: 0;
    letter-spacing: 0.02em;
    box-shadow: 0 0 6px rgba(16, 185, 129, 0.22), inset 0 1px 0 rgba(255,255,255,0.10);
  }

  .you-badge-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--success-500);
    flex-shrink: 0;
    animation: aurora-pulse 2s ease-in-out infinite;
  }

  .user-sub {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: var(--text-xs);
    color: var(--text-secondary);
    flex-wrap: nowrap;
    overflow: hidden;
  }

  .sub-live-label {
    color: var(--success-500);
    font-weight: 600;
  }

  .sep { color: var(--text-tertiary); flex-shrink: 0; }

  .locate-icon {
    color: var(--text-tertiary);
    transition: color var(--duration-fast) var(--ease-out), transform var(--duration-fast) var(--ease-spring);
    display: flex;
    align-items: center;
  }
  .user-item-btn:hover .locate-icon {
    color: var(--primary-400);
    transform: scale(1.15);
  }

  @media (prefers-reduced-motion: reduce) {
    .presence-ring-self,
    .you-badge-dot { animation: none; }
  }
</style>
