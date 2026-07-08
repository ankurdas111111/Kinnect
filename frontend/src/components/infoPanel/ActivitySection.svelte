<script>
  import { run } from 'svelte/legacy';

  import { onDestroy } from 'svelte';
  import { authUser } from '../../lib/stores/auth.js';
  import { dailyActivity } from '../../lib/stores/activity.js';
  import { emitGetDailyActivity } from '../../lib/socket.js';

  // ── F9: Daily activity ─────────────────────────────────────────────────────
  let activityOpen = $state(false);
  let loading = $state(false);
  let loadTimer = $state(null);

  let myUserId = $derived($authUser?.userId);
  let myActivityDays = $derived(myUserId ? ($dailyActivity.get(myUserId) || []) : []);
  let hasData = $derived(myUserId ? $dailyActivity.has(myUserId) : false);

  function toggleActivity() {
    activityOpen = !activityOpen;
    if (activityOpen && myUserId) {
      emitGetDailyActivity(myUserId);
      loading = true;
      clearTimeout(loadTimer);
      loadTimer = setTimeout(() => { loading = false; }, 1500);
    }
  }

  // Response arrived — stop showing the skeleton immediately
  run(() => {
    if (loading && hasData) {
      loading = false;
      clearTimeout(loadTimer);
    }
  });

  onDestroy(() => clearTimeout(loadTimer));

  function formatActivityDate(dateStr) {
    if (!dateStr) return '';
    const d = new Date(dateStr + 'T00:00:00');
    return d.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' });
  }
</script>

<!-- ── F9: DAILY ACTIVITY SUMMARY ────────────────────────────────── -->
<div class="feature-section">
  <button class="collapsible-header" onclick={toggleActivity} aria-expanded={activityOpen}>
    <span class="card-eyebrow" style="margin:0">Activity</span>
    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true" style="transform: rotate({activityOpen ? 180 : 0}deg); transition: transform 200ms"><polyline points="6 9 12 15 18 9"/></svg>
  </button>
  {#if activityOpen}
    <div class="activity-list">
      {#if loading && !hasData}
        <!-- Skeleton while the 7-day summary loads over the socket -->
        <div class="skeleton-block" role="status" aria-label="Loading activity summary" aria-busy="true">
          <div class="skel-row skel-wide"></div>
          <div class="skel-row skel-mid"></div>
          <div class="skel-row skel-short"></div>
        </div>
      {:else if myActivityDays.length === 0}
        <p class="empty-hint">No activity data yet.</p>
      {:else}
        {#each myActivityDays as day}
          <div class="activity-row">
            <span class="activity-date">{formatActivityDate(day.date)}</span>
            <span class="activity-stat">{(day.distanceM / 1000).toFixed(1)} km</span>
            <span class="activity-stat">{day.activeMinutes} min</span>
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

  /* F9: activity summary */
  .activity-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 200px;
    overflow-y: auto;
  }

  .activity-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 5px 8px;
    background: var(--surface-inset);
    border-radius: var(--radius-md);
    font-size: var(--text-xs);
  }

  .activity-date {
    flex: 1;
    font-weight: 600;
    color: var(--text-primary);
  }

  .activity-stat {
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--primary-400);
    font-weight: 700;
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
