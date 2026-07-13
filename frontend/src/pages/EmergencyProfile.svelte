<script>
  /**
   * EmergencyProfile — orchestrator. Owns ALL state, save/emit, socket, and
   * localStorage logic; components/emergency/* children are presentational
   * (props + callbacks, zero socket imports). Storage shape, validation, and
   * legacy-contact migration live in lib/emergencyProfile.js (tested).
   */
  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { authUser } from '../lib/stores/auth.js';
  import { toasts } from '../lib/stores/toast.js';
  import { socket } from '../lib/socket.js';
  import {
    STORAGE_KEY, DEFAULTS, MAX_CONTACTS, TRACKED_FIELDS,
    isValidPhone, isValidDob, migrateProfile, syncLegacyContact, countFilled,
  } from '../lib/emergencyProfile.js';
  import '../components/emergency/emergency.css';
  import CompletenessMeter from '../components/emergency/CompletenessMeter.svelte';
  import EpSection from '../components/emergency/EpSection.svelte';
  import PersonalInfoSection from '../components/emergency/PersonalInfoSection.svelte';
  import EmergencyContactsSection from '../components/emergency/EmergencyContactsSection.svelte';
  import EscalationSection from '../components/emergency/EscalationSection.svelte';
  import MedicalInfoSection from '../components/emergency/MedicalInfoSection.svelte';
  import ResponderNotesSection from '../components/emergency/ResponderNotesSection.svelte';

  $effect(() => { if (!$authUser) push('/login'); });
  $effect(() => { if ($authUser?.displayName && !profile.fullName) profile.fullName = $authUser.displayName; });

  // ── State (parent-owned; children receive props/bindables only) ───────────
  let profile = $state({ ...DEFAULTS });
  // Critical sections default OPEN — an anxious user must never hunt for them.
  let openSections = $state({ personal: true, contacts: true, escalation: true, medical: true, responder: true, qr: false });
  let saving = $state(false), saveSuccess = $state(false);
  let panicPhone1 = $state(''), panicPhone2 = $state(''), panicSaving = $state(false);
  let heartbeatEnabled = $state(false), heartbeatDeadline = $state('10:00');
  let settingsLoading = $state(true); // getEmergencySettings hydration in flight
  let saveTimer = null, panicTimer = null, settingsTimer = null;
  const todayIso = new Date().toISOString().split('T')[0];

  // ── Derived: completeness + validation ────────────────────────────────────
  let filledCount = $derived(countFilled(profile));
  let progress = $derived(Math.round((filledCount / TRACKED_FIELDS.length) * 100));
  let contactPhoneErrors = $derived((profile.emergencyContacts || []).map(c =>
    c.phone && !isValidPhone(c.phone) ? 'Enter a valid phone number' : ''));
  let doctorPhoneError = $derived(profile.doctorPhone && !isValidPhone(profile.doctorPhone) ? 'Enter a valid phone number' : '');
  let dobError = $derived(profile.dob && !isValidDob(profile.dob) ? 'Enter a valid date of birth' : '');
  let hasErrors = $derived(!!(doctorPhoneError || dobError || contactPhoneErrors.some(Boolean)));
  let lastUpdatedLabel = $derived(fmtDate(profile.updatedAt));
  let shareCode = $derived($authUser?.shareCode ?? '');

  function fmtDate(iso) {
    if (!iso) return null;
    try { return new Date(iso).toLocaleString([], { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' }); }
    catch { return null; }
  }

  function addContact() {
    if (profile.emergencyContacts.length >= MAX_CONTACTS) return;
    profile.emergencyContacts = [...profile.emergencyContacts, { name: '', relation: '', phone: '', address: '' }];
    if (!openSections.contacts) openSections.contacts = true;
  }
  function removeContact(i) {
    profile.emergencyContacts = profile.emergencyContacts.filter((_, idx) => idx !== i);
  }

  // ── Save to localStorage (round-trips through the tested lib functions) ───
  function save() {
    if (hasErrors) { toasts.error('Please fix the errors before saving.'); return; }
    saving = true;
    const updated = { ...syncLegacyContact(profile), updatedAt: new Date().toISOString() };
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
      profile = updated;
      saveSuccess = true;
      toasts.success('Emergency profile saved.');
      clearTimeout(saveTimer);
      saveTimer = setTimeout(() => { saveSuccess = false; }, 2200);
    } catch { toasts.error('Could not save profile. Check device storage.'); }
    finally { saving = false; }
  }

  // ── Server-side escalation settings (panic relay + heartbeat) ─────────────
  function savePanicPhones() {
    panicSaving = true;
    socket.emit('setEmergencyPhones', { phone1: panicPhone1.trim(), phone2: panicPhone2.trim() });
    clearTimeout(panicTimer);
    panicTimer = setTimeout(() => { panicSaving = false; }, 4000); // ack fallback
  }
  function saveHeartbeat() {
    socket.emit('setHeartbeat', { enabled: heartbeatEnabled, deadline: heartbeatDeadline });
    toasts.add(heartbeatEnabled ? 'Heartbeat check enabled' : 'Heartbeat check disabled');
  }

  // ── Socket listeners — named refs so socket.off() removes ONLY ours ───────
  const _onEmergencySettings = (data) => {
    if (settingsTimer) { clearTimeout(settingsTimer); settingsTimer = null; }
    panicPhone1 = data?.phone1 || ''; panicPhone2 = data?.phone2 || '';
    heartbeatEnabled = !!data?.heartbeatEnabled;
    heartbeatDeadline = data?.heartbeatDeadline || '10:00';
    settingsLoading = false;
  };
  const _onEmergencyPhonesUpdated = () => {
    if (panicTimer) { clearTimeout(panicTimer); panicTimer = null; }
    panicSaving = false;
    toasts.add('Emergency SMS contacts saved');
  };
  const _onConnect = () => socket.emit('getEmergencySettings');

  onMount(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) profile = migrateProfile(JSON.parse(raw)); // DEFAULTS merge + legacy single-contact migration
      else if ($authUser?.displayName) profile.fullName = $authUser.displayName;
    } catch { /* ignore parse errors */ }
    socket.on('emergencySettings', _onEmergencySettings);
    socket.on('emergencyPhonesUpdated', _onEmergencyPhonesUpdated);
    socket.on('connect', _onConnect);
    if (socket.connected) socket.emit('getEmergencySettings');
    // Never strand the skeleton if the response is lost — show empty fields instead.
    settingsTimer = setTimeout(() => { settingsLoading = false; }, 4000);
  });

  onDestroy(() => {
    socket.off('emergencySettings', _onEmergencySettings);
    socket.off('emergencyPhonesUpdated', _onEmergencyPhonesUpdated);
    socket.off('connect', _onConnect);
    clearTimeout(saveTimer); clearTimeout(panicTimer); clearTimeout(settingsTimer);
  });
</script>

<div class="ep-root page-enter">
  <header class="ep-header">
    <button class="ep-back-btn tactile" aria-label="Back to map" onclick={() => push('/')}>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="15 18 9 12 15 6"/></svg>
    </button>
    <div class="ep-header-title"><span class="ep-header-heading">Emergency Profile</span></div>
    <button class="ep-save-btn tactile" class:ep-save-btn--success={saveSuccess} aria-label="Save emergency profile" disabled={saving || hasErrors} onclick={save}>
      {#if saveSuccess}
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
      {:else}
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>
        Save
      {/if}
    </button>
  </header>

  <main class="ep-body">
    <CompletenessMeter {progress} {filledCount} totalFields={TRACKED_FIELDS.length} lastUpdated={lastUpdatedLabel} />

    <div class="ep-warning-card" role="note" aria-label="SOS sharing notice">
      <span class="ep-warning-icon" aria-hidden="true">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="7.86 2 16.14 2 22 7.86 22 16.14 16.14 22 7.86 22 2 16.14 2 7.86 7.86 2"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
      </span>
      <p class="ep-warning-text">
        In an SOS, this profile is shared with your emergency contacts so first responders have your medical information immediately.
      </p>
    </div>

    <EpSection id="section-personal" title="Personal Information" bind:open={openSections.personal}>
      {#snippet icon()}<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>{/snippet}
      <PersonalInfoSection bind:profile {dobError} {todayIso} />
    </EpSection>

    <EpSection id="section-contacts" title="Emergency Contacts" count={profile.emergencyContacts.length} bind:open={openSections.contacts}>
      {#snippet icon()}<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07A19.5 19.5 0 013.07 9.81a19.79 19.79 0 01-3.07-8.63A2 2 0 012 .92h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L6.09 8.91A16 16 0 0015.1 17.9l1.27-1.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 16.92z"/></svg>{/snippet}
      <EmergencyContactsSection bind:profile {contactPhoneErrors} onadd={addContact} onremove={removeContact} />
    </EpSection>

    <EpSection id="section-escalation" title="Panic Relay &amp; Heartbeat" bind:open={openSections.escalation}>
      {#snippet icon()}<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>{/snippet}
      <EscalationSection
        loading={settingsLoading}
        bind:panicPhone1 bind:panicPhone2 {panicSaving}
        bind:heartbeatEnabled bind:heartbeatDeadline
        onsavePanic={savePanicPhones} onsaveHeartbeat={saveHeartbeat}
      />
    </EpSection>

    <EpSection id="section-medical" title="Medical Information" bind:open={openSections.medical}>
      {#snippet icon()}<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>{/snippet}
      <MedicalInfoSection bind:profile {doctorPhoneError} />
    </EpSection>

    <EpSection id="section-responder" title="First Responder Notes" bind:open={openSections.responder}>
      {#snippet icon()}<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="7" width="20" height="14" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/></svg>{/snippet}
      <ResponderNotesSection bind:profile />
    </EpSection>

    <EpSection id="section-qr" title="SOS Share Link" bind:open={openSections.qr}>
      {#snippet icon()}<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="16" y="16" width="2" height="2"/><rect x="20" y="16" width="2" height="2"/><rect x="16" y="20" width="2" height="2"/><rect x="20" y="20" width="2" height="2"/></svg>{/snippet}
      <div class="ep-qr-info-card" role="note">
        <span class="ep-qr-info-icon" aria-hidden="true">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
        </span>
        <div class="ep-qr-info-text">
          <p class="ep-qr-title">Share this page link with emergency contacts</p>
          <p class="ep-qr-body">
            When you trigger an SOS, Kinnect automatically sends a real-time watch link to all visible contacts. That link lets your emergency contacts and first responders track your location live.
          </p>
          {#if shareCode}
            <div class="ep-watch-link-block" aria-label="Example SOS watch link format">
              <span class="ep-watch-link-label">Your SOS link format</span>
              <code class="ep-watch-link-code">/watch/&lt;sos-token&gt;</code>
              <p class="ep-watch-link-note">
                A unique token is generated each time you trigger SOS. Share your Kinnect profile code <strong>{shareCode}</strong> with trusted contacts so they are always in your visible network.
              </p>
            </div>
          {/if}
        </div>
      </div>
    </EpSection>

    <div class="ep-privacy-note" role="note">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
      This information is stored locally on your device only. Kinnect does not upload your medical data to any server.
    </div>

    <button class="ep-save-bottom-btn tactile" class:ep-save-bottom-btn--success={saveSuccess} disabled={saving || hasErrors} aria-label="Save emergency profile" onclick={save}>
      {#if saveSuccess}
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
        Saved
      {:else}
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>
        Save Profile
      {/if}
    </button>
  </main>
</div>

<style>
  /* Page shell only — field/card grammar lives in emergency.css + children. */
  .ep-root {
    min-height: 100dvh;
    display: flex;
    flex-direction: column;
    background: var(--surface-0);
    font-family: var(--font-sans);
    color: var(--text-primary);
  }

  .ep-header {
    position: sticky;
    top: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: calc(env(safe-area-inset-top, 0px) + var(--space-3)) var(--space-4) var(--space-3);
    background: var(--surface-2);
    backdrop-filter: blur(20px) saturate(1.6);
    -webkit-backdrop-filter: blur(20px) saturate(1.6);
    border-bottom: 1px solid var(--border-default);
  }
  .ep-back-btn {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border-radius: var(--radius-md, 10px);
    border: 1px solid var(--border-default);
    background: var(--surface-1);
    color: var(--text-primary);
    cursor: pointer;
    transition: background 150ms var(--ease-out), transform 100ms var(--ease-out);
  }
  .ep-back-btn:hover { background: var(--surface-3); }
  .ep-back-btn:active { transform: scale(0.94); }
  .ep-header-title { flex: 1; min-width: 0; text-align: center; }
  .ep-header-heading { font-size: var(--text-base); font-weight: 600; letter-spacing: -0.01em; color: var(--text-primary); }

  .ep-save-btn,
  .ep-save-bottom-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-1-5);
    border: none;
    background: var(--primary-500);
    color: var(--text-on-primary);
    font-weight: 600;
    font-family: var(--font-sans);
    cursor: pointer;
    box-shadow: var(--shadow-primary);
    transition: background 150ms var(--ease-out), transform 120ms var(--ease-out);
  }
  .ep-save-btn { flex-shrink: 0; min-height: 44px; padding: 0 var(--space-4); border-radius: var(--radius-md, 10px); font-size: var(--text-sm); }
  .ep-save-bottom-btn { width: 100%; height: 52px; gap: var(--space-2); border-radius: var(--radius-button, 14px); font-size: var(--text-base); font-weight: 700; margin-top: var(--space-1); }
  .ep-save-btn:hover:not(:disabled),
  .ep-save-bottom-btn:hover:not(:disabled) { background: var(--primary-600); }
  .ep-save-btn:active:not(:disabled) { transform: scale(0.95); }
  .ep-save-bottom-btn:active:not(:disabled) { transform: scale(0.97); }
  .ep-save-btn:disabled,
  .ep-save-bottom-btn:disabled { opacity: 0.5; cursor: not-allowed; box-shadow: none; }
  .ep-save-btn--success,
  .ep-save-bottom-btn--success {
    background: var(--success-500) !important;
    box-shadow: 0 4px 14px var(--success-500-20) !important;
  }

  .ep-body {
    flex: 1;
    padding: var(--space-5) var(--space-4) calc(env(safe-area-inset-bottom, 0px) + var(--space-8));
    max-width: 640px;
    width: 100%;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  /* ── Warning card ─────────────────────────────────────────────────────── */
  .ep-warning-card {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2-5);
    padding: var(--space-3) var(--space-3-5);
    border-radius: var(--radius-lg, 14px);
    background: color-mix(in oklch, var(--warning-500) 8%, transparent);
    border: 1px solid color-mix(in oklch, var(--warning-500) 22%, transparent);
  }
  .ep-warning-icon { flex-shrink: 0; color: var(--warning-500); margin-top: 1px; }
  .ep-warning-text { font-size: var(--text-sm); color: var(--warning-600); line-height: var(--leading-relaxed, 1.625); }
  :global([data-theme='dark']) .ep-warning-text { color: var(--warning-400); }

  /* ── QR / share card ──────────────────────────────────────────────────── */
  .ep-qr-info-card {
    display: flex;
    gap: var(--space-3);
    padding: var(--space-3-5);
    border-radius: var(--radius-lg, 14px);
    background: var(--primary-500-08);
    border: 1px solid var(--primary-500-12);
  }
  .ep-qr-info-icon { flex-shrink: 0; color: var(--primary-500); margin-top: 1px; }
  .ep-qr-info-text { display: flex; flex-direction: column; gap: var(--space-1-5); }
  .ep-qr-title { font-size: var(--text-sm); font-weight: 600; color: var(--text-primary); }
  .ep-qr-body { font-size: var(--text-sm); color: var(--text-secondary); line-height: var(--leading-relaxed, 1.625); }
  .ep-watch-link-block {
    margin-top: var(--space-2);
    padding: var(--space-2-5) var(--space-3);
    border-radius: var(--radius-md, 10px);
    background: var(--surface-inset, var(--surface-3));
    border: 1px solid var(--border-subtle);
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }
  .ep-watch-link-label {
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-tertiary);
  }
  .ep-watch-link-code { font-family: var(--font-mono, monospace); font-size: var(--text-sm); color: var(--primary-600); background: none; padding: 0; }
  :global([data-theme='dark']) .ep-watch-link-code { color: var(--primary-400); }
  .ep-watch-link-note { font-size: var(--text-xs); color: var(--text-secondary); line-height: var(--leading-relaxed, 1.625); margin-top: 2px; }

  /* ── Privacy note ─────────────────────────────────────────────────────── */
  .ep-privacy-note {
    display: flex;
    align-items: flex-start;
    gap: var(--space-1-5);
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    padding: 0 var(--space-1);
    line-height: var(--leading-relaxed, 1.625);
  }
  .ep-privacy-note svg { flex-shrink: 0; margin-top: 1px; }

  @media (prefers-reduced-motion: reduce) {
    .ep-back-btn, .ep-save-btn, .ep-save-bottom-btn { transition: none; }
    .ep-back-btn:active,
    .ep-save-btn:active:not(:disabled),
    .ep-save-bottom-btn:active:not(:disabled) { transform: none; }
  }
</style>
