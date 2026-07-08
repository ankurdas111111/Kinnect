<script>
  import { passive } from 'svelte/legacy';

  /**
   * MagneticButton — 2026 magnetic attraction effect.
   *
   * The button content subtly follows the cursor/touch position,
   * creating a gravitational pull feel for primary CTAs.
   *
   * Uses rAF lerp — GPU compositor only (transform).
   * Cleans up all listeners on destroy.
   *
   * Props:
   *   strength  — max displacement in px (default 6)
   *   disabled  — opt out of magnetic effect (default false)
   *   className — extra class on wrapper
   *
   * Bug fix: inner element was using `display: contents` which creates no layout
   * box, so `will-change: transform` and `transform` style had zero effect.
   * The magnetic movement was completely invisible. Fixed: inner now uses
   * `display: inline-flex` with `width: 100%` so it forms a real box that
   * can receive the transform while still filling the wrapper.
   */
  import { onDestroy } from 'svelte';

  /**
   * @typedef {Object} Props
   * @property {number} [strength]
   * @property {boolean} [disabled]
   * @property {string} [className]
   * @property {import('svelte').Snippet} [children]
   */

  /** @type {Props} */
  let {
    strength = 6,
    disabled = false,
    className = '',
    children
  } = $props();

  let el = $state();
  let inner = $state();
  let tx = 0, ty = 0;
  let cx = 0, cy = 0;
  let raf = null;

  function lerp(a, b, t) { return a + (b - a) * t; }

  function tick() {
    cx = lerp(cx, tx, 0.12);
    cy = lerp(cy, ty, 0.12);
    if (inner) {
      inner.style.transform = `translate(${cx.toFixed(2)}px, ${cy.toFixed(2)}px)`;
    }
    const stillMoving = Math.abs(cx - tx) > 0.04 || Math.abs(cy - ty) > 0.04;
    if (stillMoving) {
      raf = requestAnimationFrame(tick);
    } else {
      cx = tx; cy = ty;
      if (inner) inner.style.transform = `translate(${cx.toFixed(2)}px, ${cy.toFixed(2)}px)`;
      raf = null;
    }
  }

  function startTick() {
    if (!raf) raf = requestAnimationFrame(tick);
  }

  function onMouseMove(e) {
    if (disabled || !el) return;
    const r = el.getBoundingClientRect();
    const dx = (e.clientX - r.left - r.width / 2) / (r.width / 2);
    const dy = (e.clientY - r.top - r.height / 2) / (r.height / 2);
    tx = dx * strength;
    ty = dy * strength;
    startTick();
  }

  function onMouseLeave() {
    tx = 0; ty = 0;
    startTick();
  }

  function onTouchMove(e) {
    if (disabled || !el || !e.touches[0]) return;
    const r = el.getBoundingClientRect();
    const touch = e.touches[0];
    const dx = (touch.clientX - r.left - r.width / 2) / (r.width / 2);
    const dy = (touch.clientY - r.top - r.height / 2) / (r.height / 2);
    tx = dx * strength;
    ty = dy * strength;
    startTick();
  }

  function onTouchEnd() {
    tx = 0; ty = 0;
    startTick();
  }

  onDestroy(() => {
    if (raf) { cancelAnimationFrame(raf); raf = null; }
  });
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="mag-root {className}"
  bind:this={el}
  onmousemove={onMouseMove}
  onmouseleave={onMouseLeave}
  use:passive={['touchmove', () => onTouchMove]}
  ontouchend={onTouchEnd}
>
  <!-- Bug fix: was display:contents — a `contents` element creates no box,
       so transform and will-change had zero effect. Now inline-flex fills
       the wrapper and properly receives the magnetic transform. -->
  <div class="mag-inner" bind:this={inner}>
    {@render children?.()}
  </div>
</div>

<style>
  .mag-root {
    display: inline-flex;
    position: relative;
  }

  .mag-inner {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    will-change: transform;
    /* transform applied by JS rAF loop */
  }

  @media (prefers-reduced-motion: reduce) {
    .mag-inner {
      will-change: auto;
      transform: none !important;
    }
  }
</style>
