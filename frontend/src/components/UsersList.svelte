<script>
  import { run } from 'svelte/legacy';

  import { createEventDispatcher, onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { otherUsers, myLocation, focusUser, mapTappedUser } from '../lib/stores/map.js';
  import { authUser } from '../lib/stores/auth.js';
  import { socket } from '../lib/socket.js';
  import { banner } from '../lib/stores/sos.js';
  import { calculateDistance } from '../lib/tracking.js';
  import { haptics } from '../lib/haptics.js';
  import { setMobileTab, setSheetOpen } from '../lib/stores/uiShell.js';
  import VirtualList from './primitives/VirtualList.svelte';
  import RecentChatsStrip from './usersList/RecentChatsStrip.svelte';
  import PeopleSectionHeader from './usersList/PeopleSectionHeader.svelte';
  import SelfRow from './usersList/SelfRow.svelte';
  import UserRow from './usersList/UserRow.svelte';
  import UsersListSkeleton from './usersList/UsersListSkeleton.svelte';
  import PeopleEmptyState from './usersList/PeopleEmptyState.svelte';
  import QuickActionsSheet from './usersList/QuickActionsSheet.svelte';
  import DeleteConfirmSheet from './usersList/DeleteConfirmSheet.svelte';

  /**
   * @typedef {Object} Props
   * @property {boolean} [embedded]
   */

  /** @type {Props} */
  let { embedded = false } = $props();

  const dispatch = createEventDispatcher();

  let isAdmin = $derived($authUser && $authUser.role === 'admin');

  // Sort: SOS first, then online (with location) → online (no location) → offline
  // Pre-compute distances once per reactive run to avoid O(n log n) calculateDistance calls in comparator
  let userList = $derived((() => {
    const users = Array.from($otherUsers.values());
    const distCache = new Map();
    if ($myLocation?.latitude != null) {
      for (const u of users) {
        if (u.latitude != null && u.longitude != null) {
          distCache.set(u.socketId, calculateDistance(
            $myLocation.latitude, $myLocation.longitude, u.latitude, u.longitude
          ));
        }
      }
    }
    return users.sort((a, b) => {
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
          return (distCache.get(a.socketId) ?? Infinity) - (distCache.get(b.socketId) ?? Infinity);
        }
      }
      return 0;
    });
  })());

  // Initial-sync skeleton — show placeholder rows instead of a blank list
  // while the first people payload is still arriving over the socket.
  let initialSyncDone = $state(false);
  onMount(() => {
    const t = setTimeout(() => { initialSyncDone = true; }, 1400);
    return () => clearTimeout(t);
  });
  run(() => {
    if (userList.length > 0 || $myLocation) initialSyncDone = true;
  });

  // initialLoad flag — true only during the FIRST render of the list.
  // Passed to UserRow to gate the stagger entrance animation.
  // VirtualList scroll recycling and later reactive updates see false,
  // so recycled rows never re-stagger. The $effect commits listEverShown
  // on the first reactive run where the list is non-empty; subsequent
  // re-renders (re-sorts, new members, scroll recycling) see false.
  let listEverShown = $state(false);
  let rowInitialLoad = $derived(!listEverShown && userList.length > 0);
  $effect(() => {
    if (userList.length > 0 && !listEverShown) {
      listEverShown = true;
    }
  });

  function locateUser(socketId) {
    haptics.tap();
    focusUser.set(socketId);
  }

  // Invite / add-people — routes to the existing Connect (sharing) flow.
  // Reuses the uiShell store actions so the CTA works without prop-drilling.
  function handleAddPeople() {
    setMobileTab('share');
    setSheetOpen(true);
    dispatch('addPeople');
  }

  // Admin delete — uses a non-blocking confirmation sheet
  let deletingUser = $state(null);
  let deleteConfirmUser = $state(null);

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

  // ── Quick actions sheet ──────────────────────────────────────────────────
  let quickUser = $state(null); // user shown in action sheet

  // Feature 2: When map marker is tapped on mobile, open quick-action sheet
  run(() => {
    if ($mapTappedUser) {
      quickUser = $mapTappedUser;
      mapTappedUser.set(null);
    }
  });

  // ── Pull-to-refresh ──────────────────────────────────────────────────────
  let pullStartY = 0;
  let pullDelta = $state(0);
  let pullRefreshing = $state(false);
  let pullBodyEl = $state();

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
</script>

<div class="panel-shell panel-right panel-base" class:embedded-view={embedded} transition:fly={{ x: 400, duration: 250, easing: cubicOut }}>
  {#if !embedded}
    <div class="panel-header">
      <h3>People</h3>
      <button class="btn btn-icon btn-ghost panel-close-btn" aria-label="Close" onclick={() => dispatch('close')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
  {/if}

  <div
    class="panel-body panel-list-body users-list-body"
    bind:this={pullBodyEl}
    ontouchstart={onPullTouchStart}
    ontouchmove={onPullTouchMove}
    ontouchend={onPullTouchEnd}
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
    <RecentChatsStrip {userList} on:secretChat />

    <!-- Section label -->
    <PeopleSectionHeader count={($myLocation ? 1 : 0) + userList.length} />

    <!-- Self entry — always shown when tracking -->
    <SelfRow on:locate={(e) => locateUser(e.detail)} />

    <!-- Other users -->
    {#if userList.length === 0 && !$myLocation && !initialSyncDone}
      <UsersListSkeleton />
    {:else if userList.length === 0 && !$myLocation}
      <PeopleEmptyState on:addPeople={handleAddPeople} />
    {:else if userList.length === 0}
      <PeopleEmptyState solo on:addPeople={handleAddPeople} />
    {:else}
      <div class="vlist-region">
        <VirtualList items={userList} itemHeight={88}  >
          {#snippet children({ item: user, index })}
                            <UserRow
              {user}
              {index}
              {isAdmin}
              {deletingUser}
              initialLoad={rowInitialLoad}
              on:locate={(e) => locateUser(e.detail)}
              on:quickActions={(e) => quickUser = e.detail}
              on:secretChat
              on:delete={(e) => deleteUser(e.detail)}
            />
                                    {/snippet}
                        </VirtualList>
      </div>
    {/if}
  </div>

  <!-- ── Long-press quick actions sheet ─────────────────────────────────── -->
  <QuickActionsSheet
    user={quickUser}
    on:close={() => quickUser = null}
    on:locate={(e) => { locateUser(e.detail); quickUser = null; }}
    on:secretChat={(e) => { dispatch('secretChat', e.detail); quickUser = null; }}
  />

  <!-- ── Admin delete confirmation dialog (non-blocking, accessible) ─────── -->
  <DeleteConfirmSheet
    user={deleteConfirmUser}
    on:cancel={cancelDelete}
    on:confirm={confirmDelete}
  />
</div>

<style>
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
    border: 2.5px solid color-mix(in srgb, var(--primary-400) 25%, transparent);
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

  @media (prefers-reduced-motion: reduce) {
    .pull-spinner { animation: none; }
  }
</style>
