<script>
  let { user } = $props();

  // Collapse the badge stack to ONE glanceable pill. Priority: SOS > ride >
  // crowd > quiet. Any remaining active states fold into a subtle "+N" hint and
  // stay reachable via the row long-press quick-actions sheet.
  const LABELS = { sos: 'SOS', ride: 'Sharing ride', crowd: 'Group mode', quiet: 'Quiet hours' };

  let badges = $derived((() => {
    const list = [];
    if (user.sos?.active) list.push('sos');
    if (user.rideShare?.active) list.push('ride');
    if (user.crowdMode?.active) list.push('crowd');
    if (user.quietHoursActive) list.push('quiet');
    return list;
  })());
  let primary = $derived(badges[0] ?? null);
  let extra = $derived(Math.max(0, badges.length - 1));
  let extraTitle = $derived(badges.slice(1).map((b) => LABELS[b]).join(', '));
</script>

{#if primary === 'sos'}
  <span class="sos-badge-pill">
    <span class="sos-badge-dot" aria-hidden="true"></span>
    SOS
  </span>
{:else if primary === 'ride'}
  <span class="ride-badge" title="Sharing ride{user.rideShare.vehicle ? ': ' + user.rideShare.vehicle : ''}">
    <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><rect x="1" y="3" width="15" height="13"/><polygon points="16 8 20 8 23 11 23 16 16 16 16 8"/><circle cx="5.5" cy="18.5" r="2.5"/><circle cx="18.5" cy="18.5" r="2.5"/></svg>
    Ride
  </span>
{:else if primary === 'crowd'}
  <span class="crowd-badge" title="Festival mode active">
    <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
    Group
  </span>
{:else if primary === 'quiet'}
  <span class="quiet-badge" title="Quiet Hours — location approximate">
    <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
    Quiet
  </span>
{/if}

{#if extra > 0}
  <span class="more-badge" title={extraTitle} aria-label="{extra} more: {extraTitle}">+{extra}</span>
{/if}

{#if user.statusMessage}
  <span class="status-msg-badge" title="{user.statusMessage}">{user.statusMessage}</span>
{/if}

<style>
  /* SOS badge — high urgency */
  .sos-badge-pill {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-family: var(--font-display);
    font-size: 0.6875rem;
    font-weight: 800;
    color: white;
    background: linear-gradient(135deg, var(--danger-500, #ef4444), var(--danger-700, #b91c1c));
    border-radius: var(--radius-full);
    padding: 2px 8px 2px 5px;
    line-height: 1.3;
    flex-shrink: 0;
    letter-spacing: 0.04em;
    box-shadow:
      0 0 8px color-mix(in srgb, var(--danger-500) 55%, transparent),
      0 0 16px color-mix(in srgb, var(--danger-500) 30%, transparent),
      inset 0 1px 0 color-mix(in srgb, white 20%, transparent);
    animation: sos-urgent-pulse 1s ease-in-out infinite;
  }

  .sos-badge-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: color-mix(in srgb, white 90%, transparent);
    flex-shrink: 0;
  }

  .status-msg-badge {
    display: inline-block;
    font-size: 0.6875rem;
    color: var(--text-secondary);
    font-style: italic;
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    opacity: 0.85;
    flex-shrink: 1;
  }

  /* Ride share badge — color-mix replaces hardcoded rgba */
  .ride-badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 0.625rem;
    font-weight: 700;
    color: var(--primary-400);
    background: color-mix(in srgb, var(--primary-500) 12%, transparent);
    border: 1px solid color-mix(in srgb, var(--primary-500) 22%, transparent);
    border-radius: var(--radius-full);
    padding: 1px 6px 1px 4px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  /* Crowd / Festival mode badge — color-mix replaces hardcoded rgba */
  .crowd-badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 0.625rem;
    font-weight: 700;
    color: var(--warning-500);
    background: color-mix(in srgb, var(--warning-500) 12%, transparent);
    border: 1px solid color-mix(in srgb, var(--warning-500) 22%, transparent);
    border-radius: var(--radius-full);
    padding: 1px 6px 1px 4px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  /* Quiet hours badge — same grammar as ride badge */
  .quiet-badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 0.625rem;
    font-weight: 700;
    color: var(--primary-400);
    background: color-mix(in srgb, var(--primary-500) 12%, transparent);
    border: 1px solid color-mix(in srgb, var(--primary-500) 22%, transparent);
    border-radius: var(--radius-full);
    padding: 1px 6px 1px 4px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  /* "+N" overflow hint — subtle, non-competing with the primary pill */
  .more-badge {
    display: inline-flex;
    align-items: center;
    font-family: var(--font-display);
    font-size: 0.625rem;
    font-weight: 700;
    color: var(--text-tertiary);
    background: var(--surface-inset, rgba(255, 255, 255, 0.05));
    border-radius: var(--radius-full);
    padding: 1px 5px;
    white-space: nowrap;
    flex-shrink: 0;
    font-variant-numeric: tabular-nums;
  }

  @media (prefers-reduced-motion: reduce) {
    .sos-badge-pill { animation: none; }
  }
</style>
