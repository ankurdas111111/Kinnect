<script>
  import { createEventDispatcher } from 'svelte';
  import { socket, emitIAmSafe } from '../../lib/socket.js';
  import { banner, mySosActive, myLiveLinks } from '../../lib/stores/sos.js';
  import { rideShare } from '../../lib/stores/rideShare.js';
  import { crowdMode } from '../../lib/stores/crowdMode.js';
  import { getShareOrigin } from '../../lib/env.js';

  
  /**
   * @typedef {Object} Props
   * @property {boolean} [showWalkWithMe] - Embedded mode also shows the Walk With Me action.
   */

  /** @type {Props} */
  let { showWalkWithMe = false } = $props();

  const dispatch = createEventDispatcher();

  let onMyWayActive = $state(false);

  function getMedicalSnapshot() {
    try {
      const raw = localStorage.getItem('kinnect_emergency_profile');
      if (!raw) return null;
      const p = JSON.parse(raw);
      const hasData = p.bloodType || p.allergies || p.medications || p.emergencyName;
      return hasData ? p : null;
    } catch { return null; }
  }

  function toggleSOS() {
    if (!socket.connected) {
      banner.set({ type: 'info', text: 'Trying to reconnect... hang tight.', actions: [] });
      setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 3000);
      return;
    }
    if (!$mySosActive) socket.emit('triggerSOS', { reason: 'SOS', medicalCard: getMedicalSnapshot() });
    else socket.emit('cancelSOS');
  }

  function checkIn() {
    socket.emit('checkInAck');
    banner.set({ type: 'info', text: 'Check-in sent — your family knows you\'re okay.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
  }

  // ── F2: I'm Safe ───────────────────────────────────────────────────────────
  function iAmSafe() {
    emitIAmSafe();
    banner.set({ type: 'info', text: 'You broadcast: I\'m safe.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
  }

  function onMyWay() {
    socket.emit('onMyWay', {});
    onMyWayActive = true;

    // Build WhatsApp share message — include live link URL if one exists
    const links = $myLiveLinks;
    let waText = "I'm on my way! 🚀";
    if (links && links.length > 0) {
      const liveUrl = getShareOrigin() + '/#/live/' + links[0].token;
      waText += ` Track me live: ${liveUrl}`;
    }
    window.open('https://wa.me/?text=' + encodeURIComponent(waText), '_blank', 'noopener');

    banner.set({ type: 'info', text: 'Your family knows you\'re on the way.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
  }

  function cancelOnMyWay() {
    socket.emit('cancelOnMyWay');
    onMyWayActive = false;
  }

  function attest() {
    socket.emit('attest');
    banner.set({ type: 'info', text: 'Your location has been verified.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
  }

  function toggleCrowdMode() {
    const next = !$crowdMode.active;
    crowdMode.update(s => ({ ...s, active: next }));
    socket.emit('toggleCrowdMode', { enabled: next, radiusM: $crowdMode.radiusM });
  }
</script>

<!-- ── SAFETY ZONE ───────────────────────────────────────────────── -->
<div class="safety-zone">
  <span class="card-eyebrow safety-eyebrow">Quick Actions</span>
  <div class="safety-actions">
    <button
      class="sos-action-btn"
      class:sos-live={$mySosActive}
      onclick={toggleSOS}
      aria-label={$mySosActive ? 'Cancel SOS' : 'Send emergency SOS alert'}
    >
      <span class="sos-icon-wrap" aria-hidden="true">
        {#if $mySosActive}
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        {:else}
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="13"/><circle cx="12" cy="18" r="1" fill="currentColor" stroke="none"/></svg>
        {/if}
      </span>
      {$mySosActive ? 'Cancel SOS' : 'SOS'}
    </button>
    <button class="ok-action-btn" onclick={checkIn} aria-label="Send scheduled check-in">
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
      I'm OK
    </button>
    <!-- F2: I'm Safe broadcast -->
    <button class="ok-action-btn ok-action-btn--safe" onclick={iAmSafe} aria-label="Broadcast I am safe to everyone who can see you">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
      I'm Safe
    </button>
  </div>
  <div class="safety-actions safety-actions--gap">
    {#if onMyWayActive}
      <button class="ok-action-btn ok-action-btn--active" onclick={cancelOnMyWay} aria-label="Cancel On My Way broadcast">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        Cancel
      </button>
    {:else}
      <button class="ok-action-btn" onclick={onMyWay} aria-label="Broadcast On My Way and share via WhatsApp">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
      On My Way
      </button>
    {/if}
    <button class="ok-action-btn" onclick={attest} aria-label="Confirm your location is genuine" title="Confirm your location is real">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
      Verify Location
    </button>
  </div>
  <div class="safety-actions safety-actions--gap">
    <button class="ok-action-btn" class:ok-action-btn--active={$rideShare.active} onclick={() => dispatch('openRideShare')} aria-label="Share My Ride with family">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="1" y="3" width="15" height="13"/><polygon points="16 8 20 8 23 11 23 16 16 16 16 8"/><circle cx="5.5" cy="18.5" r="2.5"/><circle cx="18.5" cy="18.5" r="2.5"/></svg>
      {$rideShare.active ? 'Ride Active' : 'Share My Ride'}
    </button>
    <button class="ok-action-btn" class:ok-action-btn--crowd={$crowdMode.active} onclick={toggleCrowdMode} aria-label="Toggle Festival / Crowd mode">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
      {$crowdMode.active ? 'Group Active' : 'Stay Together'}
    </button>
    {#if showWalkWithMe}
      <button class="ok-action-btn" onclick={() => dispatch('openWalkWithMe')} aria-label="Walk With Me — virtual escort">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><circle cx="12" cy="5" r="1.5"/><path d="M9 20l1.5-5 2.5 2 2.5-7"/><path d="M6 9h12"/></svg>
        Walk With Me
      </button>
    {/if}
  </div>
</div>

<style>
  /* ── Safety Zone ────────────────────────────────────────────────── */
  .safety-zone {
    background: rgba(239, 68, 68, 0.03);
    border: 1px solid rgba(239, 68, 68, 0.10);
    border-radius: var(--radius-xl);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }
  :global([data-theme="dark"]) .safety-zone {
    background: rgba(239, 68, 68, 0.05);
    border-color: rgba(239, 68, 68, 0.14);
  }
  .safety-eyebrow { color: rgba(239, 68, 68, 0.55); }

  .safety-actions {
    display: grid;
    grid-template-columns: 1fr auto auto;
    gap: var(--space-2);
    align-items: stretch;
  }

  /* SOS primary action */
  .sos-action-btn {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
    padding: var(--space-3) var(--space-4);
    background: linear-gradient(135deg, var(--danger-600), var(--danger-500));
    border: 1px solid rgba(239, 68, 68, 0.40);
    border-top-color: rgba(255, 100, 100, 0.48);
    border-radius: var(--radius-lg);
    color: white;
    font-family: var(--font-display);
    font-weight: 800;
    font-size: var(--text-base);
    letter-spacing: -0.01em;
    cursor: pointer;
    min-height: 52px;
    box-shadow: var(--glow-sos), var(--shadow-md);
    animation: sos-urgent-pulse 2.5s ease-in-out infinite;
    transition: opacity 200ms;
  }
  .sos-action-btn.sos-live {
    background: var(--surface-2);
    color: var(--text-secondary);
    border-color: var(--border-default);
    box-shadow: var(--shadow-sm);
    animation: none;
  }
  .sos-action-btn:hover:not(.sos-live) {
    filter: brightness(1.08);
  }
  .sos-icon-wrap {
    width: 26px;
    height: 26px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.18);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .sos-action-btn.sos-live .sos-icon-wrap {
    background: rgba(239, 68, 68, 0.12);
    color: var(--danger-500);
  }

  /* Secondary row gap modifier — replaces inline margin-top */
  .safety-actions--gap {
    margin-top: var(--space-2);
  }

  /* I'm OK / secondary action buttons */
  .ok-action-btn {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 3px;
    padding: var(--space-2) var(--space-3);
    background: rgba(16, 185, 129, 0.08);
    border: 1px solid rgba(16, 185, 129, 0.18);
    border-radius: var(--radius-lg);
    color: var(--success-500);
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    cursor: pointer;
    min-height: 52px;
    min-width: 60px;
    transition: background 150ms var(--ease-out), box-shadow 150ms var(--ease-out);
  }
  .ok-action-btn:hover {
    background: rgba(16, 185, 129, 0.14);
    box-shadow: 0 0 16px rgba(16, 185, 129, 0.18);
  }
  .ok-action-btn--active {
    background: rgba(239, 68, 68, 0.10);
    border-color: rgba(239, 68, 68, 0.25);
    color: var(--danger-500);
  }
  .ok-action-btn--active:hover {
    background: rgba(239, 68, 68, 0.16);
  }
  .ok-action-btn--crowd {
    background: rgba(245, 158, 11, 0.12);
    border-color: rgba(245, 158, 11, 0.28);
    color: var(--warning-500, #f59e0b);
  }
  .ok-action-btn--crowd:hover {
    background: rgba(245, 158, 11, 0.18);
  }
  /* F2: I'm Safe button — teal/green variant */
  .ok-action-btn--safe {
    background: rgba(6, 182, 212, 0.10);
    border-color: rgba(6, 182, 212, 0.24);
    color: var(--cyan-500, #06b6d4);
  }
  .ok-action-btn--safe:hover {
    background: rgba(6, 182, 212, 0.18);
  }

  @media (prefers-reduced-motion: reduce) {
    .sos-action-btn { animation: none; }
  }
</style>
