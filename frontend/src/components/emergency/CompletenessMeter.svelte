<script>
  /**
   * CompletenessMeter — SVG progress ring + status badge for the emergency
   * profile. Progress semantics (how full the form is), deliberately NOT
   * CountdownRing (which encodes temporal decay). Stays local to
   * components/emergency/*.
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
    <span class="ep-badge" class:ep-badge--complete={isComplete} class:ep-badge--incomplete={!isComplete}>
      {#if isComplete}
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
        Profile Complete
      {:else}
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        Incomplete
      {/if}
    </span>
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
  .ep-ring-pct {
    position: absolute;
    inset: 0;
    display: grid;
    place-items: center;
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
  }
  .ep-ring-pct-sign {
    font-size: 0.62em;
    font-weight: 600;
    color: var(--text-tertiary);
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

  /* ── Status badge ─────────────────────────────────────────────────────── */
  .ep-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1-5);
    align-self: flex-start;
    padding: var(--space-1) var(--space-2-5);
    border-radius: var(--radius-full, 9999px);
    font-size: var(--text-xs);
    font-weight: 700;
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }
  .ep-badge--complete {
    background: var(--success-500-12);
    color: var(--success-600);
    border: 1px solid var(--success-500-20);
  }
  :global([data-theme='dark']) .ep-badge--complete {
    color: var(--success-400);
  }
  .ep-badge--incomplete {
    background: color-mix(in oklch, var(--warning-500) 12%, transparent);
    color: var(--warning-600);
    border: 1px solid color-mix(in oklch, var(--warning-500) 22%, transparent);
  }
  :global([data-theme='dark']) .ep-badge--incomplete {
    color: var(--warning-400);
  }

  @media (prefers-reduced-motion: reduce) {
    .ep-ring-fill { transition: none; }
  }
</style>
