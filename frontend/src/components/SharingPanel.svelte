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
  import { getShareOrigin } from '../lib/env.js';
  import { getUserColor } from '../lib/getUserColor.js';
  import CopyButton from './primitives/CopyButton.svelte';
  import ShareMyRide from './ShareMyRide.svelte';
  import InviteSheet from './InviteSheet.svelte';
  import { rideShare } from '../lib/stores/rideShare.js';

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
  let busyAction = null;
  function leaveRoom(code) {
    if (busyAction) return;
    busyAction = 'leave-' + code;
    socket.emit('leaveRoom', { code });
    setTimeout(() => { busyAction = null; }, 5000);
  }
  function removeContact(userId) {
    if (busyAction) return;
    busyAction = 'remove-' + userId;
    socket.emit('removeContact', { userId });
    setTimeout(() => { busyAction = null; }, 5000);
  }
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
        await Share.share({ title: 'See where I am right now', text: 'Follow my live location on Kinnect', url, dialogTitle: 'Share live link' });
        return;
      } catch (_) {
        // User dismissed share sheet or plugin failed — fall through to clipboard
      }
    }
    navigator.clipboard.writeText(url).catch(() => {});
    banner.set({ type: 'info', text: 'Link copied — share it with anyone.', actions: [] });
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

  // ── Quick Actions (Share My Ride + On My Way + Invite Family) ──
  let rideShareOpen = false;
  let inviteOpen = false;

  // ── My Code share ──────────────────────────────────────────────
  let myCodeCopied = false;

  function copyMyCode() {
    const code = $myShareCode || $authUser?.shareCode;
    if (!code) return;
    navigator.clipboard.writeText(code).then(() => {
      myCodeCopied = true;
      setTimeout(() => { myCodeCopied = false; }, 2500);
    }).catch(() => {});
  }

  function shareMyCodeViaWA() {
    const code = $myShareCode || $authUser?.shareCode;
    if (!code) return;
    const name = $authUser?.displayName?.split(' ')[0] ?? 'Me';
    const url = getShareOrigin();
    const text =
      `Hey! Add me on Kinnect 📍\n` +
      `Enter my code *${code}* in the app to see my live location.\n\n` +
      `Download here: ${url}`;
    window.open('https://wa.me/?text=' + encodeURIComponent(text), '_blank', 'noopener');
  }

  function onMyWay() {
    const links = $myLiveLinks;
    let waText = "I'm on my way!";
    if (links && links.length > 0) {
      const liveUrl = getShareOrigin() + '/#/live/' + links[0].token;
      waText += ` Track me live: ${liveUrl}`;
    }
    window.open('https://wa.me/?text=' + encodeURIComponent(waText), '_blank', 'noopener');
    socket.emit('onMyWay', {});
  }
</script>

{#if embedded}
  <div class="panel-body sharing-root">

    <!-- ── QUICK ACTIONS ───────────────────────────────────────── -->
    <div class="quick-actions-row">
      <button class="quick-action-card" class:quick-action-active={$rideShare.active} on:click={() => rideShareOpen = true}>
        <div class="qa-icon qa-icon-ride">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="1" y="3" width="15" height="13"/><polygon points="16 8 20 8 23 11 23 16 16 16 16 8"/><circle cx="5.5" cy="18.5" r="2.5"/><circle cx="18.5" cy="18.5" r="2.5"/></svg>
        </div>
        <span class="qa-label">{$rideShare.active ? 'Ride Active' : 'Share Ride'}</span>
        {#if $rideShare.active}
          <span class="qa-live-dot" aria-hidden="true"></span>
        {/if}
      </button>
      <button class="quick-action-card" on:click={onMyWay}>
        <div class="qa-icon qa-icon-omw">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
        </div>
        <span class="qa-label">On My Way</span>
      </button>
      <button class="quick-action-card qa-invite" on:click={() => inviteOpen = true}>
        <div class="qa-icon qa-icon-invite">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/><path d="M12 0C5.373 0 0 5.373 0 12c0 2.016.5 3.914 1.37 5.582L0 24l6.618-1.342A11.954 11.954 0 0 0 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0zm0 21.818a9.818 9.818 0 0 1-5.013-1.375l-.36-.213-3.727.757.788-3.613-.234-.372A9.818 9.818 0 0 1 2.182 12C2.182 6.566 6.566 2.182 12 2.182c5.433 0 9.818 4.384 9.818 9.818 0 5.433-4.385 9.818-9.818 9.818z"/></svg>
        </div>
        <span class="qa-label">Invite Family</span>
      </button>
    </div>
    <ShareMyRide bind:open={rideShareOpen} />
    <InviteSheet bind:open={inviteOpen} />

    <!-- ── ROOMS ─────────────────────────────────────────────────── -->
    <div class="sharing-section">
      <div class="sharing-section-header">
        <span class="section-header-label">Rooms</span>
        {#if $myRooms.length > 0}
          <span class="section-badge">{$myRooms.length}</span>
        {/if}
      </div>
      <div class="rooms-create-row">
        <input class="input" bind:value={roomName} placeholder="Group name (e.g. Family)" />
        <button class="btn btn-primary btn-sm" on:click={createRoom} disabled={loading.createRoom}>{loading.createRoom ? '…' : 'Create'}</button>
      </div>
      <div class="rooms-join-row">
        <input class="input" bind:value={joinCode} placeholder="Paste a group code" on:keydown={e => e.key === 'Enter' && joinRoom()} />
        <button class="btn btn-secondary btn-sm" on:click={joinRoom} disabled={loading.joinRoom}>{loading.joinRoom ? '…' : 'Join'}</button>
      </div>

      {#if $myRooms.length === 0}
        <div class="section-empty">
          <div class="section-empty-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          </div>
          <span class="section-empty-text">No groups yet — create one and share the code with your family</span>
        </div>
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
                <button class="btn btn-danger btn-sm" on:click={() => leaveRoom(room.code)} disabled={busyAction === 'leave-' + room.code}>{busyAction === 'leave-' + room.code ? 'Leaving...' : 'Leave Group'}</button>
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
        <span class="section-header-label">Contacts</span>
        {#if $myContacts.length > 0}
          <span class="section-badge">{$myContacts.length}</span>
        {/if}
      </div>

      <!-- My Code card -->
      {#if $myShareCode || $authUser?.shareCode}
        {@const code = $myShareCode || $authUser?.shareCode}
        <div class="my-code-card">
          <div class="my-code-left">
            <span class="my-code-label">My Code</span>
            <span class="my-code-value">{code}</span>
          </div>
          <div class="my-code-actions">
            <button class="my-code-btn my-code-btn--copy" on:click={copyMyCode} aria-label="Copy my code">
              {#if myCodeCopied}
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="#4ade80" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
                <span style="color:#4ade80">Copied</span>
              {:else}
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                <span>Copy</span>
              {/if}
            </button>
            <button class="my-code-btn my-code-btn--wa" on:click={shareMyCodeViaWA} aria-label="Share my code on WhatsApp">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/><path d="M12 0C5.373 0 0 5.373 0 12c0 2.016.5 3.914 1.37 5.582L0 24l6.618-1.342A11.954 11.954 0 0 0 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0zm0 21.818a9.818 9.818 0 0 1-5.013-1.375l-.36-.213-3.727.757.788-3.613-.234-.372A9.818 9.818 0 0 1 2.182 12C2.182 6.566 6.566 2.182 12 2.182c5.433 0 9.818 4.384 9.818 9.818 0 5.433-4.385 9.818-9.818 9.818z"/></svg>
              <span>WhatsApp</span>
            </button>
          </div>
        </div>
      {/if}

      <div class="rooms-create-row">
        <input class="input" bind:value={contactCode} placeholder="Paste their family code" on:keydown={e => e.key === 'Enter' && addContact()} />
        <button class="btn btn-primary btn-sm" on:click={addContact} disabled={loading.addContact}>{loading.addContact ? '…' : 'Add'}</button>
      </div>
      {#if $myContacts.length === 0}
        <div class="section-empty">
          <div class="section-empty-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
          </div>
          <span class="section-empty-text">No contacts yet — share your code with family to get started</span>
        </div>
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
                  {#if isGuardianOf(c.userId)}<span class="badge badge-primary badge-xs" style="margin-left:4px">You watch</span>{/if}
                  {#if isWardOf(c.userId)}<span class="badge badge-primary badge-xs" style="margin-left:4px">Watches you</span>{/if}
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
                <button class="btn btn-danger btn-sm" on:click={() => removeContact(c.userId)} disabled={busyAction === 'remove-' + c.userId}>{busyAction === 'remove-' + c.userId ? 'Removing…' : 'Remove'}</button>
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
        <span class="section-header-label">Live Sharing</span>
        {#if $myLiveLinks.length > 0}
          <span class="section-badge section-badge-live">{$myLiveLinks.length}</span>
        {/if}
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
        <button class="btn btn-primary btn-sm" on:click={generateLiveLink}>Share Live Location</button>
      </div>
      {#if $myLiveLinks.length === 0}
        <div class="section-empty section-empty--with-cta">
          <div class="section-empty-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M1 6l10.5 6L22 6"/><rect x="1" y="4" width="21" height="16" rx="2"/></svg>
          </div>
          <div class="section-empty-body">
            <span class="section-empty-text">No live links active</span>
            <span class="section-empty-cta-hint">Pick a duration above, then tap Share Live Location</span>
          </div>
        </div>
      {:else}
        <div class="broadcasts-list">
          {#each $myLiveLinks as link, i}
            {@const url = getShareOrigin() + '/#/live/' + link.token}
            {@const waText = encodeURIComponent('Watch my live location on Kinnect: ' + url)}
            <div class="broadcast-card card-glow-link animate-slide-up stagger-item" style="animation-delay:{i*40}ms">
              <div class="broadcast-header">
                <span class="rec-dot animate-rec-blink" aria-hidden="true"></span>
                <span class="broadcast-label">Live</span>
                <span class="broadcast-expiry">{link.expiresAt ? 'Expires ' + new Date(link.expiresAt).toLocaleTimeString() : 'No expiry'}</span>
              </div>
              <div class="live-link-actions">
                <CopyButton text={url} label="Copy Link" />
                <a
                  class="btn-wa"
                  href="https://wa.me/?text={waText}"
                  target="_blank"
                  rel="noopener noreferrer"
                  aria-label="Share via WhatsApp"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                    <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 0 1-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 0 1-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 0 1 2.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0 0 12.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 0 0 5.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 0 0-3.48-8.413Z"/>
                  </svg>
                  WhatsApp
                </a>
                <button class="btn btn-danger btn-sm" on:click={() => revokeLink(link.token)}>Stop Sharing</button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    {#if !hasAny}
      <div class="empty-state-hero">
        <p>Share your family code to connect with your people.</p>
      </div>
    {/if}
  </div>
{:else}
  <div class="panel-shell panel-left panel-base">
    <div class="panel-header">
      <h3>Connect</h3>
      <button class="btn btn-icon btn-ghost" aria-label="Close connect panel" on:click={() => dispatch('close')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
    <div class="panel-body">
      <p class="mini">Manage sharing from the sidebar.</p>
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
    gap: var(--space-2);
    margin-bottom: var(--space-1);
  }

  /* Section label */
  .section-header-label {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    flex: 1;
  }

  /* Section badge */
  .section-badge {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-full);
    padding: 1px 8px;
    min-width: 22px;
    text-align: center;
  }
  .section-badge-live {
    color: var(--accent-link);
    background: rgba(6, 182, 212, 0.08);
    border-color: rgba(6, 182, 212, 0.20);
  }

  /* Section empty state */
  .section-empty {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-4);
    background: var(--surface-inset);
    border: 1px dashed var(--border-subtle);
    border-radius: var(--radius-xl);
  }
  .section-empty-icon {
    width: 40px;
    height: 40px;
    border-radius: var(--radius-lg);
    background: linear-gradient(135deg, rgba(99,102,241,0.12) 0%, rgba(139,92,246,0.07) 100%);
    border: 1px solid rgba(99, 102, 241, 0.16);
    color: var(--primary-400);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .section-empty-text {
    font-size: var(--text-sm);
    color: var(--text-tertiary);
    line-height: var(--leading-normal);
  }

  /* ── Person cards (Your People) ──────────────────────────────── */
  .person-card {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2-5) var(--space-1);
    border-radius: var(--radius-lg);
    transition: background 180ms var(--ease-out);
  }
  .person-card:hover {
    background: var(--surface-hover);
  }
  .person-ring-wrap {
    flex-shrink: 0;
    position: relative;
  }
  .person-avatar-circle {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: white;
    background: var(--person-color, var(--primary-500));
    position: relative;
    transition: box-shadow 300ms var(--ease-out);
  }
  .person-avatar-circle.state-now {
    box-shadow:
      0 0 0 2.5px var(--surface-0),
      0 0 0 4.5px var(--person-color, var(--primary-500)),
      0 0 14px var(--person-color, var(--primary-500));
  }
  .person-avatar-circle.state-recent {
    box-shadow:
      0 0 0 2.5px var(--surface-0),
      0 0 0 4px rgba(99,102,241,0.45);
  }
  .person-avatar-circle.state-sos {
    box-shadow:
      0 0 0 2.5px var(--surface-0),
      0 0 0 4.5px #ef4444,
      0 0 18px rgba(239,68,68,0.55);
    animation: sos-urgent-pulse 0.9s ease-in-out infinite;
  }
  .person-avatar-circle.state-gone {
    box-shadow: none;
    filter: grayscale(0.5);
    opacity: 0.7;
  }
  .person-name-block {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .person-name {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .person-status {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .person-status.status-now { color: var(--success-500); font-weight: 600; }
  .person-status.status-sos { color: var(--danger-500); font-weight: 700; }
  .activity-chip {
    font-size: var(--text-2xs);
    color: var(--text-tertiary);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-full);
    padding: 1px 7px;
    width: fit-content;
    margin-top: 2px;
  }

  /* ── Room cards ───────────────────────────────────────────────── */
  .room-card {
    background: var(--surface-1);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-xl);
    padding: var(--space-3);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    transition: background 150ms var(--ease-out), box-shadow 150ms var(--ease-out);
  }
  .room-card:hover {
    background: var(--surface-2);
    box-shadow: var(--shadow-sm);
  }
  .room-card-header {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
  }
  .room-card-icon {
    width: 36px;
    height: 36px;
    border-radius: var(--radius-md);
    background: rgba(99,102,241,0.12);
    color: var(--primary-400);
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-size: var(--text-base);
    font-weight: 800;
    flex-shrink: 0;
  }
  .room-card-meta {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .room-card-name {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    display: flex;
    align-items: center;
    gap: var(--space-1);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .room-card-code {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    letter-spacing: 0.05em;
  }
  .room-card-actions {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    flex-wrap: wrap;
    padding-top: var(--space-1);
  }
  .room-members-row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
    padding-top: var(--space-2);
    border-top: 1px solid var(--border-subtle);
  }
  .room-member-chip {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-full);
    padding: 3px 10px;
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 120ms, color 120ms;
  }
  .room-member-chip:hover {
    background: var(--surface-hover);
    color: var(--text-primary);
  }
  .room-member-avatar {
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: rgba(99,102,241,0.15);
    color: var(--primary-400);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-size: 9px;
    font-weight: 700;
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
    width: 44px;
    height: 44px;
    border-radius: var(--radius-full);
    background: rgba(20, 184, 166, 0.08);
    border: 1px solid rgba(20, 184, 166, 0.20);
    color: var(--primary-400);
    cursor: pointer;
    flex-shrink: 0;
    transition: background var(--duration-fast) var(--ease-out),
                transform var(--duration-fast) var(--ease-spring);
    touch-action: manipulation;
  }
  .locate-pill:hover {
    background: rgba(20, 184, 166, 0.16);
    transform: scale(1.08);
  }
  .locate-pill:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 2px;
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
    font-family: var(--font-display);
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
    font-family: var(--font-display);
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
    align-items: center;
  }

  /* WhatsApp share button */
  .btn-wa {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 5px var(--space-3);
    background: #25D366;
    color: #fff;
    border-radius: var(--radius-full);
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    text-decoration: none;
    cursor: pointer;
    transition: background 150ms var(--ease-out), transform 120ms var(--ease-spring), box-shadow 150ms var(--ease-out);
    flex-shrink: 0;
    white-space: nowrap;
  }
  .btn-wa:hover {
    background: #128C7E;
    transform: scale(1.04);
    box-shadow: 0 2px 12px rgba(37, 211, 102, 0.35);
  }
  .btn-wa:active {
    transform: scale(0.97);
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

  /* ── Quick Actions (Share Ride / On My Way / Invite) ─────────── */
  .quick-actions-row {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--space-2);
    margin-bottom: var(--space-4);
  }

  .quick-action-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1-5);
    padding: var(--space-3-5) var(--space-2);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg, 14px);
    cursor: pointer;
    position: relative;
    transition: background 150ms, border-color 150ms, transform 120ms;
    -webkit-tap-highlight-color: transparent;
  }
  .quick-action-card:hover {
    background: var(--surface-hover);
    border-color: var(--border-default);
  }
  .quick-action-card:active {
    transform: scale(0.96);
  }

  .quick-action-active {
    background: rgba(16, 185, 129, 0.10);
    border-color: rgba(16, 185, 129, 0.35);
  }
  .quick-action-active:hover {
    background: rgba(16, 185, 129, 0.15);
  }

  .qa-icon {
    width: 40px;
    height: 40px;
    border-radius: var(--radius-sm2);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .qa-icon-ride {
    background: linear-gradient(135deg, rgba(99,102,241,0.15), rgba(139,92,246,0.10));
    border: 1px solid rgba(99,102,241,0.20);
    color: var(--primary-400, #818cf8);
  }

  .qa-icon-omw {
    background: linear-gradient(135deg, rgba(20,184,166,0.15), rgba(16,185,129,0.10));
    border: 1px solid rgba(20,184,166,0.20);
    color: var(--primary-500, #14b8a6);
  }

  .qa-icon-invite {
    background: linear-gradient(135deg, rgba(37,211,102,0.14), rgba(37,211,102,0.06));
    border: 1px solid rgba(37,211,102,0.22);
    color: #25d366;
  }

  /* ── My Code card ─────────────────────────────────────────────── */
  .my-code-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: var(--space-2-5);
    padding: var(--space-3) var(--space-3-5);
    margin-bottom: var(--space-2);
    background: var(--surface-inset);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
  }

  .my-code-left {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .my-code-label {
    font-size: var(--text-2xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--text-tertiary);
    font-family: var(--font-display);
  }

  .my-code-value {
    font-size: var(--text-xl);
    font-weight: 800;
    color: var(--text-primary);
    letter-spacing: 0.12em;
    font-family: var(--font-mono);
  }

  .my-code-actions {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    flex-shrink: 0;
  }

  .my-code-btn {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-2) var(--space-3);
    min-height: 44px;
    border-radius: var(--radius-sm2);
    border: 1px solid var(--border-default);
    background: var(--surface-hover);
    color: var(--text-secondary);
    font-size: var(--text-xs);
    font-weight: 600;
    font-family: var(--font-display);
    cursor: pointer;
    white-space: nowrap;
    transition: background var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }
  .my-code-btn:hover { background: var(--surface-active); }
  .my-code-btn:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }

  .my-code-btn--wa {
    border-color: rgba(37, 211, 102, 0.25);
    background: rgba(37, 211, 102, 0.07);
    color: #25d366;
  }
  .my-code-btn--wa:hover { background: rgba(37, 211, 102, 0.14); }

  .qa-label {
    font-family: var(--font-display);
    font-size: var(--text-xs, 11px);
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: 0.01em;
    text-align: center;
    line-height: 1.25;
  }

  .qa-live-dot {
    position: absolute;
    top: 8px;
    right: 8px;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--success-500);
    box-shadow: 0 0 6px rgba(16,185,129,0.50);
    animation: qa-pulse 2s ease-in-out infinite;
  }
  @keyframes qa-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }

  /* Empty state with inline CTA hint */
  .section-empty--with-cta {
    align-items: flex-start;
  }
  .section-empty-body {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .section-empty-cta-hint {
    font-size: var(--text-xs);
    color: var(--primary-500);
    font-weight: 500;
  }
</style>
