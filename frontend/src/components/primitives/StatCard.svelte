<script>
  /**
   * StatCard — mono-numeral stat tile. Calm, no loops, no hover lift.
   * The container (Card / bento) owns elevation; this component owns typography.
   *
   * Props:
   *   label — string (required)         — uppercase micro-label
   *   value — string | number (required) — the big number
   *   unit  — string                    — baseline-aligned suffix
   *   tint  — 'primary' | 'neutral'     — chip background; two tints only (calm core)
   *   icon  — Snippet                   — optional leading glyph (aria-hidden)
   *
   * Value changes: 300ms opacity crossfade only — no movement.
   * People stare at these during emergencies; no jitter — CONTRACTS.md §10.
   */

  /** @type {{ label: string, value: string | number, unit?: string, tint?: 'primary' | 'neutral', icon?: import('svelte').Snippet }} */
  let {
    label,
    value,
    unit = '',
    tint = 'neutral',
    icon,
  } = $props();
</script>

<div class="stat-card tint-{tint}">
  {#if icon}
    <span class="stat-icon" aria-hidden="true">
      {@render icon()}
    </span>
  {/if}

  <div class="stat-body">
    <!-- Value: display font, tabular nums, weight 800 — exact RoutePlayback .stat-val recipe -->
    <p class="stat-val">
      {value}<span class="stat-unit">{unit}</span>
    </p>
    <p class="stat-lbl">{label}</p>
  </div>
</div>

<style>
  /* ── Card shell ─────────────────────────────────────────────────────────── */
  .stat-card {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius-card, 20px);
    border: 1px solid var(--border-subtle);
  }

  /* Two tints only — calm core, CONTRACTS.md §10 */
  .tint-neutral { background: var(--surface-inset); }
  .tint-primary { background: var(--primary-500-12); }

  /* ── Icon ────────────────────────────────────────────────────────────────── */
  .stat-icon {
    display: grid;
    place-items: center;
    flex-shrink: 0;
    color: var(--text-tertiary);
  }

  .stat-icon :global(svg) {
    width: 18px;
    height: 18px;
  }

  /* ── Body ────────────────────────────────────────────────────────────────── */
  .stat-body {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  /* Value — a calm reading, not a KPI shout. CONTRACTS.md §10 */
  .stat-val {
    margin: 0;
    font-family: var(--font-sans);
    font-size: var(--text-lg);
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.01em;
    color: var(--text-primary);
    line-height: 1.1;
    /* 300ms opacity crossfade on value changes — no movement, CONTRACTS.md §10 */
    transition: opacity var(--duration-slow) var(--ease-out);
  }

  /* Unit suffix: smaller, tertiary, baseline-aligned via inline */
  .stat-unit {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text-tertiary);
    margin-left: 2px;
  }

  /* Quiet sentence-case label */
  .stat-lbl {
    margin: 0;
    font-size: 12px;
    color: var(--text-tertiary);
    letter-spacing: 0.02em;
  }

  /* No hover/tilt/loops — CONTRACTS.md §10 */
</style>
