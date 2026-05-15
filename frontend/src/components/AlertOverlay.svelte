<script>
  import { alertState, sosNarratives, activeSosUsers, geofenceShake } from '../lib/stores/sos.js';
  import Modal from './primitives/Modal.svelte';
  import { haptics } from '../lib/haptics.js';

  // ── Derive full narrative data (narrative + optional medicalCard) ─────────
  // sosNarratives stores the full payload: { sosToken, userId, narrative, medicalCard? }
  $: activeSosData = (() => {
    for (const [userId, sos] of $activeSosUsers) {
      const n = $sosNarratives.get(userId);
      if (n) return n;
      if (sos.sos?.narrative) return { narrative: sos.sos.narrative };
    }
    return null;
  })();

  $: activeNarrative  = activeSosData?.narrative   || null;
  $: activeMedicalCard = activeSosData?.medicalCard || null;

  // Expand/collapse medical card
  let medCardOpen = false;
  $: if (activeMedicalCard) medCardOpen = true; // auto-expand when data arrives

  let audioCtx   = null;
  let oscillator = null;
  let shaking    = false;

  $: if ($alertState.visible && $alertState.alarmMs > 0) {
    startAlarm();
    triggerHaptic();
    triggerShake();
  } else {
    stopAlarm();
  }

  // Feature 8: geofence breach shake — camera shake without audio alarm
  let _prevGeofenceShake = 0;
  $: if ($geofenceShake > _prevGeofenceShake) {
    _prevGeofenceShake = $geofenceShake;
    triggerShake();
    haptics.warning?.();
  }

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

      <p class="alert-body">{$alertState.body}</p>

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
        <div class="med-card" role="region" aria-label="Emergency medical information">
          <button
            class="med-card-header"
            on:click={toggleMedCard}
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

              <!-- Blood type — the most critical field, shown large -->
              {#if activeMedicalCard.bloodType}
                <div class="med-row med-row-bloodtype">
                  <span class="med-blood-label">Blood Type</span>
                  <span class="med-blood-value">{activeMedicalCard.bloodType}</span>
                </div>
              {/if}

              <!-- Allergies — highlighted as critical -->
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
                    <span class="med-field-value">{activeMedicalCard.allergies}</span>
                  </div>
                </div>
              {/if}

              <!-- Medications -->
              {#if activeMedicalCard.medications?.trim()}
                <div class="med-row">
                  <div class="med-field-content">
                    <span class="med-field-label">Medications</span>
                    <span class="med-field-value">{activeMedicalCard.medications}</span>
                  </div>
                </div>
              {/if}

              <!-- Medical conditions -->
              {#if activeMedicalCard.conditions?.trim()}
                <div class="med-row">
                  <div class="med-field-content">
                    <span class="med-field-label">Conditions</span>
                    <span class="med-field-value">{activeMedicalCard.conditions}</span>
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
      {/if}
    </div>

    <svelte:fragment slot="footer">
      {#each $alertState.actions as action}
        <button class="btn {action.kind || 'btn-primary'} btn-lg" on:click={() => { if (action.onClick) action.onClick(); dismiss(); }}>{action.label}</button>
      {/each}
      <button class="btn btn-secondary btn-lg" on:click={dismiss}>Got it</button>
    </svelte:fragment>
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

  .alert-sos-icon {
    width: 72px;
    height: 72px;
    border-radius: 50%;
    background: radial-gradient(circle at 35% 35%, rgba(239,68,68,0.25) 0%, rgba(239,68,68,0.12) 100%);
    border: 2px solid rgba(239, 68, 68, 0.55);
    border-top-color: rgba(255, 80, 80, 0.80);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--danger-400);
    position: relative;
    animation: sos-neon-ring 1.4s ease-in-out infinite;
    flex-shrink: 0;
    box-shadow:
      0 0 0 6px rgba(239, 68, 68, 0.12),
      0 0 0 12px rgba(239, 68, 68, 0.06),
      0 0 0 20px rgba(239, 68, 68, 0.03),
      0 0 40px rgba(239, 68, 68, 0.35),
      inset 0 1px 0 rgba(255,255,255,0.15);
  }

  /* Outer ring expander — radiates outward every pulse */
  .alert-sos-icon::before {
    content: '';
    position: absolute;
    inset: -14px;
    border-radius: 50%;
    border: 2px solid rgba(239, 68, 68, 0.35);
    animation: sos-ring-radiate 1.4s ease-out infinite;
    pointer-events: none;
  }

  /* Second ring — offset by half period for continuous effect */
  .alert-sos-icon::after {
    content: '';
    position: absolute;
    inset: -14px;
    border-radius: 50%;
    border: 1.5px solid rgba(239, 68, 68, 0.20);
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
    background: rgba(245,158,11,0.15);
    color: var(--warning-500);
    border: 1px solid rgba(245,158,11,0.30);
    box-shadow: 0 0 6px rgba(245,158,11,0.20);
  }
  .narrative-chip.battery {
    background: rgba(16,185,129,0.12);
    color: var(--success-500);
    border: 1px solid rgba(16,185,129,0.25);
    box-shadow: 0 0 6px rgba(16,185,129,0.18);
  }
  .narrative-chip.trigger {
    background: rgba(239,68,68,0.12);
    color: var(--danger-500);
    border: 1px solid rgba(239,68,68,0.28);
    box-shadow: 0 0 6px rgba(239,68,68,0.22);
    animation: chip-breathe-sos 1.8s ease-in-out infinite;
  }

  /* ── Medical Card (Feature 9) ────────────────────────────────────────────── */
  .med-card {
    width: 100%;
    max-width: 380px;
    margin-top: var(--space-1);
    border-radius: var(--radius-lg);
    /* Token-based: dark mode uses the same values since danger tokens
       already pick up the correct shade from the theme */
    background: rgba(239, 68, 68, 0.05);
    border: 1px solid rgba(239, 68, 68, 0.20);
    overflow: hidden;
    text-align: left;
  }

  /* Dark-mode override — uses [data-theme] to match the app's theme system,
     not prefers-color-scheme which ignores the user's in-app theme toggle. */
  :global([data-theme="dark"]) .med-card,
  :global(:root:not([data-theme="light"])) .med-card {
    background: rgba(239, 68, 68, 0.08);
    border-color: rgba(239, 68, 68, 0.25);
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
  .med-card-header:hover { background: rgba(239,68,68,0.14); }
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
    background: rgba(239, 68, 68, 0.14);
    color: var(--danger-500);
    flex-shrink: 0;
    box-shadow: 0 0 8px rgba(239, 68, 68, 0.25);
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

  .med-card-chevron {
    color: var(--danger-400);
    display: flex;
    align-items: center;
    transition: transform 220ms cubic-bezier(0.16, 1, 0.3, 1);
  }
  .med-card-chevron.open { transform: rotate(180deg); }

  /* Constrain body height so it scrolls on short phones (667px viewport).
     100dvh accounts for iOS browser chrome; safe-area insets cover notch and home indicator. */
  .med-card-body {
    display: flex;
    flex-direction: column;
    gap: 1px;
    background: rgba(239, 68, 68, 0.08);
    border-top: 1px solid rgba(239, 68, 68, 0.15);
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

  /* Blood type — large, prominent */
  .med-row-bloodtype {
    align-items: center;
    justify-content: space-between;
    background: rgba(239, 68, 68, 0.06);
  }
  .med-blood-label {
    font-family: var(--font-display);
    font-size: var(--text-2xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--danger-500);
  }
  .med-blood-value {
    font-size: 22px;
    font-weight: 800;
    color: var(--danger-600);
    line-height: 1;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.02em;
  }

  /* Allergies — red highlight */
  .med-row-alert {
    background: rgba(239, 68, 68, 0.07);
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
    gap: 2px;
    min-width: 0;
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

  .med-phone-link {
    color: var(--primary-500);
    text-decoration: none;
    font-weight: 600;
  }
  .med-phone-link:hover { text-decoration: underline; }
  .med-relation { opacity: 0.75; font-weight: 400; }
  .med-address  { opacity: 0.7; font-size: 0.8em; }

  /* Reduced motion — disable shake animation and icon pulse */
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
  }
</style>
