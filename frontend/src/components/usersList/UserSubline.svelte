<script>
  import { myLocation } from '../../lib/stores/map.js';
  import { arrivalProjections } from '../../lib/stores/arrivals.js';
  import { calculateDistance, formatDistance } from '../../lib/tracking.js';
  import { computeActivityStatus } from '../../lib/activityStatus.js';
  import { formatAge } from '../../lib/presence.js';

  let { user } = $props();

  function formatEta(seconds) {
    if (!seconds || seconds <= 0) return null;
    if (seconds < 60) return '< 1 min';
    const m = Math.round(seconds / 60);
    if (m < 60) return `~${m} min`;
    const h = Math.floor(m / 60);
    const rem = m % 60;
    return rem > 0 ? `~${h}h ${rem}m` : `~${h}h`;
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

  // Freshness age for offline state — uses canonical formatAge from presence.js
  // (replaces the local onlineStatus helper; same vocabulary as FreshnessChip).
  let offlineAge = $derived.by(() => {
    if (user.online !== false) return '';
    const ts = user.lastSeen ?? user.lastUpdate;
    if (!ts) return 'Offline';
    const age = Date.now() - ts;
    const ageStr = formatAge(age);
    return ageStr ? `Offline · ${ageStr}` : 'Offline';
  });
</script>

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
    <!-- Offline — freshness age via canonical presence.js formatAge -->
    <span class="offline-label">{offlineAge}</span>
  {/if}
</div>

<style>
  /* Sub-row — distance leads; activity / GPS / ETA demoted to a lighter tier. */
  .user-sub {
    display: flex;
    align-items: baseline;
    gap: 4px;
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    flex-wrap: nowrap;
    overflow: hidden;
  }

  /* Distance — dominant secondary value */
  .distance-label {
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.01em;
    flex-shrink: 0;
  }

  .sep { color: var(--text-tertiary); opacity: 0.6; flex-shrink: 0; }

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

  /* Activity status badge — demoted weight, keeps semantic color */
  .activity-badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: var(--text-xs, 11px);
    font-weight: 500;
  }

  .activity-badge-dot {
    display: inline-block;
    width: 5px;
    height: 5px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  /* GPS accuracy — dot + label (least prominent) */
  .acc-dot {
    display: inline-block;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .acc-label { font-weight: 500; font-size: var(--text-xs); opacity: 0.9; }
  .acc-high  { color: var(--success-500); }
  .acc-high.acc-dot { background: var(--success-500); }
  .acc-good  { color: var(--warning-500); }
  .acc-good.acc-dot { background: var(--warning-500); }
  .acc-low   { color: var(--danger-400); }
  .acc-low.acc-dot { background: var(--danger-400); }

  /* ETA chip */
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
</style>
