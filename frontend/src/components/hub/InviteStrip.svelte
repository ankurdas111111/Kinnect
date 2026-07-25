<script>
  /**
   * InviteStrip — "empty seats" caring invite loop.
   *
   * Renders the circle as filled avatars plus a few ghost "+ seat" bubbles for
   * the gap up to a small target. Tapping a seat opens the existing InviteSheet
   * (WhatsApp / copy link / share code). Only shown when you have SOME family
   * but fewer than the target — the empty (0) case is handled by the roster's
   * GhostConstellation.
   *
   * DB load: ZERO — pure client compute + the device share sheet.
   */
  import { getUserColor } from '../../lib/getUserColor.js';
  import AvatarRing from '../primitives/AvatarRing.svelte';
  import InviteSheet from '../InviteSheet.svelte';

  /** @type {{ members: Array<any>, target?: number }} */
  let { members = [], target = 4 } = $props();

  let inviteOpen = $state(false);
  let seats = $derived(Math.min(3, Math.max(0, target - members.length)));
  let show = $derived(members.length > 0 && seats > 0);

  function initials(n) { return (n || '?').split(' ').map((s) => s[0]).join('').toUpperCase().slice(0, 2) || '?'; }
</script>

{#if show}
  <section class="invite" aria-label="Invite family">
    <div class="invite-row">
      {#each members.slice(0, 4) as m (m.socketId)}
        {@const color = getUserColor(m.userId)}
        <span class="seat" style="--mc:{color}">
          <AvatarRing ring={m.online ? 'live' : 'offline'} size={40}>
            <span class="seat-init">{initials(m.displayName)}</span>
          </AvatarRing>
        </span>
      {/each}
      {#each Array(seats) as _, i (i)}
        <button class="ghost-seat" onclick={() => (inviteOpen = true)} aria-label="Invite someone to your circle">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        </button>
      {/each}
    </div>
    <button class="invite-cta" onclick={() => (inviteOpen = true)}>
      Invite someone to your circle
    </button>
  </section>

  <InviteSheet bind:open={inviteOpen} />
{/if}

<style>
  .invite { display: flex; flex-direction: column; gap: var(--space-2); }
  .invite-row { display: flex; align-items: center; gap: var(--space-2); flex-wrap: wrap; }
  .seat { display: inline-flex; }
  .seat-init { font-size: 13px; font-weight: 800; color: var(--mc, var(--primary-400)); line-height: 1; user-select: none; }

  .ghost-seat {
    width: 44px; height: 44px; border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    background: transparent;
    border: 1.5px dashed var(--border-strong);
    color: var(--text-tertiary); cursor: pointer;
    transition: border-color var(--duration-fast, 150ms) var(--ease-out), color var(--duration-fast, 150ms) var(--ease-out), transform var(--duration-fast, 120ms) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .ghost-seat:hover { border-color: var(--primary-400); color: var(--primary-300); }
  .ghost-seat:active { transform: scale(0.94); }
  .ghost-seat:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }

  .invite-cta {
    align-self: flex-start; min-height: 44px; padding: 0 var(--space-3);
    background: none; border: none; cursor: pointer;
    color: var(--primary-300); font-size: var(--text-xs, 12px); font-weight: 600;
    -webkit-tap-highlight-color: transparent;
  }
  .invite-cta:hover { color: var(--primary-200); text-decoration: underline; }
  .invite-cta:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; border-radius: var(--radius-sm, 6px); }

  @media (prefers-reduced-motion: reduce) {
    .ghost-seat:active { transform: none; }
  }
</style>
