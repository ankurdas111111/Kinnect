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
    background: var(--surface-inset, rgba(15, 23, 42, 0.10));
    border: 1px solid var(--border-subtle, rgba(15, 23, 42, 0.15));
    color: var(--text-secondary);
    font-size: 11px;
    font-weight: 600;
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
  }

  .pill.ok {
    background: rgba(16, 185, 129, 0.14);
    border-color: rgba(16, 185, 129, 0.3);
    color: var(--success-700, #047857);
  }

  .pill.warn {
    background: rgba(245, 158, 11, 0.14);
    border-color: rgba(245, 158, 11, 0.32);
    color: var(--warning-700, #b45309);
  }

  :global([data-theme="dark"]) .pill.ok {
    background: color-mix(in oklch, var(--success-500) 18%, transparent);
    border-color: color-mix(in oklch, var(--success-400) 35%, transparent);
    color: var(--success-400, #34d399);
  }

  :global([data-theme="dark"]) .pill.warn {
    background: color-mix(in oklch, var(--warning-500) 18%, transparent);
    border-color: color-mix(in oklch, var(--warning-400) 35%, transparent);
    color: var(--warning-400, #fbbf24);
  }
</style>
