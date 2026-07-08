<script>
  /**
   * SavedPlacesPanel — orchestrates saved places, zone stories, and alerts.
   * Split into:
   *   - PlacesListSection   (places list + zone story + add form)
   *   - PlaceAlertsSection  (arrival/departure alerts + speed alerts)
   */
  import { createEventDispatcher, onMount } from 'svelte';
  import { banner } from '../lib/stores/sos.js';
  import { savedPlaces, placeAlerts, speedAlerts } from '../lib/stores/places.js';
  import { otherUsers, myLocation } from '../lib/stores/map.js';
  import { authUser } from '../lib/stores/auth.js';
  import { apiGet, apiPost, apiDelete } from '../lib/api.js';
  import PlacesListSection from './PlacesListSection.svelte';
  import PlaceAlertsSection from './PlaceAlertsSection.svelte';

  /**
   * @typedef {Object} Props
   * @property {boolean} [embedded]
   */

  /** @type {Props} */
  let { embedded = false } = $props();

  const dispatch = createEventDispatcher();

  const iconOptions = [
    { value: 'home',   label: 'Home'   },
    { value: 'work',   label: 'Work'   },
    { value: 'school', label: 'School' },
    { value: 'gym',    label: 'Gym'    },
    { value: 'pin',    label: 'Other'  },
  ];
  const iconEmoji = { home: '🏠', work: '💼', school: '🏫', gym: '🏋️', pin: '📍' };

  const PALERT_KEY = 'kinnect_place_alerts';
  const SALERT_KEY = 'kinnect_speed_alerts';

  // Zone story state
  let storyPlaceId = $state(null);
  let storyVisits = $state([]);
  let storyLoading = $state(false);


  function buildUserList(others, auth) {
    const list = [];
    if (auth) list.push({ id: auth.userId, name: 'Me' });
    for (const u of others.values()) {
      if (u.userId) list.push({ id: u.userId, name: u.displayName || u.userId.slice(0, 6) });
    }
    return list;
  }

  function persist(key, value) {
    try { localStorage.setItem(key, JSON.stringify(value)); } catch (_) {}
  }

  onMount(async () => {
    try {
      const pa = localStorage.getItem(PALERT_KEY);
      if (pa) placeAlerts.set(JSON.parse(pa));
      const sa = localStorage.getItem(SALERT_KEY);
      if (sa) speedAlerts.set(JSON.parse(sa));
    } catch (_) {}

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

  // ── Places CRUD ───────────────────────────────────────────────────────────

  async function handleAddPlace(name, radius, icon) {
    const loc = $myLocation;
    if (!loc) {
      banner.set({ type: 'info', text: 'Start location tracking first so we can pin your current position', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
      return false;
    }
    const res = await apiPost('/api/places', {
      name,
      latitude: loc.latitude,
      longitude: loc.longitude,
      radiusM: radius,
      icon,
    });
    if (res && res.id) {
      savedPlaces.update(arr => [...arr, {
        id: res.id, name: res.name,
        lat: res.latitude, lng: res.longitude,
        radiusM: res.radiusM, icon: res.icon || 'pin',
      }]);
      banner.set({ type: 'info', text: `"${res.name}" saved at your current location`, actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2500);
      return true;
    }
    banner.set({ type: 'sos', text: res?.error || 'Failed to save place — check your connection and try again', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
    return false;
  }

  async function handleRemovePlace(placeId) {
    await apiDelete(`/api/places/${placeId}`);
    savedPlaces.update(arr => arr.filter(p => p.id !== placeId));
    placeAlerts.update(arr => {
      const next = arr.filter(a => a.placeId !== placeId);
      persist(PALERT_KEY, next);
      return next;
    });
    if (storyPlaceId === placeId) storyPlaceId = null;
  }

  async function handleViewStory(placeId) {
    if (storyPlaceId === placeId) { storyPlaceId = null; return; }
    storyPlaceId = placeId;
    storyVisits = [];
    storyLoading = true;
    const res = await apiGet(`/api/places/${placeId}/story`);
    storyLoading = false;
    storyVisits = Array.isArray(res) ? res : [];
  }

  // ── Alerts ────────────────────────────────────────────────────────────────

  function handleAddPlaceAlert(targetId, placeId, onArrive, onDepart) {
    const place = $savedPlaces.find(p => p.id === placeId);
    const user  = visibleUsers.find(u => u.id === targetId);
    const alert = {
      id: Date.now().toString(),
      targetId,
      targetName: user?.name || targetId,
      placeId,
      placeName: place?.name || '?',
      onArrive,
      onDepart,
    };
    placeAlerts.update(arr => {
      const next = [...arr, alert];
      persist(PALERT_KEY, next);
      return next;
    });
    banner.set({ type: 'info', text: 'Arrival/departure alert added', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2500);
  }

  function handleRemovePlaceAlert(alertId) {
    placeAlerts.update(arr => {
      const next = arr.filter(a => a.id !== alertId);
      persist(PALERT_KEY, next);
      return next;
    });
  }

  function handleAddSpeedAlert(targetId, thresholdKmh) {
    const user = visibleUsers.find(u => u.id === targetId);
    const alert = {
      id: Date.now().toString(),
      targetId,
      targetName: user?.name || targetId,
      thresholdKmh,
    };
    speedAlerts.update(arr => {
      const next = [...arr, alert];
      persist(SALERT_KEY, next);
      return next;
    });
    banner.set({ type: 'info', text: `Speed alert set — you'll be notified above ${thresholdKmh} km/h`, actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2500);
  }

  function handleRemoveSpeedAlert(alertId) {
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
  let visibleUsers = $derived(buildUserList($otherUsers, $authUser));
</script>

{#if embedded}
  <div class="panel-body places-panel">

    <PlacesListSection
      places={$savedPlaces}
      {iconOptions}
      {iconEmoji}
      onAdd={handleAddPlace}
      onRemove={handleRemovePlace}
      onViewStory={handleViewStory}
      {storyPlaceId}
      {storyLoading}
      {storyVisits}
    />

    <div class="section-divider" role="separator"></div>

    <PlaceAlertsSection
      places={$savedPlaces}
      {visibleUsers}
      {getUserName}
      {iconEmoji}
      onAddPlaceAlert={handleAddPlaceAlert}
      onRemovePlaceAlert={handleRemovePlaceAlert}
      onAddSpeedAlert={handleAddSpeedAlert}
      onRemoveSpeedAlert={handleRemoveSpeedAlert}
    />

  </div>
{:else}
  <div class="panel-shell panel-left panel-base">
    <div class="panel-header">
      <h3>Places</h3>
      <button
        class="btn btn-icon btn-ghost"
        onclick={() => dispatch('close')}
        aria-label="Close places panel"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
    <div class="panel-body">
      <p class="stub-hint">Manage saved places from the sidebar.</p>
    </div>
  </div>
{/if}

<style>
  .places-panel { padding: 0; }

  .section-divider {
    height: 1px;
    background: var(--border-subtle);
    margin: var(--space-2) 0;
  }

  .stub-hint {
    font-size: var(--text-sm);
    color: var(--text-tertiary);
    padding: var(--space-4);
  }
</style>
