<script>
  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { authUser } from '../lib/stores/auth.js';
  import { otherUsers, myLocation, tracking, focusUser } from '../lib/stores/map.js';
  import { mySosActive } from '../lib/stores/sos.js';
  import { connectivityStore } from '../lib/stores/connectivity.js';
  import { getUserColor, getUserColorLight } from '../lib/getUserColor.js';
  import { calculateDistance, formatDistance } from '../lib/tracking.js';
  import TiltCard from '../components/primitives/TiltCard.svelte';
  import FamilyOrbit from '../components/primitives/FamilyOrbit.svelte';
  import { clearHubBadge } from '../lib/stores/hubBadge.js';

  $: if (!$authUser) push('/login');

  // ── Feature freshness tracking ──────────────────────────────────────────────
  const VIS_KEYS = {
    activity:  'kinnect_vis_activity',
    replay:    'kinnect_vis_replay',
    emergency: 'kinnect_vis_emergency',
    checkins:  'kinnect_vis_checkins',
  };
  let visited = { activity: true, replay: true, emergency: true, checkins: true };

  function visitFeature(key, route) {
    if (key) {
      localStorage.setItem(VIS_KEYS[key], '1');
      visited = { ...visited, [key]: true };
    }
    push(route);
  }

  // ── Clock ──────────────────────────────────────────────────────────────────
  let now = new Date();
  let clockInterval;
  onMount(() => {
    clockInterval = setInterval(() => { now = new Date(); }, 15000);
    clearHubBadge();
    visited = Object.fromEntries(
      Object.entries(VIS_KEYS).map(([k, v]) => [k, !!localStorage.getItem(v)])
    );
  });
  onDestroy(() => clearInterval(clockInterval));

  $: timeStr = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  $: dateStr = now.toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric' });

  // ── Greeting ───────────────────────────────────────────────────────────────
  function greeting() {
    const h = now.getHours();
    if (h < 5)  return 'Up late';
    if (h < 12) return 'Good morning';
    if (h < 17) return 'Good afternoon';
    return 'Good evening';
  }
  $: firstName = ($authUser?.displayName || '').split(' ')[0] || 'there';

  // ── Members ────────────────────────────────────────────────────────────────
  $: members = Array.from($otherUsers.values());
  $: onlineCount  = members.filter(m => m.online).length;
  $: movingCount  = members.filter(m => m.online && m.speed > 1).length;
  $: sosMembers   = members.filter(m => m.sos?.active);
  $: allSafe      = sosMembers.length === 0 && !$mySosActive;

  function getInitials(name) {
    return (name || '').split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2) || '?';
  }

  function presence(user) {
    if (user.sos?.active) return 'sos';
    if (!user.online)     return 'offline';
    if (user.speed > 1)   return 'moving';
    return 'online';
  }

  function distText(user) {
    if (!user.lat || !user.lng || !$myLocation) return null;
    return formatDistance(calculateDistance(
      $myLocation.latitude, $myLocation.longitude, user.lat, user.lng
    ));
  }

  function speedKmh(user) {
    return user.speed ? (user.speed * 3.6).toFixed(0) : '0';
  }

  // ── Safety ring SVG ────────────────────────────────────────────────────────
  const R    = 38;
  const CIRC = 2 * Math.PI * R;
  $: safetyScore  = allSafe
    ? (onlineCount > 0 ? Math.min(100, 55 + onlineCount * 8) : 55)
    : Math.max(10, 30 - sosMembers.length * 20);
  $: ringOffset   = CIRC * (1 - safetyScore / 100);
  $: ringColor    = allSafe ? '#10b981' : '#ef4444';

  // ── Deterministic stars (3 size/colour classes) ───────────────────────────
  const stars = Array.from({ length: 90 }, (_, i) => {
    const a = (i * 2654435761) >>> 0;
    const b = (i * 1664525 + 1013904223) >>> 0;
    return {
      top:   (a % 1000) / 10,
      left:  (b % 1000) / 10,
      size:  0.9 + (a % 28) / 14,
      delay: (b % 60) / 10,
      dur:   2 + (a % 50) / 10,
      cls:   i % 7 === 0 ? 'fd-star-bright' : i % 4 === 0 ? 'fd-star-blue' : '',
    };
  });

  // ── Shooting star ──────────────────────────────────────────────────────────
  let shootingStarActive = false;
  let shootX = 10, shootY = 10;

  // ── Dynamic quotes ─────────────────────────────────────────────────────────
  const QUOTES = [
    { text: "Family is not an important thing. It's everything.", author: "Michael J. Fox" },
    { text: "The bond that links your true family is not blood, but respect and joy in each other's life.", author: "Richard Bach" },
    { text: "Friends are the family we choose for ourselves.", author: "Edna Buchanan" },
    { text: "A real friend walks in when the rest of the world walks out.", author: "Walter Winchell" },
    { text: "In family life, love is the oil that eases friction.", author: "Friedrich Nietzsche" },
    { text: "Where there is family, there is love that will never fade.", author: "" },
    { text: "The memories we make with our family are everything.", author: "Candace Cameron Bure" },
    { text: "Home is wherever I'm with my people.", author: "" },
    { text: "A friend is one who knows you and loves you just the same.", author: "Elbert Hubbard" },
    { text: "Family means no one gets left behind or forgotten.", author: "Lilo & Stitch" },
  ];
  let quoteIdx = Math.floor(Math.random() * QUOTES.length);
  let quoteVisible = true;
  let quoteInterval, shootInterval;

  // ── Gyroscope + scroll parallax ─────────────────────────────────────────────
  let gx = 0, gy = 0;       // normalised tilt −1…1
  let scrollY = 0;
  let scrollRef;
  let gyroActive = false;
  const noMotion = typeof window !== 'undefined'
    && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;

  function handleOrientation(e) {
    if (noMotion) return;
    const gamma = Math.max(-30, Math.min(30, e.gamma || 0));
    const beta  = Math.max(-30, Math.min(30, (e.beta || 0) - 55));
    gx = gamma / 30;
    gy = beta  / 30;
    gyroActive = true;
  }
  function handleMouseParallax(e) {
    if (gyroActive || noMotion) return;
    gx = (e.clientX / window.innerWidth  - 0.5) * 2;
    gy = (e.clientY / window.innerHeight - 0.5) * 2;
  }
  function handleScroll() {
    scrollY = scrollRef?.scrollTop || 0;
  }

  // ── Entrance animation ─────────────────────────────────────────────────────
  let mounted = false;
  onMount(() => {
    requestAnimationFrame(() => { mounted = true; });

    // Quote rotation — fade out, swap, fade in
    quoteInterval = setInterval(() => {
      quoteVisible = false;
      setTimeout(() => {
        quoteIdx = (quoteIdx + 1) % QUOTES.length;
        quoteVisible = true;
      }, 650);
    }, 8000);

    // Shooting star: fires 2s after mount, then every ~11s
    function fireStar() {
      shootX = 5 + Math.random() * 35;
      shootY = 5 + Math.random() * 40;
      shootingStarActive = true;
      setTimeout(() => { shootingStarActive = false; }, 1300);
    }
    setTimeout(fireStar, 2000);
    shootInterval = setInterval(fireStar, 11000);

    // Gyroscope (mobile)
    if (!noMotion && typeof DeviceOrientationEvent !== 'undefined') {
      window.addEventListener('deviceorientation', handleOrientation, { passive: true });
    }
  });
  onDestroy(() => {
    clearInterval(clockInterval);
    clearInterval(quoteInterval);
    clearInterval(shootInterval);
    window.removeEventListener('deviceorientation', handleOrientation);
  });
</script>

<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="fd" class:fd-mounted={mounted} bind:this={scrollRef} on:scroll={handleScroll} on:mousemove={handleMouseParallax}>

  <!-- ══ Starfield — parallax layer 1 (slowest) ═══════════════════════════ -->
  <div class="fd-stars" style="transform:translate3d({gx*-16}px,{gy*-11 + scrollY*-0.08}px,0)" aria-hidden="true">
    {#each stars as s}
      <span
        class="fd-star {s.cls}"
        style="top:{s.top}%;left:{s.left}%;width:{s.size}px;height:{s.size}px;animation-delay:{s.delay}s;animation-duration:{s.dur}s"
      ></span>
    {/each}
  </div>

  <!-- Milky Way — parallax layer 2 -->
  <div class="fd-milkyway" style="transform:translate3d({gx*-22}px,{gy*-15 + scrollY*-0.12}px,0)" aria-hidden="true"></div>

  <!-- Nebula — parallax layer 3 (deepest shift, moves most) -->
  <div class="fd-nebula-layer" style="transform:translate3d({gx*-30}px,{gy*-22 + scrollY*-0.18}px,0)" aria-hidden="true">
    <div class="fd-nebula fd-nebula-a"></div>
    <div class="fd-nebula fd-nebula-b"></div>
    <div class="fd-nebula fd-nebula-c"></div>
  </div>

  <!-- Shooting star -->
  {#if shootingStarActive}
    <span
      class="fd-shooting-star"
      style="top:{shootY}%;left:{shootX}%"
      aria-hidden="true"
    ></span>
  {/if}

  <!-- ══ Header ═════════════════════════════════════════════════════════════ -->
  <header class="fd-header">
    <button class="fd-back" on:click={() => push('/')} aria-label="Back to map">
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="15 18 9 12 15 6"/></svg>
      Map
    </button>
    <div class="fd-clock" aria-label="Current time">{timeStr}</div>
  </header>

  <!-- ══ Cosmos — greeting + orbit + stats in one unified hero ════════════ -->
  <section class="fd-cosmos" aria-label="Family orbital view">

    <!-- Top row: greeting (left) + compact safety ring (right) -->
    <div class="cosmos-top">
      <div class="cosmos-greeting">
        <div class="cosmos-greet-line">
          <span class="cosmos-greet-text">{greeting()},</span>
          <span class="cosmos-name">{firstName}</span>
        </div>
        <p class="cosmos-date">{dateStr}</p>
      </div>

      <!-- Compact safety ring — replaces the full-size card -->
      <div class="cosmos-ring" class:cosmos-ring-sos={!allSafe} aria-label="Safety score {safetyScore}">
        <svg width="52" height="52" viewBox="0 0 52 52" aria-hidden="true">
          <circle cx="26" cy="26" r="19" stroke="rgba(255,255,255,0.07)" stroke-width="5" fill="none"/>
          <circle
            cx="26" cy="26" r="19"
            stroke="{ringColor}" stroke-width="5" fill="none"
            stroke-linecap="round"
            stroke-dasharray="{2 * Math.PI * 19}"
            stroke-dashoffset="{2 * Math.PI * 19 * (1 - safetyScore / 100)}"
            transform="rotate(-90 26 26)"
            style="transition: stroke-dashoffset 1.2s cubic-bezier(0.4,0,0.2,1), stroke 0.5s ease;"
          />
          <circle
            cx="26" cy="26" r="19"
            stroke="{ringColor}" stroke-width="3" fill="none"
            stroke-linecap="round"
            stroke-dasharray="{2 * Math.PI * 19}"
            stroke-dashoffset="{2 * Math.PI * 19 * (1 - safetyScore / 100)}"
            transform="rotate(-90 26 26)"
            opacity="0.22"
            style="filter:blur(3px);transition: stroke-dashoffset 1.2s cubic-bezier(0.4,0,0.2,1), stroke 0.5s ease;"
          />
          <text x="26" y="30" text-anchor="middle" fill="white" font-size="11" font-weight="800" font-family="var(--font-display,system-ui)">{safetyScore}</text>
        </svg>
        <div class="cosmos-ring-badge" class:ring-badge-safe={allSafe} class:ring-badge-sos={!allSafe}>
          <span class="ring-badge-dot"></span>
          {allSafe ? 'All safe' : `${sosMembers.length + ($mySosActive ? 1 : 0)} SOS`}
        </div>
      </div>
    </div>

    <!-- Full-width orbit visualization -->
    <FamilyOrbit />

    <!-- Dynamic family/friendship quote -->
    <div class="cosmos-quote" class:quote-visible={quoteVisible} aria-live="polite" aria-atomic="true">
      <span class="quote-glyph" aria-hidden="true">"</span>
      <p class="quote-text">{QUOTES[quoteIdx].text}</p>
      {#if QUOTES[quoteIdx].author}
        <span class="quote-author">— {QUOTES[quoteIdx].author}</span>
      {/if}
    </div>

    <!-- Stats strip — 4 inline chips -->
    <div class="cosmos-stats">
      <div class="cosmos-stat">
        <span class="cosmos-stat-val">{onlineCount}</span>
        <span class="cosmos-stat-lbl">Online</span>
      </div>
      <div class="cosmos-stat" class:cosmos-stat-live={$tracking}>
        <span class="cosmos-stat-val">{$tracking ? 'ON' : 'OFF'}</span>
        <span class="cosmos-stat-lbl">Tracking</span>
        {#if $tracking}<span class="cosmos-live-ring" aria-hidden="true"></span>{/if}
      </div>
      <div class="cosmos-stat">
        <span class="cosmos-stat-val">{movingCount}</span>
        <span class="cosmos-stat-lbl">Moving</span>
      </div>
      <div class="cosmos-stat" class:cosmos-stat-warn={!$connectivityStore.isOnline}>
        <span class="cosmos-stat-val">{$connectivityStore.isOnline ? 'OK' : 'OFF'}</span>
        <span class="cosmos-stat-lbl">Network</span>
      </div>
    </div>
  </section>

  <!-- ══ Network ════════════════════════════════════════════════════════════ -->
  <section class="fd-section">
    <header class="fd-section-header">
      <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
      <h2>Your Network</h2>
      <span class="fd-section-badge">{members.length}</span>
    </header>

    {#if members.length === 0}
      <TiltCard>
        <div class="fd-empty">
          <div class="fd-empty-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          </div>
          <p>No one in your network yet.</p>
          <button class="fd-cta" on:click={() => push('/')}>
            Open Map to add people
          </button>
        </div>
      </TiltCard>
    {:else}
      <div class="fd-member-grid">
        {#each members as user (user.userId)}
          {@const color = getUserColor(user.userId)}
          {@const colorLight = getUserColorLight(user.userId)}
          {@const pres = presence(user)}
          {@const dist = distText(user)}
          <TiltCard intensity={13} shine={true}>
            <button
              class="fd-member"
              class:member-sos={pres === 'sos'}
              class:member-offline={pres === 'offline'}
              class:member-moving={pres === 'moving'}
              style="--mc:{color};--mcl:{colorLight}"
              on:click={() => { focusUser.set(user.userId); push('/'); }}
              aria-label="View {user.displayName || 'member'} on map"
            >
              <!-- Card glow -->
              <div class="member-glow" aria-hidden="true"></div>

              <!-- Top: avatar + status -->
              <div class="member-top">
                <div class="member-avatar-wrap">
                  <div class="member-avatar">
                    <span class="member-initials">{getInitials(user.displayName)}</span>
                  </div>
                  <!-- Presence ring -->
                  <span
                    class="presence-dot"
                    class:dot-sos={pres === 'sos'}
                    class:dot-offline={pres === 'offline'}
                    class:dot-moving={pres === 'moving'}
                    class:dot-online={pres === 'online'}
                    aria-label="Status: {pres}"
                  ></span>
                </div>

                <!-- Battery pill -->
                {#if user.batteryPct != null}
                  <div class="battery-pill" class:batt-low={user.batteryPct < 20}>
                    <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="2" y="7" width="16" height="10" rx="2"/><path d="M22 11v2"/></svg>
                    {user.batteryPct}%
                  </div>
                {/if}
              </div>

              <!-- Name + status -->
              <div class="member-body">
                <span class="member-name">{user.displayName || 'Unknown'}</span>
                <span class="member-status" class:status-sos={pres === 'sos'}>
                  {#if pres === 'sos'}
                    SOS Active
                  {:else if pres === 'moving'}
                    {speedKmh(user)} km/h
                  {:else if pres === 'offline'}
                    Offline
                  {:else}
                    Online
                  {/if}
                </span>
                {#if dist}
                  <span class="member-dist">{dist} away</span>
                {/if}
              </div>

              <!-- Locate icon -->
              <div class="member-locate" aria-hidden="true">
                <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
              </div>
            </button>
          </TiltCard>
        {/each}
      </div>
    {/if}
  </section>

  <!-- ══ Quick Actions ═════════════════════════════════════════════════════ -->
  <section class="fd-section">
    <header class="fd-section-header">
      <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
      <h2>Quick Actions</h2>
    </header>
    <div class="fd-actions">

      <TiltCard intensity={10} shine={true}>
        <button class="fd-action fd-action-map" on:click={() => visitFeature(null, '/')}>
          <div class="action-orb">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 6v16l7-4 8 4 7-4V2l-7 4-8-4-7 4z"/><line x1="8" y1="2" x2="8" y2="18"/><line x1="16" y1="6" x2="16" y2="22"/></svg>
          </div>
          <span>Live Map</span>
        </button>
      </TiltCard>

      <TiltCard intensity={10} shine={true}>
        <button class="fd-action fd-action-activity" on:click={() => visitFeature('activity', '/activity')}>
          <div class="action-orb">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
            {#if !visited.activity}<span class="action-new-dot" aria-label="Unvisited"></span>{/if}
          </div>
          <span>Activity</span>
        </button>
      </TiltCard>

      <TiltCard intensity={10} shine={true}>
        <button class="fd-action fd-action-history" on:click={() => visitFeature('replay', '/replay')}>
          <div class="action-orb">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 .49-4.61"/></svg>
            {#if !visited.replay}<span class="action-new-dot" aria-label="Unvisited"></span>{/if}
          </div>
          <span>Route History</span>
        </button>
      </TiltCard>

      <TiltCard intensity={10} shine={true}>
        <button class="fd-action fd-action-sos" class:sos-pulse={$mySosActive} on:click={() => visitFeature('emergency', '/emergency')}>
          <div class="action-orb">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
            {#if !visited.emergency}<span class="action-new-dot action-new-dot-red" aria-label="Unvisited"></span>{/if}
          </div>
          <span>Emergency</span>
        </button>
      </TiltCard>

      <TiltCard intensity={10} shine={true}>
        <button class="fd-action fd-action-safety" on:click={() => visitFeature('checkins', '/checkins')}>
          <div class="action-orb">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
            {#if !visited.checkins}<span class="action-new-dot action-new-dot-violet" aria-label="Unvisited"></span>{/if}
          </div>
          <span>Check-ins</span>
        </button>
      </TiltCard>

      <TiltCard intensity={10} shine={true}>
        <button class="fd-action fd-action-network" on:click={() => visitFeature(null, '/')}>
          <div class="action-orb">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="5" r="2"/><circle cx="5" cy="19" r="2"/><circle cx="19" cy="19" r="2"/><line x1="12" y1="7" x2="12" y2="11"/><line x1="8.5" y1="16.5" x2="12" y2="11"/><line x1="15.5" y1="16.5" x2="12" y2="11"/></svg>
          </div>
          <span>Network</span>
        </button>
      </TiltCard>

    </div>
  </section>

  <!-- Bottom padding for safe area -->
  <div style="height:calc(var(--safe-bottom,0px) + 24px)"></div>
</div>

<style>
  /* ══ Page shell ═══════════════════════════════════════════════════════════ */
  .fd {
    height: 100dvh;
    background: #050812;
    color: #fff;
    font-family: var(--font-sans, system-ui, sans-serif);
    position: relative;
    overflow-x: hidden;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    overscroll-behavior-y: contain;
    padding: 0 0 var(--safe-bottom, 0px);

    /* Mount animation */
    opacity: 0;
    transform: translateY(12px);
    transition: opacity 0.4s ease, transform 0.4s ease;
  }
  .fd.fd-mounted {
    opacity: 1;
    transform: translateY(0);
  }

  /* ══ Starfield — parallax layer ═══════════════════════════════════════════ */
  .fd-stars {
    position: fixed;
    inset: 0;
    pointer-events: none;
    z-index: 0;
    will-change: transform;
    transition: transform 0.12s ease-out;
  }
  .fd-star {
    position: absolute;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.65);
    animation: star-twinkle linear infinite;
  }
  /* Bright star class — extra glow */
  .fd-star-bright {
    background: rgba(255, 255, 255, 0.95) !important;
    box-shadow: 0 0 3px rgba(255,255,255,0.6), 0 0 6px rgba(200,200,255,0.3);
    animation: star-twinkle-bright linear infinite !important;
  }
  /* Blue-tinted star class */
  .fd-star-blue {
    background: rgba(180, 200, 255, 0.75) !important;
    animation: star-twinkle-blue linear infinite !important;
  }
  @keyframes star-twinkle {
    0%, 100% { opacity: 0.12; transform: scale(1); }
    50%       { opacity: 0.75; transform: scale(1.25); }
  }
  @keyframes star-twinkle-bright {
    0%, 100% { opacity: 0.3;  transform: scale(1); }
    40%       { opacity: 1.0; transform: scale(1.5); box-shadow: 0 0 6px rgba(255,255,255,0.8); }
    60%       { opacity: 0.9; transform: scale(1.3); }
  }
  @keyframes star-twinkle-blue {
    0%, 100% { opacity: 0.1;  transform: scale(1); }
    50%       { opacity: 0.6; transform: scale(1.2); }
  }

  /* ══ Milky Way band — parallax layer ═════════════════════════════════════ */
  .fd-milkyway {
    position: fixed;
    inset: 0;
    pointer-events: none;
    z-index: 0;
    will-change: transform;
    transition: transform 0.12s ease-out;
    background: linear-gradient(
      128deg,
      transparent 10%,
      rgba(160, 120, 255, 0.018) 28%,
      rgba(120, 160, 255, 0.028) 38%,
      rgba(180, 140, 255, 0.022) 48%,
      rgba(100, 180, 255, 0.016) 58%,
      transparent 72%
    );
  }

  /* ══ Shooting star ════════════════════════════════════════════════════════ */
  .fd-shooting-star {
    position: fixed;
    width: 130px;
    height: 1.5px;
    background: linear-gradient(90deg, transparent, rgba(255,255,255,0.95) 40%, rgba(200,220,255,0.6), transparent);
    border-radius: 2px;
    transform: rotate(28deg);
    transform-origin: left center;
    animation: shoot-across 1.2s cubic-bezier(0.15, 0, 0.75, 1) forwards;
    pointer-events: none;
    z-index: 0;
    filter: drop-shadow(0 0 2px rgba(255,255,255,0.7)) drop-shadow(0 0 6px rgba(180,200,255,0.4));
  }
  @keyframes shoot-across {
    0%   { transform: translateX(0)    translateY(0)   rotate(28deg); opacity: 0; }
    6%   { opacity: 1; }
    82%  { opacity: 0.9; }
    100% { transform: translateX(110vw) translateY(55px) rotate(28deg); opacity: 0; }
  }

  /* ══ Nebula — parallax layer (deepest shift) ═════════════════════════════ */
  .fd-nebula-layer {
    position: fixed;
    inset: 0;
    pointer-events: none;
    z-index: 0;
    will-change: transform;
    transition: transform 0.14s ease-out;
  }
  .fd-nebula {
    position: absolute;
    border-radius: 50%;
    pointer-events: none;
    filter: blur(80px);
  }
  .fd-nebula-a {
    width: 600px; height: 500px;
    top: -180px; left: -120px;
    background: radial-gradient(ellipse, rgba(99,102,241,0.20) 0%, transparent 70%);
    animation: nebula-drift 28s ease-in-out infinite;
  }
  .fd-nebula-b {
    width: 500px; height: 500px;
    bottom: -100px; right: -100px;
    background: radial-gradient(ellipse, rgba(16,185,129,0.14) 0%, transparent 70%);
    animation: nebula-drift 35s ease-in-out infinite reverse;
  }
  .fd-nebula-c {
    width: 400px; height: 400px;
    top: 40%; left: 55%;
    background: radial-gradient(ellipse, rgba(139,92,246,0.12) 0%, transparent 70%);
    animation: nebula-drift 22s ease-in-out infinite;
  }
  /* Nebula now breathes (scale pulse) + drifts */
  @keyframes nebula-drift {
    0%, 100% { transform: translate(0,    0)    scale(1.00); opacity: 1.0; }
    33%       { transform: translate(40px, 60px) scale(1.08); opacity: 0.82; }
    66%       { transform: translate(-30px,30px) scale(0.94); opacity: 0.90; }
  }

  /* ══ Header ═══════════════════════════════════════════════════════════════ */
  .fd-header {
    position: sticky;
    top: 0;
    z-index: 10;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: calc(var(--safe-top, 0px) + 12px) 20px 12px;
    background: rgba(5, 8, 18, 0.72);
    backdrop-filter: blur(20px) saturate(1.6);
    -webkit-backdrop-filter: blur(20px) saturate(1.6);
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }
  .fd-back {
    display: flex;
    align-items: center;
    gap: 4px;
    background: rgba(255,255,255,0.07);
    border: 1px solid rgba(255,255,255,0.10);
    border-radius: 20px;
    padding: 6px 12px 6px 8px;
    color: rgba(255,255,255,0.75);
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s ease, color 0.15s ease;
  }
  .fd-back:hover { background: rgba(255,255,255,0.12); color: #fff; }
  .fd-clock {
    font-size: 22px;
    font-weight: 700;
    letter-spacing: -0.03em;
    color: rgba(255,255,255,0.9);
    font-variant-numeric: tabular-nums;
    font-family: var(--font-display, system-ui, sans-serif);
  }

  /* ══ Cosmos — greeting + orbit + stats unified ════════════════════════════ */
  .fd-cosmos {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding-top: 4px;
    /* Depth entrance — fly in from scale(0.88) */
    opacity: 0;
    transform: scale(0.88) translateY(35px);
    transition: opacity 0.9s ease 0.12s, transform 0.9s cubic-bezier(0.34,1.56,0.64,1) 0.12s;
  }
  .fd-mounted .fd-cosmos {
    opacity: 1;
    transform: scale(1) translateY(0);
  }

  /* Top row: greeting left, compact safety ring right */
  .cosmos-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 18px 20px 0;
    box-sizing: border-box;
    gap: 12px;
  }
  .cosmos-greeting { flex: 1; min-width: 0; }
  .cosmos-greet-line {
    display: flex;
    align-items: baseline;
    gap: 8px;
    flex-wrap: wrap;
  }
  .cosmos-greet-text {
    font-size: clamp(1.4rem, 4.5vw, 2rem);
    font-weight: 400;
    color: rgba(255,255,255,0.50);
    font-family: var(--font-display, system-ui, sans-serif);
  }
  .cosmos-name {
    font-size: clamp(1.6rem, 5vw, 2.2rem);
    font-weight: 800;
    background: linear-gradient(135deg, #fff 30%, rgba(139,92,246,0.85) 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    letter-spacing: -0.03em;
    font-family: var(--font-display, system-ui, sans-serif);
  }
  .cosmos-date {
    margin: 3px 0 0;
    font-size: 12px;
    color: rgba(255,255,255,0.30);
    font-weight: 500;
    letter-spacing: 0.01em;
  }

  /* Compact safety ring widget */
  .cosmos-ring {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    flex-shrink: 0;
  }
  .cosmos-ring-badge {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 10px;
    font-weight: 700;
    padding: 2px 7px;
    border-radius: 20px;
    white-space: nowrap;
  }
  .ring-badge-safe {
    background: rgba(16,185,129,0.15);
    border: 1px solid rgba(16,185,129,0.3);
    color: #10b981;
  }
  .ring-badge-sos {
    background: rgba(239,68,68,0.15);
    border: 1px solid rgba(239,68,68,0.3);
    color: #f87171;
  }
  .ring-badge-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: currentColor;
    animation: dot-pulse 1.8s ease-in-out infinite;
  }
  @keyframes dot-pulse {
    0%, 100% { opacity: 1; transform: scale(1); }
    50%       { opacity: 0.4; transform: scale(0.7); }
  }

  /* Stats strip — 4 chips in one row, tight */
  .cosmos-stats {
    display: flex;
    gap: 8px;
    width: 100%;
    padding: 0 16px 4px;
    box-sizing: border-box;
  }
  .cosmos-stat {
    flex: 1;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 12px;
    padding: 9px 6px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    align-items: center;
    position: relative;
    overflow: hidden;
    box-shadow: inset 0 1px 0 rgba(255,255,255,0.05);
    /* CSS 3D float — each chip gently bobs in zero gravity */
    transform-style: preserve-3d;
    animation: stat-3d-float 8s ease-in-out infinite;
    transition: transform 0.25s ease;
  }
  .cosmos-stat:nth-child(1) { animation-delay: 0s; }
  .cosmos-stat:nth-child(2) { animation-delay: 2s; }
  .cosmos-stat:nth-child(3) { animation-delay: 4s; }
  .cosmos-stat:nth-child(4) { animation-delay: 6s; }
  @keyframes stat-3d-float {
    0%, 100% { transform: perspective(300px) rotateX(3deg)  rotateY(0deg);  }
    25%       { transform: perspective(300px) rotateX(-1deg) rotateY(5deg);  }
    50%       { transform: perspective(300px) rotateX(-4deg) rotateY(0deg);  }
    75%       { transform: perspective(300px) rotateX(-1deg) rotateY(-5deg); }
  }
  .cosmos-stat-val {
    font-size: 18px;
    font-weight: 800;
    color: #fff;
    letter-spacing: -0.04em;
    font-variant-numeric: tabular-nums;
    line-height: 1;
    font-family: var(--font-display, system-ui, sans-serif);
  }
  .cosmos-stat-lbl {
    font-size: 9px;
    font-weight: 600;
    color: rgba(255,255,255,0.32);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  .cosmos-stat-live { border-color: rgba(16,185,129,0.3); }
  .cosmos-stat-live .cosmos-stat-val { color: #10b981; }
  .cosmos-stat-warn { border-color: rgba(245,158,11,0.3); }
  .cosmos-stat-warn .cosmos-stat-val { color: #f59e0b; }
  .cosmos-live-ring {
    position: absolute;
    inset: -1px;
    border-radius: inherit;
    border: 1px solid rgba(16,185,129,0.4);
    animation: live-ring 2s ease-in-out infinite;
    pointer-events: none;
  }
  @keyframes live-ring {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0.3; }
  }

  /* ══ Section — staggered depth entrance ═══════════════════════════════════ */
  .fd-section {
    position: relative;
    z-index: 1;
    padding: 24px 20px 0;
    /* Depth entrance */
    opacity: 0;
    transform: scale(0.91) translateY(28px);
  }
  /* First section (Network) — stagger 0.35s */
  .fd-mounted .fd-section:first-of-type {
    opacity: 1;
    transform: scale(1) translateY(0);
    transition: opacity 0.8s ease 0.35s, transform 0.8s cubic-bezier(0.34,1.56,0.64,1) 0.35s;
  }
  /* Second section (Quick Actions) — stagger 0.55s */
  .fd-mounted .fd-section:last-of-type {
    opacity: 1;
    transform: scale(1) translateY(0);
    transition: opacity 0.8s ease 0.55s, transform 0.8s cubic-bezier(0.34,1.56,0.64,1) 0.55s;
  }
  .fd-section-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 14px;
    color: rgba(255,255,255,0.45);
  }
  .fd-section-header h2 {
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    margin: 0;
    flex: 1;
  }
  .fd-section-badge {
    font-size: 11px;
    font-weight: 700;
    background: rgba(99,102,241,0.2);
    border: 1px solid rgba(99,102,241,0.3);
    color: rgba(139,92,246,0.9);
    padding: 2px 8px;
    border-radius: 20px;
  }

  /* ══ Member grid ══════════════════════════════════════════════════════════ */
  .fd-member-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 10px;
  }

  .fd-member {
    position: relative;
    width: 100%;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(var(--mc, 99,102,241), 0.2);
    border-radius: 18px;
    padding: 14px 12px 12px;
    cursor: pointer;
    text-align: left;
    display: flex;
    flex-direction: column;
    gap: 10px;
    overflow: hidden;
    box-shadow: 0 4px 20px rgba(0,0,0,0.35), inset 0 1px 0 rgba(255,255,255,0.05);
    transition: border-color 0.25s ease, box-shadow 0.25s ease;
    -webkit-tap-highlight-color: transparent;
  }
  .fd-member:hover {
    border-color: rgba(255,255,255,0.18);
    box-shadow: 0 8px 32px rgba(0,0,0,0.45), 0 0 0 1px rgba(255,255,255,0.08), inset 0 1px 0 rgba(255,255,255,0.08);
  }
  .fd-member:active { transform: scale(0.97); }

  /* SOS variant */
  .fd-member.member-sos {
    border-color: rgba(239,68,68,0.4);
    animation: member-sos-pulse 1.5s ease-in-out infinite;
  }
  @keyframes member-sos-pulse {
    0%, 100% { box-shadow: 0 4px 20px rgba(0,0,0,0.35), 0 0 0 0 rgba(239,68,68,0.3); }
    50%       { box-shadow: 0 4px 20px rgba(0,0,0,0.35), 0 0 0 6px rgba(239,68,68,0); }
  }
  /* Offline variant */
  .fd-member.member-offline { opacity: 0.55; }

  /* Glow behind card */
  .member-glow {
    position: absolute;
    bottom: -20px; left: 50%;
    transform: translateX(-50%);
    width: 80px; height: 40px;
    background: var(--mc, #6366f1);
    border-radius: 50%;
    filter: blur(24px);
    opacity: 0.2;
    pointer-events: none;
  }

  /* Top: avatar + battery */
  .member-top {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
  }
  .member-avatar-wrap {
    position: relative;
    flex-shrink: 0;
  }
  .member-avatar {
    width: 46px;
    height: 46px;
    border-radius: 50%;
    background: var(--mcl, rgba(99,102,241,0.15));
    border: 2.5px solid var(--mc, #6366f1);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .member-initials {
    font-size: 16px;
    font-weight: 800;
    color: var(--mc, #6366f1);
    user-select: none;
    line-height: 1;
  }

  .presence-dot {
    position: absolute;
    bottom: 1px;
    right: 1px;
    width: 11px;
    height: 11px;
    border-radius: 50%;
    border: 2px solid #050812;
  }
  .dot-online  { background: #10b981; }
  .dot-moving  { background: #3b82f6; animation: moving-pulse 1s ease infinite; }
  .dot-offline { background: #475569; }
  .dot-sos     { background: #ef4444; animation: sos-pulse 0.8s ease infinite; }

  @keyframes moving-pulse {
    0%, 100% { transform: scale(1); }
    50%       { transform: scale(1.4); }
  }
  @keyframes sos-pulse {
    0%, 100% { transform: scale(1); opacity: 1; }
    50%       { transform: scale(1.6); opacity: 0.6; }
  }

  /* Battery */
  .battery-pill {
    display: flex;
    align-items: center;
    gap: 3px;
    font-size: 10px;
    font-weight: 700;
    color: rgba(255,255,255,0.4);
    background: rgba(255,255,255,0.06);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 20px;
    padding: 3px 6px;
  }
  .batt-low { color: #f87171; border-color: rgba(239,68,68,0.3); }

  /* Body */
  .member-body {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .member-name {
    font-size: 13px;
    font-weight: 700;
    color: #fff;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .member-status {
    font-size: 11px;
    color: rgba(255,255,255,0.45);
    font-weight: 500;
  }
  .status-sos { color: #f87171; font-weight: 700; }
  .member-dist {
    font-size: 10px;
    color: rgba(255,255,255,0.28);
    font-variant-numeric: tabular-nums;
  }

  /* Locate icon */
  .member-locate {
    position: absolute;
    bottom: 10px;
    right: 10px;
    color: rgba(255,255,255,0.18);
  }

  /* ══ Empty state ══════════════════════════════════════════════════════════ */
  .fd-empty {
    background: rgba(255,255,255,0.03);
    border: 1px dashed rgba(255,255,255,0.10);
    border-radius: 18px;
    padding: 32px 20px;
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    color: rgba(255,255,255,0.35);
    font-size: 14px;
  }
  .fd-empty-icon {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: rgba(99,102,241,0.1);
    border: 1px solid rgba(99,102,241,0.2);
    display: flex;
    align-items: center;
    justify-content: center;
    color: rgba(99,102,241,0.7);
  }
  .fd-cta {
    background: linear-gradient(135deg, #4f46e5, #7c3aed);
    color: #fff;
    border: none;
    border-radius: 12px;
    padding: 10px 18px;
    font-size: 13px;
    font-weight: 700;
    cursor: pointer;
    box-shadow: 0 4px 16px rgba(99,102,241,0.35);
    transition: transform 0.15s ease, box-shadow 0.15s ease;
  }
  .fd-cta:hover { transform: translateY(-1px); box-shadow: 0 6px 24px rgba(99,102,241,0.45); }
  .fd-cta:active { transform: scale(0.97); }

  /* ══ Quick actions ════════════════════════════════════════════════════════ */
  .fd-actions {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 10px;
  }
  @media (max-width: 360px) {
    .fd-actions { grid-template-columns: repeat(2, 1fr); }
  }

  .fd-action {
    position: relative;
    width: 100%;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 18px;
    padding: 18px 12px 14px;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    overflow: hidden;
    box-shadow: inset 0 1px 0 rgba(255,255,255,0.05);
    transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.12s ease;
    -webkit-tap-highlight-color: transparent;
    color: rgba(255,255,255,0.8);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.01em;
  }
  .fd-action:hover { border-color: rgba(255,255,255,0.15); }
  .fd-action:active { transform: scale(0.95); }

  .action-orb {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
    transition: transform 0.2s ease;
  }
  .fd-action:hover .action-orb { transform: scale(1.08) translateY(-2px); }

  /* "New feature" freshness dot — amber by default, colour variants per feature */
  .action-new-dot {
    position: absolute;
    top: -1px;
    right: -1px;
    width: 9px;
    height: 9px;
    border-radius: 50%;
    background: #f59e0b;
    border: 2px solid #050812;
    box-shadow: 0 0 6px rgba(245, 158, 11, 0.7);
    animation: dot-appear 0.3s cubic-bezier(0.34,1.56,0.64,1) both;
  }
  .action-new-dot-red    { background: #ef4444; box-shadow: 0 0 6px rgba(239,68,68,0.7); }
  .action-new-dot-violet { background: #8b5cf6; box-shadow: 0 0 6px rgba(139,92,246,0.7); }
  @keyframes dot-appear {
    from { transform: scale(0); opacity: 0; }
    to   { transform: scale(1); opacity: 1; }
  }

  /* Orb color themes */
  .fd-action-map      .action-orb { background: rgba(99,102,241,0.18);  border: 1px solid rgba(99,102,241,0.3);  color: #818cf8; }
  .fd-action-sos      .action-orb { background: rgba(239,68,68,0.15);   border: 1px solid rgba(239,68,68,0.3);   color: #f87171; }
  .fd-action-activity .action-orb { background: rgba(16,185,129,0.15);  border: 1px solid rgba(16,185,129,0.3);  color: #34d399; }
  .fd-action-history  .action-orb { background: rgba(245,158,11,0.15);  border: 1px solid rgba(245,158,11,0.3);  color: #fbbf24; }
  .fd-action-safety   .action-orb { background: rgba(6,182,212,0.15);   border: 1px solid rgba(6,182,212,0.3);   color: #22d3ee; }
  .fd-action-network  .action-orb { background: rgba(139,92,246,0.15);  border: 1px solid rgba(139,92,246,0.3);  color: #a78bfa; }

  /* Active SOS action */
  .fd-action-sos.sos-pulse {
    border-color: rgba(239,68,68,0.4);
    animation: action-sos 1.4s ease infinite;
  }
  @keyframes action-sos {
    0%, 100% { box-shadow: 0 0 0 0 rgba(239,68,68,0.25); }
    50%       { box-shadow: 0 0 0 8px rgba(239,68,68,0); }
  }

  /* ══ Dynamic quote ════════════════════════════════════════════════════════ */
  .cosmos-quote {
    position: relative;
    width: calc(100% - 40px);
    padding: 10px 20px 12px;
    margin: 0 0 6px;
    text-align: center;
    opacity: 0;
    transform: translateY(10px);
    transition: opacity 0.65s ease, transform 0.65s ease;
    pointer-events: none;
    overflow: hidden;
  }
  .cosmos-quote.quote-visible {
    opacity: 1;
    transform: translateY(0);
  }
  /* Giant decorative opening quote mark */
  .quote-glyph {
    position: absolute;
    top: -8px;
    left: 12px;
    font-size: 72px;
    line-height: 1;
    font-family: Georgia, 'Times New Roman', serif;
    font-style: italic;
    color: rgba(139, 92, 246, 0.18);
    pointer-events: none;
    user-select: none;
    letter-spacing: -0.05em;
  }
  .quote-text {
    position: relative;
    font-size: 12.5px;
    font-style: italic;
    font-family: Georgia, 'Times New Roman', serif;
    color: rgba(255, 255, 255, 0.48);
    line-height: 1.7;
    letter-spacing: 0.015em;
    margin: 0 0 5px;
    padding: 0 8px;
  }
  .quote-author {
    display: block;
    font-size: 10px;
    font-weight: 700;
    font-style: normal;
    font-family: var(--font-sans, system-ui, sans-serif);
    color: rgba(139, 92, 246, 0.6);
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  /* ══ Reduced motion ═══════════════════════════════════════════════════════ */
  @media (prefers-reduced-motion: reduce) {
    .fd-star, .fd-star-bright, .fd-star-blue,
    .fd-nebula, .fd-nebula-a, .fd-nebula-b, .fd-nebula-c,
    .fd-shooting-star, .cosmos-stat,
    .cosmos-quote { animation: none !important; transition: none !important; opacity: 1 !important; transform: none !important; }
    .fd-stars, .fd-milkyway, .fd-nebula-layer { transition: none !important; }
    .fd-cosmos, .fd-section { opacity: 1 !important; transform: none !important; transition: none !important; }
  }
</style>
