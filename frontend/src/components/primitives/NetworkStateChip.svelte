<script>
  import { run } from 'svelte/legacy';

  import { onDestroy } from 'svelte';

  /**
   * @typedef {Object} Props
   * @property {boolean} [isOnline]
   * @property {boolean} [socketConnected]
   * @property {number} [bufferedCount]
   */

  /** @type {Props} */
  let { isOnline = true, socketConnected = false, bufferedCount = 0 } = $props();

  // Don't flash "Reconnecting" for brief drops (< 2.5s).
  // The socket reconnects within 200–500ms during normal background/foreground
  // cycles, so this prevents constant visual noise during those transitions.
  let showReconnecting = $state(false);
  let _timer = $state(null);

  run(() => {
    if (!socketConnected) {
      if (!_timer) {
        _timer = setTimeout(() => {
          _timer = null;
          if (!socketConnected) showReconnecting = true;
        }, 2500);
      }
    } else {
      if (_timer) { clearTimeout(_timer); _timer = null; }
      showReconnecting = false;
    }
  });

  onDestroy(() => { if (_timer) { clearTimeout(_timer); _timer = null; } });
</script>

<div class="network-chip" class:offline={!isOnline || showReconnecting}>
  <span class="dot" aria-hidden="true"></span>
  <span class="label">
    {#if !isOnline}
      Offline
    {:else if showReconnecting}
      Reconnecting
    {:else}
      Live
    {/if}
  </span>
  {#if bufferedCount > 0}
    <span class="buffered">buffering {bufferedCount}</span>
  {/if}
</div>

<style>
  .network-chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 6px 10px;
    border-radius: 999px;
    background: color-mix(in oklch, var(--success-500) 12%, transparent);
    border: 1px solid color-mix(in oklch, var(--success-500) 28%, transparent);
    color: var(--success-700);
    font-size: 11px;
    font-weight: 600;
  }

  .network-chip.offline {
    background: color-mix(in oklch, var(--warning-500) 12%, transparent);
    border-color: color-mix(in oklch, var(--warning-500) 28%, transparent);
    color: var(--warning-700);
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: currentColor;
  }

  .buffered {
    font-weight: 500;
    opacity: 0.9;
  }
</style>
