<script>
  import { run } from 'svelte/legacy';

  import { onDestroy } from 'svelte';
  import { geofenceLog } from '../../lib/stores/places.js';
  import { emitGetGeofenceLog } from '../../lib/socket.js';

  // ── F6: Geofence log ───────────────────────────────────────────────────────
  let geofenceLogOpen = $state(false);
  let loading = $state(false);
  let loadTimer = $state(null);

  function toggleGeofenceLog() {
    geofenceLogOpen = !geofenceLogOpen;
    if (geofenceLogOpen) {
      emitGetGeofenceLog();
      loading = true;
      clearTimeout(loadTimer);
      loadTimer = setTimeout(() => { loading = false; }, 1500);
    }
  }

  // Data arrived — stop showing the skeleton immediately
  run(() => {
    if (loading && $geofenceLog.length > 0) {
      loading = false;
      clearTimeout(loadTimer);
    }
  });

  onDestroy(() => clearTimeout(loadTimer));

  function formatGeofenceTs(ts) {
    if (!ts) return '';
    return new Date(ts).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
  }
</script>

<!-- ── F6: GEOFENCE LOG ───────────────────────────────────────────── -->
<div class="feature-section">
  <button class="collapsible-header" onclick={toggleGeofenceLog} aria-expanded={geofenceLogOpen}>
    <span class="card-eyebrow" style="margin:0">Zone History</span>
    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true" style="transform: rotate({geofenceLogOpen ? 180 : 0}deg); transition: transform 200ms"><polyline points="6 9 12 15 18 9"/></svg>
  </button>
  {#if geofenceLogOpen}
    <div class="log-list">
      {#if loading && $geofenceLog.length === 0}
        <!-- Skeleton while the log loads over the socket -->
        <div class="skeleton-block" role="status" aria-label="Loading zone history" aria-busy="true">
          <div class="skel-row skel-wide"></div>
          <div class="skel-row skel-mid"></div>
          <div class="skel-row skel-short"></div>
        </div>
      {:else if $geofenceLog.length === 0}
        <p class="empty-hint">No geofence events yet.</p>
      {:else}
        {#each $geofenceLog as ev}
          <div class="log-item" class:log-entry={ev.eventType === 'entry'} class:log-exit={ev.eventType === 'exit'}>
            <span class="log-badge">{ev.eventType === 'entry' ? 'In' : 'Out'}</span>
            <span class="log-name">{ev.fenceName || 'Geofence'}</span>
            <span class="log-time">{formatGeofenceTs(ev.ts)}</span>
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>

<style>
  .feature-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    background: var(--surface-1);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-xl);
    padding: var(--space-3) var(--space-4);
  }

  .collapsible-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    width: 100%;
    min-height: 44px;
    color: var(--text-primary);
  }
  .collapsible-header:hover { opacity: 0.8; }

  /* F6: geofence log */
  .log-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 180px;
    overflow-y: auto;
  }

  .log-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 5px 8px;
    border-radius: var(--radius-md);
    font-size: var(--text-xs);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
  }

  .log-badge {
    font-size: 9px;
    font-weight: 800;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    padding: 2px 6px;
    border-radius: var(--radius-full);
    flex-shrink: 0;
  }

  .log-entry .log-badge {
    background: rgba(16, 185, 129, 0.14);
    color: var(--success-500);
    border: 1px solid rgba(16, 185, 129, 0.25);
  }

  .log-exit .log-badge {
    background: rgba(239, 68, 68, 0.12);
    color: var(--danger-500);
    border: 1px solid rgba(239, 68, 68, 0.22);
  }

  .log-name {
    flex: 1;
    font-weight: 600;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .log-time {
    font-size: 10px;
    color: var(--text-tertiary);
    flex-shrink: 0;
  }

  .empty-hint {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    text-align: center;
    padding: 8px 0;
    margin: 0;
  }

  /* ── Loading skeleton ─────────────────────────────────────────────────── */
  .skeleton-block {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-2) 0;
  }

  .skel-row {
    height: var(--space-4);
    border-radius: var(--radius-sm);
    background: var(--skeleton-base, rgba(255,255,255,0.05));
    animation: skel-pulse var(--skeleton-duration, 1.6s) ease-in-out infinite;
  }

  .skel-wide  { width: 100%; }
  .skel-mid   { width: 70%; }
  .skel-short { width: 45%; }

  @keyframes skel-pulse {
    0%, 100% { opacity: 0.5; }
    50%       { opacity: 1; }
  }

  @media (prefers-reduced-motion: reduce) {
    .skel-row { animation: none; opacity: 0.7; }
  }
</style>
