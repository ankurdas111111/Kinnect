<script>
  import { run } from 'svelte/legacy';

  import { createEventDispatcher } from 'svelte';
  import { walkDestination } from '../lib/stores/map.js';
  import ShareMyRide from './ShareMyRide.svelte';
  import WalkWithMe from './WalkWithMe.svelte';
  import Modal from './primitives/Modal.svelte';
  import GpsStatusCard from './infoPanel/GpsStatusCard.svelte';
  import IdentityCard from './infoPanel/IdentityCard.svelte';
  import SafetyActions from './infoPanel/SafetyActions.svelte';
  import SpeedAlertSection from './infoPanel/SpeedAlertSection.svelte';
  import GeofenceLogSection from './infoPanel/GeofenceLogSection.svelte';
  import ProximityAlertsSection from './infoPanel/ProximityAlertsSection.svelte';
  import ActivitySection from './infoPanel/ActivitySection.svelte';
  import StatusMessageSection from './infoPanel/StatusMessageSection.svelte';
  import MeetingPointsSection from './infoPanel/MeetingPointsSection.svelte';
  import BulletinBoardSection from './infoPanel/BulletinBoardSection.svelte';
  import GuardianSection from './infoPanel/GuardianSection.svelte';
  import RequestsSection from './infoPanel/RequestsSection.svelte';

  /**
   * @typedef {Object} Props
   * @property {boolean} [embedded]
   */

  /** @type {Props} */
  let { embedded = false } = $props();

  let rideShareOpen = $state(false);
  let walkWithMeOpen = $state(false);

  // Auto-open Walk With Me when PlaceSearch sets a destination
  run(() => {
    if ($walkDestination) { walkWithMeOpen = true; }
  });

  const dispatch = createEventDispatcher();
</script>

<!-- ═══════════════════════════════════════════════════════════════════════
     EMBEDDED MODE (used in sidebar + bottom sheet)
     ═══════════════════════════════════════════════════════════════════════ -->
{#if embedded}
  <div class="panel-body info-root">
    <GpsStatusCard />
    <IdentityCard />
    <SafetyActions
      showWalkWithMe
      on:openRideShare={() => rideShareOpen = true}
      on:openWalkWithMe={() => walkWithMeOpen = true}
    />
    <SpeedAlertSection />
    <GeofenceLogSection />
    <ProximityAlertsSection />
    <ActivitySection />
    <StatusMessageSection />
    <MeetingPointsSection />
    <BulletinBoardSection />
    <GuardianSection />
    <RequestsSection animated />
  </div>

<!-- ═══════════════════════════════════════════════════════════════════════
     PANEL SHELL MODE (desktop standalone panel)
     ═══════════════════════════════════════════════════════════════════════ -->
{:else}
  <div class="panel-shell panel-left panel-base">
    <div class="panel-header">
      <h3>You</h3>
      <button class="btn btn-icon btn-ghost" aria-label="Close info panel" onclick={() => dispatch('close')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>

    <div class="panel-body info-root">
      <GpsStatusCard shell />
      <IdentityCard />
      <SafetyActions on:openRideShare={() => rideShareOpen = true} />
      <SpeedAlertSection />
      <GeofenceLogSection />
      <ProximityAlertsSection />
      <ActivitySection />
      <StatusMessageSection />
      <MeetingPointsSection />
      <BulletinBoardSection />
      <GuardianSection />
      <RequestsSection />
    </div>
  </div>
{/if}

<ShareMyRide bind:open={rideShareOpen} />

<!-- Walk With Me — Modal primitive gives focus-trap + ESC for free -->
<Modal
  open={walkWithMeOpen}
  title="Walk With Me"
  size="sm"
  on:close={() => walkWithMeOpen = false}
>
  {#snippet children()}
    <WalkWithMe on:close={() => walkWithMeOpen = false} />
  {/snippet}
</Modal>

<style>
  /* ── Root container ─────────────────────────────────────────────── */
  .info-root {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }
</style>
