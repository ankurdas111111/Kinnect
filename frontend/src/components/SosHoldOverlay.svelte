<script>
  /**
   * SosHoldOverlay — the "hold to send" affordance for SOS (Hearth 04a),
   * restyled to the Stitch "SOS Hold Trigger" spec: warm dotted paper scrim,
   * beacon halos, a 240-unit radial stroke countdown ring around a vermilion
   * disc, and the reference's copy register ("Hold steady…", "Release anytime
   * to safely cancel without sending.").
   *
   * The design replaces the confirm dialog with a hold: "impossible to send
   * by accident, no confirm dialog". While the user holds, this overlay shows
   * a filling ring, a countdown, and exactly what will happen.
   *
   * Presentational only — MainApp owns the timer and the socket emit, so the
   * emergency path stays in one place and this component can never fire an
   * SOS on its own. Everything here is pointer-events: none; the disc is a
   * mirror of the FAB being held, not a second trigger.
   *
   * Props:
   *   holding    — hold in progress
   *   progress   — 0..1 through the hold
   *   secondsLeft— whole seconds remaining (displayed in the ring)
   *   recipients — human list, e.g. "Arjun, Meera and Nani"
   */
  /** @type {{ holding?: boolean, progress?: number, secondsLeft?: number, recipients?: string }} */
  let { holding = false, progress = 0, secondsLeft = 2, recipients = '' } = $props();

  /* Stitch ring geometry: 240-unit viewBox, r 110, stroke 12 (CIRC ≈ 691.15) */
  const R = 110;
  const CIRC = 2 * Math.PI * R;
  let dash = $derived(CIRC * (1 - Math.min(Math.max(progress, 0), 1)));
</script>

{#if holding}
  <div class="sos-hold" role="status" aria-live="assertive">
    <div class="sos-hold-inner">
      <div class="sos-hold-ring" aria-hidden="true">
        <!-- Breathing beacon halos (transform/opacity only — GPU) -->
        <span class="sos-hold-beacon"></span>
        <span class="sos-hold-halo"></span>

        <!-- Radial stroke countdown ring -->
        <svg viewBox="0 0 240 240" class="sos-hold-svg">
          <circle class="sos-hold-track" cx="120" cy="120" r={R} />
          <circle
            class="sos-hold-arc"
            cx="120" cy="120" r={R}
            style="stroke-dasharray:{CIRC};stroke-dashoffset:{dash}"
          />
        </svg>

        <!-- Vermilion disc — mirrors the FAB under the user's thumb -->
        <span class="sos-hold-disc">
          <span class="sos-hold-mark">SOS</span>
          <span class="sos-hold-count">{secondsLeft}</span>
          <span class="sos-hold-prompt">Hold to alert</span>
        </span>
      </div>

      <p class="sos-hold-title verdict-voice">Hold steady&hellip;</p>
      <p class="sos-hold-sub">Release anytime to safely cancel without sending.</p>
      <p class="sos-hold-detail">
        Sends your live location{#if recipients}&nbsp;to <strong>{recipients}</strong>{:else}&nbsp;to your family{/if},
        and keeps sharing until you say you&rsquo;re safe.
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
    /* Warm dotted-paper scrim (Stitch .bg-sos-canvas), token-built */
    background-color: color-mix(in oklch, var(--surface-0) 86%, transparent);
    background-image: radial-gradient(
      color-mix(in oklch, var(--text-primary) 10%, transparent) 1px,
      transparent 1px
    );
    background-size: var(--space-6) var(--space-6);
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
    /* The overlay is feedback, not a target — never swallow the pointer that
       is mid-hold on the FAB underneath. Release = cancel, always. */
    pointer-events: none;
  }

  .sos-hold-inner {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: var(--space-2);
    max-width: 34ch;
  }

  /* ── Ring assembly ──────────────────────────────────────────────────── */
  .sos-hold-ring {
    position: relative;
    width: min(240px, 60vw);
    aspect-ratio: 1;
    margin-bottom: var(--space-3);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  /* Outer breathing beacon — transform/opacity only, GPU-composited */
  .sos-hold-beacon {
    position: absolute;
    inset: -10%;
    border-radius: var(--radius-full);
    background: color-mix(in oklch, var(--danger-500) 12%, transparent);
    animation: sos-hold-beacon 2.2s var(--ease-out, ease-out) infinite;
    will-change: transform, opacity;
  }
  @keyframes sos-hold-beacon {
    0%   { transform: scale(0.86); opacity: 0.9; }
    70%  { transform: scale(1.12); opacity: 0;   }
    100% { transform: scale(1.12); opacity: 0;   }
  }

  /* Inner static halo (Stitch's second vermilion/20 ring) */
  .sos-hold-halo {
    position: absolute;
    inset: 2%;
    border-radius: var(--radius-full);
    background: color-mix(in oklch, var(--danger-500) 16%, transparent);
  }

  .sos-hold-svg {
    position: relative;
    width: 100%;
    height: 100%;
    transform: rotate(-90deg);
  }
  .sos-hold-track {
    fill: none;
    stroke: color-mix(in oklch, var(--border-strong) 40%, transparent);
    stroke-width: 12;
  }
  .sos-hold-arc {
    fill: none;
    stroke: var(--danger-500);
    stroke-width: 12;
    stroke-linecap: round;
    /* dashoffset is a paint property, but it is the standard way to draw a
       radial countdown; progress updates every frame from MainApp's rAF and
       this short linear transition keeps the arc butter-smooth between them. */
    transition: stroke-dashoffset 100ms linear;
  }

  /* ── Vermilion disc (display-only mirror of the held FAB) ───────────── */
  .sos-hold-disc {
    position: absolute;
    inset: 12.5%;
    border-radius: var(--radius-full);
    background: var(--danger-500);
    border: var(--space-1) solid var(--surface-0);
    box-shadow: var(--shadow-danger);
    color: var(--text-on-danger);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-0-5);
  }
  .sos-hold-mark {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 800;
    letter-spacing: 0.18em;
    opacity: 0.9;
  }
  .sos-hold-count {
    font-family: var(--font-display);
    font-size: var(--text-4xl);
    font-weight: 700;
    line-height: var(--leading-tight);
    letter-spacing: -0.02em;
    font-variant-numeric: tabular-nums;
  }
  .sos-hold-prompt {
    font-family: var(--font-sans);
    font-size: var(--text-2xs);
    font-weight: 600;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    opacity: 0.9;
  }

  /* ── Copy — verdict voice + functional rigor ────────────────────────── */
  .sos-hold-title {
    margin: 0;
    font-family: var(--font-serif, var(--font-display));
    font-style: italic;
    font-size: var(--text-2xl, 1.375rem);
    color: var(--text-primary);
  }
  .sos-hold-sub {
    margin: 0;
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--danger-600, var(--danger-500));
  }
  /* Night: --danger-600 darkens; lift the cancel line so it stays legible */
  :global(:root[data-theme="dark"]) .sos-hold-sub {
    color: var(--danger-400, var(--danger-500));
  }
  .sos-hold-detail {
    margin: var(--space-2) 0 0;
    font-size: var(--text-sm);
    line-height: var(--leading-normal);
    color: var(--text-secondary);
  }
  .sos-hold-detail strong {
    color: var(--text-primary);
    font-weight: 600;
  }

  /* Reduced motion: no arc tween, no beacon — the numeric countdown and the
     stepwise arc position still communicate progress clearly. */
  @media (prefers-reduced-motion: reduce) {
    .sos-hold-arc { transition: none; }
    .sos-hold-beacon { animation: none; opacity: 0.5; transform: scale(1); }
  }
</style>
