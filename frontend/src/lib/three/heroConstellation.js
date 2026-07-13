/**
 * heroConstellation.js — "The Kin Constellation in true depth."
 *
 * The ONE surface in the 14-page rebrand that earns a JS 3D library: the
 * Landing hero's volumetric particle field. A WebGLRenderer draws ~2,500
 * GPU point sprites in perspective depth-fog — stars + 4 additive-glow family
 * beacons — with a slow parallax dolly driven passively by scroll progress and
 * a ±3° cursor tilt. This is depth, never content: it fades in OVER the static
 * SVG hero poster (the LCP element), so a failed import / lost context / gate
 * flip always leaves a finished hero with zero layout shift.
 *
 * WebGL-first (NOT three/webgpu): adversarial review pinned the WebGLRenderer +
 * hand-written GLSL path (~140KB gz) over the node-material WebGPU build to stay
 * under the 220KB gz cap. WebGPU is a later upgrade behind the same 'three'
 * chunk if the cap ever grows.
 *
 * NEVER runs on native: this module is aliased to capacitor-stub.js at build
 * time for VITE_TARGET=capacitor (see vite.config.js), so 0 three bytes ship in
 * the APK/IPA. The stub's default export is {}, so mount is undefined there —
 * the Landing gate also guards isNativePlatform() at runtime.
 *
 * Public API:
 *   mount(canvas, opts) -> { dispose }
 *     opts.getScroll()  -> 0..1 hero scroll progress (read passively per-frame;
 *                          no scroll listener here — no thrash)
 *     opts.onMounted()  -> optional; called once the first frame renders
 *   dispose() releases the GL context, geometry, material and all rAF/listeners.
 *
 * Palette twins (raw-color-ok, canvas-safe): the member wheel + brand + a
 * moonstone star tint, kept in sync with tokens-oklch.css. var() cannot resolve
 * in a GLSL attribute, so hex twins are the only option here — same rule the
 * canvas globe already follows in lib/getUserColor.js.
 */

import * as THREE from 'three';

// ── Palette (canvas-safe hex twins — keep in sync with tokens-oklch.css) ─────
// Member wheel: harbor / cobalt / orchid / plum (member-*-halo, the luminous
// variant — beacons glow). Brand = self beacon. Moonstone = faint star tint.
const BEACON_HEX = [
  0x6b7cdf, // raw-color-ok — brand primary-400 (self beacon) — sync tokens-oklch.css --primary-400
  0x2cc3d2, // raw-color-ok — member-1-halo (harbor)  — sync tokens-oklch.css
  0x56baef, // raw-color-ok — member-2-halo (cobalt)  — sync tokens-oklch.css
  0xba9cef, // raw-color-ok — member-3-halo (orchid)  — sync tokens-oklch.css
  0xd094dd, // raw-color-ok — member-4-halo (plum)    — sync tokens-oklch.css
];
const STAR_HEX = 0x8ea0ea;  // raw-color-ok — primary-300 moonstone star tint — sync tokens-oklch.css
const FOG_HEX  = 0x0a0a1a;  // raw-color-ok — night-canvas fog (indigo-black) — sync tokens-oklch.css

// ── Scene tuning ─────────────────────────────────────────────────────────────
const STAR_N   = 2500;   // adaptive: halved on constrained devices below
const BEACON_N = 5;      // self + 4 family
const FIELD_R  = 60;     // half-extent of the star cloud (world units)
const FOG_NEAR = 40;
const FOG_FAR  = 130;

// ── GLSL: additive point sprites with fog attenuation + soft round falloff ───
const VERT = /* glsl */ `
  attribute float aSize;
  attribute float aTwinkle;   // phase offset for per-star breathing
  attribute vec3  aColor;
  uniform float uTime;
  uniform float uPixelRatio;
  varying vec3  vColor;
  varying float vFog;
  void main() {
    vColor = aColor;
    vec4 mv = modelViewMatrix * vec4(position, 1.0);
    // slow twinkle on size only (transform-equivalent; cheap)
    float tw = 0.75 + 0.25 * sin(uTime * 1.4 + aTwinkle);
    gl_PointSize = aSize * tw * uPixelRatio * (140.0 / -mv.z);
    // linear fog factor 1(near)->0(far) so distant stars melt into the canvas
    float depth = -mv.z;
    vFog = clamp((${FOG_FAR.toFixed(1)} - depth) / (${FOG_FAR.toFixed(1)} - ${FOG_NEAR.toFixed(1)}), 0.0, 1.0);
    gl_Position = projectionMatrix * mv;
  }
`;

const FRAG = /* glsl */ `
  precision mediump float;
  varying vec3  vColor;
  varying float vFog;
  void main() {
    // round sprite: radial soft falloff, discard the square corners
    vec2 uv = gl_PointCoord - vec2(0.5);
    float d = length(uv);
    if (d > 0.5) discard;
    float alpha = smoothstep(0.5, 0.0, d);
    gl_FragColor = vec4(vColor, alpha * vFog);
  }
`;

/** Constrained device? halve the star count to protect the CPU/GPU loop. */
function starBudget() {
  const mem = typeof navigator !== 'undefined' ? navigator.deviceMemory : undefined;
  return typeof mem === 'number' && mem <= 8 ? Math.round(STAR_N * 0.6) : STAR_N;
}

/** Kill-criterion counter — no analytics infra exists in lib/, so localStorage.
 *  Bumped once per real mount; read manually to judge whether the scene earns
 *  its bundle. Best-effort; never throws into the mount path. */
function bumpMountCounter() {
  try {
    if (typeof localStorage === 'undefined') return;
    const k = 'kin.heroConstellation.mounts';
    localStorage.setItem(k, String((parseInt(localStorage.getItem(k), 10) || 0) + 1));
  } catch { /* private mode / disabled storage — ignore */ }
}

/**
 * Mount the constellation onto an existing <canvas>.
 * @param {HTMLCanvasElement} canvas
 * @param {{ getScroll?: () => number, onMounted?: () => void }} [opts]
 * @returns {{ dispose: () => void }}
 */
export function mount(canvas, opts = {}) {
  const getScroll = typeof opts.getScroll === 'function' ? opts.getScroll : () => 0;

  let disposed = false;
  let raf = null;
  let firstFrame = true;

  // ── Renderer ───────────────────────────────────────────────────────────────
  let renderer;
  try {
    renderer = new THREE.WebGLRenderer({
      canvas,
      alpha: true,           // static SVG poster shows through
      antialias: false,      // additive points don't need MSAA; saves fill
      powerPreference: 'high-performance',
      failIfMajorPerformanceCaveat: false,
    });
  } catch {
    // Context creation failed — leave the poster, surface nothing.
    return { dispose() {} };
  }

  const pixelRatio = Math.min(window.devicePixelRatio || 1, 2);
  renderer.setPixelRatio(pixelRatio);
  renderer.setClearColor(0x000000, 0); // raw-color-ok — fully transparent clear (poster shows through)

  const scene = new THREE.Scene();
  scene.fog = new THREE.Fog(FOG_HEX, FOG_NEAR, FOG_FAR);

  const camera = new THREE.PerspectiveCamera(46, 1, 0.1, 400);
  camera.position.set(0, 0, 96);

  // ── Geometry: stars + beacons packed into one BufferGeometry ─────────────────
  const starN = starBudget();
  const total = starN + BEACON_N;
  const positions = new Float32Array(total * 3);
  const colors    = new Float32Array(total * 3);
  const sizes     = new Float32Array(total);
  const twinkles  = new Float32Array(total);

  const star = new THREE.Color(STAR_HEX);
  for (let i = 0; i < starN; i++) {
    const o = i * 3;
    // random point in a flattened box (wider than deep) for a sky-slab look
    positions[o]     = (Math.random() * 2 - 1) * FIELD_R * 1.6;
    positions[o + 1] = (Math.random() * 2 - 1) * FIELD_R * 0.9;
    positions[o + 2] = (Math.random() * 2 - 1) * FIELD_R * 0.7 - 10;
    // subtle per-star hue drift toward one member twin for a colored haze
    const c = star.clone();
    if (Math.random() < 0.12) {
      c.lerp(new THREE.Color(BEACON_HEX[1 + ((Math.random() * 4) | 0)]), 0.5);
    }
    colors[o] = c.r; colors[o + 1] = c.g; colors[o + 2] = c.b;
    sizes[i] = 1.1 + Math.random() * 1.8;
    twinkles[i] = Math.random() * Math.PI * 2;
  }

  // Beacons: larger, brighter, clustered near the focal plane so they read as
  // "family in the foreground." Self (brand) sits centre-ish; members orbit.
  const beaconLayout = [
    [0, 2, 8], [-26, 10, -4], [24, -6, 2], [-14, -14, 6], [16, 16, -6],
  ];
  for (let b = 0; b < BEACON_N; b++) {
    const i = starN + b;
    const o = i * 3;
    const [x, y, z] = beaconLayout[b];
    positions[o] = x; positions[o + 1] = y; positions[o + 2] = z;
    const c = new THREE.Color(BEACON_HEX[b]);
    colors[o] = c.r; colors[o + 1] = c.g; colors[o + 2] = c.b;
    sizes[i] = 8.5 + (b === 0 ? 3 : 0); // self beacon slightly larger
    twinkles[i] = b * 1.3;
  }

  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
  geometry.setAttribute('aColor',   new THREE.BufferAttribute(colors, 3));
  geometry.setAttribute('aSize',    new THREE.BufferAttribute(sizes, 1));
  geometry.setAttribute('aTwinkle', new THREE.BufferAttribute(twinkles, 1));

  const material = new THREE.ShaderMaterial({
    uniforms: {
      uTime:       { value: 0 },
      uPixelRatio: { value: pixelRatio },
    },
    vertexShader: VERT,
    fragmentShader: FRAG,
    transparent: true,
    depthWrite: false,
    depthTest: false,
    blending: THREE.AdditiveBlending, // per-particle glow
  });

  const points = new THREE.Points(geometry, material);
  scene.add(points);

  // ── Passive cursor parallax (±3° tilt, rAF-lerped — no per-move work) ────────
  let tiltX = 0, tiltY = 0, targetTX = 0, targetTY = 0;
  function onPointer(e) {
    const nx = (e.clientX / window.innerWidth) * 2 - 1;
    const ny = (e.clientY / window.innerHeight) * 2 - 1;
    targetTY = nx * 0.052;  // ~±3° in radians
    targetTX = ny * 0.052;
  }
  window.addEventListener('pointermove', onPointer, { passive: true });

  // ── Sizing ───────────────────────────────────────────────────────────────────
  function resize() {
    const w = canvas.clientWidth || window.innerWidth;
    const h = canvas.clientHeight || Math.round(window.innerHeight * 0.9);
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
  }
  const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(resize) : null;
  if (ro) ro.observe(canvas);
  else window.addEventListener('resize', resize);
  resize();

  // ── Pause hygiene: visibility + off-screen (IntersectionObserver) ────────────
  let pageVisible = typeof document === 'undefined' || !document.hidden;
  let onScreen = true;
  function running() { return !disposed && pageVisible && onScreen; }

  function onVisibility() {
    pageVisible = !document.hidden;
    if (running() && raf === null) { last = performance.now(); loop(last); }
  }
  document.addEventListener('visibilitychange', onVisibility);

  // Pause when the hero scrolls >150% out of the viewport (rootMargin buffer).
  const io = typeof IntersectionObserver !== 'undefined'
    ? new IntersectionObserver((entries) => {
        for (const en of entries) onScreen = en.isIntersecting;
        if (running() && raf === null) { last = performance.now(); loop(last); }
      }, { rootMargin: '50% 0px 50% 0px', threshold: 0 })
    : null;
  if (io) io.observe(canvas);

  // ── Context-loss: tear down permanently for the session, keep the poster ─────
  function onContextLost(ev) {
    ev.preventDefault();
    dispose();
  }
  canvas.addEventListener('webglcontextlost', onContextLost, false);

  // ── rAF loop — reads scroll passively, no scroll handler ─────────────────────
  const clock = new THREE.Clock();
  let last = performance.now();
  function loop() {
    if (!running()) { raf = null; return; }
    raf = requestAnimationFrame(loop);

    const t = clock.getElapsedTime();
    material.uniforms.uTime.value = t;

    // scroll dolly: hand the field forward to the product mockup as the story
    // begins (progress 0 -> 1 pushes the camera in and lifts it slightly).
    const p = Math.min(Math.max(getScroll(), 0), 1);
    camera.position.z = 96 - p * 46;
    camera.position.y = p * 8;

    // cursor tilt (lerped)
    tiltX += (targetTX - tiltX) * 0.05;
    tiltY += (targetTY - tiltY) * 0.05;
    points.rotation.x = tiltX + Math.sin(t * 0.05) * 0.01;
    points.rotation.y = tiltY + t * 0.012; // slow perpetual drift
    camera.lookAt(0, camera.position.y * 0.4, 0);

    renderer.render(scene, camera);

    if (firstFrame) {
      firstFrame = false;
      bumpMountCounter();
      if (typeof opts.onMounted === 'function') { try { opts.onMounted(); } catch { /* caller error — non-fatal */ } }
    }
  }
  loop();

  // ── Teardown: release everything (SPA max-16-GL-context hygiene) ─────────────
  function dispose() {
    if (disposed) return;
    disposed = true;
    if (raf !== null) { cancelAnimationFrame(raf); raf = null; }
    window.removeEventListener('pointermove', onPointer);
    document.removeEventListener('visibilitychange', onVisibility);
    canvas.removeEventListener('webglcontextlost', onContextLost, false);
    if (ro) ro.disconnect(); else window.removeEventListener('resize', resize);
    if (io) io.disconnect();
    geometry.dispose();
    material.dispose();
    try { renderer.dispose(); renderer.forceContextLoss(); } catch { /* already lost */ }
  }

  return { dispose };
}

export default { mount };
