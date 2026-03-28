<script>
  import { onMount, onDestroy } from 'svelte';
  import maplibregl from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';
  import { createRealtimeSocket } from '../lib/realtimeClient.js';
  import { createMapIcon, formatCoordinate, escapeAttr } from '../lib/tracking.js';
  import { animateMarkerTo } from '../lib/markerInterpolator.js';
  import { MAP_STYLE } from '../lib/mapStyle.js';

  export let params = {};

  let mapContainer;
  let map;
  let marker = null;
  let markerPopup = null;
  let socket = null;
  let bannerText = 'Connecting...';
  let bannerSos = false;
  let sosActive = false;
  let hasInit = false;
  let initTimeout = null;
  let watchedName = '';
  let watchedPhone = '';

  function clearInitTimeout() {
    if (initTimeout) {
      clearTimeout(initTimeout);
      initTimeout = null;
    }
  }

  function scheduleInitTimeout() {
    clearInitTimeout();
    initTimeout = setTimeout(() => {
      if (!hasInit) {
        bannerText = 'Unable to load watch session. The link may be invalid, expired, or unavailable.';
        bannerSos = true;
      }
    }, 8000);
  }

  $: token = params.token || '';

  function setBanner(sos) {
    if (!sos?.active) { bannerText = 'Watch link connected.'; bannerSos = false; sosActive = false; return; }
    const count = typeof sos.ackCount === 'number' ? sos.ackCount : (sos.acks?.length || 0);
    const who = count ? `Acknowledged (${count})` : 'Not yet acknowledged';
    bannerText = `SOS: ${sos.reason || 'SOS'} - ${who}`;
    bannerSos = true;
    sosActive = true;
    try { if (navigator.vibrate) navigator.vibrate([200, 100, 200]); } catch (_) {}
  }

  let followMode = true;
  let sosNarrative = null;

  function update(u) {
    if (!u || typeof u.latitude !== 'number') return;
    watchedName = u.displayName || 'User';
    watchedPhone = u.mobile || '';
    const lngLat = [u.longitude, u.latitude];
    const popupHtml = `<strong>${escapeAttr(u.displayName || 'User')}</strong><br>Lat: ${formatCoordinate(u.latitude)}<br>Lng: ${formatCoordinate(u.longitude)}<br>Speed: ${u.speed || '0'} km/h`;
    if (!marker) {
      const el = createMapIcon('var(--danger-500)', '', { pulse: true, markerType: 'sos' });
      markerPopup = new maplibregl.Popup({ offset: [0, -44], maxWidth: '280px' }).setHTML(popupHtml);
      marker = new maplibregl.Marker({ element: el, anchor: 'bottom' })
        .setLngLat(lngLat).setPopup(markerPopup).addTo(map);
      map.jumpTo({ center: lngLat, zoom: 16 });
    } else {
      animateMarkerTo('watch-target', marker, lngLat, 300);
      if (markerPopup) markerPopup.setHTML(popupHtml);
      if (followMode) {
        const bounds = map.getBounds();
        if (!bounds.contains(lngLat)) {
          map.easeTo({ center: lngLat, duration: 500 });
        }
      }
    }
  }

  onMount(() => {
    map = new maplibregl.Map({ container: mapContainer, style: MAP_STYLE, center: [78.9629, 20.5937], zoom: 4, attributionControl: true });
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'bottom-right');

    if (!token) {
      bannerText = 'Invalid watch link.';
      bannerSos = true;
      return;
    }

    socket = createRealtimeSocket({ auth: { viewer: true } });
    socket.on('connect', () => { scheduleInitTimeout(); socket.emit('watchJoin', { token }); });
    socket.on('connect_error', () => { if (!hasInit) scheduleInitTimeout(); bannerText = 'Connection error. Retrying...'; bannerSos = true; });
    socket.on('disconnect', () => { if (!hasInit) scheduleInitTimeout(); bannerText = 'Disconnected. Reconnecting...'; bannerSos = true; });
    socket.on('watchInit', (payload) => {
      hasInit = true; clearInitTimeout(); bannerText = 'Connected.';
      update(payload?.user); setBanner(payload?.sos);
      if (payload?.sos?.narrative) sosNarrative = payload.sos.narrative;
    });
    socket.on('watchUpdate', (payload) => {
      update(payload?.user); setBanner(payload?.sos);
      if (payload?.sos?.narrative) sosNarrative = payload.sos.narrative;
    });
    socket.on('sosNarrative', (data) => {
      if (data?.narrative) sosNarrative = data.narrative;
    });
  });

  onDestroy(() => {
    clearInitTimeout();
    if (socket) socket.disconnect();
    if (map) map.remove();
  });
</script>

<div class="watch-page" class:sos-active={sosActive}>
  <div class="watch-map" bind:this={mapContainer}></div>
  <div class="watch-header" class:sos-state={bannerSos} role="status" aria-live="polite">
    {#if hasInit && watchedName}
      <div class="watch-identity">
        <span class="watch-eyebrow">Watching</span>
        <h1 class="watch-name">{watchedName}</h1>
      </div>
      {#if watchedPhone}
        <a href="tel:{watchedPhone}" class="watch-call-btn" aria-label="Call {watchedName}">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M6.62 10.79a15.05 15.05 0 0 0 6.59 6.59l2.2-2.2a1 1 0 0 1 1.01-.24 11.36 11.36 0 0 0 3.56.57 1 1 0 0 1 1 1V20a1 1 0 0 1-1 1A17 17 0 0 1 3 4a1 1 0 0 1 1-1h3.5a1 1 0 0 1 1 1c0 1.25.2 2.45.57 3.56a1 1 0 0 1-.25 1.02z"/></svg>
          Call
        </a>
      {/if}
      {#if sosActive}
        <div class="watch-sos-badge">
          <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/></svg>
          SOS Active
        </div>
      {/if}
    {:else}
      {#if bannerSos}
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
      {/if}
      <span class="watch-status-text">{bannerText}</span>
    {/if}
  </div>
  {#if sosActive && sosNarrative}
    <div class="narrative-panel">
      <span class="narrative-eyebrow">Crisis Context</span>
      {#if sosNarrative.motionSummary}
        <div class="narrative-row">
          <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          {sosNarrative.motionSummary}
        </div>
      {/if}
      {#if sosNarrative.batteryPct != null}
        <div class="narrative-row">
          <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="2" y="7" width="18" height="10" rx="1"/><line x1="22" y1="11" x2="22" y2="13"/></svg>
          Battery {sosNarrative.batteryPct}%
        </div>
      {/if}
      {#if sosNarrative.triggerRule && sosNarrative.triggerRule !== 'manual'}
        <div class="narrative-row trigger">Auto-triggered: {sosNarrative.triggerRule}</div>
      {/if}
    </div>
  {/if}

  <div class="bottom-controls">
    <button class="btn btn-sm" class:btn-primary={followMode} class:btn-secondary={!followMode} on:click={() => followMode = !followMode} aria-label={followMode ? 'Disable auto-follow' : 'Enable auto-follow'}>
      {followMode ? 'Following' : 'Follow'}
    </button>
    <a href="/#/register" class="btn btn-sm btn-secondary" aria-label="Sign up for Kinnect">Sign up for Kinnect</a>
  </div>
</div>

<style>
  .watch-page {
    position: relative;
    width: 100%;
    height: 100vh;
    height: 100dvh;
  }

  .watch-page.sos-active {
    outline: 3px solid var(--danger-500);
    outline-offset: -3px;
    animation: sos-border-pulse 2s ease-in-out infinite;
  }

  @keyframes sos-border-pulse {
    0%, 100% { outline-color: var(--danger-500); }
    50% { outline-color: transparent; }
  }

  .watch-map { position: absolute; inset: 0; z-index: 1; }

  /* MERIDIAN: Glass header with large name display */
  .watch-header {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: 50;
    padding: calc(var(--space-3) + env(safe-area-inset-top, 0px)) var(--space-5) var(--space-3);
    background: rgba(8, 8, 16, 0.78);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border-bottom: 1px solid rgba(255, 255, 255, 0.07);
    color: white;
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-height: 64px;
    transition: background 400ms var(--ease-out);
  }

  .watch-header.sos-state {
    background: rgba(220, 38, 38, 0.88);
    border-bottom-color: rgba(255, 255, 255, 0.14);
    box-shadow: 0 4px 28px rgba(220, 38, 38, 0.40);
  }

  .watch-identity {
    flex: 1;
    min-width: 0;
  }

  .watch-eyebrow {
    display: block;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    opacity: 0.60;
    margin-bottom: 2px;
  }

  .watch-name {
    font-size: 32px;
    font-weight: 900;
    letter-spacing: -0.03em;
    margin: 0;
    line-height: 1.1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* MERIDIAN: Tap-to-call — green pill button in the watch header */
  .watch-call-btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 7px 14px;
    background: #16a34a;
    color: white;
    border-radius: var(--radius-full);
    font-size: 13px;
    font-weight: 700;
    text-decoration: none;
    white-space: nowrap;
    flex-shrink: 0;
    box-shadow: 0 0 18px rgba(22, 163, 74, 0.45);
    transition: background 150ms, box-shadow 150ms;
  }
  .watch-call-btn:hover {
    background: #15803d;
    box-shadow: 0 0 24px rgba(22, 163, 74, 0.60);
  }

  .watch-sos-badge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 5px 12px;
    background: rgba(0, 0, 0, 0.30);
    border: 1px solid rgba(255, 255, 255, 0.20);
    border-radius: var(--radius-full);
    font-size: 11px;
    font-weight: 700;
    white-space: nowrap;
    flex-shrink: 0;
    animation: sos-badge-pulse 2s ease-in-out infinite;
  }

  @keyframes sos-badge-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.65; }
  }

  .watch-status-text {
    font-size: var(--text-sm);
    font-weight: 600;
    opacity: 0.88;
  }

  .bottom-controls {
    position: absolute;
    bottom: var(--space-4);
    left: 50%;
    transform: translateX(-50%);
    z-index: 50;
    display: flex;
    gap: var(--space-2);
    box-shadow: var(--shadow-lg);
  }
  .bottom-controls a {
    text-decoration: none;
  }

  @media (min-width: 768px) {
    .bottom-controls {
      left: auto;
      right: var(--space-4);
      transform: none;
    }
  }

  .narrative-panel {
    position: absolute;
    bottom: calc(var(--space-4) + 48px);
    left: var(--space-3);
    right: var(--space-3);
    z-index: 50;
    background: rgba(8, 8, 16, 0.88);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border: 1px solid rgba(239, 68, 68, 0.22);
    border-radius: var(--radius-lg);
    padding: var(--space-3) var(--space-4);
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .narrative-eyebrow {
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.10em;
    text-transform: uppercase;
    color: var(--danger-500);
    opacity: 0.8;
  }

  .narrative-row {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: rgba(255,255,255,0.75);
    font-weight: 500;
  }

  .narrative-row.trigger {
    color: #f59e0b;
  }

  @media (min-width: 768px) {
    .narrative-panel {
      left: auto;
      right: var(--space-4);
      width: 260px;
      bottom: calc(var(--space-4) + 48px);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .watch-page.sos-active {
      animation: none;
      outline-color: var(--danger-500);
    }
  }
</style>
