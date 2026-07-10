<script>
  import { onMount, onDestroy } from 'svelte';
  import maplibregl from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';
  import { createRealtimeSocket } from '../lib/realtimeClient.js';
  import { createMapIcon, formatCoordinate, escapeAttr } from '../lib/tracking.js';
  import { animateMarkerTo } from '../lib/markerInterpolator.js';
  import { MAP_STYLE } from '../lib/mapStyle.js';
  import Card from '../components/primitives/Card.svelte';

  let { params = {} } = $props();

  let mapContainer = $state();
  let map;
  let marker = null;
  let markerPopup = null;
  let socket = null;
  let bannerText = $state('Connecting...');
  let bannerSos = $state(false);
  let sosActive = $state(false);
  let hasInit = $state(false);
  let initTimeout = null;
  let watchedName = $state('');
  let watchedPhone = $state('');

  // Animated status badge state — calm colour by connection state.
  let connState = $derived.by(() => {
    if (sosActive) return 'sos';
    if (bannerSos) return 'issue';
    if (hasInit) return 'live';
    return 'connecting';
  });

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

  let token = $derived(params.token || '');

  function setBanner(sos) {
    if (!sos?.active) { bannerText = 'Watch link connected.'; bannerSos = false; sosActive = false; return; }
    const count = typeof sos.ackCount === 'number' ? sos.ackCount : (sos.acks?.length || 0);
    const who = count ? `Acknowledged (${count})` : 'Not yet acknowledged';
    bannerText = `SOS: ${sos.reason || 'SOS'} - ${who}`;
    bannerSos = true;
    sosActive = true;
    try { if (navigator.vibrate) navigator.vibrate([200, 100, 200]); } catch (_) {}
  }

  let followMode = $state(true);
  let sosNarrative = $state(null);

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
      <span class="watch-status-badge" data-state={connState}>
        <span class="sdot" aria-hidden="true"></span>
        <span class="watch-status-text">{bannerText}</span>
      </span>
    {/if}
  </div>
  {#if sosActive && sosNarrative}
    <div class="narrative-dock">
      <Card variant="glass" glow="danger" padding="md" hover={false}>
        <div class="narrative-body">
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
      </Card>
    </div>
  {/if}

  <div class="bottom-controls">
    <button class="btn btn-sm" class:btn-primary={followMode} class:btn-secondary={!followMode} onclick={() => followMode = !followMode} aria-label={followMode ? 'Disable auto-follow' : 'Enable auto-follow'}>
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
    font-size: clamp(1.5rem, 2.4vw, 2rem);
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
    min-height: 44px;
    padding: 7px 16px;
    background: var(--success-600, #16a34a);
    color: white;
    border-radius: var(--radius-full);
    font-size: 13px;
    font-weight: 700;
    text-decoration: none;
    white-space: nowrap;
    flex-shrink: 0;
    box-shadow: var(--glow-live-sm);
    transition: background 150ms, box-shadow 150ms;
  }
  .watch-call-btn:hover {
    background: var(--success-700, #15803d);
    box-shadow: var(--glow-live);
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

  /* Animated status badge with state-driven glow (pre-identity states) */
  .watch-status-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1-5);
    min-width: 0;
    padding: 8px 14px;
    border-radius: var(--radius-full);
    color: white;
    transition:
      background 320ms var(--ease-out),
      box-shadow 320ms var(--ease-out);
  }

  .watch-status-text {
    font-size: var(--text-sm);
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sdot {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    background: currentColor;
    flex-shrink: 0;
  }

  .watch-status-badge[data-state="connecting"] {
    background: var(--primary-500-12);
    box-shadow: var(--glow-primary-sm);
  }
  .watch-status-badge[data-state="connecting"] .sdot {
    background: var(--primary-300);
    animation: badge-blink 1s ease-in-out infinite;
  }

  .watch-status-badge[data-state="live"] {
    background: var(--success-500-20);
    box-shadow: var(--glow-live-sm);
  }
  .watch-status-badge[data-state="live"] .sdot {
    background: var(--success-400);
    animation: badge-pulse 1.8s ease-in-out infinite;
  }

  .watch-status-badge[data-state="issue"] {
    background: var(--danger-500-20);
    box-shadow: var(--glow-sos-sm);
  }
  .watch-status-badge[data-state="issue"] .sdot {
    background: var(--danger-400);
    animation: badge-blink 0.85s ease-in-out infinite;
  }

  .watch-status-badge[data-state="sos"] {
    background: var(--danger-500-20);
    box-shadow: var(--glow-sos-sm);
  }
  .watch-status-badge[data-state="sos"] .sdot {
    background: var(--danger-400);
    animation: badge-pulse 1s ease-in-out infinite;
  }

  @keyframes badge-pulse {
    0%, 100% { opacity: 1; transform: scale(1); }
    50% { opacity: 0.55; transform: scale(0.82); }
  }
  @keyframes badge-blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.25; }
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

  /* SOS crisis-context dock — danger-glow Card wrapper */
  .narrative-dock {
    position: absolute;
    bottom: calc(var(--space-4) + 48px);
    left: var(--space-3);
    right: var(--space-3);
    z-index: 50;
    animation: slide-up-in 340ms var(--ease-spring) both;
  }

  .narrative-body {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .narrative-eyebrow {
    font-size: 10px;
    font-weight: 800;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--danger-400);
  }

  .narrative-row {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: var(--text-secondary);
    font-weight: 500;
  }

  .narrative-row.trigger {
    color: var(--warning-500, #f59e0b);
    font-weight: 600;
  }

  @media (min-width: 768px) {
    .narrative-dock {
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
    .watch-status-badge .sdot,
    .watch-sos-badge {
      animation: none;
    }
    .narrative-dock {
      animation: none;
    }
  }
</style>
