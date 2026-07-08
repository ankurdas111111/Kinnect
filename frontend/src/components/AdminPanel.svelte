<script>
  import { createEventDispatcher } from 'svelte';
  import { socket } from '../lib/socket.js';
  import { authUser } from '../lib/stores/auth.js';
  import { otherUsers, myLocation } from '../lib/stores/map.js';
  import { canManage } from '../lib/stores/guardians.js';
  import { banner } from '../lib/stores/sos.js';

  /**
   * @typedef {Object} Props
   * @property {boolean} [embedded]
   */

  /** @type {Props} */
  let { embedded = false } = $props();

  const dispatch = createEventDispatcher();


  let targetId = $state('me');
  let autoSosEnabled = $state(false);
  let noMoveMin = $state(5);
  let hardStopMin = $state(2);
  let geofenceEnabled = $state(false);
  let geofenceRadius = $state(0);
  let checkInEnabled = $state(false);
  let checkInIntervalMin = $state(5);
  let checkInOverdueMin = $state(7);
  let keepForever = $state(false);


  function buildTargetOptions(users, cm, admin) {
    const opts = [{ value: 'me', label: ($authUser?.displayName || 'Me') + ' (me)' }];
    if (admin) {
      for (const [, u] of users) {
        opts.push({ value: u.socketId, label: u.displayName || u.socketId });
      }
    } else {
      for (const [uid, name] of cm) {
        if (uid === $authUser?.userId) continue;
        for (const [, u] of users) {
          if (u.userId === uid) opts.push({ value: u.socketId, label: u.displayName || u.socketId });
        }
      }
    }
    return opts;
  }

  function getTarget() {
    if (targetId === 'me') {
      return { socketId: null, latitude: $myLocation?.latitude, longitude: $myLocation?.longitude };
    }
    const u = $otherUsers.get(targetId);
    return u || { socketId: targetId };
  }

  function applySettings() {
    const target = getTarget();
    const sid = target.socketId || undefined;

    socket.emit('setAutoSos', {
      socketId: sid, enabled: autoSosEnabled,
      noMoveMinutes: noMoveMin > 0 ? noMoveMin : 5,
      hardStopMinutes: hardStopMin > 0 ? hardStopMin : 2,
      geofence: geofenceEnabled
    });

    if (geofenceEnabled) {
      if (!geofenceRadius) { banner.set({ type: 'info', text: 'Set a geofence radius first.', actions: [] }); return; }
      const lat = target.latitude ?? $myLocation?.latitude;
      const lng = target.longitude ?? $myLocation?.longitude;
      if (lat == null || lng == null) { banner.set({ type: 'info', text: 'Start Tracking to set geofence center.', actions: [] }); return; }
      socket.emit('setGeofence', { socketId: sid, enabled: true, centerLat: lat, centerLng: lng, radiusM: geofenceRadius });
    } else {
      socket.emit('setGeofence', { socketId: sid, enabled: false });
    }

    socket.emit('setCheckInRules', {
      socketId: sid, enabled: checkInEnabled,
      intervalMinutes: checkInIntervalMin > 0 ? checkInIntervalMin : 5,
      overdueMinutes: checkInOverdueMin > 0 ? checkInOverdueMin : 7
    });

    if (keepForever && sid) socket.emit('setRetentionForever', { socketId: sid, forever: keepForever });

    banner.set({ type: 'info', text: 'Admin settings applied.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 1500);
  }
  let isAdmin = $derived($authUser && $authUser.role === 'admin');
  let hasManageables = $derived(isAdmin || $canManage.size > 0);
  let targetOptions = $derived(buildTargetOptions($otherUsers, $canManage, isAdmin));
</script>

{#if hasManageables}
  {#if embedded}
    <div class="panel-body">
      <div class="section">
        <label class="label" for="admin-target-user-embedded">Choose a person</label>
        <select id="admin-target-user-embedded" class="select" bind:value={targetId}>
          {#each targetOptions as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>
      </div>

      <hr class="divider" />

      <!-- MERIDIAN: Amber card for auto-SOS config -->
      <div class="auto-sos-card">
        <span class="label-eyebrow auto-sos-eyebrow">
          <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/></svg>
          Auto-SOS
        </span>
        <div class="section" style="padding: 0;">
          <label class="toggle"><input type="checkbox" bind:checked={autoSosEnabled}><span class="toggle-track"></span>Auto SOS</label>
          <div class="field-row">
            <span class="label mini">Alert if still for (min)</span>
            <input class="input w-input-sm" type="number" bind:value={noMoveMin} min="1" />
          </div>
          <div class="field-row">
            <span class="label mini">Alert on sudden stop (min)</span>
            <input class="input w-input-sm" type="number" bind:value={hardStopMin} min="1" />
          </div>
        </div>
      </div>

      <hr class="divider" />

      <div class="section">
        <label class="toggle"><input type="checkbox" bind:checked={geofenceEnabled}><span class="toggle-track"></span>Geofence</label>
        <div class="field-row">
          <span class="label mini">Safe zone radius (m)</span>
          <input class="input w-input-md" type="number" bind:value={geofenceRadius} min="0" />
        </div>
      </div>

      <hr class="divider" />

      <div class="section">
        <label class="toggle"><input type="checkbox" bind:checked={checkInEnabled}><span class="toggle-track"></span>Check-In</label>
        <div class="field-row">
          <span class="label mini">Check-in every (min)</span>
          <input class="input w-input-sm" type="number" bind:value={checkInIntervalMin} min="1" />
        </div>
        <div class="field-row">
          <span class="label mini">Alert if late by (min)</span>
          <input class="input w-input-sm" type="number" bind:value={checkInOverdueMin} min="1" />
        </div>
      </div>

      {#if isAdmin}
        <hr class="divider" />
        <div class="section">
          <label class="toggle"><input type="checkbox" bind:checked={keepForever}><span class="toggle-track"></span>Save location history</label>
        </div>
      {/if}

      <button class="btn btn-primary full-width mt-4" onclick={applySettings}>Save</button>
    </div>
  {:else}
    <div class="panel-shell panel-left panel-base">
      <div class="panel-header">
        <h3>Safety Settings</h3>
        <button class="btn btn-icon btn-ghost" aria-label="Close admin panel" onclick={() => dispatch('close')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>

      <div class="panel-body">
        <div class="section">
          <label class="label" for="admin-target-user-panel">Choose a person</label>
          <select id="admin-target-user-panel" class="select" bind:value={targetId}>
            {#each targetOptions as opt}
              <option value={opt.value}>{opt.label}</option>
            {/each}
          </select>
        </div>
        <hr class="divider" />
        <div class="section">
          <label class="toggle"><input type="checkbox" bind:checked={autoSosEnabled}><span class="toggle-track"></span>Auto SOS</label>
        </div>
        <hr class="divider" />
        <div class="section">
          <label class="toggle"><input type="checkbox" bind:checked={geofenceEnabled}><span class="toggle-track"></span>Geofence</label>
        </div>
        <hr class="divider" />
        <div class="section">
          <label class="toggle"><input type="checkbox" bind:checked={checkInEnabled}><span class="toggle-track"></span>Check-In</label>
        </div>
        <button class="btn btn-primary full-width mt-4" onclick={applySettings}>Save</button>
      </div>
    </div>
  {/if}
{:else}
  {#if embedded}
    <div class="panel-body">
      <div class="empty-state">
        <p>You need admin or guardian access to manage someone's safety settings.</p>
      </div>
    </div>
  {/if}
{/if}

<style>
  /* MERIDIAN: Amber card for auto-SOS section */
  .auto-sos-card {
    background: rgba(245, 158, 11, 0.07);
    border: 1px solid rgba(245, 158, 11, 0.22);
    border-radius: var(--radius-md);
    padding: var(--space-3);
    margin: var(--space-2) 0;
  }

  .auto-sos-eyebrow {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.10em;
    text-transform: uppercase;
    color: var(--warning-600, #d97706);
    margin-bottom: var(--space-2);
  }
</style>
