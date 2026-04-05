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
  let storyPlaceId = null;     // which place's story is open
  let storyVisits = [];        // fetched visits
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

  // ── Persistence helpers (localStorage — only for alerts, places now live in backend) ──
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
    // Load saved places from backend (canonical source)
    const res = await apiGet('/api/places');
    if (Array.isArray(res)) {
      // Map backend field names to frontend convention
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

  // ── Saved Places ─────────────────────────────────────────────────────────────
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
      banner.set({ type: 'sos', text: res?.error || 'Failed to save place', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
    }
  }

  async function removePlace(placeId) {
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
    const d = new Date(iso);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
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

  // ── Speed Alerts ─────────────────────────────────────────────────────────────
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

    <!-- ── Your Spots ─────────────────────────────────────────────────── -->
    <h4 class="section-title-bold">Saved Places</h4>
    <div class="section-content">
      <p class="hint">Save places like home or work. You'll get notified when family members arrive or leave.</p>

      {#each $savedPlaces as place}
        <div class="list-item">
          <div class="item-icon">{iconEmoji[place.icon] ?? '📍'}</div>
          <div class="item-info">
            <span class="item-name">{place.name}</span>
            <span class="item-detail">{place.radiusM}m radius · {place.lat?.toFixed(4)}, {place.lng?.toFixed(4)}</span>
          </div>
          <button class="btn-icon-sm story-btn" on:click={() => viewStory(place.id)} title="View visit history">
            {storyPlaceId === place.id ? '▲' : '📋'}
          </button>
          <button class="btn-icon-sm" on:click={() => removePlace(place.id)} title="Remove">✕</button>
        </div>
        {#if storyPlaceId === place.id}
          <div class="zone-story">
            {#if storyLoading}
              <p class="story-empty">Loading...</p>
            {:else if storyVisits.length === 0}
              <p class="story-empty">No visits in the last 7 days.</p>
            {:else}
              {#each storyVisits as v}
                <div class="story-row">
                  <div class="story-avatar">{v.displayName?.charAt(0) ?? '?'}</div>
                  <div class="story-info">
                    <span class="story-name">{v.displayName}</span>
                    <span class="story-time">
                      {formatDate(v.arrivedAt)} · {formatTime(v.arrivedAt)}
                      {#if v.departedAt} – {formatTime(v.departedAt)}{/if}
                      {#if v.durationSeconds} · {formatDuration(v.durationSeconds)}{/if}
                    </span>
                  </div>
                  {#if !v.departedAt}<span class="story-badge-here">Here now</span>{/if}
                </div>
              {/each}
            {/if}
          </div>
        {/if}
      {/each}

      {#if showAddPlace}
        <div class="add-form">
          <input
            type="text"
            bind:value={newPlaceName}
            class="field-input field-full"
            placeholder="Name this place (Home, Work, School...)"
            maxlength="100"
            autocomplete="off"
          />
          <div class="form-row">
            <select bind:value={newPlaceIcon} class="field-input field-sm">
              {#each iconOptions as opt}
                <option value={opt.value}>{iconEmoji[opt.value]} {opt.label}</option>
              {/each}
            </select>
            <label class="field-label-inline">
              Radius
              <input type="number" bind:value={newPlaceRadius} class="field-input field-num" min="50" max="5000" step="50" />
              m
            </label>
          </div>
          <div class="form-actions">
            <button class="btn btn-primary btn-sm" on:click={addPlace} disabled={!newPlaceName.trim()}>
              Save This Place
            </button>
            <button class="btn btn-secondary btn-sm" on:click={() => { showAddPlace = false; newPlaceName = ''; }}>Cancel</button>
          </div>
        </div>
      {:else}
        <button class="btn btn-secondary btn-sm add-btn" on:click={() => showAddPlace = true}>Add a Place</button>
      {/if}
    </div>

    <hr class="divider" />

    <!-- ── Arrival Pings ──────────────────────────────────────────────── -->
    <h4 class="section-title-bold">Arrival Alerts</h4>
    <div class="section-content">
      <p class="hint">Get notified when someone arrives at or leaves one of your saved places.</p>

      {#if $placeAlerts.length === 0}
        <p class="empty">No alerts yet. Add one for home so your family knows when you arrive.</p>
      {/if}
      {#each $placeAlerts as alert}
        <div class="list-item">
          <div class="item-icon">🔔</div>
          <div class="item-info">
            <span class="item-name">{getUserName(alert.targetId)} at {alert.placeName || '?'}</span>
            <span class="item-detail">
              {[alert.onArrive && 'Arrive', alert.onDepart && 'Depart'].filter(Boolean).join(' + ')}
            </span>
          </div>
          <button class="btn-icon-sm" on:click={() => removePlaceAlert(alert.id)} title="Remove">✕</button>
        </div>
      {/each}

      {#if $savedPlaces.length > 0}
        <div class="add-form">
          <div class="form-row">
            <select bind:value={alertTargetId} class="field-input field-sm">
              <option value="">Choose a person</option>
              {#each visibleUsers as u}
                <option value={u.id}>{u.name}</option>
              {/each}
            </select>
            <select bind:value={alertPlaceId} class="field-input field-sm">
              <option value="">Choose a place</option>
              {#each $savedPlaces as p}
                <option value={p.id}>{iconEmoji[p.icon] ?? '📍'} {p.name}</option>
              {/each}
            </select>
          </div>
          <div class="form-row">
            <label class="check-label"><input type="checkbox" bind:checked={alertOnArrive} /> When they arrive</label>
            <label class="check-label"><input type="checkbox" bind:checked={alertOnDepart} /> When they leave</label>
            <button class="btn btn-primary btn-sm" on:click={addPlaceAlert} disabled={!alertTargetId || !alertPlaceId || (!alertOnArrive && !alertOnDepart)}>Add</button>
          </div>
        </div>
      {:else}
        <p class="empty">Save a place first to set up alerts.</p>
      {/if}
    </div>

    <hr class="divider" />

    <!-- ── Speed Checks ─────────────────────────────────────────────────── -->
    <h4 class="section-title-bold">Speed Alerts</h4>
    <div class="section-content">
      <p class="hint">Get a heads-up when someone drives faster than a set limit.</p>

      {#if $speedAlerts.length === 0}
        <p class="empty">No speed alerts yet.</p>
      {/if}
      {#each $speedAlerts as sa}
        <div class="list-item">
          <div class="item-icon">🚗</div>
          <div class="item-info">
            <span class="item-name">{getUserName(sa.targetId)}</span>
            <span class="item-detail">Alert above {sa.thresholdKmh} km/h</span>
          </div>
          <button class="btn-icon-sm" on:click={() => removeSpeedAlert(sa.id)} title="Remove">✕</button>
        </div>
      {/each}

      <div class="add-form">
        <div class="form-row">
          <select bind:value={speedTargetId} class="field-input field-sm">
            <option value="">Choose a person</option>
            {#each visibleUsers as u}
              <option value={u.id}>{u.name}</option>
            {/each}
          </select>
          <label class="field-label-inline">
            Limit
            <input type="number" bind:value={speedThreshold} class="field-input field-num" min="10" max="300" step="5" />
            km/h
          </label>
          <button class="btn btn-primary btn-sm" on:click={addSpeedAlert} disabled={!speedTargetId}>Add</button>
        </div>
      </div>
    </div>

  </div>
{:else}
  <div class="panel-shell panel-left panel-base">
    <div class="panel-header">
      <h3>Places</h3>
      <button class="panel-close" on:click={() => dispatch('close')} aria-label="Close">&times;</button>
    </div>
    <div class="panel-body">
      <p>Open Places from the sidebar.</p>
    </div>
  </div>
{/if}

<style>
  .places-panel { padding: 0; }

  h4.section-title-bold {
    margin: var(--space-4) 0 var(--space-1);
    padding: 0 var(--space-4);
  }

  .section-content { padding: 0 var(--space-4) var(--space-2); }

  .hint {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    line-height: 1.5;
    margin: 0 0 var(--space-2);
  }

  /* ── List items ──────────────────────────────────────────────────────────── */
  .list-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 0;
    border-bottom: 1px solid var(--border-subtle);
  }
  .list-item:last-of-type { border-bottom: none; }

  .item-icon { font-size: 18px; flex-shrink: 0; }

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

  .item-detail { font-size: var(--text-xs); color: var(--text-tertiary); }

  .btn-icon-sm {
    width: 26px;
    height: 26px;
    border-radius: 50%;
    border: none;
    background: var(--surface-3);
    color: var(--text-tertiary);
    cursor: pointer;
    font-size: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: background 0.15s, color 0.15s;
  }
  .btn-icon-sm:hover { background: rgba(220, 38, 38, 0.18); color: #f87171; }
  .story-btn:hover { background: var(--surface-3); color: var(--text-secondary); }

  /* ── Zone Story Timeline ──────────────────────────────────────────────────── */
  .zone-story {
    margin: 4px 0 8px;
    padding: 8px 10px;
    background: var(--surface-inset);
    border-left: 3px solid var(--color-primary, #6366f1);
    border-radius: 0 var(--radius-md) var(--radius-md) 0;
  }
  .story-empty { font-size: var(--text-xs); color: var(--text-tertiary); margin: 0; }
  .story-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 5px 0;
    border-bottom: 1px solid var(--border-subtle);
  }
  .story-row:last-of-type { border-bottom: none; }
  .story-avatar {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: var(--color-primary, #6366f1);
    color: #fff;
    font-size: 11px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    text-transform: uppercase;
  }
  .story-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1px; }
  .story-name { font-size: var(--text-xs); font-weight: 600; color: var(--text-primary); }
  .story-time { font-size: 10px; color: var(--text-tertiary); }
  .story-badge-here {
    font-size: 10px;
    font-weight: 600;
    color: #22c55e;
    background: rgba(34, 197, 94, 0.15);
    padding: 2px 6px;
    border-radius: 999px;
    flex-shrink: 0;
  }

  /* ── Add form ────────────────────────────────────────────────────────────── */
  .add-form {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 8px;
    padding: 12px;
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
  }

  .form-row {
    display: flex;
    gap: 8px;
    align-items: center;
    flex-wrap: wrap;
  }

  .field-input {
    padding: 8px 10px;
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    background: var(--surface-3);
    color: var(--text-primary);
  }
  .field-input::placeholder { color: var(--text-tertiary); }
  .field-input option { background: var(--surface-2); color: var(--text-primary); }
  .field-full { width: 100%; box-sizing: border-box; }
  .field-sm  { flex: 1; min-width: 90px; padding: 6px 8px; font-size: var(--text-xs); }
  .field-num { width: 64px; flex: none; padding: 6px 8px; font-size: var(--text-xs); }

  .field-input:focus {
    outline: none;
    border-color: var(--primary-500);
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.25);
  }

  .field-label-inline {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: var(--text-xs);
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .check-label {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: var(--text-xs);
    color: var(--text-primary);
    cursor: pointer;
    user-select: none;
  }

  .form-actions { display: flex; gap: 8px; flex-wrap: wrap; }

  .btn-sm { padding: 6px 14px; font-size: var(--text-xs); min-height: 32px; }

  .add-btn { align-self: flex-start; margin-top: 4px; }

  .empty { font-size: var(--text-xs); color: var(--text-tertiary); margin: 4px 0 6px; }

  .divider {
    border: none;
    border-top: 1px solid var(--border-subtle);
    margin: var(--space-2) 0;
  }
</style>
