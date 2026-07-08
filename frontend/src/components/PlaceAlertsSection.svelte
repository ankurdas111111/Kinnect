<script>
  /**
   * PlaceAlertsSection — arrival/departure alerts and speed alerts.
   * Receives stores and callbacks from SavedPlacesPanel (the orchestrator).
   */
  import { placeAlerts, speedAlerts } from '../lib/stores/places.js';


  let {
    places = [],
    visibleUsers = [],
    getUserName,
    iconEmoji = {},
    onAddPlaceAlert,
    onRemovePlaceAlert,
    onAddSpeedAlert,
    onRemoveSpeedAlert
  } = $props();

  let alertTargetId = $state('');
  let alertPlaceId = $state('');
  let alertOnArrive = $state(true);
  let alertOnDepart = $state(true);

  let speedTargetId = $state('');
  let speedThreshold = $state(80);

  function handleAddPlaceAlert() {
    if (!alertTargetId || !alertPlaceId) return;
    onAddPlaceAlert(alertTargetId, alertPlaceId, alertOnArrive, alertOnDepart);
    alertTargetId = '';
    alertPlaceId = '';
  }

  function handleAddSpeedAlert() {
    if (!speedTargetId) return;
    onAddSpeedAlert(speedTargetId, speedThreshold);
    speedTargetId = '';
  }
</script>

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
      {#if places.length === 0}
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
          onclick={() => onRemovePlaceAlert(alert.id)}
          aria-label="Remove alert: {getUserName(alert.targetId)} at {alert.placeName}"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>
        </button>
      </div>
    {/each}
  {/if}

  {#if places.length > 0}
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
          {#each places as p}
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
        onclick={handleAddPlaceAlert}
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
          onclick={() => onRemoveSpeedAlert(sa.id)}
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
        onclick={handleAddSpeedAlert}
        disabled={!speedTargetId}
        style="min-height: 44px;"
      >
        Add
      </button>
    </div>
  </div>
</div>

<style>
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

  .icon-action:hover { background: var(--surface-hover); color: var(--text-primary); }
  .icon-action:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }
  .icon-action--danger:hover { background: rgba(239, 68, 68, 0.10); color: var(--danger-500); }

  .inline-empty { padding: var(--space-2) 0 var(--space-3); }

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
  .field-input option { background: var(--surface-2); color: var(--text-primary); }
  .field-input:focus {
    outline: none;
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px rgba(20, 184, 166, 0.18);
  }

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

  .sr-only {
    position: absolute;
    width: 1px; height: 1px;
    padding: 0; margin: -1px;
    overflow: hidden;
    clip: rect(0,0,0,0);
    white-space: nowrap;
    border: 0;
  }
</style>
