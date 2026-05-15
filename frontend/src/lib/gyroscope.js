/**
 * gyroscope.js — Device orientation → CSS custom property bridge
 *
 * Sets --gyro-x and --gyro-y on :root, enabling CSS parallax layers
 * via var(--gyro-layer-1), var(--gyro-layer-2), var(--gyro-layer-3).
 *
 * Only activates on mobile (pointer: coarse) and when the API is available.
 * Completely no-ops when prefers-reduced-motion is set.
 * GPU safe: only drives CSS custom properties used for transform.
 */

let started = false;
let rafId = null;
let rawX = 0;
let rawY = 0;
let smoothX = 0;
let smoothY = 0;
const LERP = 0.08;
const MAX_TILT = 12; // degrees of tilt → pixels of parallax

function lerp(a, b, t) { return a + (b - a) * t; }

function clamp(v, min, max) { return Math.max(min, Math.min(max, v)); }

function tick() {
  smoothX = lerp(smoothX, rawX, LERP);
  smoothY = lerp(smoothY, rawY, LERP);

  if (Math.abs(smoothX - rawX) > 0.01 || Math.abs(smoothY - rawY) > 0.01) {
    rafId = requestAnimationFrame(tick);
  } else {
    smoothX = rawX;
    smoothY = rawY;
    rafId = null;
  }

  document.documentElement.style.setProperty('--gyro-x', smoothX.toFixed(3));
  document.documentElement.style.setProperty('--gyro-y', smoothY.toFixed(3));
}

function handleOrientation(e) {
  // beta = front-to-back tilt (-180 to 180), gamma = left-right (-90 to 90)
  const beta  = e.beta  ?? 0;
  const gamma = e.gamma ?? 0;

  // Normalize to -1..1 range for CSS custom property
  rawX = clamp(gamma / MAX_TILT, -1, 1);
  rawY = clamp((beta - 45) / MAX_TILT, -1, 1); // 45° is natural hold angle

  if (!rafId) rafId = requestAnimationFrame(tick);
}

export function startGyroscope() {
  if (started) return;
  if (typeof window === 'undefined') return;

  // Only on touch devices
  const isTouchDevice = window.matchMedia('(pointer: coarse)').matches;
  if (!isTouchDevice) return;

  // Respect reduced motion
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

  if (!window.DeviceOrientationEvent) return;

  // iOS 13+ requires permission
  if (typeof DeviceOrientationEvent.requestPermission === 'function') {
    // Will be called on first user gesture in parent component
    return;
  }

  started = true;
  window.addEventListener('deviceorientation', handleOrientation, { passive: true });
}

export async function requestGyroscopePermission() {
  if (typeof window === 'undefined') return false;
  if (typeof DeviceOrientationEvent.requestPermission !== 'function') {
    startGyroscope();
    return true;
  }
  try {
    const permission = await DeviceOrientationEvent.requestPermission();
    if (permission === 'granted') {
      started = true;
      window.addEventListener('deviceorientation', handleOrientation, { passive: true });
      return true;
    }
  } catch (e) {
    // Permission denied or unavailable — silent fail
  }
  return false;
}

export function stopGyroscope() {
  if (!started) return;
  started = false;
  window.removeEventListener('deviceorientation', handleOrientation);
  if (rafId) {
    cancelAnimationFrame(rafId);
    rafId = null;
  }
  // Reset CSS properties
  document.documentElement.style.setProperty('--gyro-x', '0');
  document.documentElement.style.setProperty('--gyro-y', '0');
}
