<script>
  /**
   * CompletenessMeter — SVG progress ring for the emergency profile. Progress
   * semantics (how full the form is), deliberately NOT CountdownRing (which
   * encodes temporal decay). Stays local to components/emergency/*.
   *
   * Presentational: props only.
   *   progress    — 0–100 integer
   *   filledCount — number of filled tracked fields
   *   totalFields — total tracked fields
   *   lastUpdated — formatted "Updated …" string or null
   */

  /** @type {{ progress: number, filledCount: number, totalFields: number, lastUpdated?: string | null }} */
  let { progress, filledCount, totalFields, lastUpdated = null } = $props();

  const RING_R = 20;
  const RING_CIRC = 2 * Math.PI * RING_R;
  let ringOffset = $derived(RING_CIRC * (1 - progress / 100));
  let isComplete = $derived(progress === 100);
</script>

<div class="ep-meter-card" aria-live="polite">
  <div
    class="ep-meter-ring"
    role="progressbar"
    aria-valuenow={progress}
    aria-valuemin="0"
    aria-valuemax="100"
    aria-label="Profile completion"
  >
    <svg viewBox="0 0 44 44" width="56" height="56" aria-hidden="true">
      <circle class="ep-ring-track" cx="22" cy="22" r={RING_R} />
      <circle
        class="ep-ring-fill"
        class:ep-ring-fill--complete={isComplete}
        cx="22" cy="22" r={RING_R}
        stroke-dasharray={RING_CIRC}
        stroke-dashoffset={ringOffset}
      />
    </svg>
    <span class="ep-ring-pct">{progress}<span class="ep-ring-pct-sign">%</span></span>
  </div>

  <div class="ep-meter-info">
    <span class="ep-meter-count">{filledCount} of {totalFields} key fields complete</span>
    {#if lastUpdated}
      <span class="ep-last-updated">Updated {lastUpdated}</span>
    {/if}
  </div>
</div>

<style>
  .ep-meter-card {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-4);
    border-radius: var(--radius-xl, 20px);
    background: var(--surface-2);
    border: 1px solid var(--border-default);
    box-shadow: var(--shadow-sm);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
  }
  .ep-meter-ring {
    position: relative;
    flex-shrink: 0;
    width: 56px;
    height: 56px;
    display: grid;
    place-items: center;
  }
  .ep-meter-ring svg {
    display: block;
    transform: rotate(-90deg);
  }
  .ep-ring-track {
    fill: none;
    stroke: var(--gray-200, var(--border-default));
    stroke-width: 4;
  }
  :global([data-theme='dark']) .ep-ring-track {
    stroke: var(--border-default);
  }
  .ep-ring-fill {
    fill: none;
    stroke: var(--primary-500);
    stroke-width: 4;
    stroke-linecap: round;
    /* stroke-dashoffset is an SVG attribute animation — cheap, functional
       progress feedback. Survives reduced-motion as an instant jump. */
    transition: stroke-dashoffset 400ms var(--ease-out), stroke 250ms var(--ease-out);
  }
  .ep-ring-fill--complete { stroke: var(--success-500); }
  /* Number + % render as one unit — same color/baseline, not a detached pair */
  .ep-ring-pct {
    position: absolute;
    inset: 0;
    display: grid;
    place-items: center;
    font-size: var(--text-xl);
    font-weight: 700;
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
  }
  .ep-ring-pct-sign {
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    margin-left: 1px;
  }
  .ep-meter-info {
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5);
    min-width: 0;
  }
  .ep-meter-count {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text-secondary);
  }
  .ep-last-updated {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }

  @media (prefers-reduced-motion: reduce) {
    .ep-ring-fill { transition: none; }
  }
</style>
