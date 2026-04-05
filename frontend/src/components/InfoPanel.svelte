<script>
  import { createEventDispatcher } from 'svelte';
  import { myLocation, tracking, walkDestination } from '../lib/stores/map.js';
  import { authUser } from '../lib/stores/auth.js';
  import { myShareCode, myContactInfo } from '../lib/stores/rooms.js';
  import { socket } from '../lib/socket.js';
  import { banner, mySosActive, myLiveLinks } from '../lib/stores/sos.js';
  import { myGuardianData, pendingIncomingRequests } from '../lib/stores/guardians.js';
  import { formatCoordinate } from '../lib/tracking.js';
  import { getShareOrigin } from '../lib/env.js';
  import { trackingMetrics } from '../lib/stores/metrics.js';
  import { latencyMetrics } from '../lib/stores/latency.js';
  import CopyButton from './primitives/CopyButton.svelte';
  import ShareMyRide from './ShareMyRide.svelte';
  import WalkWithMe from './WalkWithMe.svelte';
  import { rideShare } from '../lib/stores/rideShare.js';
  import { crowdMode } from '../lib/stores/crowdMode.js';

  export let embedded = false;
  let statsOpen = false;
  let debugTaps = 0;
  let showQr = false;
  let rideShareOpen = false;
  let walkWithMeOpen = false;

  // Auto-open Walk With Me when PlaceSearch sets a destination
  $: if ($walkDestination) { walkWithMeOpen = true; }

  function toggleCrowdMode() {
    const next = !$crowdMode.active;
    crowdMode.update(s => ({ ...s, active: next }));
    socket.emit('toggleCrowdMode', { enabled: next, radiusM: $crowdMode.radiusM });
  }

  const dispatch = createEventDispatcher();

  function getMedicalSnapshot() {
    try {
      const raw = localStorage.getItem('kinnect_emergency_profile');
      if (!raw) return null;
      const p = JSON.parse(raw);
      const hasData = p.bloodType || p.allergies || p.medications || p.emergencyName;
      return hasData ? p : null;
    } catch { return null; }
  }

  function toggleSOS() {
    if (!socket.connected) {
      banner.set({ type: 'info', text: 'Trying to reconnect... hang tight.', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
      return;
    }
    if (!$mySosActive) socket.emit('triggerSOS', { reason: 'SOS', medicalCard: getMedicalSnapshot() });
    else socket.emit('cancelSOS');
  }

  function checkIn() {
    socket.emit('checkInAck');
    banner.set({ type: 'info', text: 'Check-in sent — your family knows you\'re okay.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
  }

  let onMyWayActive = false;

  // Ambient status message
  let statusDraft = '';
  let statusExpiry = '60'; // minutes; '0' = no expiry

  function saveStatusMessage() {
    socket.emit('setStatusMessage', {
      message: statusDraft.trim(),
      expiryMinutes: statusExpiry === '0' ? 0 : parseInt(statusExpiry, 10),
    });
    banner.set({ type: 'info', text: statusDraft.trim() ? 'Status updated.' : 'Status cleared.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 1500);
  }

  function clearStatusMessage() {
    statusDraft = '';
    socket.emit('setStatusMessage', { message: '', expiryMinutes: 0 });
  }

  function onMyWay() {
    socket.emit('onMyWay', {});
    onMyWayActive = true;

    // Build WhatsApp share message — include live link URL if one exists
    const links = $myLiveLinks;
    let waText = "I'm on my way! 🚀";
    if (links && links.length > 0) {
      const liveUrl = getShareOrigin() + '/#/live/' + links[0].token;
      waText += ` Track me live: ${liveUrl}`;
    }
    window.open('https://wa.me/?text=' + encodeURIComponent(waText), '_blank', 'noopener');

    banner.set({ type: 'info', text: 'Your family knows you\'re on the way.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
  }

  function cancelOnMyWay() {
    socket.emit('cancelOnMyWay');
    onMyWayActive = false;
  }

  function attest() {
    socket.emit('attest');
    banner.set({ type: 'info', text: 'Your location has been verified.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
  }

  let busyRequests = new Set();
  function approveRequest(req, idx) {
    const key = req.type + '-' + req.from;
    if (busyRequests.has(key)) return;
    busyRequests.add(key);
    busyRequests = busyRequests;
    if (req.type === 'roomAdmin') {
      socket.emit('voteRoomAdmin', { roomCode: req.roomCode, userId: req.from, vote: 'approve' });
    } else if (req.type === 'guardianInvite') {
      socket.emit('approveGuardian', { wardUserId: req.from });
    } else {
      socket.emit('approveGuardian', { guardianUserId: req.from });
    }
    pendingIncomingRequests.update(arr => {
      if (req.type === 'roomAdmin') {
        return arr.map((r, i) => i === idx ? { ...r, myVote: 'approve' } : r);
      }
      arr.splice(idx, 1); return [...arr];
    });
    setTimeout(() => { busyRequests.delete(key); busyRequests = busyRequests; }, 5000);
  }

  function denyRequest(req, idx) {
    const key = req.type + '-' + req.from;
    if (busyRequests.has(key)) return;
    busyRequests.add(key);
    busyRequests = busyRequests;
    if (req.type === 'roomAdmin') {
      socket.emit('voteRoomAdmin', { roomCode: req.roomCode, userId: req.from, vote: 'deny' });
    } else if (req.type === 'guardianInvite') {
      socket.emit('denyGuardian', { wardUserId: req.from });
    } else {
      socket.emit('denyGuardian', { guardianUserId: req.from });
    }
    pendingIncomingRequests.update(arr => {
      if (req.type === 'roomAdmin') {
        return arr.map((r, i) => i === idx ? { ...r, myVote: 'deny' } : r);
      }
      arr.splice(idx, 1); return [...arr];
    });
    setTimeout(() => { busyRequests.delete(key); busyRequests = busyRequests; }, 5000);
  }

  function revokeGuardian(wardId, guardianId) {
    if (wardId) socket.emit('revokeGuardian', { wardUserId: wardId });
    else if (guardianId) socket.emit('revokeGuardian', { guardianUserId: guardianId });
  }

  function getRequestLabel(req) {
    if (req.type === 'roomAdmin') return `${req.fromName || 'Someone'} wants Admin in ${req.roomCode}`;
    if (req.type === 'guardianInvite') return `${req.fromName || 'Someone'} wants you to be their Guardian`;
    return `${req.fromName || 'Someone'} wants to be your Guardian`;
  }

  function tapGps() {
    debugTaps++;
    if (debugTaps >= 5) { statsOpen = !statsOpen; debugTaps = 0; }
  }

  $: accClass = $trackingMetrics.lastAccuracy != null
    ? $trackingMetrics.lastAccuracy <= 15 ? 'green'
    : $trackingMetrics.lastAccuracy <= 50 ? 'yellow' : 'red'
    : '';
  $: accLabel = $trackingMetrics.lastAccuracy != null
    ? ($trackingMetrics.lastAccuracy <= 15 ? 'Sharp' : $trackingMetrics.lastAccuracy <= 50 ? 'Decent' : 'Rough') + ` ±${$trackingMetrics.lastAccuracy}m`
    : $tracking ? 'Getting location...' : 'Tap Track to start';
</script>

<!-- ═══════════════════════════════════════════════════════════════════════
     EMBEDDED MODE (used in sidebar + bottom sheet)
     ═══════════════════════════════════════════════════════════════════════ -->
{#if embedded}
  <div class="panel-body info-root">

    <!-- ── GPS LIVE STATUS ──────────────────────────────────────────── -->
    {#if $myLocation}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <div class="gps-live-card" class:is-tracking={$tracking} on:click={tapGps} role="presentation" aria-hidden="true">
        <div class="gps-signal-left">
          <span class="gps-ping" class:active={$tracking && $trackingMetrics.lastAccuracy != null}></span>
          <div class="gps-coord-block">
            <span class="gps-accuracy-label">{accLabel}</span>
            <div class="gps-sub">
              {#if $trackingMetrics.lastAccuracy != null}
                <span class="accuracy-dot {accClass}"></span>
              {/if}
              <span>{$myLocation.formattedTime || 'Live'}</span>
            </div>
          </div>
        </div>
        {#if ($myLocation.speed || 0) >= 1}
          <div class="speed-pill">
            <span class="speed-num">{Math.round($myLocation.speed)}</span>
            <span class="speed-unit">km/h</span>
          </div>
        {/if}
      </div>
    {:else}
      <div class="gps-acquire-card animate-breathe">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="acquire-icon" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M16.24 7.76a6 6 0 0 1 0 8.49"/><path d="M7.76 7.76a6 6 0 0 0 0 8.49"/><path d="M20.49 3.51a12 12 0 0 1 0 16.97"/><path d="M3.51 3.51a12 12 0 0 0 0 16.97"/></svg>
        <div>
          <p class="acquire-title">Getting your location...</p>
          <p class="acquire-hint">Tap <strong>Track</strong> to share your location</p>
        </div>
      </div>
    {/if}

    <!-- ── DEBUG STATS (hidden tap-to-reveal) ─────────────────────── -->
    {#if statsOpen && $tracking && $trackingMetrics.fixCount > 0}
      <div class="tracking-stats">
        <div class="stat-row"><span>Accuracy</span><span>{$trackingMetrics.lastAccuracy ?? '-'}m (avg {$trackingMetrics.avgAccuracy ?? '-'}m)</span></div>
        <div class="stat-row"><span>Fixes</span><span>{$trackingMetrics.fixCount}</span></div>
        <div class="stat-row"><span>Rate</span><span>{$trackingMetrics.updatesPerSec}/s</span></div>
        <div class="stat-row"><span>Kalman</span><span>{$trackingMetrics.kalmanCorrectionM}m correction</span></div>
        <div class="stat-row"><span>Filter</span><span>{$trackingMetrics.filterState}</span></div>
        {#if $latencyMetrics.lastE2eMs != null}
          <div class="stat-row"><span>E2E Latency</span><span class="latency-value" class:latency-good={$latencyMetrics.lastE2eMs < 300} class:latency-ok={$latencyMetrics.lastE2eMs >= 300 && $latencyMetrics.lastE2eMs < 800} class:latency-bad={$latencyMetrics.lastE2eMs >= 800}>{$latencyMetrics.lastE2eMs}ms (avg {$latencyMetrics.avgE2eMs}ms)</span></div>
        {/if}
        {#if $latencyMetrics.lastServerHopMs != null}
          <div class="stat-row"><span>Server Hop</span><span>{$latencyMetrics.lastServerHopMs}ms</span></div>
        {/if}
      </div>
    {/if}

    <!-- ── IDENTITY CARD ────────────────────────────────────────────── -->
    <div class="identity-card">
      <span class="card-eyebrow">Your Code</span>
      <div class="identity-body">
        <code class="signal-code">{$myShareCode || '—'}</code>
        <div class="signal-btns">
          <CopyButton text={$myShareCode || ''} label="Copy" />
          {#if $myShareCode}
            <button class="qr-icon-btn" on:click={() => showQr = true} aria-label="Show QR code for signal code" title="QR code">
              <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="3" height="3"/><rect x="18" y="14" width="3" height="3"/><rect x="14" y="18" width="3" height="3"/><rect x="18" y="18" width="3" height="3"/></svg>
            </button>
          {/if}
        </div>
      </div>
      {#if $myContactInfo?.email || $myContactInfo?.mobile}
        <span class="identity-meta">{[$myContactInfo.email, $myContactInfo.mobile].filter(Boolean).join(' · ')}</span>
      {/if}
    </div>

    <!-- ── SAFETY ZONE ───────────────────────────────────────────────── -->
    <div class="safety-zone">
      <span class="card-eyebrow safety-eyebrow">Safety</span>
      <div class="safety-actions">
        <button
          class="sos-action-btn"
          class:sos-live={$mySosActive}
          on:click={toggleSOS}
          aria-label={$mySosActive ? 'Cancel SOS' : 'Send emergency SOS alert'}
        >
          <span class="sos-icon-wrap" aria-hidden="true">
            {#if $mySosActive}
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="13"/><circle cx="12" cy="18" r="1" fill="currentColor" stroke="none"/></svg>
            {/if}
          </span>
          {$mySosActive ? 'Cancel SOS' : 'SOS'}
        </button>
        <button class="ok-action-btn" on:click={checkIn} aria-label="Send scheduled check-in">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
          I'm OK
        </button>
      </div>
      <div class="safety-actions" style="margin-top:6px;">
        {#if onMyWayActive}
          <button class="ok-action-btn ok-action-btn--active" on:click={cancelOnMyWay} aria-label="Cancel On My Way broadcast">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            Cancel
          </button>
        {:else}
          <button class="ok-action-btn" on:click={onMyWay} aria-label="Broadcast On My Way and share via WhatsApp">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
            On My Way
          </button>
        {/if}
        <button class="ok-action-btn" on:click={attest} aria-label="Confirm your location is genuine" title="Confirm your location is real">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
          I'm Safe
        </button>
      </div>
      <div class="safety-actions" style="margin-top:6px;">
        <button class="ok-action-btn" class:ok-action-btn--active={$rideShare.active} on:click={() => rideShareOpen = true} aria-label="Share My Ride with family">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="1" y="3" width="15" height="13"/><polygon points="16 8 20 8 23 11 23 16 16 16 16 8"/><circle cx="5.5" cy="18.5" r="2.5"/><circle cx="18.5" cy="18.5" r="2.5"/></svg>
          {$rideShare.active ? 'Ride Active' : 'Share My Ride'}
        </button>
        <button class="ok-action-btn" class:ok-action-btn--crowd={$crowdMode.active} on:click={toggleCrowdMode} aria-label="Toggle Festival / Crowd mode">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          {$crowdMode.active ? 'Group Active' : 'Stay Together'}
        </button>
        <button class="ok-action-btn" on:click={() => walkWithMeOpen = true} aria-label="Walk With Me — virtual escort">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><circle cx="12" cy="5" r="1.5"/><path d="M9 20l1.5-5 2.5 2 2.5-7"/><path d="M6 9h12"/></svg>
          Walk With Me
        </button>
      </div>
    </div>

    <!-- ── AMBIENT STATUS MESSAGE ─────────────────────────────────── -->
    <div class="status-msg-zone">
      <span class="card-eyebrow">My Status</span>
      <div class="status-msg-row">
        <input
          class="status-msg-input"
          type="text"
          maxlength="60"
          placeholder='e.g. "At school until 3pm"'
          bind:value={statusDraft}
          aria-label="Set a status message visible to your family"
        />
        <select class="status-expiry-select" bind:value={statusExpiry} aria-label="Status expires after">
          <option value="60">1h</option>
          <option value="240">4h</option>
          <option value="480">8h</option>
          <option value="1440">Today</option>
          <option value="0">Always</option>
        </select>
      </div>
      <div class="status-msg-actions">
        <button class="btn btn-primary btn-sm" on:click={saveStatusMessage} disabled={statusDraft.trim() === ''}>Set</button>
        <button class="btn btn-ghost btn-sm" on:click={clearStatusMessage}>Clear</button>
      </div>
    </div>

    <!-- ── GUARDIAN NETWORK ──────────────────────────────────────────── -->
    {#if $myGuardianData.asGuardian.length > 0 || $myGuardianData.asWard.length > 0}
      <div class="network-section">
        <span class="card-eyebrow">Guardians</span>
        {#each $myGuardianData.asGuardian as g}
          <div class="network-item">
            <div class="network-avatar">{(g.wardName || '?')[0].toUpperCase()}</div>
            <div class="network-info">
              <span class="network-name">{g.wardName}</span>
              <span class="network-role">You are their guardian</span>
            </div>
            <span class="network-status-badge" class:active={g.status === 'active'} class:pending={g.status !== 'active'}>{g.status}</span>
            {#if g.status === 'pending' && g.initiatedBy === 'ward'}
              <button class="btn btn-primary btn-sm" on:click={() => socket.emit('approveGuardian', { wardUserId: g.wardId })}>Accept</button>
              <button class="btn btn-danger btn-sm" on:click={() => socket.emit('denyGuardian', { wardUserId: g.wardId })}>Decline</button>
            {:else if g.status === 'active'}
              <button class="btn btn-danger btn-sm" on:click={() => revokeGuardian(g.wardId, null)}>Revoke</button>
            {:else if g.status === 'pending'}
              <button class="btn btn-danger btn-sm" on:click={() => revokeGuardian(g.wardId, null)}>Cancel</button>
            {/if}
          </div>
        {/each}
        {#each $myGuardianData.asWard as g}
          <div class="network-item">
            <div class="network-avatar guardian-av">{(g.guardianName || '?')[0].toUpperCase()}</div>
            <div class="network-info">
              <span class="network-name">{g.guardianName}</span>
              <span class="network-role">Your guardian</span>
            </div>
            <span class="network-status-badge" class:active={g.status === 'active'} class:pending={g.status !== 'active'}>{g.status}</span>
            {#if g.status === 'pending' && g.initiatedBy === 'guardian'}
              <button class="btn btn-primary btn-sm" on:click={() => socket.emit('approveGuardian', { guardianUserId: g.guardianId })}>Accept</button>
              <button class="btn btn-danger btn-sm" on:click={() => socket.emit('denyGuardian', { guardianUserId: g.guardianId })}>Decline</button>
            {:else if g.status === 'pending' && g.initiatedBy === 'ward'}
              <button class="btn btn-danger btn-sm" on:click={() => revokeGuardian(null, g.guardianId)}>Cancel</button>
            {:else if g.status === 'active'}
              <span class="network-caption">Guardian controls this link</span>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

    <!-- ── PENDING REQUESTS ──────────────────────────────────────────── -->
    {#if $pendingIncomingRequests.length > 0}
      <div class="requests-section">
        <span class="card-eyebrow">Requests <span class="req-count">{$pendingIncomingRequests.length}</span></span>
        {#each $pendingIncomingRequests as req, idx}
          <div class="request-card animate-slide-up stagger-item">
            <p class="req-label">{getRequestLabel(req)}</p>
            {#if req.expiresIn}<span class="req-expiry">{req.expiresIn}</span>{/if}
            {#if req.type === 'roomAdmin'}
              <p class="req-vote-meta">
                {req.approvals || 0} approve · {req.denials || 0} deny · need {Math.floor((req.totalEligible || 1) / 2) + 1} of {req.totalEligible || '?'}
              </p>
              <div class="req-actions">
                {#if req.myVote === 'approve'}
                  <span class="badge badge-success badge-xs">You approved</span>
                {:else if req.myVote === 'deny'}
                  <span class="badge badge-danger badge-xs">You denied</span>
                {:else}
                  <button class="btn btn-primary btn-sm" on:click={() => approveRequest(req, idx)} disabled={busyRequests.has(req.type + '-' + req.from)}>Approve</button>
                  <button class="btn btn-danger btn-sm" on:click={() => denyRequest(req, idx)} disabled={busyRequests.has(req.type + '-' + req.from)}>Deny</button>
                {/if}
              </div>
            {:else}
              <div class="req-actions">
                <button class="btn btn-primary btn-sm" on:click={() => approveRequest(req, idx)} disabled={busyRequests.has(req.type + '-' + req.from)}>Approve</button>
                <button class="btn btn-danger btn-sm" on:click={() => denyRequest(req, idx)} disabled={busyRequests.has(req.type + '-' + req.from)}>Deny</button>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

  </div>
  <ShareMyRide bind:open={rideShareOpen} />
  {#if walkWithMeOpen}
    <div class="wwm-overlay">
      <div class="wwm-modal">
        <button class="wwm-close" on:click={() => walkWithMeOpen = false} aria-label="Close">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
        <WalkWithMe on:close={() => walkWithMeOpen = false} />
      </div>
    </div>
  {/if}

<!-- ═══════════════════════════════════════════════════════════════════════
     PANEL SHELL MODE (desktop standalone panel)
     ═══════════════════════════════════════════════════════════════════════ -->
{:else}
  <div class="panel-shell panel-left panel-base">
    <div class="panel-header">
      <h3>You</h3>
      <button class="btn btn-icon btn-ghost" aria-label="Close info panel" on:click={() => dispatch('close')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>

    <div class="panel-body info-root">

      <!-- GPS LIVE STATUS -->
      {#if $myLocation}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <div class="gps-live-card" class:is-tracking={$tracking} on:click={tapGps} role="presentation" aria-hidden="true">
          <div class="gps-signal-left">
            <span class="gps-ping" class:active={$tracking && $trackingMetrics.lastAccuracy != null}></span>
            <div class="gps-coord-block">
              <span class="gps-accuracy-label">{accLabel}</span>
              <div class="gps-sub">
                {#if $trackingMetrics.lastAccuracy != null}
                  <span class="accuracy-dot {accClass}"></span>
                {/if}
                <span>{$myLocation.formattedTime || 'Live'}</span>
              </div>
            </div>
          </div>
          {#if ($myLocation.speed || 0) >= 1}
            <div class="speed-pill">
              <span class="speed-num">{Math.round($myLocation.speed)}</span>
              <span class="speed-unit">km/h</span>
            </div>
          {/if}
        </div>
      {:else}
        <div class="gps-acquire-card animate-breathe">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="acquire-icon" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M16.24 7.76a6 6 0 0 1 0 8.49"/><path d="M7.76 7.76a6 6 0 0 0 0 8.49"/><path d="M20.49 3.51a12 12 0 0 1 0 16.97"/><path d="M3.51 3.51a12 12 0 0 0 0 16.97"/></svg>
          <div>
            <p class="acquire-title">Getting your location...</p>
            <p class="acquire-hint">Tap <strong>Track</strong> in the top bar to share your location.</p>
          </div>
        </div>
      {/if}

      {#if statsOpen && $tracking && $trackingMetrics.fixCount > 0}
        <div class="tracking-stats">
          <div class="stat-row"><span>Accuracy</span><span>{$trackingMetrics.lastAccuracy ?? '-'}m (avg {$trackingMetrics.avgAccuracy ?? '-'}m)</span></div>
          <div class="stat-row"><span>Fixes</span><span>{$trackingMetrics.fixCount}</span></div>
          <div class="stat-row"><span>Rate</span><span>{$trackingMetrics.updatesPerSec}/s</span></div>
          <div class="stat-row"><span>Kalman</span><span>{$trackingMetrics.kalmanCorrectionM}m correction</span></div>
          <div class="stat-row"><span>Filter</span><span>{$trackingMetrics.filterState}</span></div>
          {#if $latencyMetrics.lastE2eMs != null}
            <div class="stat-row"><span>E2E Latency</span><span class="latency-value" class:latency-good={$latencyMetrics.lastE2eMs < 300} class:latency-ok={$latencyMetrics.lastE2eMs >= 300 && $latencyMetrics.lastE2eMs < 800} class:latency-bad={$latencyMetrics.lastE2eMs >= 800}>{$latencyMetrics.lastE2eMs}ms (avg {$latencyMetrics.avgE2eMs}ms)</span></div>
          {/if}
          {#if $latencyMetrics.lastServerHopMs != null}
            <div class="stat-row"><span>Server Hop</span><span>{$latencyMetrics.lastServerHopMs}ms</span></div>
          {/if}
        </div>
      {/if}

      <!-- IDENTITY CARD -->
      <div class="identity-card">
        <span class="card-eyebrow">Your Code</span>
        <div class="identity-body">
          <code class="signal-code">{$myShareCode || '—'}</code>
          <div class="signal-btns">
            <CopyButton text={$myShareCode || ''} label="Copy" />
            {#if $myShareCode}
              <button class="qr-icon-btn" on:click={() => showQr = true} aria-label="Show QR code for signal code" title="QR code">
                <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="3" height="3"/><rect x="18" y="14" width="3" height="3"/><rect x="14" y="18" width="3" height="3"/><rect x="18" y="18" width="3" height="3"/></svg>
              </button>
            {/if}
          </div>
        </div>
        {#if $myContactInfo?.email || $myContactInfo?.mobile}
          <span class="identity-meta">{[$myContactInfo.email, $myContactInfo.mobile].filter(Boolean).join(' · ')}</span>
        {/if}
      </div>

      <!-- SAFETY ZONE -->
      <div class="safety-zone">
        <span class="card-eyebrow safety-eyebrow">Safety</span>
        <div class="safety-actions">
          <button class="sos-action-btn" class:sos-live={$mySosActive} on:click={toggleSOS} aria-label={$mySosActive ? 'Cancel SOS' : 'Send emergency SOS alert'}>
            <span class="sos-icon-wrap" aria-hidden="true">
              {#if $mySosActive}
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              {:else}
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="13"/><circle cx="12" cy="18" r="1" fill="currentColor" stroke="none"/></svg>
              {/if}
            </span>
            {$mySosActive ? 'Cancel SOS' : 'SOS'}
          </button>
          <button class="ok-action-btn" on:click={checkIn} aria-label="Send scheduled check-in">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
            I'm OK
          </button>
        </div>
        <div class="safety-actions" style="margin-top:6px;">
          {#if onMyWayActive}
            <button class="ok-action-btn ok-action-btn--active" on:click={cancelOnMyWay} aria-label="Cancel On My Way broadcast">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              Cancel
            </button>
          {:else}
            <button class="ok-action-btn" on:click={onMyWay} aria-label="Broadcast On My Way and share via WhatsApp">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
              On My Way
            </button>
          {/if}
          <button class="ok-action-btn" on:click={attest} aria-label="Confirm your location is genuine" title="Confirm your location is real">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
            Verify Location
          </button>
        </div>
        <div class="safety-actions" style="margin-top:6px;">
          <button class="ok-action-btn" class:ok-action-btn--active={$rideShare.active} on:click={() => rideShareOpen = true} aria-label="Share My Ride with family">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="1" y="3" width="15" height="13"/><polygon points="16 8 20 8 23 11 23 16 16 16 16 8"/><circle cx="5.5" cy="18.5" r="2.5"/><circle cx="18.5" cy="18.5" r="2.5"/></svg>
            {$rideShare.active ? 'Ride Active' : 'Share My Ride'}
          </button>
          <button class="ok-action-btn" class:ok-action-btn--crowd={$crowdMode.active} on:click={toggleCrowdMode} aria-label="Toggle Festival / Crowd mode">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
            {$crowdMode.active ? 'Group Active' : 'Stay Together'}
          </button>
        </div>
      </div>

      <!-- AMBIENT STATUS MESSAGE -->
      <div class="status-msg-zone">
        <span class="card-eyebrow">My Status</span>
        <div class="status-msg-row">
          <input
            class="status-msg-input"
            type="text"
            maxlength="60"
            placeholder='e.g. "At school until 3pm"'
            bind:value={statusDraft}
            aria-label="Set a status message visible to your family"
          />
          <select class="status-expiry-select" bind:value={statusExpiry} aria-label="Status expires after">
            <option value="60">1h</option>
            <option value="240">4h</option>
            <option value="480">8h</option>
            <option value="1440">Today</option>
            <option value="0">Always</option>
          </select>
        </div>
        <div class="status-msg-actions">
          <button class="btn btn-primary btn-sm" on:click={saveStatusMessage} disabled={statusDraft.trim() === ''}>Set</button>
          <button class="btn btn-ghost btn-sm" on:click={clearStatusMessage}>Clear</button>
        </div>
      </div>

      <!-- GUARDIAN NETWORK -->
      {#if $myGuardianData.asGuardian.length > 0 || $myGuardianData.asWard.length > 0}
        <div class="network-section">
          <span class="card-eyebrow">Guardians</span>
          {#each $myGuardianData.asGuardian as g}
            <div class="network-item">
              <div class="network-avatar">{(g.wardName || '?')[0].toUpperCase()}</div>
              <div class="network-info">
                <span class="network-name">{g.wardName}</span>
                <span class="network-role">You are their guardian</span>
              </div>
              <span class="network-status-badge" class:active={g.status === 'active'} class:pending={g.status !== 'active'}>{g.status}</span>
              {#if g.status === 'pending' && g.initiatedBy === 'ward'}
                <button class="btn btn-primary btn-sm" on:click={() => socket.emit('approveGuardian', { wardUserId: g.wardId })}>Accept</button>
                <button class="btn btn-danger btn-sm" on:click={() => socket.emit('denyGuardian', { wardUserId: g.wardId })}>Decline</button>
              {:else if g.status === 'active'}
                <button class="btn btn-danger btn-sm" on:click={() => revokeGuardian(g.wardId, null)}>Revoke</button>
              {:else if g.status === 'pending'}
                <button class="btn btn-danger btn-sm" on:click={() => revokeGuardian(g.wardId, null)}>Cancel</button>
              {/if}
            </div>
          {/each}
          {#each $myGuardianData.asWard as g}
            <div class="network-item">
              <div class="network-avatar guardian-av">{(g.guardianName || '?')[0].toUpperCase()}</div>
              <div class="network-info">
                <span class="network-name">{g.guardianName}</span>
                <span class="network-role">Your guardian</span>
              </div>
              <span class="network-status-badge" class:active={g.status === 'active'} class:pending={g.status !== 'active'}>{g.status}</span>
              {#if g.status === 'pending' && g.initiatedBy === 'guardian'}
                <button class="btn btn-primary btn-sm" on:click={() => socket.emit('approveGuardian', { guardianUserId: g.guardianId })}>Accept</button>
                <button class="btn btn-danger btn-sm" on:click={() => socket.emit('denyGuardian', { guardianUserId: g.guardianId })}>Decline</button>
              {:else if g.status === 'pending' && g.initiatedBy === 'ward'}
                <button class="btn btn-danger btn-sm" on:click={() => revokeGuardian(null, g.guardianId)}>Cancel</button>
              {:else if g.status === 'active'}
                <span class="network-caption">Guardian controls this link</span>
              {/if}
            </div>
          {/each}
        </div>
      {/if}

      <!-- PENDING REQUESTS -->
      {#if $pendingIncomingRequests.length > 0}
        <div class="requests-section">
          <span class="card-eyebrow">Requests <span class="req-count">{$pendingIncomingRequests.length}</span></span>
          {#each $pendingIncomingRequests as req, idx}
            <div class="request-card">
              <p class="req-label">{getRequestLabel(req)}</p>
              {#if req.expiresIn}<span class="req-expiry">{req.expiresIn}</span>{/if}
              {#if req.type === 'roomAdmin'}
                <p class="req-vote-meta">{req.approvals || 0} approve · {req.denials || 0} deny · need {Math.floor((req.totalEligible || 1) / 2) + 1} of {req.totalEligible || '?'}</p>
                <div class="req-actions">
                  {#if req.myVote === 'approve'}
                    <span class="badge badge-success badge-xs">You approved</span>
                  {:else if req.myVote === 'deny'}
                    <span class="badge badge-danger badge-xs">You denied</span>
                  {:else}
                    <button class="btn btn-primary btn-sm" on:click={() => approveRequest(req, idx)} disabled={busyRequests.has(req.type + '-' + req.from)}>Approve</button>
                    <button class="btn btn-danger btn-sm" on:click={() => denyRequest(req, idx)} disabled={busyRequests.has(req.type + '-' + req.from)}>Deny</button>
                  {/if}
                </div>
              {:else}
                <div class="req-actions">
                  <button class="btn btn-primary btn-sm" on:click={() => approveRequest(req, idx)} disabled={busyRequests.has(req.type + '-' + req.from)}>Approve</button>
                  <button class="btn btn-danger btn-sm" on:click={() => denyRequest(req, idx)} disabled={busyRequests.has(req.type + '-' + req.from)}>Deny</button>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}

    </div>
  </div>
  <ShareMyRide bind:open={rideShareOpen} />
  {#if walkWithMeOpen}
    <div class="wwm-overlay">
      <div class="wwm-modal">
        <button class="wwm-close" on:click={() => walkWithMeOpen = false} aria-label="Close">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
        <WalkWithMe on:close={() => walkWithMeOpen = false} />
      </div>
    </div>
  {/if}
{/if}

<!-- ── QR code modal (fixed overlay, always in DOM) ──────────────────────── -->
{#if showQr}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="qr-backdrop" on:click={() => showQr = false} role="dialog" aria-modal="true" aria-label="Signal code QR">
    <div class="qr-modal" on:click|stopPropagation>
      <button class="qr-close-btn" on:click={() => showQr = false} aria-label="Close QR code">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
      <span class="qr-title">Your Family Code</span>
      <div class="qr-image-wrap">
        <img
          src="https://api.qrserver.com/v1/create-qr-code/?size=180x180&data={encodeURIComponent(getShareOrigin() + '/#/add-contact/' + $myShareCode)}&margin=6&bgcolor=ffffff&color=0f0f23"
          alt="QR code for signal code {$myShareCode}"
          width="180"
          height="180"
          class="qr-image"
          loading="lazy"
        />
      </div>
      <code class="qr-code-display">{$myShareCode}</code>
      <p class="qr-hint">Anyone who scans this can connect with you on Kinnect</p>
    </div>
  </div>
{/if}

<style>
  /* ── Root container ─────────────────────────────────────────────── */
  .info-root {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  /* ── Metric hero grid ───────────────────────────────────────────── */
  .metrics-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-2);
    margin-bottom: var(--space-2);
  }
  .metric-hero-card {
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-top-color: var(--border-highlight);
    border-radius: var(--radius-lg);
    padding: var(--space-3);
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }
  .metric-wide { grid-column: 1 / -1; }
  .metric-eyebrow {
    font-size: 0.6rem;
    font-weight: 700;
    letter-spacing: 0.10em;
    text-transform: uppercase;
    color: var(--text-tertiary);
  }
  .metric-hero-value {
    font-family: var(--font-mono);
    font-size: var(--text-xl);
    font-weight: 800;
    letter-spacing: -0.02em;
    line-height: 1.1;
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
  }
  .metric-unit {
    font-size: var(--text-xs);
    font-weight: 400;
    color: var(--text-tertiary);
    margin-left: 2px;
  }
  .signal-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-xs);
    color: var(--text-secondary);
    padding: var(--space-1-5) 0;
  }

  /* ── GPS Live Card ──────────────────────────────────────────────── */
  .gps-live-card {
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-top-color: var(--border-highlight);
    border-radius: var(--radius-xl);
    padding: var(--space-4);
    display: flex;
    align-items: center;
    gap: var(--space-3);
    cursor: default;
    transition:
      border-color 500ms var(--ease-out),
      background 500ms var(--ease-out),
      box-shadow 500ms var(--ease-out);
    user-select: none;
  }
  .gps-live-card.is-tracking {
    border-color: rgba(16, 185, 129, 0.30);
    background: linear-gradient(135deg, rgba(16, 185, 129, 0.07) 0%, transparent 60%);
    box-shadow: 0 0 20px rgba(16, 185, 129, 0.08);
  }
  .gps-accuracy-label {
    font-family: var(--font-display);
    font-size: var(--text-base);
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.01em;
    line-height: 1.2;
  }
  .gps-signal-left {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex: 1;
    min-width: 0;
  }

  /* Pulsing GPS dot */
  .gps-ping {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--text-tertiary);
    flex-shrink: 0;
    position: relative;
    transition: background 400ms var(--ease-out);
  }
  .gps-ping.active {
    background: var(--success-500);
    animation: gps-pulse 2.2s ease-in-out infinite;
  }
  .gps-ping.active::before {
    content: '';
    position: absolute;
    inset: -4px;
    border-radius: 50%;
    background: rgba(16, 185, 129, 0.28);
    animation: gps-ring 2.2s ease-out infinite;
  }
  @keyframes gps-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.55; }
  }
  @keyframes gps-ring {
    0%   { transform: scale(1); opacity: 0.6; }
    100% { transform: scale(2.8); opacity: 0; }
  }

  .gps-coord-block {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .coord-line {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-primary);
    letter-spacing: 0.02em;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .gps-sub {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 10px;
    font-weight: 500;
    color: var(--text-tertiary);
    white-space: nowrap;
  }

  /* Accuracy dot (reuse from global pattern) */
  .accuracy-dot {
    display: inline-block;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--gray-400);
    flex-shrink: 0;
  }
  .accuracy-dot.green  { background: var(--success-500); }
  .accuracy-dot.yellow { background: var(--warning-500); }
  .accuracy-dot.red    { background: var(--danger-500); }

  /* Speed badge */
  .speed-pill {
    margin-left: auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0;
    background: rgba(99, 102, 241, 0.09);
    border: 1px solid rgba(99, 102, 241, 0.16);
    border-radius: var(--radius-md);
    padding: var(--space-1-5) var(--space-3);
    flex-shrink: 0;
    min-width: 52px;
    text-align: center;
  }
  .speed-num {
    font-size: var(--text-xl);
    font-weight: 800;
    color: var(--primary-400);
    line-height: 1;
    letter-spacing: -0.04em;
  }
  .speed-unit {
    font-size: 9px;
    color: var(--text-tertiary);
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
  }

  /* GPS Acquiring state */
  .gps-acquire-card {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-4);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-top-color: var(--border-highlight);
    border-radius: var(--radius-xl);
  }
  .acquire-icon { color: var(--primary-400); flex-shrink: 0; }
  .acquire-title {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-secondary);
    margin: 0;
  }
  .acquire-hint {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    margin: 2px 0 0;
  }
  .acquire-hint strong { color: var(--text-secondary); }

  /* Debug stats */
  .tracking-stats {
    padding: var(--space-2) var(--space-3);
    background: var(--surface-inset);
    border-radius: var(--radius-md);
    font-size: var(--text-2xs);
    color: var(--text-secondary);
  }
  .stat-row {
    display: flex;
    justify-content: space-between;
    padding: 1px 0;
  }
  .stat-row span:first-child { font-weight: 600; }
  .latency-good { color: var(--success-500); font-weight: 600; }
  .latency-ok   { color: var(--warning-500); font-weight: 600; }
  .latency-bad  { color: var(--danger-500);  font-weight: 600; }

  /* ── Identity Card ──────────────────────────────────────────────── */
  .identity-card {
    background: linear-gradient(135deg, rgba(99, 102, 241, 0.07) 0%, rgba(139, 92, 246, 0.03) 100%);
    border: 1px solid rgba(99, 102, 241, 0.15);
    border-top-color: rgba(99, 102, 241, 0.28);
    border-radius: var(--radius-xl);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
  .identity-body {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
  }
  .signal-code {
    font-family: var(--font-mono);
    font-size: var(--text-xl);
    font-weight: 700;
    letter-spacing: 0.08em;
    color: var(--primary-400);
  }
  .identity-meta {
    font-size: 11px;
    color: var(--text-tertiary);
  }

  /* ── Safety Zone ────────────────────────────────────────────────── */
  .safety-zone {
    background: rgba(239, 68, 68, 0.03);
    border: 1px solid rgba(239, 68, 68, 0.10);
    border-radius: var(--radius-xl);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }
  :global([data-theme="dark"]) .safety-zone {
    background: rgba(239, 68, 68, 0.05);
    border-color: rgba(239, 68, 68, 0.14);
  }
  .safety-eyebrow { color: rgba(239, 68, 68, 0.55); }

  .safety-actions {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: var(--space-2);
    align-items: stretch;
  }

  /* SOS primary action */
  .sos-action-btn {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
    padding: var(--space-3) var(--space-4);
    background: linear-gradient(135deg, var(--danger-600), var(--danger-500));
    border: 1px solid rgba(239, 68, 68, 0.40);
    border-top-color: rgba(255, 100, 100, 0.48);
    border-radius: var(--radius-lg);
    color: white;
    font-family: var(--font-display);
    font-weight: 800;
    font-size: var(--text-base);
    letter-spacing: -0.01em;
    cursor: pointer;
    min-height: 52px;
    box-shadow: var(--glow-sos), var(--shadow-md);
    animation: sos-urgent-pulse 2.5s ease-in-out infinite;
    transition: opacity 200ms;
  }
  .sos-action-btn.sos-live {
    background: var(--surface-2);
    color: var(--text-secondary);
    border-color: var(--border-default);
    box-shadow: var(--shadow-sm);
    animation: none;
  }
  .sos-action-btn:hover:not(.sos-live) {
    filter: brightness(1.08);
  }
  .sos-icon-wrap {
    width: 26px;
    height: 26px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.18);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .sos-action-btn.sos-live .sos-icon-wrap {
    background: rgba(239, 68, 68, 0.12);
    color: var(--danger-500);
  }

  /* I'm OK secondary action */
  .ok-action-btn {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 3px;
    padding: var(--space-2) var(--space-3);
    background: rgba(16, 185, 129, 0.08);
    border: 1px solid rgba(16, 185, 129, 0.18);
    border-radius: var(--radius-lg);
    color: var(--success-500);
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    cursor: pointer;
    min-height: 52px;
    min-width: 60px;
    transition: background 150ms var(--ease-out), box-shadow 150ms var(--ease-out);
  }
  .ok-action-btn:hover {
    background: rgba(16, 185, 129, 0.14);
    box-shadow: 0 0 16px rgba(16, 185, 129, 0.18);
  }
  .ok-action-btn--active {
    background: rgba(239, 68, 68, 0.10);
    border-color: rgba(239, 68, 68, 0.25);
    color: var(--danger-500);
  }
  .ok-action-btn--active:hover {
    background: rgba(239, 68, 68, 0.16);
  }
  .ok-action-btn--crowd {
    background: rgba(245, 158, 11, 0.12);
    border-color: rgba(245, 158, 11, 0.28);
    color: var(--warning-500, #f59e0b);
  }
  .ok-action-btn--crowd:hover {
    background: rgba(245, 158, 11, 0.18);
  }

  /* ── Ambient Status Message ──────────────────────────────────────── */
  .status-msg-zone {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    background: var(--surface-1);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-xl);
    padding: var(--space-3) var(--space-4);
  }
  .status-msg-row {
    display: flex;
    gap: 6px;
    align-items: center;
  }
  .status-msg-input {
    flex: 1;
    font-size: var(--text-sm);
    padding: 7px 10px;
    border-radius: var(--radius-md);
    border: 1px solid var(--border-subtle);
    background: var(--surface-3);
    color: var(--text-primary);
    min-width: 0;
  }
  .status-expiry-select {
    font-size: var(--text-xs);
    padding: 7px 6px;
    border-radius: var(--radius-md);
    border: 1px solid var(--border-subtle);
    background: var(--surface-3);
    color: var(--text-secondary);
    cursor: pointer;
  }
  .status-msg-actions {
    display: flex;
    gap: 6px;
  }

  /* ── Network / Guardian section ──────────────────────────────────── */
  .network-section,
  .requests-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  /* ── Request cards ───────────────────────────────────────────────── */
  .request-card {
    padding: var(--space-3) var(--space-4);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-left: 3px solid var(--warning-500);
    border-radius: var(--radius-lg);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
  .req-label {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text-primary);
    margin: 0;
  }
  .req-expiry {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }
  .req-vote-meta {
    font-size: var(--text-xs);
    color: var(--text-secondary);
    margin: 0;
  }
  .req-actions {
    display: flex;
    gap: var(--space-2);
  }
  .req-count {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 18px;
    height: 18px;
    border-radius: var(--radius-full);
    background: var(--warning-500);
    color: white;
    font-size: 10px;
    font-weight: 800;
    padding: 0 4px;
    margin-left: 4px;
  }

  /* ── Signal code actions group ──────────────────────────────────── */
  .signal-btns {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    flex-shrink: 0;
  }

  /* ── QR icon button ──────────────────────────────────────────────── */
  .qr-icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: var(--radius-md);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    color: var(--text-tertiary);
    cursor: pointer;
    flex-shrink: 0;
    transition: background 150ms var(--ease-out), color 150ms, transform 120ms var(--ease-spring);
  }
  .qr-icon-btn:hover {
    background: var(--surface-2);
    color: var(--primary-400);
    transform: scale(1.08);
  }
  .qr-icon-btn:active { transform: scale(0.93); }

  /* ── QR modal ────────────────────────────────────────────────────── */
  .qr-backdrop {
    position: fixed;
    inset: 0;
    z-index: 6000;
    background: rgba(5, 5, 18, 0.72);
    backdrop-filter: blur(8px) saturate(1.4);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4);
    animation: qr-fade-in 180ms var(--ease-out) both;
  }
  @keyframes qr-fade-in { from { opacity: 0; } to { opacity: 1; } }
  @media (prefers-reduced-motion: reduce) { .qr-backdrop { animation: none; } }

  .qr-modal {
    position: relative;
    background: var(--surface-1);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-2xl, 20px);
    padding: var(--space-6, 24px) var(--space-5, 20px) var(--space-4);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    max-width: 260px;
    width: 100%;
    box-shadow: 0 24px 60px rgba(0,0,0,0.5);
    animation: qr-slide-up 220ms var(--ease-spring, cubic-bezier(0.34,1.56,0.64,1)) both;
  }
  @keyframes qr-slide-up { from { transform: translateY(16px) scale(0.95); opacity: 0; } to { transform: none; opacity: 1; } }
  @media (prefers-reduced-motion: reduce) { .qr-modal { animation: none; } }

  .qr-close-btn {
    position: absolute;
    top: var(--space-3);
    right: var(--space-3);
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-md);
    background: var(--surface-2);
    border: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 120ms, color 120ms;
  }
  .qr-close-btn:hover { background: var(--surface-3, var(--surface-2)); color: var(--text-primary); }

  .qr-title {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
  }

  .qr-image-wrap {
    background: white;
    border-radius: var(--radius-lg);
    padding: 8px;
    line-height: 0;
    box-shadow: 0 4px 16px rgba(0,0,0,0.2);
  }

  .qr-image {
    display: block;
    width: 180px;
    height: 180px;
    border-radius: 4px;
  }

  .qr-code-display {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--primary-400);
    letter-spacing: 0.08em;
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: 4px 10px;
  }

  .qr-hint {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    text-align: center;
    line-height: 1.45;
    margin: 0;
    max-width: 200px;
  }

  /* Walk With Me modal */
  .wwm-overlay {
    position: fixed; inset: 0; z-index: 60;
    background: rgba(0,0,0,0.55);
    display: flex; align-items: center; justify-content: center;
    animation: wwm-fade-in 0.2s ease;
  }
  @keyframes wwm-fade-in { from { opacity: 0; } }
  .wwm-modal {
    position: relative;
    width: min(380px, 92vw);
    max-height: 85vh;
    overflow-y: auto;
    background: var(--surface-0, #0a0e1a);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 20px;
    box-shadow: 0 16px 48px rgba(0,0,0,0.5);
  }
  .wwm-close {
    position: absolute; top: 12px; right: 12px; z-index: 2;
    width: 32px; height: 32px; border-radius: 50%;
    background: rgba(255,255,255,0.06); border: none;
    display: flex; align-items: center; justify-content: center;
    color: rgba(255,255,255,0.4); cursor: pointer;
  }
  .wwm-close:hover { background: rgba(255,255,255,0.12); }
</style>
