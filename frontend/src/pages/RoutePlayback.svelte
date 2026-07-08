<script>
  import { run } from 'svelte/legacy';

  import { onMount, onDestroy } from 'svelte';
  import { push, querystring } from 'svelte-spa-router';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { authUser } from '../lib/stores/auth.js';
  import { otherUsers } from '../lib/stores/map.js';
  import { socket } from '../lib/socket.js';
  import Card from '../components/primitives/Card.svelte';
  import Skeleton from '../components/primitives/Skeleton.svelte';



  // ── Trail data & playback state ─────────────────────────────────────────────
  let points = $state([]);
  let loading = $state(false);
  let error = $state(null);
  let windowMinutes = $state(60);
  const WINDOWS = [15, 30, 60];

  let mapEl = $state();
  let map = null;
  let mapReady = false;

  // Playback
  let playing = $state(false);
  let playProgress = $state(0);
  let playIndex = $state(0);
  let playTimer = null;
  let movingMarker = null;


  function calcDist(pts) {
    let d = 0;
    for (let i = 1; i < pts.length; i++) {
      const R = 6371;
      const dLat = (pts[i].lat - pts[i-1].lat) * Math.PI / 180;
      const dLng = (pts[i].lng - pts[i-1].lng) * Math.PI / 180;
      const a = Math.sin(dLat/2)**2 + Math.cos(pts[i-1].lat*Math.PI/180) * Math.cos(pts[i].lat*Math.PI/180) * Math.sin(dLng/2)**2;
      d += R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1-a));
    }
    return Math.round(d * 10) / 10;
  }

  // ── Socket trail request ────────────────────────────────────────────────────
  let trailTimeout;
  function requestTrail() {
    if (!targetUserId) { loading = false; error = null; return; }
    loading = true;
    error = null;
    playing = false;
    playProgress = 0;
    playIndex = 0;
    socket.emit('getRecentTrail', { targetUserId, windowMinutes });
    clearTimeout(trailTimeout);
    trailTimeout = setTimeout(() => {
      if (loading) { loading = false; error = 'Took too long — try again'; }
    }, 15000);
  }

  function onTrailData(data) {
    clearTimeout(trailTimeout);
    loading = false;
    points = (data.points || data || []).slice().sort((a, b) => a.ts - b.ts);
    if (points.length === 0) { error = 'No location data for this time'; return; }
    renderTrail();
  }

  function onTrailError(data) {
    clearTimeout(trailTimeout);
    loading = false;
    error = data?.error || 'Could not load the route';
  }

  // ── MapLibre GL ─────────────────────────────────────────────────────────────
  const MAP_STYLE = {
    version: 8,
    sources: { osm: { type: 'raster', tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'], tileSize: 256, attribution: '© OpenStreetMap contributors' } },
    layers: [{ id: 'osm', type: 'raster', source: 'osm' }],
  };

  async function initMap() {
    const maplibregl = (await import('maplibre-gl')).default;
    if (!mapEl) return;
    map = new maplibregl.Map({
      container: mapEl,
      style: MAP_STYLE,
      zoom: 14,
      center: [0, 0],
      attributionControl: false,
    });
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'bottom-right');
    map.on('load', () => { mapReady = true; if (points.length) renderTrail(); });
  }

  async function renderTrail() {
    if (!map || !mapReady || points.length === 0) return;

    const coords = points.map(p => [p.lng, p.lat]);

    ['replay-line', 'replay-start', 'replay-end'].forEach(id => {
      if (map.getLayer(id)) map.removeLayer(id);
    });
    ['replay-route', 'replay-start-src', 'replay-end-src'].forEach(id => {
      if (map.getSource(id)) map.removeSource(id);
    });
    if (movingMarker) { movingMarker.remove(); movingMarker = null; }

    map.addSource('replay-route', {
      type: 'geojson',
      data: { type: 'Feature', geometry: { type: 'LineString', coordinates: coords } },
    });
    map.addLayer({
      id: 'replay-line',
      type: 'line',
      source: 'replay-route',
      layout: { 'line-cap': 'round', 'line-join': 'round' },
      paint: { 'line-color': '#14b8a6', 'line-width': 4, 'line-opacity': 0.85 },
    });

    map.addSource('replay-start-src', { type: 'geojson', data: { type: 'Feature', geometry: { type: 'Point', coordinates: coords[0] } } });
    map.addLayer({ id: 'replay-start', type: 'circle', source: 'replay-start-src', paint: { 'circle-radius': 7, 'circle-color': '#10b981', 'circle-stroke-width': 2.5, 'circle-stroke-color': '#fff' } });

    map.addSource('replay-end-src', { type: 'geojson', data: { type: 'Feature', geometry: { type: 'Point', coordinates: coords[coords.length - 1] } } });
    map.addLayer({ id: 'replay-end', type: 'circle', source: 'replay-end-src', paint: { 'circle-radius': 7, 'circle-color': '#ef4444', 'circle-stroke-width': 2.5, 'circle-stroke-color': '#fff' } });

    const lats = points.map(p => p.lat), lngs = points.map(p => p.lng);
    map.fitBounds([[Math.min(...lngs), Math.min(...lats)], [Math.max(...lngs), Math.max(...lats)]], { padding: 48 });

    const el = document.createElement('div');
    el.className = 'moving-dot';
    const { default: ml } = await import('maplibre-gl');
    movingMarker = new ml.Marker({ element: el }).setLngLat(coords[0]).addTo(map);
    playIndex = 0;
    playProgress = 0;
  }

  // ── Playback ─────────────────────────────────────────────────────────────────
  function togglePlay() {
    const prefReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (prefReduced) {
      playIndex = points.length - 1;
      playProgress = 100;
      if (movingMarker && points.length) movingMarker.setLngLat([points[points.length-1].lng, points[points.length-1].lat]);
      return;
    }
    playing = !playing;
    if (playing) {
      if (playIndex >= points.length - 1) { playIndex = 0; playProgress = 0; }
      tick();
    } else {
      clearTimeout(playTimer);
    }
  }

  function tick() {
    if (!playing || playIndex >= points.length - 1) {
      playing = false;
      return;
    }
    playIndex++;
    playProgress = Math.round((playIndex / (points.length - 1)) * 100);
    if (movingMarker && points[playIndex]) {
      movingMarker.setLngLat([points[playIndex].lng, points[playIndex].lat]);
    }
    const delay = Math.max(40, Math.min(300, 8000 / points.length));
    playTimer = setTimeout(tick, delay);
  }

  function onScrub(e) {
    clearTimeout(playTimer);
    playing = false;
    const pct = Number(e.target.value);
    playProgress = pct;
    playIndex = Math.round((pct / 100) * (points.length - 1));
    if (movingMarker && points[playIndex]) {
      movingMarker.setLngLat([points[playIndex].lng, points[playIndex].lat]);
    }
  }

  function setWindow(w) {
    windowMinutes = w;
    requestTrail();
  }

  function fmtTs(ts) {
    if (!ts) return '--:--';
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }

  onMount(() => {
    socket.on('recentTrail', onTrailData);
    socket.on('trailError', onTrailError);
    initMap().then(() => requestTrail());
  });

  onDestroy(() => {
    clearTimeout(playTimer);
    clearTimeout(trailTimeout);
    socket.off('recentTrail', onTrailData);
    socket.off('trailError', onTrailError);
    if (map) map.remove();
  });
  run(() => {
    if (!$authUser) push('/login');
  });
  // ── Parse userId from query string ─────────────────────────────────────────
  let params = $derived(new URLSearchParams($querystring || ''));
  let targetUserId = $derived(params.get('userId') || '');
  let targetUser = $derived(targetUserId
    ? (() => { for (const [, u] of $otherUsers) { if (u.userId === targetUserId) return u; } return null; })()
    : null);
  let displayName = $derived(targetUser?.displayName || targetUserId || 'Route');
  // Stats
  let totalPoints = $derived(points.length);
  let totalDistKm = $derived(calcDist(points));
  let durationMin = $derived(points.length > 1
    ? Math.round((points[points.length-1].ts - points[0].ts) / 60000)
    : 0);
  let currentTs = $derived(points[playIndex]?.ts);
  // Fraction (0–1) driving the transform-based progress fill
  let playFrac = $derived(Math.max(0, Math.min(1, playProgress / 100)));
</script>

<svelte:head>
  <link rel="stylesheet" href="https://unpkg.com/maplibre-gl/dist/maplibre-gl.css" />
  <style>
    /* Moving marker — inlined because it targets a DOM element created imperatively */
    .moving-dot {
      width: 18px;
      height: 18px;
      background: white;
      border: 3px solid #14b8a6;
      border-radius: 50%;
      box-shadow: 0 0 12px rgba(20, 184, 166, 0.65), 0 2px 8px rgba(0,0,0,0.35);
    }
  </style>
</svelte:head>

<div class="replay-page page-enter aurora-ambient">
  <!-- Header -->
  <header class="rp-header">
    <button class="icon-btn" onclick={() => push('/')} aria-label="Back to map">
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><polyline points="15 18 9 12 15 6"/></svg>
    </button>
    <div class="rp-title-group">
      <h1 class="rp-title">Route Replay</h1>
      {#if displayName}<span class="rp-sub">{displayName}</span>{/if}
    </div>
    <!-- Window selector -->
    <div class="window-pills" role="group" aria-label="Time window">
      {#each WINDOWS as w}
        <button
          class="window-pill tactile"
          class:active={windowMinutes === w}
          onclick={() => setWindow(w)}
          aria-pressed={windowMinutes === w}
          aria-label="{w} minutes"
        >{w}m</button>
      {/each}
    </div>
  </header>

  <!-- Map -->
  <div class="map-wrap" bind:this={mapEl} aria-label="Route map" role="img">
    {#if loading}
      <div class="map-overlay" transition:fade={{ duration: 200 }}>
        <!-- Skeleton shimmer instead of spinner -->
        <div class="load-skeleton" role="status" aria-label="Loading route…" aria-busy="true">
          <Skeleton variant="line" width="100%" height="12px" />
          <Skeleton variant="line" width="70%" height="12px" />
          <Skeleton variant="line" width="45%" height="12px" />
        </div>
        <p class="overlay-hint">Loading route…</p>
      </div>
    {:else if !targetUserId && !loading}
      <div class="map-overlay" transition:fade={{ duration: 200 }}>
        <div class="empty-icon" aria-hidden="true">
          <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
        </div>
        <p class="empty-title">Choose someone to replay</p>
        <p class="empty-sub">Open a person's card on the map and tap <strong>Route History</strong> to see their journey here.</p>
        <button class="action-btn" onclick={() => push('/')}>Open Map</button>
      </div>
    {:else if error}
      <div class="map-overlay" transition:fade={{ duration: 200 }}>
        <div class="err-icon" aria-hidden="true">
          <svg xmlns="http://www.w3.org/2000/svg" width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><circle cx="12" cy="16" r="0.5" fill="currentColor"/></svg>
        </div>
        <p class="err-msg">{error}</p>
        <button class="action-btn" onclick={requestTrail}>Try again</button>
      </div>
    {/if}
  </div>

  <!-- Controls panel -->
  {#if !loading && !error && points.length > 0}
    <div class="controls-panel" transition:fly={{ y: 200, duration: 320, easing: cubicOut }}>
      <!-- Stats bento -->
      <div class="stats-bento bento-grid" style="--bento-cols: 3;" role="region" aria-label="Route statistics">
        <Card variant="glass" padding="sm" hover={false}>
          <div class="stat">
            <span class="stat-icon" aria-hidden="true">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="6" cy="19" r="3"/><path d="M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15"/><circle cx="18" cy="5" r="3"/></svg>
            </span>
            <span class="stat-text">
              <span class="stat-val">{totalDistKm}<span class="stat-unit">km</span></span>
              <span class="stat-lbl">Distance</span>
            </span>
          </div>
        </Card>
        <Card variant="glass" padding="sm" hover={false}>
          <div class="stat">
            <span class="stat-icon" aria-hidden="true">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><polyline points="12 7 12 12 15 14"/></svg>
            </span>
            <span class="stat-text">
              <span class="stat-val">{durationMin}<span class="stat-unit">min</span></span>
              <span class="stat-lbl">Duration</span>
            </span>
          </div>
        </Card>
        <Card variant="glass" padding="sm" hover={false}>
          <div class="stat">
            <span class="stat-icon" aria-hidden="true">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="5" cy="12" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="19" cy="12" r="1.6"/></svg>
            </span>
            <span class="stat-text">
              <span class="stat-val">{totalPoints}</span>
              <span class="stat-lbl">Points</span>
            </span>
          </div>
        </Card>
      </div>

      <!-- Scrubber + timestamp -->
      <div class="scrubber-row" role="region" aria-label="Playback scrubber">
        <time
          class="ts-display"
          aria-live="polite"
          aria-label="Current time: {fmtTs(currentTs)}"
        >{fmtTs(currentTs)}</time>
        <div class="scrubber-wrap" style="--frac: {playFrac};">
          <div class="scrubber-track" aria-hidden="true">
            <div class="scrubber-fill"></div>
          </div>
          <input
            type="range"
            min="0"
            max="100"
            step="1"
            value={playProgress}
            oninput={onScrub}
            class="scrubber"
            aria-label="Playback position"
            aria-valuemin="0"
            aria-valuemax="100"
            aria-valuenow={playProgress}
          />
        </div>
        <span class="ts-end">{fmtTs(points[points.length-1]?.ts)}</span>
      </div>

      <!-- Play/Pause button -->
      <button
        class="play-btn"
        class:playing
        onclick={togglePlay}
        aria-label={playing ? 'Pause playback' : 'Play route'}
        aria-pressed={playing}
      >
        {#if playing}
          <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
        {:else}
          <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><polygon points="5 3 19 12 5 21 5 3"/></svg>
        {/if}
        {playing ? 'Pause' : 'Play'}
      </button>
    </div>
  {/if}
</div>

<style>
  .replay-page {
    display: flex;
    flex-direction: column;
    height: 100dvh;
    background: var(--surface-0);
    color: var(--text-primary);
    font-family: var(--font-sans);
    overflow: hidden;
  }

  /* ── Header ─────────────────────────────────────────────────────────────── */
  .rp-header {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
    padding: calc(env(safe-area-inset-top, 0px) + var(--space-2-5)) var(--space-3-5) var(--space-2-5);
    background: var(--glass-bg-strong);
    backdrop-filter: var(--glass-blur);
    -webkit-backdrop-filter: var(--glass-blur);
    border-bottom: 1px solid var(--border-subtle);
    z-index: 100;
    flex-shrink: 0;
  }

  /* Back button — 44px touch target */
  .icon-btn {
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--surface-hover);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-full);
    color: var(--text-primary);
    cursor: pointer;
    flex-shrink: 0;
    transition: background var(--duration-fast) var(--ease-out),
                transform var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }
  .icon-btn:hover { background: var(--surface-active); }
  .icon-btn:active { transform: scale(0.88); }
  .icon-btn:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 2px;
  }

  .rp-title-group { flex: 1; min-width: 0; }

  .rp-title {
    margin: 0;
    font-family: var(--font-display);
    font-size: var(--text-2xl);
    font-weight: 800;
    letter-spacing: -0.03em;
    line-height: 1.15;
  }

  .rp-sub {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    display: block;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Window pills */
  .window-pills {
    display: flex;
    gap: var(--space-1);
    flex-shrink: 0;
  }

  .window-pill {
    padding: var(--space-1) var(--space-2-5);
    min-height: 44px;
    border-radius: var(--radius-full);
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 600;
    background: var(--surface-hover);
    border: 1px solid var(--border-default);
    color: var(--text-secondary);
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out),
                border-color var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }
  .window-pill:hover { background: var(--surface-active); }
  .window-pill.active {
    background: var(--primary-500-20);
    border-color: var(--primary-500-30);
    color: var(--primary-400);
  }
  .window-pill:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 2px;
  }

  /* ── Map ─────────────────────────────────────────────────────────────────── */
  .map-wrap {
    flex: 1;
    position: relative;
    overflow: hidden;
  }
  :global(.map-wrap .maplibregl-map) { position: absolute; inset: 0; }

  .map-overlay {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-3);
    background: var(--glass-bg-strong);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    z-index: 10;
    color: var(--text-secondary);
    padding: var(--space-8);
    text-align: center;
  }

  /* Loading skeleton (replaces spinner) */
  .load-skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    width: 180px;
  }

  .overlay-hint {
    font-size: var(--text-sm);
    color: var(--text-tertiary);
    margin: 0;
  }

  /* Empty / error states */
  .empty-icon {
    width: 72px;
    height: 72px;
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--primary-400);
    margin-bottom: var(--space-1);
  }

  .empty-title {
    font-family: var(--font-display);
    font-size: var(--text-base);
    font-weight: 700;
    color: var(--text-primary);
    margin: 0;
  }

  .empty-sub {
    font-size: var(--text-sm);
    color: var(--text-secondary);
    text-align: center;
    max-width: 240px;
    margin: 0;
    line-height: var(--leading-normal);
  }

  .err-icon {
    width: 56px;
    height: 56px;
    border-radius: var(--radius-full);
    background: var(--danger-500-12);
    border: 1px solid var(--danger-500-20);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--danger-400);
  }

  .err-msg {
    font-size: var(--text-sm);
    color: var(--text-secondary);
    text-align: center;
    max-width: 200px;
    margin: 0;
  }

  .action-btn {
    padding: var(--space-2) var(--space-5);
    min-height: 44px;
    border-radius: var(--radius-full);
    background: var(--primary-500-20);
    border: 1px solid var(--primary-500-30);
    color: var(--primary-400);
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 600;
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }
  .action-btn:hover { background: var(--primary-500-30); }
  .action-btn:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 2px;
  }

  /* ── Controls panel ──────────────────────────────────────────────────────── */
  .controls-panel {
    background: var(--glass-bg-strong);
    backdrop-filter: blur(32px) saturate(1.8);
    -webkit-backdrop-filter: blur(32px) saturate(1.8);
    border-top: 1px solid var(--border-subtle);
    padding: var(--space-3-5) var(--space-4) calc(var(--space-3-5) + env(safe-area-inset-bottom, 0px));
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    flex-shrink: 0;
  }

  /* ── Stats bento ─────────────────────────────────────────────────────────── */
  .stats-bento {
    --bento-gap: var(--space-2-5);
  }

  .stat {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1-5);
    text-align: center;
  }

  .stat-icon {
    width: 36px;
    height: 36px;
    flex-shrink: 0;
    display: grid;
    place-items: center;
    border-radius: var(--radius-md);
    background: var(--primary-500-12);
    color: var(--primary-400);
  }
  .stat-icon :global(svg) { width: 18px; height: 18px; }

  .stat-text {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1px;
    min-width: 0;
  }

  .stat-val {
    display: inline-flex;
    align-items: baseline;
    gap: 2px;
    font-family: var(--font-display);
    font-size: var(--text-lg);
    font-weight: 800;
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.02em;
    line-height: 1.1;
  }

  .stat-unit {
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
  }

  .stat-lbl {
    font-size: var(--text-2xs);
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  /* ── Scrubber / progress bar ─────────────────────────────────────────────── */
  .scrubber-row {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
  }

  .ts-display, .ts-end {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    flex-shrink: 0;
    font-family: var(--font-mono);
  }

  /* Layered progress: transform-scaled fill behind an accessible transparent range input */
  .scrubber-wrap {
    position: relative;
    flex: 1;
    display: flex;
    align-items: center;
    height: 44px;
  }

  .scrubber-track {
    position: absolute;
    left: 0;
    right: 0;
    height: 6px;
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    overflow: hidden;
    pointer-events: none;
  }

  .scrubber-fill {
    position: absolute;
    inset: 0;
    transform-origin: left center;
    transform: scaleX(var(--frac, 0));
    background: linear-gradient(90deg, var(--primary-600), var(--primary-400));
    border-radius: inherit;
    transition: transform 120ms var(--ease-out);
  }

  .scrubber {
    position: relative;
    z-index: 1;
    flex: 1;
    width: 100%;
    height: 44px;
    margin: 0;
    -webkit-appearance: none;
    appearance: none;
    background: transparent;
    cursor: pointer;
    outline: none;
  }

  .scrubber:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 3px;
    border-radius: var(--radius-full);
  }

  .scrubber::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: var(--space-5);
    height: var(--space-5);
    border-radius: var(--radius-full);
    background: var(--text-inverse, white);
    border: 2.5px solid var(--primary-500);
    box-shadow: var(--shadow-sm);
    cursor: pointer;
    transition: transform var(--duration-fast) var(--ease-spring);
  }

  .scrubber::-webkit-slider-thumb:hover {
    transform: scale(1.2);
  }

  .scrubber::-moz-range-thumb {
    width: var(--space-5);
    height: var(--space-5);
    border-radius: var(--radius-full);
    background: var(--text-inverse, white);
    border: 2.5px solid var(--primary-500);
    cursor: pointer;
  }

  /* Play button */
  .play-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    width: 100%;
    min-height: 52px;
    padding: var(--space-3);
    border-radius: var(--radius-lg);
    background: linear-gradient(135deg, var(--primary-500), var(--primary-700));
    border: none;
    color: var(--text-inverse);
    font-family: var(--font-display);
    font-size: var(--text-base);
    font-weight: 700;
    cursor: pointer;
    transition: transform var(--duration-fast) var(--ease-spring),
                box-shadow var(--duration-normal) var(--ease-out),
                background var(--duration-normal) var(--ease-out);
    box-shadow: var(--shadow-primary);
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }

  .play-btn:hover {
    box-shadow: var(--shadow-primary), var(--glow-primary);
  }

  .play-btn:active { transform: scale(0.97); }

  .play-btn:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 3px;
  }

  .play-btn.playing {
    background: var(--surface-2);
    color: var(--text-primary);
    box-shadow: var(--shadow-sm);
  }

  /* ── Reduced motion ──────────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .scrubber-fill { transition: none; }
    .scrubber::-webkit-slider-thumb:hover { transform: none; }
    .play-btn:hover { box-shadow: var(--shadow-primary); }
    .play-btn:active { transform: none; }
  }
</style>
