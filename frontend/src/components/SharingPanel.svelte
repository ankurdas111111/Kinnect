<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { socket } from '../lib/socket.js';
  import { banner, myLiveLinks } from '../lib/stores/sos.js';
  import { myRooms, myShareCode, myContactInfo } from '../lib/stores/rooms.js';
  import { myContacts } from '../lib/stores/contacts.js';
  import { myGuardianData } from '../lib/stores/guardians.js';
  import { authUser } from '../lib/stores/auth.js';
  import { otherUsers, focusUser } from '../lib/stores/map.js';
  import { getPresenceState, getPresenceText } from '../lib/tracking.js';
  import { getUserColor } from '../lib/getUserColor.js';
  import CopyButton from './primitives/CopyButton.svelte';

  function locateContact(userId) {
    // Find by userId in otherUsers map
    for (const [sid, u] of $otherUsers) {
      if (u.userId === userId) {
        focusUser.set(sid);
        return;
      }
    }
    // Fallback: set userId and let map resolve
    focusUser.set(userId);
  }

  export let embedded = false;

  const dispatch = createEventDispatcher();

  let roomName = '';
  let joinCode = '';
  let contactCode = '';
  let loading = { createRoom: false, joinRoom: false, addContact: false };
  let selectedLinkDuration = '24h';
  let loadingTimers = {};

  function withLoading(key, fn) {
    if (loading[key]) return;
    loading = { ...loading, [key]: true };
    fn();
    if (loadingTimers[key]) clearTimeout(loadingTimers[key]);
    loadingTimers[key] = setTimeout(() => clearLoading(key), 8000);
  }

  function clearLoading(key) {
    if (loadingTimers[key]) {
      clearTimeout(loadingTimers[key]);
      delete loadingTimers[key];
    }
    if (!loading[key]) return;
    loading = { ...loading, [key]: false };
  }

  function clearRoomLoading() {
    clearLoading('createRoom');
    clearLoading('joinRoom');
  }

  function clearContactLoading() {
    clearLoading('addContact');
  }

  function createRoom() {
    withLoading('createRoom', () => { socket.emit('createRoom', { name: roomName.trim() }); roomName = ''; });
  }
  function joinRoom() {
    if (!joinCode.trim()) return;
    withLoading('joinRoom', () => { socket.emit('joinRoom', { code: joinCode.trim().toUpperCase() }); joinCode = ''; });
  }
  function addContact() {
    if (!contactCode.trim()) return;
    withLoading('addContact', () => { socket.emit('addContact', { shareCode: contactCode.trim().toUpperCase() }); contactCode = ''; });
  }
  function leaveRoom(code) { socket.emit('leaveRoom', { code }); }
  function removeContact(userId) { socket.emit('removeContact', { userId }); }
  let guardianDurations = {};
  let roomAdminDurations = {};

  function requestAdmin(code) {
    var dur = roomAdminDurations[code] || null;
    socket.emit('requestRoomAdmin', { roomCode: code, expiresIn: dur });
  }
  function revokeAdmin(code, uid) { socket.emit('revokeRoomAdmin', { roomCode: code, userId: uid }); }
  function voteRoomAdmin(code, userId, vote) { socket.emit('voteRoomAdmin', { roomCode: code, userId, vote }); }
  function hasPendingAdminRequest(room) { return (room.pendingAdminRequests || []).some(r => r.isMe); }
  function requestGuardian(userId) {
    var dur = guardianDurations[userId] || null;
    socket.emit('requestGuardian', { contactUserId: userId, expiresIn: dur });
  }
  function inviteGuardian(userId) {
    var dur = guardianDurations[userId] || null;
    socket.emit('inviteGuardian', { contactUserId: userId, expiresIn: dur });
  }

  function isPendingGuardianOf(userId) {
    return $myGuardianData.asGuardian?.some(g => g.wardId === userId && g.status === 'pending');
  }
  function isPendingWardOf(userId) {
    return $myGuardianData.asWard?.some(g => g.guardianId === userId && g.status === 'pending');
  }
  function createLiveLink(dur) {
    socket.emit('createLiveLink', { duration: dur === 'forever' ? null : dur });
  }
  function generateLiveLink() { createLiveLink(selectedLinkDuration); }
  async function copyLink(url) {
    // On Android native: use system share sheet so users can send via WhatsApp, SMS, etc.
    const isNative = typeof window !== 'undefined' && window.Capacitor?.isNativePlatform?.();
    if (isNative) {
      try {
        const { Share } = await import('@capacitor/share');
        await Share.share({ title: 'Watch my live location', text: 'Follow my live location on Kinnect', url, dialogTitle: 'Share live link' });
        return;
      } catch (_) {
        // User dismissed share sheet or plugin failed — fall through to clipboard
      }
    }
    navigator.clipboard.writeText(url).catch(() => {});
    banner.set({ type: 'info', text: 'Link copied. Let the watching begin.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
  }
  function revokeLink(token) { socket.emit('revokeLiveLink', { token }); }

  function isGuardianOf(userId) { return $myGuardianData.asGuardian?.some(g => g.wardId === userId && g.status === 'active'); }
  function isWardOf(userId) { return $myGuardianData.asWard?.some(g => g.guardianId === userId && g.status === 'active'); }

  $: hasAny = ($myRooms.length > 0 || $myContacts.length > 0);

  function getInitials(name) {
    if (!name) return '?';
    return name.split(' ').filter(Boolean).map(n => n[0]).join('').toUpperCase().slice(0, 2);
  }

  // Cross-reference contacts with live otherUsers map for presence
  function getContactUser(userId) {
    for (const u of $otherUsers.values()) {
      if (u.userId === userId) return u;
    }
    return null;
  }

  function contactPresenceState(userId) {
    const u = getContactUser(userId);
    if (!u) return 'gone';
    return getPresenceState(u);
  }

  function contactPresenceText(userId) {
    const u = getContactUser(userId);
    if (!u) return 'Not on map';
    return getPresenceText(u);
  }

  function contactBatteryLow(userId) {
    const u = getContactUser(userId);
    return u && u.batteryPct != null && u.batteryPct <= 20;
  }

  onMount(() => {
    var onRoomCreated = () => clearRoomLoading();
    var onRoomJoined = () => clearRoomLoading();
    var onRoomError = () => clearRoomLoading();
    var onContactAdded = () => clearContactLoading();
    var onContactError = () => clearContactLoading();
    socket.on('roomCreated', onRoomCreated);
    socket.on('roomJoined', onRoomJoined);
    socket.on('roomError', onRoomError);
    socket.on('contactAdded', onContactAdded);
    socket.on('contactError', onContactError);

    return () => {
      socket.off('roomCreated', onRoomCreated);
      socket.off('roomJoined', onRoomJoined);
      socket.off('roomError', onRoomError);
      socket.off('contactAdded', onContactAdded);
      socket.off('contactError', onContactError);
      Object.keys(loadingTimers).forEach((k) => clearTimeout(loadingTimers[k]));
    };
  });
</script>

{#if embedded}
  <div class="panel-body sharing-root">

    <!-- ── ROOMS ─────────────────────────────────────────────────── -->
    <div class="sharing-section">
      <div class="sharing-section-header">
        <span class="card-eyebrow">Groups</span>
      </div>
      <div class="rooms-create-row">
        <input class="input" bind:value={roomName} placeholder="New group name" />
        <button class="btn btn-primary btn-sm" on:click={createRoom} disabled={loading.createRoom}>{loading.createRoom ? '…' : 'Create'}</button>
      </div>
      <div class="rooms-join-row">
        <input class="input" bind:value={joinCode} placeholder="Enter room code to join" on:keydown={e => e.key === 'Enter' && joinRoom()} />
        <button class="btn btn-secondary btn-sm" on:click={joinRoom} disabled={loading.joinRoom}>{loading.joinRoom ? '…' : 'Join'}</button>
      </div>

      {#if $myRooms.length === 0}
        <p class="empty-state">No groups yet. Create one and invite your people.</p>
      {:else}
        <div class="rooms-list">
          {#each $myRooms as room}
            <div class="room-card animate-slide-up" style="border-top: 3px solid {getUserColor(room.code)}; border-top-left-radius: var(--radius-xl); border-top-right-radius: var(--radius-xl);">
              <div class="room-card-header">
                <div class="room-card-icon">{(room.name || 'G')[0].toUpperCase()}</div>
                <div class="room-card-meta">
                  <span class="room-card-name">
                    {room.name}
                    {#if room.myRoomRole === 'admin'}<span class="badge badge-success badge-xs" style="margin-left:6px">Admin</span>{/if}
                  </span>
                  <span class="room-card-code">{room.code}</span>
                </div>
                <button class="btn btn-danger btn-sm" on:click={() => leaveRoom(room.code)}>Leave</button>
              </div>

              {#if (room.members || []).length > 0}
                <div class="room-members-row">
                  {#each (room.members || []) as m}
                    <button class="room-member-chip" on:click={() => locateContact(m.userId)} title="Find {m.displayName || 'member'} on map">
                      <span class="room-member-avatar">{(m.displayName || '?')[0].toUpperCase()}</span>
                      {m.displayName || 'Member'}
                      {#if m.roomRole === 'admin'}<span class="badge badge-success badge-xs">A</span>{/if}
                      {#if room.myRoomRole === 'admin' && m.userId !== $authUser?.userId && m.roomRole === 'admin'}
                        <button class="btn-revoke-inline" on:click|stopPropagation={() => revokeAdmin(room.code, m.userId)}>×</button>
                      {/if}
                    </button>
                  {/each}
                </div>
              {/if}

              <div class="room-card-actions">
                {#if room.myRoomRole !== 'admin' && !hasPendingAdminRequest(room)}
                  <select class="duration-select" bind:value={roomAdminDurations[room.code]}>
                    <option value={null}>Permanent</option>
                    <option value="1h">1 Hour</option>
                    <option value="6h">6 Hours</option>
                    <option value="24h">24 Hours</option>
                    <option value="7d">7 Days</option>
                    <option value="30d">30 Days</option>
                  </select>
                  <button class="btn btn-secondary btn-sm" on:click={() => requestAdmin(room.code)}>Request Admin</button>
                {:else if hasPendingAdminRequest(room)}
                  <span class="badge badge-warning badge-xs">Admin Requested</span>
                {/if}
              </div>

              {#if (room.pendingAdminRequests || []).length > 0}
                <div class="pending-admin-section">
                  {#each room.pendingAdminRequests as par}
                    <div class="pending-admin-item">
                      <div class="pending-admin-info">
                        <span class="text-sm">{par.isMe ? 'Your admin request' : `${par.fromName} wants Admin`}</span>
                        {#if par.expiresIn}<span class="mini"> ({par.expiresIn})</span>{/if}
                        <span class="mini vote-count">{par.approvals}/{par.totalEligible} approve, {par.denials}/{par.totalEligible} deny (need {Math.floor(par.totalEligible / 2) + 1})</span>
                      </div>
                      {#if !par.isMe}
                        <div class="pending-admin-actions">
                          {#if par.myVote === 'approve'}
                            <span class="badge badge-success badge-xs">Approved</span>
                          {:else if par.myVote === 'deny'}
                            <span class="badge badge-danger badge-xs">Denied</span>
                          {:else}
                            <button class="btn btn-primary btn-xs" on:click={() => voteRoomAdmin(room.code, par.from, 'approve')}>Approve</button>
                            <button class="btn btn-danger btn-xs" on:click={() => voteRoomAdmin(room.code, par.from, 'deny')}>Deny</button>
                          {/if}
                        </div>
                      {/if}
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- ── YOUR PEOPLE ────────────────────────────────────────────── -->
    <div class="sharing-section">
      <div class="sharing-section-header">
        <span class="card-eyebrow">Your People</span>
      </div>
      <div class="rooms-create-row">
        <input class="input" bind:value={contactCode} placeholder="Paste their signal code" on:keydown={e => e.key === 'Enter' && addContact()} />
        <button class="btn btn-primary btn-sm" on:click={addContact} disabled={loading.addContact}>{loading.addContact ? '…' : 'Add'}</button>
      </div>
      {#if $myContacts.length === 0}
        <p class="empty-state">No contacts yet. Share your signal code and connect.</p>
      {:else}
          {#each $myContacts as c, i}
            {@const pState = contactPresenceState(c.userId)}
            {@const pText  = contactPresenceText(c.userId)}
            {@const batLow = contactBatteryLow(c.userId)}
            {@const contactUser = getContactUser(c.userId)}
            {@const actCtx = contactUser?.activityContext || ''}
            <div class="person-card animate-slide-up stagger-item" style="animation-delay:{i*40}ms">
              <!-- Presence avatar ring -->
              <div class="person-ring-wrap">
                <div
                  class="person-avatar-circle state-{pState}"
                  class:battery-low={batLow}
                  style="--person-color:{getUserColor(c.userId)};"
                  role="img"
                  aria-label="{c.displayName}, {pText}"
                >
                  {getInitials(c.displayName)}
                </div>
              </div>

              <!-- Name + presence status -->
              <div class="person-name-block">
                <span class="person-name">
                  {c.displayName}
                  {#if isGuardianOf(c.userId)}<span class="badge badge-primary badge-xs" style="margin-left:4px">Ward</span>{/if}
                  {#if isWardOf(c.userId)}<span class="badge badge-primary badge-xs" style="margin-left:4px">Guardian</span>{/if}
                  {#if isPendingGuardianOf(c.userId) || isPendingWardOf(c.userId)}<span class="badge badge-warning badge-xs" style="margin-left:4px">Pending</span>{/if}
                </span>
                <span class="person-status" class:status-now={pState==='now'} class:status-sos={pState==='sos'}>{pText}</span>
                {#if actCtx}<span class="activity-chip">{actCtx}</span>{/if}
              </div>

              <!-- Actions: locate + context menu -->
              <div class="contact-actions-col">
                <button class="locate-pill" on:click={() => locateContact(c.userId)} title="Find {c.displayName} on map" aria-label="Locate {c.displayName}">
                  <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="10" r="3"/><path d="M12 2a8 8 0 0 0-8 8c0 1.892.402 3.13 1.5 4.5L12 22l6.5-7.5c1.098-1.37 1.5-2.608 1.5-4.5a8 8 0 0 0-8-8z"/></svg>
                </button>
                <button class="btn btn-danger btn-sm" on:click={() => removeContact(c.userId)}>Remove</button>
                {#if !isGuardianOf(c.userId) && !isWardOf(c.userId) && !isPendingGuardianOf(c.userId) && !isPendingWardOf(c.userId)}
                  <select class="duration-select" bind:value={guardianDurations[c.userId]}>
                    <option value={null}>Permanent</option>
                    <option value="1h">1h</option>
                    <option value="24h">24h</option>
                    <option value="7d">7d</option>
                    <option value="30d">30d</option>
                  </select>
                  <button class="btn btn-secondary btn-sm" on:click={() => requestGuardian(c.userId)} title="You watch them">Watch</button>
                  <button class="btn btn-secondary btn-sm" on:click={() => inviteGuardian(c.userId)} title="They watch you">Be Watched</button>
                {/if}
              </div>
            </div>
          {/each}
      {/if}
    </div>

    <!-- ── LIVE BROADCASTS ───────────────────────────────────────── -->
    <div class="sharing-section">
      <div class="sharing-section-header">
        <span class="card-eyebrow">Live Broadcasts</span>
      </div>
      <div class="live-link-toolbar">
        <div class="duration-pills" role="group" aria-label="Broadcast duration">
          {#each [['1h','1h'],['6h','6h'],['24h','24h'],['48h','48h'],['forever','∞']] as [val, label]}
            <button
              class="pill-btn"
              class:pill-active={selectedLinkDuration === val}
              on:click={() => selectedLinkDuration = val}
              aria-pressed={selectedLinkDuration === val}
            >{label}</button>
          {/each}
        </div>
        <button class="btn btn-primary btn-sm" on:click={generateLiveLink}>Start Broadcast</button>
      </div>
      {#if $myLiveLinks.length === 0}
        <p class="empty-state">No broadcasts running.</p>
      {:else}
        <div class="broadcasts-list">
          {#each $myLiveLinks as link, i}
            {@const url = window.location.origin + '/#/live/' + link.token}
            <div class="broadcast-card card-glow-link animate-slide-up stagger-item" style="animation-delay:{i*40}ms">
              <div class="broadcast-header">
                <span class="rec-dot animate-rec-blink" aria-hidden="true"></span>
                <span class="broadcast-label">Live</span>
                <span class="broadcast-expiry">{link.expiresAt ? 'Expires ' + new Date(link.expiresAt).toLocaleTimeString() : 'Until I Say Stop'}</span>
              </div>
              <div class="live-link-actions">
                <CopyButton text={url} label="Copy Link" />
                <button class="btn btn-danger btn-sm" on:click={() => revokeLink(link.token)}>End Broadcast</button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    {#if !hasAny}
      <div class="empty-state-hero">
        <p>Just you out here. Share your signal code to connect with someone.</p>
      </div>
    {/if}
  </div>
{:else}
  <div class="panel-shell panel-left panel-base">
    <div class="panel-header">
      <h3>Sharing</h3>
      <button class="btn btn-icon btn-ghost" aria-label="Close sharing panel" on:click={() => dispatch('close')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
    <div class="panel-body">
      <p class="mini">Use the sidebar for sharing controls.</p>
    </div>
  </div>
{/if}

<style>
  /* ── Sharing root container ──────────────────────────────────── */
  .sharing-root {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  /* ── Section structure — no dividers, just grouping ─────────── */
  .sharing-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
  .sharing-section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--space-1);
  }

  /* Create / join rows */
  .rooms-create-row,
  .rooms-join-row {
    display: flex;
    gap: var(--space-2);
  }
  .rooms-create-row .input,
  .rooms-join-row .input {
    flex: 1;
  }

  /* Rooms list */
  .rooms-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  /* Broadcasts list */
  .broadcasts-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  /* Empty state */
  .empty-state {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    padding: var(--space-2) 0;
    font-style: italic;
  }
  .empty-state-hero {
    padding: var(--space-4);
    text-align: center;
    font-size: var(--text-sm);
    color: var(--text-tertiary);
    border: 1px dashed var(--border-subtle);
    border-radius: var(--radius-xl);
  }

  /* Revoke inline button */
  .btn-revoke-inline {
    background: none;
    border: none;
    cursor: pointer;
    color: var(--danger-400);
    font-size: 13px;
    font-weight: 700;
    padding: 0 2px;
    line-height: 1;
    border-radius: 2px;
    transition: color 100ms;
  }
  .btn-revoke-inline:hover { color: var(--danger-500); }

  /* ── Contact cards — person-first ─────────────────────────────── */
  .contact-card {
    display: grid;
    grid-template-columns: 38px 1fr auto;
    gap: var(--space-3);
    align-items: start;
    padding: var(--space-3) var(--space-3);
    border-radius: var(--radius-lg);
    border: 1px solid var(--border-subtle);
    background: var(--surface-1);
    transition: background 180ms var(--ease-out), border-color 180ms var(--ease-out), transform 150ms var(--ease-spring), box-shadow 180ms var(--ease-out);
  }
  .contact-card:hover {
    background: var(--surface-2);
    border-color: var(--border-default);
    transform: translateY(-1px);
    box-shadow: var(--shadow-sm);
  }

  .contact-avatar-wrap { padding-top: 2px; }

  .contact-avatar {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    background: var(--primary-100);
    color: var(--primary-700);
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    font-size: var(--text-xs);
    flex-shrink: 0;
    text-transform: uppercase;
    transition: box-shadow 300ms var(--ease-out);
  }

  .contact-avatar.avatar-guardian {
    background: rgba(139, 92, 246, 0.12);
    color: var(--accent-guardian);
    box-shadow: 0 0 0 2px rgba(139, 92, 246, 0.20);
  }

  :global([data-theme="dark"]) .contact-avatar {
    background: rgba(99, 102, 241, 0.18);
    color: var(--primary-300);
  }

  .contact-name-row {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    flex-wrap: wrap;
  }

  /* ── Contact actions column ────────────────────────────────────────── */
  .contact-actions-col {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: var(--space-1);
    flex-shrink: 0;
  }

  .locate-pill {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: rgba(99, 102, 241, 0.10);
    border: 1px solid rgba(99, 102, 241, 0.22);
    color: var(--primary-400);
    cursor: pointer;
    transition: background 150ms var(--ease-out), transform 120ms var(--ease-spring);
  }
  .locate-pill:hover {
    background: rgba(99, 102, 241, 0.20);
    transform: scale(1.10);
  }

  /* ── Room members ────────────────────────────────────────────────── */
  .room-members {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 8px;
    margin-top: 4px;
    font-size: var(--text-xs);
    color: var(--text-secondary);
  }
  .room-member {
    display: inline-flex;
    align-items: center;
    gap: 3px;
  }
  .member-locate-btn {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    background: none;
    border: 1px solid transparent;
    cursor: pointer;
    color: var(--text-secondary);
    padding: 2px 6px;
    border-radius: var(--radius-sm);
    font: inherit;
    font-size: var(--text-xs);
    transition: color 150ms, background 150ms, border-color 150ms;
    line-height: 1.3;
  }
  .member-locate-btn:hover {
    color: var(--primary-500);
    background: var(--surface-inset);
    border-color: var(--border-glow-primary);
  }
  .member-locate-btn svg { flex-shrink: 0; opacity: 0.5; transition: opacity 150ms; }
  .member-locate-btn:hover svg { opacity: 1; }

  /* ── Locate button (inline) ──────────────────────────────────────── */
  .locate-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-tertiary);
    padding: 2px;
    border-radius: var(--radius-sm);
    transition: color 150ms, background 150ms;
    flex-shrink: 0;
  }
  .locate-btn:hover { color: var(--primary-500); background: var(--surface-inset); }

  /* ── Duration pills ─────────────────────────────────────────────── */
  .live-link-toolbar {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-2);
    width: 100%;
  }
  .live-link-toolbar .btn { width: 100%; }
  .duration-pills {
    display: flex;
    gap: var(--space-1);
    flex-wrap: wrap;
  }
  .pill-btn {
    padding: var(--space-1) var(--space-3);
    font-size: var(--text-xs);
    font-weight: 600;
    border: 1px solid var(--border-default);
    border-radius: var(--radius-full);
    background: var(--surface-1);
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 120ms var(--ease-out), color 120ms var(--ease-out), transform 120ms var(--ease-spring), box-shadow 150ms var(--ease-out);
  }
  .pill-btn:hover { background: var(--surface-2); color: var(--text-primary); }
  .pill-btn.pill-active {
    background: var(--primary-600);
    color: white;
    border-color: var(--primary-500);
    box-shadow: var(--glow-primary), var(--shadow-xs);
    transform: scale(1.05);
    animation: pill-select 300ms var(--ease-spring);
  }

  /* ── Broadcast card — cyan glow ─────────────────────────────────── */
  .broadcast-card {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius-lg);
    background: rgba(6, 182, 212, 0.05);
    border: 1px solid rgba(6, 182, 212, 0.20);
    border-top-color: rgba(6, 182, 212, 0.38);
    box-shadow: var(--glow-link);
  }
  .broadcast-header {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
  }
  .rec-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--color-rec);
    flex-shrink: 0;
  }
  .broadcast-label {
    font-weight: 700;
    font-size: var(--text-sm);
    color: var(--accent-link);
  }
  .broadcast-expiry {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    margin-left: auto;
  }

  .live-link-item {
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5);
  }
  .live-link-actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
  }

  /* ── Admin controls ─────────────────────────────────────────────── */
  .onboarding { padding: var(--space-4); text-align: center; color: var(--text-tertiary); font-size: var(--text-sm); }
  .pending-admin-section {
    width: 100%;
    margin-top: var(--space-2);
    padding: var(--space-2) 0 0;
    border-top: 1px dashed var(--border-default);
  }
  .pending-admin-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    padding: var(--space-1) 0;
  }
  .pending-admin-info { display: flex; flex-direction: column; gap: 1px; }
  .pending-admin-actions { display: flex; gap: var(--space-1); flex-shrink: 0; }
  .vote-count { color: var(--text-secondary); }
  .badge-danger { background: var(--danger-500); color: #fff; }
  .btn-xs {
    font-size: var(--text-2xs);
    padding: 2px 8px;
    border-radius: var(--radius-sm);
  }
  .duration-select {
    font-size: var(--text-xs);
    padding: 2px 6px;
    border: 1px solid var(--border-default);
    border-radius: var(--radius-sm);
    background: var(--surface-1);
    color: var(--text-primary);
    cursor: pointer;
    max-width: 90px;
  }
</style>
