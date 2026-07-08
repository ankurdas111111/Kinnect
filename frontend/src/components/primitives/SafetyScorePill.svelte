<script>
  
  /**
   * @typedef {Object} Props
   * @property {any} [score] - Props - number 0-100 or null
   * @property {any} [breakdown] - object with component scores or null
   */

  /** @type {Props} */
  let { score = null, breakdown = null } = $props();

  let tooltipVisible = $state(false);

  let pillClass = $derived(score == null
    ? 'pill-gray'
    : score >= 85 ? 'pill-green'
    : score >= 60 ? 'pill-amber'
    : 'pill-red');

  let isPulse = $derived(score != null && score < 60);

  function formatScore(s) {
    return s == null ? '—' : Math.round(s);
  }

  function toggleTooltip() {
    if (breakdown) tooltipVisible = !tooltipVisible;
  }

  function handleKeydown(e) {
    if (e.key === 'Enter' || e.key === ' ') toggleTooltip();
    if (e.key === 'Escape') tooltipVisible = false;
  }

  const LABELS = {
    gps: 'GPS Accuracy',
    silence: 'Signal Freshness',
    checkIn: 'Check-in',
    geofence: 'Geofence',
    attest: 'Attestation',
  };
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<span
  class="safety-pill {pillClass}"
  class:pill-pulse={isPulse}
  class:pill-clickable={breakdown != null}
  onclick={toggleTooltip}
  onkeydown={handleKeydown}
  role={breakdown ? 'button' : 'status'}
  tabindex={breakdown ? 0 : -1}
  aria-label="Safety score {formatScore(score)} out of 100{breakdown ? '. Click for breakdown.' : ''}"
>
  {formatScore(score)}
</span>

{#if tooltipVisible && breakdown}
  <div class="score-tooltip" role="tooltip">
    <div class="tooltip-title">Score Breakdown</div>
    {#each Object.entries(breakdown) as [key, val]}
      {#if key !== 'total' && LABELS[key]}
        <div class="tooltip-row">
          <span class="tooltip-label">{LABELS[key]}</span>
          <span class="tooltip-val">{Math.round(val)}</span>
        </div>
      {/if}
    {/each}
    <div class="tooltip-divider"></div>
    <div class="tooltip-row tooltip-total">
      <span class="tooltip-label">Total</span>
      <span class="tooltip-val">{formatScore(score)}</span>
    </div>
  </div>
{/if}

<style>
  .safety-pill {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 32px;
    height: 20px;
    padding: 0 7px;
    border-radius: var(--radius-full);
    font-size: 11px;
    font-weight: 800;
    letter-spacing: -0.01em;
    font-variant-numeric: tabular-nums;
    line-height: 1;
    transition: opacity 150ms;
    position: relative;
  }
  .pill-clickable { cursor: pointer; }
  .pill-clickable:hover { opacity: 0.82; }

  .pill-green { background: rgba(16, 185, 129, 0.16); color: var(--success-500); border: 1px solid rgba(16, 185, 129, 0.28); }
  .pill-amber { background: rgba(245, 158, 11, 0.14); color: var(--warning-500); border: 1px solid rgba(245, 158, 11, 0.26); }
  .pill-red   { background: rgba(239, 68, 68, 0.14);  color: var(--danger-500);  border: 1px solid rgba(239, 68, 68, 0.26); }
  .pill-gray  { background: var(--surface-2); color: var(--text-tertiary); border: 1px solid var(--border-subtle); }

  .pill-pulse { animation: sos-urgent-pulse 2.5s ease-in-out infinite; }

  /* Tooltip */
  .score-tooltip {
    position: absolute;
    z-index: 200;
    top: calc(100% + 6px);
    left: 50%;
    transform: translateX(-50%);
    background: var(--surface-2);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-lg);
    padding: var(--space-2) var(--space-3);
    min-width: 180px;
    box-shadow: var(--shadow-lg);
    animation: sheet-arrive 200ms var(--ease-spring);
  }
  .tooltip-title {
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-tertiary);
    margin-bottom: var(--space-1-5);
  }
  .tooltip-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--space-3);
    padding: 2px 0;
  }
  .tooltip-label { font-size: 11px; color: var(--text-secondary); }
  .tooltip-val   { font-size: 11px; font-weight: 700; font-variant-numeric: tabular-nums; color: var(--text-primary); }
  .tooltip-divider { height: 1px; background: var(--border-subtle); margin: var(--space-1-5) 0; }
  .tooltip-total .tooltip-label { font-weight: 700; color: var(--text-primary); }
</style>
