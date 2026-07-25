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

  /** @type {{ onAddPeople?: () => void, onSecretChat?: (peer: any) => void, showVerdict?: boolean }} */
  let { onAddPeople, onSecretChat, showVerdict = true } = $props();
</script>

<div class="family-panel">
  {#if showVerdict}
    <VerdictStrip onopen={() => push('/dashboard')} />
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
</style>
