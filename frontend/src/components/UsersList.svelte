<script>
  import { createEventDispatcher, onDestroy } from 'svelte';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { otherUsers, myLocation, focusUser, mapTappedUser } from '../lib/stores/map.js';
  import { authUser } from '../lib/stores/auth.js';
  import { socket } from '../lib/socket.js';
  import { banner } from '../lib/stores/sos.js';
  import { secretChats } from '../lib/stores/secretChat.js';
  import { formatTimestamp, escHtml, calculateDistance, formatDistance } from '../lib/tracking.js';
  import { haptics } from '../lib/haptics.js';
  import VirtualList from './primitives/VirtualList.svelte';
  import KinnectNexus from './primitives/KinnectNexus.svelte';
  import { computeActivityStatus, formatActivityAge } from '../lib/activityStatus.js';
  import { arrivalProjections } from '../lib/stores/arrivals.js';

  function formatEta(seconds) {
    if (!seconds || seconds <= 0) return null;
    if (seconds < 60) return '< 1 min';
    const m = Math.round(seconds / 60);
    if (m < 60) return `~${m} min`;
    const h = Math.floor(m / 60);
    const rem = m % 60;
    return rem > 0 ? `~${h}h ${rem}m` : `~${h}h`;
  }

  function locateUser(socketId) {
    haptics.tap();
    focusUser.set(socketId);
  }

  export let embedded = false;

  const dispatch = createEventDispatcher();

  $: isAdmin = $authUser && $authUser.role === 'admin';

  // Sort: SOS first, then online (with location) → online (no location) → offline
  $: userList = Array.from($otherUsers.values())
    .sort((a, b) => {
      if (a.sos?.active && !b.sos?.active) return -1;
      if (!a.sos?.active && b.sos?.active) return 1;
      if (a.online !== false && b.online === false) return -1;
      if (a.online === false && b.online !== false) return 1;
      if (a.online !== false && b.online !== false) {
        const aHasLoc = a.latitude != null && a.longitude != null;
        const bHasLoc = b.latitude != null && b.longitude != null;
        if (aHasLoc && !bHasLoc) return -1;
        if (!aHasLoc && bHasLoc) return 1;
        if ($myLocation && aHasLoc && bHasLoc) {
          const da = calculateDistance($myLocation.latitude, $myLocation.longitude, a.latitude, a.longitude);
          const db = calculateDistance($myLocation.latitude, $myLocation.longitude, b.latitude, b.longitude);
          return da - db;
        }
      }
      return 0;
    });

  function onlineStatus(user) {
    if (user.online === false) {
      if (!user.offlineExpiresAt) return 'Offline · kept forever';
      const ms = user.offlineExpiresAt - Date.now();
      if (ms <= 0) return 'Offline · expiring soon';
      const mins = Math.floor(ms / 60000);
      const h = Math.floor(mins / 60);
      const m = mins % 60;
      return h <= 0 ? `Offline · ${m}m left` : `Offline · ${h}h ${m}m left`;
    }
    return 'Online';
  }

  // Admin delete — uses dispatch to show confirmation in a non-blocking modal
  let deletingUser = null;
  let deleteConfirmUser = null;

  function deleteUser(user) {
    if (!isAdmin || deletingUser) return;
    deleteConfirmUser = user;
  }

  function confirmDelete() {
    if (!deleteConfirmUser) return;
    const user = deleteConfirmUser;
    deleteConfirmUser = null;
    deletingUser = user.socketId;
    socket.emit('adminDeleteUser', { socketId: user.socketId });
    banner.set({ type: 'info', text: `Deleted ${user.displayName}`, actions: [] });
    setTimeout(() => { banner.set({ type: null, text: null, actions: [] }); deletingUser = null; }, 3000);
  }

  function cancelDelete() {
    deleteConfirmUser = null;
  }

  // ── Swipe-right to locate on map ────────────────────────────────────────
  let swipeStartX = 0;
  let swipeStartY = 0;

  function onTouchStart(e) {
    swipeStartX = e.touches[0].clientX;
    swipeStartY = e.touches[0].clientY;
  }

  function onTouchEnd(e, user) {
    const dx = e.changedTouches[0].clientX - swipeStartX;
    const dy = Math.abs(e.changedTouches[0].clientY - swipeStartY);
    if (dx > 60 && dy < 30) {
      locateUser(user.socketId);
    } else if (dx < -60 && dy < 30 && user.userId) {
      // Swipe-left — covert fast path to chat (no visual indicator by design)
      haptics.tap?.();
      dispatch('secretChat', { id: user.userId, name: user.displayName });
    }
  }

  // ── Long-press quick actions ─────────────────────────────────────────────
  let quickUser = null;        // user shown in action sheet
  let lpTimer = null;          // long-press timer
  let lpSuppressClick = false; // prevent click from firing after long-press
  let lpStartX = 0;
  let lpStartY = 0;

  function rowPD(e, user) {
    if (e.button != null && e.button !== 0) return; // ignore right-click
    lpStartX = e.clientX;
    lpStartY = e.clientY;
    lpTimer = setTimeout(() => {
      lpTimer = null;
      quickUser = user;
      lpSuppressClick = true;
      haptics.confirm?.();
    }, 250);
  }

  function rowPM(e) {
    if (!lpTimer) return;
    if (Math.abs(e.clientX - lpStartX) > 10 || Math.abs(e.clientY - lpStartY) > 10) {
      clearTimeout(lpTimer);
      lpTimer = null;
    }
  }

  function rowPU() {
    if (lpTimer) { clearTimeout(lpTimer); lpTimer = null; }
  }

  // Cancel long-press timer on component destroy to prevent firing on unmounted component
  onDestroy(() => {
    if (lpTimer) { clearTimeout(lpTimer); lpTimer = null; }
  });

  function rowClick(user) {
    if (lpSuppressClick) { lpSuppressClick = false; return; }
    if (user.latitude == null || user.longitude == null) {
      // No location — open action sheet for available options (chat accessible via avatar tap)
      quickUser = user;
      return;
    }
    locateUser(user.socketId);
  }

  function closeQuickActions() { quickUser = null; }

  // Feature 2: When map marker is tapped on mobile, open quick-action sheet
  $: if ($mapTappedUser) {
    quickUser = $mapTappedUser;
    mapTappedUser.set(null);
  }

  // ── Pull-to-refresh ──────────────────────────────────────────────────────
  let pullStartY = 0;
  let pullDelta = 0;
  let pullRefreshing = false;
  let pullBodyEl;

  function onPullTouchStart(e) {
    if (pullBodyEl && pullBodyEl.scrollTop > 0) return;
    pullStartY = e.touches[0].clientY;
    pullDelta = 0;
  }

  function onPullTouchMove(e) {
    if (!pullStartY) return;
    if (pullBodyEl && pullBodyEl.scrollTop > 0) { pullStartY = 0; return; }
    const dy = e.touches[0].clientY - pullStartY;
    if (dy > 0) pullDelta = Math.min(dy, 90);
  }

  function onPullTouchEnd() {
    if (pullDelta > 60 && !pullRefreshing) {
      pullRefreshing = true;
      haptics.confirm?.();
      dispatch('refresh');
      setTimeout(() => { pullRefreshing = false; pullDelta = 0; }, 800);
    } else {
      pullDelta = 0;
    }
    pullStartY = 0;
  }

  function qaLocate() {
    if (!quickUser) return;
    locateUser(quickUser.socketId);
    quickUser = null;
  }

  function qaCopy() {
    if (!quickUser?.latitude) return;
    navigator.clipboard?.writeText(`${quickUser.latitude.toFixed(6)}, ${quickUser.longitude.toFixed(6)}`).catch(() => {});
    haptics.success?.();
    quickUser = null;
  }

  function onUserRowKeydown(event, socketId) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      locateUser(socketId);
    }
  }

  // Premium avatar palette — richer, more saturated for social feel
  const AVATAR_PALETTES = [
    { bg: 'linear-gradient(135deg, rgba(99,102,241,0.22) 0%, rgba(79,70,229,0.14) 100%)',   color: '#818cf8', solid: '#6366f1' },  // indigo
    { bg: 'linear-gradient(135deg, rgba(16,185,129,0.22) 0%, rgba(5,150,105,0.14) 100%)',   color: '#34d399', solid: '#10b981' },  // emerald
    { bg: 'linear-gradient(135deg, rgba(245,158,11,0.22) 0%, rgba(217,119,6,0.14) 100%)',   color: '#fbbf24', solid: '#f59e0b' },  // amber
    { bg: 'linear-gradient(135deg, rgba(239,68,68,0.20) 0%, rgba(220,38,38,0.12) 100%)',    color: '#f87171', solid: '#ef4444' },  // red
    { bg: 'linear-gradient(135deg, rgba(139,92,246,0.22) 0%, rgba(124,58,237,0.14) 100%)',  color: '#a78bfa', solid: '#8b5cf6' },  // violet
    { bg: 'linear-gradient(135deg, rgba(6,182,212,0.20) 0%, rgba(8,145,178,0.12) 100%)',    color: '#22d3ee', solid: '#06b6d4' },  // cyan
    { bg: 'linear-gradient(135deg, rgba(251,113,133,0.20) 0%, rgba(225,29,72,0.12) 100%)',  color: '#fb7185', solid: '#e11d48' },  // rose
    { bg: 'linear-gradient(135deg, rgba(52,211,153,0.22) 0%, rgba(16,185,129,0.14) 100%)',  color: '#6ee7b7', solid: '#34d399' },  // teal
  ];

  function getAvatarPalette(name) {
    const code = (name || '?').toUpperCase().charCodeAt(0);
    return AVATAR_PALETTES[code % AVATAR_PALETTES.length];
  }

  function getAvatarStyle(name) {
    const p = getAvatarPalette(name);
    return `background: ${p.bg}; color: ${p.color};`;
  }

  function getPresenceRingStyle(user) {
    if (user.sos?.active) return `box-shadow: 0 0 0 2.5px #ef4444, 0 0 0 5px rgba(239,68,68,0.15), 0 0 16px rgba(239,68,68,0.35);`;
    if (user.online !== false) {
      const p = getAvatarPalette(user.displayName);
      return `box-shadow: 0 0 0 2.5px ${p.solid}, 0 0 0 5px rgba(0,0,0,0.5), 0 0 12px ${p.solid}44;`;
    }
    return `box-shadow: 0 0 0 2px rgba(107,114,128,0.3); opacity: 0.6;`;
  }

  function getAccuracyLabel(acc) {
    if (acc == null) return null;
    if (acc <= 15) return 'High';
    if (acc <= 50) return 'Good';
    return 'Low';
  }

  function getAccuracyClass(acc) {
    if (acc == null) return '';
    if (acc <= 15) return 'acc-high';
    if (acc <= 50) return 'acc-good';
    return 'acc-low';
  }

  // ── Recent chats strip ───────────────────────────────────────────────────
  $: _userById = new Map(
    userList.filter(u => u.userId).map(u => [u.userId, u])
  );
  $: recentChatPeers = (() => {
    const out = [];
    for (const [peerId, chat] of $secretChats) {
      if (!chat.messages || !chat.messages.length) continue;
      const u = _userById.get(peerId);
      if (!u) continue;
      out.push({
        id: peerId,
        name: u.displayName || '?',
        user: u,
        latestAt: chat.messages[0]?.createdAt || 0,
        hasUnread: chat.messages.some(m => !m.seenAt && m.senderId === peerId),
      });
    }
    return out.sort((a, b) => new Date(b.latestAt) - new Date(a.latestAt)).slice(0, 6);
  })();
</script>

<div class="panel-shell panel-right panel-base" class:embedded-view={embedded} transition:fly={{ x: 400, duration: 250, easing: cubicOut }}>
  {#if !embedded}
    <div class="panel-header">
      <h3>People</h3>
      <button class="btn btn-icon btn-ghost panel-close-btn" aria-label="Close" on:click={() => dispatch('close')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
  {/if}

  <div
    class="panel-body panel-list-body users-list-body"
    bind:this={pullBodyEl}
    on:touchstart={onPullTouchStart}
    on:touchmove={onPullTouchMove}
    on:touchend={onPullTouchEnd}
  >
    <!-- Pull-to-refresh indicator -->
    {#if pullDelta > 0 || pullRefreshing}
      <div class="pull-indicator" style="height: {pullRefreshing ? 44 : pullDelta * 0.5}px; opacity: {pullRefreshing ? 1 : pullDelta / 60};" aria-hidden="true">
        {#if pullRefreshing}
          <span class="pull-spinner"></span>
        {:else}
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" style="transform: rotate({(pullDelta / 60) * 180}deg)">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        {/if}
      </div>
    {/if}

    <!-- Recent chats strip — quick-access avatars for ongoing conversations -->
    {#if recentChatPeers.length > 0}
      <div class="recent-chats-strip" aria-label="Recent chats">
        <span class="recent-label">Chats</span>
        <div class="recent-scroll" role="list">
          {#each recentChatPeers as peer (peer.id)}
            <button
              class="recent-avatar-btn"
              on:click={() => dispatch('secretChat', { id: peer.id, name: peer.name })}
              aria-label="Chat with {peer.name}{peer.hasUnread ? ', unread messages' : ''}"
              role="listitem"
            >
              <div class="recent-avatar" style="{getAvatarStyle(peer.name)}">
                {(peer.name || '?')[0].toUpperCase()}
                {#if peer.hasUnread}
                  <span class="recent-unread" aria-hidden="true"></span>
                {/if}
              </div>
              <span class="recent-name">{peer.name.split(' ')[0]}</span>
            </button>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Section label -->
    {#if $myLocation || userList.length > 0}
      <div class="people-section-header">
        <span class="people-section-label">
          <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><circle cx="12" cy="8" r="4"/><path d="M20 21a8 8 0 1 0-16 0"/></svg>
          People
        </span>
        <span class="people-section-count">{($myLocation ? 1 : 0) + userList.length}</span>
      </div>
    {/if}

    <!-- Self entry — always shown when tracking -->
    {#if $myLocation}
      <button
        class="user-item user-item-btn me"
        on:click={() => locateUser('__self__')}
        aria-label="Locate yourself on map"
      >
        <div class="user-avatar self-avatar">
          {($authUser?.displayName || 'Y')[0].toUpperCase()}
          <span class="presence-ring-self" aria-hidden="true"></span>
        </div>
        <div class="user-meta">
          <div class="user-name-row">
            <strong class="user-name">{$authUser?.displayName || 'You'}</strong>
            <span class="you-badge animate-live-badge">
              <span class="you-badge-dot" aria-hidden="true"></span>
              Live
            </span>
          </div>
          <div class="user-sub">
            {#if $myLocation.speed != null && $myLocation.speed > 0.5}
              <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="5 12 12 5 19 12"/></svg>
              <span>{parseFloat($myLocation.speed).toFixed(0)} km/h</span>
              <span class="sep">·</span>
            {/if}
            <span class="sub-live-label">Sharing now</span>
          </div>
        </div>
        <span class="locate-icon" aria-hidden="true">
          <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="10" r="3"/><path d="M12 2a8 8 0 0 0-8 8c0 1.892.402 3.13 1.5 4.5L12 22l6.5-7.5c1.098-1.37 1.5-2.608 1.5-4.5a8 8 0 0 0-8-8z"/></svg>
        </span>
      </button>
    {/if}

    <!-- Other users -->
    {#if userList.length === 0 && !$myLocation}
      <!-- Full empty state — nobody here yet -->
      <div class="empty-state-container">
        <KinnectNexus />
        <p class="empty-title">Your people will appear here</p>
        <p class="empty-desc">Share your code with friends or family so you can see each other on the map</p>
        <button class="empty-cta" on:click={() => dispatch('addPeople')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/></svg>
          Add people
        </button>
      </div>
    {:else if userList.length === 0}
      <!-- Just self is visible — prompt to add people -->
      <div class="empty-state-container empty-state-solo">
        <p class="empty-desc">Invite friends or family to Kinnect — they'll appear here once connected</p>
      </div>
    {:else}
      <div class="vlist-region">
        <VirtualList items={userList} itemHeight={88} let:item={user}>
          <div
            class="user-item user-item-btn stagger-item animate-slide-up"
            class:user-sos={user.sos?.active}
            class:user-offline={user.online === false}
            role="button"
            tabindex="0"
            on:click={() => rowClick(user)}
            on:keydown={(e) => onUserRowKeydown(e, user.socketId)}
            on:touchstart={onTouchStart}
            on:touchend={(e) => onTouchEnd(e, user)}
            on:pointerdown={(e) => rowPD(e, user)}
            on:pointermove={rowPM}
            on:pointerup={rowPU}
            on:pointercancel={rowPU}
          >
            <div class="user-avatar" style="{getAvatarStyle(user.displayName)}{user.online === false ? 'filter: saturate(0.4);' : ''}">
              {(user.displayName || 'U')[0].toUpperCase()}
              <!-- Presence ring overlay — colored ring signals status -->
              <span
                class="presence-ring"
                class:ring-sos={user.sos?.active}
                class:ring-offline={user.online === false}
                class:ring-online={user.online !== false && !user.sos?.active}
                style={user.online !== false && !user.sos?.active ? getPresenceRingStyle(user) : ''}
                aria-hidden="true"
              ></span>
            </div>
            <div class="user-meta">
              <div class="user-name-row">
                <strong class="user-name">{user.displayName || 'User'}</strong>
                {#if user.sos?.active}
                  <span class="sos-badge-pill">
                    <span class="sos-badge-dot" aria-hidden="true"></span>
                    SOS
                  </span>
                {/if}
                {#if user.statusMessage}
                  <span class="status-msg-badge" title="{user.statusMessage}">{user.statusMessage}</span>
                {/if}
                {#if user.rideShare?.active}
                  <span class="ride-badge" title="Sharing ride{user.rideShare.vehicle ? ': ' + user.rideShare.vehicle : ''}">
                    <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><rect x="1" y="3" width="15" height="13"/><polygon points="16 8 20 8 23 11 23 16 16 16 16 8"/><circle cx="5.5" cy="18.5" r="2.5"/><circle cx="18.5" cy="18.5" r="2.5"/></svg>
                    Ride
                  </span>
                {/if}
                {#if user.crowdMode?.active}
                  <span class="crowd-badge" title="Festival mode active">
                    <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
                    Group
                  </span>
                {/if}
                {#if user.quietHoursActive}
                  <span class="quiet-badge" title="Quiet Hours — location approximate">
                    <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
                    Quiet
                  </span>
                {/if}
              </div>
              <div class="user-sub">
                {#if user.online !== false}
                  {#if user.latitude == null || user.longitude == null}
                    <!-- Connected but not sharing location -->
                    <span class="location-off-label">
                      <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="1" y1="1" x2="23" y2="23"/><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z" stroke-dasharray="3 3"/></svg>
                      Location off
                    </span>
                  {:else}
                    {@const actStatus = computeActivityStatus(user)}
                    {#if $myLocation}
                      <span class="distance-label">{formatDistance(calculateDistance($myLocation.latitude, $myLocation.longitude, user.latitude, user.longitude)) || 'Near'}</span>
                    {/if}
                    {#if actStatus && actStatus.label !== 'Offline'}
                      {#if $myLocation}<span class="sep">·</span>{/if}
                      <span class="activity-badge" style:color={actStatus.color} aria-label="{actStatus.label}">
                        <span class="activity-badge-dot" style:background={actStatus.dotColor} aria-hidden="true"></span>
                        {actStatus.label}
                      </span>
                    {/if}
                    {#if user.accuracy != null}
                      <span class="sep">·</span>
                      <span class="acc-dot {getAccuracyClass(user.accuracy)}" aria-hidden="true"></span>
                      <span class="acc-label {getAccuracyClass(user.accuracy)}" aria-label="GPS accuracy: {getAccuracyLabel(user.accuracy)}">{getAccuracyLabel(user.accuracy)}</span>
                    {/if}
                    {#if user.userId && $arrivalProjections.has(user.userId)}
                      {@const proj = $arrivalProjections.get(user.userId)}
                      {#if proj?.etaSeconds && proj?.placeName}
                        <span class="sep">·</span>
                        <span class="eta-chip" title="Heading to {proj.placeName}">
                          <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="5 12 12 5 19 12"/><polyline points="5 19 12 12 19 19"/></svg>
                          {proj.placeName} {formatEta(proj.etaSeconds)}
                        </span>
                      {/if}
                    {/if}
                  {/if}
                {:else}
                  <span class="offline-label">{onlineStatus(user)}</span>
                {/if}
              </div>
              {#if (user.formattedTime || user.lastUpdate) && user.online !== false}
                <div class="user-updated">
                  {user.formattedTime || formatTimestamp(user.lastUpdate)}
                </div>
              {/if}
            </div>
            <div class="user-actions">
              {#if user.batteryPct != null}
                <span
                  class="bat-chip"
                  class:bat-low={user.batteryPct <= 20}
                  class:bat-ok={user.batteryPct > 20 && user.batteryPct <= 50}
                  class:bat-good={user.batteryPct > 50}
                  aria-label="Battery {user.batteryPct}%"
                >
                  <!-- Battery icon -->
                  <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="2" y="7" width="16" height="10" rx="2"/><path d="M22 11v2"/></svg>
                  {user.batteryPct}%
                </span>
              {/if}
              {#if user.latitude != null && user.longitude != null}
                <span class="locate-icon" aria-hidden="true">
                  <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="10" r="3"/><path d="M12 2a8 8 0 0 0-8 8c0 1.892.402 3.13 1.5 4.5L12 22l6.5-7.5c1.098-1.37 1.5-2.608 1.5-4.5a8 8 0 0 0-8-8z"/></svg>
                </span>
              {/if}
              {#if isAdmin}
                <button class="btn btn-danger btn-sm" on:click|stopPropagation={() => deleteUser(user)} disabled={deletingUser === user.socketId}>×</button>
              {/if}
            </div>
          </div>
        </VirtualList>
      </div>
    {/if}
  </div>

  <!-- ── Long-press quick actions sheet ─────────────────────────────────── -->
  {#if quickUser}
    <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
    <div
      class="qa-backdrop"
      on:click={closeQuickActions}
      aria-hidden="true"
      transition:fade={{ duration: 200 }}
    ></div>
    <div
      class="qa-sheet"
      role="dialog"
      aria-label="Quick actions for {quickUser.displayName}"
      aria-modal="true"
      transition:fly={{ y: 180, duration: 300, easing: cubicOut }}
    >
      <div class="qa-handle" aria-hidden="true"></div>

      <!-- User identity -->
      <div class="qa-user-header">
        <div class="qa-avatar" style="{getAvatarStyle(quickUser.displayName)}">
          {(quickUser.displayName || 'U')[0].toUpperCase()}
          {#if quickUser.sos?.active}
            <span class="qa-sos-ring" aria-hidden="true"></span>
          {/if}
        </div>
        <div class="qa-user-info">
          <strong class="qa-user-name">{quickUser.displayName}</strong>
          <span class="qa-user-status" class:qa-status-sos={quickUser.sos?.active} class:qa-status-offline={quickUser.online === false}>
            {quickUser.sos?.active ? 'SOS Active' : quickUser.online === false ? 'Offline' : 'Online'}
          </span>
        </div>
      </div>

      <!-- Actions -->
      <div class="qa-actions" role="group" aria-label="Quick actions">
        <button class="qa-action-btn" on:click={qaLocate} disabled={!quickUser.latitude}>
          <div class="qa-action-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
          </div>
          <span>Locate on Map</span>
        </button>

        {#if quickUser.userId}
          <button class="qa-action-btn" on:click={() => { dispatch('secretChat', { id: quickUser.userId, name: quickUser.displayName }); quickUser = null; }}>
            <div class="qa-action-icon">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
            </div>
            <span>Chat</span>
          </button>
        {/if}

        <button class="qa-action-btn" on:click={qaCopy} disabled={!quickUser.latitude}>
          <div class="qa-action-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
          </div>
          <span>Copy Coordinates</span>
        </button>
      </div>

      <button class="qa-cancel-btn" on:click={closeQuickActions}>Cancel</button>
    </div>
  {/if}

  <!-- ── Admin delete confirmation dialog (non-blocking, accessible) ─────── -->
  {#if deleteConfirmUser}
    <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
    <div class="qa-backdrop" on:click={cancelDelete} aria-hidden="true" transition:fade={{ duration: 150 }}></div>
    <div
      class="qa-sheet delete-confirm-sheet"
      role="alertdialog"
      aria-label="Confirm delete {deleteConfirmUser.displayName}"
      aria-modal="true"
      transition:fly={{ y: 120, duration: 250, easing: cubicOut }}
    >
      <div class="qa-handle" aria-hidden="true"></div>
      <p class="delete-confirm-text">
        Remove <strong>{deleteConfirmUser.displayName}</strong> from this session? This will disconnect them.
      </p>
      <div class="delete-confirm-actions">
        <button class="qa-cancel-btn" on:click={cancelDelete}>Cancel</button>
        <button class="delete-confirm-btn" on:click={confirmDelete}>Remove</button>
      </div>
    </div>
  {/if}
</div>

<style>
  /* ── Recent chats strip ─────────────────────────────────────────────────── */
  .recent-chats-strip {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2) var(--space-4) var(--space-2);
    border-bottom: 1px solid var(--border-subtle);
  }

  .recent-label {
    font-family: var(--font-display);
    font-size: var(--text-2xs);
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.07em;
    flex-shrink: 0;
  }

  .recent-scroll {
    display: flex;
    gap: var(--space-3);
    overflow-x: auto;
    overflow-y: visible;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
    flex: 1;
    padding: 4px 0;
  }

  .recent-scroll::-webkit-scrollbar { display: none; }

  .recent-avatar-btn {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;
    min-width: 44px;
    -webkit-tap-highlight-color: transparent;
    flex-shrink: 0;
  }

  .recent-avatar-btn:active { transform: scale(0.88); transition: transform 80ms; }

  .recent-avatar {
    position: relative;
    width: 38px;
    height: 38px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 15px;
    font-weight: 700;
    font-family: var(--font-display);
    border: 2px solid rgba(255,255,255,0.08);
    transition: border-color 120ms;
  }

  :global([data-theme="light"]) .recent-avatar {
    border-color: rgba(0,0,0,0.06);
  }

  .recent-avatar-btn:hover .recent-avatar {
    border-color: rgba(20,184,166,0.4);
  }

  .recent-unread {
    position: absolute;
    top: -1px;
    right: -1px;
    width: 9px;
    height: 9px;
    background: var(--danger-500);
    border-radius: 50%;
    border: 2px solid var(--surface-base);
    box-shadow: 0 0 4px rgba(239,68,68,0.5);
  }

  .recent-name {
    font-size: 10px;
    font-weight: 600;
    color: var(--text-secondary);
    max-width: 44px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: center;
    line-height: 1;
  }

  /* ── Pull-to-refresh indicator ─────────────────────────────────────────── */
  .pull-indicator {
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    transition: height 120ms ease, opacity 120ms ease;
    color: var(--primary-400);
    flex-shrink: 0;
  }

  .pull-spinner {
    width: 18px;
    height: 18px;
    border: 2.5px solid rgba(20, 184, 166, 0.25);
    border-top-color: var(--primary-400);
    border-radius: 50%;
    animation: pull-spin 0.7s linear infinite;
  }

  @keyframes pull-spin {
    to { transform: rotate(360deg); }
  }

  /* ── VirtualList layout ────────────────────────────────────────────────── */
  .users-list-body {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 0;
  }

  .vlist-region {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  /* ── Section header ────────────────────────────────────────────────────── */
  .people-section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-3) var(--space-4) var(--space-2);
  }

  .people-section-label {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  .people-section-count {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    background: var(--surface-inset);
    border-radius: var(--radius-full);
    padding: 1px 7px;
    min-width: 20px;
    text-align: center;
  }

  /* ── User item — no borders, use spacing + hover bg ───────────────────── */
  .user-item {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2-5) var(--space-4);
    min-height: 76px;
  }

  .user-item-btn {
    width: 100%;
    background: none;
    border: none;
    border-bottom: none;
    cursor: pointer;
    text-align: left;
    color: inherit;
    font: inherit;
    border-radius: 0;
    transition:
      background var(--duration-fast) var(--ease-out),
      transform var(--duration-fast) var(--ease-spring);
    -webkit-tap-highlight-color: transparent;
    position: relative;
  }

  .user-item-btn:hover {
    background: var(--surface-hover);
  }

  .user-item-btn:active {
    background: var(--surface-active);
    transform: scale(0.96);
    transition-duration: 60ms;
  }

  /* SOS — urgent red left accent + gradient sweep for readability */
  .user-sos {
    background:
      linear-gradient(90deg, rgba(239, 68, 68, 0.14) 0%, rgba(239, 68, 68, 0.06) 40%, transparent 80%),
      rgba(239, 68, 68, 0.10);
    box-shadow: inset 3px 0 0 var(--danger-500);
  }
  .user-sos:hover {
    background:
      linear-gradient(90deg, rgba(239, 68, 68, 0.20) 0%, rgba(239, 68, 68, 0.10) 40%, transparent 80%),
      rgba(239, 68, 68, 0.12);
  }

  /* Offline — dim only the avatar; name/sub stay legible */
  .user-offline .user-avatar {
    opacity: 0.50;
  }

  /* Self row — subtle primary left accent */
  .me {
    background: rgba(99, 102, 241, 0.04);
    box-shadow: inset 3px 0 0 rgba(99, 102, 241, 0.50);
  }
  .me:hover { background: rgba(99, 102, 241, 0.08); }

  /* ── Avatar — 44px social-grade ────────────────────────────────────────── */
  .user-avatar {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 1.0625rem; /* 17px — legible initials */
    flex-shrink: 0;
    text-transform: uppercase;
    line-height: 1;
    position: relative;
    transition: transform var(--duration-normal) var(--ease-spring);
  }

  .user-item-btn:hover .user-avatar {
    transform: scale(1.05);
  }

  /* Self avatar — gradient + strong glow */
  .self-avatar {
    background: linear-gradient(135deg, var(--primary-400) 0%, var(--primary-700) 100%);
    color: white;
    font-size: 1.125rem;
    box-shadow:
      0 0 0 2.5px var(--primary-500),
      0 0 0 5px rgba(0, 0, 0, 0.55),
      0 0 16px rgba(99, 102, 241, 0.40);
  }

  /* Presence ring — wraps avatar, shows relationship status */
  .presence-ring {
    position: absolute;
    inset: -3px;
    border-radius: 50%;
    pointer-events: none;
    transition: box-shadow var(--duration-normal) var(--ease-out);
  }

  /* SOS presence ring — animated danger */
  .presence-ring.ring-sos {
    box-shadow:
      0 0 0 2.5px var(--danger-500),
      0 0 0 5px rgba(0, 0, 0, 0.5),
      0 0 16px rgba(239, 68, 68, 0.45);
    animation: sos-urgent-pulse 1s ease-in-out infinite;
  }

  /* Offline presence ring — muted */
  .presence-ring.ring-offline {
    box-shadow: 0 0 0 2px rgba(107, 114, 128, 0.30);
  }

  /* Online presence ring — gentle scale breathe (no box-shadow conflict with inline style) */
  .presence-ring.ring-online {
    animation: ring-scale-breathe 2.8s ease-in-out infinite;
  }

  @keyframes ring-scale-breathe {
    0%, 100% { transform: scale(1);    opacity: 0.85; }
    50%       { transform: scale(1.07); opacity: 1;    }
  }

  /* Self: pulsing live ring */
  .presence-ring-self {
    position: absolute;
    inset: -3px;
    border-radius: 50%;
    pointer-events: none;
    animation: aurora-pulse 2.8s ease-in-out infinite;
  }

  /* ── Meta ──────────────────────────────────────────────────────────────── */
  .user-meta {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .user-name-row {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    min-width: 0;
    overflow: hidden;
  }

  .user-name {
    font-family: var(--font-display);
    font-size: var(--text-base);    /* 16px — legible primary label */
    font-weight: 600;
    letter-spacing: -0.01em;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--text-primary);
    line-height: 1.25;
  }

  /* Premium "Live" badge for self — pulsing dot */
  .you-badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-family: var(--font-display);
    font-size: 0.6875rem; /* 11px */
    font-weight: 700;
    color: var(--success-500);
    background: rgba(16, 185, 129, 0.12);
    border: 1px solid rgba(16, 185, 129, 0.25);
    border-radius: var(--radius-full);
    padding: 2px 7px 2px 5px;
    line-height: 1.3;
    flex-shrink: 0;
    letter-spacing: 0.02em;
  }

  .you-badge-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--success-500);
    flex-shrink: 0;
    animation: aurora-pulse 2s ease-in-out infinite;
  }

  /* SOS badge — high urgency */
  .sos-badge-pill {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-family: var(--font-display);
    font-size: 0.6875rem;
    font-weight: 800;
    color: white;
    background: linear-gradient(135deg, #ef4444, #b91c1c);
    border-radius: var(--radius-full);
    padding: 2px 8px 2px 5px;
    line-height: 1.3;
    flex-shrink: 0;
    letter-spacing: 0.04em;
    box-shadow: 0 0 8px rgba(239, 68, 68, 0.45);
    animation: sos-urgent-pulse 1s ease-in-out infinite;
  }

  .sos-badge-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.9);
    flex-shrink: 0;
  }

  .status-msg-badge {
    display: inline-block;
    font-size: 0.6875rem;
    color: var(--text-secondary);
    font-style: italic;
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    opacity: 0.85;
    flex-shrink: 1;
  }

  /* Ride share badge */
  .ride-badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 0.625rem;
    font-weight: 700;
    color: var(--primary-400, #818cf8);
    background: rgba(99, 102, 241, 0.12);
    border: 1px solid rgba(99, 102, 241, 0.22);
    border-radius: var(--radius-full);
    padding: 1px 6px 1px 4px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  /* Crowd / Festival mode badge */
  .crowd-badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 0.625rem;
    font-weight: 700;
    color: var(--warning-500, #f59e0b);
    background: rgba(245, 158, 11, 0.12);
    border: 1px solid rgba(245, 158, 11, 0.22);
    border-radius: var(--radius-full);
    padding: 1px 6px 1px 4px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .quiet-badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 0.625rem;
    font-weight: 700;
    color: var(--primary-400, #818cf8);
    background: rgba(99, 102, 241, 0.12);
    border: 1px solid rgba(99, 102, 241, 0.22);
    border-radius: var(--radius-full);
    padding: 1px 6px 1px 4px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  /* Sub-row — distance, speed, GPS quality */
  .user-sub {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: var(--text-xs);
    color: var(--text-secondary);
    flex-wrap: nowrap;
    overflow: hidden;
  }

  .distance-label {
    font-weight: 600;
    color: var(--text-secondary);
  }

  .sub-live-label {
    color: var(--success-500);
    font-weight: 600;
  }

  .sep { color: var(--text-tertiary); flex-shrink: 0; }

  /* Timestamp — tertiary, compact */
  .user-updated {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    letter-spacing: 0.01em;
  }

  .offline-label {
    color: var(--text-tertiary);
    font-style: italic;
    font-size: var(--text-xs);
  }

  .location-off-label {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    opacity: 0.65;
  }

  /* Activity status badge */
  .activity-badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: var(--text-xs, 11px);
    font-weight: 600;
  }

  .activity-badge-dot {
    display: inline-block;
    width: 5px;
    height: 5px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  /* GPS accuracy — dot + label */
  .acc-dot {
    display: inline-block;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .acc-label { font-weight: 500; font-size: var(--text-xs); }
  .acc-high  { color: var(--success-500); }
  .acc-high.acc-dot { background: var(--success-500); }
  .acc-good  { color: var(--warning-500); }
  .acc-good.acc-dot { background: var(--warning-500); }
  .acc-low   { color: var(--danger-400); }
  .acc-low.acc-dot { background: var(--danger-400); }

  /* ETA chip — token-based colors (no hardcoded hex) */
  .eta-chip {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--warning-600, #d97706);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 120px;
  }
  :global([data-theme="dark"]) .eta-chip { color: var(--warning-300, #fcd34d); }

  /* ── Actions column ─────────────────────────────────────────────────────── */
  .user-actions {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    flex-shrink: 0;
  }

  .locate-icon {
    color: var(--text-tertiary);
    transition: color var(--duration-fast) var(--ease-out), transform var(--duration-fast) var(--ease-spring);
    display: flex;
    align-items: center;
  }
  .user-item-btn:hover .locate-icon {
    color: var(--primary-400);
    transform: scale(1.15);
  }

  /* Battery chip — icon + percentage */
  .bat-chip {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-family: var(--font-display);
    font-size: 0.6875rem; /* 11px */
    font-weight: 700;
    padding: 3px 6px;
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    color: var(--text-tertiary);
    letter-spacing: 0.01em;
  }
  .bat-low  {
    color: var(--danger-400);
    background: rgba(239, 68, 68, 0.12);
    box-shadow: 0 0 0 1px rgba(239, 68, 68, 0.20);
  }
  .bat-ok   { color: var(--warning-500); background: rgba(245, 158, 11, 0.12); }
  .bat-good { color: var(--success-500); background: rgba(16, 185, 129, 0.10); }

  /* ── Empty state ─────────────────────────────────────────────────────────── */
  .empty-state-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: var(--space-6) var(--space-6) var(--space-10);
    text-align: center;
    gap: var(--space-2);
  }

  .empty-state-solo {
    padding: var(--space-4) var(--space-4) 0;
    align-items: flex-start;
    text-align: left;
  }

  .empty-title {
    font-family: var(--font-display);
    font-size: var(--text-lg);
    font-weight: 700;
    color: var(--text-primary);
    margin: var(--space-2) 0 0;
    letter-spacing: -0.01em;
  }

  .empty-desc {
    font-size: var(--text-sm);
    color: var(--text-tertiary);
    max-width: 200px;
    line-height: var(--leading-relaxed);
    margin: 0;
  }

  /* CTA button in empty state */
  .empty-cta {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1-5);
    margin-top: var(--space-3);
    padding: var(--space-2-5) var(--space-4);
    min-height: 44px;
    background: linear-gradient(135deg, var(--primary-500, #14b8a6), var(--primary-700, #0f766e));
    color: white;
    border: none;
    border-radius: var(--radius-lg);
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    cursor: pointer;
    box-shadow: 0 4px 14px rgba(20, 184, 166, 0.30);
    transition: transform var(--duration-fast) var(--ease-spring), box-shadow var(--duration-fast);
    -webkit-tap-highlight-color: transparent;
  }
  .empty-cta:hover { transform: translateY(-1px); box-shadow: 0 6px 20px rgba(20, 184, 166, 0.40); }
  .empty-cta:active { transform: scale(0.97); }

  /* ── Embedded panel ────────────────────────────────────────────────────── */
  .embedded-view {
    position: static;
    top: auto;
    right: auto;
    bottom: auto;
    left: auto;
    width: 100%;
    max-width: none;
    border: none;
    box-shadow: none;
    animation: none;
  }
  .embedded-view .panel-body {
    padding-top: 0;
  }

  /* ── Quick actions sheet (long-press) ───────────────────────────────────── */
  .qa-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    backdrop-filter: blur(3px);
    -webkit-backdrop-filter: blur(3px);
    z-index: 5400;
    touch-action: none;
  }

  .qa-sheet {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 5401;
    background: var(--surface-2, rgba(20, 20, 40, 0.98));
    backdrop-filter: blur(32px) saturate(1.8);
    -webkit-backdrop-filter: blur(32px) saturate(1.8);
    border-top: 1px solid var(--border-subtle, rgba(255,255,255,0.08));
    border-radius: 20px 20px 0 0;
    box-shadow: 0 -8px 48px rgba(0,0,0,0.40), 0 -1px 0 rgba(255,255,255,0.06);
    padding: 8px 16px calc(24px + env(safe-area-inset-bottom, 0px));
    will-change: transform;
  }

  .qa-handle {
    width: 40px;
    height: 5px;
    background: var(--gray-400, rgba(255,255,255,0.22));
    border-radius: 999px;
    margin: 4px auto 16px;
  }

  .qa-user-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 4px 16px;
    border-bottom: 1px solid var(--border-subtle, rgba(255,255,255,0.07));
    margin-bottom: 8px;
  }

  .qa-avatar {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 1.125rem;
    flex-shrink: 0;
    position: relative;
  }

  .qa-sos-ring {
    position: absolute;
    inset: -3px;
    border-radius: 50%;
    box-shadow: 0 0 0 2.5px var(--danger-500), 0 0 12px rgba(239,68,68,0.5);
    animation: sos-urgent-pulse 1s ease-in-out infinite;
    pointer-events: none;
  }

  .qa-user-info {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }

  .qa-user-name {
    font-family: var(--font-display);
    font-size: var(--text-base, 16px);
    font-weight: 700;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .qa-user-status {
    font-size: var(--text-xs, 12px);
    color: var(--success-500, #10b981);
    font-weight: 600;
  }

  .qa-status-offline { color: var(--text-tertiary); }
  .qa-status-sos { color: var(--danger-500, #ef4444); }

  .qa-actions {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 8px;
  }

  .qa-action-btn {
    display: flex;
    align-items: center;
    gap: 14px;
    width: 100%;
    padding: 14px 16px;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-primary);
    font-family: var(--font-sans);
    font-size: var(--text-base, 16px);
    font-weight: 500;
    text-align: left;
    border-radius: var(--radius-lg, 12px);
    transition: background var(--duration-fast, 120ms) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }

  .qa-action-btn:hover {
    background: var(--surface-hover, rgba(255,255,255,0.06));
  }

  .qa-action-btn:active {
    background: var(--surface-active, rgba(255,255,255,0.10));
    transform: scale(0.98);
    transition-duration: 60ms;
  }

  .qa-action-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .qa-action-icon {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: rgba(99,102,241,0.14);
    border: 1px solid rgba(99,102,241,0.22);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--primary-400, #818cf8);
    flex-shrink: 0;
  }

  .qa-cancel-btn {
    display: block;
    width: 100%;
    padding: 15px;
    background: var(--surface-inset, rgba(255,255,255,0.04));
    border: 1px solid var(--border-subtle, rgba(255,255,255,0.08));
    border-radius: var(--radius-lg, 12px);
    color: var(--text-secondary);
    font-family: var(--font-sans);
    font-size: var(--text-base, 16px);
    font-weight: 600;
    cursor: pointer;
    text-align: center;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    margin-top: 4px;
  }

  .qa-cancel-btn:hover { background: var(--surface-hover); }
  .qa-cancel-btn:active { transform: scale(0.98); transition-duration: 60ms; }

  /* ── Admin delete confirmation ─────────────────────────────────────────── */
  .delete-confirm-sheet {
    padding-bottom: calc(20px + env(safe-area-inset-bottom, 0px));
  }

  .delete-confirm-text {
    font-size: var(--text-base);
    color: var(--text-primary);
    text-align: center;
    margin: 0 0 var(--space-4);
    line-height: var(--leading-relaxed);
    padding: 0 var(--space-2);
  }

  .delete-confirm-actions {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-3);
  }

  .delete-confirm-btn {
    padding: 14px;
    background: rgba(239, 68, 68, 0.12);
    border: 1px solid rgba(239, 68, 68, 0.28);
    border-radius: var(--radius-lg);
    color: var(--danger-400, #f87171);
    font-family: var(--font-sans);
    font-size: var(--text-base);
    font-weight: 700;
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    min-height: 44px;
  }
  .delete-confirm-btn:hover { background: rgba(239, 68, 68, 0.20); }
  .delete-confirm-btn:active { transform: scale(0.97); transition-duration: 60ms; }
</style>
