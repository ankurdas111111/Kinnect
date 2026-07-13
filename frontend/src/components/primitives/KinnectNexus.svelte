<script>
  // KinnectNexus — 3D orbital brand animation
  // Pure CSS, zero JS. Theme-aware via CSS custom properties.
  // Shows in empty state when no family members are visible yet.
</script>

<div class="nexus" role="presentation" aria-hidden="true">
  <!-- Ambient glow halos -->
  <div class="aura a-far"></div>
  <div class="aura a-near"></div>

  <!-- 3D orbital gyroscope -->
  <div class="stage">
    <!-- Ring 1: equatorial — tilted near-horizontal -->
    <div class="ring r-eq">
      <span class="sat s1"></span>
    </div>

    <!-- Ring 2: meridional — tilted near-vertical -->
    <div class="ring r-me">
      <span class="sat s2"></span>
    </div>

    <!-- Ring 3: polar / diagonal -->
    <div class="ring r-po">
      <span class="sat s3"></span>
    </div>

    <!-- Kinnect world orb -->
    <div class="orb">
      <div class="orb-face"></div>
      <div class="orb-gloss"></div>
      <!-- Kinnect location-pin brand mark -->
      <div class="orb-pin">
        <svg viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
          <path d="M10 1.5A6 6 0 0 0 4 7.5C4 12 10 18.5 10 18.5S16 12 16 7.5a6 6 0 0 0-6-6zm0 8.5a2.5 2.5 0 1 1 0-5 2.5 2.5 0 0 1 0 5z"/>
        </svg>
      </div>
    </div>

    <!-- Bright nexus core -->
    <div class="core"></div>
  </div>

  <!-- Radial particle jets -->
  <div class="jets">
    <span class="jet j1"></span>
    <span class="jet j2"></span>
    <span class="jet j3"></span>
    <span class="jet j4"></span>
    <span class="jet j5"></span>
    <span class="jet j6"></span>
  </div>
</div>

<style>
  /* ── Root container ────────────────────────────────────────────────────── */
  .nexus {
    position: relative;
    width: 180px;
    height: 180px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  /* ── Ambient aura halos ────────────────────────────────────────────────── */
  .aura {
    position: absolute;
    border-radius: 50%;
    pointer-events: none;
  }
  .a-far {
    width: 180px;
    height: 180px;
    background: radial-gradient(
      circle,
      color-mix(in srgb, var(--primary-500, #14b8a6) 22%, transparent) 0%,
      transparent 68%
    );
    animation: nexus-aura 4.5s ease-in-out infinite;
  }
  .a-near {
    width: 120px;
    height: 120px;
    background: radial-gradient(
      circle,
      color-mix(in srgb, var(--primary-400, #2dd4bf) 30%, transparent) 0%,
      transparent 65%
    );
    animation: nexus-aura 3.8s ease-in-out infinite 0.7s;
  }

  /* ── 3D Stage — perspective container ─────────────────────────────────── */
  .stage {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 140px;
    height: 140px;
    margin: -70px 0 0 -70px;
    transform-style: preserve-3d;
    perspective: 580px;
  }

  /* ── Orbital rings ─────────────────────────────────────────────────────── */
  .ring {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 128px;
    height: 128px;
    margin: -64px 0 0 -64px;
    border-radius: 50%;
    transform-style: preserve-3d;
    pointer-events: none;
  }

  /* Visible ring trace */
  .ring::before {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: 50%;
    border: 1.5px solid;
    pointer-events: none;
  }

  /* Equatorial ring — near-horizontal orbital plane */
  .r-eq { animation: orbit-eq 3.4s linear infinite; }
  .r-eq::before {
    border-color: color-mix(in srgb, var(--primary-400, #2dd4bf) 55%, transparent);
    box-shadow: 0 0 10px color-mix(in srgb, var(--primary-400, #2dd4bf) 28%, transparent);
  }

  /* Meridional ring — near-vertical orbital plane */
  .r-me { animation: orbit-me 5s linear infinite reverse; }
  .r-me::before {
    border-color: color-mix(in srgb, var(--primary-300, #5eead4) 40%, transparent);
    box-shadow: 0 0 8px color-mix(in srgb, var(--primary-300, #5eead4) 20%, transparent);
  }

  /* Polar / diagonal ring */
  .r-po { animation: orbit-po 6.2s linear infinite; }
  .r-po::before {
    border-color: color-mix(in srgb, var(--primary-500, #14b8a6) 45%, transparent);
    box-shadow: 0 0 9px color-mix(in srgb, var(--primary-500, #14b8a6) 22%, transparent);
  }

  /* ── Satellite dots — sit at 12-o'clock of each ring, orbit with it ───── */
  .sat {
    position: absolute;
    border-radius: 50%;
    /* 14×14 dot centered horizontally, at the top edge = 12 o'clock */
    top: -7px;
    left: calc(50% - 7px);
    width: 14px;
    height: 14px;
  }
  .s1 {
    background: var(--primary-200, #99f6e4);
    box-shadow:
      0 0 8px var(--primary-300, #5eead4),
      0 0 20px color-mix(in srgb, var(--primary-400, #2dd4bf) 55%, transparent);
  }
  .s2 {
    background: var(--primary-300, #5eead4);
    box-shadow:
      0 0 8px var(--primary-400, #2dd4bf),
      0 0 20px color-mix(in srgb, var(--primary-500, #14b8a6) 50%, transparent);
  }
  .s3 {
    width: 11px;
    height: 11px;
    top: -5.5px;
    left: calc(50% - 5.5px);
    background: var(--primary-400, #2dd4bf);
    box-shadow:
      0 0 7px var(--primary-500, #14b8a6),
      0 0 16px color-mix(in srgb, var(--primary-500, #14b8a6) 45%, transparent);
  }

  /* ── Central orb — the Kinnect world ──────────────────────────────────── */
  .orb {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 78px;
    height: 78px;
    margin: -39px 0 0 -39px;
    border-radius: 50%;
    animation: orb-float 4.2s ease-in-out infinite;
  }

  .orb-face {
    position: absolute;
    inset: 0;
    border-radius: 50%;
    background: radial-gradient(
      circle at 36% 30%,
      rgba(255,255,255,0.28) 0%,
      var(--primary-300, #5eead4) 28%,
      var(--primary-500, #14b8a6) 56%,
      var(--primary-800, #065f46) 82%,
      rgba(0,0,0,0.28) 100%
    );
    box-shadow:
      0 0 0 1.5px rgba(255,255,255,0.14),
      0 0 16px var(--primary-500, #14b8a6),
      0 0 48px color-mix(in srgb, var(--primary-500, #14b8a6) 40%, transparent),
      0 0 90px color-mix(in srgb, var(--primary-600, #0d9488) 18%, transparent),
      inset 0 2px 8px rgba(255,255,255,0.22),
      inset 0 0 24px rgba(0,0,0,0.28);
    animation: orb-pulse 3s ease-in-out infinite;
  }

  /* Specular gloss highlight — teardrop at top-left */
  .orb-gloss {
    position: absolute;
    top: 11%;
    left: 16%;
    width: 28%;
    height: 20%;
    background: radial-gradient(ellipse at 40% 50%, rgba(255,255,255,0.54) 0%, transparent 100%);
    border-radius: 50%;
    transform: rotate(-22deg);
    pointer-events: none;
  }

  /* Kinnect location-pin brand mark */
  .orb-pin {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: rgba(255,255,255,0.90);
    filter: drop-shadow(0 0 5px rgba(255,255,255,0.65));
  }
  .orb-pin svg {
    width: 30px;
    height: 30px;
  }

  /* ── Nexus bright core point ───────────────────────────────────────────── */
  .core {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 14px;
    height: 14px;
    margin: -7px 0 0 -7px;
    border-radius: 50%;
    background: rgba(255,255,255,0.95);
    box-shadow:
      0 0 6px var(--primary-200, #99f6e4),
      0 0 16px var(--primary-400, #2dd4bf),
      0 0 36px color-mix(in srgb, var(--primary-400, #2dd4bf) 65%, transparent);
    animation: core-pulse 2.2s ease-in-out infinite;
  }

  /* ── Radial particle jets ──────────────────────────────────────────────── */
  .jets {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 0;
    height: 0;
    pointer-events: none;
  }
  .jet {
    position: absolute;
    top: 0;
    left: -1px;
    width: 2px;
    height: 40px;
    border-radius: 9999px;
    transform-origin: 1px 0;
    background: linear-gradient(to bottom, var(--primary-300, #5eead4) 0%, transparent 100%);
    opacity: 0;
    animation: jet-burst 3.2s ease-out infinite;
  }
  /* Each jet fires at a unique angle + staggered delay */
  .j1 { --jdeg:   0deg; animation-delay: 0s; }
  .j2 { --jdeg:  60deg; animation-delay: 0.53s; }
  .j3 { --jdeg: 120deg; animation-delay: 1.06s; }
  .j4 { --jdeg: 180deg; animation-delay: 1.60s; }
  .j5 { --jdeg: 240deg; animation-delay: 2.13s; }
  .j6 { --jdeg: 300deg; animation-delay: 2.66s; }

  /* ─── Keyframes ────────────────────────────────────────────────────────── */

  /* Orbital rotations — preserve base tilt angle, spin around Z */
  @keyframes orbit-eq {
    from { transform: rotateX(72deg) rotateZ(0deg); }
    to   { transform: rotateX(72deg) rotateZ(360deg); }
  }
  @keyframes orbit-me {
    from { transform: rotateY(68deg) rotateZ(0deg); }
    to   { transform: rotateY(68deg) rotateZ(360deg); }
  }
  @keyframes orbit-po {
    from { transform: rotateX(42deg) rotateY(32deg) rotateZ(0deg); }
    to   { transform: rotateX(42deg) rotateY(32deg) rotateZ(360deg); }
  }

  @keyframes orb-float {
    0%, 100% { transform: translateY(0px); }
    50%       { transform: translateY(-7px); }
  }
  @keyframes orb-pulse {
    0%, 100% { filter: brightness(1.0) saturate(1.05); }
    50%       { filter: brightness(1.18) saturate(1.25); }
  }
  @keyframes core-pulse {
    0%, 100% { opacity: 0.72; transform: scale(1.0); }
    50%       { opacity: 1.00; transform: scale(1.38); }
  }
  @keyframes nexus-aura {
    0%, 100% { opacity: 0.72; transform: scale(1.00); }
    50%       { opacity: 1.00; transform: scale(1.10); }
  }
  /* Jet shoots from center outward — scaleY grows away from transform-origin */
  @keyframes jet-burst {
    0%   { opacity: 0;    transform: rotate(var(--jdeg, 0deg)) scaleY(0.12); }
    18%  { opacity: 0.80; transform: rotate(var(--jdeg, 0deg)) scaleY(1.00); }
    100% { opacity: 0;    transform: rotate(var(--jdeg, 0deg)) scaleY(2.20); }
  }




  /* ── Daylight theme — clean teal on white ──────────────────────────────── */
  :global([data-theme="light"]) .orb-face {
    background: radial-gradient(
      circle at 36% 30%,
      rgba(255,255,255,0.52) 0%,
      #2dd4bf 28%,
      #0d9488 54%,
      #134e4a 80%,
      rgba(0,0,0,0.12) 100%
    );
    box-shadow:
      0 0 0 1.5px rgba(13,148,136,0.28),
      0 0 14px rgba(13,148,136,0.50),
      0 0 36px rgba(13,148,136,0.22),
      inset 0 2px 8px rgba(255,255,255,0.44),
      inset 0 0 16px rgba(0,0,0,0.10);
  }
  :global([data-theme="light"]) .a-far {
    background: radial-gradient(circle, rgba(13,148,136,0.18) 0%, transparent 68%);
  }
  :global([data-theme="light"]) .a-near {
    background: radial-gradient(circle, rgba(13,148,136,0.24) 0%, transparent 65%);
  }

  /* ── Respect reduced motion ────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .ring, .orb, .orb-face, .core, .aura, .jet {
      animation: none !important;
    }
  }
</style>
