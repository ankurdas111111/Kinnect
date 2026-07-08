<script>
  
  /**
   * @typedef {Object} Props
   * @property {string} [variant] - AnimatedMeshBackground — aurora orbs + spatial grid + floating particles.
Replaces manual decorative divs duplicated in Login and Register.
Props:
variant  — 'brand' (teal, default) | 'neutral' (monochrome)
grid     — render spatial depth grid overlay (default true)
particles — render floating ambient particle dots (default true)
Bug fixes applied:
- Added missing `variant` prop (Login/Register were passing variant="brand"
to an undeclared prop, causing a Svelte console warning)
- Fixed orb-3: removed base transform:translateX(-50%) which conflicted
with keyframe starting at translate(0,0), causing a snap on animation
start. Now uses left:calc(50% - 150px) for centering instead.
   * @property {boolean} [grid]
   * @property {boolean} [particles]
   */

  /** @type {Props} */
  let { variant = 'brand', grid = true, particles = true } = $props();
</script>

<!-- Animated gradient blob -->
<div class="amb-bg" aria-hidden="true"><div class="amb-blob"></div></div>

<!-- Aurora orbs — slow-drifting atmospheric glows -->
<div class="amb-orb amb-orb-1" aria-hidden="true"></div>
<div class="amb-orb amb-orb-2" aria-hidden="true"></div>
<div class="amb-orb amb-orb-3" aria-hidden="true"></div>

<!-- Spatial depth grid -->
{#if grid}
  <div class="amb-grid" aria-hidden="true"></div>
{/if}

<!-- Floating ambient particles -->
{#if particles}
  <div class="amb-particles" aria-hidden="true">
    <span class="amb-p large"></span>
    <span class="amb-p"></span>
    <span class="amb-p"></span>
    <span class="amb-p large"></span>
    <span class="amb-p"></span>
    <span class="amb-p"></span>
    <span class="amb-p large"></span>
    <span class="amb-p"></span>
    <span class="amb-p"></span>
    <span class="amb-p large"></span>
    <span class="amb-p"></span>
    <span class="amb-p"></span>
  </div>
{/if}

<style>
  /* ── Gradient blob ──────────────────────────────────────────────────── */
  .amb-bg {
    position: fixed;
    inset: 0;
    z-index: 0;
    pointer-events: none;
    overflow: hidden;
  }

  .amb-blob {
    position: absolute;
    width: 700px;
    height: 700px;
    border-radius: 50%;
    background: radial-gradient(
      circle at center,
      rgba(20, 184, 166, 0.20) 0%,
      rgba(13, 148, 136, 0.14) 35%,
      rgba(16, 185, 129, 0.07) 60%,
      transparent 75%
    );
    top: -200px;
    left: -150px;
    animation: amb-blob-drift 22s ease-in-out infinite;
    filter: blur(60px);
    will-change: transform;
  }

  .amb-blob::after {
    content: '';
    position: absolute;
    width: 500px;
    height: 500px;
    border-radius: 50%;
    background: radial-gradient(
      circle at center,
      rgba(20, 184, 166, 0.08) 0%,
      rgba(13, 148, 136, 0.06) 40%,
      transparent 70%
    );
    bottom: -100px;
    right: -100px;
    animation: amb-blob-drift 28s ease-in-out infinite reverse;
    filter: blur(50px);
    will-change: transform;
  }

  /* ── Aurora orbs ────────────────────────────────────────────────────── */
  .amb-orb {
    position: fixed;
    border-radius: 50%;
    pointer-events: none;
    filter: blur(80px);
    will-change: transform;
    z-index: 0;
  }

  .amb-orb-1 {
    width: 500px; height: 500px;
    background: radial-gradient(circle, rgba(20, 184, 166, 0.22) 0%, transparent 65%);
    top: -120px; right: -100px;
    animation: amb-drift-3 20s ease-in-out infinite;
  }

  .amb-orb-2 {
    width: 400px; height: 400px;
    background: radial-gradient(circle, rgba(16, 185, 129, 0.16) 0%, transparent 65%);
    bottom: 10%; left: -80px;
    animation: amb-drift-2 26s ease-in-out infinite reverse;
  }

  /* Bug fix: was using transform:translateX(-50%) as base style which conflicted
     with keyframes starting at translate(0,0) — caused a visible snap at t=0.
     Fixed: center using left:calc(50% - 150px) (half of 300px width) instead. */
  .amb-orb-3 {
    width: 300px; height: 300px;
    background: radial-gradient(circle, rgba(6, 182, 212, 0.12) 0%, transparent 65%);
    top: 40%;
    left: calc(50% - 150px);
    animation: amb-drift-1 32s ease-in-out infinite;
  }

  /* ── Spatial grid ───────────────────────────────────────────────────── */
  .amb-grid {
    position: fixed;
    inset: 0;
    z-index: 0;
    pointer-events: none;
    background-image:
      linear-gradient(0deg, rgba(20, 184, 166, 0.025) 1px, transparent 1px),
      linear-gradient(90deg, rgba(20, 184, 166, 0.025) 1px, transparent 1px);
    background-size: 60px 60px;
    opacity: 0.6;
    mask-image: radial-gradient(ellipse 80% 80% at 50% 50%, black 30%, transparent 100%);
    -webkit-mask-image: radial-gradient(ellipse 80% 80% at 50% 50%, black 30%, transparent 100%);
  }

  /* ── Ambient particles ──────────────────────────────────────────────── */
  .amb-particles {
    position: fixed;
    inset: 0;
    z-index: 0;
    pointer-events: none;
    overflow: hidden;
  }

  .amb-p {
    position: absolute;
    width: 2px; height: 2px;
    border-radius: 50%;
    background: rgba(20, 184, 166, 0.5);
    box-shadow: 0 0 6px rgba(20, 184, 166, 0.4);
    animation: amb-particle-float linear infinite;
    will-change: transform, opacity;
  }

  .amb-p:nth-child(1)  { left: 8%;  top: 20%; animation-duration: 6s;  animation-delay: -1s; }
  .amb-p:nth-child(2)  { left: 18%; top: 70%; animation-duration: 8s;  animation-delay: -3s; opacity: 0.3; }
  .amb-p:nth-child(3)  { left: 30%; top: 15%; animation-duration: 7s;  animation-delay: -5s; }
  .amb-p:nth-child(4)  { left: 45%; top: 85%; animation-duration: 9s;  animation-delay: -2s; opacity: 0.25; }
  .amb-p:nth-child(5)  { left: 60%; top: 30%; animation-duration: 5s;  animation-delay: -4s; }
  .amb-p:nth-child(6)  { left: 75%; top: 60%; animation-duration: 10s; animation-delay: -1s; opacity: 0.2; }
  .amb-p:nth-child(7)  { left: 85%; top: 45%; animation-duration: 7s;  animation-delay: -6s; }
  .amb-p:nth-child(8)  { left: 90%; top: 10%; animation-duration: 6s;  animation-delay: -3s; opacity: 0.35; }
  .amb-p:nth-child(9)  { left: 12%; top: 50%; animation-duration: 8s;  animation-delay: -7s; }
  .amb-p:nth-child(10) { left: 55%; top: 75%; animation-duration: 9s;  animation-delay: -2s; opacity: 0.28; }
  .amb-p:nth-child(11) { left: 38%; top: 40%; animation-duration: 11s; animation-delay: -5s; }
  .amb-p:nth-child(12) { left: 70%; top: 25%; animation-duration: 7s;  animation-delay: -8s; opacity: 0.22; }

  .amb-p.large {
    width: 4px; height: 4px;
    background: rgba(45, 212, 191, 0.4);
    box-shadow: 0 0 10px rgba(45, 212, 191, 0.5), 0 0 20px rgba(20, 184, 166, 0.25);
  }

  /* ── Keyframes ──────────────────────────────────────────────────────── */
  @keyframes amb-blob-drift {
    0%, 100% { transform: translate(0, 0) scale(1); }
    25%       { transform: translate(80px, 60px) scale(1.08); }
    50%       { transform: translate(40px, 120px) scale(0.96); }
    75%       { transform: translate(-40px, 40px) scale(1.04); }
  }

  @keyframes amb-drift-1 {
    0%, 100% { transform: translate(0, 0) scale(1); }
    20%       { transform: translate(60px, -40px) scale(1.06); }
    40%       { transform: translate(30px, 80px) scale(0.97); }
    60%       { transform: translate(-50px, 30px) scale(1.04); }
    80%       { transform: translate(20px, -20px) scale(1.02); }
  }

  @keyframes amb-drift-2 {
    0%, 100% { transform: translate(0, 0) scale(1); }
    25%       { transform: translate(-70px, 50px) scale(1.08); }
    50%       { transform: translate(-20px, 100px) scale(0.94); }
    75%       { transform: translate(60px, -30px) scale(1.05); }
  }

  @keyframes amb-drift-3 {
    0%, 100% { transform: translate(0, 0) scale(1); }
    33%       { transform: translate(40px, -60px) scale(1.10); }
    66%       { transform: translate(-30px, 40px) scale(0.96); }
  }

  @keyframes amb-particle-float {
    0%, 100% {
      transform: translateY(0) translateX(0) scale(1);
      opacity: 0.25;
    }
    33% {
      transform: translateY(-12px) translateX(6px) scale(1.2);
      opacity: 0.4;
    }
    66% {
      transform: translateY(8px) translateX(-4px) scale(0.9);
      opacity: 0.15;
    }
  }

  /* ── Reduced motion ─────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .amb-blob,
    .amb-blob::after,
    .amb-orb-1,
    .amb-orb-2,
    .amb-orb-3,
    .amb-p {
      animation: none;
      will-change: auto;
    }
    .amb-p { opacity: 0.12; }
  }
</style>
