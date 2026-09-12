<script>
  import { run } from 'svelte/legacy';

  /**
   * SosParticleBurst — visual confirmation burst when SOS is triggered.
   * Renders a full-screen particle explosion in danger-red to signal
   * that the SOS event fired. Auto-dismisses via 'done' event.
   *
   * Props:
   *   active — true while the burst should be visible
   *
   * Events:
   *   done — fired after animation completes so the parent can reset active
   */
  import { fade } from 'svelte/transition';
  import { createEventDispatcher, onDestroy } from 'svelte';

  /**
   * @typedef {Object} Props
   * @property {boolean} [active]
   */

  /** @type {Props} */
  let { active = false } = $props();

  const dispatch = createEventDispatcher();
  let _timer = $state();

  run(() => {
    clearTimeout(_timer);
    if (active) _timer = setTimeout(() => dispatch('done'), 900);
  });

  onDestroy(() => clearTimeout(_timer));
</script>

{#if active}
  <div
    class="spb-overlay"
    aria-hidden="true"
    transition:fade={{ duration: 250 }}
  >
    <div class="spb-center">
      <!-- Expanding shock rings -->
      <div class="spb-ring spb-ring-1"></div>
      <div class="spb-ring spb-ring-2"></div>
      <!-- Central flash -->
      <div class="spb-flash"></div>
      <!-- 8 particles radiate outward at 45° increments -->
      <span class="spb-p spb-p-1"></span>
      <span class="spb-p spb-p-2"></span>
      <span class="spb-p spb-p-3"></span>
      <span class="spb-p spb-p-4"></span>
      <span class="spb-p spb-p-5"></span>
      <span class="spb-p spb-p-6"></span>
      <span class="spb-p spb-p-7"></span>
      <span class="spb-p spb-p-8"></span>
    </div>
  </div>
{/if}

<style>
  .spb-overlay {
    position: fixed;
    inset: 0;
    z-index: var(--z-overlay, 3000);
    pointer-events: none;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  /* Zero-size anchor so all children position relative to screen centre */
  .spb-center {
    position: relative;
    width: 0;
    height: 0;
  }

  /* ── Central flash ──────────────────────────────────────────────────── */
  .spb-flash {
    position: absolute;
    width: 80px; height: 80px;
    border-radius: 50%;
    background: color-mix(in oklch, var(--danger-500) 90%, transparent);
    transform: translate(-50%, -50%);
    box-shadow:
      0 0 40px color-mix(in oklch, var(--danger-500) 70%, transparent),
      0 0 80px color-mix(in oklch, var(--danger-500) 40%, transparent);
    animation: spb-flash 650ms cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }

  /* ── Shock rings ────────────────────────────────────────────────────── */
  .spb-ring {
    position: absolute;
    border-radius: 50%;
    border: 2px solid color-mix(in oklch, var(--danger-500) 70%, transparent);
    transform: translate(-50%, -50%);
    animation: spb-ring-expand 800ms cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }

  .spb-ring-1 {
    width: 60px; height: 60px;
    animation-delay: 0ms;
  }

  .spb-ring-2 {
    width: 60px; height: 60px;
    border-color: color-mix(in oklch, var(--danger-300) 50%, transparent);
    animation-delay: 120ms;
  }

  /* ── Radial particles ───────────────────────────────────────────────── */
  .spb-p {
    position: absolute;
    width: 7px; height: 7px;
    border-radius: 50%;
    background: color-mix(in oklch, var(--danger-500) 90%, transparent);
    box-shadow: 0 0 8px color-mix(in oklch, var(--danger-500) 60%, transparent);
    transform: translate(-50%, -50%);
    animation: spb-particle 750ms cubic-bezier(0.16, 1, 0.3, 1) forwards;
    will-change: transform, opacity;
  }

  /* Each particle gets a unique direction via --spb-angle */
  .spb-p-1 { --spb-angle: 0deg;    animation-delay:  20ms; }
  .spb-p-2 { --spb-angle: 45deg;   animation-delay:  40ms; }
  .spb-p-3 { --spb-angle: 90deg;   animation-delay:  60ms; }
  .spb-p-4 { --spb-angle: 135deg;  animation-delay:  80ms; }
  .spb-p-5 { --spb-angle: 180deg;  animation-delay: 100ms; }
  .spb-p-6 { --spb-angle: 225deg;  animation-delay: 120ms; }
  .spb-p-7 { --spb-angle: 270deg;  animation-delay: 140ms; }
  .spb-p-8 { --spb-angle: 315deg;  animation-delay: 160ms; }

  /* ── Keyframes ──────────────────────────────────────────────────────── */
  @keyframes spb-flash {
    0%   { transform: translate(-50%, -50%) scale(0.05); opacity: 1; }
    35%  { transform: translate(-50%, -50%) scale(1.15); opacity: 0.90; }
    100% { transform: translate(-50%, -50%) scale(3.0);  opacity: 0; }
  }

  @keyframes spb-ring-expand {
    0%   { transform: translate(-50%, -50%) scale(0.4);  opacity: 1; }
    100% { transform: translate(-50%, -50%) scale(7);    opacity: 0; }
  }

  @keyframes spb-particle {
    0% {
      transform: translate(-50%, -50%) rotate(var(--spb-angle)) translateX(0px);
      opacity: 1;
    }
    100% {
      transform: translate(-50%, -50%) rotate(var(--spb-angle)) translateX(130px);
      opacity: 0;
    }
  }

  /* Suppress all animation for motion-sensitive users */
  @media (prefers-reduced-motion: reduce) {
    .spb-overlay { display: none; }
  }
</style>
