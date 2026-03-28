<script>
  import { onMount, onDestroy } from 'svelte';

  export let lastUpdateAt = null; // unix ms
  export let movementPhase = null; // 'stationary' | 'walking' | 'driving' | 'transit'

  // Decay constants (seconds) per movement phase
  const DECAY = {
    stationary: 60,
    walking: 180,
    driving: 600,
    transit: 900,
    default: 300,
  };

  let now = Date.now();
  let intervalId;

  $: elapsed = lastUpdateAt ? (now - lastUpdateAt) / 1000 : Infinity;
  $: k = 1 / (DECAY[movementPhase] ?? DECAY.default);
  $: confidence = lastUpdateAt ? Math.exp(-k * elapsed) : 0;
  $: dots = Math.round(confidence * 5);
  $: dotColor = dots >= 4 ? 'dot-green' : dots >= 2 ? 'dot-amber' : 'dot-red';

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
