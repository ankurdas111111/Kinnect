<script>
  import { passive } from 'svelte/legacy';

  
  /**
   * @typedef {Object} Props
   * @property {number} [intensity] - TiltCard — GPU-composited 3D perspective tilt.
Uses rAF lerp (no JS frameworks) → stays off main thread as much as possible.
Exports:
intensity  — max tilt degrees (default 12)
shine      — show specular highlight following cursor (default true)
disabled   — opt-out on low-power devices or when not needed
   * @property {boolean} [shine]
   * @property {boolean} [disabled]
   * @property {import('svelte').Snippet} [children]
   */

  /** @type {Props} */
  let {
    intensity = 12,
    shine = true,
    disabled = false,
    children
  } = $props();

  let el = $state();
  // Current rendered tilt (lerped toward target)
  let cx = 0, cy = 0;
  // Target tilt (set immediately on pointer move)
  let tx = 0, ty = 0;
  // Shine position (%)
  let sx = $state(50), sy = $state(50);
  let hovering = $state(false);
  let raf = null;

  function lerp(a, b, t) { return a + (b - a) * t; }

  function tick() {
    cx = lerp(cx, tx, 0.14);
    cy = lerp(cy, ty, 0.14);
    if (el) {
      el.style.transform = `perspective(900px) rotateX(${cx}deg) rotateY(${cy}deg)`;
    }
    if (Math.abs(cx - tx) > 0.04 || Math.abs(cy - ty) > 0.04) {
      raf = requestAnimationFrame(tick);
    } else {
      // Snap to final
      cx = tx; cy = ty;
      if (el) el.style.transform = `perspective(900px) rotateX(${cx}deg) rotateY(${cy}deg)`;
      raf = null;
    }
  }

  function startTick() {
    if (!raf) raf = requestAnimationFrame(tick);
  }

  function applyPointer(clientX, clientY) {
    if (disabled || !el) return;
    const r = el.getBoundingClientRect();
    const dx = (clientX - r.left - r.width  / 2) / (r.width  / 2); // −1…1
    const dy = (clientY - r.top  - r.height / 2) / (r.height / 2); // −1…1
    tx = -dy * intensity;
    ty =  dx * intensity;
    sx = ((clientX - r.left) / r.width)  * 100;
    sy = ((clientY - r.top)  / r.height) * 100;
    startTick();
  }

  function handleMouseMove(e) {
    hovering = true;
    applyPointer(e.clientX, e.clientY);
  }

  function handleMouseLeave() {
    hovering = false;
    tx = 0; ty = 0;
    startTick();
  }

  function handleTouchMove(e) {
    const t = e.touches[0];
    if (!t) return;
    hovering = true;
    applyPointer(t.clientX, t.clientY);
  }

  function handleTouchEnd() {
    hovering = false;
    tx = 0; ty = 0;
    startTick();
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="tilt-root"
  class:disabled
  bind:this={el}
  onmousemove={handleMouseMove}
  onmouseleave={handleMouseLeave}
  use:passive={['touchmove', () => handleTouchMove]}
  ontouchend={handleTouchEnd}
>
  {#if shine && hovering && !disabled}
    <div
      class="tilt-shine"
      aria-hidden="true"
      style="background: radial-gradient(circle at {sx}% {sy}%, rgba(255,255,255,0.13) 0%, transparent 58%);"
    ></div>
  {/if}
  {@render children?.()}
</div>

<style>
  .tilt-root {
    will-change: transform;
    transform-style: preserve-3d;
    position: relative;
    /* 3D depth shadow that reacts to tilt */
    transition: box-shadow 0.3s ease, filter 0.3s ease;
  }
  .tilt-root:hover {
    /* Elevate on hover for depth feel */
    box-shadow:
      0 16px 48px rgba(0, 0, 0, 0.14),
      0 6px 16px rgba(0, 0, 0, 0.08);
  }
  .tilt-root.disabled {
    transform: none !important;
  }
  .tilt-shine {
    position: absolute;
    inset: 0;
    pointer-events: none;
    z-index: 9;
    border-radius: inherit;
    /* Enhanced specular — brighter, more visible reflection */
    mix-blend-mode: overlay;
    opacity: 0.8;
  }
</style>
