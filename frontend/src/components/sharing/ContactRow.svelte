<script>
  /**
   * ContactRow — presentational contact row (CONTRACTS.md §4 AvatarRing).
   * One status grammar: AvatarRing (live/sos/offline) + presence text +
   * optional activity chip — color + shape + text, never color alone.
   *
   * Owns NO socket/emit logic. Guardian/watch mutations are callback props;
   * the parent keeps the emits, store reads and pending/active resolution.
   */
  import AvatarRing from '../primitives/AvatarRing.svelte';
  import { getUserColor } from '../../lib/getUserColor.js';
  import Card from '../primitives/Card.svelte';

  /**
   * @type {{
   *   contact: any,
   *   presenceState?: 'now' | 'recent' | 'gone' | 'sos',
   *   presenceText?: string,
   *   activityContext?: string,
   *   batteryLow?: boolean,
   *   isGuardian?: boolean,
   *   isWard?: boolean,
   *   isPending?: boolean,
   *   removing?: boolean,
   *   guardianDuration?: string | null,
   *   onlocate?: (userId: string) => void,
   *   onremove?: (userId: string) => void,
   *   onwatch?: (userId: string) => void,
   *   onbewatched?: (userId: string) => void,
   * }}
   */
  let {
    contact,
    presenceState = 'gone',
    presenceText = '',
    activityContext = '',
    batteryLow = false,
    isGuardian = false,
    isWard = false,
    isPending = false,
    removing = false,
    guardianDuration = $bindable(null),
    onlocate,
    onremove,
    onwatch,
    onbewatched,
  } = $props();

  // Presence → ring grammar. sos wins; live for fresh; offline otherwise.
  const ring = $derived(
    presenceState === 'sos' ? 'sos' : presenceState === 'now' ? 'live' : 'offline'
  );
  const accent = $derived(getUserColor(contact.userId));

  function initials(name) {
    if (!name) return '?';
    return name.split(' ').filter(Boolean).map((n) => n[0]).join('').toUpperCase().slice(0, 2);
  }

  const canRelate = $derived(!isGuardian && !isWard && !isPending);
</script>

<Card variant="glass" glow="primary" padding="none">
  <div class="contact-row animate-slide-up">
    <div class="avatar-wrap">
      <AvatarRing {ring} size={44} label={`${contact.displayName}, ${presenceText}`}>
        <span class="avatar-circle" class:battery-low={batteryLow} style="--person-color: {accent};">
          {initials(contact.displayName)}
        </span>
      </AvatarRing>
    </div>

    <div class="name-block">
      <span class="name">
        {contact.displayName}
        {#if isGuardian}<span class="badge badge-primary badge-xs rel-badge">You watch</span>{/if}
        {#if isWard}<span class="badge badge-primary badge-xs rel-badge">Watches you</span>{/if}
        {#if isPending}<span class="badge badge-warning badge-xs rel-badge">Pending</span>{/if}
      </span>
      <span class="status" class:status-now={presenceState === 'now'} class:status-sos={presenceState === 'sos'}>{presenceText}</span>
      {#if activityContext}<span class="activity-chip">{activityContext}</span>{/if}
    </div>

    <div class="actions">
      <button
        class="locate-pill"
        onclick={() => onlocate?.(contact.userId)}
        title="Find {contact.displayName} on map"
        aria-label="Locate {contact.displayName}"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="10" r="3"/><path d="M12 2a8 8 0 0 0-8 8c0 1.892.402 3.13 1.5 4.5L12 22l6.5-7.5c1.098-1.37 1.5-2.608 1.5-4.5a8 8 0 0 0-8-8z"/></svg>
      </button>
      <button
        class="btn btn-danger btn-sm tactile"
        onclick={() => onremove?.(contact.userId)}
        disabled={removing}
      >{removing ? 'Removing…' : 'Remove'}</button>
      {#if canRelate}
        <select class="duration-select" bind:value={guardianDuration} aria-label="Watch duration">
          <option value={null}>Permanent</option>
          <option value="1h">1h</option>
          <option value="24h">24h</option>
          <option value="7d">7d</option>
          <option value="30d">30d</option>
        </select>
        <button class="btn btn-secondary btn-sm tactile" onclick={() => onwatch?.(contact.userId)} title="You watch them">Watch</button>
        <button class="btn btn-secondary btn-sm tactile" onclick={() => onbewatched?.(contact.userId)} title="They watch you">Be Watched</button>
      {/if}
    </div>
  </div>
</Card>

<style>
  .contact-row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3);
    border-radius: inherit;
  }
  .avatar-wrap { flex-shrink: 0; }

  .avatar-circle {
    width: 100%;
    height: 100%;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-on-primary);
    background: var(--person-color, var(--primary-500));
  }
  .avatar-circle.battery-low { filter: saturate(0.6); }

  .name-block {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .name {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .rel-badge { margin-left: var(--space-1); }
  .status {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .status.status-now { color: var(--success-500); font-weight: 600; }
  .status.status-sos { color: var(--danger-500); font-weight: 700; }
  .activity-chip {
    font-size: var(--text-2xs);
    color: var(--text-tertiary);
    background: var(--glass-chip-bg, var(--surface-inset));
    border: 1px solid var(--glass-chip-border, var(--border-subtle));
    border-radius: var(--radius-full);
    padding: 1px 7px;
    width: fit-content;
    margin-top: 2px;
  }

  .actions {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: var(--space-1);
    flex-shrink: 0;
  }
  .locate-pill {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border-radius: var(--radius-full);
    background: var(--primary-500-08);
    border: 1px solid var(--primary-500-20);
    color: var(--primary-400);
    cursor: pointer;
    flex-shrink: 0;
    transition: background var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }
  .locate-pill:hover { background: var(--primary-500-12); }
  .locate-pill:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .duration-select {
    font-size: var(--text-xs);
    padding: 2px 6px;
    border: 1px solid var(--border-default);
    border-radius: var(--radius-sm);
    background: var(--surface-1);
    color: var(--text-primary);
    cursor: pointer;
    max-width: 90px;
    min-height: 44px;
  }
</style>
