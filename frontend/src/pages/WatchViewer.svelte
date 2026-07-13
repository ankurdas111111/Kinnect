<script>
  import { onMount, onDestroy } from 'svelte';
  import maplibregl from 'maplibre-gl';
  import { createRealtimeSocket } from '../lib/realtimeClient.js';
  import { createMapIcon, formatCoordinate, escapeAttr } from '../lib/tracking.js';
  import { animateMarkerTo } from '../lib/markerInterpolator.js';
  import { deriveConnState, CONNECTION_TIMEOUT_MS } from '../lib/presence.js';
  import Card from '../components/primitives/Card.svelte';
  import StatusBadge from '../components/primitives/StatusBadge.svelte';
  import SharedLiveShell from '../components/SharedLiveShell.svelte';

  let { params = {} } = $props();

  let map = null;
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

  // Canonical connection state — shared derivation (lib/presence.js).
  // deriveConnState never returns 'sos' (SOS is an orthogonal safety state,
  // surfaced by the sos-active border + banner). This badge only renders in the
  // pre-identity phase, before sosActive can be true.
  let connState = $derived(
    deriveConnState({
      initialized: hasInit,
      online: hasInit,
      issue: bannerSos && !sosActive,
      connecting: !hasInit,
    })
  );

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
    }, CONNECTION_TIMEOUT_MS);
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
    if (!map || !u || typeof u.latitude !== 'number') return;
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

  function connect() {
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
  }

  function handleMap(m) {
    map = m;
    connect();
  }

  onDestroy(() => {
    clearInitTimeout();
    if (socket) socket.disconnect();
    // map lifecycle is owned by SharedLiveShell.
  });
</script>

<SharedLiveShell
  position="top"
  {connState}
  statusLabel={bannerText}
  {sosActive}
  onMap={handleMap}
>
  {#snippet header()}
    <div class="watch-header-inner" role="status" aria-live="polite">
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
        <StatusBadge state={connState} label={bannerText} announce />
      {/if}
    </div>
  {/snippet}

  {#snippet docks()}
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
  {/snippet}
</SharedLiveShell>

<style>
  .watch-header-inner {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
    min-width: 0;
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

  /* Tap-to-call — green pill button in the watch header */
  .watch-call-btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    min-height: 44px;
    padding: 7px 16px;
    background: var(--success-600);
    color: var(--text-inverse, white);
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
    background: var(--success-700);
    box-shadow: var(--glow-live);
  }

  .watch-sos-badge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 5px 12px;
    background: color-mix(in oklch, var(--text-primary) 30%, transparent);
    border: 1px solid color-mix(in oklch, var(--text-inverse, white) 20%, transparent);
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
    animation: slide-up-in 340ms var(--ease-out) both;
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
    color: var(--warning-500);
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
    .watch-sos-badge {
      animation: none;
    }
    .narrative-dock {
      animation: none;
    }
  }
</style>
