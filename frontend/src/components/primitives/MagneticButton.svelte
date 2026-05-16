<script>
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
   */
  import { onDestroy } from 'svelte';

  export let strength = 6;
  export let disabled = false;
  export let className = '';

  let el;
  let inner;
  let tx = 0, ty = 0;
  let cx = 0, cy = 0;
  let raf = null;
  let inside = false;

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
    inside = false;
    tx = 0; ty = 0;
    startTick();
  }

  function onMouseEnter() {
    inside = true;
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

<!-- svelte-ignore a11y-no-static-element-interactions -->
<div
  class="mag-root {className}"
  bind:this={el}
  on:mousemove={onMouseMove}
  on:mouseleave={onMouseLeave}
  on:mouseenter={onMouseEnter}
  on:touchmove|passive={onTouchMove}
  on:touchend={onTouchEnd}
>
  <div class="mag-inner" bind:this={inner}>
    <slot />
  </div>
</div>

<style>
  .mag-root {
    display: inline-flex;
    position: relative;
  }

  .mag-inner {
    will-change: transform;
    display: contents;
    /* transform applied by JS */
  }
</style>
