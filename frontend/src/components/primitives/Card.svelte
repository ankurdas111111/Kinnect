<script>
  /**
   * Card — Interactive glass card with depth effects
   *
   * Props:
   *   variant   — 'default' | 'elevated' | 'glass' | 'outlined' | 'solid'
   *   hover     — enable hover elevation + lift (TECHNIQUE 14: layered shadow + translateY)
   *   tilt      — enable 3D tilt on mouse move
   *   glow      — color name for glow: 'primary' | 'success' | 'danger' | 'warning' | null
   *   padding   — 'none' | 'sm' | 'md' | 'lg'
   *   clickable — adds cursor:pointer and active press
   *   intensity — tilt max degrees (default 8)
   *   noise     — enable CSS feTurbulence noise texture overlay (TECHNIQUE 4, default true)
   */
  import { spring } from 'svelte/motion';
  import { createEventDispatcher } from 'svelte';

  export let variant   = 'default';
  export let hover     = true;
  export let tilt      = false;
  export let glow      = null;
  export let padding   = 'md';
  export let clickable = false;
  export let intensity = 8;
  // TECHNIQUE 4: noise texture — enabled by default on default/elevated/glass variants,
  // disabled on outlined (transparent bg) variants where it's not meaningful
  export let noise     = variant !== 'outlined';

  const dispatch = createEventDispatcher();

  // Spring for press depth on clickable
  const pressZ = spring(0, { stiffness: 500, damping: 30 });

  // Tilt state
  let el;
  let raf = null;
  let cx = 0, cy = 0, tx = 0, ty = 0;
  let hovering = false;
  let sx = 50, sy = 50; // shine position %

  function lerp(a, b, t) { return a + (b - a) * t; }

  function runTick() {
    cx = lerp(cx, tx, 0.12);
    cy = lerp(cy, ty, 0.12);
    if (el) {
      el.style.transform =
        `perspective(800px) rotateX(${cx}deg) rotateY(${cy}deg) translateZ(${$pressZ}px)`;
    }
    if (Math.abs(cx - tx) > 0.05 || Math.abs(cy - ty) > 0.05) {
      raf = requestAnimationFrame(runTick);
    } else {
      cx = tx; cy = ty;
      if (el) {
        el.style.transform =
          `perspective(800px) rotateX(${cx}deg) rotateY(${cy}deg) translateZ(${$pressZ}px)`;
      }
      raf = null;
    }
  }

  function startTick() { if (!raf) raf = requestAnimationFrame(runTick); }

  function applyPointer(clientX, clientY) {
    if (!tilt || !el) return;
    const r  = el.getBoundingClientRect();
    const dx = (clientX - r.left - r.width  / 2) / (r.width  / 2);
    const dy = (clientY - r.top  - r.height / 2) / (r.height / 2);
    tx = -dy * intensity;
    ty =  dx * intensity;
    sx = ((clientX - r.left) / r.width)  * 100;
    sy = ((clientY - r.top)  / r.height) * 100;
    startTick();
  }

  function onMouseMove(e)  { hovering = true;  applyPointer(e.clientX, e.clientY); }
  function onMouseLeave()  { hovering = false; tx = 0; ty = 0; startTick(); }
  function onTouchMove(e)  {
    const t = e.touches[0];
    if (!t) return;
    hovering = true;
    applyPointer(t.clientX, t.clientY);
  }
  function onTouchEnd()    { hovering = false; tx = 0; ty = 0; startTick(); }

  function onPointerDown() {
    if (!clickable) return;
    pressZ.set(-4);
  }
  function onPointerUp() {
    if (!clickable) return;
    pressZ.set(0);
  }
  function onClick(e) {
    if (clickable) dispatch('click', e);
  }
</script>

<!-- svelte-ignore a11y-no-static-element-interactions -->
<div
  class="card card-{variant} pad-{padding}"
  class:hover-lift={hover}
  class:clickable
  class:noise-surface={noise}
  class:glow-primary={glow === 'primary'}
  class:glow-success={glow === 'success'}
  class:glow-danger={glow === 'danger'}
  class:glow-warning={glow === 'warning'}
  class:tilt-active={tilt}
  bind:this={el}
  on:mousemove={onMouseMove}
  on:mouseleave={onMouseLeave}
  on:touchmove|passive={onTouchMove}
  on:touchend={onTouchEnd}
  on:pointerdown={onPointerDown}
  on:pointerup={onPointerUp}
  on:click={onClick}
  role={clickable ? 'button' : undefined}
  tabindex={clickable ? 0 : undefined}
>
  <!-- Shine layer (follows cursor during tilt) -->
  {#if tilt && hovering}
    <div
      class="card-shine"
      aria-hidden="true"
      style="background: radial-gradient(circle at {sx}% {sy}%, rgba(255,255,255,0.11) 0%, transparent 55%);"
    ></div>
  {/if}

  <!-- Top-edge glow line (always rendered, themed by glow prop) -->
  <div class="card-edge-line" aria-hidden="true"></div>

  <slot />
</div>

<style>
  /* ── Base ─────────────────────────────────────────────────────────────── */
  .card {
    position: relative;
    border-radius: var(--radius-xl, 14px);
    overflow: hidden;
    will-change: transform;
    transform-style: preserve-3d;
    transition:
      box-shadow 280ms var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1)),
      border-color 280ms var(--ease-out);
  }

  /* ── Padding ─────────────────────────────────────────────────────────── */
  .pad-none { padding: 0; }
  .pad-sm   { padding: var(--space-3); }
  .pad-md   { padding: var(--space-4) var(--space-5); }
  .pad-lg   { padding: var(--space-6) var(--space-8); }

  /* ── Variants ─────────────────────────────────────────────────────────── */
  .card-default {
    background: var(--surface-1, rgba(15, 15, 30, 0.72));
    border: 1px solid var(--border-default);
    box-shadow:
      0 4px 16px rgba(0, 0, 0, 0.18),
      0 1px 4px  rgba(0, 0, 0, 0.12),
      inset 0 1px 0 rgba(255, 255, 255, 0.07);
  }

  .card-elevated {
    background: var(--surface-2, rgba(18, 18, 36, 0.85));
    border: 1px solid var(--border-default);
    border-top-color: rgba(255, 255, 255, 0.10);
    box-shadow:
      0 8px 32px rgba(0, 0, 0, 0.28),
      0 2px 8px  rgba(0, 0, 0, 0.18),
      inset 0 1px 0 rgba(255, 255, 255, 0.09),
      inset 0 -1px 0 rgba(0, 0, 0, 0.15);
  }

  .card-glass {
    background: var(--glass-bg, rgba(15, 15, 30, 0.60));
    backdrop-filter: var(--glass-blur, blur(24px) saturate(1.8));
    -webkit-backdrop-filter: var(--glass-blur, blur(24px) saturate(1.8));
    border: 1px solid var(--glass-border, rgba(255, 255, 255, 0.10));
    border-top-color: var(--glass-border-strong, rgba(255, 255, 255, 0.18));
    box-shadow: var(--glass-shadow,
      0 8px 32px rgba(0, 0, 0, 0.28),
      0 0 0 1px rgba(255, 255, 255, 0.04)
    );
  }

  .card-outlined {
    background: transparent;
    border: 1px solid var(--border-default);
  }

  .card-outlined:hover {
    border-color: var(--border-strong);
    background: var(--surface-hover, rgba(255,255,255,0.04));
  }

  .card-solid {
    background: var(--surface-2, rgba(18, 18, 36, 0.92));
    border: 1px solid var(--border-strong);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.20);
  }

  /* ── Hover lift (TECHNIQUE 14: layered shadow depth + translateY) ─────── */
  .card.hover-lift {
    cursor: default;
    transition:
      box-shadow 280ms var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1)),
      transform   280ms var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1)),
      filter      280ms var(--ease-out),
      border-color 280ms;
  }

  .card-default.hover-lift:hover,
  .card-elevated.hover-lift:hover,
  .card-solid.hover-lift:hover {
    /* TECHNIQUE 14: translateY(-2px) + layered shadow system */
    transform: translateY(-2px);
    box-shadow:
      0 4px 8px  rgba(0, 0, 0, 0.10),
      0 8px 24px rgba(0, 0, 0, 0.18),
      0 16px 48px rgba(0, 0, 0, 0.14),
      0 0 0 1px rgba(255, 255, 255, 0.05),
      inset 0 1px 0 rgba(255, 255, 255, 0.10);
    filter: brightness(1.04);
  }

  .card-glass.hover-lift:hover {
    /* TECHNIQUE 14: glass variant — stronger depth + Liquid Glass 2.0 */
    transform: translateY(-2px);
    box-shadow:
      0 4px 8px  rgba(0, 0, 0, 0.14),
      0 8px 24px rgba(0, 0, 0, 0.22),
      0 20px 56px rgba(0, 0, 0, 0.28),
      0 0 0 1px rgba(255, 255, 255, 0.06);
    filter: brightness(1.06);
    border-color: var(--glass-border-strong, rgba(255, 255, 255, 0.22));
    backdrop-filter: blur(32px) saturate(180%) brightness(1.08);
    -webkit-backdrop-filter: blur(32px) saturate(180%) brightness(1.08);
  }

  /* ── Clickable ────────────────────────────────────────────────────────── */
  .card.clickable {
    cursor: pointer;
  }

  .card.clickable:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 3px;
  }

  /* ── Tilt ─────────────────────────────────────────────────────────────── */
  .card.tilt-active {
    will-change: transform;
  }

  /* ── Glow variants ────────────────────────────────────────────────────── */
  .card.glow-primary { box-shadow: var(--glow-primary, 0 0 24px rgba(20,184,166,0.38)), 0 4px 16px rgba(0,0,0,0.20); }
  .card.glow-success { box-shadow: 0 0 24px rgba(16,185,129,0.38), 0 4px 16px rgba(0,0,0,0.20); }
  .card.glow-danger  { box-shadow: var(--glow-sos, 0 0 24px rgba(239,68,68,0.42)), 0 4px 16px rgba(0,0,0,0.20); }
  .card.glow-warning { box-shadow: 0 0 24px rgba(245,158,11,0.38), 0 4px 16px rgba(0,0,0,0.20); }

  /* ── Shine layer ──────────────────────────────────────────────────────── */
  .card-shine {
    position: absolute;
    inset: 0;
    pointer-events: none;
    z-index: 1;
    border-radius: inherit;
    mix-blend-mode: overlay;
    opacity: 0.9;
  }

  /* ── Edge accent line ─────────────────────────────────────────────────── */
  .card-edge-line {
    position: absolute;
    top: 0;
    left: 12%;
    right: 12%;
    height: 1px;
    background: linear-gradient(
      90deg,
      transparent 0%,
      rgba(20, 184, 166, 0.55) 30%,
      rgba(20, 184, 166, 0.55) 70%,
      transparent 100%
    );
    box-shadow: 0 0 6px rgba(20, 184, 166, 0.30);
    pointer-events: none;
    border-radius: 0 0 2px 2px;
  }

  .card.glow-danger  .card-edge-line { background: linear-gradient(90deg, transparent, rgba(239,68,68,0.55), transparent); }
  .card.glow-success .card-edge-line { background: linear-gradient(90deg, transparent, rgba(16,185,129,0.55), transparent); }
  .card.glow-warning .card-edge-line { background: linear-gradient(90deg, transparent, rgba(245,158,11,0.55), transparent); }

  /* ── Reduced motion ───────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .card, .card.hover-lift, .card.tilt-active {
      transition: none;
      transform: none !important;
    }
    .card-shine { display: none; }
  }
</style>
