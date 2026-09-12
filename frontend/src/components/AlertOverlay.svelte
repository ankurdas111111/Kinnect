<script>
  import { run } from 'svelte/legacy';

  import { alertState, sosNarratives, activeSosUsers, geofenceShake } from '../lib/stores/sos.js';
  import Modal from './primitives/Modal.svelte';
  import Card from './primitives/Card.svelte';
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

  // Format a medical card row — only render when value is non-empty
  function hasMedField(card, ...keys) {
    return keys.some(k => card?.[k]?.trim?.());
  }

  // Split a free-text medical field into individual chips (comma / newline / semicolon).
  // Purely presentational — the underlying binding is unchanged.
  function chipList(value) {
    return (value || '')
      .split(/[,\n;]+/)
      .map(s => s.trim())
      .filter(Boolean);
  }
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

  let personName = $derived(sosPerson?.displayName || 'them');
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
  let hasFix = $derived(sosPerson?.latitude != null);
  let whereHeadline = $derived(
    sosPerson?.locationLabel || distanceText || (hasFix ? 'On the map' : null)
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

      <!-- ── Medical Card (Feature 9) ──────────────────────────────────── -->
      {#if activeMedicalCard}
        <div class="med-card-wrap">
          <Card variant="glass" glow="danger" padding="none" hover={false}>
            <div class="med-card" role="region" aria-label="Emergency medical information">
              <button
                class="med-card-header"
                onclick={toggleMedCard}
                aria-expanded={medCardOpen}
                aria-controls="med-card-body"
              >
                <span class="med-card-icon" aria-hidden="true">
                  <!-- Medical cross icon -->
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="3" y="3" width="18" height="18" rx="3"/>
                    <line x1="12" y1="8" x2="12" y2="16"/>
                    <line x1="8"  y1="12" x2="16" y2="12"/>
                  </svg>
                </span>
                <span class="med-card-title">Medical Card</span>
                <span class="med-card-chevron" class:open={medCardOpen} aria-hidden="true">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="6 9 12 15 18 9"/>
                  </svg>
                </span>
              </button>

              {#if medCardOpen}
                <div class="med-card-body" id="med-card-body">

                  <!-- Blood type — the single most critical field: extra-large, centered -->
                  {#if activeMedicalCard.bloodType}
                    <div class="med-row med-row-bloodtype">
                      <span class="med-blood-label">Blood Type</span>
                      <span class="med-blood-value">{activeMedicalCard.bloodType}</span>
                    </div>
                  {/if}

                  <!-- Allergies — highlighted as critical (danger chips) -->
                  {#if activeMedicalCard.allergies?.trim()}
                    <div class="med-row med-row-alert">
                      <span class="med-field-icon" aria-hidden="true">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                          <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
                          <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
                        </svg>
                      </span>
                      <div class="med-field-content">
                        <span class="med-field-label">Allergies</span>
                        <div class="med-chips">
                          {#each chipList(activeMedicalCard.allergies) as item}
                            <span class="med-chip med-chip-danger">{item}</span>
                          {/each}
                        </div>
                      </div>
                    </div>
                  {/if}

                  <!-- Medications — warning chips -->
                  {#if activeMedicalCard.medications?.trim()}
                    <div class="med-row">
                      <div class="med-field-content">
                        <span class="med-field-label">Medications</span>
                        <div class="med-chips">
                          {#each chipList(activeMedicalCard.medications) as item}
                            <span class="med-chip med-chip-warning">{item}</span>
                          {/each}
                        </div>
                      </div>
                    </div>
                  {/if}

                  <!-- Medical conditions — neutral chips -->
                  {#if activeMedicalCard.conditions?.trim()}
                    <div class="med-row">
                      <div class="med-field-content">
                        <span class="med-field-label">Conditions</span>
                        <div class="med-chips">
                          {#each chipList(activeMedicalCard.conditions) as item}
                            <span class="med-chip med-chip-neutral">{item}</span>
                          {/each}
                        </div>
                      </div>
                    </div>
                  {/if}

                  <!-- Emergency contacts (array — new format) -->
                  {#if activeMedicalCard.emergencyContacts?.length}
                    {#each activeMedicalCard.emergencyContacts as contact, ci}
                      <div class="med-row med-row-contact">
                        <span class="med-field-icon" aria-hidden="true">
                          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07A19.5 19.5 0 013.07 9.81a19.79 19.79 0 01-3.07-8.63A2 2 0 012 .92h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L6.09 8.91A16 16 0 0015.1 17.9l1.27-1.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 16.92z"/>
                          </svg>
                        </span>
                        <div class="med-field-content">
                          <span class="med-field-label">
                            Emergency Contact {activeMedicalCard.emergencyContacts.length > 1 ? ci + 1 : ''}
                            {#if contact.relation}<span class="med-relation"> · {contact.relation}</span>{/if}
                          </span>
                          <span class="med-field-value">
                            {contact.name || ''}
                            {#if contact.phone}
                              {contact.name ? ' · ' : ''}<a class="med-phone-link" href="tel:{contact.phone}">{contact.phone}</a>
                            {/if}
                            {#if contact.address}
                              <span class="med-address"> · {contact.address}</span>
                            {/if}
                          </span>
                        </div>
                      </div>
                    {/each}
                  <!-- Fallback: legacy single contact fields -->
                  {:else if hasMedField(activeMedicalCard, 'emergencyName', 'emergencyPhone')}
                    <div class="med-row med-row-contact">
                      <span class="med-field-icon" aria-hidden="true">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                          <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07A19.5 19.5 0 013.07 9.81a19.79 19.79 0 01-3.07-8.63A2 2 0 012 .92h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L6.09 8.91A16 16 0 0015.1 17.9l1.27-1.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 16.92z"/>
                        </svg>
                      </span>
                      <div class="med-field-content">
                        <span class="med-field-label">Emergency Contact</span>
                        <span class="med-field-value">
                          {activeMedicalCard.emergencyName || ''}
                          {#if activeMedicalCard.emergencyPhone}
                            {activeMedicalCard.emergencyName ? ' · ' : ''}<a class="med-phone-link" href="tel:{activeMedicalCard.emergencyPhone}">{activeMedicalCard.emergencyPhone}</a>
                          {/if}
                        </span>
                      </div>
                    </div>
                  {/if}

                  <!-- Doctor -->
                  {#if hasMedField(activeMedicalCard, 'doctorName', 'doctorPhone')}
                    <div class="med-row med-row-contact">
                      <span class="med-field-icon" aria-hidden="true">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                          <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/><circle cx="12" cy="7" r="4"/>
                        </svg>
                      </span>
                      <div class="med-field-content">
                        <span class="med-field-label">Primary Doctor</span>
                        <span class="med-field-value">
                          {activeMedicalCard.doctorName || ''}
                          {#if activeMedicalCard.doctorPhone}
                            {activeMedicalCard.doctorName ? ' · ' : ''}<a class="med-phone-link" href="tel:{activeMedicalCard.doctorPhone}">{activeMedicalCard.doctorPhone}</a>
                          {/if}
                        </span>
                      </div>
                    </div>
                  {/if}

                  <!-- Language / responder notes -->
                  {#if activeMedicalCard.language?.trim()}
                    <div class="med-row">
                      <div class="med-field-content">
                        <span class="med-field-label">Language</span>
                        <span class="med-field-value">{activeMedicalCard.language}</span>
                      </div>
                    </div>
                  {/if}

                  {#if activeMedicalCard.responderNotes?.trim()}
                    <div class="med-row">
                      <div class="med-field-content">
                        <span class="med-field-label">Responder Notes</span>
                        <span class="med-field-value">{activeMedicalCard.responderNotes}</span>
                      </div>
                    </div>
                  {/if}

                </div>
              {/if}
            </div>
          </Card>
        </div>
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
  .med-card-wrap {
    width: 100%;
    max-width: 380px;
    margin-top: var(--space-1);
  }

  .med-card {
    width: 100%;
    overflow: hidden;
    text-align: left;
  }

  .med-card-header {
    width: 100%;
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2-5) var(--space-3-5);
    min-height: 44px;
    background: transparent;
    border: none;
    cursor: pointer;
    text-align: left;
    transition: background var(--duration-fast) var(--ease-out);
  }
  .med-card-header:hover { background: var(--danger-500-12); }
  .med-card-header:focus-visible {
    outline: 2px solid var(--danger-400);
    outline-offset: -2px;
  }

  .med-card-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: var(--radius-sm);
    background: var(--danger-500-12);
    color: var(--danger-500);
    flex-shrink: 0;
    box-shadow: 0 0 8px color-mix(in oklch, var(--danger-500) 25%, transparent);
  }

  .med-card-title {
    flex: 1;
    font-family: var(--font-display);
    font-size: var(--text-2xs);
    font-weight: 700;
    color: var(--danger-600);
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  /* Chevron — spring affordance (transform-only, reduced-motion safe) */
  .med-card-chevron {
    color: var(--danger-400);
    display: flex;
    align-items: center;
    transition: transform var(--duration-normal) var(--ease-spring);
  }
  .med-card-chevron.open { transform: rotate(180deg) scale(1.08); }

  /* Constrain body height so it scrolls on short phones (667px viewport).
     100dvh accounts for iOS browser chrome; safe-area insets cover notch and home indicator. */
  .med-card-body {
    display: flex;
    flex-direction: column;
    gap: 1px;
    background: var(--danger-500-12);
    border-top: 1px solid color-mix(in oklch, var(--danger-500) 15%, transparent);
    max-height: calc(100dvh - 160px - env(safe-area-inset-top, 0px) - env(safe-area-inset-bottom, 0px));
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
  }

  /* med-row: dark-first surface — uses surface token so light theme overrides naturally */
  .med-row {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2);
    padding: 9px 14px;
    /* Token-aware: var(--surface-1) resolves to a dark surface in dark mode,
       and a light surface in [data-theme="light"] — no hardcoded white */
    background: var(--surface-1, rgba(15, 23, 42, 0.70));
    backdrop-filter: blur(8px);
  }

  :global([data-theme="light"]) .med-row {
    background: rgba(255, 255, 255, 0.92);
  }

  /* Blood type — extra-large, high-contrast, centered (most critical field) */
  .med-row-bloodtype {
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-1);
    text-align: center;
    padding: var(--space-3) var(--space-3-5);
    background: var(--danger-500-12);
  }
  .med-blood-label {
    font-family: var(--font-display);
    font-size: var(--text-2xs);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--danger-400);
  }
  .med-blood-value {
    font-family: var(--font-display);
    font-size: var(--text-4xl);
    font-weight: 800;
    color: var(--danger-500);
    line-height: 1;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.02em;
    text-shadow: 0 0 18px color-mix(in oklch, var(--danger-500) 35%, transparent);
  }

  /* Allergies — red highlight */
  .med-row-alert {
    background: color-mix(in oklch, var(--danger-500) 7%, transparent);
  }
  .med-row-alert .med-field-label { color: var(--danger-600); }

  /* Contact rows — empty rule kept for selector specificity */
  .med-row-contact {}

  .med-field-icon {
    flex-shrink: 0;
    color: var(--danger-400);
    margin-top: 2px;
    display: flex;
    align-items: center;
  }

  .med-field-content {
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5);
    min-width: 0;
    flex: 1;
  }

  .med-field-label {
    font-family: var(--font-display);
    font-size: var(--text-2xs);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--text-tertiary);
  }

  .med-field-value {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text-primary);
    line-height: 1.4;
    word-break: break-word;
  }

  /* ── Medical field chips — hierarchy: danger > warning > neutral ─────────── */
  .med-chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1-5);
  }
  .med-chip {
    display: inline-block;
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 600;
    padding: var(--space-1) var(--space-2-5);
    border-radius: var(--radius-full);
    line-height: 1.3;
    /* no truncation — long entries wrap within the chip */
    word-break: break-word;
    white-space: normal;
  }
  .med-chip-danger {
    background: var(--danger-500-12);
    color: var(--danger-300);
    border: 1px solid var(--danger-500-20);
  }
  .med-chip-warning {
    background: color-mix(in oklch, var(--warning-500) 14%, transparent);
    color: var(--warning-500);
    border: 1px solid color-mix(in oklch, var(--warning-500) 28%, transparent);
  }
  .med-chip-neutral {
    background: var(--surface-2, rgba(255, 255, 255, 0.06));
    color: var(--text-secondary);
    border: 1px solid var(--border-default);
  }

  .med-phone-link {
    color: var(--primary-500);
    text-decoration: none;
    font-weight: 600;
  }
  .med-phone-link:hover { text-decoration: underline; }
  .med-relation { opacity: 0.75; font-weight: 400; }
  .med-address  { opacity: 0.7; font-size: 0.8em; }

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
