<script>
  /**
   * FamilyOrbit — Family solar system visualization.
   *
   * Each family member orbits the central Kinnect sun as a colored satellite.
   * Orbital radius = inner to outer by join-order (could be distance in future).
   * Orbital speed = activity level: SOS fastest → Moving → Online → Offline slowest.
   * Colors = getUserColor(userId) — each person's unique deterministic hue.
   * Click a satellite → focuses that member on the live map.
   */
  import { push } from 'svelte-spa-router';
  import { otherUsers, myLocation, focusUser } from '../../lib/stores/map.js';
  import { authUser } from '../../lib/stores/auth.js';
  import { getUserColor } from '../../lib/getUserColor.js';

  // ── Layout constants ──────────────────────────────────────────────────────
  const SCENE = 330;        // SVG canvas px — fills most mobile screens
  const CX    = SCENE / 2;  // scene center x
  const CY    = SCENE / 2;  // scene center y
  const SUN_R = 28;         // central sun radius
  const SAT_R = 13;         // satellite dot radius
  const BASE  = 60;         // innermost orbit radius (px from center)
  const STEP  = 18;         // px between consecutive orbits
  const MAX   = 6;          // max members shown in orbit (6th = 60+5*18=150, edge=163 < 165 ✓)

  // ── Reactive member data ──────────────────────────────────────────────────
  $: allMembers = Array.from($otherUsers.values());

  $: orbitList = allMembers.slice(0, MAX).map((user, i) => {
    const isSos     = !!user.sos?.active;
    const isOnline  = user.online !== false;
    const isMoving  = isOnline && (user.speed || 0) > 1;
    const color     = isSos ? '#ef4444' : getUserColor(user.userId);

    // Orbit size
    const r   = BASE + i * STEP;

    // Speed: SOS > moving > online > offline
    const dur = isSos ? 5 : isMoving ? 9 : isOnline ? 20 : 55;

    // Distribute members evenly around the orbit at start
    const startFrac = i / Math.max(allMembers.length, 1);
    const delay     = -(startFrac * dur); // negative = pre-advance

    return { user, color, r, dur, delay, isSos, isOnline, isMoving };
  });

  $: overflow = allMembers.length - MAX;

  // ── Helpers ────────────────────────────────────────────────────────────────
  function initials(name) {
    return (name || '').split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2) || '?';
  }

  function locateMember(user) {
    focusUser.set(user.userId);
    push('/');
  }
</script>

<div class="fo-wrap" aria-label="Family orbital view">

  <!-- ── Central SVG for orbit traces ───────────────────────────────────── -->
  <svg
    class="fo-svg"
    width={SCENE}
    height={SCENE}
    viewBox="0 0 {SCENE} {SCENE}"
    aria-hidden="true"
  >
    <!-- Orbit trace rings — one per member -->
    {#each orbitList as { color, r, isSos }}
      <circle
        cx={CX} cy={CY} r={r}
        fill="none"
        stroke={isSos ? 'rgba(239,68,68,0.25)' : color}
        stroke-width="1"
        stroke-dasharray="3 6"
        opacity={isSos ? '0.6' : '0.28'}
      />
    {/each}

    <!-- Ghost traces when nobody has joined yet -->
    {#if allMembers.length === 0}
      <circle cx={CX} cy={CY} r={BASE}          fill="none" stroke="rgba(255,255,255,0.06)" stroke-width="1" stroke-dasharray="3 7"/>
      <circle cx={CX} cy={CY} r={BASE + STEP}   fill="none" stroke="rgba(255,255,255,0.04)" stroke-width="1" stroke-dasharray="3 7"/>
      <circle cx={CX} cy={CY} r={BASE + STEP*2} fill="none" stroke="rgba(255,255,255,0.025)" stroke-width="1" stroke-dasharray="3 7"/>
    {/if}

    <!-- Connecting line from sun to each satellite origin (decorative) -->
    {#each orbitList.slice(0, 3) as { color, r, isSos }}
      <circle cx={CX} cy={CY} r={r} fill="none"
        stroke={isSos ? 'rgba(239,68,68,0.08)' : color}
        stroke-width={r / 4}
        opacity="0.04"
      />
    {/each}
  </svg>

  <!-- ── Central sun — Kinnect branded orb ─────────────────────────────── -->
  <div class="fo-sun" aria-label="{$authUser?.displayName || 'You'} — your location">
    <div class="fo-sun-face">
      <!-- Kinnect location pin -->
      <svg viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
        <path d="M10 1.5A6 6 0 0 0 4 7.5C4 12 10 18.5 10 18.5S16 12 16 7.5a6 6 0 0 0-6-6zm0 8.5a2.5 2.5 0 1 1 0-5 2.5 2.5 0 0 1 0 5z"/>
      </svg>
    </div>
    <div class="fo-sun-corona" aria-hidden="true"></div>
    <div class="fo-sun-corona fo-sun-corona-2" aria-hidden="true"></div>
  </div>

  <!-- ── Orbiting member satellites ─────────────────────────────────────── -->
  {#each orbitList as { user, color, r, dur, delay, isSos, isOnline, isMoving } (user.userId)}
    <!-- Belt: zero-size pivot at scene center. Rotation drives orbit position. -->
    <div
      class="fo-belt"
      style="--r:{r}px; --dur:{dur}s; --delay:{delay}s; --color:{color};"
      class:fo-belt-offline={!isOnline}
    >
      <!-- Satellite: offset right by r, counter-rotates to stay upright -->
      <button
        class="fo-sat"
        class:fo-sat-sos={isSos}
        class:fo-sat-offline={!isOnline}
        class:fo-sat-moving={isMoving}
        on:click={() => locateMember(user)}
        aria-label="{user.displayName} — {isSos ? 'SOS' : isMoving ? 'Moving' : isOnline ? 'Online' : 'Offline'} — tap to locate"
      >
        <span class="fo-initials">{initials(user.displayName)}</span>
        {#if isSos}
          <span class="fo-sos-ring" aria-hidden="true"></span>
        {/if}
      </button>
    </div>
  {/each}

  <!-- ── Overflow indicator when > MAX members ──────────────────────────── -->
  {#if overflow > 0}
    <div class="fo-overflow" aria-label="{overflow} more members not shown">
      +{overflow}
    </div>
  {/if}

  <!-- ── Empty state hint ───────────────────────────────────────────────── -->
  {#if allMembers.length === 0}
    <div class="fo-empty-hint" aria-live="polite">
      Waiting for family to join
    </div>
  {/if}
</div>

<!-- ── Name tags below the orbit ──────────────────────────────────────────── -->
{#if orbitList.length > 0}
  <div class="fo-tags" role="list" aria-label="Family members">
    {#each orbitList as { user, color, isSos, isOnline, isMoving } (user.userId)}
      <button
        class="fo-tag"
        class:fo-tag-sos={isSos}
        class:fo-tag-offline={!isOnline}
        style="--color:{color}"
        on:click={() => locateMember(user)}
        role="listitem"
        aria-label="{user.displayName}"
      >
        <span class="fo-tag-dot" aria-hidden="true"></span>
        <span class="fo-tag-name">{user.displayName}</span>
        {#if isMoving && !isSos}
          <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" aria-hidden="true"><polyline points="13 17 18 12 13 7"/><polyline points="6 17 11 12 6 7"/></svg>
        {/if}
      </button>
    {/each}
  </div>
{/if}

<style>
  /* ── Outer wrapper — fills available width, max 330px ───────────────── */
  .fo-wrap {
    position: relative;
    width: min(330px, calc(100vw - 20px));
    height: min(330px, calc(100vw - 20px));
    margin: 0 auto;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  /* ── SVG trace layer ─────────────────────────────────────────────────── */
  .fo-svg {
    position: absolute;
    inset: 0;
    pointer-events: none;
  }

  /* ── Central sun ─────────────────────────────────────────────────────── */
  .fo-sun {
    position: absolute;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 4;
  }

  .fo-sun-face {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: radial-gradient(
      circle at 36% 32%,
      rgba(255,255,255,0.30) 0%,
      var(--primary-300, #5eead4) 28%,
      var(--primary-500, #14b8a6) 56%,
      var(--primary-800, #065f46) 84%,
      rgba(0,0,0,0.28) 100%
    );
    box-shadow:
      0 0 0 2px rgba(255,255,255,0.14),
      0 0 18px var(--primary-500, #14b8a6),
      0 0 52px color-mix(in srgb, var(--primary-500, #14b8a6) 40%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    color: rgba(255,255,255,0.90);
    filter: drop-shadow(0 0 4px rgba(255,255,255,0.55));
    animation: sun-breathe 3.5s ease-in-out infinite;
    z-index: 4;
  }
  .fo-sun-face svg {
    width: 26px;
    height: 26px;
  }

  /* Corona rings */
  .fo-sun-corona {
    position: absolute;
    border-radius: 50%;
    border: 1.5px solid;
    border-color: color-mix(in srgb, var(--primary-400, #2dd4bf) 35%, transparent);
    pointer-events: none;
    animation: corona-expand 3.5s ease-out infinite;
  }
  .fo-sun-corona { width: 68px; height: 68px; }
  .fo-sun-corona-2 {
    width: 84px;
    height: 84px;
    animation-delay: 1.75s;
    opacity: 0.6;
  }

  /* ── Orbit belt — 0×0 pivot at scene center, spins to drive orbit ────── */
  .fo-belt {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 0;
    height: 0;
    animation: fo-spin var(--dur, 20s) linear var(--delay, 0s) infinite;
    z-index: 3;
  }
  .fo-belt-offline {
    /* offline members orbit at half-speed */
    animation-duration: calc(var(--dur, 55s) * 1.0);
  }

  /* ── Satellite ───────────────────────────────────────────────────────── */
  .fo-sat {
    /* Positioned right of the belt pivot by --r, vertically centered */
    position: absolute;
    top: -13px;                          /* half of 26px sat diameter */
    left: calc(var(--r, 54px) - 13px);  /* offset by r – half diameter */
    width: 26px;
    height: 26px;
    border-radius: 50%;
    background: color-mix(in srgb, var(--color, #6366f1) 22%, rgba(5,8,18,0.90));
    border: 2px solid var(--color, #6366f1);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    box-shadow:
      0 0 10px color-mix(in srgb, var(--color, #6366f1) 45%, transparent),
      inset 0 1px 0 rgba(255,255,255,0.12);
    /* Counter-rotate to keep satellite upright while belt spins */
    animation: fo-counter var(--dur, 20s) linear var(--delay, 0s) infinite;
    transition: box-shadow 0.2s ease, transform 0.15s ease;
    -webkit-tap-highlight-color: transparent;
  }
  .fo-sat:hover {
    box-shadow:
      0 0 18px color-mix(in srgb, var(--color, #6366f1) 75%, transparent),
      inset 0 1px 0 rgba(255,255,255,0.18);
    transform: scale(1.18);
    z-index: 10;
  }
  .fo-sat:active { transform: scale(0.90); }

  .fo-initials {
    font-size: 8px;
    font-weight: 900;
    color: var(--color, #6366f1);
    line-height: 1;
    user-select: none;
    letter-spacing: -0.03em;
    font-family: var(--font-display, system-ui, sans-serif);
  }

  /* SOS satellite — red, bright, urgent */
  .fo-sat-sos {
    border-color: #ef4444;
    background: rgba(239,68,68,0.20);
    box-shadow: 0 0 14px rgba(239,68,68,0.70), 0 0 28px rgba(239,68,68,0.30);
    animation: fo-counter var(--dur, 5s) linear var(--delay, 0s) infinite;
  }
  .fo-sat-sos .fo-initials { color: #f87171; }

  /* Expanding SOS ring */
  .fo-sos-ring {
    position: absolute;
    inset: -5px;
    border-radius: 50%;
    border: 2px solid rgba(239,68,68,0.55);
    pointer-events: none;
    animation: sos-expand 1s ease-out infinite;
  }

  /* Offline — desaturated, dim */
  .fo-sat-offline {
    opacity: 0.32;
    filter: saturate(0.15) brightness(0.7);
  }

  /* Moving — brighter glow */
  .fo-sat-moving {
    box-shadow:
      0 0 14px color-mix(in srgb, var(--color, #6366f1) 70%, transparent),
      0 0 24px color-mix(in srgb, var(--color, #6366f1) 35%, transparent),
      inset 0 1px 0 rgba(255,255,255,0.18);
  }

  /* ── Overflow count ──────────────────────────────────────────────────── */
  .fo-overflow {
    position: absolute;
    bottom: 8px;
    right: 12px;
    font-size: 10px;
    font-weight: 800;
    color: rgba(255,255,255,0.35);
    background: rgba(255,255,255,0.07);
    border: 1px solid rgba(255,255,255,0.10);
    border-radius: 20px;
    padding: 2px 7px;
    letter-spacing: 0.02em;
  }

  /* ── Empty state hint ────────────────────────────────────────────────── */
  .fo-empty-hint {
    position: absolute;
    bottom: 14px;
    left: 50%;
    transform: translateX(-50%);
    font-size: 11px;
    font-weight: 500;
    color: rgba(255,255,255,0.22);
    white-space: nowrap;
    letter-spacing: 0.04em;
    pointer-events: none;
  }

  /* ── Name tags row ───────────────────────────────────────────────────── */
  .fo-tags {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 6px;
    padding: 4px 16px 0;
    max-width: 320px;
    margin: 0 auto;
  }

  .fo-tag {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 4px 9px;
    border-radius: 9999px;
    background: color-mix(in srgb, var(--color, #6366f1) 12%, rgba(5,8,18,0.7));
    border: 1px solid color-mix(in srgb, var(--color, #6366f1) 30%, transparent);
    color: color-mix(in srgb, var(--color, #6366f1) 85%, rgba(255,255,255,0.9));
    font-size: 11px;
    font-weight: 700;
    cursor: pointer;
    letter-spacing: 0.01em;
    transition: background 0.15s ease, box-shadow 0.15s ease, transform 0.12s ease;
    -webkit-tap-highlight-color: transparent;
    font-family: var(--font-display, system-ui, sans-serif);
  }
  .fo-tag:hover {
    background: color-mix(in srgb, var(--color, #6366f1) 20%, rgba(5,8,18,0.8));
    box-shadow: 0 0 10px color-mix(in srgb, var(--color, #6366f1) 30%, transparent);
    transform: translateY(-1px);
  }
  .fo-tag:active { transform: scale(0.95); }

  .fo-tag-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--color, #6366f1);
    flex-shrink: 0;
    box-shadow: 0 0 5px var(--color, #6366f1);
  }
  .fo-tag-name {
    max-width: 70px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .fo-tag-sos {
    --color: #ef4444;
    animation: tag-sos-pulse 1.2s ease-in-out infinite;
  }
  .fo-tag-offline {
    opacity: 0.45;
    filter: saturate(0.2);
  }

  /* ── Keyframes ───────────────────────────────────────────────────────── */
  @keyframes fo-spin {
    from { transform: rotate(0deg); }
    to   { transform: rotate(360deg); }
  }
  @keyframes fo-counter {
    from { transform: rotate(0deg); }
    to   { transform: rotate(-360deg); }
  }
  @keyframes sun-breathe {
    0%, 100% { filter: brightness(1.00) saturate(1.00) drop-shadow(0 0 4px rgba(255,255,255,0.55)); }
    50%       { filter: brightness(1.18) saturate(1.20) drop-shadow(0 0 8px rgba(255,255,255,0.75)); }
  }
  @keyframes corona-expand {
    0%   { opacity: 0.55; transform: scale(1.00); }
    100% { opacity: 0;    transform: scale(1.60); }
  }
  @keyframes sos-expand {
    from { transform: scale(1);   opacity: 0.80; }
    to   { transform: scale(1.9); opacity: 0; }
  }
  @keyframes tag-sos-pulse {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0.55; }
  }

  /* ── Reduced motion ──────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .fo-belt, .fo-sat, .fo-sun-face, .fo-sun-corona, .fo-sos-ring {
      animation: none !important;
    }
  }
</style>
