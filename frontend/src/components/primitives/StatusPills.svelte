<script>
  import NetworkStateChip from './NetworkStateChip.svelte';

  /**
   * @typedef {Object} Props
   * @property {boolean} [trackingActive]
   * @property {any} [lastAccuracy]
   * @property {any} [latencyMs]
   * @property {boolean} [isOnline]
   * @property {boolean} [socketConnected]
   * @property {number} [bufferedCount]
   */

  /** @type {Props} */
  let {
    trackingActive = false,
    lastAccuracy = null,
    latencyMs = null,
    isOnline = true,
    socketConnected = false,
    bufferedCount = 0
  } = $props();
</script>

<!-- Quiet by default: chips appear only when something needs attention.
     The header already narrates the normal states ("Sharing live",
     "Location paused") — repeating them here was noise, and raw telemetry
     (GPS metres, latency ms) never belonged in a family app. -->
<div class="status-pills" role="status" aria-live="polite">
  {#if !isOnline || !socketConnected || bufferedCount > 0}
    <NetworkStateChip {isOnline} {socketConnected} {bufferedCount} />
  {/if}
  {#if trackingActive && lastAccuracy != null && lastAccuracy > 120}
    <div class="pill warn">Weak GPS signal</div>
  {/if}
</div>

<style>
  .status-pills {
    display: flex;
    gap: 6px;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    padding: 0 2px;
    scrollbar-width: none;
  }

  .status-pills::-webkit-scrollbar {
    display: none;
  }

  .pill {
    display: inline-flex;
    align-items: center;
    white-space: nowrap;
    padding: 6px 10px;
    border-radius: 999px;
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    font-size: 11px;
    font-weight: 600;
  }

  .pill.ok {
    background: color-mix(in oklch, var(--success-500) 12%, transparent);
    border-color: color-mix(in oklch, var(--success-500) 28%, transparent);
    color: var(--success-700);
  }

  .pill.warn {
    background: color-mix(in oklch, var(--warning-500) 12%, transparent);
    border-color: color-mix(in oklch, var(--warning-500) 28%, transparent);
    color: var(--warning-700);
  }
</style>
