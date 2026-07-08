<script>
  import { onMount, onDestroy } from 'svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [lastUpdateAt] - unix ms
   * @property {any} [movementPhase] - 'stationary' | 'walking' | 'driving' | 'transit'
   */

  /** @type {Props} */
  let { lastUpdateAt = null, movementPhase = null } = $props();

  // Decay constants (seconds) per movement phase
  const DECAY = {
    stationary: 60,
    walking: 180,
    driving: 600,
    transit: 900,
    default: 300,
  };

  let now = $state(Date.now());
  let intervalId;

  let elapsed = $derived(lastUpdateAt ? (now - lastUpdateAt) / 1000 : Infinity);
  let k = $derived(1 / (DECAY[movementPhase] ?? DECAY.default));
  let confidence = $derived(lastUpdateAt ? Math.exp(-k * elapsed) : 0);
  let dots = $derived(Math.round(confidence * 5));
  let dotColor = $derived(dots >= 4 ? 'dot-green' : dots >= 2 ? 'dot-amber' : 'dot-red');

  onMount(() => {
    intervalId = setInterval(() => { now = Date.now(); }, 10_000);
  });
  onDestroy(() => {
    if (intervalId) clearInterval(intervalId);
  });
</script>

<div class="confidence-bar" aria-label="Signal confidence {dots} of 5" role="meter" aria-valuenow={dots} aria-valuemin={0} aria-valuemax={5}>
  {#each Array(5) as _, i}
    <span class="conf-dot {i < dots ? dotColor : 'dot-empty'}"></span>
  {/each}
</div>

<style>
  .confidence-bar {
    display: inline-flex;
    align-items: center;
    gap: 3px;
  }
  .conf-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    flex-shrink: 0;
    transition: background 400ms var(--ease-out);
  }
  .dot-green  { background: var(--success-500); }
  .dot-amber  { background: var(--warning-500); }
  .dot-red    { background: var(--danger-500); }
  .dot-empty  { background: var(--border-default); }
</style>
