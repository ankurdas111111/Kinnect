<script>
  import { createEventDispatcher } from 'svelte';
  import TiltCard from './TiltCard.svelte';

  export let location = null;
  export let trackingActive = false;
  export let bufferedCount = 0;
  export let socketConnected = false;

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
      on:click={() => dispatch('toggleTracking')}
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
    <button class="btn btn-secondary" on:click={() => dispatch('centerOnMe')}>Center Me</button>
    <button class="btn btn-secondary" on:click={() => dispatch('toggleFollow')}>Follow</button>
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
  .now-card {
    background: var(--glass-3d, rgba(255,255,255,0.65));
    border: 1px solid var(--glass-3d-border, #e2e8f0);
    border-top-color: rgba(255, 255, 255, 0.25);
    border-radius: 20px;
    padding: 14px;
    /* 3D floating card with depth */
    box-shadow:
      var(--elevation-3, 0 10px 28px rgba(15, 23, 42, 0.08)),
      inset 0 1px 0 rgba(255, 255, 255, 0.18),
      inset 0 -1px 0 rgba(0, 0, 0, 0.05);
    backdrop-filter: var(--glass-3d-blur, blur(24px) saturate(2.0));
    -webkit-backdrop-filter: var(--glass-3d-blur, blur(24px) saturate(2.0));
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
    color: var(--text-secondary, #64748b);
  }

  .toggle {
    min-width: 82px;
    min-height: 44px;
  }

  .toggle.live {
    background: var(--danger-500, #ef4444);
  }

  .stats {
    margin-top: 12px;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .stat {
    background: var(--surface-inset, rgba(15, 23, 42, 0.04));
    border-radius: 12px;
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    /* 3D inset stat cell */
    border: 1px solid var(--border-subtle, rgba(0,0,0,0.04));
    box-shadow:
      inset 0 2px 4px rgba(0, 0, 0, 0.04),
      0 1px 0 rgba(255, 255, 255, 0.08);
  }

  .stat.wide {
    grid-column: span 2;
  }

  .stat span {
    font-size: 11px;
    color: var(--text-secondary, #64748b);
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
    color: var(--text-secondary, #64748b);
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
</style>
