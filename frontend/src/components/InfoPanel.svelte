<script>
  import { createEventDispatcher } from 'svelte';
  import { myLocation, tracking } from '../lib/stores/map.js';
  import { authUser } from '../lib/stores/auth.js';
  import { myShareCode, myContactInfo } from '../lib/stores/rooms.js';
  import { socket } from '../lib/socket.js';
  import { banner, mySosActive } from '../lib/stores/sos.js';
  import { myGuardianData, pendingIncomingRequests } from '../lib/stores/guardians.js';
  import { formatCoordinate } from '../lib/tracking.js';
  import { trackingMetrics } from '../lib/stores/metrics.js';
  import { latencyMetrics } from '../lib/stores/latency.js';
  import CopyButton from './primitives/CopyButton.svelte';

  export let embedded = false;
  let statsOpen = false;
  let debugTaps = 0;

  const dispatch = createEventDispatcher();

  function toggleSOS() {
    if (!socket.connected) {
      banner.set({ type: 'info', text: 'Not connected — reconnecting. Try again in a moment.', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
      return;
    }
    if (!$mySosActive) socket.emit('triggerSOS', { reason: 'SOS' });
    else socket.emit('cancelSOS');
  }

  function checkIn() {
    socket.emit('checkInAck');
    banner.set({ type: 'info', text: 'Check-in sent.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
  }

  function onMyWay() {
    socket.emit('onMyWay', {});
    banner.set({ type: 'info', text: 'On My Way broadcast sent.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
  }

  function attest() {
    socket.emit('attest');
    banner.set({ type: 'info', text: 'Safety confirmed.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
  }

  function approveRequest(req, idx) {
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
  }

  function denyRequest(req, idx) {
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
    : $tracking ? 'Acquiring fix…' : 'Tap Track to begin';
</script>

<!-- ═══════════════════════════════════════════════════════════════════════
     EMBEDDED MODE (used in sidebar + bottom sheet)
     ═══════════════════════════════════════════════════════════════════════ -->
{#if embedded}
  <div class="panel-body info-root">

    <!-- ── GPS LIVE STATUS ──────────────────────────────────────────── -->
    {#if $myLocation}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <div class:is-tracking={$tracking} on:click={tapGps} role="presentation" aria-hidden="true">
        <div class="metrics-grid">
          <div class="metric-hero-card">
            <span class="metric-eyebrow">Latitude</span>
            <span class="metric-hero-value">{formatCoordinate($myLocation.latitude)}<span class="metric-unit">° N</span></span>
          </div>
          <div class="metric-hero-card">
            <span class="metric-eyebrow">Longitude</span>
            <span class="metric-hero-value">{formatCoordinate($myLocation.longitude)}<span class="metric-unit">° E</span></span>
          </div>
          {#if ($myLocation.speed || 0) >= 1}
            <div class="metric-hero-card metric-wide">
              <span class="metric-eyebrow">Speed</span>
              <span class="metric-hero-value">{$myLocation.speed}<span class="metric-unit">km/h</span></span>
            </div>
          {/if}
        </div>
        <div class="signal-row">
          <span class="gps-ping" class:active={$tracking && $trackingMetrics.lastAccuracy != null}></span>
          {#if $trackingMetrics.lastAccuracy != null}
            <span class="accuracy-dot {accClass}"></span>
          {/if}
          <span>{accLabel}{$myLocation.formattedTime ? ' · ' + $myLocation.formattedTime : ''}</span>
        </div>
      </div>
    {:else}
      <div class="gps-acquire-card animate-breathe">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="acquire-icon" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M16.24 7.76a6 6 0 0 1 0 8.49"/><path d="M7.76 7.76a6 6 0 0 0 0 8.49"/><path d="M20.49 3.51a12 12 0 0 1 0 16.97"/><path d="M3.51 3.51a12 12 0 0 0 0 16.97"/></svg>
        <div>
          <p class="acquire-title">Finding your position…</p>
          <p class="acquire-hint">Tap <strong>Track</strong> to start</p>
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
      <span class="card-eyebrow">Signal Code</span>
      <div class="identity-body">
        <code class="signal-code">{$myShareCode || '—'}</code>
        <CopyButton text={$myShareCode || ''} label="Copy" />
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
          {$mySosActive ? 'Cancel SOS' : 'Emergency SOS'}
        </button>
        <button class="ok-action-btn" on:click={checkIn} aria-label="Send check-in — I'm OK">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
          I'm OK
        </button>
      </div>
      <div class="safety-actions" style="margin-top:6px;">
        <button class="ok-action-btn" on:click={onMyWay} aria-label="Broadcast On My Way">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
          On My Way
        </button>
        <button class="ok-action-btn" on:click={attest} aria-label="Confirm your location is genuine" title="Confirm your location is real">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
          I'm Safe
        </button>
      </div>
    </div>

    <!-- ── GUARDIAN NETWORK ──────────────────────────────────────────── -->
    {#if $myGuardianData.asGuardian.length > 0 || $myGuardianData.asWard.length > 0}
      <div class="network-section">
        <span class="card-eyebrow">Guardian Network</span>
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
                  <button class="btn btn-primary btn-sm" on:click={() => approveRequest(req, idx)}>Approve</button>
                  <button class="btn btn-danger btn-sm" on:click={() => denyRequest(req, idx)}>Deny</button>
                {/if}
              </div>
            {:else}
              <div class="req-actions">
                <button class="btn btn-primary btn-sm" on:click={() => approveRequest(req, idx)}>Approve</button>
                <button class="btn btn-danger btn-sm" on:click={() => denyRequest(req, idx)}>Deny</button>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

  </div>

<!-- ═══════════════════════════════════════════════════════════════════════
     PANEL SHELL MODE (desktop standalone panel)
     ═══════════════════════════════════════════════════════════════════════ -->
{:else}
  <div class="panel-shell panel-left panel-base">
    <div class="panel-header">
      <h3>Info</h3>
      <button class="btn btn-icon btn-ghost" aria-label="Close info panel" on:click={() => dispatch('close')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>

    <div class="panel-body info-root">

      <!-- GPS LIVE STATUS -->
      {#if $myLocation}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <div class:is-tracking={$tracking} on:click={tapGps} role="presentation" aria-hidden="true">
          <div class="metrics-grid">
            <div class="metric-hero-card">
              <span class="metric-eyebrow">Latitude</span>
              <span class="metric-hero-value">{formatCoordinate($myLocation.latitude)}<span class="metric-unit">° N</span></span>
            </div>
            <div class="metric-hero-card">
              <span class="metric-eyebrow">Longitude</span>
              <span class="metric-hero-value">{formatCoordinate($myLocation.longitude)}<span class="metric-unit">° E</span></span>
            </div>
            {#if ($myLocation.speed || 0) >= 1}
              <div class="metric-hero-card metric-wide">
                <span class="metric-eyebrow">Speed</span>
                <span class="metric-hero-value">{$myLocation.speed}<span class="metric-unit">km/h</span></span>
              </div>
            {/if}
          </div>
          <div class="signal-row">
            <span class="gps-ping" class:active={$tracking && $trackingMetrics.lastAccuracy != null}></span>
            {#if $trackingMetrics.lastAccuracy != null}
              <span class="accuracy-dot {accClass}"></span>
            {/if}
            <span>{accLabel}{$myLocation.formattedTime ? ' · ' + $myLocation.formattedTime : ''}</span>
          </div>
        </div>
      {:else}
        <div class="gps-acquire-card animate-breathe">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="acquire-icon" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M16.24 7.76a6 6 0 0 1 0 8.49"/><path d="M7.76 7.76a6 6 0 0 0 0 8.49"/><path d="M20.49 3.51a12 12 0 0 1 0 16.97"/><path d="M3.51 3.51a12 12 0 0 0 0 16.97"/></svg>
          <div>
            <p class="acquire-title">Finding your position…</p>
            <p class="acquire-hint">Hit <strong>Track</strong> in the top bar to begin.</p>
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
        <span class="card-eyebrow">Signal Code</span>
        <div class="identity-body">
          <code class="signal-code">{$myShareCode || '—'}</code>
          <CopyButton text={$myShareCode || ''} label="Copy" />
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
            {$mySosActive ? 'Cancel SOS' : 'Emergency SOS'}
          </button>
          <button class="ok-action-btn" on:click={checkIn} aria-label="Send check-in — I'm OK">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
            I'm OK
          </button>
        </div>
      </div>

      <!-- GUARDIAN NETWORK -->
      {#if $myGuardianData.asGuardian.length > 0 || $myGuardianData.asWard.length > 0}
        <div class="network-section">
          <span class="card-eyebrow">Guardian Network</span>
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
                    <button class="btn btn-primary btn-sm" on:click={() => approveRequest(req, idx)}>Approve</button>
                    <button class="btn btn-danger btn-sm" on:click={() => denyRequest(req, idx)}>Deny</button>
                  {/if}
                </div>
              {:else}
                <div class="req-actions">
                  <button class="btn btn-primary btn-sm" on:click={() => approveRequest(req, idx)}>Approve</button>
                  <button class="btn btn-danger btn-sm" on:click={() => denyRequest(req, idx)}>Deny</button>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}

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
    border-radius: var(--radius-xl);
    padding: var(--space-4);
    display: flex;
    align-items: center;
    gap: var(--space-3);
    cursor: default;
    transition: border-color 500ms var(--ease-out), background 500ms var(--ease-out);
    user-select: none;
  }
  .gps-live-card.is-tracking {
    border-color: rgba(16, 185, 129, 0.22);
    background: linear-gradient(135deg, rgba(16, 185, 129, 0.05) 0%, transparent 60%);
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
    border-radius: var(--radius-xl);
  }
  .acquire-icon { color: var(--text-tertiary); flex-shrink: 0; }
  .acquire-title {
    font-size: var(--text-sm);
    font-weight: 600;
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
    font-family: var(--font-sans);
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
    font-family: var(--font-sans);
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
</style>
