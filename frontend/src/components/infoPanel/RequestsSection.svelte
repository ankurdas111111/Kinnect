<script>
  import { socket } from '../../lib/socket.js';
  import { pendingIncomingRequests } from '../../lib/stores/guardians.js';

  
  /**
   * @typedef {Object} Props
   * @property {boolean} [animated] - Embedded mode animates request cards in with the global stagger utilities.
   */

  /** @type {Props} */
  let { animated = false } = $props();

  let busyRequests = $state(new Set());

  function approveRequest(req, idx) {
    const key = req.type + '-' + req.from;
    if (busyRequests.has(key)) return;
    busyRequests.add(key);
    busyRequests = busyRequests;
    if (req.type === 'roomAdmin') {
      socket.emit('voteRoomAdmin', { roomCode: req.roomCode, userId: req.from, vote: 'approve' });
    } else if (req.type === 'guardianInvite') {
      socket.emit('approveGuardian', { wardUserId: req.from });
    } else {
      socket.emit('approveGuardian', { guardianUserId: req.from });
    }
    pendingIncomingRequests.update(arr => {
      if (req.type === 'roomAdmin') {
        return arr.map((r, i) => i === idx ? { ...r, myVote: 'approve' } : r);
      }
      arr.splice(idx, 1); return [...arr];
    });
    setTimeout(() => { busyRequests.delete(key); busyRequests = busyRequests; }, 5000);
  }

  function denyRequest(req, idx) {
    const key = req.type + '-' + req.from;
    if (busyRequests.has(key)) return;
    busyRequests.add(key);
    busyRequests = busyRequests;
    if (req.type === 'roomAdmin') {
      socket.emit('voteRoomAdmin', { roomCode: req.roomCode, userId: req.from, vote: 'deny' });
    } else if (req.type === 'guardianInvite') {
      socket.emit('denyGuardian', { wardUserId: req.from });
    } else {
      socket.emit('denyGuardian', { guardianUserId: req.from });
    }
    pendingIncomingRequests.update(arr => {
      if (req.type === 'roomAdmin') {
        return arr.map((r, i) => i === idx ? { ...r, myVote: 'deny' } : r);
      }
      arr.splice(idx, 1); return [...arr];
    });
    setTimeout(() => { busyRequests.delete(key); busyRequests = busyRequests; }, 5000);
  }

  function getRequestLabel(req) {
    if (req.type === 'roomAdmin') return `${req.fromName || 'Someone'} wants Admin in ${req.roomCode}`;
    if (req.type === 'guardianInvite') return `${req.fromName || 'Someone'} wants you to be their Guardian`;
    return `${req.fromName || 'Someone'} wants to be your Guardian`;
  }
</script>

<!-- ── PENDING REQUESTS ──────────────────────────────────────────── -->
{#if $pendingIncomingRequests.length > 0}
  <div class="requests-section">
    <span class="card-eyebrow">Requests <span class="req-count">{$pendingIncomingRequests.length}</span></span>
    {#each $pendingIncomingRequests as req, idx}
      <div class="request-card" class:animate-slide-up={animated} class:stagger-item={animated}>
        <p class="req-label">{getRequestLabel(req)}</p>
        {#if req.expiresIn}<span class="req-expiry">{req.expiresIn}</span>{/if}
        {#if req.type === 'roomAdmin'}
          <p class="req-vote-meta">
            {req.approvals || 0} approve · {req.denials || 0} deny · need {Math.floor((req.totalEligible || 1) / 2) + 1} of {req.totalEligible || '?'}
          </p>
          <div class="req-actions">
            {#if req.myVote === 'approve'}
              <span class="badge badge-success badge-xs">You approved</span>
            {:else if req.myVote === 'deny'}
              <span class="badge badge-danger badge-xs">You denied</span>
            {:else}
              <button class="btn btn-primary btn-sm" onclick={() => approveRequest(req, idx)} disabled={busyRequests.has(req.type + '-' + req.from)}>Approve</button>
              <button class="btn btn-danger btn-sm" onclick={() => denyRequest(req, idx)} disabled={busyRequests.has(req.type + '-' + req.from)}>Deny</button>
            {/if}
          </div>
        {:else}
          <div class="req-actions">
            <button class="btn btn-primary btn-sm" onclick={() => approveRequest(req, idx)} disabled={busyRequests.has(req.type + '-' + req.from)}>Approve</button>
            <button class="btn btn-danger btn-sm" onclick={() => denyRequest(req, idx)} disabled={busyRequests.has(req.type + '-' + req.from)}>Deny</button>
          </div>
        {/if}
      </div>
    {/each}
  </div>
{/if}

<style>
  .requests-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  /* ── Request cards ───────────────────────────────────────────────── */
  .request-card {
    padding: var(--space-3) var(--space-4);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-left: 3px solid var(--warning-500);
    border-radius: var(--radius-lg);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
  .req-label {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text-primary);
    margin: 0;
  }
  .req-expiry {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }
  .req-vote-meta {
    font-size: var(--text-xs);
    color: var(--text-secondary);
    margin: 0;
  }
  .req-actions {
    display: flex;
    gap: var(--space-2);
  }
  .req-count {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 18px;
    height: 18px;
    border-radius: var(--radius-full);
    background: var(--warning-500);
    color: white;
    font-size: 10px;
    font-weight: 800;
    padding: 0 4px;
    margin-left: 4px;
  }
</style>
