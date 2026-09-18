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

<!-- Hearth calm row: one plain sentence about how they're doing.
     Every signal is kept (activity, distance, GPS accuracy, ETA, offline age)
     but spoken as quiet body text instead of a chip stack. -->
<span class="user-sub">
  {#if user.online !== false}
    {#if user.latitude == null || user.longitude == null}
      <!-- Connected but not sharing location -->
      <span class="sub-muted">Location off for now</span>
    {:else}
      {@const actStatus = computeActivityStatus(user)}
      {#if actStatus && actStatus.label !== 'Offline'}
        <span class="sub-lead">{actStatus.label}</span>
      {/if}
      {#if $myLocation}
        {#if actStatus && actStatus.label !== 'Offline'}<span class="sep" aria-hidden="true">·</span>{/if}
        <span class="sub-piece">{formatDistance(calculateDistance($myLocation.latitude, $myLocation.longitude, user.latitude, user.longitude)) || 'Near'} away</span>
      {/if}
      {#if user.accuracy != null}
        <span class="sep" aria-hidden="true">·</span>
        <span class="sub-quiet" aria-label="GPS accuracy: {getAccuracyLabel(user.accuracy)}">GPS {getAccuracyLabel(user.accuracy)}</span>
      {/if}
      {#if user.userId && $arrivalProjections.has(user.userId)}
        {@const proj = $arrivalProjections.get(user.userId)}
        {#if proj?.etaSeconds && proj?.placeName}
          <span class="sep" aria-hidden="true">·</span>
          <span class="sub-eta" title="Heading to {proj.placeName}">{proj.placeName} {formatEta(proj.etaSeconds)}</span>
        {/if}
      {/if}
    {/if}
  {:else}
    <!-- Offline — last-known freshness age, never blank -->
    <span class="sub-muted">{offlineAge}</span>
  {/if}
</span>

<style>
  /* One continuous 16px sentence in the functional register. */
  .user-sub {
    display: inline;
    min-width: 0;
    font-family: var(--font-sans);
    font-size: var(--text-base);
    line-height: var(--leading-normal);
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sub-lead {
    color: var(--text-secondary);
    font-weight: 500;
  }

  .sub-piece { color: var(--text-secondary); }

  .sep {
    color: var(--text-tertiary);
    opacity: 0.7;
    margin: 0 var(--space-0-5);
  }

  .sub-quiet { color: var(--text-tertiary); }

  .sub-muted { color: var(--text-tertiary); }

  /* ETA — a passive schedule cue rides the ochre register (small text uses
     the darker warning step for AA on paper). */
  .sub-eta {
    color: var(--warning-600);
    font-weight: 500;
    white-space: nowrap;
  }
</style>
