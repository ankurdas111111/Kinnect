<script>
  import { createEventDispatcher } from 'svelte';
  import TiltCard from './TiltCard.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [location]
   * @property {boolean} [trackingActive]
   * @property {number} [bufferedCount]
   * @property {boolean} [socketConnected]
   */

  /** @type {Props} */
  let {
    location = null,
    trackingActive = false,
    bufferedCount = 0,
    socketConnected = false
  } = $props();

  const dispatch = createEventDispatcher();
</script>

<TiltCard intensity={8} shine={true}>
<section class="now-card">
  <div class="head">
    <div>
      <h3>Now</h3>
      <p>{trackingActive ? 'Realtime tracking is active' : 'Tracking is paused'}</p>
    </div>
    <button
      class="btn btn-primary toggle"
      class:live={trackingActive}
      onclick={() => dispatch('toggleTracking')}
      aria-pressed={trackingActive}
    >
      {trackingActive ? 'Stop' : 'Start'}
    </button>
  </div>

  {#if location}
    <div class="stats">
      <div class="stat">
        <span>Speed</span>
        <strong>{(location.speed || 0).toFixed(1)} km/h</strong>
      </div>
      <div class="stat">
        <span>Accuracy</span>
        <strong>~{Math.round(location.accuracy || 0)} m</strong>
      </div>
      <div class="stat wide">
        <span>Last update</span>
        <strong>{location.formattedTime || '--'}</strong>
      </div>
    </div>
  {:else}
    <div class="skeleton">
      <div class="line"></div>
      <div class="line short"></div>
    </div>
  {/if}

  <div class="footer">
    <button class="btn btn-secondary" onclick={() => dispatch('centerOnMe')}>Center Me</button>
    <button class="btn btn-secondary" onclick={() => dispatch('toggleFollow')}>Follow</button>
    <span class="meta">
      {#if !socketConnected}
        reconnecting
      {:else if bufferedCount > 0}
        buffered {bufferedCount}
      {:else}
        live
      {/if}
    </span>
  </div>
</section>
</TiltCard>

<style>
  /* Hearth: warm paper card, no glass. */
  .now-card {
    background: var(--surface-1);
    border: 1px solid var(--border-default);
    border-radius: 20px;
    padding: 14px;
    box-shadow: var(--shadow-sm);
    margin-bottom: 12px;
    transform-style: preserve-3d;
  }

  .head {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: center;
  }

  h3 {
    margin: 0;
    font-size: 17px;
  }

  p {
    margin: 2px 0 0;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .toggle {
    min-width: 82px;
    min-height: 44px;
  }

  /* Stopping a share is a calm act, not an emergency — sage, never red. */
  .toggle.live {
    background: var(--success-600);
    color: var(--text-inverse);
  }

  .stats {
    margin-top: 12px;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .stat {
    background: var(--surface-inset);
    border-radius: 12px;
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    border: 1px solid var(--border-subtle);
  }

  .stat.wide {
    grid-column: span 2;
  }

  .stat span {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .stat strong {
    font-size: 14px;
  }

  .footer {
    margin-top: 12px;
    display: flex;
    gap: 8px;
    align-items: center;
    flex-wrap: wrap;
  }

  .footer .btn {
    min-height: 44px;
  }

  .meta {
    font-size: 11px;
    color: var(--text-secondary);
    margin-left: auto;
  }

  .skeleton {
    margin-top: 12px;
  }

  .line {
    height: 12px;
    border-radius: 999px;
    background: linear-gradient(90deg, var(--border-default), var(--surface-hover), var(--border-default));
    background-size: 220% 100%;
    animation: shimmer 1.2s linear infinite;
    margin-bottom: 8px;
  }

  .line.short {
    width: 65%;
  }

  @keyframes shimmer {
    from { background-position: 200% 0; }
    to { background-position: -20% 0; }
  }

  @media (prefers-reduced-motion: reduce) {
    .line { animation: none; }
  }
</style>
