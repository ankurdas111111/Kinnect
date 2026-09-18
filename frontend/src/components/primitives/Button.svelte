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

  /**
   * @typedef {Object} Props
   * @property {string} [variant]
   * @property {string} [size]
   * @property {boolean} [loading]
   * @property {boolean} [disabled]
   * @property {boolean} [fullWidth]
   * @property {string} [type]
   * @property {any} [href] - renders as <a> when set
   * @property {import('svelte').Snippet} [icon]
   * @property {import('svelte').Snippet} [children]
   */

  /** @type {Props} */
  let {
    variant = 'primary',
    size = 'md',
    loading = false,
    disabled = false,
    fullWidth = false,
    type = 'button',
    href = null,
    icon,
    children
  } = $props();

  const dispatch = createEventDispatcher();

  // Spring for press scale
  const scale = spring(1, { stiffness: 600, damping: 28 });

  // Ripple state
  let ripples = $state([]);
  let btnEl = $state();

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

  let tag = $derived(href ? 'a' : 'button');
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<svelte:element
  this={tag}
  {href}
  {type}
  class="btn btn-{variant} btn-{size}"
  class:full-width={fullWidth}
  class:loading
  class:icon-only={icon && !children}
  disabled={disabled || loading || undefined}
  aria-disabled={disabled || loading || undefined}
  aria-busy={loading || undefined}
  bind:this={btnEl}
  style="transform: scale({$scale})"
  onpointerdown={onPointerDown}
  onpointerup={onPointerUp}
  onpointerleave={onPointerLeave}
  onclick={handleClick}
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
    {#if icon}
      <span class="btn-icon-wrap">{@render icon?.()}</span>
    {/if}
    {#if children}
      <span class="btn-label">{@render children?.()}</span>
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

  /* ── PRIMARY — flat ember fill, the one accent ────────────────────────── */
  .btn-primary {
    background: var(--primary-500);
    color: var(--text-on-primary);
    border: 1px solid transparent;
    box-shadow: var(--shadow-sm);
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--primary-600);
    box-shadow: var(--shadow-md);
  }

  /* ── SECONDARY — paper with a soft ink outline ────────────────────────── */
  .btn-secondary {
    background: var(--surface-1);
    color: var(--text-primary);
    border: 1px solid var(--border-default);
    box-shadow: var(--shadow-xs);
  }

  .btn-secondary:hover:not(:disabled) {
    background: var(--surface-hover);
    border-color: var(--border-strong);
  }

  /* ── GHOST ────────────────────────────────────────────────────────────── */
  .btn-ghost {
    background: transparent;
    color: var(--text-secondary);
    border: 1px solid transparent;
  }

  .btn-ghost:hover:not(:disabled) {
    background: var(--surface-hover);
    color: var(--text-primary);
    border-color: var(--border-subtle);
  }

  /* ── DANGER — destructive is ink + outline, never vermilion.
     Vermilion belongs to SOS surfaces alone; the label carries the meaning. */
  .btn-danger {
    background: transparent;
    color: var(--text-primary);
    border: 1.5px solid var(--border-strong);
  }

  .btn-danger:hover:not(:disabled) {
    background: var(--surface-hover);
    border-color: var(--ink-3);
  }

  /* ── SUCCESS — flat sage ──────────────────────────────────────────────── */
  .btn-success {
    background: var(--success-500);
    color: var(--text-inverse);
    border: 1px solid transparent;
    box-shadow: var(--shadow-sm);
  }

  .btn-success:hover:not(:disabled) {
    background: var(--success-600);
  }

  /* ── Shimmer layer retired on paper — kept as an inert element so the
     markup contract stays identical. ─────────────────────────────────── */
  .btn-shimmer {
    display: none;
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
    background: oklch(0.99 0.004 80 / 0.22);
    animation: btn-ripple-expand 580ms var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1)) forwards;
    pointer-events: none;
  }

  .btn-ghost .btn-ripple,
  .btn-secondary .btn-ripple {
    background: color-mix(in oklch, var(--primary-500) 18%, transparent);
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
