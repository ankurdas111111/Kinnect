<script>
  /**
   * CountdownRing — conic-gradient temporal decay ring. 1Hz interval only.
   *
   * Props:
   *   deadline — number (epoch ms)       — when the countdown hits zero
   *   total    — number (ms)             — full window for the 100% arc
   *   size     — number | 'hero'         — px; 'hero' = 184px with large numeric center
   *   label    — string                  — accessible name context (role="timer")
   *   onexpire — () => void              — fires once when remaining ≤ 0
   *   children — Snippet                 — center content; hero default renders m:ss if absent
   *
   * PAINT BUDGET: the ring is a conic-gradient background. Updating a conic-gradient
   * is a PAINT operation, not a compositor transform. At 1Hz (setInterval 1000ms)
   * this costs one tiny repaint per second — fine. At rAF (60Hz) it repaints every
   * frame — fatal on mid-range Android WebViews. Do NOT "smooth" this to
   * requestAnimationFrame. If smoothness is ever demanded, the answer is an SVG
   * stroke-dashoffset transform, not a faster interval.
   *
   * A11y: role="timer" on wrapper; aria-label updated at 1Hz without aria-live
   * (timers are polled by AT; a live region here would spam — CONTRACTS.md §9).
   * Color thresholds are paired with the numeric value (text) — never color alone.
   */


  /** @type {{ deadline: number, total: number, size?: number | 'hero', label?: string, onexpire?: () => void, children?: import('svelte').Snippet }} */
  let {
    deadline,
    total,
    size = 48,
    label = 'Time remaining',
    onexpire,
    children,
  } = $props();

  // ── Computed geometry ───────────────────────────────────────────────────────
  const isHero   = $derived(size === 'hero');
  const sizePx   = $derived(isHero ? 184 : Number(size));
  // Inner circle is track - ring width (13px each side)
  const innerPx  = $derived(sizePx - 26);

  // ── Tick state ──────────────────────────────────────────────────────────────
  // Initializer intentionally captures deadline's mount-time value so the
  // first paint shows the real remaining time; the $effect below re-syncs on
  // every deadline change (svelte state_referenced_locally is expected here).
  let remaining = $state(Math.max(0, deadline - Date.now()));
  let expired   = $state(false);

  // Threshold classes (> 25% green, ≤ 25% amber, ≤ 10% red) — CONTRACTS.md §9
  const thresholdClass = $derived.by(() => {
    const frac = total > 0 ? remaining / total : 0;
    if (frac <= 0.10) return 'ring-danger';
    if (frac <= 0.25) return 'ring-warn';
    return 'ring-ok';
  });

  // Conic-gradient fill percentage (0–100)
  const pct = $derived(
    total > 0 ? Math.max(0, Math.min(100, (remaining / total) * 100)) : 0
  );

  // Formatted m:ss for hero default readout and aria-label
  function fmtMss(ms) {
    const s = Math.max(0, Math.ceil(ms / 1000));
    const m = Math.floor(s / 60);
    const sec = s % 60;
    return `${m}:${String(sec).padStart(2, '0')}`;
  }

  const ariaLabel = $derived(`${label}: ${fmtMss(remaining)}`);
  const displayTime = $derived(fmtMss(remaining));

  // ── 1Hz interval — restarted when `deadline` changes (e.g. the user extends
  // a sharing window), cleared via $effect cleanup on change/unmount.
  // NOTE: only `deadline` is read synchronously here — reading `remaining`/
  // `expired` in the effect body would make their interval writes re-trigger
  // the effect every second (tracked-read loop).
  $effect(() => {
    const initial = Math.max(0, deadline - Date.now());
    remaining = initial;
    expired = initial <= 0;
    if (initial <= 0) return;

    const timer = setInterval(() => {
      const rem = deadline - Date.now();
      if (rem <= 0) {
        remaining = 0;
        if (!expired) {
          expired = true;
          onexpire?.();
        }
        clearInterval(timer);
      } else {
        remaining = rem;
      }
    }, 1000);

    return () => clearInterval(timer);
  });
</script>

<div
  class="ring-wrap {thresholdClass}"
  class:hero={isHero}
  role="timer"
  aria-label={ariaLabel}
  style="
    --size: {sizePx}px;
    --inner: {innerPx}px;
    --pct: {pct}%;
  "
>
  <!-- Static track layer (surface) + conic fill (separate bg on wrapper) -->
  <div class="ring-inner">
    {#if children}
      {@render children()}
    {:else if isHero}
      <!-- Hero default: large monospace countdown — CONTRACTS.md §9 -->
      <span class="ring-value {thresholdClass}-text">{displayTime}</span>
      <span class="ring-caption">remaining</span>
    {/if}
  </div>

  <!-- Danger glow pulse — opacity-only, GPU safe, class fx-ambient -->
  {#if thresholdClass === 'ring-danger'}
    <div class="danger-glow fx-ambient" aria-hidden="true"></div>
  {/if}
</div>

<style>
  /* ── Wrapper: conic-gradient ring ───────────────────────────────────────── */
  .ring-wrap {
    position: relative;
    width: var(--size);
    height: var(--size);
    border-radius: 50%;
    display: grid;
    place-items: center;
    flex-shrink: 0;
    /* Color crossfade 600ms ease-out on threshold crossing — one paint per
       threshold, not a loop — CONTRACTS.md §9 */
    transition: background 600ms var(--ease-out, ease);
  }

  /* Threshold color tokens — CONTRACTS.md §9 */
  .ring-ok     { background: conic-gradient(from -90deg, var(--status-live)  var(--pct), var(--surface-inset) 0); }
  .ring-warn   { background: conic-gradient(from -90deg, var(--warning-500)  var(--pct), var(--surface-inset) 0); }
  .ring-danger { background: conic-gradient(from -90deg, var(--danger-500)   var(--pct), var(--surface-inset) 0); }

  /* ── Inner track (donut hole) ────────────────────────────────────────────── */
  .ring-inner {
    position: relative;
    z-index: 1;
    width: var(--inner);
    height: var(--inner);
    border-radius: 50%;
    background: var(--surface-1);
    border: 1px solid var(--border-subtle);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
  }

  /* ── Hero size (184px) ───────────────────────────────────────────────────── */
  .ring-wrap.hero {
    margin: var(--space-2) 0;
  }

  /* Monospace numeric value — CONTRACTS.md §9 */
  .ring-value {
    font-family: var(--font-mono);
    font-size: clamp(1.375rem, 6vw, 1.875rem);
    font-weight: 900;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.02em;
    line-height: 1.05;
    text-align: center;
    /* Color transition on value (text only, not the ring) — one crossfade per threshold */
    transition: color 300ms var(--ease-out);
  }

  /* Threshold text colors — paired with numeric to satisfy "never color alone" */
  .ring-ok-text     { color: var(--status-live); }
  .ring-warn-text   { color: var(--warning-500); }
  .ring-danger-text { color: var(--danger-500); }

  .ring-caption {
    font-size: var(--text-2xs);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--text-tertiary);
  }

  /* ── Danger glow pulse — opacity only (GPU), class fx-ambient ───────────── */
  .danger-glow {
    position: absolute;
    inset: -8px;
    border-radius: 50%;
    /* Box-shadow only paints once per frame during the opacity keyframe —
       acceptable; this is NOT a box-shadow transition on a loop */
    box-shadow: 0 0 30px 4px color-mix(in oklch, var(--danger-500) 20%, transparent);
    pointer-events: none;
    animation: ring-glow 1.4s var(--ease-in-out, ease-in-out) infinite;
  }

  @keyframes ring-glow {
    0%, 100% { opacity: 0.35; }
    50%       { opacity: 1; }
  }

  /* ── Reduced motion: static ring + numbers keep ticking (functional) ─────── */
  @media (prefers-reduced-motion: reduce) {
    .danger-glow { animation: none; opacity: 0.6; }
    .ring-wrap { transition: none; }
    .ring-value { transition: none; }
  }
</style>
