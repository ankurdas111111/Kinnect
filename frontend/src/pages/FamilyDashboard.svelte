<script>
  import { run } from 'svelte/legacy';

  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { authUser } from '../lib/stores/auth.js';
  import { otherUsers, myLocation, tracking, focusUser } from '../lib/stores/map.js';
  import { mySosActive } from '../lib/stores/sos.js';
  import { connectivityStore } from '../lib/stores/connectivity.js';
  import { getUserColor, getUserColorLight } from '../lib/getUserColor.js';
  import { calculateDistance, formatDistance } from '../lib/tracking.js';
  import FamilyOrbit from '../components/primitives/FamilyOrbit.svelte';
  import GlobeCanvas from '../components/primitives/GlobeCanvas.svelte';
  import GhostConstellation from '../components/primitives/GhostConstellation.svelte';
  import { allowWebGL, allowMotion } from '../lib/stores/effects.js';
  import { clearHubBadge } from '../lib/stores/hubBadge.js';
  import { prefersCoarsePointer } from '../lib/deviceCapability.js';
  import AvatarRing from '../components/primitives/AvatarRing.svelte';

  run(() => {
    if (!$authUser) push('/login');
  });

  const VIS_KEYS = {
    activity: 'kinnect_vis_activity', replay: 'kinnect_vis_replay',
    emergency: 'kinnect_vis_emergency', checkins: 'kinnect_vis_checkins',
  };
  let visited = $state({ activity: true, replay: true, emergency: true, checkins: true });

  function visitFeature(key, route) {
    if (key) { localStorage.setItem(VIS_KEYS[key], '1'); visited = { ...visited, [key]: true }; }
    push(route);
  }

  let now = $state(new Date());
  let clockInterval;
  onMount(() => {
    clockInterval = setInterval(() => { now = new Date(); }, 15000);
    clearHubBadge();
    visited = Object.fromEntries(Object.entries(VIS_KEYS).map(([k, v]) => [k, !!localStorage.getItem(v)]));
  });
  onDestroy(() => clearInterval(clockInterval));

  let timeStr = $derived(now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }));
  let dateStr = $derived(now.toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric' }));

  function greeting() {
    const h = now.getHours();
    if (h < 5) return 'Up late'; if (h < 12) return 'Good morning';
    if (h < 17) return 'Good afternoon'; return 'Good evening';
  }
  let firstName = $derived(($authUser?.displayName || '').split(' ')[0] || 'there');

  let members = $derived(Array.from($otherUsers.values()));
  let onlineCount = $derived(members.filter(m => m.online).length);
  let movingCount = $derived(members.filter(m => m.online && m.speed > 1).length);
  let sosMembers  = $derived(members.filter(m => m.sos?.active));
  let allSafe     = $derived(sosMembers.length === 0 && !$mySosActive);

  function getInitials(n) { return (n||'').split(' ').map(s=>s[0]).join('').toUpperCase().slice(0,2)||'?'; }
  function presence(u) { if(u.sos?.active) return 'sos'; if(!u.online) return 'offline'; if(u.speed>1) return 'moving'; return 'online'; }
  function presenceLabel(u) {
    if (u.sos?.active) return 'SOS active';
    if (!u.online) return 'offline';
    if (u.speed > 1) return 'moving at ' + (u.speed * 3.6).toFixed(0) + ' km/h';
    return 'online';
  }
  function distText(u) { if(!u.lat||!u.lng||!$myLocation) return null; return formatDistance(calculateDistance($myLocation.latitude,$myLocation.longitude,u.lat,u.lng)); }
  function speedKmh(u) { return u.speed ? (u.speed*3.6).toFixed(0) : '0'; }

  const SR = 21; const SC = 2 * Math.PI * SR;
  let safetyScore = $derived(allSafe ? (onlineCount > 0 ? Math.min(100, 55 + onlineCount * 8) : 55) : Math.max(10, 30 - sosMembers.length * 20));
  let ringOffset = $derived(SC * (1 - safetyScore / 100));
  let ringColor = $derived(allSafe ? 'var(--success-500)' : 'var(--danger-500)');
  let alertCount = $derived(sosMembers.length + ($mySosActive ? 1 : 0));

  // Curated quotes — family-safety context, not generic motivational platitudes
  const QUOTES = [
    { text: "Knowing your people are safe lets you sleep at night.", author: "Kinnect" },
    { text: "The greatest protection any person can have is a loving family.", author: "Anonymous" },
    { text: "Family means no one gets left behind or forgotten.", author: "Lilo & Stitch" },
    { text: "Families are the compass that guides us.", author: "Brad Henry" },
    { text: "Peace of mind comes when you know your family is okay.", author: "Kinnect" },
    { text: "Alone we can do so little; together we can do so much.", author: "Helen Keller" },
    { text: "The strength of a family lies in its loyalty to each other.", author: "Mario Puzo" },
    { text: "Being safe is the foundation upon which everything else is built.", author: "Kinnect" },
  ];
  // ── Quote cycle timing (named constant — no magic numbers) ───────────────
  const QUOTE_CYCLE_MS = 10_000;
  const QUOTE_FADE_MS  = 400; // must match .d-quote transition duration

  let quoteIdx = $state(Math.floor(Math.random() * QUOTES.length));
  let quoteVisible = $state(true);
  let quoteInterval;

  let mounted = $state(false);
  function cycleQuote() {
    quoteVisible = false;
    setTimeout(() => { quoteIdx = (quoteIdx + 1) % QUOTES.length; quoteVisible = true; }, QUOTE_FADE_MS);
  }

  // Mouse glow tracking — only wired on non-coarse-pointer devices
  const isCoarsePointer = typeof window !== 'undefined' ? prefersCoarsePointer() : true;
  let mouseX = $state(0), mouseY = $state(0);
  function handleMouseMove(e) { mouseX = e.clientX; mouseY = e.clientY; mouseOnDash = true; }
  let mouseOnDash = $state(false);

  // ── Globe sizing — named scale constants next to the resize logic ────────
  //   MOBILE_SCALE  : globe width as fraction of viewport width (mobile)
  //   VH_SCALE      : globe height cap as fraction of viewport height (desktop)
  //   LEFT_W_SCALE  : globe width cap as fraction of available left column (desktop)
  const GLOBE_MOBILE_SCALE  = 0.62;
  const GLOBE_VH_SCALE      = 0.54;
  const GLOBE_LEFT_W_SCALE  = 0.52;

  let globeSize = $state(400);
  function updateGlobeSize() {
    if (typeof window === 'undefined') return;
    const vw = window.innerWidth;
    const vh = window.innerHeight;
    if (vw < 768) {
      // Mobile: globe inline in content, fits without dominating the viewport
      globeSize = Math.round(Math.min(260, Math.max(200, vw * GLOBE_MOBILE_SCALE)));
    } else {
      const sidebarW = vw >= 1200 ? Math.min(460, vw * 0.35) : Math.min(420, vw * 0.38);
      const leftW = Math.max(400, vw - sidebarW);
      globeSize = Math.round(Math.min(520, Math.max(320, Math.min(vh * GLOBE_VH_SCALE, leftW * GLOBE_LEFT_W_SCALE))));
    }
  }

  onMount(() => {
    requestAnimationFrame(() => { mounted = true; });
    quoteInterval = setInterval(cycleQuote, QUOTE_CYCLE_MS);
    updateGlobeSize();
    window.addEventListener('resize', updateGlobeSize);
  });
  onDestroy(() => {
    clearInterval(clockInterval);
    clearInterval(quoteInterval);
    window.removeEventListener('resize', updateGlobeSize);
  });
</script>

<div class="d" class:d-ready={mounted}
  onmousemove={!isCoarsePointer ? handleMouseMove : undefined}
  onmouseleave={!isCoarsePointer ? () => mouseOnDash = false : undefined}>
  <div class="d-aurora fx-ambient" aria-hidden="true"></div>
  <div class="d-noise" aria-hidden="true"></div>

  <!-- Mouse cursor glow — fine pointer (mouse/trackpad) only, absent on touch -->
  {#if !isCoarsePointer}
    <div class="d-cursor-glow" class:d-cursor-visible={mouseOnDash && $allowMotion}
      style="left:{mouseX}px;top:{mouseY}px" aria-hidden="true"></div>
  {/if}

  <!-- FULL-SCREEN ORBIT BACKGROUND -->
  <div class="d-orbit-bg" aria-hidden="true">
    <FamilyOrbit />
  </div>

  <!-- LEFT COLUMN — desktop: centered globe + info + quote -->
  <div class="d-left-hud" aria-hidden="false">
    <div class="d-globe-col">

      <!-- TOP: eyebrow + live status pills -->
      <div class="d-globe-top">
        <div class="d-globe-eyebrow">
          <span class="d-globe-blip"></span>
          FAMILY NETWORK
        </div>
        <div class="d-globe-status-row">
          <span class="d-gsr-pill" class:gsr-on={$tracking}>
            <span class="d-gsr-dot"></span>
            Tracking {$tracking ? 'ON' : 'OFF'}
          </span>
          <span class="d-gsr-pill" class:gsr-on={onlineCount > 0}>
            <span class="d-gsr-dot"></span>
            {onlineCount} online
          </span>
          <span class="d-gsr-pill" class:gsr-safe={allSafe} class:gsr-alert={!allSafe}>
            <span class="d-gsr-dot"></span>
            {allSafe ? 'All safe' : `${alertCount} alert`}
          </span>
          <span class="d-gsr-pill" class:gsr-on={$connectivityStore.isOnline} class:gsr-warn={!$connectivityStore.isOnline}>
            <span class="d-gsr-dot"></span>
            {$connectivityStore.isOnline ? 'Connected' : 'Offline'}
          </span>
        </div>
      </div>

      <!-- CENTER: Globe with flanking stat cards on large screens -->
      <div class="d-globe-center">
        <!-- Left: Network card -->
        <div class="d-globe-side d-globe-side-l">
          <div class="d-side-card">
            <div class="d-side-label">Network</div>
            <div class="d-side-big">{members.length}</div>
            <div class="d-side-sub">Members</div>
            <div class="d-side-divider"></div>
            <div class="d-side-row">
              <span class="d-sd dot-on"></span><span>{onlineCount} online</span>
            </div>
            <div class="d-side-row">
              <span class="d-sd dot-mv"></span><span>{movingCount} moving</span>
            </div>
            <div class="d-side-row">
              <span class="d-sd d-sd-muted"></span>
              <span>{members.length - onlineCount} offline</span>
            </div>
          </div>
        </div>

        {#if $allowWebGL}
          <GlobeCanvas size={globeSize} />
        {:else}
          <!-- Calm / low-end / minimal: skip WebGL, show a static ambient globe -->
          <div class="d-globe-fallback fx-ambient" style="width:{globeSize}px;height:{globeSize}px" aria-hidden="true">
            <span class="d-gf-glyph">🌍</span>
          </div>
        {/if}

        <!-- Right: Safety card -->
        <div class="d-globe-side d-globe-side-r">
          <div class="d-side-card">
            <div class="d-side-label">Safety</div>
            <div class="d-side-big" style="color:{ringColor}">{safetyScore}</div>
            <div class="d-side-sub">Score</div>
            <div class="d-side-divider"></div>
            <div class="d-side-row" class:row-safe={allSafe} class:row-sos={!allSafe}>
              <span class="d-sd" style="background:{ringColor}"></span>
              <span>{allSafe ? 'All safe' : `${sosMembers.length} SOS`}</span>
            </div>
            <div class="d-side-row">
              <span class="d-sd" class:dot-on={$connectivityStore.isOnline} class:d-sd-warn={!$connectivityStore.isOnline}></span>
              <span>{$connectivityStore.isOnline ? 'Connected' : 'Offline'}</span>
            </div>
            <div class="d-side-row">
              <span class="d-sd" class:dot-mv={$tracking} class:d-sd-muted={!$tracking}></span>
              <span>Tracking {$tracking ? 'on' : 'off'}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- BOTTOM: member ring + location + quote -->
      <div class="d-globe-bottom">

        <!-- Interactive member avatars — click to focus on map -->
        {#if members.length > 0}
          <div class="d-member-ring" role="group" aria-label="Family members">
            {#each members.slice(0, 7) as user (user.userId)}
              {@const color = getUserColor(user.userId)}
              {@const pres = presence(user)}
              {@const ringState = pres === 'sos' ? 'sos' : (pres === 'offline' ? 'offline' : 'live')}
              <button
                class="d-mr-bubble tactile"
                style="--mc:{color}"
                onclick={() => { focusUser.set(user.userId); push('/'); }}
                aria-label="{user.displayName} — {presenceLabel(user)}, view on map"
              >
                <AvatarRing ring={ringState} size={44}>
                  <span class="d-mr-init">{getInitials(user.displayName)}</span>
                </AvatarRing>
              </button>
            {/each}
            {#if members.length > 7}
              <div class="d-mr-more" aria-hidden="true">+{members.length - 7}</div>
            {/if}
          </div>
        {/if}

        {#if $myLocation?.latitude != null}
          <div class="d-loc-info">
            <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/></svg>
            <span class="d-loc-you">You are here</span>
            <span class="d-loc-coords">
              {Math.abs($myLocation.latitude).toFixed(3)}°{$myLocation.latitude >= 0 ? 'N' : 'S'}
              &nbsp;·&nbsp;
              {Math.abs($myLocation.longitude).toFixed(3)}°{$myLocation.longitude >= 0 ? 'E' : 'W'}
            </span>
          </div>
        {:else}
          <div class="d-loc-info d-loc-unknown">
            <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
            Location not shared
          </div>
        {/if}

        <button
          class="d-quote-globe"
          class:dqg-on={quoteVisible}
          onclick={cycleQuote}
          aria-label="Cycle quote"
        >
          <span class="d-qg-mark" aria-hidden="true">"</span>
          <p class="d-qg-text">{QUOTES[quoteIdx].text}"</p>
          <span class="d-qg-author">— {QUOTES[quoteIdx].author}</span>
          <span class="d-qg-cycle" aria-hidden="true">↻ next</span>
        </button>
      </div>

    </div>
  </div>

  <!-- HEADER — glass, floating -->
  <header class="d-header">
    <button class="d-back tactile" onclick={() => push('/')} aria-label="Back to map">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="15 18 9 12 15 6"/></svg>
      Map
    </button>
    <div class="d-clock">{timeStr}</div>
  </header>

  <!-- CONTENT OVERLAY — scrollable panels floating over the orbit -->
  <div class="d-content">

    <!-- Hero: greeting + safety -->
    <section class="d-hero">
      <div class="d-greet">
        <h1 class="d-name">
          <span class="d-greet-word">{greeting()},&nbsp;</span><span class="d-name-word">{firstName}</span>
        </h1>
        <p class="d-date">{dateStr}</p>
      </div>
      <div class="d-safety" aria-label="Safety score {safetyScore}">
        <svg width="56" height="56" viewBox="0 0 56 56" aria-hidden="true">
          <circle cx="28" cy="28" r="{SR}" stroke="rgba(255,255,255,0.06)" stroke-width="4" fill="none"/>
          <circle cx="28" cy="28" r="{SR}" stroke="{ringColor}" stroke-width="4" fill="none"
            stroke-linecap="round" stroke-dasharray="{SC}" stroke-dashoffset="{ringOffset}"
            transform="rotate(-90 28 28)" class="d-ring-progress"/>
        </svg>
        <span class="d-safety-score">{safetyScore}</span>
        <div class="d-safety-badge" class:badge-safe={allSafe} class:badge-sos={!allSafe}>
          <span class="d-badge-dot"></span>
          {allSafe ? 'All safe' : `${alertCount} SOS`}
        </div>
      </div>
    </section>

    <!-- Mobile Globe — inline in scroll flow, hidden on desktop (desktop uses d-left-hud) -->
    <div class="d-mobile-globe">
      {#if $allowWebGL}
        <GlobeCanvas size={globeSize} />
      {:else}
        <div class="d-globe-fallback fx-ambient" style="width:{globeSize}px;height:{globeSize}px" aria-hidden="true">
          <span class="d-gf-glyph">🌍</span>
        </div>
      {/if}
      {#if $myLocation?.latitude != null}
        <div class="d-mob-coords">
          <span class="d-mob-you">You are here</span>
          <span class="d-mob-ll">
            {Math.abs($myLocation.latitude).toFixed(3)}°{$myLocation.latitude >= 0 ? 'N' : 'S'}
            &nbsp;·&nbsp;
            {Math.abs($myLocation.longitude).toFixed(3)}°{$myLocation.longitude >= 0 ? 'E' : 'W'}
          </span>
        </div>
      {/if}
    </div>

    <!-- Quote (centered, over orbit) -->
    <div class="d-quote" class:quote-on={quoteVisible} aria-live="polite">
      <p class="d-quote-text">{QUOTES[quoteIdx].text}</p>
      <span class="d-quote-author">— {QUOTES[quoteIdx].author}</span>
    </div>

    <!-- Stats — bento: large safety focal tile + smaller stat tiles -->
    <section class="d-stats bento-grid">
      <div class="d-stat-hero bento-col-2 bento-row-2" class:hero-safe={allSafe} class:hero-alert={!allSafe}>
        <span class="d-sh-label">Safety Score</span>
        <span class="d-sh-val" style="color:{ringColor}">{safetyScore}</span>
        <span class="d-sh-badge" class:badge-safe={allSafe} class:badge-sos={!allSafe}>
          <span class="d-badge-dot"></span>
          {allSafe ? 'All safe' : `${alertCount} SOS`}
        </span>
      </div>
      <div class="d-stat">
        <span class="d-stat-val">{onlineCount}</span>
        <span class="d-stat-lbl">Online</span>
      </div>
      <div class="d-stat" class:stat-active={$tracking}>
        <span class="d-stat-val">{$tracking ? 'ON' : 'OFF'}</span>
        <span class="d-stat-lbl">Tracking</span>
        {#if $tracking}<span class="d-stat-dot" aria-hidden="true"></span>{/if}
      </div>
      <div class="d-stat">
        <span class="d-stat-val">{movingCount}</span>
        <span class="d-stat-lbl">Moving</span>
      </div>
      <div class="d-stat" class:stat-warn={!$connectivityStore.isOnline}>
        <span class="d-stat-val">{$connectivityStore.isOnline ? 'OK' : 'OFF'}</span>
        <span class="d-stat-lbl">Network</span>
      </div>
    </section>

    <!-- Network -->
    <section class="d-panel d-panel-network">
      <header class="d-panel-head">
        <h2>Your Network</h2>
        <span class="d-badge">{members.length}</span>
      </header>
      {#if members.length === 0}
        <GhostConstellation
          title="Add your first family member"
          body="Invite someone to share locations, or open the map to start watching over your people."
          ctaLabel="Open Map"
          memberCount={members.length}
          oninvite={() => push('/')}
        />
      {:else}
        <div class="d-members">
          {#each members as user (user.userId)}
            {@const color = getUserColor(user.userId)}
            {@const pres = presence(user)}
            {@const dist = distText(user)}
            {@const ringState = pres === 'sos' ? 'sos' : (pres === 'offline' ? 'offline' : 'live')}
            <button class="d-member" class:m-sos={pres==='sos'} class:m-off={pres==='offline'} class:m-on={pres==='online'} class:m-mv={pres==='moving'}
              style="--mc:{color}" onclick={() => { focusUser.set(user.userId); push('/'); }}>
              <AvatarRing ring={ringState} size={32} label="{user.displayName} — {presenceLabel(user)}">
                <span class="m-init">{getInitials(user.displayName)}</span>
              </AvatarRing>
              <div class="m-info">
                <span class="m-name">{user.displayName || 'Unknown'}</span>
                <span class="m-status" class:m-sos-text={pres==='sos'}>
                  {pres==='sos'?'SOS':pres==='moving'?speedKmh(user)+' km/h':pres==='offline'?'Offline':'Online'}
                </span>
              </div>
              {#if dist}<span class="m-dist">{dist}</span>{/if}
            </button>
          {/each}
        </div>
      {/if}
    </section>

    <!-- Quick Actions — bento: Live Map + Network are wide focal tiles -->
    <section class="d-panel d-panel-actions">
      <header class="d-panel-head"><h2>Quick Actions</h2></header>
      <div class="d-actions bento-grid">
        <button class="d-act act-map act-hero tactile bento-col-2" onclick={() => visitFeature(null, '/')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 6v16l7-4 8 4 7-4V2l-7 4-8-4-7 4z"/><line x1="8" y1="2" x2="8" y2="18"/><line x1="16" y1="6" x2="16" y2="22"/></svg>
          <span>Live Map</span>
        </button>
        <button class="d-act act-activity tactile" onclick={() => visitFeature('activity', '/activity')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
          <span>Activity</span>
          {#if !visited.activity}<span class="d-dot"></span>{/if}
        </button>
        <button class="d-act act-replay tactile" onclick={() => visitFeature('replay', '/replay')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 .49-4.61"/></svg>
          <span>Routes</span>
          {#if !visited.replay}<span class="d-dot"></span>{/if}
        </button>
        <button class="d-act act-sos tactile" class:act-sos-on={$mySosActive} onclick={() => visitFeature('emergency', '/emergency')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
          <span>Emergency</span>
          {#if !visited.emergency}<span class="d-dot d-dot-red"></span>{/if}
        </button>
        <button class="d-act act-checkin tactile" onclick={() => visitFeature('checkins', '/checkins')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
          <span>Check-ins</span>
          {#if !visited.checkins}<span class="d-dot d-dot-cyan"></span>{/if}
        </button>
        <button class="d-act act-network tactile bento-col-2" onclick={() => visitFeature(null, '/')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="5" r="2"/><circle cx="5" cy="19" r="2"/><circle cx="19" cy="19" r="2"/><line x1="12" y1="7" x2="12" y2="11"/><line x1="8.5" y1="16.5" x2="12" y2="11"/><line x1="15.5" y1="16.5" x2="12" y2="11"/></svg>
          <span>Network</span>
        </button>
      </div>
    </section>

    <div class="d-spacer" style="height:calc(var(--safe-bottom,0px) + 28px)"></div>
  </div>
</div>

<style>
  :root { --ease-expo: cubic-bezier(0.16,1,0.3,1); }

  /* ═══ Shell ═════════════════════════════════════════════════════════ */
  .d {
    height: 100dvh;
    background: var(--surface-0, #050812);
    color: var(--text-primary, #fff);
    font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
    position: relative;
    overflow: hidden;
    opacity: 0; transition: opacity 0.5s ease;
  }
  .d.d-ready { opacity: 1; }

  /* ═══ Aurora + noise (behind everything) ════════════════════════════ */
  .d-aurora {
    position: fixed; inset: 0; pointer-events: none; z-index: 0;
    background:
      radial-gradient(ellipse 70% 50% at 15% 20%, rgba(99,102,241,0.12) 0%, transparent 65%),
      radial-gradient(ellipse 60% 45% at 85% 75%, rgba(139,92,246,0.08) 0%, transparent 60%);
  }
  .d-noise {
    position: fixed; inset: 0; pointer-events: none; z-index: 0; opacity: 0.025;
    background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
    background-size: 180px; mix-blend-mode: overlay;
  }

  /* ═══ ORBIT — FULL SCREEN BACKGROUND ═══════════════════════════════ */
  .d-orbit-bg {
    position: absolute;
    inset: 0;
    z-index: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    pointer-events: auto;
  }
  /* Orbit is hidden on all viewports — the GlobeCanvas is the primary 3D visual on
     both desktop (left HUD) and mobile (inline in scroll flow). Hiding saves RAF CPU. */
  .d-orbit-bg { display: none; }

  /* ═══ Static globe fallback (calm / minimal / low-end — no WebGL) ═══ */
  .d-globe-fallback {
    border-radius: 50%;
    max-width: 100%;
    display: grid;
    place-items: center;
    background:
      radial-gradient(circle at 50% 38%, var(--primary-500-20) 0%, var(--primary-500-08) 52%, transparent 72%);
    border: 1px solid var(--primary-500-12);
    box-shadow: inset 0 1px 0 var(--border-highlight, rgba(255,255,255,0.10));
  }
  .d-gf-glyph {
    font-size: clamp(48px, 22%, 96px);
    line-height: 1;
    opacity: 0.9;
    user-select: none;
  }

  /* ═══ HEADER — floating glass bar ══════════════════════════════════ */
  .d-header {
    position: fixed; top: 0; left: 0; right: 0; z-index: 20;
    display: flex; align-items: center; justify-content: space-between;
    padding: calc(var(--safe-top, 0px) + 10px) 20px 10px;
    background: rgba(5,8,18,0.55);
    backdrop-filter: blur(24px) saturate(1.5);
    -webkit-backdrop-filter: blur(24px) saturate(1.5);
    border-bottom: 1px solid rgba(255,255,255,0.04);
  }
  .d-back {
    display: flex; align-items: center; gap: 4px;
    min-height: 44px; /* WCAG 2.5.8 — minimum touch target */
    padding: 0 var(--space-3) 0 var(--space-2);
    background: rgba(255,255,255,0.06); border: 1px solid rgba(255,255,255,0.08);
    border-radius: 22px;
    color: var(--text-secondary); font-size: var(--text-xs); font-weight: 600;
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out), color var(--duration-fast) var(--ease-out);
  }
  .d-back:hover { background: rgba(255,255,255,0.10); color: var(--text-primary); }
  .d-back:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .d-clock {
    font-size: clamp(1.125rem, 1.4vw, 1.25rem); font-weight: 700; letter-spacing: -0.03em;
    color: var(--text-primary); font-variant-numeric: tabular-nums;
    font-family: var(--font-display, system-ui);
  }

  /* ═══ CONTENT OVERLAY — scrolls over the orbit ═════════════════════ */
  .d-content {
    position: relative; z-index: 5;
    height: 100dvh;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    overscroll-behavior-y: contain;
    padding-top: calc(var(--safe-top, 0px) + 44px);
    pointer-events: none; /* let clicks through to orbit */
  }
  /* Re-enable pointer events on actual content */
  .d-content > * { pointer-events: auto; }

  /* On mobile: content flows normally (orbit behind, content on top) */
  /* On desktop: no-scroll sidebar — everything fits in 100dvh */
  @media (min-width: 768px) {
    .d-content {
      position: absolute;
      top: 0; right: 0; bottom: 0;
      width: min(420px, 38vw);
      padding: calc(var(--safe-top, 0px) + 52px) 16px calc(var(--safe-bottom, 0px) + 12px) 16px;
      background: linear-gradient(90deg, transparent 0%, rgba(5,8,18,0.6) 30%, rgba(5,8,18,0.88) 100%);
      backdrop-filter: blur(12px);
      -webkit-backdrop-filter: blur(12px);
      /* KEY: no outer scroll — flex distributes sections */
      overflow: hidden;
      display: flex;
      flex-direction: column;
    }
    /* Hero + stats never shrink */
    .d-hero { flex-shrink: 0; }
    .d-stats { flex-shrink: 0; padding-top: 8px; }
    /* Network panel grows to fill remaining space */
    .d-panel-network {
      flex: 1;
      min-height: 0;
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }
    .d-panel-network .d-members {
      flex: 1;
      min-height: 0;
      overflow-y: auto;
      scrollbar-width: thin;
      scrollbar-color: rgba(255,255,255,0.06) transparent;
    }
    .d-panel-network :global(.empty),
    .d-panel-network :global(.gc) {
      flex: 1; min-height: 0;
      justify-content: center;
    }
    /* Actions panel never shrinks */
    .d-panel-actions { flex-shrink: 0; }
    /* Hide mobile-only spacer */
    .d-spacer { display: none; }
    /* Compact hero on desktop */
    .d-hero { padding: 10px 16px 0; gap: 8px; }
    .d-name { font-size: clamp(1.4rem, 3.5vw, 1.8rem) !important; }
    .d-panel { margin: 8px 0 0; padding: 10px 12px 10px; }
    .d-safety svg { width: 48px; height: 48px; }
    .d-safety-score { font-size: 11px !important; top: 14px !important; }
  }
  @media (min-width: 1200px) {
    .d-content { width: min(460px, 35vw); }
  }

  /* ═══ Stagger — driven by shared tokens, no inline magic durations ════ */
  .d-hero, .d-stats, .d-panel {
    opacity: 0; transform: translateY(10px);
    transition: opacity var(--duration-slow, 500ms) var(--ease-expo),
                transform var(--duration-slow, 500ms) var(--ease-expo);
  }
  .d-ready .d-hero  { opacity: 1; transform: none; transition-delay: var(--stagger-base, 50ms); }
  .d-ready .d-stats { opacity: 1; transform: none;
    transition-delay: calc(var(--stagger-base, 50ms) + var(--stagger-step, 40ms) * 2); }
  .d-ready .d-panel { opacity: 1; transform: none;
    transition-delay: calc(var(--stagger-base, 50ms) + var(--stagger-step, 40ms) * 4); }
  .d-ready .d-panel + .d-panel {
    transition-delay: calc(var(--stagger-base, 50ms) + var(--stagger-step, 40ms) * 6); }
  /* .d-quote fade: uses --duration-slow for the crossfade (matches QUOTE_FADE_MS=400ms) */
  .d-quote { opacity: 0; transition: opacity var(--duration-slow, 400ms) var(--ease-out, ease); }
  .d-quote.quote-on { opacity: 1; }


  /* ═══ Hero ═════════════════════════════════════════════════════════ */
  .d-hero {
    display: flex; align-items: flex-start; justify-content: space-between;
    padding: 16px 20px 0; gap: 12px;
  }
  .d-greet { flex: 1; min-width: 0; }
  .d-name {
    font-size: clamp(1.7rem, 5vw, 2.4rem); font-weight: 400;
    letter-spacing: -0.03em; line-height: 1.15; margin: 0;
    font-family: var(--font-display, system-ui);
    color: var(--text-secondary); /* base: muted for greeting prefix */
  }
  .d-greet-word {
    /* greeting prefix — medium weight, muted */
    font-weight: 400;
    color: var(--text-secondary);
  }
  .d-name-word {
    /* first name — bold, white→violet gradient using design tokens */
    font-weight: 800;
    background: linear-gradient(135deg, #fff 30%, var(--primary-300, #c4b5fd) 100%);
    -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text;
  }
  .d-date { margin: 5px 0 0; font-size: 12px; font-weight: 400; color: rgba(255,255,255,0.28); letter-spacing: 0.01em; }

  .d-safety { position: relative; display: flex; flex-direction: column; align-items: center; gap: 3px; flex-shrink: 0; }
  .d-ring-progress { transition: stroke-dashoffset 1s cubic-bezier(0.4,0,0.2,1), stroke 0.4s; }
  .d-safety-score { position: absolute; top: 16px; left: 50%; transform: translateX(-50%); font-size: 13px; font-weight: 800; color: rgba(255,255,255,0.9); font-family: var(--font-display, system-ui); }
  .d-safety-badge { display: flex; align-items: center; gap: 4px; font-size: 9px; font-weight: 700; padding: 2px 7px; border-radius: 20px; white-space: nowrap; }
  .badge-safe { background: rgba(16,185,129,0.12); border: 1px solid rgba(16,185,129,0.25); color: var(--success-500); }
  .badge-sos  { background: rgba(239,68,68,0.12); border: 1px solid rgba(239,68,68,0.25); color: var(--danger-400); }
  .d-badge-dot { width: 4px; height: 4px; border-radius: 50%; background: currentColor; animation: pulse 2s ease-in-out infinite; }
  @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:0.3} }

  /* ═══ Quote ════════════════════════════════════════════════════════ */
  .d-quote {
    padding: 8px 20px; text-align: center;
    opacity: 0; transition: opacity 0.6s ease;
  }
  .d-quote.quote-on { opacity: 1; }
  .d-quote-text { font-size: 11.5px; font-style: italic; font-family: Georgia, serif; color: rgba(255,255,255,0.30); line-height: 1.6; margin: 0 0 3px; }
  .d-quote-author { font-size: 8px; font-weight: 700; color: rgba(139,92,246,0.4); text-transform: uppercase; letter-spacing: 0.06em; }
  @media (min-width: 768px) {
    .d-quote { display: none; } /* Replaced by .d-quote-left in the left HUD on desktop */
  }

  /* ═══ Stats — bento (large safety focal tile + smaller stat tiles) ═ */
  .d-stats { padding: 12px 20px 0; }
  .d-stat {
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.07);
    border-radius: 12px; padding: 8px 6px;
    display: flex; flex-direction: column; gap: 1px; align-items: center; justify-content: center;
    position: relative; overflow: hidden;
    backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px);
    transition: border-color 0.2s, background 0.2s;
  }
  .d-stat:active { background: rgba(255,255,255,0.07); }
  .d-stat-val { font-size: 18px; font-weight: 800; color: var(--text-primary); line-height: 1; letter-spacing: -0.04em; font-variant-numeric: tabular-nums; font-family: var(--font-display, system-ui); }
  .d-stat-lbl { font-size: var(--text-2xs, 8px); font-weight: 600; color: var(--text-tertiary); text-transform: uppercase; letter-spacing: 0.07em; }
  .stat-active { border-color: rgba(16,185,129,0.25); }
  .stat-active .d-stat-val { color: var(--success-400); }
  .stat-warn { border-color: rgba(245,158,11,0.25); }
  .stat-warn .d-stat-val { color: var(--warning-400); }
  .d-stat-dot { position: absolute; top: 5px; right: 5px; width: 4px; height: 4px; border-radius: 50%; background: var(--success-400); animation: pulse 2s ease-in-out infinite; }

  /* Large focal tile — safety score anchors the bento, draws the eye first */
  .d-stat-hero {
    display: flex; flex-direction: column; justify-content: center; align-items: flex-start;
    gap: 6px; padding: 14px 16px;
    border-radius: 14px;
    position: relative; overflow: hidden;
    background: var(--success-500-12);
    border: 1px solid var(--success-500-20);
    backdrop-filter: blur(14px); -webkit-backdrop-filter: blur(14px);
  }
  .d-stat-hero.hero-alert {
    background: var(--danger-500-12);
    border-color: var(--danger-500-20);
  }
  .d-sh-label {
    font-size: var(--text-2xs, 8px); font-weight: 700; text-transform: uppercase;
    letter-spacing: 0.08em; color: var(--text-tertiary);
  }
  .d-sh-val {
    font-size: clamp(1.875rem, 2.9vw, 2.5rem); font-weight: 800; line-height: 1; letter-spacing: -0.05em;
    font-family: var(--font-display, system-ui); font-variant-numeric: tabular-nums;
  }
  .d-sh-badge {
    display: inline-flex; align-items: center; gap: 4px;
    font-size: 9px; font-weight: 700; padding: 2px 8px; border-radius: 20px; white-space: nowrap;
  }
  /* Mobile: focal tile is a full-width banner, not a tall 2-row block */
  @media (max-width: 767px) {
    .d-stat-hero { grid-row: span 1; }
  }

  /* ═══ Panel (glass card for Network & Actions) ═════════════════════ */
  .d-panel {
    margin: 12px 20px 0;
    background: rgba(255,255,255,0.03);
    border: 1px solid rgba(255,255,255,0.06);
    border-radius: 16px;
    padding: 14px 14px 12px;
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
  }
  .d-panel-head { display: flex; align-items: center; gap: 6px; margin-bottom: 10px; }
  .d-panel-head h2 { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.08em; color: rgba(255,255,255,0.35); margin: 0; flex: 1; }
  .d-badge { font-size: 9px; font-weight: 700; background: rgba(99,102,241,0.12); border: 1px solid rgba(99,102,241,0.2); color: rgba(139,92,246,0.8); padding: 1px 6px; border-radius: 20px; }

  /* ═══ Members — list layout (works in sidebar) ═════════════════════ */
  .d-members { display: flex; flex-direction: column; gap: 6px; }
  .d-member {
    display: flex; align-items: center; gap: 10px;
    background: rgba(255,255,255,0.03);
    border: 1px solid rgba(255,255,255,0.05);
    border-radius: 12px; padding: 8px 10px;
    cursor: pointer;
    transition: border-color 0.15s, background 0.15s, transform 0.1s;
    -webkit-tap-highlight-color: transparent;
  }
  .d-member:hover { border-color: rgba(255,255,255,0.12); background: rgba(255,255,255,0.05); }
  .d-member:active { transform: scale(0.98); }
  /* Presence-tinted backgrounds — subtle, glanceable state at a glance */
  .d-member.m-on {
    background: color-mix(in oklch, var(--success-500) 7%, transparent);
    border-color: color-mix(in oklch, var(--success-500) 18%, transparent);
  }
  .d-member.m-mv {
    background: color-mix(in oklch, var(--info-500, #3b82f6) 8%, transparent);
    border-color: color-mix(in oklch, var(--info-500, #3b82f6) 20%, transparent);
  }
  .d-member.m-sos { background: var(--danger-500-12); border-color: rgba(239,68,68,0.25); }
  .d-member.m-off { opacity: 0.45; }

  /* .m-av and .m-dot replaced by AvatarRing — ring grammar now shared */
  .m-init { font-size: 11px; font-weight: 800; color: var(--mc,#6366f1); line-height: 1; user-select: none; }
  .m-info { flex: 1; min-width: 0; }
  .m-name { display: block; font-size: var(--text-xs); font-weight: 700; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .m-status { font-size: 10px; color: rgba(255,255,255,0.35); }
  .m-sos-text { color: var(--danger-400); font-weight: 700; }
  .m-dist { font-size: 9px; color: rgba(255,255,255,0.22); font-variant-numeric: tabular-nums; flex-shrink: 0; }

  /* ═══ Actions — bento (Live Map + Network are wide focal tiles) ═════ */
  .d-actions { align-items: stretch; }
  .d-act {
    position: relative;
    background: rgba(255,255,255,0.03); border: 1px solid rgba(255,255,255,0.06);
    border-radius: 12px; padding: 12px 6px 10px;
    display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 6px;
    cursor: pointer; color: rgba(255,255,255,0.65); font-size: 11px; font-weight: 600;
    transition: border-color 0.2s, background 0.2s;
    -webkit-tap-highlight-color: transparent;
    min-height: 44px;
  }
  .d-act:hover { border-color: rgba(255,255,255,0.12); background: rgba(255,255,255,0.05); }
  /* Wide focal tiles read as the primary path in — subtle brand wash */
  .d-act.act-hero {
    background: var(--primary-500-08);
    border-color: var(--primary-500-20);
    font-size: 12px;
  }
  .d-act.act-hero:hover { background: var(--primary-500-12); border-color: var(--primary-500-30); }
  .act-map { color: var(--primary-400); } .act-activity { color: var(--success-300, #34d399); } .act-replay { color: var(--warning-300, #fbbf24); }
  .act-sos { color: var(--danger-400); } .act-checkin { color: var(--info-300, #22d3ee); } .act-network { color: var(--primary-300); }
  .act-sos-on { border-color: rgba(239,68,68,0.25); animation: sos-b 2s ease-in-out infinite; }
  @keyframes sos-b { 0%,100%{border-color:rgba(239,68,68,0.15)} 50%{border-color:rgba(239,68,68,0.45)} }

  .d-dot { position: absolute; top: 6px; right: 6px; width: 6px; height: 6px; border-radius: 50%; background: var(--warning-400); box-shadow: 0 0 5px rgba(245,158,11,0.6); }
  .d-dot-red { background: var(--danger-500); box-shadow: 0 0 5px rgba(239,68,68,0.6); }
  .d-dot-cyan { background: var(--info-400, #22d3ee); box-shadow: 0 0 5px rgba(34,211,238,0.6); }

  /* ═══ Mobile Globe section ══════════════════════════════════════════ */
  .d-mobile-globe {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    padding: 4px 20px 4px;
  }
  @media (min-width: 768px) {
    .d-mobile-globe { display: none; }
  }
  .d-mob-coords {
    display: flex; flex-direction: column; align-items: center; gap: 2px;
  }
  .d-mob-you {
    font-size: 8px; font-weight: 800;
    color: rgba(167,139,250,0.55);
    text-transform: uppercase; letter-spacing: 0.12em;
  }
  .d-mob-ll {
    font-size: 10px; font-weight: 600;
    font-family: var(--font-mono, monospace);
    color: rgba(255,255,255,0.22);
    letter-spacing: 0.03em;
    font-variant-numeric: tabular-nums;
  }

  /* ═══ Reduced motion ═══════════════════════════════════════════════ */
  @media (prefers-reduced-motion: reduce) {
    /* Disable all decorative animations */
    .d-aurora, .d-badge-dot, .d-stat-dot, .d-globe-blip,
    .d-member.m-sos, .d-act.act-sos-on,
    .gsr-on .d-gsr-dot, .gsr-alert .d-gsr-dot {
      animation: none !important;
    }
    /* Show content immediately — simultaneous fade, no stagger */
    .d-hero, .d-stats, .d-panel, .d-quote, .d-quote-globe {
      opacity: 1 !important;
      transform: none !important;
      transition: none !important;
      transition-delay: 0ms !important;
    }
    /* Disable globe tilt hover transforms — can cause motion sickness */
    .d-globe-side-l .d-side-card:hover,
    .d-globe-side-r .d-side-card:hover {
      transform: none !important;
    }
    /* Disable ring progress animation */
    .d-ring-progress {
      transition: none !important;
    }
  }

  /* ═══ Mouse cursor glow — rendered only on non-coarse-pointer via {#if} ═ */
  /* CSS just controls visibility transitions; existence is JS-gated.          */
  .d-cursor-glow {
    position: fixed;
    width: 360px; height: 360px;
    border-radius: 50%;
    background: radial-gradient(circle, var(--primary-500-08, rgba(99,102,241,0.055)) 0%, transparent 70%);
    transform: translate(-50%, -50%);
    pointer-events: none;
    z-index: 2;
    opacity: 0;
    transition: opacity var(--duration-slow, 400ms) var(--ease-out, ease);
  }
  .d-cursor-glow.d-cursor-visible { opacity: 1; }
  @media (prefers-reduced-motion: reduce) {
    .d-cursor-glow { display: none !important; }
  }

  /* ═══ Left HUD overlay — globe fills the left area ════════════════ */
  .d-left-hud {
    display: none;
  }
  @media (min-width: 768px) {
    .d-left-hud {
      display: flex;
      align-items: stretch;   /* stretch so globe-col fills full height */
      justify-content: center;
      position: absolute;
      top: 0; bottom: 0; left: 0;
      right: min(420px, 38vw);
      pointer-events: none;
      z-index: 3;
    }
    .d-left-hud > :global(*) { pointer-events: auto; }
  }
  @media (min-width: 1200px) {
    .d-left-hud { right: min(460px, 35vw); }
  }

  /* ── Globe column layout ────────────────────────────────────────── */
  .d-globe-col {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    padding: calc(var(--safe-top, 0px) + 64px) 24px 32px;
    max-width: 480px;
    width: 100%;
    height: 100%;
    pointer-events: auto;
  }

  /* ── Top section (eyebrow + status pills) ─────────────────────── */
  .d-globe-top {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    width: 100%;
  }

  /* ── Status pills row ─────────────────────────────────────────── */
  .d-globe-status-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    justify-content: center;
  }
  .d-gsr-pill {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 4px 10px 4px 8px;
    border-radius: 20px;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.07);
    font-size: 9px;
    font-weight: 700;
    color: rgba(255,255,255,0.35);
    letter-spacing: 0.04em;
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    transition: border-color 0.2s, color 0.2s;
  }
  .d-gsr-dot {
    width: 5px; height: 5px;
    border-radius: 50%;
    background: rgba(255,255,255,0.18);
    flex-shrink: 0;
    transition: background 0.2s, box-shadow 0.2s;
  }
  /* ON / green */
  .gsr-on {
    border-color: rgba(16,185,129,0.22);
    color: var(--success-300, rgba(52,211,153,0.75));
  }
  .gsr-on .d-gsr-dot {
    background: var(--success-500, #10b981);
    box-shadow: 0 0 5px rgba(16,185,129,0.55);
    animation: blip-g 2.5s ease-in-out infinite;
  }
  /* SAFE / green */
  .gsr-safe {
    border-color: rgba(16,185,129,0.22);
    color: var(--success-300, rgba(52,211,153,0.75));
  }
  .gsr-safe .d-gsr-dot {
    background: var(--success-500, #10b981);
    box-shadow: 0 0 5px rgba(16,185,129,0.55);
  }
  /* ALERT / red */
  .gsr-alert {
    border-color: rgba(239,68,68,0.28);
    color: var(--danger-400);
  }
  .gsr-alert .d-gsr-dot {
    background: var(--danger-500, #ef4444);
    box-shadow: 0 0 5px rgba(239,68,68,0.60);
    animation: pulse 1.5s ease-in-out infinite;
  }
  /* WARN / amber */
  .gsr-warn {
    border-color: rgba(245,158,11,0.24);
    color: var(--warning-300, rgba(251,191,36,0.75));
  }
  .gsr-warn .d-gsr-dot {
    background: var(--warning-400, #f59e0b);
    box-shadow: 0 0 5px rgba(245,158,11,0.50);
  }

  /* ── Bottom section (loc + quote) ────────────────────────────── */
  .d-globe-bottom {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    width: 100%;
  }

  .d-globe-eyebrow {
    display: flex; align-items: center; gap: 7px;
    font-size: 8.5px; font-weight: 800;
    color: rgba(255,255,255,0.14);
    letter-spacing: 0.15em; text-transform: uppercase;
    font-family: var(--font-mono, monospace);
  }
  .d-globe-blip {
    width: 5px; height: 5px; border-radius: 50%;
    background: var(--success-500);
    box-shadow: 0 0 6px rgba(16,185,129,0.7);
    animation: blip-g 2.5s ease-in-out infinite;
    flex-shrink: 0;
  }
  @keyframes blip-g { 0%,100%{opacity:1} 50%{opacity:0.2} }

  .d-loc-info {
    display: flex; align-items: center; gap: 6px; flex-wrap: wrap; justify-content: center;
    color: rgba(167,139,250,0.55);
  }
  .d-loc-you {
    font-size: 9px; font-weight: 800;
    color: rgba(167,139,250,0.65);
    text-transform: uppercase; letter-spacing: 0.10em;
  }
  .d-loc-coords {
    font-size: 11px; font-weight: 600;
    font-family: var(--font-mono, monospace);
    color: rgba(255,255,255,0.30);
    letter-spacing: 0.03em;
    font-variant-numeric: tabular-nums;
  }
  .d-loc-unknown {
    font-size: 9px; font-weight: 600;
    color: rgba(255,255,255,0.18);
    letter-spacing: 0.06em; text-transform: uppercase;
    gap: 5px;
  }

  /* ── Quote in globe column — button for keyboard accessibility ────── */
  .d-quote-globe {
    position: relative;
    width: 100%; max-width: 340px;
    padding: 16px 20px 14px;
    background: rgba(5,8,18,0.50);
    border: 1px solid rgba(255,255,255,0.06);
    border-top: 1px solid rgba(139,92,246,0.12);
    border-radius: 12px;
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    cursor: pointer;
    text-align: center;
    opacity: 0; transform: translateY(8px);
    transition: opacity var(--duration-slow, 500ms) var(--ease-out, ease),
                transform var(--duration-slow, 500ms) var(--ease-out, ease),
                border-color var(--duration-fast, 150ms),
                background var(--duration-fast, 150ms);
    color: inherit;
    font: inherit;
  }
  .d-quote-globe.dqg-on { opacity: 1; transform: translateY(0); }
  .d-quote-globe:hover {
    background: rgba(99,102,241,0.06);
    border-color: rgba(99,102,241,0.18);
  }
  .d-quote-globe:focus-visible {
    outline: 2px solid rgba(99,102,241,0.6);
    outline-offset: 2px;
  }
  .d-qg-mark {
    display: block;
    font-size: clamp(1.75rem, 2.6vw, 2.25rem); font-family: Georgia, serif;
    color: rgba(139,92,246,0.14);
    line-height: 1; margin-bottom: -8px;
    user-select: none; pointer-events: none;
  }
  .d-qg-text {
    font-size: 12px; font-style: italic;
    font-family: Georgia, 'Times New Roman', serif;
    color: rgba(255,255,255,0.38);
    line-height: 1.7; margin: 0 0 8px;
  }
  .d-qg-author {
    display: block;
    font-size: 9px; font-weight: 700;
    color: rgba(139,92,246,0.50);
    text-transform: uppercase; letter-spacing: 0.08em;
    margin-bottom: 8px;
  }
  .d-qg-cycle {
    display: block; font-size: 8px; font-weight: 600;
    color: rgba(255,255,255,0.10); letter-spacing: 0.05em;
    transition: color 0.2s;
  }
  .d-quote-globe:hover .d-qg-cycle { color: rgba(139,92,246,0.40); }

  /* ═══ Globe center — globe + flanking side cards ════════════════════ */
  .d-globe-center {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    gap: 14px;
  }

  /* Side cards hidden below 960px (not enough room beside globe) */
  .d-globe-side { display: none; flex: 1; max-width: 155px; }
  @media (min-width: 960px) {
    .d-globe-side { display: flex; align-items: center; justify-content: center; }
  }

  .d-side-card {
    width: 100%;
    background: rgba(5,8,18,0.55);
    border: 1px solid rgba(255,255,255,0.07);
    border-radius: 16px;
    padding: 14px 12px 12px;
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    text-align: center;
    cursor: default;
    transition: transform 0.35s cubic-bezier(0.34,1.56,0.64,1), border-color 0.2s, box-shadow 0.2s;
    transform-style: preserve-3d;
  }
  .d-globe-side-l .d-side-card:hover {
    transform: perspective(500px) rotateY(6deg) translateY(-3px);
    border-color: rgba(139,92,246,0.20);
    box-shadow: 4px 8px 28px rgba(0,0,0,0.22);
  }
  .d-globe-side-r .d-side-card:hover {
    transform: perspective(500px) rotateY(-6deg) translateY(-3px);
    border-color: rgba(139,92,246,0.20);
    box-shadow: -4px 8px 28px rgba(0,0,0,0.22);
  }

  .d-side-label {
    font-size: 8px; font-weight: 800; text-transform: uppercase;
    letter-spacing: 0.14em; color: rgba(255,255,255,0.20);
    margin-bottom: 6px;
    font-family: var(--font-mono, monospace);
  }
  .d-side-big {
    font-size: clamp(1.875rem, 2.7vw, 2.375rem); font-weight: 800;
    color: rgba(255,255,255,0.90);
    line-height: 1; letter-spacing: -0.05em;
    font-family: var(--font-display, system-ui);
    font-variant-numeric: tabular-nums;
    transition: color 0.4s;
  }
  .d-side-sub {
    font-size: 8px; font-weight: 600;
    color: rgba(255,255,255,0.20); text-transform: uppercase;
    letter-spacing: 0.08em; margin-top: 2px;
  }
  .d-side-divider {
    height: 1px; background: rgba(255,255,255,0.06); margin: 10px 0 8px;
  }
  .d-side-row {
    display: flex; align-items: center; gap: 6px;
    font-size: 9px; font-weight: 600; color: rgba(255,255,255,0.28);
    margin-top: 5px; text-align: left;
  }
  .row-safe { color: var(--success-400); }
  .row-sos  { color: var(--danger-400); }
  .d-sd {
    width: 5px; height: 5px; border-radius: 50%; flex-shrink: 0;
    background: rgba(255,255,255,0.15);
  }
  .d-sd.d-sd-muted { background: rgba(255,255,255,0.12); }
  .d-sd.d-sd-warn { background: var(--warning-400, #f59e0b); }
  .d-sd.dot-on { background: var(--success-500); box-shadow: 0 0 4px rgba(16,185,129,0.55); }
  .d-sd.dot-mv { background: var(--info-500, #3b82f6); box-shadow: 0 0 4px rgba(59,130,246,0.55); }

  /* ═══ Interactive member avatar ring ════════════════════════════════ */
  .d-member-ring {
    display: flex; align-items: center; gap: 8px;
    flex-wrap: wrap; justify-content: center;
  }
  /* d-mr-bubble is now a <button> — reset button defaults */
  .d-mr-bubble {
    position: relative; width: 44px; height: 44px;
    border-radius: 50%;
    background: color-mix(in srgb, var(--mc,#6366f1) 14%, rgba(5,8,18,0.75));
    border: 2px solid color-mix(in srgb, var(--mc,#6366f1) 60%, transparent);
    display: flex; align-items: center; justify-content: center;
    cursor: pointer;
    transition: transform var(--duration-standard, 250ms) var(--ease-spring, cubic-bezier(0.34,1.56,0.64,1)),
                box-shadow var(--duration-standard, 250ms),
                border-color var(--duration-fast, 150ms);
    -webkit-tap-highlight-color: transparent;
    user-select: none;
    padding: 0;
    color: inherit;
  }
  .d-mr-bubble:hover {
    transform: scale(1.18) translateY(-3px);
    border-color: var(--mc, #6366f1);
    box-shadow: 0 6px 20px color-mix(in srgb, var(--mc,#6366f1) 40%, transparent);
  }
  .d-mr-bubble:focus-visible {
    outline: 2px solid var(--mc, #6366f1);
    outline-offset: 2px;
  }
  /* .d-mr-dot replaced by AvatarRing — ring grammar now shared */
  .d-mr-init {
    font-size: 12px; font-weight: 800;
    color: color-mix(in srgb, var(--mc,#6366f1) 90%, white);
    line-height: 1; user-select: none; pointer-events: none;
  }
  .d-mr-more {
    width: 44px; height: 44px; border-radius: 50%;
    background: rgba(255,255,255,0.04);
    border: 2px solid rgba(255,255,255,0.10);
    display: flex; align-items: center; justify-content: center;
    font-size: 9px; font-weight: 700; color: rgba(255,255,255,0.30);
  }
</style>
