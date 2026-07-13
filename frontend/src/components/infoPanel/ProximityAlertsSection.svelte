<script>
  import { run } from 'svelte/legacy';

  import { onDestroy } from 'svelte';
  import { proximityAlerts } from '../../lib/stores/places.js';
  import { emitSetProximityAlert, emitRemoveProximityAlert, emitListProximityAlerts } from '../../lib/socket.js';
  import SectionHeader from '../primitives/SectionHeader.svelte';

  // ── F7: Proximity alerts ───────────────────────────────────────────────────
  let proximitySection = $state(false);
  let proximityTargetId = $state('');
  let proximityRadiusM = $state('500');
  let loading = $state(false);
  let loadTimer = $state(null);

  function openProximitySection() {
    proximitySection = true;
    emitListProximityAlerts();
    loading = true;
    clearTimeout(loadTimer);
    loadTimer = setTimeout(() => { loading = false; }, 1500);
  }

  // Data arrived — stop showing the skeleton immediately
  run(() => {
    if (loading && $proximityAlerts.length > 0) {
      loading = false;
      clearTimeout(loadTimer);
    }
  });

  onDestroy(() => clearTimeout(loadTimer));

  function saveProximityAlert() {
    const r = parseInt(proximityRadiusM, 10);
    if (!proximityTargetId || isNaN(r) || r <= 0) return;
    emitSetProximityAlert(proximityTargetId, r);
    proximityTargetId = '';
    proximityRadiusM = '500';
  }
</script>

<!-- ── F7: PROXIMITY ALERTS ───────────────────────────────────────── -->
<div class="feature-section">
  <button class="collapsible-header" onclick={openProximitySection} aria-expanded={proximitySection}>
    <SectionHeader title="Proximity Alerts" level={4}>
      {#snippet action()}
        <svg class="chevron" class:rotated={proximitySection} xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="6 9 12 15 18 9"/></svg>
      {/snippet}
    </SectionHeader>
  </button>
  {#if proximitySection}
    <div class="feature-row feature-row--mt">
      <input
        class="feature-input"
        type="text"
        placeholder="Contact user ID"
        bind:value={proximityTargetId}
        aria-label="Target user ID for proximity alert"
      />
      <input
        class="feature-input feature-input--sm"
        type="number"
        min="50"
        max="50000"
        placeholder="Radius (m)"
        bind:value={proximityRadiusM}
        aria-label="Alert radius in metres"
      />
      <button class="btn btn-primary btn-sm" onclick={saveProximityAlert}>Add</button>
    </div>
    {#if loading && $proximityAlerts.length === 0}
      <!-- Skeleton while existing alerts load over the socket -->
      <div class="skeleton-block" role="status" aria-label="Loading proximity alerts" aria-busy="true">
        <div class="skel-row skel-wide"></div>
        <div class="skel-row skel-mid"></div>
      </div>
    {:else if $proximityAlerts.length > 0}
      <div class="alert-list">
        {#each $proximityAlerts as a}
          <div class="alert-item">
            <span class="alert-name">{a.targetName || a.targetUserId}</span>
            <span class="alert-radius">{a.radiusM}m</span>
            <button class="btn btn-danger btn-xs" onclick={() => emitRemoveProximityAlert(a.targetUserId)} aria-label="Remove proximity alert for {a.targetName || a.targetUserId}">Remove</button>
          </div>
        {/each}
      </div>
    {/if}
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

  .chevron {
    flex-shrink: 0;
    transition: transform 200ms var(--ease-out);
  }
  .chevron.rotated { transform: rotate(180deg); }
  @media (prefers-reduced-motion: reduce) { .chevron { transition: none; } }

  .feature-row--mt { margin-top: var(--space-1-5); }

  .feature-row {
    display: flex;
    gap: 6px;
    align-items: center;
    flex-wrap: wrap;
  }

  .feature-input {
    flex: 1;
    min-width: 80px;
    font-size: var(--text-sm);
    padding: 7px 10px;
    border-radius: var(--radius-md);
    border: 1px solid var(--border-subtle);
    background: var(--surface-3);
    color: var(--text-primary);
  }
  @media (max-width: 767px) {
    .feature-input {
      min-height: 44px;
      font-size: var(--text-base);
    }
  }

  .feature-input--sm {
    flex: 0 0 90px;
  }

  /* F7: proximity alerts */
  .alert-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .alert-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    font-size: var(--text-xs);
  }

  .alert-name {
    flex: 1;
    font-weight: 600;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .alert-radius {
    font-family: var(--font-mono);
    font-size: 10px;
    color: var(--text-tertiary);
    flex-shrink: 0;
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

  .skel-wide { width: 100%; }
  .skel-mid  { width: 70%; }

  @keyframes skel-pulse {
    0%, 100% { opacity: 0.5; }
    50%       { opacity: 1; }
  }

  @media (prefers-reduced-motion: reduce) {
    .skel-row { animation: none; opacity: 0.7; }
  }
</style>
