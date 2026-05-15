<script>
  /**
   * FamilyOrbit — Interactive 3D Holographic Radar
   *
   * Full-size canvas radar with touch/click interactions:
   * - Tap anywhere: ripple shockwave
   * - Tap center: energy burst explosion
   * - Rotating scan, HUD arcs, energy beams to satellites
   */
  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { otherUsers, focusUser } from '../../lib/stores/map.js';
  import { authUser } from '../../lib/stores/auth.js';
  import { getUserColor } from '../../lib/getUserColor.js';

  // ── Scene ──────────────────────────────────────────────────────────────
  let sceneEl;
  let sceneSize = 400;

  const BASE  = 72;
  const STEP  = 24;
  const MAX   = 6;
  const SAT_D = 32;

  $: CX = sceneSize / 2;
  $: CY = sceneSize / 2;

  // ── 3D projection ─────────────────────────────────────────────────────
  const TILT  = 68 * (Math.PI / 180);
  const COS_T = Math.cos(TILT);
  const SIN_T = Math.sin(TILT);
  const PERSP = 0.5;

  function project(r, a) {
    const x3 = r * Math.cos(a);
    const z3 = r * Math.sin(a);
    const sy  = z3 * COS_T;
    const dep = z3 * SIN_T;
    const sc  = 1 + (dep * PERSP) / sceneSize;
    return { x: CX + x3 * sc, y: CY + sy * sc, scale: sc, depth: dep };
  }

  // ── Reactive data ─────────────────────────────────────────────────────
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

  // ── Canvas + state ────────────────────────────────────────────────────
  let canvas;
  let rafId, t0 = 0, lastFrame = 0;
  let renderList = [];
  let scanAngle = 0;
  let elapsed = 0;

  const TRAIL_LAGS = [0.06, 0.14, 0.24];

  // ── Interactive effects ───────────────────────────────────────────────
  let ripples = [];      // { x, y, t, maxR, color }
  let coreBurst = 0;     // 0..1 burst animation progress
  let coreBurstTime = 0;

  function addRipple(x, y, isCenter = false) {
    ripples.push({
      x, y,
      t: elapsed,
      maxR: isCenter ? sceneSize * 0.6 : sceneSize * 0.35,
      color: isCenter ? 'rgba(139,92,246,' : 'rgba(99,102,241,',
      duration: isCenter ? 1.2 : 0.8,
    });
    if (isCenter) {
      coreBurst = 1;
      coreBurstTime = elapsed;
    }
    // Cap ripples
    if (ripples.length > 8) ripples = ripples.slice(-8);
  }

  function handlePointer(e) {
    const rect = sceneEl.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    const dx = x - rect.width / 2;
    const dy = y - rect.height / 2;
    const distFromCenter = Math.sqrt(dx * dx + dy * dy);

    if (distFromCenter < 35) {
      addRipple(rect.width / 2, rect.height / 2, true);
    } else {
      addRipple(x, y, false);
    }
  }

  // ── Particles ─────────────────────────────────────────────────────────
  const PARTICLES = Array.from({ length: 55 }, (_, i) => {
    const a = (i * 2654435761) >>> 0;
    const b = (i * 1664525 + 1013904223) >>> 0;
    return {
      r: 25 + (a % 160),
      phase: (b % 628) / 100,
      speed: 0.03 + (a % 40) / 800,
      size: 0.4 + (b % 18) / 18,
      brightness: 0.12 + (a % 25) / 100,
    };
  });

  // ── Drawing ───────────────────────────────────────────────────────────
  function draw(ctx, W, H, e) {
    ctx.clearRect(0, 0, W, H);
    const cx = W / 2, cy = H / 2;

    const gridRings = allMembers.length > 0
      ? orbitDescs.map(od => od.r)
      : [BASE, BASE + STEP, BASE + STEP * 2];
    const outerR = (gridRings[gridRings.length - 1] || BASE + STEP * 2) + 20;

    // ── Ambient glow ──────────────────────────────────────────────
    const glow = ctx.createRadialGradient(cx, cy, 0, cx, cy, outerR);
    glow.addColorStop(0, 'rgba(99,102,241,0.14)');
    glow.addColorStop(0.3, 'rgba(99,102,241,0.06)');
    glow.addColorStop(0.7, 'rgba(139,92,246,0.02)');
    glow.addColorStop(1, 'transparent');
    ctx.fillStyle = glow;
    ctx.fillRect(0, 0, W, H);

    // ── Grid rings ────────────────────────────────────────────────
    for (const r of gridRings) {
      const ry = r * COS_T;
      ctx.beginPath();
      ctx.ellipse(cx, cy, r, ry, 0, 0, Math.PI * 2);
      ctx.strokeStyle = 'rgba(99,102,241,0.07)';
      ctx.lineWidth = 0.6;
      ctx.stroke();
    }

    // ── Crosshair ─────────────────────────────────────────────────
    const chLen = outerR + 10;
    ctx.strokeStyle = 'rgba(99,102,241,0.04)';
    ctx.lineWidth = 0.5;
    ctx.beginPath();
    ctx.moveTo(cx - chLen, cy); ctx.lineTo(cx + chLen, cy);
    ctx.moveTo(cx, cy - chLen * COS_T); ctx.lineTo(cx, cy + chLen * COS_T);
    ctx.stroke();

    // ── HUD arc segments (rotating partial arcs) ──────────────────
    ctx.lineWidth = 1;
    const arcR = outerR + 5;
    const arcRy = arcR * COS_T;
    for (let i = 0; i < 4; i++) {
      const startA = e * 0.15 + (i * Math.PI / 2);
      const endA = startA + 0.35;
      ctx.beginPath();
      ctx.ellipse(cx, cy, arcR, arcRy, 0, startA, endA);
      ctx.strokeStyle = `rgba(139,92,246,${0.15 - i * 0.02})`;
      ctx.stroke();
    }
    // Counter-rotating arcs
    const arcR2 = outerR + 12;
    for (let i = 0; i < 3; i++) {
      const startA = -e * 0.1 + (i * Math.PI * 2 / 3);
      const endA = startA + 0.25;
      ctx.beginPath();
      ctx.ellipse(cx, cy, arcR2, arcR2 * COS_T, 0, startA, endA);
      ctx.strokeStyle = `rgba(99,102,241,${0.10 - i * 0.02})`;
      ctx.stroke();
    }

    // ── Tick marks ────────────────────────────────────────────────
    ctx.strokeStyle = 'rgba(139,92,246,0.10)';
    ctx.lineWidth = 1;
    for (const r of gridRings) {
      for (let a = 0; a < Math.PI * 2; a += Math.PI / 8) {
        const p1 = project(r - 3, a);
        const p2 = project(r + 3, a);
        ctx.beginPath();
        ctx.moveTo(p1.x, p1.y);
        ctx.lineTo(p2.x, p2.y);
        ctx.stroke();
      }
    }

    // ── Scan sweep ────────────────────────────────────────────────
    scanAngle += 0.012;
    const scanR = outerR + 8;
    const sx = cx + Math.cos(scanAngle) * scanR;
    const sy = cy + Math.sin(scanAngle) * scanR * COS_T;

    // Main scan line
    const sg = ctx.createLinearGradient(cx, cy, sx, sy);
    sg.addColorStop(0, 'rgba(139,92,246,0.0)');
    sg.addColorStop(0.3, 'rgba(139,92,246,0.22)');
    sg.addColorStop(0.8, 'rgba(139,92,246,0.08)');
    sg.addColorStop(1, 'rgba(139,92,246,0.0)');
    ctx.strokeStyle = sg;
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.moveTo(cx, cy);
    ctx.lineTo(sx, sy);
    ctx.stroke();

    // Sweep fan trail
    for (let i = 1; i <= 20; i++) {
      const a = scanAngle - i * 0.025;
      const alpha = 0.08 * (1 - i / 20);
      const ex = cx + Math.cos(a) * scanR;
      const ey = cy + Math.sin(a) * scanR * COS_T;
      ctx.strokeStyle = `rgba(139,92,246,${alpha})`;
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(cx, cy);
      ctx.lineTo(ex, ey);
      ctx.stroke();
    }

    // ── Orbit rings (near bright, far dim) ────────────────────────
    for (const od of orbitDescs) {
      const ry = od.r * COS_T;
      // Near half
      ctx.beginPath();
      ctx.ellipse(cx, cy, od.r, ry, 0, 0.05, Math.PI - 0.05);
      ctx.lineWidth = od.isSos ? 2.5 : 1.5;
      ctx.strokeStyle = od.color;
      ctx.globalAlpha = od.isOnline ? 0.5 : 0.1;
      ctx.stroke();
      // Bloom
      ctx.lineWidth = 5;
      ctx.globalAlpha = od.isOnline ? 0.08 : 0.02;
      ctx.stroke();
      // Far half
      ctx.beginPath();
      ctx.ellipse(cx, cy, od.r, ry, 0, Math.PI + 0.05, -0.05);
      ctx.lineWidth = od.isSos ? 1.2 : 0.7;
      ctx.globalAlpha = od.isOnline ? 0.14 : 0.04;
      ctx.stroke();
      ctx.globalAlpha = 1;
    }

    // ── Energy beams (center → each satellite) ────────────────────
    for (const sat of renderList) {
      if (!sat.isOnline) continue;
      const alpha = 0.04 + (sat.depth > 0 ? 0.04 : 0);
      const bg = ctx.createLinearGradient(cx, cy, sat.x, sat.y);
      bg.addColorStop(0, `rgba(139,92,246,0)`);
      bg.addColorStop(0.3, `rgba(139,92,246,${alpha})`);
      bg.addColorStop(0.7, sat.color.includes('hsl')
        ? `rgba(139,92,246,${alpha * 0.8})`
        : sat.color.replace(')', `,${alpha * 0.8})`).replace('rgb(', 'rgba('));
      bg.addColorStop(1, 'rgba(139,92,246,0)');
      ctx.strokeStyle = bg;
      ctx.lineWidth = sat.isSos ? 1.5 : 0.8;
      ctx.beginPath();
      ctx.moveTo(cx, cy);
      ctx.lineTo(sat.x, sat.y);
      ctx.stroke();
    }

    // ── Particles ─────────────────────────────────────────────────
    for (const pt of PARTICLES) {
      const a = pt.phase + e * pt.speed;
      const p = project(pt.r, a);
      const alpha = pt.brightness * Math.min(1, 0.45 + p.depth / 140);
      if (alpha < 0.015) continue;
      ctx.fillStyle = `rgba(165,180,252,${alpha})`;
      ctx.beginPath();
      ctx.arc(p.x, p.y, pt.size * p.scale, 0, Math.PI * 2);
      ctx.fill();
    }

    // ── Trail dots ────────────────────────────────────────────────
    for (const sat of renderList) {
      if (!sat.isMoving) continue;
      for (let ti = 0; ti < sat.trail.length; ti++) {
        const tp = sat.trail[ti];
        const sz = Math.max(1.5, 3.5 * tp.scale * (0.85 - ti * 0.22));
        ctx.fillStyle = sat.color;
        ctx.globalAlpha = 0.4 - ti * 0.12;
        ctx.beginPath();
        ctx.arc(tp.x, tp.y, sz, 0, Math.PI * 2);
        ctx.fill();
      }
      ctx.globalAlpha = 1;
    }

    // ── Interactive ripples ───────────────────────────────────────
    const now = e;
    ripples = ripples.filter(r => (now - r.t) < r.duration);
    for (const rp of ripples) {
      const progress = (now - rp.t) / rp.duration;
      const radius = rp.maxR * progress;
      const opacity = 0.35 * (1 - progress);
      // Ring
      ctx.beginPath();
      ctx.arc(rp.x, rp.y, radius, 0, Math.PI * 2);
      ctx.strokeStyle = rp.color + opacity + ')';
      ctx.lineWidth = 2 * (1 - progress);
      ctx.stroke();
      // Inner fill
      if (progress < 0.3) {
        const fillGrad = ctx.createRadialGradient(rp.x, rp.y, 0, rp.x, rp.y, radius);
        fillGrad.addColorStop(0, rp.color + (opacity * 0.5) + ')');
        fillGrad.addColorStop(1, rp.color + '0)');
        ctx.fillStyle = fillGrad;
        ctx.fill();
      }
    }

    // ── Core burst effect ─────────────────────────────────────────
    if (coreBurst > 0) {
      const burstProgress = (now - coreBurstTime) / 0.6;
      if (burstProgress < 1) {
        const br = outerR * 1.2 * burstProgress;
        const bAlpha = 0.25 * (1 - burstProgress);
        // Shockwave ring
        ctx.beginPath();
        ctx.arc(cx, cy, br, 0, Math.PI * 2);
        ctx.strokeStyle = `rgba(139,92,246,${bAlpha})`;
        ctx.lineWidth = 3 * (1 - burstProgress);
        ctx.stroke();
        // Flash
        const flashGrad = ctx.createRadialGradient(cx, cy, 0, cx, cy, br * 0.6);
        flashGrad.addColorStop(0, `rgba(200,180,255,${bAlpha * 0.6})`);
        flashGrad.addColorStop(1, 'rgba(139,92,246,0)');
        ctx.fillStyle = flashGrad;
        ctx.beginPath();
        ctx.arc(cx, cy, br * 0.6, 0, Math.PI * 2);
        ctx.fill();
        // Particle burst — 12 particles flying outward
        for (let i = 0; i < 12; i++) {
          const pa = (i / 12) * Math.PI * 2;
          const pr = br * (0.3 + burstProgress * 0.7);
          const px = cx + Math.cos(pa) * pr;
          const py = cy + Math.sin(pa) * pr * COS_T;
          const pAlpha = 0.5 * (1 - burstProgress);
          ctx.fillStyle = `rgba(200,180,255,${pAlpha})`;
          ctx.beginPath();
          ctx.arc(px, py, 2 * (1 - burstProgress), 0, Math.PI * 2);
          ctx.fill();
        }
      } else {
        coreBurst = 0;
      }
    }

    // ── Empty-state ghost rings ───────────────────────────────────
    if (allMembers.length === 0) {
      ctx.setLineDash([4, 12]);
      ctx.lineWidth = 0.8;
      ctx.strokeStyle = 'rgba(255,255,255,0.05)';
      for (const r of [BASE, BASE + STEP, BASE + STEP * 2]) {
        ctx.beginPath();
        ctx.ellipse(cx, cy, r, r * COS_T, 0, 0, Math.PI * 2);
        ctx.stroke();
      }
      ctx.setLineDash([]);
    }
  }

  // ── Animation loop ────────────────────────────────────────────────────
  function tick(ts) {
    rafId = requestAnimationFrame(tick);
    if (ts - lastFrame < 32) return;
    lastFrame = ts;
    if (!t0) t0 = ts;
    elapsed = (ts - t0) / 1000;

    const items = orbitDescs.map(od => {
      const a     = od.phase + od.angSpd * elapsed;
      const proj  = project(od.r, a);
      const trail = TRAIL_LAGS.map(lag =>
        project(od.r, od.phase + od.angSpd * (elapsed - lag))
      );
      return { ...od, ...proj, a, trail };
    });
    renderList = items.sort((a, b) => a.depth - b.depth);

    if (canvas) {
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      const rect = canvas.getBoundingClientRect();
      const W = rect.width;
      const H = rect.height;
      if (canvas.width !== W * dpr || canvas.height !== H * dpr) {
        canvas.width = W * dpr;
        canvas.height = H * dpr;
      }
      const ctx = canvas.getContext('2d');
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      draw(ctx, W, H, elapsed);
    }
  }

  function initials(name) {
    return (name || '').split(' ').map(s => s[0]).join('').toUpperCase().slice(0, 2) || '?';
  }
  function locate(user) {
    focusUser.set(user.userId);
    push('/');
  }

  // ── Responsive sizing ─────────────────────────────────────────────────
  function updateSize() {
    const vw = window.innerWidth;
    const vh = window.innerHeight;
    if (vw >= 768) {
      // Desktop: fill viewport (parent CSS controls max with !important)
      const maxDim = Math.min(vw, vh) * 0.85;
      sceneSize = Math.min(700, Math.max(400, maxDim));
    } else {
      // Mobile: nearly full width
      sceneSize = allMembers.length > 0
        ? Math.min(480, vw - 16)
        : Math.min(400, vw - 24);
    }
  }

  onMount(() => {
    updateSize();
    window.addEventListener('resize', updateSize);
    let started = false;
    const observer = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting && !started) {
        started = true;
        rafId = requestAnimationFrame(tick);
      }
    }, { threshold: 0.1 });
    if (canvas) observer.observe(canvas);
    return () => observer.disconnect();
  });
  onDestroy(() => {
    if (rafId) cancelAnimationFrame(rafId);
    window.removeEventListener('resize', updateSize);
  });
</script>

<!-- svelte-ignore a11y-no-static-element-interactions -->
<div
  class="fo-scene"
  bind:this={sceneEl}
  style="width:{sceneSize}px;height:{sceneSize}px"
  on:pointerdown={handlePointer}
  aria-label="Family orbital view — tap to interact"
>
  <canvas bind:this={canvas} class="fo-canvas" aria-hidden="true"></canvas>

  <!-- Center sphere (tappable) -->
  <button
    class="fo-core"
    class:fo-core-burst={coreBurst > 0}
    on:click|stopPropagation={() => addRipple(sceneSize/2, sceneSize/2, true)}
    aria-label="{$authUser?.displayName || 'You'} — tap for pulse"
  >
    <div class="fo-core-glow" aria-hidden="true"></div>
    <div class="fo-core-ring fo-core-ring-1" aria-hidden="true"></div>
    <div class="fo-core-ring fo-core-ring-2" aria-hidden="true"></div>
    <div class="fo-core-sphere">
      <div class="fo-core-hl" aria-hidden="true"></div>
      <svg viewBox="0 0 20 20" fill="currentColor" width="20" height="20" aria-hidden="true">
        <path d="M10 1.5A6 6 0 0 0 4 7.5C4 12 10 18.5 10 18.5S16 12 16 7.5a6 6 0 0 0-6-6zm0 8.5a2.5 2.5 0 1 1 0-5 2.5 2.5 0 0 1 0 5z"/>
      </svg>
    </div>
    <div class="fo-core-pulse fo-cp-1" aria-hidden="true"></div>
    <div class="fo-core-pulse fo-cp-2" aria-hidden="true"></div>
    <div class="fo-core-pulse fo-cp-3" aria-hidden="true"></div>
  </button>

  <!-- Satellites -->
  {#each renderList as sat (sat.user.userId)}
    {@const sz = SAT_D * sat.scale}
    {@const depthOp = Math.min(1, 0.48 + sat.depth / 155)}
    <button
      class="fo-sat"
      class:fo-sat-sos={sat.isSos}
      class:fo-sat-offline={!sat.isOnline}
      style="
        left:{sat.x}px;top:{sat.y}px;
        width:{sz}px;height:{sz}px;
        --c:{sat.color};
        opacity:{sat.isOnline ? depthOp : 0.2};
        z-index:{Math.round((sat.depth + 160) * 10) + 1};
        font-size:{Math.max(8, 10 * sat.scale)}px;
      "
      on:click|stopPropagation={() => { addRipple(sat.x, sat.y); locate(sat.user); }}
      aria-label="{sat.user.displayName} — {sat.isSos ? 'SOS' : sat.isMoving ? 'Moving' : sat.isOnline ? 'Online' : 'Offline'}"
    >
      <span class="fo-init">{initials(sat.user.displayName)}</span>
      {#if sat.isSos}
        <span class="fo-sos-ring" aria-hidden="true"></span>
        <span class="fo-sos-ring fo-sos-ring-2" aria-hidden="true"></span>
      {/if}
    </button>
  {/each}

  {#if overflow > 0}
    <div class="fo-overflow">+{overflow}</div>
  {/if}

  {#if allMembers.length === 0}
    <div class="fo-empty">Scanning for family...</div>
  {/if}

  <!-- Tap hint (fades after first interaction) -->
  <div class="fo-hint" aria-hidden="true">tap to interact</div>
</div>

<!-- Tags -->
{#if renderList.length > 0}
  <div class="fo-tags" role="list">
    {#each renderList as sat (sat.user.userId)}
      <button
        class="fo-tag" class:fo-tag-sos={sat.isSos} class:fo-tag-off={!sat.isOnline}
        style="--c:{sat.color}"
        on:click={() => locate(sat.user)} role="listitem"
      >
        <span class="fo-td"></span>
        <span class="fo-tn">{sat.user.displayName}</span>
      </button>
    {/each}
  </div>
{/if}

<style>
  .fo-scene {
    position: relative;
    margin: 0 auto;
    flex-shrink: 0;
    overflow: visible;
    background: transparent;
    cursor: crosshair;
    touch-action: manipulation;
    -webkit-tap-highlight-color: transparent;
  }

  .fo-canvas {
    position: absolute;
    inset: 0;
    width: 100%; height: 100%;
    pointer-events: none;
    z-index: 1;
  }

  /* ── Center core (interactive button) ────────────────────────────── */
  .fo-core {
    position: absolute;
    left: 50%; top: 50%;
    transform: translate(-50%, -50%);
    z-index: 40;
    display: flex;
    align-items: center;
    justify-content: center;
    background: none; border: none; padding: 0;
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
    /* Large touch target */
    width: 70px; height: 70px;
  }
  .fo-core:active .fo-core-sphere {
    transform: scale(0.88);
    filter: brightness(1.4);
  }
  .fo-core-burst .fo-core-sphere {
    filter: brightness(1.6) !important;
  }

  .fo-core-glow {
    position: absolute;
    width: 130px; height: 130px;
    border-radius: 50%;
    background: radial-gradient(circle,
      rgba(99,102,241,0.2) 0%,
      rgba(99,102,241,0.06) 45%,
      transparent 70%
    );
    pointer-events: none;
    animation: breathe 4s ease-in-out infinite;
  }

  .fo-core-ring {
    position: absolute;
    border-radius: 50%;
    pointer-events: none;
    border: 1px dashed rgba(139,92,246,0.2);
  }
  .fo-core-ring-1 {
    width: 62px; height: 62px;
    animation: spin 10s linear infinite;
  }
  .fo-core-ring-2 {
    width: 78px; height: 78px;
    animation: spin 16s linear infinite reverse;
    border-color: rgba(99,102,241,0.12);
  }

  .fo-core-sphere {
    position: relative;
    width: 48px; height: 48px;
    border-radius: 50%;
    background: radial-gradient(
      circle at 34% 26%,
      rgba(255,255,255,0.92) 0%,
      #c4b5fd 11%,
      var(--primary-500, #6366f1) 28%,
      var(--primary-700, #4338ca) 52%,
      rgba(8, 4, 38, 0.92) 80%
    );
    box-shadow:
      0 0 0 1.5px rgba(255,255,255,0.18),
      0 0 16px var(--primary-400, #818cf8),
      0 0 40px color-mix(in srgb, var(--primary-500, #6366f1) 55%, transparent),
      0 0 80px color-mix(in srgb, var(--primary-600, #4f46e5) 22%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    color: rgba(255,255,255,0.92);
    z-index: 2;
    animation: breathe 4s ease-in-out infinite;
    transition: transform 0.15s ease, filter 0.15s ease;
  }
  .fo-core-sphere svg {
    filter: drop-shadow(0 0 5px rgba(255,255,255,0.7));
  }

  .fo-core-hl {
    position: absolute;
    width: 14px; height: 14px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(255,255,255,0.9) 0%, transparent 70%);
    top: 5px; left: 8px;
    filter: blur(1.5px);
    pointer-events: none;
  }

  .fo-core-pulse {
    position: absolute;
    width: 48px; height: 48px;
    border-radius: 50%;
    border: 1px solid rgba(139,92,246,0.35);
    pointer-events: none;
  }
  .fo-cp-1 { animation: pulse-out 3.5s ease-out infinite; }
  .fo-cp-2 { animation: pulse-out 3.5s ease-out infinite 1.2s; }
  .fo-cp-3 { animation: pulse-out 3.5s ease-out infinite 2.4s; }

  @keyframes breathe { 0%,100% { filter:brightness(1); } 50% { filter:brightness(1.08); } }
  @keyframes pulse-out { 0% { transform:scale(1); opacity:0.45; } 100% { transform:scale(3.2); opacity:0; } }
  @keyframes spin { from { transform:rotate(0deg); } to { transform:rotate(360deg); } }

  /* ── Satellites ──────────────────────────────────────────────────── */
  .fo-sat {
    position: absolute;
    transform: translate(-50%,-50%);
    border-radius: 50%;
    background: color-mix(in srgb, var(--c,#6366f1) 22%, rgba(5,8,22,0.88));
    border: 2px solid var(--c,#6366f1);
    box-shadow:
      0 0 10px color-mix(in srgb, var(--c,#6366f1) 45%, transparent),
      0 0 24px color-mix(in srgb, var(--c,#6366f1) 18%, transparent),
      inset 0 1px 0 rgba(255,255,255,0.14);
    display: flex;
    align-items: center; justify-content: center;
    cursor: pointer;
    will-change: left, top, width, height, opacity;
    transition: transform 0.12s ease, box-shadow 0.12s ease;
    -webkit-tap-highlight-color: transparent;
  }
  .fo-sat:hover {
    transform: translate(-50%,-50%) scale(1.22);
    box-shadow:
      0 0 14px color-mix(in srgb, var(--c,#6366f1) 60%, transparent),
      0 0 32px color-mix(in srgb, var(--c,#6366f1) 25%, transparent);
  }
  .fo-sat:active {
    transform: translate(-50%,-50%) scale(0.85);
    filter: brightness(1.3);
  }

  .fo-init {
    font-weight: 900;
    color: var(--c,#6366f1);
    line-height: 1;
    user-select: none;
    letter-spacing: -0.04em;
    font-family: var(--font-display, system-ui);
    text-shadow: 0 0 8px var(--c,#6366f1);
  }

  .fo-sat-sos { --c:#ef4444 !important; background:rgba(239,68,68,0.2) !important; border-color:#ef4444 !important; }
  .fo-sat-sos .fo-init { color:#f87171; text-shadow:0 0 10px #ef4444; }

  .fo-sos-ring {
    position:absolute; inset:-5px;
    border-radius:50%;
    border:2px solid rgba(239,68,68,0.5);
    pointer-events:none;
    animation: sos-exp 1s ease-out infinite;
  }
  .fo-sos-ring-2 { inset:-10px; border-color:rgba(239,68,68,0.25); animation-delay:0.5s; }
  @keyframes sos-exp { from{transform:scale(1);opacity:0.6;} to{transform:scale(1.6);opacity:0;} }

  .fo-sat-offline { filter:saturate(0.05) brightness(0.45); }

  /* ── Overflow / empty / hint ─────────────────────────────────────── */
  .fo-overflow {
    position:absolute; bottom:10px; right:14px;
    font-size:9px; font-weight:800;
    color:rgba(255,255,255,0.3);
    background:rgba(255,255,255,0.05);
    border:1px solid rgba(255,255,255,0.08);
    border-radius:20px; padding:2px 7px;
    z-index:100;
  }

  .fo-empty {
    position:absolute; bottom:12px; left:50%;
    transform:translateX(-50%);
    font-size:10px; font-weight:700;
    color:rgba(139,92,246,0.4);
    white-space:nowrap; pointer-events:none;
    z-index:100; letter-spacing:0.08em;
    text-transform:uppercase;
    animation: scan-pulse 2s ease-in-out infinite;
  }

  .fo-hint {
    position:absolute; bottom:12px; left:50%;
    transform:translateX(-50%);
    font-size:9px; font-weight:600;
    color:rgba(139,92,246,0.25);
    letter-spacing:0.06em;
    text-transform:uppercase;
    pointer-events:none; z-index:100;
    animation: hint-fade 4s ease-in-out forwards;
  }

  @keyframes scan-pulse { 0%,100%{opacity:0.3;} 50%{opacity:0.7;} }
  @keyframes hint-fade { 0%{opacity:0;} 20%{opacity:0.3;} 80%{opacity:0.3;} 100%{opacity:0;} }

  /* ── Tags ────────────────────────────────────────────────────────── */
  .fo-tags {
    display:flex; flex-wrap:wrap; justify-content:center;
    gap:5px; padding:6px 16px 0;
    max-width:380px; margin:0 auto;
  }
  .fo-tag {
    display:inline-flex; align-items:center; gap:4px;
    padding:4px 10px; border-radius:9999px;
    background:color-mix(in srgb, var(--c,#6366f1) 10%, rgba(5,8,18,0.65));
    border:1px solid color-mix(in srgb, var(--c,#6366f1) 25%, transparent);
    color:color-mix(in srgb, var(--c,#6366f1) 80%, rgba(255,255,255,0.9));
    font-size:11px; font-weight:700;
    cursor:pointer;
    transition:background 0.15s, box-shadow 0.15s, transform 0.12s;
    -webkit-tap-highlight-color:transparent;
    font-family:var(--font-display, system-ui);
  }
  .fo-tag:hover {
    background:color-mix(in srgb, var(--c,#6366f1) 22%, rgba(5,8,18,0.8));
    box-shadow:0 0 10px color-mix(in srgb, var(--c,#6366f1) 30%, transparent);
    transform:translateY(-1px);
  }
  .fo-tag:active { transform:scale(0.94); }
  .fo-td {
    width:5px; height:5px; border-radius:50%;
    background:var(--c,#6366f1);
    box-shadow:0 0 5px var(--c,#6366f1);
  }
  .fo-tn { max-width:70px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .fo-tag-sos { --c:#ef4444; animation:tag-blink 1.2s ease-in-out infinite; }
  .fo-tag-off { opacity:0.38; filter:saturate(0.1); }
  @keyframes tag-blink { 0%,100%{opacity:1;} 50%{opacity:0.5;} }

  @media (prefers-reduced-motion:reduce) {
    .fo-core-pulse, .fo-core-ring, .fo-core-sphere, .fo-core-glow,
    .fo-sos-ring, .fo-tag-sos, .fo-empty, .fo-hint { animation:none!important; }
  }
</style>
