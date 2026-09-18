<script>
  import { myLocation, tracking } from '../../lib/stores/map.js';
  import { trackingMetrics } from '../../lib/stores/metrics.js';
  import { latencyMetrics } from '../../lib/stores/latency.js';

  
  /**
   * @typedef {Object} Props
   * @property {boolean} [shell] - Panel-shell mode shows the "top bar" hint variant.
   */

  /** @type {Props} */
  let { shell = false } = $props();

  let statsOpen = $state(false);
  let debugTaps = 0;

  function tapGps() {
    debugTaps++;
    if (debugTaps >= 5) { statsOpen = !statsOpen; debugTaps = 0; }
  }

  let accClass = $derived($trackingMetrics.lastAccuracy != null
    ? $trackingMetrics.lastAccuracy <= 15 ? 'green'
    : $trackingMetrics.lastAccuracy <= 50 ? 'yellow' : 'red'
    : '');
  let accLabel = $derived($trackingMetrics.lastAccuracy != null
    ? ($trackingMetrics.lastAccuracy <= 15 ? 'Sharp' : $trackingMetrics.lastAccuracy <= 50 ? 'Decent' : 'Rough') + ` ±${$trackingMetrics.lastAccuracy}m`
    : $tracking ? 'Getting location...' : 'Tap Track to start');
</script>

<!-- ── GPS LIVE STATUS ──────────────────────────────────────────── -->
{#if $myLocation}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div class="gps-live-card" class:is-tracking={$tracking} onclick={tapGps} role="presentation">
    <div class="gps-signal-left">
      <span class="gps-ping" class:active={$tracking && $trackingMetrics.lastAccuracy != null} aria-hidden="true"></span>
      <div class="gps-coord-block">
        <span class="gps-accuracy-label" aria-live="polite" aria-atomic="true">{accLabel}</span>
        <div class="gps-sub" aria-hidden="true">
          {#if $trackingMetrics.lastAccuracy != null}
            <span class="accuracy-dot {accClass}"></span>
          {/if}
          <span>{$myLocation.formattedTime || 'Live'}</span>
        </div>
      </div>
    </div>
    {#if ($myLocation.speed || 0) >= 1}
      <div class="speed-pill" aria-live="polite" aria-atomic="true" aria-label="{Math.round($myLocation.speed)} km/h">
        <span class="speed-num" aria-hidden="true">{Math.round($myLocation.speed)}</span>
        <span class="speed-unit" aria-hidden="true">km/h</span>
      </div>
    {/if}
  </div>
{:else}
  <div class="gps-acquire-card animate-breathe">
    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="acquire-icon" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M16.24 7.76a6 6 0 0 1 0 8.49"/><path d="M7.76 7.76a6 6 0 0 0 0 8.49"/><path d="M20.49 3.51a12 12 0 0 1 0 16.97"/><path d="M3.51 3.51a12 12 0 0 0 0 16.97"/></svg>
    <div>
      <p class="acquire-title">Getting your location...</p>
      {#if shell}
        <p class="acquire-hint">Tap <strong>Track</strong> in the top bar to share your location.</p>
      {:else}
        <p class="acquire-hint">Tap <strong>Track</strong> to share your location</p>
      {/if}
    </div>
  </div>
{/if}

<!-- ── DEBUG STATS (hidden tap-to-reveal) ─────────────────────── -->
{#if statsOpen && $tracking && $trackingMetrics.fixCount > 0}
  <div class="tracking-stats">
    <div class="stat-row"><span>Accuracy</span><span>{$trackingMetrics.lastAccuracy ?? '-'}m (avg {$trackingMetrics.avgAccuracy ?? '-'}m)</span></div>
    <div class="stat-row"><span>Fixes</span><span>{$trackingMetrics.fixCount}</span></div>
    <div class="stat-row"><span>Rate</span><span>{$trackingMetrics.updatesPerSec}/s</span></div>
    <div class="stat-row"><span>Kalman</span><span>{$trackingMetrics.kalmanCorrectionM}m correction</span></div>
    <div class="stat-row"><span>Filter</span><span>{$trackingMetrics.filterState}</span></div>
    {#if $latencyMetrics.lastE2eMs != null}
      <div class="stat-row"><span>E2E Latency</span><span class="latency-value" class:latency-good={$latencyMetrics.lastE2eMs < 300} class:latency-ok={$latencyMetrics.lastE2eMs >= 300 && $latencyMetrics.lastE2eMs < 800} class:latency-bad={$latencyMetrics.lastE2eMs >= 800}>{$latencyMetrics.lastE2eMs}ms (avg {$latencyMetrics.avgE2eMs}ms)</span></div>
    {/if}
    {#if $latencyMetrics.lastServerHopMs != null}
      <div class="stat-row"><span>Server Hop</span><span>{$latencyMetrics.lastServerHopMs}ms</span></div>
    {/if}
  </div>
{/if}

<style>
  /* ── GPS Live Card ──────────────────────────────────────────────── */
  .gps-live-card {
    background: var(--surface-2);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-card, 20px);
    padding: var(--space-4);
    display: flex;
    align-items: center;
    gap: var(--space-3);
    cursor: default;
    transition:
      border-color 500ms var(--ease-out),
      background 500ms var(--ease-out),
      box-shadow 500ms var(--ease-out);
    user-select: none;
  }
  /* Live = a quiet sage wash, not a glow rig — sage says "settled". */
  .gps-live-card.is-tracking {
    border-color: color-mix(in oklch, var(--success-500) 25%, transparent);
    background: color-mix(in oklch, var(--success-500) 7%, transparent);
  }
  .gps-accuracy-label {
    font-family: var(--font-display);
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--text-primary);
    letter-spacing: -0.01em;
    line-height: 1.2;
  }
  .gps-signal-left {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex: 1;
    min-width: 0;
  }

  /* Pulsing GPS dot */
  .gps-ping {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--text-tertiary);
    flex-shrink: 0;
    position: relative;
    transition: background 400ms var(--ease-out);
  }
  .gps-ping.active {
    background: var(--success-500);
    box-shadow: 0 0 8px color-mix(in oklch, var(--success-500) 40%, transparent);
    animation: gps-pulse 2.2s ease-in-out infinite;
  }
  .gps-ping.active::before {
    content: '';
    position: absolute;
    inset: -4px;
    border-radius: 50%;
    background: color-mix(in oklch, var(--success-500) 22%, transparent);
    animation: gps-ring 2.2s ease-out infinite;
  }
  @keyframes gps-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.55; }
  }
  @keyframes gps-ring {
    0%   { transform: scale(1); opacity: 0.6; }
    100% { transform: scale(2.8); opacity: 0; }
  }

  .gps-coord-block {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .gps-sub {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    font-weight: 500;
    color: var(--text-tertiary);
    white-space: nowrap;
  }

  /* Accuracy dot (reuse from global pattern) */
  .accuracy-dot {
    display: inline-block;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--status-offline);
    flex-shrink: 0;
  }
  /* Rough accuracy is a "needs a look", never an alarm — ochre, not red. */
  .accuracy-dot.green  { background: var(--success-500); }
  .accuracy-dot.yellow { background: var(--warning-500); }
  .accuracy-dot.red    { background: var(--warning-700); }

  /* Speed badge */
  .speed-pill {
    margin-left: auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0;
    background: var(--primary-500-12, color-mix(in oklch, var(--primary-500) 12%, transparent));
    border-radius: var(--radius-input);
    padding: var(--space-1-5) var(--space-3);
    flex-shrink: 0;
    min-width: 52px;
    text-align: center;
  }
  .speed-num {
    font-size: var(--text-xl);
    font-weight: 700;
    color: var(--primary-700);
    line-height: 1;
    letter-spacing: -0.02em;
    font-variant-numeric: tabular-nums;
  }
  .speed-unit {
    font-size: 11px;
    color: var(--text-tertiary);
    font-weight: 500;
    letter-spacing: 0.04em;
  }

  /* GPS Acquiring state */
  .gps-acquire-card {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-4);
    background: var(--surface-2);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-card, 20px);
  }
  .acquire-icon { color: var(--primary-700); flex-shrink: 0; }
  .acquire-title {
    font-family: var(--font-display);
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
  }
  .acquire-hint {
    font-size: var(--text-sm);
    color: var(--text-secondary);
    margin: 2px 0 0;
  }
  .acquire-hint strong { color: var(--text-primary); font-weight: 600; }

  /* Debug stats */
  .tracking-stats {
    padding: var(--space-2) var(--space-3);
    background: var(--surface-inset);
    border-radius: var(--radius-md);
    font-size: var(--text-2xs);
    color: var(--text-secondary);
  }
  .stat-row {
    display: flex;
    justify-content: space-between;
    padding: 1px 0;
  }
  .stat-row span:first-child { font-weight: 600; }
  /* Latency grades stay sage/ochre — a slow socket is not an emergency. */
  .latency-good { color: var(--success-600); font-weight: 600; }
  .latency-ok   { color: var(--warning-600); font-weight: 600; }
  .latency-bad  { color: var(--warning-700); font-weight: 700; }

  @media (prefers-reduced-motion: reduce) {
    .gps-ping.active,
    .gps-ping.active::before { animation: none; }
  }
</style>
