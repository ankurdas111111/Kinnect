<script>
  /**
   * FamilyPanel — the "Is everyone OK?" intent home.
   *
   * Lives in the mobile Family sheet tab AND the desktop sidebar Family tab.
   * Verdict strip up top (1.5s answer), then the live roster, with Hub +
   * Activity one tap away. Renders in the sheet rather than navigating to
   * /dashboard so the map stays alive behind it (Map.svelte re-init is the
   * most expensive mount in the app).
   *
   * DB load: ZERO — verdict store + WS-fed rosters only.
   */
  import { push } from 'svelte-spa-router';
  import VerdictStrip from './VerdictStrip.svelte';
  import SidebarLinkRow from '../layout/SidebarLinkRow.svelte';
  import UsersList from '../UsersList.svelte';
  import { familyVerdict } from '../../lib/stores/verdict.js';
  import { myRooms } from '../../lib/stores/rooms.js';

  // Hearth 02c: the panel answers first — family name, the day, then the
  // verdict as a sentence. Rooms are this app's families, so the first room
  // name is the family name; falls back to a neutral label when there is none.
  let familyName = $derived($myRooms?.[0]?.name || 'Your family');
  let today = $derived(new Date().toLocaleDateString([], { weekday: 'long' }));
  let nowTime = $derived(new Date().toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' }));

  /** @type {{ onAddPeople?: () => void, onSecretChat?: (peer: any) => void, showVerdict?: boolean }} */
  let { onAddPeople, onSecretChat, showVerdict = true } = $props();
</script>

<div class="family-panel">
  <!-- Hearth 02c/02d: the panel answers first, on mobile AND desktop. This
       replaces the compact VerdictStrip here — it said the same thing in less
       space, and the design leads with the full sentence. `showVerdict` now
       only controls the tappable jump-to-Hub strip for hosts that want it. -->
  <header class="fp-head">
    <p class="fp-family">{familyName}</p>
    <p class="fp-when">{today} · {nowTime}</p>
    <p class="fp-verdict verdict-voice">{$familyVerdict.sentence}</p>
    {#if $familyVerdict.detail}
      <p class="fp-detail">{$familyVerdict.detail}</p>
    {/if}
  </header>
  {#if showVerdict}
    <VerdictStrip onopen={() => push('/dashboard')} compact />
  {/if}

  <SidebarLinkRow links={[
    { label: 'Open Hub', route: '/dashboard', icon: 'hub' },
    { label: 'Activity', route: '/activity', icon: 'activity' },
  ]} />

  <UsersList
    embedded={true}
    on:addPeople={() => onAddPeople?.()}
    on:secretChat={(e) => onSecretChat?.(e.detail)}
  />
</div>

<style>
  .family-panel {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    min-height: 0;
  }

  /* Hearth 02c — the sheet answers first */
  .fp-head { display: flex; flex-direction: column; gap: 2px; }
  .fp-family {
    margin: 0;
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
  }
  .fp-when {
    margin: 0 0 var(--space-2);
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    font-variant-numeric: tabular-nums;
  }
  .fp-verdict {
    margin: 0;
    font-size: var(--text-xl);
    line-height: 1.2;
    color: var(--text-primary);
  }
  .fp-detail {
    margin: var(--space-1) 0 0;
    font-size: var(--text-sm);
    line-height: 1.45;
    color: var(--text-secondary);
  }
</style>
