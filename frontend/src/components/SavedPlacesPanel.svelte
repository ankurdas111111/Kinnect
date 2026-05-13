<script>
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import { socket } from '../lib/socket.js';
  import { banner } from '../lib/stores/sos.js';
  import { savedPlaces, placeAlerts, speedAlerts } from '../lib/stores/places.js';
  import { otherUsers, myLocation } from '../lib/stores/map.js';
  import { authUser } from '../lib/stores/auth.js';
  import { apiGet, apiPost, apiDelete } from '../lib/api.js';

  export let embedded = false;

  const dispatch = createEventDispatcher();

  // ── Saved Places ─────────────────────────────────────────────────────────────
  let newPlaceName = '';
  let newPlaceRadius = 100;
  let newPlaceIcon = 'pin';
  let showAddPlace = false;

  // ── Arrival / Departure Alerts ───────────────────────────────────────────────
  let alertTargetId = '';
  let alertPlaceId = '';
  let alertOnArrive = true;
  let alertOnDepart = true;

  // ── Speed Alerts ─────────────────────────────────────────────────────────────
  let speedTargetId = '';
  let speedThreshold = 80;

  const iconOptions = [
    { value: 'home', label: 'Home' },
    { value: 'work', label: 'Work' },
    { value: 'school', label: 'School' },
    { value: 'gym', label: 'Gym' },
    { value: 'pin', label: 'Other' },
  ];

  const PALERT_KEY  = 'kinnect_place_alerts';
  const SALERT_KEY  = 'kinnect_speed_alerts';

  // Zone Story state — per-place visit history
  let storyPlaceId = null;
  let storyVisits = [];
  let storyLoading = false;

  $: visibleUsers = buildUserList($otherUsers, $authUser);

  function buildUserList(others, auth) {
    const list = [];
    if (auth) list.push({ id: auth.userId, name: 'Me' });
    for (const u of others.values()) {
      if (u.userId) list.push({ id: u.userId, name: u.displayName || u.userId.slice(0, 6) });
    }
    return list;
  }

  function loadAlertsFromStorage() {
    try {
      const pa = localStorage.getItem(PALERT_KEY);
      if (pa) placeAlerts.set(JSON.parse(pa));
      const sa = localStorage.getItem(SALERT_KEY);
      if (sa) speedAlerts.set(JSON.parse(sa));
    } catch (_) {}
  }

  function persist(key, value) {
    try { localStorage.setItem(key, JSON.stringify(value)); } catch (_) {}
  }

  onMount(async () => {
    loadAlertsFromStorage();
    const res = await apiGet('/api/places');
    if (Array.isArray(res)) {
      savedPlaces.set(res.map(p => ({
        id: p.id,
        name: p.name,
        lat: p.latitude,
        lng: p.longitude,
        radiusM: p.radiusM,
        icon: p.icon || 'pin',
      })));
    }
  });

  async function addPlace() {
    if (!newPlaceName.trim()) return;
    const loc = $myLocation;
    if (!loc) {
      banner.set({ type: 'info', text: 'Start location tracking first so we can pin your current position', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
      return;
    }
    const res = await apiPost('/api/places', {
      name: newPlaceName.trim(),
      latitude: loc.latitude,
      longitude: loc.longitude,
      radiusM: newPlaceRadius,
      icon: newPlaceIcon,
    });
    if (res && res.id) {
      savedPlaces.update(arr => [...arr, {
        id: res.id,
        name: res.name,
        lat: res.latitude,
        lng: res.longitude,
        radiusM: res.radiusM,
        icon: res.icon || 'pin',
      }]);
      newPlaceName = '';
      newPlaceRadius = 100;
      newPlaceIcon = 'pin';
      showAddPlace = false;
      banner.set({ type: 'info', text: `"${res.name}" saved at your current location`, actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2500);
    } else {
      banner.set({ type: 'sos', text: res?.error || 'Failed to save place — check your connection and try again', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
    }
  }

  async function removePlace(placeId, placeName) {
    await apiDelete(`/api/places/${placeId}`);
    savedPlaces.update(arr => arr.filter(p => p.id !== placeId));
    placeAlerts.update(arr => {
      const next = arr.filter(a => a.placeId !== placeId);
      persist(PALERT_KEY, next);
      return next;
    });
    if (storyPlaceId === placeId) storyPlaceId = null;
  }

  // ── Zone Story ────────────────────────────────────────────────────────────────
  async function viewStory(placeId) {
    if (storyPlaceId === placeId) { storyPlaceId = null; return; }
    storyPlaceId = placeId;
    storyVisits = [];
    storyLoading = true;
    const res = await apiGet(`/api/places/${placeId}/story`);
    storyLoading = false;
    storyVisits = Array.isArray(res) ? res : [];
  }

  function formatDuration(seconds) {
    if (!seconds) return '';
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    if (h > 0) return `${h}h ${m}m`;
    return `${m}m`;
  }

  function formatTime(iso) {
    if (!iso) return '';
    return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function formatDate(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    const today = new Date();
    if (d.toDateString() === today.toDateString()) return 'Today';
    const yesterday = new Date(today);
    yesterday.setDate(today.getDate() - 1);
    if (d.toDateString() === yesterday.toDateString()) return 'Yesterday';
    return d.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' });
  }

  // ── Arrival / Departure Alerts ───────────────────────────────────────────────
  function addPlaceAlert() {
    if (!alertTargetId || !alertPlaceId) return;
    const place = $savedPlaces.find(p => p.id === alertPlaceId);
    const user  = visibleUsers.find(u => u.id === alertTargetId);
    const alert = {
      id: Date.now().toString(),
      targetId: alertTargetId,
      targetName: user?.name || alertTargetId,
      placeId: alertPlaceId,
      placeName: place?.name || '?',
      onArrive: alertOnArrive,
      onDepart: alertOnDepart,
    };
    placeAlerts.update(arr => {
      const next = [...arr, alert];
      persist(PALERT_KEY, next);
      return next;
    });
    alertTargetId = '';
    alertPlaceId = '';
    banner.set({ type: 'info', text: 'Arrival/departure alert added', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2500);
  }

  function removePlaceAlert(alertId) {
    placeAlerts.update(arr => {
      const next = arr.filter(a => a.id !== alertId);
      persist(PALERT_KEY, next);
      return next;
    });
  }

  function addSpeedAlert() {
    if (!speedTargetId) return;
    const user = visibleUsers.find(u => u.id === speedTargetId);
    const alert = {
      id: Date.now().toString(),
      targetId: speedTargetId,
      targetName: user?.name || speedTargetId,
      thresholdKmh: speedThreshold,
    };
    speedAlerts.update(arr => {
      const next = [...arr, alert];
      persist(SALERT_KEY, next);
      return next;
    });
    speedTargetId = '';
    banner.set({ type: 'info', text: `Speed alert set — you'll be notified above ${speedThreshold} km/h`, actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2500);
  }

  function removeSpeedAlert(alertId) {
    speedAlerts.update(arr => {
      const next = arr.filter(a => a.id !== alertId);
      persist(SALERT_KEY, next);
      return next;
    });
  }

  function getUserName(userId) {
    if ($authUser && userId === $authUser.userId) return 'Me';
    for (const u of $otherUsers.values()) {
      if (u.userId === userId) return u.displayName || userId.slice(0, 6);
    }
    return userId?.slice(0, 6) || '?';
  }

  const iconEmoji = { home: '🏠', work: '💼', school: '🏫', gym: '🏋️', pin: '📍' };
</script>

{#if embedded}
  <div class="panel-body places-panel">

    <!-- ── Saved Places ─────────────────────────────────────────────────── -->
    <div class="section-header">
      <span class="section-label">Saved Places</span>
      {#if $savedPlaces.length > 0}
        <span class="section-badge">{$savedPlaces.length}</span>
      {/if}
    </div>

    <div class="section-content">
      <p class="hint">Save home, work, school. Get notified when family arrives or leaves.</p>

      {#if $savedPlaces.length === 0 && !showAddPlace}
        <!-- Empty state with CTA -->
        <div class="empty-state">
          <div class="empty-icon" aria-hidden="true">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
          </div>
          <p class="empty-title">No saved places yet</p>
          <p class="empty-sub">Add home or work to get arrival and departure alerts.</p>
          <button class="btn btn-primary btn-sm" on:click={() => showAddPlace = true}>
            Add a Place
          </button>
        </div>
      {:else}
        {#each $savedPlaces as place (place.id)}
          <div class="list-item">
            <div class="item-icon" aria-hidden="true">{iconEmoji[place.icon] ?? '📍'}</div>
            <div class="item-info">
              <span class="item-name">{place.name}</span>
              <span class="item-detail">{place.radiusM}m radius</span>
            </div>
            <button
              class="icon-action"
              on:click={() => viewStory(place.id)}
              aria-label="View visit history for {place.name}"
              aria-expanded={storyPlaceId === place.id}
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
            </button>
            <button
              class="icon-action icon-action--danger"
              on:click={() => removePlace(place.id, place.name)}
              aria-label="Remove {place.name}"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>
            </button>
          </div>

          {#if storyPlaceId === place.id}
            <div class="zone-story" role="region" aria-label="Visit history for {place.name}">
              {#if storyLoading}
                <div class="story-skeleton" role="status" aria-label="Loading visit history" aria-busy="true">
                  <div class="skel-row skel-wide"></div>
                  <div class="skel-row skel-mid"></div>
                </div>
              {:else if storyVisits.length === 0}
                <p class="story-empty">No visits in the last 7 days.</p>
              {:else}
                {#each storyVisits as v}
                  <div class="story-row">
                    <div class="story-avatar" aria-hidden="true">{v.displayName?.charAt(0) ?? '?'}</div>
                    <div class="story-info">
                      <span class="story-name">{v.displayName}</span>
                      <span class="story-time">
                        {formatDate(v.arrivedAt)} · {formatTime(v.arrivedAt)}
                        {#if v.departedAt} – {formatTime(v.departedAt)}{/if}
                        {#if v.durationSeconds} · {formatDuration(v.durationSeconds)}{/if}
                      </span>
                    </div>
                    {#if !v.departedAt}
                      <span class="story-badge-here" aria-label="Currently here">Here now</span>
                    {/if}
                  </div>
                {/each}
              {/if}
            </div>
          {/if}
        {/each}
      {/if}

      {#if showAddPlace}
        <div class="add-form" role="form" aria-label="Add a saved place">
          <input
            type="text"
            bind:value={newPlaceName}
            class="field-input field-full"
            placeholder="Name this place (Home, Work, School…)"
            maxlength="100"
            autocomplete="off"
            aria-label="Place name"
          />
          <div class="form-row">
            <label class="sr-only" for="place-icon">Icon</label>
            <select id="place-icon" bind:value={newPlaceIcon} class="field-input field-sm" aria-label="Place icon">
              {#each iconOptions as opt}
                <option value={opt.value}>{iconEmoji[opt.value]} {opt.label}</option>
              {/each}
            </select>
            <label class="field-label-inline">
              Radius
              <input
                type="number"
                bind:value={newPlaceRadius}
                class="field-input field-num"
                min="50"
                max="5000"
                step="50"
                aria-label="Geofence radius in meters"
              />
              <span aria-hidden="true">m</span>
            </label>
          </div>
          <div class="form-actions">
            <button
              class="btn btn-primary btn-sm"
              on:click={addPlace}
              disabled={!newPlaceName.trim()}
            >
              Save This Place
            </button>
            <button
              class="btn btn-secondary btn-sm"
              on:click={() => { showAddPlace = false; newPlaceName = ''; }}
            >
              Cancel
            </button>
          </div>
        </div>
      {:else if $savedPlaces.length > 0}
        <button class="btn btn-secondary btn-sm add-btn" on:click={() => showAddPlace = true}>
          + Add a Place
        </button>
      {/if}
    </div>

    <div class="section-divider" role="separator"></div>

    <!-- ── Arrival Alerts ──────────────────────────────────────────────── -->
    <div class="section-header">
      <span class="section-label">Arrival Alerts</span>
      {#if $placeAlerts.length > 0}
        <span class="section-badge">{$placeAlerts.length}</span>
      {/if}
    </div>
    <div class="section-content">
      <p class="hint">Get notified when someone arrives at or leaves a saved place.</p>

      {#if $placeAlerts.length === 0}
        <div class="inline-empty">
          <p class="inline-empty-text">No alerts yet.</p>
          {#if $savedPlaces.length === 0}
            <p class="inline-empty-cta-hint">Save a place above first to set up alerts.</p>
          {:else}
            <p class="inline-empty-cta-hint">Add an alert below — know when your family arrives home.</p>
          {/if}
        </div>
      {:else}
        {#each $placeAlerts as alert (alert.id)}
          <div class="list-item">
            <div class="item-icon" aria-hidden="true">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>
            </div>
            <div class="item-info">
              <span class="item-name">{getUserName(alert.targetId)} at {alert.placeName || '?'}</span>
              <span class="item-detail">
                {[alert.onArrive && 'Arrive', alert.onDepart && 'Depart'].filter(Boolean).join(' + ')}
              </span>
            </div>
            <button
              class="icon-action icon-action--danger"
              on:click={() => removePlaceAlert(alert.id)}
              aria-label="Remove alert: {getUserName(alert.targetId)} at {alert.placeName}"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>
            </button>
          </div>
        {/each}
      {/if}

      {#if $savedPlaces.length > 0}
        <div class="add-form" role="form" aria-label="Add arrival alert">
          <div class="form-row">
            <label class="sr-only" for="alert-person">Person</label>
            <select id="alert-person" bind:value={alertTargetId} class="field-input field-sm" aria-label="Choose a person">
              <option value="">Choose a person</option>
              {#each visibleUsers as u}
                <option value={u.id}>{u.name}</option>
              {/each}
            </select>
            <label class="sr-only" for="alert-place">Place</label>
            <select id="alert-place" bind:value={alertPlaceId} class="field-input field-sm" aria-label="Choose a place">
              <option value="">Choose a place</option>
              {#each $savedPlaces as p}
                <option value={p.id}>{iconEmoji[p.icon] ?? '📍'} {p.name}</option>
              {/each}
            </select>
          </div>
          <div class="form-row form-row--check">
            <label class="check-label">
              <input type="checkbox" bind:checked={alertOnArrive} />
              When they arrive
            </label>
            <label class="check-label">
              <input type="checkbox" bind:checked={alertOnDepart} />
              When they leave
            </label>
          </div>
          <button
            class="btn btn-primary btn-sm"
            on:click={addPlaceAlert}
            disabled={!alertTargetId || !alertPlaceId || (!alertOnArrive && !alertOnDepart)}
          >
            Add Alert
          </button>
        </div>
      {/if}
    </div>

    <div class="section-divider" role="separator"></div>

    <!-- ── Speed Alerts ─────────────────────────────────────────────────── -->
    <div class="section-header">
      <span class="section-label">Speed Alerts</span>
      {#if $speedAlerts.length > 0}
        <span class="section-badge">{$speedAlerts.length}</span>
      {/if}
    </div>
    <div class="section-content">
      <p class="hint">Get notified when someone drives faster than a set limit.</p>

      {#if $speedAlerts.length === 0}
        <div class="inline-empty">
          <p class="inline-empty-text">No speed alerts yet.</p>
          <p class="inline-empty-cta-hint">Set a limit below to protect family members driving.</p>
        </div>
      {:else}
        {#each $speedAlerts as sa (sa.id)}
          <div class="list-item">
            <div class="item-icon" aria-hidden="true">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><circle cx="12" cy="16" r=".5" fill="currentColor"/></svg>
            </div>
            <div class="item-info">
              <span class="item-name">{getUserName(sa.targetId)}</span>
              <span class="item-detail">Alert above {sa.thresholdKmh} km/h</span>
            </div>
            <button
              class="icon-action icon-action--danger"
              on:click={() => removeSpeedAlert(sa.id)}
              aria-label="Remove speed alert for {getUserName(sa.targetId)}"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>
            </button>
          </div>
        {/each}
      {/if}

      <div class="add-form" role="form" aria-label="Add speed alert">
        <div class="form-row">
          <label class="sr-only" for="speed-person">Person</label>
          <select id="speed-person" bind:value={speedTargetId} class="field-input field-sm" aria-label="Choose a person">
            <option value="">Choose a person</option>
            {#each visibleUsers as u}
              <option value={u.id}>{u.name}</option>
            {/each}
          </select>
          <label class="field-label-inline">
            Limit
            <input
              type="number"
              bind:value={speedThreshold}
              class="field-input field-num"
              min="10"
              max="300"
              step="5"
              aria-label="Speed limit in km/h"
            />
            <span aria-hidden="true">km/h</span>
          </label>
          <button
            class="btn btn-primary btn-sm"
            on:click={addSpeedAlert}
            disabled={!speedTargetId}
            style="min-height: 44px;"
          >
            Add
          </button>
        </div>
      </div>
    </div>

  </div>
{:else}
  <div class="panel-shell panel-left panel-base">
    <div class="panel-header">
      <h3>Places</h3>
      <button class="panel-close" on:click={() => dispatch('close')} aria-label="Close places panel">&times;</button>
    </div>
    <div class="panel-body">
      <p>Open Places from the sidebar.</p>
    </div>
  </div>
{/if}

<style>
  .places-panel { padding: 0; }

  /* ── Section header ────────────────────────────────────────────────────── */
  .section-header {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-4) var(--space-4) var(--space-1);
  }

  .section-label {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    flex: 1;
  }

  .section-badge {
    font-family: var(--font-display);
    font-size: var(--text-2xs);
    font-weight: 700;
    color: var(--text-tertiary);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-full);
    padding: 1px var(--space-2);
    min-width: 20px;
    text-align: center;
  }

  .section-content { padding: 0 var(--space-4) var(--space-3); }

  .section-divider {
    height: 1px;
    background: var(--border-subtle);
    margin: var(--space-2) 0;
  }

  .hint {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    line-height: var(--leading-normal);
    margin: 0 0 var(--space-2);
  }

  /* ── List items ──────────────────────────────────────────────────────────── */
  .list-item {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
    padding: var(--space-2) 0;
    border-bottom: 1px solid var(--border-subtle);
  }
  .list-item:last-of-type { border-bottom: none; }

  .item-icon {
    font-size: var(--text-lg);
    flex-shrink: 0;
    width: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
  }

  .item-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .item-name {
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .item-detail {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }

  /* Icon action buttons — 44px touch target */
  .icon-action {
    width: 44px;
    height: 44px;
    border-radius: var(--radius-sm2);
    border: none;
    background: transparent;
    color: var(--text-tertiary);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: background var(--duration-fast) var(--ease-out),
                color var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }

  .icon-action:hover {
    background: var(--surface-hover);
    color: var(--text-primary);
  }

  .icon-action:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 2px;
  }

  .icon-action--danger:hover {
    background: rgba(239, 68, 68, 0.10);
    color: var(--danger-500);
  }

  /* ── Empty states ────────────────────────────────────────────────────────── */
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-6) var(--space-4);
    text-align: center;
    background: var(--surface-inset);
    border: 1px dashed var(--border-subtle);
    border-radius: var(--radius-lg);
    margin-bottom: var(--space-2);
  }

  .empty-icon {
    width: 48px;
    height: 48px;
    border-radius: var(--radius-full);
    background: linear-gradient(135deg, rgba(20, 184, 166, 0.12) 0%, rgba(20, 184, 166, 0.06) 100%);
    border: 1px solid rgba(20, 184, 166, 0.18);
    color: var(--primary-500);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .empty-title {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    margin: 0;
  }

  .empty-sub {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    line-height: var(--leading-normal);
    margin: 0;
    max-width: 220px;
  }

  .inline-empty {
    padding: var(--space-2) 0 var(--space-3);
  }

  .inline-empty-text {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    margin: 0 0 var(--space-1);
  }

  .inline-empty-cta-hint {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    font-style: italic;
    margin: 0;
    opacity: 0.75;
  }

  /* ── Zone Story Timeline ──────────────────────────────────────────────────── */
  .zone-story {
    margin: 0 0 var(--space-2);
    padding: var(--space-2) var(--space-3);
    background: var(--surface-inset);
    border-left: 3px solid var(--primary-500);
    border-radius: 0 var(--radius-md) var(--radius-md) 0;
  }

  .story-skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-1) 0;
  }

  .skel-row {
    height: var(--space-3);
    border-radius: var(--radius-sm);
    background: var(--skeleton-base, rgba(255,255,255,0.05));
    animation: skel-pulse 1.6s ease-in-out infinite;
  }
  .skel-wide  { width: 100%; }
  .skel-mid   { width: 65%; }

  @keyframes skel-pulse {
    0%, 100% { opacity: 0.5; }
    50%       { opacity: 1; }
  }

  .story-empty {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    margin: 0;
    padding: var(--space-1) 0;
  }

  .story-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-1-5) 0;
    border-bottom: 1px solid var(--border-subtle);
  }
  .story-row:last-of-type { border-bottom: none; }

  .story-avatar {
    width: 24px;
    height: 24px;
    border-radius: var(--radius-full);
    background: var(--primary-500);
    color: var(--text-inverse);
    font-size: var(--text-2xs);
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    text-transform: uppercase;
  }

  .story-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .story-name {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-primary);
  }

  .story-time {
    font-size: var(--text-2xs);
    color: var(--text-tertiary);
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
  }

  .story-badge-here {
    font-size: var(--text-2xs);
    font-weight: 600;
    color: var(--success-600);
    background: rgba(16, 185, 129, 0.12);
    border: 1px solid rgba(16, 185, 129, 0.24);
    padding: 2px var(--space-1-5);
    border-radius: var(--radius-full);
    flex-shrink: 0;
    white-space: nowrap;
  }

  /* ── Add form ────────────────────────────────────────────────────────────── */
  .add-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    margin-top: var(--space-2);
    padding: var(--space-3);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
  }

  .form-row {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    flex-wrap: wrap;
  }

  .form-row--check {
    flex-wrap: wrap;
    gap: var(--space-3);
  }

  .field-input {
    padding: var(--space-2) var(--space-2-5);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    background: var(--surface-3);
    color: var(--text-primary);
    font-family: var(--font-sans);
    transition: border-color var(--duration-fast) var(--ease-out),
                box-shadow var(--duration-fast) var(--ease-out);
  }

  .field-input::placeholder { color: var(--text-tertiary); }

  .field-input option {
    background: var(--surface-2);
    color: var(--text-primary);
  }

  .field-input:focus {
    outline: none;
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px rgba(20, 184, 166, 0.18);
  }

  .field-full { width: 100%; box-sizing: border-box; }
  .field-sm  { flex: 1; min-width: 80px; }
  .field-num { width: 64px; flex: none; }

  .field-label-inline {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    font-size: var(--text-xs);
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .check-label {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    font-size: var(--text-xs);
    color: var(--text-primary);
    cursor: pointer;
    user-select: none;
    min-height: 44px;
  }

  .form-actions {
    display: flex;
    gap: var(--space-2);
    flex-wrap: wrap;
  }

  .add-btn {
    align-self: flex-start;
    margin-top: var(--space-2);
  }

  /* ── Screen reader only ─────────────────────────────────────────────────── */
  .sr-only {
    position: absolute;
    width: 1px; height: 1px;
    padding: 0; margin: -1px;
    overflow: hidden;
    clip: rect(0,0,0,0);
    white-space: nowrap;
    border: 0;
  }

  /* ── Reduced motion ──────────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .skel-row { animation: none; }
  }
</style>
