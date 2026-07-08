<script>
  /**
   * GyroscopeLayer — Mobile gyroscope parallax depth effect.
   *
   * Uses DeviceOrientation API to create a sense of 3D depth:
   * children move subtly as the user tilts their device.
   * Multiple depth layers can be nested with different --gyro-depth values.
   *
   * GPU only: animates transform only (translate3d).
   * Full cleanup on destroy.
   *
   * Props:
   *   maxShift — max displacement in px (default 8)
   *   depth    — multiplier for this layer's depth response (default 1.0)
   *   disabled — skip effect (default false, also auto-disabled if no API)
   */
  import { onMount, onDestroy } from 'svelte';

  /**
   * @typedef {Object} Props
   * @property {number} [maxShift]
   * @property {number} [depth]
   * @property {boolean} [disabled]
   * @property {import('svelte').Snippet} [children]
   */

  /** @type {Props} */
  let {
    maxShift = 8,
    depth = 1.0,
    disabled = false,
    children
  } = $props();

  let el = $state();
  let supported = false;
  let tx = 0, ty = 0;
  let cx = 0, cy = 0;
  let raf = null;
  let hasPermission = false;

  function lerp(a, b, t) { return a + (b - a) * t; }

  function tick() {
    cx = lerp(cx, tx, 0.08);
    cy = lerp(cy, ty, 0.08);
    if (el) {
      el.style.transform = `translate3d(${cx.toFixed(2)}px, ${cy.toFixed(2)}px, 0)`;
    }
    const stillMoving = Math.abs(cx - tx) > 0.02 || Math.abs(cy - ty) > 0.02;
    raf = stillMoving ? requestAnimationFrame(tick) : null;
  }

  function startTick() {
    if (!raf) raf = requestAnimationFrame(tick);
  }

  function onOrientation(e) {
    if (disabled) return;
    // gamma = left/right tilt (-90..90), beta = front/back tilt (-180..180)
    const gamma = Math.max(-45, Math.min(45, e.gamma || 0));
    const beta  = Math.max(-45, Math.min(45, (e.beta  || 0) - 30)); // subtract natural phone angle
    tx = (gamma / 45) * maxShift * depth;
    ty = (beta  / 45) * maxShift * depth;
    startTick();
  }

  async function requestPermission() {
    // iOS 13+ requires permission request
    if (typeof DeviceOrientationEvent !== 'undefined' &&
        typeof DeviceOrientationEvent.requestPermission === 'function') {
      try {
        const perm = await DeviceOrientationEvent.requestPermission();
        return perm === 'granted';
      } catch {
        return false;
      }
    }
    return true; // Android / other — no permission needed
  }

  onMount(async () => {
    if (disabled) return;
    if (typeof window === 'undefined' || !('DeviceOrientationEvent' in window)) return;
    // Check reduced motion preference — skip gyroscope if reduced motion
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

    supported = true;
    hasPermission = await requestPermission();
    if (hasPermission) {
      window.addEventListener('deviceorientation', onOrientation, { passive: true });
    }
  });

  onDestroy(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('deviceorientation', onOrientation);
    }
    if (raf) { cancelAnimationFrame(raf); raf = null; }
  });
</script>

<div
  class="gyro-layer"
  bind:this={el}
  aria-hidden="true"
>
  {@render children?.()}
</div>

<style>
  .gyro-layer {
    will-change: transform;
    /* Smooth start — CSS transition for initial load, JS takes over after */
  }

  @media (prefers-reduced-motion: reduce) {
    .gyro-layer {
      transform: none !important;
    }
  }
</style>
