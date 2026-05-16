<script>
  /**
   * Button — Premium interaction component
   *
   * Props:
   *   variant   — 'primary' | 'secondary' | 'ghost' | 'danger' | 'success' (default: 'primary')
   *   size      — 'sm' | 'md' | 'lg' | 'xl'                               (default: 'md')
   *   loading   — shows spinner, disables interaction
   *   disabled  — disables interaction
   *   icon      — slot: icon-only mode when no default slot
   *   fullWidth — stretches to container width
   *   type      — button type attr
   */
  import { spring } from 'svelte/motion';
  import { createEventDispatcher } from 'svelte';

  export let variant  = 'primary';
  export let size     = 'md';
  export let loading  = false;
  export let disabled = false;
  export let fullWidth = false;
  export let type     = 'button';
  export let href     = null; // renders as <a> when set

  const dispatch = createEventDispatcher();

  // Spring for press scale
  const scale = spring(1, { stiffness: 600, damping: 28 });

  // Ripple state
  let ripples = [];
  let btnEl;

  function onPointerDown(e) {
    if (disabled || loading) return;
    scale.set(0.94);
    spawnRipple(e);
  }

  function onPointerUp() {
    scale.set(1);
  }

  function onPointerLeave() {
    scale.set(1);
  }

  function spawnRipple(e) {
    if (!btnEl) return;
    const r   = btnEl.getBoundingClientRect();
    const x   = e.clientX - r.left;
    const y   = e.clientY - r.top;
    const id  = Date.now() + Math.random();
    ripples = [...ripples, { id, x, y }];
    setTimeout(() => {
      ripples = ripples.filter(r => r.id !== id);
    }, 600);
  }

  function handleClick(e) {
    if (disabled || loading) { e.preventDefault(); return; }
    dispatch('click', e);
  }

  $: tag = href ? 'a' : 'button';
</script>

<!-- svelte-ignore a11y-no-static-element-interactions -->
<svelte:element
  this={tag}
  {href}
  {type}
  class="btn btn-{variant} btn-{size}"
  class:full-width={fullWidth}
  class:loading
  class:icon-only={$$slots.icon && !$$slots.default}
  disabled={disabled || loading || undefined}
  aria-disabled={disabled || loading || undefined}
  aria-busy={loading || undefined}
  bind:this={btnEl}
  style="transform: scale({$scale})"
  on:pointerdown={onPointerDown}
  on:pointerup={onPointerUp}
  on:pointerleave={onPointerLeave}
  on:click={handleClick}
>
  <!-- Shimmer sweep layer -->
  <span class="btn-shimmer" aria-hidden="true"></span>

  <!-- Ripple container -->
  <span class="btn-ripple-layer" aria-hidden="true">
    {#each ripples as { id, x, y } (id)}
      <span class="btn-ripple" style="left:{x}px; top:{y}px;"></span>
    {/each}
  </span>

  {#if loading}
    <span class="btn-spinner" aria-hidden="true">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <circle cx="8" cy="8" r="6" stroke="currentColor" stroke-width="2" stroke-dasharray="28" stroke-dashoffset="8"/>
      </svg>
    </span>
  {:else}
    {#if $$slots.icon}
      <span class="btn-icon-wrap"><slot name="icon" /></span>
    {/if}
    {#if $$slots.default}
      <span class="btn-label"><slot /></span>
    {/if}
  {/if}
</svelte:element>

<style>
  /* ── Base ────────────────────────────────────────────────────────────── */
  .btn {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    border: none;
    cursor: pointer;
    font-family: var(--font-display);
    font-weight: 700;
    letter-spacing: 0.01em;
    white-space: nowrap;
    text-decoration: none;
    overflow: hidden;
    user-select: none;
    -webkit-user-select: none;
    -webkit-tap-highlight-color: transparent;
    will-change: transform;
    transition:
      box-shadow 200ms var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1)),
      filter     200ms var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1));
  }

  .btn:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 3px;
  }

  .btn:disabled,
  .btn.loading {
    cursor: not-allowed;
    opacity: 0.48;
    pointer-events: none;
  }

  .full-width { width: 100%; }

  /* ── Sizes ────────────────────────────────────────────────────────────── */
  .btn-sm {
    height: 32px;
    padding: 0 var(--space-3);
    font-size: var(--text-xs);
    border-radius: var(--radius-md, 8px);
    min-width: 32px;
  }

  .btn-md {
    height: 40px;
    padding: 0 var(--space-4);
    font-size: var(--text-sm);
    border-radius: var(--radius-md, 8px);
    min-width: 44px;
  }

  .btn-lg {
    height: 48px;
    padding: 0 var(--space-6);
    font-size: var(--text-base);
    border-radius: var(--radius-lg, 10px);
    min-width: 48px;
  }

  .btn-xl {
    height: 56px;
    padding: 0 var(--space-8);
    font-size: var(--text-lg);
    border-radius: var(--radius-xl, 14px);
    min-width: 56px;
  }

  /* Touch minimum on mobile */
  @media (max-width: 767px) {
    .btn-sm  { height: 36px; }
    .btn-md  { height: 44px; }
  }

  /* ── Icon-only ────────────────────────────────────────────────────────── */
  .btn.icon-only {
    padding: 0;
  }
  .btn-sm.icon-only  { width: 32px; }
  .btn-md.icon-only  { width: 44px; }
  .btn-lg.icon-only  { width: 48px; }
  .btn-xl.icon-only  { width: 56px; }

  /* ── PRIMARY ──────────────────────────────────────────────────────────── */
  .btn-primary {
    background: linear-gradient(160deg,
      var(--primary-400) 0%,
      var(--primary-600) 55%,
      var(--primary-700) 100%
    );
    color: #fff;
    border: 1px solid rgba(255, 255, 255, 0.14);
    box-shadow:
      0 6px 20px rgba(20, 184, 166, 0.38),
      0 2px 6px  rgba(20, 184, 166, 0.22),
      inset 0 1px 0 rgba(255, 255, 255, 0.22),
      inset 0 -2px 6px rgba(0, 0, 0, 0.14);
  }

  .btn-primary:hover:not(:disabled) {
    box-shadow:
      0 10px 32px rgba(20, 184, 166, 0.52),
      0 4px 12px  rgba(20, 184, 166, 0.30),
      inset 0 1px 0 rgba(255, 255, 255, 0.26),
      inset 0 -2px 6px rgba(0, 0, 0, 0.12);
    filter: brightness(1.06);
  }

  /* ── SECONDARY ────────────────────────────────────────────────────────── */
  .btn-secondary {
    background: var(--surface-2, rgba(255,255,255,0.10));
    color: var(--text-primary);
    border: 1px solid var(--border-default);
    box-shadow:
      0 2px 8px rgba(0, 0, 0, 0.08),
      inset 0 1px 0 rgba(255, 255, 255, 0.10);
  }

  .btn-secondary:hover:not(:disabled) {
    background: var(--surface-hover, rgba(255,255,255,0.14));
    border-color: var(--border-strong);
    box-shadow:
      0 4px 16px rgba(0, 0, 0, 0.14),
      inset 0 1px 0 rgba(255, 255, 255, 0.14);
  }

  /* ── GHOST ────────────────────────────────────────────────────────────── */
  .btn-ghost {
    background: transparent;
    color: var(--text-secondary);
    border: 1px solid transparent;
  }

  .btn-ghost:hover:not(:disabled) {
    background: var(--surface-hover, rgba(255,255,255,0.07));
    color: var(--text-primary);
    border-color: var(--border-subtle);
  }

  /* ── DANGER ───────────────────────────────────────────────────────────── */
  .btn-danger {
    background: linear-gradient(135deg, var(--danger-500) 0%, var(--danger-600) 100%);
    color: #fff;
    border: 1px solid rgba(255, 255, 255, 0.10);
    box-shadow:
      0 6px 20px rgba(239, 68, 68, 0.32),
      inset 0 1px 0 rgba(255, 255, 255, 0.18),
      inset 0 -2px 6px rgba(0, 0, 0, 0.15);
  }

  .btn-danger:hover:not(:disabled) {
    box-shadow:
      0 8px 28px rgba(239, 68, 68, 0.46),
      inset 0 1px 0 rgba(255, 255, 255, 0.22),
      inset 0 -2px 6px rgba(0, 0, 0, 0.12);
    filter: brightness(1.06);
  }

  /* ── SUCCESS ──────────────────────────────────────────────────────────── */
  .btn-success {
    background: linear-gradient(135deg, var(--success-500) 0%, var(--success-600) 100%);
    color: #fff;
    border: 1px solid rgba(255, 255, 255, 0.12);
    box-shadow:
      0 6px 20px rgba(16, 185, 129, 0.36),
      inset 0 1px 0 rgba(255, 255, 255, 0.20),
      inset 0 -2px 6px rgba(0, 0, 0, 0.12);
  }

  .btn-success:hover:not(:disabled) {
    filter: brightness(1.08);
    box-shadow: 0 8px 28px rgba(16, 185, 129, 0.48);
  }

  /* ── Shimmer sweep (hover) ────────────────────────────────────────────── */
  .btn-shimmer {
    position: absolute;
    inset: 0;
    background: linear-gradient(
      105deg,
      transparent   15%,
      rgba(255,255,255,0.16) 50%,
      transparent   85%
    );
    transform: translateX(-100%);
    transition: transform 420ms var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1));
    pointer-events: none;
  }

  .btn:not(.btn-ghost):not(.btn-secondary):hover .btn-shimmer {
    transform: translateX(100%);
  }

  /* ── Ripple layer ─────────────────────────────────────────────────────── */
  .btn-ripple-layer {
    position: absolute;
    inset: 0;
    pointer-events: none;
    overflow: hidden;
    border-radius: inherit;
  }

  .btn-ripple {
    position: absolute;
    transform: translate(-50%, -50%) scale(0);
    width: 160px;
    height: 160px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.22);
    animation: btn-ripple-expand 580ms var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1)) forwards;
    pointer-events: none;
  }

  .btn-ghost .btn-ripple,
  .btn-secondary .btn-ripple {
    background: rgba(20, 184, 166, 0.18);
  }

  @keyframes btn-ripple-expand {
    to { transform: translate(-50%, -50%) scale(1); opacity: 0; }
  }

  /* ── Spinner ──────────────────────────────────────────────────────────── */
  .btn-spinner {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .btn-spinner svg {
    animation: btn-spin 0.9s linear infinite;
    will-change: transform;
  }

  @keyframes btn-spin {
    to { transform: rotate(360deg); }
  }

  /* ── Icon wrap ────────────────────────────────────────────────────────── */
  .btn-icon-wrap {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  /* ── Label ────────────────────────────────────────────────────────────── */
  .btn-label {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  /* ── Reduced motion ───────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .btn { transition: none; }
    .btn-shimmer { display: none; }
    .btn-ripple { animation: none; }
    .btn-spinner svg { animation: none; }
  }
</style>
