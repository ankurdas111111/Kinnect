<script>
  import { onMount, onDestroy } from 'svelte';
  import { push, querystring } from 'svelte-spa-router';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { authUser } from '../lib/stores/auth.js';
  import { otherUsers } from '../lib/stores/map.js';
  import { socket } from '../lib/socket.js';

  $: if (!$authUser) push('/login');

  // ── Parse userId from query string ─────────────────────────────────────────
  $: params = new URLSearchParams($querystring || '');
  $: targetUserId = params.get('userId') || '';
  $: targetUser = targetUserId
    ? (() => { for (const [, u] of $otherUsers) { if (u.userId === targetUserId) return u; } return null; })()
    : null;
  $: displayName = targetUser?.displayName || targetUserId || 'Route';

  // ── Trail data & playback state ─────────────────────────────────────────────
  let points = [];      // [{lat, lng, ts}]
  let loading = true;
  let error = null;
  let windowMinutes = 60;
  const WINDOWS = [15, 30, 60];

  let mapEl;
  let map = null;
  let mapReady = false;

  // Playback
  let playing = false;
  let playProgress = 0;   // 0-100
  let playIndex = 0;
  let playTimer = null;
  let movingMarker = null;

  // Stats
  $: totalPoints = points.length;
  $: totalDistKm = calcDist(points);
  $: durationMin = points.length > 1
    ? Math.round((points[points.length-1].ts - points[0].ts) / 60000)
    : 0;

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
  function requestTrail() {
    if (!targetUserId) { loading = false; error = 'No user specified'; return; }
    loading = true;
    error = null;
    playing = false;
    playProgress = 0;
    playIndex = 0;
    socket.emit('getRecentTrail', { targetUserId, windowMinutes });
  }

  function onTrailData(data) {
    loading = false;
    points = (data.points || data || []).slice().sort((a, b) => a.ts - b.ts);
    if (points.length === 0) { error = 'No position data in this window'; return; }
    renderTrail();
  }

  function onTrailError(data) {
    loading = false;
    error = data?.error || 'Could not load trail';
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

    // Remove old layers/sources
    ['replay-line', 'replay-start', 'replay-end'].forEach(id => {
      if (map.getLayer(id)) map.removeLayer(id);
    });
    ['replay-route', 'replay-start-src', 'replay-end-src'].forEach(id => {
      if (map.getSource(id)) map.removeSource(id);
    });
    if (movingMarker) { movingMarker.remove(); movingMarker = null; }

    // Route line
    map.addSource('replay-route', {
      type: 'geojson',
      data: { type: 'Feature', geometry: { type: 'LineString', coordinates: coords } },
    });
    map.addLayer({
      id: 'replay-line',
      type: 'line',
      source: 'replay-route',
      layout: { 'line-cap': 'round', 'line-join': 'round' },
      paint: { 'line-color': '#6366f1', 'line-width': 4, 'line-opacity': 0.85 },
    });

    // Start dot
    map.addSource('replay-start-src', { type: 'geojson', data: { type: 'Feature', geometry: { type: 'Point', coordinates: coords[0] } } });
    map.addLayer({ id: 'replay-start', type: 'circle', source: 'replay-start-src', paint: { 'circle-radius': 7, 'circle-color': '#10b981', 'circle-stroke-width': 2.5, 'circle-stroke-color': '#fff' } });

    // End dot
    map.addSource('replay-end-src', { type: 'geojson', data: { type: 'Feature', geometry: { type: 'Point', coordinates: coords[coords.length - 1] } } });
    map.addLayer({ id: 'replay-end', type: 'circle', source: 'replay-end-src', paint: { 'circle-radius': 7, 'circle-color': '#ef4444', 'circle-stroke-width': 2.5, 'circle-stroke-color': '#fff' } });

    // Fit bounds
    const lats = points.map(p => p.lat), lngs = points.map(p => p.lng);
    map.fitBounds([[Math.min(...lngs), Math.min(...lats)], [Math.max(...lngs), Math.max(...lats)]], { padding: 48 });

    // Create moving marker
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
      // Instant: jump to end
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

  // ── Timestamp display ───────────────────────────────────────────────────────
  $: currentTs = points[playIndex]?.ts;
  function fmtTs(ts) {
    if (!ts) return '--:--';
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }

  // ── Lifecycle ───────────────────────────────────────────────────────────────
  onMount(() => {
    socket.on('recentTrail', onTrailData);
    socket.on('trailError', onTrailError);
    initMap().then(() => requestTrail());
  });

  onDestroy(() => {
    clearTimeout(playTimer);
    socket.off('recentTrail', onTrailData);
    socket.off('trailError', onTrailError);
    if (map) map.remove();
  });

  $: if (windowMinutes) { /* reactive re-request handled by setWindow() */ }
</script>

<svelte:head>
  <link rel="stylesheet" href="https://unpkg.com/maplibre-gl/dist/maplibre-gl.css" />
  <style>
    .moving-dot {
      width: 18px;
      height: 18px;
      background: white;
      border: 3px solid #6366f1;
      border-radius: 50%;
      box-shadow: 0 0 12px rgba(99,102,241,0.7), 0 2px 8px rgba(0,0,0,0.35);
    }
  </style>
</svelte:head>

<div class="replay-page page-enter aurora-ambient">
  <!-- Header -->
  <header class="rp-header">
    <button class="icon-btn" on:click={() => push('/')} aria-label="Back to map">
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
          class="window-pill"
          class:active={windowMinutes === w}
          on:click={() => setWindow(w)}
          aria-pressed={windowMinutes === w}
        >{w}m</button>
      {/each}
    </div>
  </header>

  <!-- Map -->
  <div class="map-wrap" bind:this={mapEl}>
    {#if loading}
      <div class="map-overlay" transition:fade={{ duration: 200 }}>
        <div class="spinner" aria-label="Loading trail data"></div>
        <p>Loading route…</p>
      </div>
    {:else if error === 'No user specified'}
      <div class="map-overlay" transition:fade={{ duration: 200 }}>
        <div class="empty-icon" aria-hidden="true">
          <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M3 12h18M12 3l9 9-9 9-9-9 9-9z"/></svg>
        </div>
        <p class="empty-title">Choose someone to replay</p>
        <p class="empty-sub">Open a person's card on the map and tap <strong>Route History</strong> to see their journey here.</p>
        <button class="retry-btn" on:click={() => push('/')}>Open Map</button>
      </div>
    {:else if error}
      <div class="map-overlay" transition:fade={{ duration: 200 }}>
        <div class="err-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><circle cx="12" cy="16" r="0.5" fill="currentColor"/></svg>
        </div>
        <p class="err-msg">{error}</p>
        <button class="retry-btn" on:click={requestTrail}>Retry</button>
      </div>
    {/if}
  </div>

  <!-- Controls panel -->
  {#if !loading && !error && points.length > 0}
    <div class="controls-panel" transition:fly={{ y: 200, duration: 320, easing: cubicOut }}>
      <!-- Stats row -->
      <div class="stats-row">
        <div class="stat-chip">
          <span class="stat-val">{totalPoints}</span>
          <span class="stat-lbl">points</span>
        </div>
        <div class="stat-chip">
          <span class="stat-val">{totalDistKm} km</span>
          <span class="stat-lbl">distance</span>
        </div>
        <div class="stat-chip">
          <span class="stat-val">{durationMin} min</span>
          <span class="stat-lbl">duration</span>
        </div>
      </div>

      <!-- Scrubber + timestamp -->
      <div class="scrubber-row">
        <time class="ts-display" aria-live="polite" aria-label="Current time: {fmtTs(currentTs)}">{fmtTs(currentTs)}</time>
        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value={playProgress}
          style="--val: {playProgress}%"
          on:input={onScrub}
          class="scrubber"
          aria-label="Playback position"
          aria-valuemin="0"
          aria-valuemax="100"
          aria-valuenow={playProgress}
        />
        <span class="ts-end">{fmtTs(points[points.length-1]?.ts)}</span>
      </div>

      <!-- Play/Pause button -->
      <button
        class="play-btn"
        class:playing
        on:click={togglePlay}
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
    background: #0a0a14;
    color: var(--text-primary);
    font-family: var(--font-sans);
    overflow: hidden;
  }

  /* Header */
  .rp-header {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: calc(env(safe-area-inset-top, 0px) + 10px) 14px 10px;
    background: rgba(5, 5, 18, 0.94);
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    border-bottom: 1px solid rgba(255,255,255,0.07);
    z-index: 100;
    flex-shrink: 0;
  }

  .icon-btn {
    width: 38px; height: 38px;
    display: flex; align-items: center; justify-content: center;
    background: rgba(255,255,255,0.07);
    border: 1px solid rgba(255,255,255,0.10);
    border-radius: 50%;
    color: var(--text-primary);
    cursor: pointer;
    flex-shrink: 0;
    transition: background 150ms, transform 100ms;
    -webkit-tap-highlight-color: transparent;
  }
  .icon-btn:hover { background: rgba(255,255,255,0.12); }
  .icon-btn:active { transform: scale(0.88); }

  .rp-title-group { flex: 1; min-width: 0; }
  .rp-title {
    margin: 0;
    font-family: var(--font-display);
    font-size: 18px;
    font-weight: 700;
    letter-spacing: -0.02em;
    line-height: 1.2;
  }
  .rp-sub {
    font-size: 12px;
    color: var(--text-tertiary);
    display: block;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .window-pills {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }
  .window-pill {
    padding: 4px 10px;
    border-radius: 999px;
    font-size: 12px;
    font-weight: 600;
    background: rgba(255,255,255,0.06);
    border: 1px solid rgba(255,255,255,0.10);
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 150ms;
    -webkit-tap-highlight-color: transparent;
  }
  .window-pill:hover { background: rgba(255,255,255,0.10); }
  .window-pill.active {
    background: rgba(99,102,241,0.20);
    border-color: rgba(99,102,241,0.40);
    color: var(--primary-300, #a5b4fc);
  }

  /* Map */
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
    gap: 12px;
    background: rgba(10, 10, 20, 0.85);
    z-index: 10;
    font-size: 14px;
    color: var(--text-secondary);
  }

  .spinner {
    width: 36px; height: 36px;
    border: 3px solid rgba(255,255,255,0.12);
    border-top-color: var(--primary-500, #6366f1);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .empty-icon {
    width: 72px; height: 72px;
    border-radius: 50%;
    background: rgba(99,102,241,0.1);
    display: flex; align-items: center; justify-content: center;
    color: var(--primary-400, #818cf8);
    margin-bottom: 4px;
  }
  .empty-title { font-size: 16px; font-weight: 700; color: var(--text-primary); margin: 0; }
  .empty-sub { font-size: 13px; color: var(--text-secondary); text-align: center; max-width: 220px; margin: 0; line-height: 1.5; }

  .err-icon {
    width: 56px; height: 56px;
    border-radius: 50%;
    background: rgba(239,68,68,0.12);
    display: flex; align-items: center; justify-content: center;
    color: var(--danger-400, #f87171);
  }
  .err-msg { font-size: 14px; color: var(--text-secondary); text-align: center; max-width: 200px; }
  .retry-btn {
    padding: 8px 20px;
    border-radius: 999px;
    background: rgba(99,102,241,0.18);
    border: 1px solid rgba(99,102,241,0.35);
    color: var(--primary-300, #a5b4fc);
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: background 150ms;
  }
  .retry-btn:hover { background: rgba(99,102,241,0.28); }

  /* Controls panel */
  .controls-panel {
    background: rgba(5, 5, 18, 0.96);
    backdrop-filter: blur(32px) saturate(1.8);
    -webkit-backdrop-filter: blur(32px) saturate(1.8);
    border-top: 1px solid rgba(255,255,255,0.07);
    padding: 14px 16px calc(14px + env(safe-area-inset-bottom, 0px));
    display: flex;
    flex-direction: column;
    gap: 12px;
    flex-shrink: 0;
  }

  .stats-row {
    display: flex;
    gap: 10px;
    justify-content: center;
  }
  .stat-chip {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1px;
    background: rgba(255,255,255,0.05);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 10px;
    padding: 8px 14px;
    min-width: 72px;
  }
  .stat-val {
    font-family: var(--font-display);
    font-size: 16px;
    font-weight: 700;
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
  }
  .stat-lbl { font-size: 10px; color: var(--text-tertiary); text-transform: uppercase; letter-spacing: 0.06em; }

  .scrubber-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .ts-display, .ts-end {
    font-size: 11px;
    color: var(--text-tertiary);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    flex-shrink: 0;
    font-family: var(--font-mono, monospace);
  }
  .scrubber {
    flex: 1;
    height: 4px;
    -webkit-appearance: none;
    appearance: none;
    border-radius: 999px;
    background: linear-gradient(to right, var(--primary-500, #6366f1) var(--val, 0%), rgba(255,255,255,0.12) var(--val, 0%));
    cursor: pointer;
    outline: none;
  }
  .scrubber::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 18px; height: 18px;
    border-radius: 50%;
    background: white;
    border: 2.5px solid var(--primary-500, #6366f1);
    box-shadow: 0 0 8px rgba(99,102,241,0.55);
    cursor: pointer;
  }
  .scrubber::-moz-range-thumb {
    width: 18px; height: 18px;
    border-radius: 50%;
    background: white;
    border: 2.5px solid var(--primary-500, #6366f1);
    cursor: pointer;
  }

  .play-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    width: 100%;
    padding: 13px;
    border-radius: 14px;
    background: linear-gradient(135deg, var(--primary-500, #6366f1), var(--primary-700, #4338ca));
    border: none;
    color: white;
    font-family: var(--font-display);
    font-size: 16px;
    font-weight: 700;
    cursor: pointer;
    transition: transform 120ms, box-shadow 150ms;
    box-shadow: 0 4px 20px rgba(99,102,241,0.40);
    -webkit-tap-highlight-color: transparent;
  }
  .play-btn:active { transform: scale(0.97); }
  .play-btn.playing {
    background: linear-gradient(135deg, #475569, #334155);
    box-shadow: 0 4px 16px rgba(0,0,0,0.30);
  }

  @media (prefers-reduced-motion: reduce) {
    .spinner { animation: none; border-top-color: var(--primary-500); }
  }
</style>
