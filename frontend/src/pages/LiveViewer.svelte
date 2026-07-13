<script>
  import { onMount, onDestroy } from 'svelte';
  import maplibregl from 'maplibre-gl';
  import { createRealtimeSocket } from '../lib/realtimeClient.js';
  import { createMapIcon, escHtml } from '../lib/tracking.js';
  import { animateMarkerTo } from '../lib/markerInterpolator.js';
  import { deriveConnState, CONNECTION_TIMEOUT_MS } from '../lib/presence.js';
  import Card from '../components/primitives/Card.svelte';
  import SharedLiveShell from '../components/SharedLiveShell.svelte';

  let { params = {} } = $props();

  let map = null;
  let marker = null;
  let hasZoomed = false;
  let socket = null;
  let viewerName = $state('');
  let showNameOverlay = $state(true);
  let statusText = $state('Connecting...');
  let online = $state(false);
  let sharedBy = $state('User');
  let sosActive = $state(false);
  let sosInfo = $state('');
  let sosAcks = $state('');
  let sosAcked = $state(false);
  let expired = $state(false);
  let checkinText = $state('');
  let checkinOverdue = $state(false);
  let sosAudioInterval = null;
  let audioCtx = null;
  let isMobile = false;
  let hasInit = false;
  let initTimeout = null;
  let connectionIssue = $state('');
  let lastOriginTs = null;
  let freshnessText = $state('');
  let freshnessInterval = null;
  let linkJoinedAt = null;
  let linkExpiresAt = $state(null);
  let expiryPercent = $state(100);

  let token = $derived(params.token || '');

  // Canonical connection state — shared derivation (lib/presence.js).
  let connState = $derived(
    deriveConnState({
      initialized: online,
      online,
      issue: !!connectionIssue,
      connecting: /connect|reconnect|error|retry/i.test(statusText || ''),
    })
  );

  // Freshness → simple 0–3 signal strength (reuses freshnessText, no new state).
  let signalLevel = $derived(
    !online ? 0
      : freshnessText === 'live' ? 3
      : freshnessText.endsWith('s ago') ? 2
      : freshnessText ? 1 : 3
  );

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
    }, CONNECTION_TIMEOUT_MS);
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
    if (!map || typeof user.latitude !== 'number') return;
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

  function handleMap(m) {
    map = m;
  }

  onMount(() => {
    checkMobile();
    window.addEventListener('resize', checkMobile);

    // Silently attempt to open the native Kinnect app if installed.
    // Uses a hidden iframe so the page URL doesn't change and no error is shown
    // to the user. If the app intercepts kinnect://, it opens. If not, nothing happens.
    const isNative = typeof window !== 'undefined' && window.Capacitor?.isNativePlatform?.();
    if (!isNative && isMobile && token) {
      const appFrame = document.createElement('iframe');
      appFrame.style.cssText = 'display:none;width:0;height:0;border:0;position:absolute;';
      appFrame.src = `kinnect://live/${token}`;
      document.body.appendChild(appFrame);
      setTimeout(() => { try { document.body.removeChild(appFrame); } catch (_) {} }, 2000);
    }

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
    // map lifecycle is owned by SharedLiveShell.
  });
</script>

<SharedLiveShell
  position="bottom"
  {connState}
  statusLabel={statusText}
  {sosActive}
  showSignal={online && !!freshnessText}
  {signalLevel}
  freshnessLabel={freshnessText}
  expiryPercent={!showNameOverlay && !expired && linkExpiresAt ? expiryPercent : null}
  onMap={handleMap}
>
  {#snippet overlay()}
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
          <input class="input input-lg" placeholder="Your name" bind:value={viewerName} onkeydown={e => e.key === 'Enter' && startViewing()} />
          <button class="btn btn-primary btn-lg" style="width:100%;margin-top:var(--space-3);" onclick={startViewing}>Start Viewing</button>
          <p class="text-sm text-muted" style="margin-top:var(--space-3);">Want your own account? <a href="/#/register">Sign up</a> or <a href="/#/login">Log in</a></p>
        </div>
      </div>
    {/if}

    {#if !showNameOverlay && !expired && connectionIssue}
      <div class="live-error">
        <span>{connectionIssue}</span>
        <button class="btn btn-sm btn-secondary" onclick={() => window.location.reload()}>Retry</button>
      </div>
    {/if}
  {/snippet}

  {#snippet cardExtras()}
    {#if !showNameOverlay && !expired && checkinText}
      <div class="glass-checkin" class:overdue={checkinOverdue}>{checkinText}</div>
    {/if}
  {/snippet}

  {#snippet docks()}
    {#if !showNameOverlay && !expired}
      <a href="/#/login" class="live-brand" aria-label="Powered by Kinnect">Powered by Kinnect</a>
    {/if}

    {#if sosActive}
      <div class="sos-dock">
        <Card variant="glass" glow="danger" padding="none" hover={false}>
          <div class="sos-banner" role="alert" aria-live="assertive">
            <div class="sos-icon" aria-hidden="true">
              <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
            </div>
            <div class="sos-content">
              <div class="sos-eyebrow">Emergency SOS</div>
              <div class="sos-text">{sosInfo}</div>
              <div class="sos-acks">{sosAcks}</div>
            </div>
            <button class="btn btn-sm sos-ack-btn" class:btn-secondary={sosAcked} class:btn-primary={!sosAcked} onclick={ackSos} disabled={sosAcked}>
              {sosAcked ? 'Acknowledged' : 'Acknowledge'}
            </button>
          </div>
        </Card>
      </div>
    {/if}
  {/snippet}
</SharedLiveShell>

<style>
  .overlay {
    position: absolute;
    inset: 0;
    z-index: 100;
    background: rgba(0, 0, 0, 0.5); /* raw-color-ok: modal scrim, no scrim token exists yet */
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
    color: var(--text-inverse, white);
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

  .glass-checkin {
    margin-top: 3px;
    font-size: var(--text-xs);
    opacity: 0.55;
  }
  .glass-checkin.overdue {
    color: var(--danger-400);
    opacity: 1;
    font-weight: 600;
  }

  /* SOS emergency dock — danger-glow Card with strong hierarchy */
  .sos-dock {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    z-index: 55;
    padding: var(--space-3);
    padding-bottom: max(var(--space-3), env(safe-area-inset-bottom, 0px));
    animation: slide-up-in 340ms var(--ease-out) both;
  }

  .sos-banner {
    color: var(--text-inverse, white);
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

  .sos-icon {
    flex-shrink: 0;
    color: var(--danger-400);
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    border-radius: var(--radius-full);
    background: var(--danger-500-20);
  }
  .sos-content { flex: 1; min-width: 0; }
  .sos-eyebrow {
    font-size: 10px;
    font-weight: 800;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--danger-400);
    margin-bottom: 2px;
  }
  .sos-text {
    font-weight: 700;
    font-size: var(--text-base);
    line-height: 1.2;
  }
  .sos-acks {
    font-size: var(--text-xs);
    opacity: 0.7;
    margin-top: 2px;
  }
  .sos-ack-btn {
    flex-shrink: 0;
    min-height: 44px;
  }

  .live-brand {
    position: absolute;
    bottom: calc(var(--space-3) + env(safe-area-inset-bottom, 0px));
    left: 50%;
    transform: translateX(-50%);
    z-index: 50;
    font-size: 10px;
    color: color-mix(in oklch, var(--text-inverse, white) 35%, transparent);
    text-decoration: none;
    letter-spacing: 0.06em;
    opacity: 1;
    transition: color 0.15s;
    white-space: nowrap;
  }
  .live-brand:hover { color: color-mix(in oklch, var(--text-inverse, white) 65%, transparent); }

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
    background: var(--color-rec);
    flex-shrink: 0;
    animation: recording-blink 1.2s ease-in-out infinite;
  }

  @keyframes recording-blink {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0.2; }
  }

  @media (prefers-reduced-motion: reduce) {
    .rec-dot {
      animation: none;
    }
    .sos-dock {
      animation: none;
    }
  }
</style>
