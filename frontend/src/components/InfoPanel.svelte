<script>
  import { run } from 'svelte/legacy';

  import { createEventDispatcher } from 'svelte';
  import { walkDestination } from '../lib/stores/map.js';
  import ShareMyRide from './ShareMyRide.svelte';
  import WalkWithMe from './WalkWithMe.svelte';
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
{#if walkWithMeOpen}
  <div class="wwm-overlay">
    <div class="wwm-modal">
      <button class="wwm-close" onclick={() => walkWithMeOpen = false} aria-label="Close">
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
      <WalkWithMe on:close={() => walkWithMeOpen = false} />
    </div>
  </div>
{/if}

<style>
  /* ── Root container ─────────────────────────────────────────────── */
  .info-root {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  /* Walk With Me modal */
  .wwm-overlay {
    /* Was z:60 — dangerously low, buried under panels/sheets.
       Modal overlays must be at --z-modal tier. */
    position: fixed; inset: 0; z-index: var(--z-modal, 5000);
    background: rgba(0,0,0,0.55);
    display: flex; align-items: center; justify-content: center;
    animation: wwm-fade-in 0.2s ease;
  }
  @keyframes wwm-fade-in { from { opacity: 0; } }
  @media (prefers-reduced-motion: reduce) { .wwm-overlay { animation: none; } }
  .wwm-modal {
    position: relative;
    width: min(380px, 92vw);
    max-height: 85vh;
    overflow-y: auto;
    background: var(--surface-0, #0a0e1a);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 20px;
    box-shadow: 0 16px 48px rgba(0,0,0,0.5);
  }
  .wwm-close {
    position: absolute; top: 12px; right: 12px; z-index: 2;
    width: 44px; height: 44px; border-radius: 50%;
    background: rgba(255,255,255,0.06); border: none;
    display: flex; align-items: center; justify-content: center;
    color: rgba(255,255,255,0.4); cursor: pointer;
    -webkit-tap-highlight-color: transparent;
  }
  .wwm-close:hover { background: rgba(255,255,255,0.12); }
</style>
