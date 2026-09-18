<script>
  /**
   * RoomCard — presentational room tile (CONTRACTS.md §1c card tier).
   * Renders room metadata, member chips, admin controls and pending
   * admin-vote rows. Owns NO socket/emit logic — every mutation is a
   * callback prop; the parent keeps the emits and store writes.
   *
   * getUserColor(room.code) is sampled ONCE here per render (not per style
   * recalc) via a $derived, per the SharingPanel critique verdict.
   */
  import { stopPropagation } from 'svelte/legacy';
  import Card from '../primitives/Card.svelte';
  import { getUserColor } from '../../lib/getUserColor.js';

  /**
   * @type {{
   *   room: any,
   *   myUserId?: string | null,
   *   leaving?: boolean,
   *   adminDuration?: string | null,
   *   onlocate?: (userId: string) => void,
   *   onleave?: () => void,
   *   onrequestadmin?: () => void,
   *   onrevokeadmin?: (userId: string) => void,
   *   onvote?: (fromUserId: string, vote: 'approve' | 'deny') => void,
   * }}
   */
  let {
    room,
    myUserId = null,
    leaving = false,
    adminDuration = $bindable(null),
    onlocate,
    onleave,
    onrequestadmin,
    onrevokeadmin,
    onvote,
  } = $props();

  const accent = $derived(getUserColor(room.code));
  const isAdmin = $derived(room.myRoomRole === 'admin');
  const members = $derived(room.members || []);
  const pending = $derived(room.pendingAdminRequests || []);
  const hasPendingMine = $derived(pending.some((r) => r.isMe));
</script>

<Card variant="glass" padding="none">
  <div class="room-card animate-slide-up" style="--room-accent: {accent};">
    <div class="room-head">
      <div class="room-icon">{(room.name || 'G')[0].toUpperCase()}</div>
      <div class="room-meta">
        <span class="room-name">
          {room.name}
          {#if isAdmin}<span class="badge badge-success badge-xs room-name-badge">Admin</span>{/if}
        </span>
        <span class="room-code">{room.code}</span>
      </div>
      <button
        class="quiet-leave-btn tactile"
        onclick={() => onleave?.()}
        disabled={leaving}
      >{leaving ? 'Leaving…' : 'Leave group'}</button>
    </div>

    {#if members.length > 0}
      <div class="room-members">
        {#each members as m}
          <button
            class="member-chip"
            onclick={() => onlocate?.(m.userId)}
            title="Find {m.displayName || 'member'} on map"
          >
            <span class="member-avatar">{(m.displayName || '?')[0].toUpperCase()}</span>
            {m.displayName || 'Member'}
            {#if m.roomRole === 'admin'}<span class="badge badge-success badge-xs">A</span>{/if}
            {#if isAdmin && m.userId !== myUserId && m.roomRole === 'admin'}
              <button
                class="revoke-inline"
                aria-label="Revoke admin from {m.displayName || 'member'}"
                onclick={stopPropagation(() => onrevokeadmin?.(m.userId))}
              >×</button>
            {/if}
          </button>
        {/each}
      </div>
    {/if}

    <div class="room-actions">
      {#if !isAdmin && !hasPendingMine}
        <select class="duration-select" bind:value={adminDuration} aria-label="Admin duration">
          <option value={null}>Permanent</option>
          <option value="1h">1 Hour</option>
          <option value="6h">6 Hours</option>
          <option value="24h">24 Hours</option>
          <option value="7d">7 Days</option>
          <option value="30d">30 Days</option>
        </select>
        <button class="btn btn-secondary btn-sm tactile" onclick={() => onrequestadmin?.()}>Request Admin</button>
      {:else if hasPendingMine}
        <span class="badge badge-warning badge-xs">Admin Requested</span>
      {/if}
    </div>

    {#if pending.length > 0}
      <div class="pending-admin">
        {#each pending as par}
          <div class="pending-row">
            <div class="pending-info">
              <span class="text-sm">{par.isMe ? 'Your admin request' : `${par.fromName} wants Admin`}</span>
              {#if par.expiresIn}<span class="mini"> ({par.expiresIn})</span>{/if}
              <span class="mini vote-count">{par.approvals}/{par.totalEligible} approve, {par.denials}/{par.totalEligible} deny (need {Math.floor(par.totalEligible / 2) + 1})</span>
            </div>
            {#if !par.isMe}
              <div class="pending-actions">
                {#if par.myVote === 'approve'}
                  <span class="badge badge-success badge-xs">Approved</span>
                {:else if par.myVote === 'deny'}
                  <span class="badge badge-denied badge-xs">Denied</span>
                {:else}
                  <button class="btn btn-primary btn-xs tactile" onclick={() => onvote?.(par.from, 'approve')}>Approve</button>
                  <button class="quiet-deny-btn tactile" onclick={() => onvote?.(par.from, 'deny')}>Deny</button>
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
</Card>

<style>
  /* ═══ Hearth: quiet paper row — tonal separation, no accent stripe ═══ */
  .room-card {
    padding: var(--space-3);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    border-radius: inherit;
  }

  .room-head {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
  }
  /* The room's wheel colour lives in its pebble, not a border stripe */
  .room-icon {
    width: 36px;
    height: 36px;
    border-radius: var(--radius-full);
    background: var(--room-accent, var(--primary-500));
    color: var(--text-inverse);
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-sans);
    font-size: var(--text-base);
    font-weight: 700;
    flex-shrink: 0;
  }
  .room-meta {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .room-name {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    display: flex;
    align-items: center;
    gap: var(--space-1);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .room-name-badge { margin-left: var(--space-1-5); }
  .room-code {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    letter-spacing: 0.05em;
  }

  .room-members {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
    padding-top: var(--space-2);
  }
  .member-chip {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    background: var(--surface-2);
    border: none;
    border-radius: var(--radius-full);
    padding: 3px 10px;
    min-height: 32px;
    font-family: var(--font-sans);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-secondary);
    cursor: pointer;
    transition: background-color var(--duration-fast) var(--ease-out),
                color var(--duration-fast) var(--ease-out);
  }
  .member-chip:hover { background: var(--surface-hover); color: var(--text-primary); }
  .member-chip:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .member-avatar {
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: var(--primary-100);
    color: var(--primary-700);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-sans);
    font-size: 9px;
    font-weight: 700;
  }
  /* Revoke is administrative, not an emergency — ink, with explicit hover */
  .revoke-inline {
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-tertiary);
    font-size: 14px;
    font-weight: 700;
    padding: 0 4px;
    line-height: 1;
    border-radius: var(--radius-sm);
    transition: color var(--duration-fast) var(--ease-out);
  }
  .revoke-inline:hover { color: var(--text-primary); }
  .revoke-inline:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 1px; }

  .room-actions {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    flex-wrap: wrap;
    padding-top: var(--space-1);
  }
  .duration-select {
    font-size: var(--text-sm);
    padding: 2px 6px;
    border: 1px solid transparent;
    border-radius: var(--radius-input);
    background: var(--surface-inset);
    color: var(--text-primary);
    cursor: pointer;
    max-width: 110px;
    min-height: 44px;
    transition: border-color var(--duration-fast) var(--ease-out);
  }
  .duration-select:focus-visible {
    outline: none;
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px var(--primary-500-20);
  }

  /* Leaving a group is a decision, not a danger — ink outline, plain words */
  .quiet-leave-btn {
    padding: var(--space-2) var(--space-3);
    min-height: 44px;
    font-family: var(--font-sans);
    font-size: var(--text-sm);
    font-weight: 600;
    background: transparent;
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-md);
    color: var(--text-primary);
    cursor: pointer;
    flex-shrink: 0;
    white-space: nowrap;
    transition: background-color var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }
  .quiet-leave-btn:hover:not(:disabled) { background: var(--surface-hover); }
  .quiet-leave-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .quiet-leave-btn:disabled { opacity: 0.5; cursor: default; }

  /* Pending votes sit on their own paper tier — no dashed rule */
  .pending-admin {
    width: 100%;
    margin-top: var(--space-2);
    padding: var(--space-2) var(--space-3);
    background: var(--surface-2);
    border-radius: var(--radius-md);
  }
  .pending-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    padding: var(--space-1) 0;
  }
  .pending-info { display: flex; flex-direction: column; gap: 1px; }
  .pending-actions { display: flex; gap: var(--space-1); flex-shrink: 0; }
  .vote-count { color: var(--text-secondary); }
  /* A denied vote is a fact, not an alarm */
  .badge-denied {
    background: var(--surface-inset);
    color: var(--text-secondary);
    border-radius: var(--radius-full);
  }
  .quiet-deny-btn {
    font-family: var(--font-sans);
    font-size: var(--text-2xs);
    font-weight: 600;
    padding: 2px 10px;
    min-height: 44px;
    background: transparent;
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-md);
    color: var(--text-primary);
    cursor: pointer;
    transition: background-color var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }
  .quiet-deny-btn:hover { background: var(--surface-hover); }
  .quiet-deny-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .btn-xs {
    font-size: var(--text-2xs);
    padding: 2px 8px;
    border-radius: var(--radius-sm);
  }

  @media (prefers-reduced-motion: reduce) {
    .member-chip,
    .revoke-inline,
    .duration-select,
    .quiet-leave-btn,
    .quiet-deny-btn { transition: none; }
  }
</style>
