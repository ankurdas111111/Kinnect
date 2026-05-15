<script>
  /**
   * ParticleField — Lightweight CSS-only ambient particle background.
   *
   * Uses CSS custom properties + animations — zero canvas, zero JS per-frame.
   * GPU-composited: only animates transform + opacity.
   * Respects prefers-reduced-motion (disables via CSS).
   *
   * Props:
   *   count      — number of particles (default 20, mobile-safe)
   *   color      — particle color (default brand teal)
   *   maxSize    — max particle size in px (default 3)
   *   className  — additional CSS class on container
   */
  export let count = 20;
  export let color = 'rgba(20, 184, 166, 0.35)';
  export let maxSize = 3;
  export let className = '';

  // Generate particles with deterministic pseudo-random positions
  // using a simple LCG — no Math.random() so SSR is safe
  function lcg(seed) {
    return ((seed * 1664525 + 1013904223) >>> 0) / 4294967296;
  }

  const particles = Array.from({ length: count }, (_, i) => {
    const s1 = (i * 2654435761 + 1) >>> 0;
    const s2 = (i * 1664525 + 13) >>> 0;
    const s3 = (i * 22695477 + 7) >>> 0;
    const s4 = (i * 214013 + 2531011) >>> 0;
    const s5 = (i * 6364136223846793005n ? (i * 134775813 + 1) >>> 0 : (i * 3) >>> 0);

    return {
      x: lcg(s1) * 100,          // % left
      y: lcg(s2) * 100,          // % top
      size: 1 + lcg(s3) * (maxSize - 1),  // px
      duration: 4 + lcg(s4) * 8,          // seconds
      delay: lcg(s1 ^ s2) * -8,           // negative = already started
      opacity: 0.15 + lcg(s3 ^ s4) * 0.30,
    };
  });
</script>

<div class="pf-container {className}" aria-hidden="true">
  {#each particles as p, i}
    <span
      class="pf-dot"
      style="
        left: {p.x.toFixed(1)}%;
        top: {p.y.toFixed(1)}%;
        width: {p.size.toFixed(1)}px;
        height: {p.size.toFixed(1)}px;
        opacity: {p.opacity.toFixed(2)};
        animation-duration: {p.duration.toFixed(1)}s;
        animation-delay: {p.delay.toFixed(1)}s;
        background: {color};
        box-shadow: 0 0 {(p.size * 3).toFixed(0)}px {color};
      "
    ></span>
  {/each}
</div>

<style>
  .pf-container {
    position: absolute;
    inset: 0;
    pointer-events: none;
    overflow: hidden;
    z-index: 0;
  }

  .pf-dot {
    position: absolute;
    border-radius: 50%;
    animation: particle-float linear infinite;
    will-change: transform, opacity;
  }

  /* particle-float defined in global.css */

  @media (prefers-reduced-motion: reduce) {
    .pf-dot {
      animation: none;
    }
  }
</style>
