<script>
  /**
   * SignalBars — 3-bar freshness strength (CONTRACTS.md §6).
   * Extracted from LiveViewer .signal-bars. Level N lights bars 1..N —
   * count-of-lit-bars is the shape cue (grayscale-safe). Color comes from
   * the consumer via currentColor.
   */

  /** @type {{ level?: 0 | 1 | 2 | 3, label?: string }} */
  let { level = 0, label = '' } = $props();
</script>

<span
  class="signal-bars"
  data-level={level}
  role={label ? 'img' : undefined}
  aria-label={label || undefined}
  aria-hidden={label ? undefined : 'true'}
>
  <i></i><i></i><i></i>
</span>

<style>
  .signal-bars {
    display: inline-flex;
    align-items: flex-end;
    gap: calc(var(--space-1) / 2);
    height: var(--space-3);
  }

  .signal-bars i {
    /* Hairline data-viz width — deliberately below the 4px layout grid. */
    width: 3px;
    border-radius: var(--radius-xs, 1px);
    background: currentColor;
    opacity: var(--opacity-dim);
    /* State transition, not decoration — survives reduced-motion. */
    transition: opacity var(--duration-slow, 300ms) var(--ease-out, cubic-bezier(0.4, 0, 0.2, 1));
  }

  .signal-bars i:nth-child(1) { height: 42%; }
  .signal-bars i:nth-child(2) { height: 67%; }
  .signal-bars i:nth-child(3) { height: 100%; }

  .signal-bars[data-level='1'] i:nth-child(1),
  .signal-bars[data-level='2'] i:nth-child(-n + 2),
  .signal-bars[data-level='3'] i {
    opacity: 1;
  }
</style>
