<script>
  import { socket } from '../../lib/socket.js';
  import { myGuardianData } from '../../lib/stores/guardians.js';
  import SectionHeader from '../primitives/SectionHeader.svelte';

  function revokeGuardian(wardId, guardianId) {
    if (wardId) socket.emit('revokeGuardian', { wardUserId: wardId });
    else if (guardianId) socket.emit('revokeGuardian', { guardianUserId: guardianId });
  }
</script>

<!-- ── GUARDIAN NETWORK ──────────────────────────────────────────── -->
{#if $myGuardianData.asGuardian.length > 0 || $myGuardianData.asWard.length > 0}
  <div class="network-section">
    <SectionHeader title="Guardians" level={4} />
    {#each $myGuardianData.asGuardian as g}
      <div class="network-item">
        <div class="network-avatar">{(g.wardName || '?')[0].toUpperCase()}</div>
        <div class="network-info">
          <span class="network-name">{g.wardName}</span>
          <span class="network-role">You are their guardian</span>
        </div>
        <span class="network-status-badge" class:active={g.status === 'active'} class:pending={g.status !== 'active'}>{g.status}</span>
        {#if g.status === 'pending' && g.initiatedBy === 'ward'}
          <button class="btn btn-primary btn-sm" onclick={() => socket.emit('approveGuardian', { wardUserId: g.wardId })}>Accept</button>
          <button class="btn btn-danger btn-sm" onclick={() => socket.emit('denyGuardian', { wardUserId: g.wardId })}>Decline</button>
        {:else if g.status === 'active'}
          <button class="btn btn-danger btn-sm" onclick={() => revokeGuardian(g.wardId, null)}>Revoke</button>
        {:else if g.status === 'pending'}
          <button class="btn btn-danger btn-sm" onclick={() => revokeGuardian(g.wardId, null)}>Cancel</button>
        {/if}
      </div>
    {/each}
    {#each $myGuardianData.asWard as g}
      <div class="network-item">
        <div class="network-avatar guardian-av">{(g.guardianName || '?')[0].toUpperCase()}</div>
        <div class="network-info">
          <span class="network-name">{g.guardianName}</span>
          <span class="network-role">Your guardian</span>
        </div>
        <span class="network-status-badge" class:active={g.status === 'active'} class:pending={g.status !== 'active'}>{g.status}</span>
        {#if g.status === 'pending' && g.initiatedBy === 'guardian'}
          <button class="btn btn-primary btn-sm" onclick={() => socket.emit('approveGuardian', { guardianUserId: g.guardianId })}>Accept</button>
          <button class="btn btn-danger btn-sm" onclick={() => socket.emit('denyGuardian', { guardianUserId: g.guardianId })}>Decline</button>
        {:else if g.status === 'pending' && g.initiatedBy === 'ward'}
          <button class="btn btn-danger btn-sm" onclick={() => revokeGuardian(null, g.guardianId)}>Cancel</button>
        {:else if g.status === 'active'}
          <span class="network-caption">Guardian controls this link</span>
        {/if}
      </div>
    {/each}
  </div>
{/if}

<style>
  /* ── Network / Guardian section ──────────────────────────────────── */
  .network-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
</style>
