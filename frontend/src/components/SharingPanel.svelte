<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { socket } from '../lib/socket.js';
  import { banner, myLiveLinks } from '../lib/stores/sos.js';
  import { myRooms, myShareCode } from '../lib/stores/rooms.js';
  import { myContacts } from '../lib/stores/contacts.js';
  import { myGuardianData } from '../lib/stores/guardians.js';
  import { authUser } from '../lib/stores/auth.js';
  import { otherUsers, focusUser } from '../lib/stores/map.js';
  import { getPresenceState, getPresenceText } from '../lib/tracking.js';
  import { getShareOrigin } from '../lib/env.js';
  import Card from './primitives/Card.svelte';
  import EmptyState from './primitives/EmptyState.svelte';
  import SectionHeader from './primitives/SectionHeader.svelte';
  import MagneticButton from './primitives/MagneticButton.svelte';
  import ShareMyRide from './ShareMyRide.svelte';
  import InviteSheet from './InviteSheet.svelte';
  import { rideShare } from '../lib/stores/rideShare.js';
  import QuickActionCard from './sharing/QuickActionCard.svelte';
  import RoomCard from './sharing/RoomCard.svelte';
  import ContactRow from './sharing/ContactRow.svelte';
  import LiveBroadcastRow from './sharing/LiveBroadcastRow.svelte';

  function locateContact(userId) {
    for (const [sid, u] of $otherUsers) {
      if (u.userId === userId) { focusUser.set(sid); return; }
    }
    focusUser.set(userId);
  }

  /**
   * @typedef {Object} Props
   * @property {boolean} [embedded]
   */

  /** @type {Props} */
  let { embedded = false } = $props();

  const dispatch = createEventDispatcher();

  let roomName = $state('');
  let joinCode = $state('');
  let contactCode = $state('');
  let loading = $state({ createRoom: false, joinRoom: false, addContact: false });
  let selectedLinkDuration = $state('24h');
  let loadingTimers = {};

  function withLoading(key, fn) {
    if (loading[key]) return;
    loading = { ...loading, [key]: true };
    fn();
    if (loadingTimers[key]) clearTimeout(loadingTimers[key]);
    loadingTimers[key] = setTimeout(() => clearLoading(key), 8000);
  }

  function clearLoading(key) {
    if (loadingTimers[key]) { clearTimeout(loadingTimers[key]); delete loadingTimers[key]; }
    if (!loading[key]) return;
    loading = { ...loading, [key]: false };
  }

  function clearRoomLoading() { clearLoading('createRoom'); clearLoading('joinRoom'); }
  function clearContactLoading() { clearLoading('addContact'); }

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

  let busyAction = $state(null);
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

  let guardianDurations = $state({});
  let roomAdminDurations = $state({});

  function requestAdmin(code) {
    socket.emit('requestRoomAdmin', { roomCode: code, expiresIn: roomAdminDurations[code] || null });
  }
  function revokeAdmin(code, uid) { socket.emit('revokeRoomAdmin', { roomCode: code, userId: uid }); }
  function voteRoomAdmin(code, userId, vote) { socket.emit('voteRoomAdmin', { roomCode: code, userId, vote }); }
  function requestGuardian(userId) {
    socket.emit('requestGuardian', { contactUserId: userId, expiresIn: guardianDurations[userId] || null });
  }
  function inviteGuardian(userId) {
    socket.emit('inviteGuardian', { contactUserId: userId, expiresIn: guardianDurations[userId] || null });
  }

  function isPendingGuardianOf(userId) {
    return $myGuardianData.asGuardian?.some(g => g.wardId === userId && g.status === 'pending');
  }
  function isPendingWardOf(userId) {
    return $myGuardianData.asWard?.some(g => g.guardianId === userId && g.status === 'pending');
  }
  function isGuardianOf(userId) { return $myGuardianData.asGuardian?.some(g => g.wardId === userId && g.status === 'active'); }
  function isWardOf(userId) { return $myGuardianData.asWard?.some(g => g.guardianId === userId && g.status === 'active'); }

  function createLiveLink(dur) {
    socket.emit('createLiveLink', { duration: dur === 'forever' ? null : dur });
  }
  function generateLiveLink() { createLiveLink(selectedLinkDuration); }
  function revokeLink(token) { socket.emit('revokeLiveLink', { token }); }

  // Cross-reference contacts with live otherUsers map for presence
  function getContactUser(userId) {
    for (const u of $otherUsers.values()) {
      if (u.userId === userId) return u;
    }
    return null;
  }
  function contactPresenceState(userId) {
    const u = getContactUser(userId);
    return u ? getPresenceState(u) : 'gone';
  }
  function contactPresenceText(userId) {
    const u = getContactUser(userId);
    return u ? getPresenceText(u) : 'Not on map';
  }
  function contactBatteryLow(userId) {
    const u = getContactUser(userId);
    return !!(u && u.batteryPct != null && u.batteryPct <= 20);
  }

  onMount(() => {
    const onRoomCreated = () => clearRoomLoading();
    const onRoomJoined = () => clearRoomLoading();
    const onRoomError = () => clearRoomLoading();
    const onContactAdded = () => clearContactLoading();
    const onContactError = () => clearContactLoading();
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

  // ── Quick Actions ──────────────────────────────────────────────
  let rideShareOpen = $state(false);
  let inviteOpen = $state(false);

  // ── My Code share ──────────────────────────────────────────────
  let myCodeCopied = $state(false);
  const myCode = $derived($myShareCode || $authUser?.shareCode || '');

  function copyMyCode() {
    if (!myCode) return;
    navigator.clipboard.writeText(myCode).then(() => {
      myCodeCopied = true;
      setTimeout(() => { myCodeCopied = false; }, 2500);
    }).catch(() => {});
  }

  function shareMyCodeViaWA() {
    if (!myCode) return;
    const url = getShareOrigin();
    const text =
      `Hey! Add me on Kinnect 📍\n` +
      `Enter my code *${myCode}* in the app to see my live location.\n\n` +
      `Download here: ${url}`;
    window.open('https://wa.me/?text=' + encodeURIComponent(text), '_blank', 'noopener');
  }

  function onMyWay() {
    const links = $myLiveLinks;
    let waText = "I'm on my way!";
    if (links && links.length > 0) {
      waText += ` Track me live: ${getShareOrigin() + '/#/live/' + links[0].token}`;
    }
    window.open('https://wa.me/?text=' + encodeURIComponent(waText), '_blank', 'noopener');
    socket.emit('onMyWay', {});
  }
</script>

{#if embedded}
  <div class="panel-body sharing-root">

    <!-- ── MY CODE HERO ─────────────────────────────────────────── -->
    {#if myCode}
      <Card variant="glass" glow="primary" padding="none">
        <div class="my-code-card bento-col-4">
          <div class="my-code-left">
            <span class="my-code-label">My Code</span>
            <span class="my-code-value">{myCode}</span>
          </div>
          <div class="my-code-actions">
            <button class="my-code-btn my-code-btn--copy tactile" class:my-code-btn--copied={myCodeCopied} onclick={copyMyCode} aria-label={myCodeCopied ? 'Code copied' : 'Copy my code'}>
              {#if myCodeCopied}
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
                <span>Copied</span>
              {:else}
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                <span>Copy</span>
              {/if}
            </button>
            <button class="my-code-btn my-code-btn--wa tactile" onclick={shareMyCodeViaWA} aria-label="Share my code on WhatsApp">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/><path d="M12 0C5.373 0 0 5.373 0 12c0 2.016.5 3.914 1.37 5.582L0 24l6.618-1.342A11.954 11.954 0 0 0 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0zm0 21.818a9.818 9.818 0 0 1-5.013-1.375l-.36-.213-3.727.757.788-3.613-.234-.372A9.818 9.818 0 0 1 2.182 12C2.182 6.566 6.566 2.182 12 2.182c5.433 0 9.818 4.384 9.818 9.818 0 5.433-4.385 9.818-9.818 9.818z"/></svg>
              <span>WhatsApp</span>
            </button>
          </div>
        </div>
      </Card>
    {/if}

    <!-- ── QUICK ACTIONS ────────────────────────────────────────── -->
    <div class="quick-actions-row">
      <QuickActionCard
        label="Share Ride"
        activeLabel="Ride Active"
        tone="ride"
        active={$rideShare.active}
        onclick={() => rideShareOpen = true}
      >
        {#snippet icon()}
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="1" y="3" width="15" height="13"/><polygon points="16 8 20 8 23 11 23 16 16 16 16 8"/><circle cx="5.5" cy="18.5" r="2.5"/><circle cx="18.5" cy="18.5" r="2.5"/></svg>
        {/snippet}
      </QuickActionCard>
      <QuickActionCard label="On My Way" tone="omw" onclick={onMyWay}>
        {#snippet icon()}
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
        {/snippet}
      </QuickActionCard>
      <QuickActionCard label="Invite Family" tone="invite" onclick={() => inviteOpen = true}>
        {#snippet icon()}
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/><path d="M12 0C5.373 0 0 5.373 0 12c0 2.016.5 3.914 1.37 5.582L0 24l6.618-1.342A11.954 11.954 0 0 0 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0zm0 21.818a9.818 9.818 0 0 1-5.013-1.375l-.36-.213-3.727.757.788-3.613-.234-.372A9.818 9.818 0 0 1 2.182 12C2.182 6.566 6.566 2.182 12 2.182c5.433 0 9.818 4.384 9.818 9.818 0 5.433-4.385 9.818-9.818 9.818z"/></svg>
        {/snippet}
      </QuickActionCard>
    </div>
    <ShareMyRide bind:open={rideShareOpen} />
    <InviteSheet bind:open={inviteOpen} />

    <!-- ── ROOMS ────────────────────────────────────────────────── -->
    <section class="sharing-section">
      <SectionHeader title="Rooms" level={3}>
        {#snippet action()}
          {#if $myRooms.length > 0}<span class="section-badge">{$myRooms.length}</span>{/if}
        {/snippet}
      </SectionHeader>
      <div class="input-row">
        <input class="input" bind:value={roomName} placeholder="Group name (e.g. Family)" />
        <MagneticButton strength={4} className="cta-mag">
          <button class="btn btn-primary btn-sm" onclick={createRoom} disabled={loading.createRoom}>{loading.createRoom ? '…' : 'Create'}</button>
        </MagneticButton>
      </div>
      <div class="input-row">
        <input class="input" bind:value={joinCode} placeholder="Paste a group code" onkeydown={e => e.key === 'Enter' && joinRoom()} />
        <button class="btn btn-secondary btn-sm tactile" onclick={joinRoom} disabled={loading.joinRoom}>{loading.joinRoom ? '…' : 'Join'}</button>
      </div>

      {#if $myRooms.length === 0}
        <EmptyState title="No groups yet" body="Create a group above and share the code with your family to see everyone on one map.">
          {#snippet icon()}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          {/snippet}
        </EmptyState>
      {:else}
        <div class="stack">
          {#each $myRooms as room (room.code)}
            <RoomCard
              {room}
              myUserId={$authUser?.userId}
              leaving={busyAction === 'leave-' + room.code}
              bind:adminDuration={roomAdminDurations[room.code]}
              onlocate={locateContact}
              onleave={() => leaveRoom(room.code)}
              onrequestadmin={() => requestAdmin(room.code)}
              onrevokeadmin={(uid) => revokeAdmin(room.code, uid)}
              onvote={(fromUid, vote) => voteRoomAdmin(room.code, fromUid, vote)}
            />
          {/each}
        </div>
      {/if}
    </section>

    <!-- ── CONTACTS ─────────────────────────────────────────────── -->
    <section class="sharing-section">
      <SectionHeader title="Contacts" level={3}>
        {#snippet action()}
          {#if $myContacts.length > 0}<span class="section-badge">{$myContacts.length}</span>{/if}
        {/snippet}
      </SectionHeader>

      <div class="input-row">
        <input class="input" bind:value={contactCode} placeholder="Paste their family code" onkeydown={e => e.key === 'Enter' && addContact()} />
        <MagneticButton strength={4} className="cta-mag">
          <button class="btn btn-primary btn-sm" onclick={addContact} disabled={loading.addContact}>{loading.addContact ? '…' : 'Add'}</button>
        </MagneticButton>
      </div>

      {#if $myContacts.length === 0}
        <EmptyState title="No contacts yet" body="Share your code above with family and friends so you can see each other on the map.">
          {#snippet icon()}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
          {/snippet}
        </EmptyState>
      {:else}
        <div class="stack">
          {#each $myContacts as c (c.userId)}
            <ContactRow
              contact={c}
              presenceState={contactPresenceState(c.userId)}
              presenceText={contactPresenceText(c.userId)}
              activityContext={getContactUser(c.userId)?.activityContext || ''}
              batteryLow={contactBatteryLow(c.userId)}
              isGuardian={isGuardianOf(c.userId)}
              isWard={isWardOf(c.userId)}
              isPending={isPendingGuardianOf(c.userId) || isPendingWardOf(c.userId)}
              removing={busyAction === 'remove-' + c.userId}
              bind:guardianDuration={guardianDurations[c.userId]}
              onlocate={locateContact}
              onremove={removeContact}
              onwatch={requestGuardian}
              onbewatched={inviteGuardian}
            />
          {/each}
        </div>
      {/if}
    </section>

    <!-- ── LIVE BROADCASTS ──────────────────────────────────────── -->
    <section class="sharing-section">
      <SectionHeader title="Live Sharing" level={3}>
        {#snippet action()}
          {#if $myLiveLinks.length > 0}<span class="section-badge section-badge-live">{$myLiveLinks.length}</span>{/if}
        {/snippet}
      </SectionHeader>
      <div class="live-link-toolbar">
        <div class="duration-pills" role="group" aria-label="Broadcast duration">
          {#each [['1h','1h'],['6h','6h'],['24h','24h'],['48h','48h'],['forever','∞']] as [val, label]}
            <button
              class="pill-btn"
              class:pill-active={selectedLinkDuration === val}
              onclick={() => selectedLinkDuration = val}
              aria-pressed={selectedLinkDuration === val}
            >{label}</button>
          {/each}
        </div>
        <button class="btn btn-primary btn-sm tactile" onclick={generateLiveLink}>Share Live Location</button>
      </div>

      {#if $myLiveLinks.length === 0}
        <div class="section-empty">
          <div class="section-empty-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M1 6l10.5 6L22 6"/><rect x="1" y="4" width="21" height="16" rx="2"/></svg>
          </div>
          <div class="section-empty-body">
            <span class="section-empty-text">No live links active</span>
            <span class="section-empty-cta-hint">Pick a duration above, then tap Share Live Location</span>
          </div>
        </div>
      {:else}
        <div class="stack">
          {#each $myLiveLinks as link (link.token)}
            {@const url = getShareOrigin() + '/#/live/' + link.token}
            <LiveBroadcastRow
              {url}
              waHref={'https://wa.me/?text=' + encodeURIComponent('Watch my live location on Kinnect: ' + url)}
              deadline={link.expiresAt ? new Date(link.expiresAt).getTime() : null}
              onstop={() => revokeLink(link.token)}
            />
          {/each}
        </div>
      {/if}
    </section>
  </div>
{:else}
  <div class="panel-shell panel-left panel-base">
    <div class="panel-header">
      <h3>Connect</h3>
      <button class="btn btn-icon btn-ghost" aria-label="Close connect panel" onclick={() => dispatch('close')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
    <div class="panel-body">
      <p class="mini">Manage sharing from the sidebar.</p>
    </div>
  </div>
{/if}

<style>
  /* ── Root + sections ─────────────────────────────────────────── */
  .sharing-root {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }
  .sharing-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .section-badge {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    background: var(--glass-chip-bg, var(--surface-inset));
    border: 1px solid var(--glass-chip-border, var(--border-subtle));
    border-radius: var(--radius-full);
    padding: 1px 8px;
    min-width: 22px;
    text-align: center;
  }
  .section-badge-live {
    color: var(--accent-link);
    background: color-mix(in oklch, var(--accent-link) 8%, transparent);
    border-color: color-mix(in oklch, var(--accent-link) 20%, transparent);
  }

  :global(.cta-mag) { flex-shrink: 0; }
  :global(.cta-mag) .btn { white-space: nowrap; }

  /* Input rows (create/join/add) */
  .input-row {
    display: flex;
    gap: var(--space-2);
  }
  .input-row .input { flex: 1; }

  /* Vertical card stacks */
  .stack {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  /* ── My Code hero ────────────────────────────────────────────── */
  .my-code-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: var(--space-2-5);
    padding: var(--space-3) var(--space-3-5);
    border-radius: inherit;
  }
  .my-code-left {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
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
    letter-spacing: 0.14em;
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
    background: var(--primary-500-12);
    border: 1px solid var(--primary-500-20);
    border-radius: var(--radius-md);
    padding: var(--space-1) var(--space-2-5);
    width: fit-content;
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
    transition: background var(--duration-fast) var(--ease-out), color var(--duration-fast) var(--ease-out), border-color var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }
  .my-code-btn:hover { background: var(--surface-active); }
  .my-code-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .my-code-btn--copied {
    background: var(--primary-500-20);
    border-color: var(--primary-500-30);
    color: var(--success-500);
  }
  .my-code-btn--copied:hover { background: var(--primary-500-20); }
  .my-code-btn--wa {
    border-color: color-mix(in oklch, var(--whatsapp-green) 25%, transparent);
    background: color-mix(in oklch, var(--whatsapp-green) 7%, transparent);
    color: var(--whatsapp-green);
  }
  .my-code-btn--wa:hover { background: color-mix(in oklch, var(--whatsapp-green) 14%, transparent); }

  /* ── Quick actions row ───────────────────────────────────────── */
  .quick-actions-row {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--space-2);
  }

  /* ── Live sharing toolbar ────────────────────────────────────── */
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
    min-height: 44px;
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 600;
    border: 1px solid var(--border-default);
    border-radius: var(--radius-full);
    background: var(--surface-1);
    color: var(--text-secondary);
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out), color var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }
  .pill-btn:hover { background: var(--surface-2); color: var(--text-primary); }
  .pill-btn.pill-active {
    background: var(--primary-600);
    color: var(--text-on-primary);
    border-color: var(--primary-500);
    box-shadow: var(--glow-primary), var(--shadow-xs);
  }
  .pill-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }

  /* ── Live sharing empty state ────────────────────────────────── */
  .section-empty {
    display: flex;
    align-items: flex-start;
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
    background: var(--primary-500-12);
    border: 1px solid var(--primary-500-20);
    color: var(--primary-400);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .section-empty-body {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .section-empty-text {
    font-size: var(--text-sm);
    color: var(--text-tertiary);
    line-height: var(--leading-normal);
  }
  .section-empty-cta-hint {
    font-size: var(--text-xs);
    color: var(--primary-500);
    font-weight: 500;
  }

  @media (max-width: 767px) {
    .sharing-root {
      width: 100%;
      max-width: 100%;
      max-height: 100dvh;
      overflow-y: auto;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .my-code-btn { transition: none; }
    .pill-btn { transition: none; }
  }
</style>
