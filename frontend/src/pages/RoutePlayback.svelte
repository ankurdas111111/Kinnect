<script>
  import { run } from 'svelte/legacy';

  import { onMount, onDestroy } from 'svelte';
  import { push, querystring } from 'svelte-spa-router';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { authUser } from '../lib/stores/auth.js';
  import { otherUsers } from '../lib/stores/map.js';
  import { socket } from '../lib/socket.js';
  import { createMapPaint } from '../lib/mapPaint.js';
  import { prefersReducedMotion } from '../lib/deviceCapability.js';
  import { haptics } from '../lib/haptics.js';
  import Skeleton from '../components/primitives/Skeleton.svelte';
  import StatCard from '../components/primitives/StatCard.svelte';
  import EmptyState from '../components/primitives/EmptyState.svelte';
  import PlaybackControls from '../components/primitives/PlaybackControls.svelte';



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
  let playProgress = $state(0);   // 0–100 (drives marker + fill)
  let playIndex = $state(0);
  let playSpeed = $state(1);       // multiplier — divides tick delay
  const SPEEDS = [1, 2, 4];
  let playTimer = null;
  let movingMarker = null;

  // ── Map paint tokens (JS strings can't read CSS vars — resolve once, rebuild on theme) ──
  const paint = createMapPaint(() => repaintTokens());


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
    // Self-host the stylesheet (bundled) — never the unpkg CDN, which white-screens the page when blocked.
    await import('maplibre-gl/dist/maplibre-gl.css');
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

  // Re-apply token colors to already-rendered layers when the theme changes.
  function repaintTokens() {
    if (!map || !mapReady) return;
    if (map.getLayer('replay-line')) map.setPaintProperty('replay-line', 'line-color', paint.colors.route);
    if (map.getLayer('replay-start')) {
      map.setPaintProperty('replay-start', 'circle-color', paint.colors.start);
      map.setPaintProperty('replay-start', 'circle-stroke-color', paint.colors.stroke);
    }
    if (map.getLayer('replay-end')) {
      map.setPaintProperty('replay-end', 'circle-color', paint.colors.end);
      map.setPaintProperty('replay-end', 'circle-stroke-color', paint.colors.stroke);
    }
    if (movingMarker) {
      const el = movingMarker.getElement();
      if (el) {
        el.style.borderColor = paint.colors.dot;
        el.style.setProperty('--dot-glow', paint.colors.dot);
      }
    }
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
      paint: { 'line-color': paint.colors.route, 'line-width': 4, 'line-opacity': 0.85 },
    });

    map.addSource('replay-start-src', { type: 'geojson', data: { type: 'Feature', geometry: { type: 'Point', coordinates: coords[0] } } });
    map.addLayer({ id: 'replay-start', type: 'circle', source: 'replay-start-src', paint: { 'circle-radius': 7, 'circle-color': paint.colors.start, 'circle-stroke-width': 2.5, 'circle-stroke-color': paint.colors.stroke } });

    map.addSource('replay-end-src', { type: 'geojson', data: { type: 'Feature', geometry: { type: 'Point', coordinates: coords[coords.length - 1] } } });
    map.addLayer({ id: 'replay-end', type: 'circle', source: 'replay-end-src', paint: { 'circle-radius': 7, 'circle-color': paint.colors.end, 'circle-stroke-width': 2.5, 'circle-stroke-color': paint.colors.stroke } });

    const lats = points.map(p => p.lat), lngs = points.map(p => p.lng);
    map.fitBounds([[Math.min(...lngs), Math.min(...lats)], [Math.max(...lngs), Math.max(...lats)]], { padding: 48 });

    const el = document.createElement('div');
    el.className = 'moving-dot';
    // Token-sourced colors set inline — the imperative marker element can't inherit the component style block.
    el.style.borderColor = paint.colors.dot;
    el.style.setProperty('--dot-glow', paint.colors.dot);
    const { default: ml } = await import('maplibre-gl');
    movingMarker = new ml.Marker({ element: el }).setLngLat(coords[0]).addTo(map);
    playIndex = 0;
    playProgress = 0;
  }

  // ── Playback ─────────────────────────────────────────────────────────────────
  // Note: marker stepping is discrete setLngLat — there is no easing to reduce,
  // so playback runs for reduced-motion users too. The scrubber is the always
  // motion-free path (drag to any point). No end-jump shortcut.
  function play() {
    if (points.length < 2) return;
    playing = true;
    if (playIndex >= points.length - 1) { playIndex = 0; playProgress = 0; }
    tick();
  }

  function pause() {
    playing = false;
    clearTimeout(playTimer);
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
    const base = Math.max(40, Math.min(300, 8000 / points.length));
    const delay = base / playSpeed;
    playTimer = setTimeout(tick, delay);
  }

  // PlaybackControls callback contract ------------------------------------------
  function handlePlay() { haptics.tap(); play(); }
  function handlePause() { haptics.tap(); pause(); }

  function handleScrub({ progress }) {
    clearTimeout(playTimer);
    playing = false;
    const pct = Math.round(progress * 100);
    playProgress = pct;
    playIndex = Math.round((pct / 100) * (points.length - 1));
    if (movingMarker && points[playIndex]) {
      movingMarker.setLngLat([points[playIndex].lng, points[playIndex].lat]);
    }
  }

  function handleSpeedChange({ speed }) {
    playSpeed = speed;
    // If mid-playback, the next tick picks up the new speed automatically.
  }

  function setWindow(w) {
    windowMinutes = w;
    requestTrail();
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
    paint.destroy();
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
  let currentTs = $derived(points[playIndex]?.ts ?? null);
  let endTs = $derived(points[points.length-1]?.ts ?? null);
  let totalDurationMs = $derived(points.length > 1 ? (points[points.length-1].ts - points[0].ts) : 0);
  // Fraction (0–1) for the PlaybackControls scrubber
  let playFrac = $derived(Math.max(0, Math.min(1, playProgress / 100)));
  // Gate the panel entrance transition — JS-driven transitions ignore the CSS
  // reduced-motion query, so we branch here.
  let panelMotion = $derived(prefersReducedMotion() ? { y: 0, duration: 0 } : { y: 200, duration: 320, easing: cubicOut });
</script>

<div class="replay-page page-enter">
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
        <EmptyState
          title="Choose someone to replay"
          body="Open a person's card on the map and tap Route History to see their journey here."
          tone="primary"
        >
          {#snippet icon()}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" aria-hidden="true"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
          {/snippet}
          {#snippet action()}
            <button class="action-btn" onclick={() => push('/')}>Open Map</button>
          {/snippet}
        </EmptyState>
      </div>
    {:else if error}
      <div class="map-overlay" transition:fade={{ duration: 200 }}>
        <EmptyState title={error} tone="danger">
          {#snippet icon()}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><circle cx="12" cy="16" r="0.5" fill="currentColor"/></svg>
          {/snippet}
          {#snippet action()}
            <button class="action-btn" onclick={requestTrail}>Try again</button>
          {/snippet}
        </EmptyState>
      </div>
    {/if}
  </div>

  <!-- Controls panel -->
  {#if !loading && !error && points.length > 0}
    <div class="controls-panel" transition:fly={panelMotion}>
      <!-- Stats bento -->
      <div class="stats-bento bento-grid" style="--bento-cols: 3;" role="region" aria-label="Route statistics">
        <StatCard label="Distance" value={totalDistKm} unit="km" tint="primary">
          {#snippet icon()}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="6" cy="19" r="3"/><path d="M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15"/><circle cx="18" cy="5" r="3"/></svg>
          {/snippet}
        </StatCard>
        <StatCard label="Duration" value={durationMin} unit="min" tint="primary">
          {#snippet icon()}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><polyline points="12 7 12 12 15 14"/></svg>
          {/snippet}
        </StatCard>
        <StatCard label="Points" value={totalPoints} tint="neutral">
          {#snippet icon()}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="5" cy="12" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="19" cy="12" r="1.6"/></svg>
          {/snippet}
        </StatCard>
      </div>

      <!-- Scrubber + transport — shared primitive in local-playback mode -->
      <PlaybackControls
        {playing}
        progress={playFrac}
        duration={totalDurationMs}
        speed={playSpeed}
        speeds={SPEEDS}
        timestamps={{ current: currentTs, end: endTs }}
        onplay={handlePlay}
        onpause={handlePause}
        onscrub={handleScrub}
        onspeedchange={handleSpeedChange}
      />
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
    padding: calc(var(--safe-top) + var(--space-2-5)) var(--space-3-5) var(--space-2-5);
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

  /* Token-sourced moving marker — border-color + glow are set inline from mapPaint,
     so the class only owns the static geometry. */
  :global(.moving-dot) {
    width: 18px;
    height: 18px;
    background: var(--surface-0);
    border: 3px solid var(--primary-400);
    border-radius: 50%;
    box-shadow: 0 0 12px var(--dot-glow, var(--primary-400)), var(--shadow-sm);
  }

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
    -webkit-tap-highlight-color: transparent;
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
    padding: var(--space-3-5) var(--space-4) calc(var(--space-3-5) + var(--safe-bottom));
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    flex-shrink: 0;
  }

  /* ── Stats bento ─────────────────────────────────────────────────────────── */
  .stats-bento {
    --bento-gap: var(--space-2-5);
  }

  /* PlaybackControls owns its own internal padding; strip the panel's default so
     the shared bar aligns flush with the stats row above it. */
  .controls-panel :global(.playback-controls) {
    padding: 0;
  }

  /* ── Reduced motion ──────────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .icon-btn:active { transform: none; }
  }
</style>
