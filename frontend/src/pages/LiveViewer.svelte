<script>
  import { onMount, onDestroy } from 'svelte';
  import maplibregl from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';
  import { createRealtimeSocket } from '../lib/realtimeClient.js';
  import { createMapIcon, escapeAttr, escHtml } from '../lib/tracking.js';
  import { animateMarkerTo } from '../lib/markerInterpolator.js';
  import { MAP_STYLE } from '../lib/mapStyle.js';

  export let params = {};

  let mapContainer;
  let map;
  let marker = null;
  let hasZoomed = false;
  let socket = null;
  let viewerName = '';
  let showNameOverlay = true;
  let statusText = 'Connecting...';
  let online = false;
  let sharedBy = 'User';
  let sosActive = false;
  let sosInfo = '';
  let sosAcks = '';
  let sosAcked = false;
  let expired = false;
  let checkinText = '';
  let checkinOverdue = false;
  let sosAudioInterval = null;
  let audioCtx = null;
  let isMobile = false;
  let hasInit = false;
  let initTimeout = null;
  let connectionIssue = '';
  let lastOriginTs = null;
  let freshnessText = '';
  let freshnessInterval = null;
  let linkJoinedAt = null;
  let linkExpiresAt = null;
  let expiryPercent = 100;

  $: token = params.token || '';

  function formatTimeAgo(ts) {
    if (!ts) return 'N/A';
    const s = Math.round((Date.now() - ts) / 1000);
    if (s < 60) return s + 's ago';
    if (s < 3600) return Math.round(s / 60) + 'm ago';
    return Math.round(s / 3600) + 'h ago';
  }

  function ensureAudio() {
    if (!audioCtx) { try { audioCtx = new (window.AudioContext || window.webkitAudioContext)(); } catch (_) {} }
  }

  function playTone(freq, duration) {
    ensureAudio();
    if (!audioCtx) return;
    try {
      const osc = audioCtx.createOscillator();
      const gain = audioCtx.createGain();
      osc.connect(gain); gain.connect(audioCtx.destination);
      osc.frequency.value = freq; gain.gain.value = 0.4;
      osc.start(); osc.stop(audioCtx.currentTime + duration / 1000);
    } catch (_) {}
  }

  function playSosSound() { playTone(880, 300); setTimeout(() => playTone(660, 300), 350); setTimeout(() => playTone(880, 300), 700); }

  function startViewing() {
    if (!viewerName.trim()) return;
    if (!token) {
      connectionIssue = 'Invalid live link.';
      statusText = connectionIssue;
      return;
    }
    showNameOverlay = false;
    statusText = 'Connecting...';
    connect();
  }

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
        online = false;
        connectionIssue = 'Unable to load live session. The link may be invalid, expired, or the user is unavailable.';
        statusText = connectionIssue;
      }
    }, 8000);
  }

  let markerPopup = null;

  function connect() {
    socket = createRealtimeSocket({ auth: { viewer: true } });

    socket.on('liveInit', (data) => {
      hasInit = true;
      clearInitTimeout();
      connectionIssue = '';
      linkJoinedAt = Date.now();
      if (data.expiresAt) linkExpiresAt = data.expiresAt;
      if (data.user) { updateMarker(data.user); online = true; statusText = 'Tracking ' + (sharedBy || 'User'); }
      else { online = false; statusText = (sharedBy || 'User') + ' (offline)'; }
      if (data.sos?.active) showSos(data.sos);
    });

    socket.on('liveUpdate', (data) => {
      if (data.user) {
        updateMarker(data.user);
        online = true;
        statusText = 'Tracking ' + sharedBy;
        if (data.user.timestamp) lastOriginTs = data.user.timestamp;
      }
    });

    socket.on('liveSosUpdate', (data) => { if (data.active) showSos(data); else hideSos(); });

    socket.on('liveCheckInUpdate', (data) => {
      if (!data?.enabled) { checkinText = ''; return; }
      const sinceMs = data.lastCheckInAt ? Date.now() - data.lastCheckInAt : Infinity;
      const overdueMs = (data.overdueMinutes || 7) * 60000;
      if (sinceMs > overdueMs) { checkinText = 'Check-in OVERDUE (' + formatTimeAgo(data.lastCheckInAt) + ')'; checkinOverdue = true; }
      else { checkinText = 'Last check-in: ' + formatTimeAgo(data.lastCheckInAt); checkinOverdue = false; }
    });

    socket.on('liveExpired', () => { expired = true; clearInitTimeout(); });
    socket.on('disconnect', () => {
      online = false;
      statusText = sharedBy + ' (reconnecting...)';
      if (!hasInit) scheduleInitTimeout();
    });
    socket.on('connect_error', () => {
      online = false;
      statusText = 'Connection error. Retrying...';
      if (!hasInit) scheduleInitTimeout();
    });
    socket.on('connect', () => {
      if (!hasInit) scheduleInitTimeout();
      if (viewerName) socket.emit('liveJoin', { token, viewerName });
    });
  }

  function updateMarker(user) {
    if (typeof user.latitude !== 'number') return;
    const lngLat = [user.longitude, user.latitude];
    const popupHtml = `<strong>${escHtml(user.displayName || 'User')}</strong><br>Speed: ${(user.speed || 0)} km/h<br>Updated: ${escHtml(user.formattedTime || 'N/A')}${user.batteryPct != null ? '<br>Battery: ' + user.batteryPct + '%' : ''}`;
    sharedBy = user.displayName || 'User';
    if (!marker) {
      const el = createMapIcon('var(--primary-500)', '', { markerType: 'contact' });
      markerPopup = new maplibregl.Popup({ offset: [0, -39], maxWidth: '280px' }).setHTML(popupHtml);
      marker = new maplibregl.Marker({ element: el, anchor: 'bottom' })
        .setLngLat(lngLat).setPopup(markerPopup).addTo(map);
    } else {
      animateMarkerTo('live-target', marker, lngLat, 300);
      if (markerPopup) markerPopup.setHTML(popupHtml);
    }
    if (!hasZoomed) { map.jumpTo({ center: lngLat, zoom: 15 }); hasZoomed = true; }
  }

  function showSos(sosData) {
    sosActive = true;
    sosInfo = (sosData.reason || 'SOS') + (sosData.at ? ' - ' + formatTimeAgo(sosData.at) : '');
    const count = sosData.ackCount || (sosData.acks?.length || 0);
    sosAcks = count > 0 ? `${count} acknowledged` : 'No acknowledgements yet';
    playSosSound();
    if (sosAudioInterval) clearInterval(sosAudioInterval);
    sosAudioInterval = setInterval(() => { if (!sosActive || sosAcked) { clearInterval(sosAudioInterval); return; } playSosSound(); }, 3000);
    try { if (navigator.vibrate) navigator.vibrate([200, 100, 200]); } catch (_) {}
  }

  function hideSos() { sosActive = false; sosAcked = false; if (sosAudioInterval) clearInterval(sosAudioInterval); }

  function ackSos() {
    if (!socket || sosAcked) return;
    sosAcked = true;
    socket.emit('liveAckSOS', {});
    if (sosAudioInterval) clearInterval(sosAudioInterval);
  }

  function checkMobile() {
    isMobile = typeof window !== 'undefined' && window.innerWidth < 768;
  }

  onMount(() => {
    checkMobile();
    window.addEventListener('resize', checkMobile);
    map = new maplibregl.Map({ container: mapContainer, style: MAP_STYLE, center: [78, 20], zoom: 5, attributionControl: true });
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'bottom-right');
    freshnessInterval = setInterval(() => {
      const now = Date.now();
      if (lastOriginTs) {
        const sec = Math.round((now - lastOriginTs) / 1000);
        freshnessText = sec < 2 ? 'live' : sec < 60 ? sec + 's ago' : Math.round(sec / 60) + 'm ago';
      }
      if (linkExpiresAt && linkJoinedAt) {
        const total = linkExpiresAt - linkJoinedAt;
        const remaining = linkExpiresAt - now;
        expiryPercent = total > 0 ? Math.max(0, Math.min(100, (remaining / total) * 100)) : 0;
      }
    }, 1000);
  });

  onDestroy(() => {
    if (typeof window !== 'undefined') window.removeEventListener('resize', checkMobile);
    clearInitTimeout();
    if (freshnessInterval) clearInterval(freshnessInterval);
    if (socket) socket.disconnect();
    if (sosAudioInterval) clearInterval(sosAudioInterval);
    if (map) map.remove();
  });
</script>

<div class="live-page" class:sos-active={sosActive}>
  <div class="live-map" bind:this={mapContainer}></div>

  {#if expired}
    <div class="overlay">
      <div class="card expired-card">
        <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--danger-500)" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
        <h2>Link Expired</h2>
        <p class="text-sm text-muted">This live share link is no longer active.</p>
        <a href="/#/login" class="btn btn-primary" style="margin-top:var(--space-4);">Open Kinnect</a>
        <p class="text-sm text-muted" style="margin-top:var(--space-2);">Don't have an account? <a href="/#/register">Create one</a></p>
      </div>
    </div>
  {:else if showNameOverlay}
    <div class="overlay">
      <div class="card name-card">
        <div class="name-card-logo">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="10" r="3"/><path d="M12 2a8 8 0 0 0-8 8c0 1.892.402 3.13 1.5 4.5L12 22l6.5-7.5c1.098-1.37 1.5-2.608 1.5-4.5a8 8 0 0 0-8-8z"/></svg>
        </div>
        <div class="live-header">
          <span class="rec-dot animate-rec-blink" aria-hidden="true"></span>
          <h2>{sharedBy !== 'User' ? `${sharedBy}'s Live Location` : 'Live Location'}</h2>
        </div>
        <p class="text-sm text-muted" style="margin-bottom:var(--space-4);">You were invited to watch. Treat this with care.</p>
        <p class="text-sm text-muted" style="margin-bottom:var(--space-4);">Enter your name to start viewing</p>
        <input class="input input-lg" placeholder="Your name" bind:value={viewerName} on:keydown={e => e.key === 'Enter' && startViewing()} />
        <button class="btn btn-primary btn-lg" style="width:100%;margin-top:var(--space-3);" on:click={startViewing}>Start Viewing</button>
        <p class="text-sm text-muted" style="margin-top:var(--space-3);">Want your own account? <a href="/#/register">Sign up</a> or <a href="/#/login">Log in</a></p>
      </div>
    </div>
  {/if}

  {#if !showNameOverlay && !expired && connectionIssue}
    <div class="live-error">
      <span>{connectionIssue}</span>
      <button class="btn btn-sm btn-secondary" on:click={() => window.location.reload()}>Retry</button>
    </div>
  {/if}

  {#if !showNameOverlay && !expired}
    <!-- MERIDIAN: Floating glass bottom card replacing old status-bar -->
    <div class="live-glass-card" class:sos-state={sosActive}>
      <div class="glass-inner">
        <div class="glass-row">
          {#if online}
            <span class="rec-dot-mini animate-rec-blink" aria-hidden="true"></span>
          {:else}
            <span class="glass-offline-dot" aria-hidden="true"></span>
          {/if}
          <span class="glass-status">{statusText}</span>
          {#if freshnessText}
            <span class="glass-fresh" class:stale={freshnessText !== 'live'}>{freshnessText}</span>
          {/if}
        </div>
        {#if checkinText}
          <div class="glass-checkin" class:overdue={checkinOverdue}>{checkinText}</div>
        {/if}
        {#if linkExpiresAt}
          <div class="expiry-bar" aria-hidden="true">
            <div class="expiry-fill" style="width:{expiryPercent}%"></div>
          </div>
        {/if}
      </div>
      <a href="/#/register" class="glass-signup">Get Kinnect</a>
    </div>
    <a href="/#/login" class="live-brand" aria-label="Powered by Kinnect">Powered by Kinnect</a>
  {/if}

  {#if sosActive}
    <div class="sos-banner" role="alert" aria-live="assertive">
      <div class="sos-icon" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
      </div>
      <div class="sos-content">
        <div class="sos-text">{sosInfo}</div>
        <div class="sos-acks">{sosAcks}</div>
      </div>
      <button class="btn btn-sm" class:btn-secondary={sosAcked} class:btn-primary={!sosAcked} on:click={ackSos} disabled={sosAcked}>
        {sosAcked ? 'Acknowledged' : 'Acknowledge'}
      </button>
    </div>
  {/if}
</div>

<style>
  .live-page {
    position: relative;
    width: 100%;
    height: 100vh;
    height: 100dvh;
  }

  .live-page.sos-active {
    outline: 3px solid var(--danger-500);
    outline-offset: -3px;
    animation: sos-border-pulse 2s ease-in-out infinite;
  }

  @keyframes sos-border-pulse {
    0%, 100% { outline-color: var(--danger-500); }
    50% { outline-color: transparent; }
  }

  .live-map { position: absolute; inset: 0; z-index: 1; }

  .overlay {
    position: absolute;
    inset: 0;
    z-index: 100;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    backdrop-filter: blur(4px);
    padding: var(--space-4);
  }

  .name-card {
    max-width: 360px;
    width: 100%;
    text-align: center;
  }

  .name-card-logo {
    width: 48px;
    height: 48px;
    background: var(--primary-600);
    border-radius: var(--radius-lg);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    margin: 0 auto var(--space-4);
    box-shadow: var(--shadow-primary);
  }

  .expired-card {
    max-width: 360px;
    width: 100%;
    text-align: center;
    padding: var(--space-8);
  }

  .expired-card h2 {
    color: var(--danger-500);
    margin-top: var(--space-3);
  }

  /* MERIDIAN: Floating glass bottom card */
  .live-glass-card {
    position: absolute;
    bottom: calc(var(--space-5) + env(safe-area-inset-bottom, 0px));
    left: 50%;
    transform: translateX(-50%);
    z-index: 50;
    width: min(92vw, 400px);
    background: rgba(8, 8, 16, 0.74);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.09);
    border-radius: var(--radius-xl);
    padding: var(--space-3) var(--space-4);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.40), 0 0 0 1px rgba(99, 102, 241, 0.06);
    display: flex;
    align-items: center;
    gap: var(--space-3);
    color: white;
    animation: slide-up-in 380ms var(--ease-spring) both;
    transition: bottom 300ms var(--ease-spring);
  }

  .live-glass-card.sos-state {
    bottom: calc(var(--space-3) + 80px + env(safe-area-inset-bottom, 0px));
    border-color: rgba(239, 68, 68, 0.28);
    box-shadow: 0 8px 32px rgba(0,0,0,0.4), 0 0 24px rgba(239,68,68,0.18);
  }

  .glass-inner {
    flex: 1;
    min-width: 0;
  }

  .glass-row {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
  }

  .rec-dot-mini {
    display: inline-block;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--color-rec, #ef4444);
    flex-shrink: 0;
    animation: recording-blink 1.2s ease-in-out infinite;
  }

  .glass-offline-dot {
    display: inline-block;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.30);
    flex-shrink: 0;
  }

  .glass-status {
    font-size: var(--text-sm);
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .glass-fresh {
    margin-left: auto;
    font-size: var(--text-xs);
    color: var(--success-400, #34d399);
    font-weight: 700;
    flex-shrink: 0;
  }
  .glass-fresh.stale {
    color: rgba(255, 255, 255, 0.40);
    font-weight: 500;
  }

  .glass-checkin {
    margin-top: 3px;
    font-size: var(--text-xs);
    opacity: 0.55;
  }
  .glass-checkin.overdue {
    color: #fca5a5;
    opacity: 1;
    font-weight: 600;
  }

  .expiry-bar {
    margin-top: var(--space-2);
    height: 3px;
    background: rgba(255, 255, 255, 0.10);
    border-radius: 2px;
    overflow: hidden;
  }

  .expiry-fill {
    height: 100%;
    background: var(--primary-400);
    border-radius: 2px;
    transition: width 1s linear;
  }

  .glass-signup {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    padding: 6px 12px;
    border-radius: var(--radius-full);
    background: var(--primary-600);
    color: white;
    font-size: var(--text-xs);
    font-weight: 700;
    text-decoration: none;
    white-space: nowrap;
    box-shadow: 0 0 16px rgba(99, 102, 241, 0.30);
    transition: background 0.15s;
  }
  .glass-signup:hover {
    background: var(--primary-500);
  }

  .sos-banner {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    z-index: 50;
    background: var(--danger-500);
    color: white;
    padding: var(--space-3) var(--space-4);
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-wrap: wrap;
  }

  .live-error {
    position: absolute;
    top: var(--space-4);
    left: 50%;
    transform: translateX(-50%);
    z-index: 60;
    background: var(--surface-2);
    border: 1px solid var(--border-default);
    box-shadow: var(--shadow-lg);
    border-radius: var(--radius-md);
    padding: var(--space-2) var(--space-3);
    display: flex;
    align-items: center;
    gap: var(--space-2);
    max-width: min(90vw, 640px);
  }

  .sos-icon { flex-shrink: 0; }
  .sos-content { flex: 1; min-width: 0; }
  .sos-text { font-weight: 700; }
  .sos-acks { font-size: var(--text-xs); opacity: 0.8; }

  .live-brand {
    position: absolute;
    bottom: calc(var(--space-3) + env(safe-area-inset-bottom, 0px));
    left: 50%;
    transform: translateX(-50%);
    z-index: 50;
    font-size: 10px;
    color: rgba(255, 255, 255, 0.35);
    text-decoration: none;
    letter-spacing: 0.06em;
    opacity: 1;
    transition: color 0.15s;
    white-space: nowrap;
  }
  .live-brand:hover { color: rgba(255, 255, 255, 0.65); }

  /* Name card recording header */
  .live-header {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
  }
  .live-header h2 { margin: 0; }

  .rec-dot {
    display: inline-block;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--color-rec, #ef4444);
    flex-shrink: 0;
    animation: recording-blink 1.2s ease-in-out infinite;
  }

  @keyframes recording-blink {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0.2; }
  }

  @media (max-width: 767px) {
    .live-glass-card {
      width: min(96vw, 400px);
      padding: var(--space-2-5) var(--space-3);
    }
    .sos-banner {
      padding: var(--space-3);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .live-page.sos-active {
      animation: none;
      outline-color: var(--danger-500);
    }
  }
</style>
