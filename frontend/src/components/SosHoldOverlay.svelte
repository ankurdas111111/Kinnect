<script>
  /**
   * SosHoldOverlay — the "hold to send" affordance for SOS (Hearth 04a).
   *
   * The design replaces the confirm dialog with a 2-second hold: "impossible
   * to send by accident, no confirm dialog". While the user holds, this
   * overlay shows a filling ring, a countdown, and exactly what will happen.
   *
   * Presentational only — MainApp owns the timer and the socket emit, so the
   * emergency path stays in one place and this component can never fire an
   * SOS on its own.
   *
   * Props:
   *   holding    — hold in progress
   *   progress   — 0..1 through the hold
   *   secondsLeft— whole seconds remaining (displayed in the ring)
   *   recipients — human list, e.g. "Arjun, Meera and Nani"
   */
  /** @type {{ holding?: boolean, progress?: number, secondsLeft?: number, recipients?: string }} */
  let { holding = false, progress = 0, secondsLeft = 2, recipients = '' } = $props();

  const R = 54;
  const CIRC = 2 * Math.PI * R;
  let dash = $derived(CIRC * (1 - Math.min(Math.max(progress, 0), 1)));
</script>

{#if holding}
  <div class="sos-hold" role="status" aria-live="assertive">
    <div class="sos-hold-inner">
      <div class="sos-hold-ring" aria-hidden="true">
        <svg viewBox="0 0 128 128" class="sos-hold-svg">
          <circle class="sos-hold-track" cx="64" cy="64" r={R} />
          <circle
            class="sos-hold-arc"
            cx="64" cy="64" r={R}
            style="stroke-dasharray:{CIRC};stroke-dashoffset:{dash}"
          />
        </svg>
        <span class="sos-hold-count">{secondsLeft}</span>
      </div>
      <p class="sos-hold-title">Keep holding…</p>
      <p class="sos-hold-sub">Release to cancel.</p>
      <p class="sos-hold-detail">
        Sends your live location{recipients ? ` to ${recipients}` : ' to your family'}, and keeps
        sharing until you say you're safe.
      </p>
    </div>
  </div>
{/if}

<style>
  .sos-hold {
    position: fixed;
    inset: 0;
    z-index: var(--z-modal, 5000);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-6);
    background: color-mix(in oklch, var(--surface-0) 78%, transparent);
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
    /* The overlay is feedback, not a target — never swallow the pointer that
       is mid-hold on the FAB underneath. */
    pointer-events: none;
  }

  .sos-hold-inner {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: var(--space-2);
    max-width: 32ch;
  }

  .sos-hold-ring {
    position: relative;
    width: 128px;
    height: 128px;
    margin-bottom: var(--space-2);
  }
  .sos-hold-svg { width: 100%; height: 100%; transform: rotate(-90deg); }
  .sos-hold-track {
    fill: none;
    stroke: color-mix(in oklch, var(--danger-500) 18%, transparent);
    stroke-width: 6;
  }
  .sos-hold-arc {
    fill: none;
    stroke: var(--danger-500);
    stroke-width: 6;
    stroke-linecap: round;
    /* dashoffset is a paint property, but it is the only way to draw a partial
       arc; it updates at most a few times per second here, not per frame. */
    transition: stroke-dashoffset 100ms linear;
  }
  .sos-hold-count {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-size: 2.75rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    color: var(--danger-500);
  }

  .sos-hold-title {
    margin: 0;
    font-family: var(--font-serif, var(--font-display));
    font-style: italic;
    font-size: var(--text-2xl, 1.5rem);
    color: var(--text-primary);
  }
  .sos-hold-sub {
    margin: 0;
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--danger-500);
  }
  .sos-hold-detail {
    margin: var(--space-2) 0 0;
    font-size: var(--text-sm);
    line-height: 1.5;
    color: var(--text-secondary);
  }

  @media (prefers-reduced-motion: reduce) {
    .sos-hold-arc { transition: none; }
  }
</style>
