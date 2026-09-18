<script>
  import { run } from 'svelte/legacy';

  import { alertState, sosNarratives, activeSosUsers, geofenceShake } from '../lib/stores/sos.js';
  import Modal from './primitives/Modal.svelte';
  import { haptics } from '../lib/haptics.js';
  import { myLocation, focusUser } from '../lib/stores/map.js';
  import { calculateDistance } from '../lib/tracking.js';
  import { startCall } from '../lib/webrtc.js';
  import { push as navigate } from 'svelte-spa-router';



  // Expand/collapse medical card
  let medCardOpen = $state(false);

  let audioCtx   = null;
  let oscillator = null;
  let shaking    = $state(false);


  // Feature 8: geofence breach shake — camera shake without audio alarm
  let _prevGeofenceShake = $state(0);

  function triggerHaptic() { haptics.sos(); }

  function triggerShake() {
    const prefersReduced = typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (prefersReduced) return;
    shaking = true;
    setTimeout(() => { shaking = false; }, 500);
  }

  function startAlarm() {
    const prefersReduced = typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (prefersReduced) return;
    try {
      audioCtx = new (window.AudioContext || window.webkitAudioContext)();
      const alarmMs = $alertState.alarmMs;
      const ready = audioCtx.state === 'suspended' ? audioCtx.resume() : Promise.resolve();
      ready.then(() => {
        oscillator = audioCtx.createOscillator();
        oscillator.type = 'sawtooth';
        oscillator.frequency.setValueAtTime(880, audioCtx.currentTime);
        oscillator.connect(audioCtx.destination);
        oscillator.start();
        setTimeout(() => stopAlarm(), alarmMs);
      }).catch(() => {});
    } catch (_) {}
  }

  function stopAlarm() {
    try {
      if (oscillator) { oscillator.stop(); oscillator = null; }
      if (audioCtx)   { audioCtx.close();  audioCtx = null;  }
    } catch (_) {}
  }

  function dismiss() {
    stopAlarm();
    alertState.set({ visible: false, title: '', body: '', actions: [], alarmMs: 0 });
  }

  function toggleMedCard() { medCardOpen = !medCardOpen; }

  // ── Derive full narrative data (narrative + optional medicalCard) ─────────
  // sosNarratives stores the full payload: { sosToken, userId, narrative, medicalCard? }
  let activeSosData = $derived((() => {
    for (const [userId, sos] of $activeSosUsers) {
      const n = $sosNarratives.get(userId);
      if (n) return n;
      if (sos.sos?.narrative) return { narrative: sos.sos.narrative };
    }
    return null;
  })());
  let activeNarrative  = $derived(activeSosData?.narrative   || null);
  let activeMedicalCard = $derived(activeSosData?.medicalCard || null);

  // ── Who and where ────────────────────────────────────────────────────────
  // The person raising the SOS, straight off the alert payload. Reading it
  // here rather than from the local users map matters: a recipient opening the
  // app cold from a push notification has no local map yet.
  let sosPerson = $derived((() => {
    for (const [, s] of $activeSosUsers) {
      if (s?.sos?.active) return s;
    }
    return null;
  })());

  // First name only: "Call Claude QA" wrapped to two lines in the action row.
  let firstName = $derived((sosPerson?.displayName || '').split(' ')[0] || 'them');

  // Pebble initials — the design renders people as pebbles, never pins.
  let initials = $derived((() => {
    const parts = (sosPerson?.displayName || '').trim().split(/\s+/).filter(Boolean);
    return ((parts[0]?.[0] || '') + (parts[1]?.[0] || '')).toUpperCase() || '!';
  })());

  let eyebrowLabel = $derived(
    sosPerson?.sos?.type === 'geofence' ? 'Safe zone alert'
      : sosPerson ? 'SOS alert'
      : 'Safety alert'
  );

  // Distance from me to them. Null when either side has no fix — an unknown
  // distance must read as unknown, never as zero.
  let distanceKm = $derived((() => {
    const me = $myLocation;
    if (!me || sosPerson?.latitude == null || sosPerson?.longitude == null) return null;
    return calculateDistance(me.latitude, me.longitude, sosPerson.latitude, sosPerson.longitude) / 1000;
  })());

  let distanceText = $derived(
    distanceKm == null ? null
      : distanceKm < 1 ? `${Math.round(distanceKm * 1000)} m from you`
      : `${distanceKm.toFixed(1)} km from you`
  );

  // The headline is whichever fact is most useful to someone about to move:
  // a place name ("Near School") if we have one, otherwise the distance, which
  // is the actionable number. "Somewhere on the map" said nothing and buried
  // the distance in the subline.
  let whereHeadline = $derived(
    sosPerson?.locationLabel || distanceText || (sosPerson?.latitude != null ? 'On the map' : null)
  );
  // Don't repeat the headline underneath it.
  let whereMeta = $derived(
    whereHeadline && whereHeadline !== distanceText ? distanceText : null
  );

  let fixAge = $derived((() => {
    const ts = sosPerson?.lastUpdate;
    if (!ts) return null;
    const s = Math.max(0, Math.round((Date.now() - ts) / 1000));
    if (s < 45) return 'just now';
    if (s < 3600) return `${Math.round(s / 60)} min ago`;
    return `${Math.round(s / 3600)} h ago`;
  })());

  function seeOnMap() {
    if (!sosPerson) return;
    focusUser.set(sosPerson.socketId || '__self__');
    dismiss();
    navigate('/');
  }

  function callThem() {
    if (!sosPerson?.userId) return;
    haptics.tap?.();
    startCall(sosPerson.userId, sosPerson.displayName || 'Contact');
    dismiss();
  }

  // Medical detail is for the person who has already arrived, or for a
  // responder they hand the phone to. It is not the first thing a parent
  // needs, so it opens on demand rather than on arrival.
  run(() => {
    if ($alertState.visible && $alertState.alarmMs > 0) {
      startAlarm();
      triggerHaptic();
      triggerShake();
    } else {
      stopAlarm();
    }
  });
  run(() => {
    if ($geofenceShake > _prevGeofenceShake) {
      _prevGeofenceShake = $geofenceShake;
      triggerShake();
      haptics.warning?.();
    }
  });
</script>

<!-- Camera shake wrapper -->
<div class="alert-shake-wrapper" class:shaking>
  <Modal open={$alertState.visible} urgent={true} title={$alertState.title || 'Alert'} on:close={dismiss}>
    <div class="alert-body-wrap">

      <!-- Eyebrow + verdict headline. The Modal's own title is visually
           clipped (it stays in the DOM as the dialog's accessible name);
           this serif sentence is the reference's headline voice. -->
      <div class="sos-eyebrow">
        <span class="sos-eyebrow-dot" aria-hidden="true"></span>
        <span class="sos-eyebrow-label">{eyebrowLabel}</span>
      </div>
      <p class="sos-headline">{$alertState.title || 'Alert'}</p>

      {#if $alertState.body}
        <p class="alert-body">{$alertState.body}</p>
      {/if}

      <!-- ── WHO — the pebble of the person who raised the alarm. ────────── -->
      {#if sosPerson}
        <section class="sos-who" aria-label="Who raised the alarm">
          <div class="sos-pebble-halo" aria-hidden="true">
            <div class="sos-pebble">{initials}</div>
            <span class="sos-pebble-mark">!</span>
          </div>
          <p class="sos-who-name">{sosPerson.displayName || 'Someone'}</p>
        </section>
      {/if}

      <!-- Critical telemetry off the narrative payload -->
      {#if activeNarrative}
        <div class="sos-telemetry" role="group" aria-label="Latest signals">
          {#if activeNarrative.motionSummary}
            <span class="sig-chip sig-motion">{activeNarrative.motionSummary}</span>
          {/if}
          {#if activeNarrative.batteryPct != null}
            <span class="sig-chip sig-battery">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="2" y="7" width="16" height="10" rx="2"/><line x1="22" y1="10" x2="22" y2="14"/></svg>
              Battery {activeNarrative.batteryPct}%
            </span>
          {/if}
          {#if activeNarrative.triggerRule && activeNarrative.triggerRule !== 'manual'}
            <span class="sig-chip sig-trigger">Auto: {activeNarrative.triggerRule}</span>
          {/if}
        </div>
      {/if}

      <!-- ── WHERE — above the medical card, because a parent 2 km away needs
           a direction to drive before they need a blood type. ─────────────── -->
      {#if sosPerson}
        <div class="sos-where" role="group" aria-label="Location">
          <span class="sos-kicker">Last known position</span>
          {#if whereHeadline}
            <p class="sos-where-place">{whereHeadline}</p>
            <p class="sos-where-meta">
              {#if whereMeta}
                <span class="sos-where-dist">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2 4.09 20.42l.7.7L12 17.9l7.21 3.22.7-.7z"/></svg>
                  {whereMeta}
                </span>
              {/if}
              {#if whereMeta && fixAge}<span aria-hidden="true"> · </span>{/if}
              {#if fixAge}<span class="sos-where-age">updated {fixAge}</span>{/if}
            </p>
          {:else}
            <p class="sos-where-place sos-where-unknown">No location yet</p>
            <p class="sos-where-meta">Their phone hasn't sent a position. Call them.</p>
          {/if}
        </div>

        <!-- ── ACT — two oversized, panic-proof actions. ──────────────────── -->
        <div class="sos-actions" role="group" aria-label="Immediate actions">
          <span class="sos-kicker">Immediate actions</span>
          <button class="sos-act sos-act-call" onclick={callThem}>
            <span class="sos-act-puck" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M6.62 10.79a15.05 15.05 0 0 0 6.59 6.59l2.2-2.2a1 1 0 0 1 1.01-.24 11.36 11.36 0 0 0 3.56.57 1 1 0 0 1 1 1V20a1 1 0 0 1-1 1A17 17 0 0 1 3 4a1 1 0 0 1 1-1h3.5a1 1 0 0 1 1 1c0 1.25.2 2.45.57 3.56a1 1 0 0 1-.25 2.11z"/></svg>
            </span>
            <span class="sos-act-text">
              <span class="sos-act-title">Call {firstName}</span>
              <span class="sos-act-sub">Opens a voice call now</span>
            </span>
          </button>
          <button class="sos-act sos-act-map" onclick={seeOnMap}>
            <span class="sos-act-puck" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
            </span>
            <span class="sos-act-text">
              <span class="sos-act-title">See them on the map</span>
              <span class="sos-act-sub">Their live position</span>
            </span>
          </button>
        </div>
      {/if}

      <!-- Medical card: loaded on demand. It is for the responder who has
           arrived, not the first three seconds, so it does not belong in the
           initial route bundle. -->
      {#if activeMedicalCard}
        {#await import('./SosMedicalCard.svelte') then M}
          <M.default card={activeMedicalCard} open={medCardOpen} ontoggle={toggleMedCard} />
        {/await}
      {/if}
    </div>

    {#snippet footer()}

        {#each $alertState.actions as action}
          <button class="sos-foot-btn {action.kind === 'btn-secondary' ? 'foot-quiet' : 'foot-strong'}" onclick={() => { if (action.onClick) action.onClick(); dismiss(); }}>{action.label}</button>
        {/each}
        <button class="sos-foot-btn foot-quiet" onclick={dismiss}>Got it</button>

      {/snippet}
  </Modal>
</div>

<style>
  .alert-shake-wrapper { display: contents; }
  .alert-shake-wrapper.shaking :global(.modal-backdrop) {
    animation: camera-shake 0.45s var(--ease-out);
  }

  /* ── Sheet — warm paper with the vermilion emergency frame. Overrides the
     generic Modal skin for this one urgent instance only. ─────────────────── */
  .alert-shake-wrapper :global(.modal-backdrop .modal-card) {
    background: var(--surface-1);
    border: none;
    border-radius: var(--radius-2xl);
    box-shadow:
      inset 0 0 0 3px var(--danger-500),
      var(--shadow-danger),
      var(--shadow-xl);
    backdrop-filter: none;
    -webkit-backdrop-filter: none;
  }
  /* Keep the spring entrance, drop the box-shadow pulse (not GPU-friendly). */
  .alert-shake-wrapper :global(.modal-backdrop .modal-card.urgent) {
    animation: modal-3d-arrive 480ms var(--ease-spring) both;
  }
  .alert-shake-wrapper :global(.modal-backdrop .modal-card::before) { display: none; }

  /* Header collapses to a floating close button; the title element stays in
     the DOM as the dialog's aria-labelledby target, read via the headline. */
  .alert-shake-wrapper :global(.modal-backdrop .modal-header) {
    position: absolute;
    top: var(--space-2);
    right: var(--space-2);
    padding: 0;
    z-index: 5;
  }
  .alert-shake-wrapper :global(.modal-backdrop .modal-title) {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
    margin: 0;
  }
  .alert-shake-wrapper :global(.modal-backdrop .modal-close) {
    width: 44px;
    height: 44px;
    border-radius: var(--radius-full);
    color: var(--text-secondary);
  }

  .alert-body-wrap {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: var(--space-3);
    text-align: left;
    padding-top: var(--space-2);
  }

  /* ── Eyebrow + headline ──────────────────────────────────────────────── */
  .sos-eyebrow {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    color: var(--danger-500);
  }
  :global([data-theme="dark"]) .sos-eyebrow { color: var(--danger-300); }
  .sos-eyebrow-label {
    font-family: var(--font-sans);
    font-size: var(--text-xs);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.12em;
  }
  .sos-eyebrow-dot {
    width: 8px;
    height: 8px;
    border-radius: var(--radius-full);
    background: currentColor;
    position: relative;
    flex-shrink: 0;
  }
  .sos-eyebrow-dot::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: var(--radius-full);
    background: currentColor;
    animation: sos-dot-ping 1.6s var(--ease-out) infinite;
  }
  @keyframes sos-dot-ping {
    0%   { transform: scale(1);   opacity: 0.55; }
    100% { transform: scale(2.8); opacity: 0;    }
  }

  /* The one honest sentence — Newsreader italic, the verdict voice. */
  .sos-headline {
    margin: 0;
    padding-right: var(--space-8); /* clear the floating close button */
    font-family: var(--font-serif);
    font-style: italic;
    font-weight: 400;
    font-size: var(--text-2xl);
    line-height: 1.3;
    letter-spacing: -0.01em;
    color: var(--text-primary);
  }

  .alert-body {
    margin: 0;
    font-family: var(--font-sans);
    font-size: var(--text-lg);
    line-height: var(--leading-relaxed);
    color: var(--text-secondary);
  }

  /* ── WHO ─────────────────────────────────────────────────────────────── */
  .sos-who {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
    padding: var(--space-3) var(--space-3-5);
    background: var(--surface-2);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
  }
  .sos-pebble-halo {
    position: relative;
    width: 56px;
    height: 56px;
    border-radius: var(--radius-full);
    background: var(--danger-500-12);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .sos-pebble-halo::before {
    content: '';
    position: absolute;
    inset: -3px;
    border-radius: var(--radius-full);
    border: 2px solid color-mix(in oklch, var(--danger-500) 45%, transparent);
    animation: sos-halo-radiate 2s var(--ease-out) infinite;
    pointer-events: none;
  }
  @keyframes sos-halo-radiate {
    0%   { transform: scale(1);    opacity: 0.8; }
    100% { transform: scale(1.45); opacity: 0;   }
  }
  .sos-pebble {
    width: 48px;
    height: 48px;
    border-radius: var(--radius-full);
    background: var(--surface-1);
    border: 2px solid var(--danger-500);
    color: var(--danger-500);
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-sans);
    font-size: var(--text-lg);
    font-weight: 700;
    letter-spacing: -0.02em;
  }
  .sos-pebble-mark {
    position: absolute;
    bottom: 0;
    right: 0;
    width: 18px;
    height: 18px;
    border-radius: var(--radius-full);
    background: var(--danger-500);
    color: var(--paper);
    font-family: var(--font-sans);
    font-size: var(--text-xs);
    font-weight: 700;
    line-height: 14px;
    text-align: center;
    border: 2px solid var(--surface-2);
  }
  .sos-who-name {
    margin: 0;
    font-family: var(--font-sans);
    font-size: var(--text-xl);
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1.25;
    overflow-wrap: anywhere;
  }

  /* ── Telemetry chips ─────────────────────────────────────────────────── */
  .sos-telemetry {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1-5);
  }
  .sig-chip {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    font-family: var(--font-sans);
    font-size: var(--text-xs);
    font-weight: 600;
    line-height: 1.3;
    padding: var(--space-1) var(--space-2-5);
    border-radius: var(--radius-full);
  }
  .sig-motion {
    background: var(--warning-500-12);
    border: 1px solid var(--warning-500-30);
    color: var(--warning-600);
  }
  .sig-battery {
    background: var(--success-500-08);
    border: 1px solid var(--success-500-20);
    color: var(--success-600);
  }
  .sig-trigger {
    background: var(--danger-500-12);
    border: 1px solid var(--danger-500-20);
    color: var(--danger-600);
    animation: sig-trigger-breathe 1.8s ease-in-out infinite;
  }
  @keyframes sig-trigger-breathe {
    0%, 100% { transform: scale(1); }
    50%      { transform: scale(1.05); }
  }
  :global([data-theme="dark"]) .sig-motion  { color: var(--warning-400); }
  :global([data-theme="dark"]) .sig-battery { color: var(--success-400); }
  :global([data-theme="dark"]) .sig-trigger { color: var(--danger-300); }

  /* ── Kickers ─────────────────────────────────────────────────────────── */
  .sos-kicker {
    display: block;
    font-family: var(--font-sans);
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--text-secondary);
  }

  /* ── WHERE ───────────────────────────────────────────────────────────── */
  .sos-where {
    width: 100%;
    padding-top: var(--space-3);
    border-top: 1px solid var(--border-subtle);
  }
  .sos-where-place {
    margin: var(--space-1) 0 0;
    font-family: var(--font-sans);
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1.25;
    overflow-wrap: anywhere;
  }
  .sos-where-unknown { font-weight: 700; }
  .sos-where-meta {
    margin: var(--space-1) 0 0;
    font-family: var(--font-sans);
    font-size: var(--text-base);
    color: var(--text-secondary);
    line-height: 1.45;
  }
  .sos-where-dist {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    color: var(--primary-700);
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }
  :global([data-theme="dark"]) .sos-where-dist { color: var(--primary-500); }
  .sos-where-age { font-variant-numeric: tabular-nums; }

  /* ── ACT ─────────────────────────────────────────────────────────────── */
  .sos-actions {
    display: flex;
    flex-direction: column;
    gap: var(--space-2-5);
    width: 100%;
    padding-top: var(--space-3);
    border-top: 1px solid var(--border-subtle);
  }
  .sos-act {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
    min-height: 58px;
    padding: var(--space-3) var(--space-4);
    border: none;
    border-radius: var(--radius-md);
    cursor: pointer;
    text-align: left;
    font-family: var(--font-sans);
    transition: transform var(--duration-fast) var(--ease-out);
  }
  .sos-act:active { transform: scale(0.98); }
  .sos-act:focus-visible {
    outline: 3px solid var(--text-primary);
    outline-offset: 2px;
  }
  .sos-act-call {
    background: var(--danger-500);
    color: var(--paper);
    box-shadow: var(--shadow-danger);
  }
  .sos-act-map {
    background: var(--ink);
    color: var(--paper);
    box-shadow: var(--shadow-md);
  }
  .sos-act-puck {
    width: 40px;
    height: 40px;
    border-radius: var(--radius-full);
    background: color-mix(in oklch, var(--paper) 22%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .sos-act-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .sos-act-title {
    font-size: var(--text-xl);
    font-weight: 700;
    line-height: 1.2;
    overflow-wrap: anywhere;
  }
  .sos-act-sub {
    font-size: var(--text-sm);
    font-weight: 500;
    opacity: 0.92;
    line-height: 1.3;
  }

  /* ── Footer — quiet, deliberate secondary responses ──────────────────── */
  .sos-foot-btn {
    flex: 1 1 10rem;
    min-height: 48px;
    padding: var(--space-2-5) var(--space-4);
    border-radius: var(--radius-md);
    font-family: var(--font-sans);
    font-size: var(--text-base);
    font-weight: 600;
    cursor: pointer;
    transition: transform var(--duration-fast) var(--ease-out);
  }
  .sos-foot-btn:active { transform: scale(0.98); }
  .sos-foot-btn:focus-visible {
    outline: 2px solid var(--danger-500);
    outline-offset: 2px;
  }
  .foot-strong {
    background: var(--danger-500-12);
    border: 1.5px solid color-mix(in oklch, var(--danger-500) 55%, transparent);
    color: var(--danger-600);
  }
  :global([data-theme="dark"]) .foot-strong { color: var(--danger-300); }
  .foot-quiet {
    background: transparent;
    border: 1px solid var(--border-strong);
    color: var(--text-secondary);
  }

  /* Reduced motion — no shake, no pings, no springs. The static vermilion
     frame still carries the urgency. */
  @media (prefers-reduced-motion: reduce) {
    .alert-shake-wrapper.shaking :global(.modal-backdrop) {
      animation: none;
    }
    .alert-shake-wrapper :global(.modal-backdrop .modal-card.urgent) {
      animation: none;
    }
    .sos-eyebrow-dot::after,
    .sos-pebble-halo::before,
    .sig-trigger {
      animation: none;
    }
    .sos-act, .sos-act:active,
    .sos-foot-btn, .sos-foot-btn:active {
      transition: none;
      transform: none;
    }
  }
</style>
