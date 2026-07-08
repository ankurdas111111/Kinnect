<script>
  import { run } from 'svelte/legacy';

  
  /**
   * @typedef {Object} Props
   * @property {boolean} [active] - SosParticleBurst — CSS-only particle burst for SOS activation.
12 particles radiate outward from the center on trigger.
Zero JS per frame — pure CSS animation with custom properties.
GPU only: animates transform + opacity.
Usage:
<SosParticleBurst active={sosActive} />
Props:
active — triggers the burst animation when true
color  — particle color (default danger red)
   * @property {string} [color]
   */

  /** @type {Props} */
  let { active = false, color = 'rgba(239, 68, 68, 0.9)' } = $props();

  // 12 particles, evenly distributed around the circle
  const PARTICLE_COUNT = 12;
  const particles = Array.from({ length: PARTICLE_COUNT }, (_, i) => ({
    angle: (i / PARTICLE_COUNT) * 360,
    distance: 40 + (i % 3) * 20,   // 40, 60, or 80px
    delay: i % 4 * 30,              // 0, 30, 60, 90ms stagger
    size: i % 2 === 0 ? 6 : 4,     // alternating large/small
    duration: 500 + (i % 3) * 100, // 500–700ms
  }));

  // Key to re-trigger burst on each SOS activation
  let burstKey = $state(0);
  run(() => {
    if (active) burstKey++;
  });
</script>

{#if active}
  {#key burstKey}
    <div class="spb-root" aria-hidden="true">
      {#each particles as p, i}
        <div
          class="spb-particle"
          style="
            --spb-angle: {p.angle}deg;
            --spb-distance: {p.distance}px;
            --spb-size: {p.size}px;
            --spb-delay: {p.delay}ms;
            --spb-duration: {p.duration}ms;
            --spb-color: {color};
          "
        ></div>
      {/each}
      <!-- Central flash ring -->
      <div class="spb-flash-ring"></div>
      <div class="spb-flash-ring spb-flash-ring-2"></div>
    </div>
  {/key}
{/if}

<style>
  .spb-root {
    position: absolute;
    inset: 0;
    pointer-events: none;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: visible;
    z-index: 1;
  }

  .spb-particle {
    position: absolute;
    width: var(--spb-size);
    height: var(--spb-size);
    border-radius: 50%;
    background: var(--spb-color, rgba(239, 68, 68, 0.9));
    top: 50%;
    left: 50%;
    margin-top: calc(var(--spb-size) / -2);
    margin-left: calc(var(--spb-size) / -2);
    /* Use CSS custom properties to drive the radial scatter */
    animation: spb-scatter var(--spb-duration, 600ms) cubic-bezier(0, 0.9, 0.57, 1) var(--spb-delay, 0ms) both;
    will-change: transform, opacity;
  }

  @keyframes spb-scatter {
    0% {
      transform:
        rotate(var(--spb-angle, 0deg))
        translateX(0)
        scale(1);
      opacity: 1;
    }
    60% {
      opacity: 0.9;
    }
    100% {
      transform:
        rotate(var(--spb-angle, 0deg))
        translateX(var(--spb-distance, 50px))
        scale(0.2);
      opacity: 0;
    }
  }

  /* Concentric flash rings */
  .spb-flash-ring {
    position: absolute;
    inset: 0;
    margin: auto;
    width: 100%;
    height: 100%;
    border-radius: 50%;
    border: 2px solid var(--spb-color, rgba(239, 68, 68, 0.8));
    animation: spb-ring-expand 600ms cubic-bezier(0, 0.9, 0.57, 1) both;
    will-change: transform, opacity;
  }

  .spb-flash-ring-2 {
    animation-delay: 80ms;
    border-color: rgba(239, 68, 68, 0.5);
    border-width: 1px;
  }

  @keyframes spb-ring-expand {
    0%   { transform: scale(0.8); opacity: 1; }
    100% { transform: scale(2.5); opacity: 0; }
  }

  @media (prefers-reduced-motion: reduce) {
    .spb-particle,
    .spb-flash-ring,
    .spb-flash-ring-2 {
      animation: none;
      opacity: 0;
    }
  }
</style>
