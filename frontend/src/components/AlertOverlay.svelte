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
      <div class="alert-sos-icon" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
          <line x1="12" y1="9" x2="12" y2="13"/>
          <line x1="12" y1="17" x2="12.01" y2="17"/>
        </svg>
      </div>

      {#if $alertState.body}
        <p class="alert-body">{$alertState.body}</p>
      {/if}

      <!-- ── WHERE — the first question a reader actually asks. Above the
           medical card, because a parent 2 km away needs a direction to drive
           before they need a blood type. ─────────────────────────────────── -->
      {#if sosPerson}
        <div class="sos-where" role="group" aria-label="Location">
          {#if whereHeadline}
            <p class="sos-where-place">{whereHeadline}</p>
            <p class="sos-where-meta">
              {#if whereMeta}<span class="sos-where-dist">{whereMeta}</span>{/if}
              {#if whereMeta && fixAge}<span aria-hidden="true"> · </span>{/if}
              {#if fixAge}<span>updated {fixAge}</span>{/if}
            </p>
          {:else}
            <p class="sos-where-place sos-where-unknown">No location yet</p>
            <p class="sos-where-meta">Their phone hasn't sent a position. Call them.</p>
          {/if}
        </div>

        <!-- ── ACT — three things a person can do, in the order they'd do
             them. These sit above the fold; medical detail is below. ─────── -->
        <div class="sos-actions">
          <button class="sos-act sos-act-primary" onclick={callThem}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M6.62 10.79a15.05 15.05 0 0 0 6.59 6.59l2.2-2.2a1 1 0 0 1 1.01-.24 11.36 11.36 0 0 0 3.56.57 1 1 0 0 1 1 1V20a1 1 0 0 1-1 1A17 17 0 0 1 3 4a1 1 0 0 1 1-1h3.5a1 1 0 0 1 1 1c0 1.25.2 2.45.57 3.56a1 1 0 0 1-.25 2.11z"/></svg>
            Call {firstName}
          </button>
          <button class="sos-act" onclick={seeOnMap}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
            See on map
          </button>
        </div>
      {/if}

      <!-- Motion narrative chips -->
      {#if activeNarrative}
        <div class="narrative-chips">
          {#if activeNarrative.motionSummary}
            <span class="narrative-chip motion">{activeNarrative.motionSummary}</span>
          {/if}
          {#if activeNarrative.batteryPct != null}
            <span class="narrative-chip battery">Battery {activeNarrative.batteryPct}%</span>
          {/if}
          {#if activeNarrative.triggerRule && activeNarrative.triggerRule !== 'manual'}
            <span class="narrative-chip trigger">Auto: {activeNarrative.triggerRule}</span>
          {/if}
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
          <button class="btn {action.kind || 'btn-primary'} btn-lg" onclick={() => { if (action.onClick) action.onClick(); dismiss(); }}>{action.label}</button>
        {/each}
        <button class="btn btn-secondary btn-lg" onclick={dismiss}>Got it</button>

      {/snippet}
  </Modal>
</div>

<style>
  .alert-shake-wrapper { position: contents; }
  .alert-shake-wrapper.shaking :global(.modal-backdrop) {
    animation: camera-shake 0.45s var(--ease-out);
  }

  .alert-body-wrap {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    text-align: center;
    padding-top: var(--space-2);
  }

  /* ── Where ────────────────────────────────────────────────────────────── */
  .sos-where {
    width: 100%;
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius-lg);
    background: var(--surface-3);
    border: 1px solid var(--border-default);
    text-align: center;
  }
  .sos-where-place {
    margin: 0;
    font-family: var(--font-display);
    font-size: var(--text-lg);
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1.25;
  }
  .sos-where-unknown { color: var(--text-secondary); font-weight: 600; }
  .sos-where-meta {
    margin: var(--space-1) 0 0;
    font-size: var(--text-sm);
    color: var(--text-secondary);
    line-height: 1.4;
  }
  .sos-where-dist { font-variant-numeric: tabular-nums; }

  /* ── Act ──────────────────────────────────────────────────────────────── */
  .sos-actions {
    display: flex;
    gap: var(--space-2);
    width: 100%;
  }
  .sos-act {
    flex: 1;
    min-height: 44px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    padding: var(--space-2-5) var(--space-3);
    border-radius: var(--radius-lg);
    border: 1.5px solid color-mix(in oklch, var(--primary-500) 45%, transparent);
    background: var(--surface-1);
    color: var(--primary-600);
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 600;
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out),
                transform var(--duration-fast) var(--ease-out);
  }
  .sos-act:hover { background: var(--primary-50); }
  .sos-act:active { transform: scale(0.98); }
  .sos-act:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .sos-act-primary {
    background: var(--primary-500);
    border-color: var(--primary-500);
    color: var(--text-on-primary);
  }
  .sos-act-primary:hover { background: var(--primary-600); }
  @media (prefers-reduced-motion: reduce) {
    .sos-act, .sos-act:active { transition: none; transform: none; }
  }

  .alert-sos-icon {
    width: 72px;
    height: 72px;
    border-radius: 50%;
    background: radial-gradient(circle at 35% 35%, color-mix(in oklch, var(--danger-500) 25%, transparent) 0%, color-mix(in oklch, var(--danger-500) 12%, transparent) 100%);
    border: 2px solid color-mix(in oklch, var(--danger-500) 55%, transparent);
    border-top-color: color-mix(in oklch, var(--danger-400) 80%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--danger-400);
    position: relative;
    animation: sos-neon-ring 1.4s ease-in-out infinite;
    flex-shrink: 0;
    box-shadow:
      0 0 0 6px color-mix(in oklch, var(--danger-500) 12%, transparent),
      0 0 0 12px color-mix(in oklch, var(--danger-500) 6%, transparent),
      0 0 0 20px color-mix(in oklch, var(--danger-500) 3%, transparent),
      0 0 40px color-mix(in oklch, var(--danger-500) 35%, transparent),
      inset 0 1px 0 rgba(255,255,255,0.15);
  }

  /* Outer ring expander — radiates outward every pulse */
  .alert-sos-icon::before {
    content: '';
    position: absolute;
    inset: -14px;
    border-radius: 50%;
    border: 2px solid color-mix(in oklch, var(--danger-500) 35%, transparent);
    animation: sos-ring-radiate 1.4s ease-out infinite;
    pointer-events: none;
  }

  /* Second ring — offset by half period for continuous effect */
  .alert-sos-icon::after {
    content: '';
    position: absolute;
    inset: -14px;
    border-radius: 50%;
    border: 1.5px solid color-mix(in oklch, var(--danger-500) 20%, transparent);
    animation: sos-ring-radiate 1.4s ease-out 0.7s infinite;
    pointer-events: none;
  }

  @keyframes sos-ring-radiate {
    0%   { transform: scale(1);    opacity: 0.8; }
    100% { transform: scale(1.55); opacity: 0;   }
  }

  .alert-body {
    font-family: var(--font-display);
    color: var(--text-secondary);
    font-size: var(--text-base);
    text-align: center;
    line-height: var(--leading-relaxed);
    margin: 0;
  }

  .narrative-chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1-5);
    justify-content: center;
    margin-top: var(--space-1);
  }

  .narrative-chip {
    display: inline-block;
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 600;
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-full);
    letter-spacing: 0.01em;
  }

  .narrative-chip.motion  {
    background: color-mix(in oklch, var(--warning-500) 15%, transparent);
    color: var(--warning-500);
    border: 1px solid color-mix(in oklch, var(--warning-500) 30%, transparent);
    box-shadow: 0 0 6px color-mix(in oklch, var(--warning-500) 20%, transparent);
  }
  .narrative-chip.battery {
    background: color-mix(in oklch, var(--success-500) 12%, transparent);
    color: var(--success-500);
    border: 1px solid color-mix(in oklch, var(--success-500) 25%, transparent);
    box-shadow: 0 0 6px color-mix(in oklch, var(--success-500) 18%, transparent);
  }
  .narrative-chip.trigger {
    background: color-mix(in oklch, var(--danger-500) 12%, transparent);
    color: var(--danger-500);
    border: 1px solid color-mix(in oklch, var(--danger-500) 28%, transparent);
    box-shadow: 0 0 6px color-mix(in oklch, var(--danger-500) 22%, transparent);
    animation: chip-breathe-sos 1.8s ease-in-out infinite;
  }

  /* ── Medical Card (Feature 9) ────────────────────────────────────────────── */
  /* The <Card variant="glass" glow="danger"> supplies the surface, danger glow,
     and top-edge accent line. The wrapper only constrains width. */

  /* Reduced motion — disable shake animation, icon pulse, and chevron spring */
  @media (prefers-reduced-motion: reduce) {
    .alert-shake-wrapper.shaking :global(.modal-backdrop) {
      animation: none;
    }
    .alert-sos-icon,
    .alert-sos-icon::before,
    .alert-sos-icon::after {
      animation: none;
    }
    .narrative-chip.trigger {
      animation: none;
    }
    .med-card-chevron {
      transition: none;
    }
    .med-card-chevron.open {
      transform: rotate(180deg);
    }
  }
</style>
