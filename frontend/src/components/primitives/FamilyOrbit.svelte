<script>
  /**
   * FamilyOrbit — True 3D Orrery
   *
   * Satellites are projected from 3D orbital coordinates to 2D screen space
   * using perspective math. The orbital plane is tilted 72° from horizontal,
   * making rings appear as narrow ellipses. Near-side satellites are larger
   * and brighter; far-side are smaller and dimmer. Depth sorting ensures
   * correct occlusion with the sun.
   *
   * Projection model:
   *   x3 = r·cos(θ)          — horizontal orbital coordinate
   *   z3 = r·sin(θ)          — in-plane coordinate (tilts into screen)
   *   screenY = z3·cos(TILT) — vertical screen displacement
   *   depth   = z3·sin(TILT) — depth (+front / −back)
   *   scale   = 1 + depth·PERSP/W — perspective scale factor
   */
  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { otherUsers, focusUser } from '../../lib/stores/map.js';
  import { authUser } from '../../lib/stores/auth.js';
  import { getUserColor } from '../../lib/getUserColor.js';

  // ── Scene ────────────────────────────────────────────────────────────────
  const SCENE = 330;
  const CX    = SCENE / 2;
  const CY    = SCENE / 2;
  const BASE  = 58;   // innermost orbit radius
  const STEP  = 19;   // spacing between orbits
  const MAX   = 6;    // max visible members
  const SAT_D = 30;   // base satellite diameter (unscaled)

  // ── 3D projection parameters ─────────────────────────────────────────────
  const TILT  = 72 * (Math.PI / 180); // orbital plane tilt from horizontal
  const COS_T = Math.cos(TILT);       // ≈ 0.309  — vertical compression
  const SIN_T = Math.sin(TILT);       // ≈ 0.951  — depth factor
  const PERSP = 0.52;                 // perspective strength

  /** Project orbital-plane (r, angle) → screen (x, y, scale, depth) */
  function project(r, a) {
    const x3 = r * Math.cos(a);
    const z3 = r * Math.sin(a);
    const sy  = z3 * COS_T;
    const dep = z3 * SIN_T;
    const sc  = 1 + (dep * PERSP) / SCENE;
    return { x: CX + x3 * sc, y: CY + sy * sc, scale: sc, depth: dep };
  }

  // ── Reactive member data ─────────────────────────────────────────────────
  $: allMembers = Array.from($otherUsers.values());
  $: overflow   = allMembers.length - MAX;

  $: orbitDescs = allMembers.slice(0, MAX).map((user, i) => {
    const isSos    = !!user.sos?.active;
    const isOnline = user.online !== false;
    const isMoving = isOnline && (user.speed || 0) > 1;
    const color    = isSos ? '#ef4444' : getUserColor(user.userId);
    const r        = BASE + i * STEP;
    const period   = isSos ? 4.5 : isMoving ? 9 : isOnline ? 22 : 58;
    const angSpd   = (2 * Math.PI) / period;
    const phase    = (i / Math.max(allMembers.length, 1)) * 2 * Math.PI;
    return { user, color, r, angSpd, phase, isSos, isOnline, isMoving };
  });

  // Precomputed ellipse rx/ry for each orbit ring
  $: rings = orbitDescs.map(od => ({
    rx: od.r,
    ry: od.r * COS_T,
    color: od.color,
    isSos: od.isSos,
    isOnline: od.isOnline,
  }));

  // ── RAF animation loop (capped ~30 fps for mobile) ──────────────────────
  let rafId, t0 = 0, lastFrame = 0;
  let renderList = [];

  const TRAIL_LAGS = [0.055, 0.115, 0.190]; // seconds behind for each trail dot

  function tick(ts) {
    rafId = requestAnimationFrame(tick);
    if (ts - lastFrame < 32) return; // ~30fps cap
    lastFrame = ts;
    if (!t0) t0 = ts;
    const e = (ts - t0) / 1000;

    const items = orbitDescs.map(od => {
      const a     = od.phase + od.angSpd * e;
      const proj  = project(od.r, a);
      const trail = TRAIL_LAGS.map(lag =>
        project(od.r, od.phase + od.angSpd * (e - lag))
      );
      return { ...od, ...proj, a, trail };
    });

    // Sort back→front so near satellites render on top of far ones
    renderList = items.sort((a, b) => a.depth - b.depth);
  }

  // ── Helpers ──────────────────────────────────────────────────────────────
  function initials(name) {
    return (name || '').split(' ').map(s => s[0]).join('').toUpperCase().slice(0, 2) || '?';
  }
  function locate(user) {
    focusUser.set(user.userId);
    push('/');
  }

  onMount(()   => { rafId = requestAnimationFrame(tick); });
  onDestroy(() => { if (rafId) cancelAnimationFrame(rafId); });
</script>

<!-- ══════════════════════════════════════════════════════════════════════════
     Scene container — all layers are absolutely positioned inside this box
══════════════════════════════════════════════════════════════════════════════ -->
<div
  class="fo-scene"
  class:fo-scene-empty={allMembers.length === 0}
  style="width:min({allMembers.length > 0 ? SCENE : 200}px,calc(100vw - 20px));height:min({allMembers.length > 0 ? SCENE : 200}px,calc(100vw - 20px))"
  aria-label="Family orbital view"
>

  <!-- ── LAYER 0: Deep-space radial vignette ──────────────────────────── -->
  <div class="fo-space" aria-hidden="true"></div>

  <!-- ── LAYER 1: Far-side (dim) orbit arcs ───────────────────────────── -->
  <!--   Path: M (right equator) → counterclockwise arc via TOP → (left equator)  -->
  <!--   The top of each ellipse = far side (behind sun)  = dim              -->
  <svg class="fo-svg fo-svg-back" width={SCENE} height={SCENE}
       viewBox="0 0 {SCENE} {SCENE}" aria-hidden="true">
    <defs>
      <filter id="fo-ring-glow" x="-30%" y="-30%" width="160%" height="160%">
        <feGaussianBlur stdDeviation="1.0" result="b"/>
        <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
      </filter>
    </defs>

    {#each rings as ring, i}
      <!-- Far half: from right → counterclockwise via top → left -->
      <path
        d="M {CX + ring.rx},{CY} A {ring.rx},{ring.ry} 0 0 0 {CX - ring.rx},{CY}"
        fill="none"
        stroke="{ring.color}"
        stroke-width="{ring.isSos ? 1.4 : 0.85}"
        opacity="{ring.isOnline ? 0.16 : 0.07}"
        filter="url(#fo-ring-glow)"
      />
    {/each}

    <!-- Empty-state ghost ellipses -->
    {#if allMembers.length === 0}
      {#each [BASE, BASE + STEP, BASE + STEP * 2] as r}
        {@const ry = r * COS_T}
        <path d="M {CX+r},{CY} A {r},{ry} 0 1 0 {CX-r},{CY}"
          fill="none" stroke="rgba(255,255,255,0.045)" stroke-width="1" stroke-dasharray="3 9"/>
      {/each}
    {/if}
  </svg>

  <!-- ── LAYER 2: Sun (z≈mid) — volumetric sphere + corona + flare ─────── -->
  <div class="fo-sun-wrap" aria-label="{$authUser?.displayName || 'You'} — centre">

    <!-- 3-layer atmospheric halo -->
    <div class="fo-halo fo-halo-3" aria-hidden="true"></div>
    <div class="fo-halo fo-halo-2" aria-hidden="true"></div>
    <div class="fo-halo fo-halo-1" aria-hidden="true"></div>

    <!-- Sun sphere -->
    <div class="fo-sun" aria-hidden="true">
      <!-- Specular highlight (top-left bright spot) -->
      <div class="fo-sun-spec" aria-hidden="true"></div>
      <!-- Kinnect icon -->
      <div class="fo-sun-icon">
        <svg viewBox="0 0 20 20" fill="currentColor" width="19" height="19" aria-hidden="true">
          <path d="M10 1.5A6 6 0 0 0 4 7.5C4 12 10 18.5 10 18.5S16 12 16 7.5a6 6 0 0 0-6-6zm0 8.5a2.5 2.5 0 1 1 0-5 2.5 2.5 0 0 1 0 5z"/>
        </svg>
      </div>
    </div>

    <!-- 3 corona pulse rings -->
    <div class="fo-corona fo-corona-1" aria-hidden="true"></div>
    <div class="fo-corona fo-corona-2" aria-hidden="true"></div>
    <div class="fo-corona fo-corona-3" aria-hidden="true"></div>

    <!-- Lens flare cross (slow rotation) -->
    <svg class="fo-flare" width="130" height="130" viewBox="-65 -65 130 130" aria-hidden="true">
      <line x1="-62" y1="0"   x2="62"  y2="0"   stroke="rgba(210,195,255,0.09)" stroke-width="1"/>
      <line x1="0"   y1="-62" x2="0"   y2="62"  stroke="rgba(210,195,255,0.09)" stroke-width="1"/>
      <line x1="-45" y1="-45" x2="45"  y2="45"  stroke="rgba(210,195,255,0.05)" stroke-width="0.7"/>
      <line x1="45"  y1="-45" x2="-45" y2="45"  stroke="rgba(210,195,255,0.05)" stroke-width="0.7"/>
      <!-- Hexagonal diffraction ring -->
      <circle cx="0" cy="0" r="36" fill="none"
        stroke="rgba(200,180,255,0.04)" stroke-width="0.8" stroke-dasharray="6 18"/>
    </svg>
  </div>

  <!-- ── LAYER 3: Near-side (bright) orbit arcs — above the sun ────────── -->
  <!--   Path: M (left equator) → counterclockwise arc via BOTTOM → (right)  -->
  <!--   The bottom of each ellipse = near side (front of sun) = bright       -->
  <svg class="fo-svg fo-svg-front" width={SCENE} height={SCENE}
       viewBox="0 0 {SCENE} {SCENE}" aria-hidden="true">
    {#each rings as ring, i}
      <!-- Near half: from left → counterclockwise via bottom → right -->
      <path
        d="M {CX - ring.rx},{CY} A {ring.rx},{ring.ry} 0 0 0 {CX + ring.rx},{CY}"
        fill="none"
        stroke="{ring.color}"
        stroke-width="{ring.isSos ? 1.9 : 1.2}"
        opacity="{ring.isOnline ? 0.62 : 0.18}"
      />

      <!-- Near-side accent glow (blurred duplicate for bloom effect) -->
      <path
        d="M {CX - ring.rx},{CY} A {ring.rx},{ring.ry} 0 0 0 {CX + ring.rx},{CY}"
        fill="none"
        stroke="{ring.color}"
        stroke-width="3"
        opacity="{ring.isOnline ? 0.10 : 0.04}"
        style="filter:blur(2px)"
      />
    {/each}
  </svg>

  <!-- ── LAYER 4: Trail particles (behind each satellite) ──────────────── -->
  {#each renderList as sat (sat.user.userId)}
    {#if sat.isMoving}
      {#each sat.trail as tp, ti}
        <span
          class="fo-trail"
          style="
            left:{tp.x}px;
            top:{tp.y}px;
            width:{Math.max(2.5, 5 * tp.scale * (0.85 - ti * 0.22))}px;
            height:{Math.max(2.5, 5 * tp.scale * (0.85 - ti * 0.22))}px;
            background:{sat.color};
            opacity:{0.42 - ti * 0.13};
            z-index:{Math.round((sat.depth + 160) * 10)};
          "
          aria-hidden="true"
        ></span>
      {/each}
    {/if}
  {/each}

  <!-- ── LAYER 5: Satellite buttons (depth-sorted) ─────────────────────── -->
  {#each renderList as sat (sat.user.userId)}
    {@const sz = SAT_D * sat.scale}
    {@const depthOp = Math.min(1, 0.52 + sat.depth / 175)}
    <button
      class="fo-sat"
      class:fo-sat-sos={sat.isSos}
      class:fo-sat-offline={!sat.isOnline}
      class:fo-sat-moving={sat.isMoving}
      style="
        left:{sat.x}px;
        top:{sat.y}px;
        width:{sz}px;
        height:{sz}px;
        --color:{sat.color};
        --sz:{sz}px;
        opacity:{sat.isOnline ? depthOp : 0.26};
        z-index:{Math.round((sat.depth + 160) * 10) + 1};
        font-size:{Math.max(7, 9 * sat.scale)}px;
        box-shadow:
          0 0 {Math.round(9 * sat.scale)}px color-mix(in srgb,{sat.color} {sat.isSos?82:50}%,transparent),
          0 0 {Math.round(20 * sat.scale)}px color-mix(in srgb,{sat.color} 22%,transparent),
          inset 0 {Math.round(sat.scale)}px 0 rgba(255,255,255,0.15);
      "
      on:click={() => locate(sat.user)}
      aria-label="{sat.user.displayName} — {sat.isSos ? 'SOS active' : sat.isMoving ? 'Moving' : sat.isOnline ? 'Online' : 'Offline'} — tap to locate on map"
    >
      <span class="fo-initials">{initials(sat.user.displayName)}</span>
      {#if sat.isSos}
        <span class="fo-sos-ring" aria-hidden="true"></span>
        <span class="fo-sos-ring fo-sos-ring-2" aria-hidden="true"></span>
      {/if}
    </button>
  {/each}

  <!-- Overflow badge -->
  {#if overflow > 0}
    <div class="fo-overflow" aria-label="{overflow} more family members">+{overflow}</div>
  {/if}

  <!-- Empty state -->
  {#if allMembers.length === 0}
    <div class="fo-empty" aria-live="polite">Waiting for family to join…</div>
  {/if}

</div>

<!-- ── Name tags below the orbit ────────────────────────────────────────── -->
{#if renderList.length > 0}
  <div class="fo-tags" role="list" aria-label="Family members">
    {#each renderList as sat (sat.user.userId)}
      <button
        class="fo-tag"
        class:fo-tag-sos={sat.isSos}
        class:fo-tag-offline={!sat.isOnline}
        style="--color:{sat.color}"
        on:click={() => locate(sat.user)}
        role="listitem"
        aria-label="{sat.user.displayName}"
      >
        <span class="fo-tag-dot" aria-hidden="true"></span>
        <span class="fo-tag-name">{sat.user.displayName}</span>
        {#if sat.isMoving && !sat.isSos}
          <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24"
               fill="none" stroke="currentColor" stroke-width="3" aria-hidden="true">
            <polyline points="13 17 18 12 13 7"/>
            <polyline points="6 17 11 12 6 7"/>
          </svg>
        {/if}
      </button>
    {/each}
  </div>
{/if}

<style>
  /* ── Scene ─────────────────────────────────────────────────────────────── */
  .fo-scene {
    position: relative;
    margin: 0 auto;
    flex-shrink: 0;
    overflow: visible;
    /* No border-radius, no background — seamlessly merges with the dashboard's
       deep-space backdrop. The orbit floats in the existing cosmos. */
    background: transparent;
  }

  /* ── Subtle depth ring — faint radial gradient that doesn't look like a disk */
  .fo-space {
    position: absolute;
    inset: -10%;
    border-radius: 50%;
    background: radial-gradient(
      ellipse 70% 60% at 50% 50%,
      rgba(99, 102, 241, 0.04) 0%,
      transparent 65%
    );
    pointer-events: none;
    z-index: 0;
  }

  /* ── SVG layers ─────────────────────────────────────────────────────────── */
  .fo-svg {
    position: absolute;
    inset: 0;
    pointer-events: none;
    z-index: 1;
  }
  .fo-svg-front { z-index: 60; }  /* above the sun (z=40) */

  /* ── Sun ────────────────────────────────────────────────────────────────── */
  .fo-sun-wrap {
    position: absolute;
    left: 50%; top: 50%;
    transform: translate(-50%, -50%);
    z-index: 40;
    display: flex;
    align-items: center;
    justify-content: center;
    pointer-events: none;
  }

  /* Volumetric atmosphere halos */
  .fo-halo {
    position: absolute;
    border-radius: 50%;
    pointer-events: none;
  }
  .fo-halo-3 {
    width: 130px; height: 130px;
    background: radial-gradient(circle, rgba(99,102,241,0.11) 0%, transparent 70%);
    animation: halo-pulse 5s ease-in-out infinite;
  }
  .fo-halo-2 {
    width: 88px; height: 88px;
    background: radial-gradient(circle, rgba(139,92,246,0.20) 0%, transparent 65%);
    animation: halo-pulse 3.8s ease-in-out infinite 0.6s;
  }
  .fo-halo-1 {
    width: 58px; height: 58px;
    background: radial-gradient(circle, rgba(200,180,255,0.32) 0%, transparent 60%);
    animation: halo-pulse 2.8s ease-in-out infinite 1.2s;
  }
  @keyframes halo-pulse {
    0%, 100% { transform: scale(1.00); opacity: 0.75; }
    50%       { transform: scale(1.14); opacity: 1.00; }
  }

  /* Main sun sphere — radial gradient simulates 3D sphere with light from top-left */
  .fo-sun {
    position: relative;
    width: 46px; height: 46px;
    border-radius: 50%;
    background: radial-gradient(
      circle at 36% 30%,
      rgba(255, 255, 255, 0.92) 0%,
      #c4b5fd                   14%,
      var(--primary-500, #6366f1) 34%,
      var(--primary-700, #4338ca) 58%,
      rgba(15, 8, 50, 0.85)     82%,
      rgba(0, 0, 0, 0.60)       100%
    );
    box-shadow:
      0 0 0 1.5px rgba(255, 255, 255, 0.22),
      0 0 14px  var(--primary-400, #818cf8),
      0 0 36px  color-mix(in srgb, var(--primary-500, #6366f1) 58%, transparent),
      0 0 80px  color-mix(in srgb, var(--primary-600, #4f46e5) 28%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    animation: sun-breathe 3.2s ease-in-out infinite;
    z-index: 1;
  }
  @keyframes sun-breathe {
    0%, 100% { filter: brightness(1.00) saturate(1.00); }
    50%       { filter: brightness(1.20) saturate(1.18); }
  }

  /* Specular highlight — offset bright spot top-left */
  .fo-sun-spec {
    position: absolute;
    width: 15px; height: 15px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(255,255,255,0.88) 0%, transparent 70%);
    top: 5px; left: 7px;
    pointer-events: none;
    filter: blur(1.5px);
    z-index: 2;
  }

  .fo-sun-icon {
    position: relative;
    z-index: 3;
    color: rgba(255, 255, 255, 0.92);
    filter: drop-shadow(0 0 5px rgba(255,255,255,0.75));
    display: flex;
    align-items: center;
    justify-content: center;
  }

  /* 3 staggered corona pulses */
  .fo-corona {
    position: absolute;
    border-radius: 50%;
    border-style: solid;
    pointer-events: none;
  }
  .fo-corona-1 {
    width: 58px; height: 58px;
    border-width: 1.5px;
    border-color: color-mix(in srgb, var(--primary-400, #818cf8) 48%, transparent);
    animation: corona-expand 3.2s ease-out infinite;
  }
  .fo-corona-2 {
    width: 58px; height: 58px;
    border-width: 1px;
    border-color: color-mix(in srgb, var(--primary-300, #a5b4fc) 32%, transparent);
    animation: corona-expand 3.2s ease-out infinite 1.07s;
  }
  .fo-corona-3 {
    width: 58px; height: 58px;
    border-width: 1px;
    border-color: color-mix(in srgb, var(--primary-300, #a5b4fc) 20%, transparent);
    animation: corona-expand 3.2s ease-out infinite 2.13s;
  }
  @keyframes corona-expand {
    0%   { transform: scale(1.00); opacity: 0.80; }
    100% { transform: scale(2.60); opacity: 0.00; }
  }

  /* Rotating lens-flare cross */
  .fo-flare {
    position: absolute;
    pointer-events: none;
    animation: flare-spin 18s linear infinite;
    opacity: 0.85;
  }
  @keyframes flare-spin {
    from { transform: rotate(0deg); }
    to   { transform: rotate(360deg); }
  }

  /* ── Satellite ──────────────────────────────────────────────────────────── */
  .fo-sat {
    position: absolute;
    transform: translate(-50%, -50%);
    border-radius: 50%;
    background: color-mix(in srgb, var(--color, #6366f1) 24%, rgba(5, 8, 22, 0.90));
    border: 2px solid var(--color, #6366f1);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    will-change: left, top, width, height, opacity;
    transition: transform 0.18s ease, border-width 0.15s ease;
    -webkit-tap-highlight-color: transparent;
  }
  .fo-sat:hover {
    transform: translate(-50%, -50%) scale(1.24);
    border-width: 2.5px;
  }
  .fo-sat:active { transform: translate(-50%, -50%) scale(0.88); }

  .fo-initials {
    font-weight: 900;
    color: var(--color, #6366f1);
    line-height: 1;
    user-select: none;
    letter-spacing: -0.04em;
    font-family: var(--font-display, system-ui, sans-serif);
  }

  /* SOS variant */
  .fo-sat-sos {
    border-color: #ef4444 !important;
    background: rgba(239, 68, 68, 0.22) !important;
  }
  .fo-sat-sos .fo-initials { color: #f87171; }

  /* Double SOS ring */
  .fo-sos-ring {
    position: absolute;
    inset: -5px;
    border-radius: 50%;
    border: 2px solid rgba(239, 68, 68, 0.60);
    pointer-events: none;
    animation: sos-pulse 1s ease-out infinite;
  }
  .fo-sos-ring-2 {
    inset: -10px;
    border-color: rgba(239, 68, 68, 0.30);
    animation: sos-pulse 1s ease-out infinite 0.5s;
  }
  @keyframes sos-pulse {
    from { transform: scale(1.0); opacity: 0.85; }
    to   { transform: scale(1.7); opacity: 0.00; }
  }

  /* Offline */
  .fo-sat-offline {
    filter: saturate(0.08) brightness(0.55);
  }

  /* ── Motion trail ───────────────────────────────────────────────────────── */
  .fo-trail {
    position: absolute;
    border-radius: 50%;
    transform: translate(-50%, -50%);
    pointer-events: none;
    will-change: left, top;
  }

  /* ── Overflow badge ─────────────────────────────────────────────────────── */
  .fo-overflow {
    position: absolute;
    bottom: 10px; right: 14px;
    font-size: 10px; font-weight: 800;
    color: rgba(255, 255, 255, 0.35);
    background: rgba(255, 255, 255, 0.07);
    border: 1px solid rgba(255, 255, 255, 0.10);
    border-radius: 20px;
    padding: 2px 8px;
    z-index: 100;
    letter-spacing: 0.02em;
  }

  /* ── Empty state ────────────────────────────────────────────────────────── */
  .fo-scene-empty {
    transition: width 0.5s ease, height 0.5s ease;
  }
  .fo-empty {
    position: absolute;
    bottom: 6px; left: 50%;
    transform: translateX(-50%);
    font-size: 11px; font-weight: 500;
    color: rgba(255, 255, 255, 0.28);
    white-space: nowrap;
    pointer-events: none;
    z-index: 100;
    letter-spacing: 0.04em;
  }

  /* ── Name tags ──────────────────────────────────────────────────────────── */
  .fo-tags {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 6px;
    padding: 4px 16px 0;
    max-width: 340px;
    margin: 0 auto;
  }
  .fo-tag {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 4px 9px;
    border-radius: 9999px;
    background: color-mix(in srgb, var(--color, #6366f1) 12%, rgba(5, 8, 18, 0.70));
    border: 1px solid color-mix(in srgb, var(--color, #6366f1) 30%, transparent);
    color: color-mix(in srgb, var(--color, #6366f1) 85%, rgba(255, 255, 255, 0.90));
    font-size: 11px; font-weight: 700;
    cursor: pointer;
    letter-spacing: 0.01em;
    transition: background 0.15s ease, box-shadow 0.15s ease, transform 0.12s ease;
    -webkit-tap-highlight-color: transparent;
    font-family: var(--font-display, system-ui, sans-serif);
  }
  .fo-tag:hover {
    background: color-mix(in srgb, var(--color, #6366f1) 22%, rgba(5, 8, 18, 0.80));
    box-shadow: 0 0 10px color-mix(in srgb, var(--color, #6366f1) 30%, transparent);
    transform: translateY(-1px);
  }
  .fo-tag:active { transform: scale(0.95); }
  .fo-tag-dot {
    width: 6px; height: 6px;
    border-radius: 50%;
    background: var(--color, #6366f1);
    flex-shrink: 0;
    box-shadow: 0 0 5px var(--color, #6366f1);
  }
  .fo-tag-name {
    max-width: 70px;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .fo-tag-sos    { --color: #ef4444; animation: tag-sos-blink 1.2s ease-in-out infinite; }
  .fo-tag-offline { opacity: 0.42; filter: saturate(0.15); }
  @keyframes tag-sos-blink {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0.52; }
  }

  /* ── Reduced motion ─────────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .fo-sun, .fo-halo, .fo-corona, .fo-flare,
    .fo-sos-ring, .fo-tag-sos {
      animation: none !important;
      transition: none !important;
    }
  }
</style>
