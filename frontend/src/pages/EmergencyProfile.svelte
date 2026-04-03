<script>
  import { onMount } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { authUser } from '../lib/stores/auth.js';
  import { toasts } from '../lib/stores/toast.js';

  // ── Auth guard ────────────────────────────────────────────────────────────
  $: if (!$authUser) push('/login');

  // ── Storage key ──────────────────────────────────────────────────────────
  const STORAGE_KEY = 'kinnect_emergency_profile';

  // ── Default profile shape ─────────────────────────────────────────────────
  const DEFAULTS = {
    fullName: '',
    dob: '',
    bloodType: '',
    // Legacy single-contact fields — kept for backward compat, synced from contacts[0] on save
    emergencyName: '',
    emergencyPhone: '',
    // Multiple emergency contacts: [{ name, relation, phone, address }]
    emergencyContacts: [],
    conditions: '',
    allergies: '',
    medications: '',
    doctorName: '',
    doctorPhone: '',
    responderNotes: '',
    language: '',
    updatedAt: null
  };

  const MAX_CONTACTS = 5;

  const BLOOD_TYPES = ['A+', 'A-', 'B+', 'B-', 'AB+', 'AB-', 'O+', 'O-'];

  // ── State ─────────────────────────────────────────────────────────────────
  let profile = { ...DEFAULTS };
  let openSections = { personal: true, contacts: false, medical: false, responder: false, qr: false };
  let saving = false;
  let saveSuccess = false;
  let saveTimer = null;

  // ── Phone validation ──────────────────────────────────────────────────────
  function isValidPhone(val) {
    if (!val) return true; // optional
    return /^[+\d\s\-().]{7,20}$/.test(val.trim());
  }

  // ── Date validation ───────────────────────────────────────────────────────
  function isValidDob(val) {
    if (!val) return true;
    const d = new Date(val);
    if (isNaN(d.getTime())) return false;
    const now = new Date();
    return d < now && d.getFullYear() > 1900;
  }

  // ── Progress computation ──────────────────────────────────────────────────
  const TRACKED_FIELDS = [
    'fullName', 'dob', 'bloodType', 'emergencyContacts',
    'conditions', 'allergies', 'medications', 'doctorName', 'doctorPhone'
  ];

  $: filledCount = TRACKED_FIELDS.filter(f => {
    if (f === 'emergencyContacts') return profile.emergencyContacts?.length > 0;
    return profile[f] && String(profile[f]).trim() !== '';
  }).length;
  $: progress = Math.round((filledCount / TRACKED_FIELDS.length) * 100);
  $: isComplete = progress === 100;

  // ── Validation errors ─────────────────────────────────────────────────────
  $: contactPhoneErrors = (profile.emergencyContacts || []).map(c =>
    c.phone && !isValidPhone(c.phone) ? 'Enter a valid phone number' : ''
  );
  $: doctorPhoneError = profile.doctorPhone && !isValidPhone(profile.doctorPhone)
    ? 'Enter a valid phone number'
    : '';
  $: dobError = profile.dob && !isValidDob(profile.dob)
    ? 'Enter a valid date of birth'
    : '';

  $: hasErrors = !!(doctorPhoneError || dobError || contactPhoneErrors.some(Boolean));

  // ── Load from localStorage ────────────────────────────────────────────────
  onMount(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw);
        profile = { ...DEFAULTS, ...parsed };
        // Migrate legacy single contact → emergencyContacts array (one-time)
        if (!profile.emergencyContacts?.length && (profile.emergencyName || profile.emergencyPhone)) {
          profile.emergencyContacts = [{
            name: profile.emergencyName || '',
            relation: '',
            phone: profile.emergencyPhone || '',
            address: ''
          }];
        }
      } else if ($authUser?.displayName) {
        profile.fullName = $authUser.displayName;
      }
    } catch {
      // ignore parse errors
    }
  });

  // ── Emergency contact helpers ─────────────────────────────────────────────
  function addContact() {
    if (profile.emergencyContacts.length >= MAX_CONTACTS) return;
    profile.emergencyContacts = [...profile.emergencyContacts, { name: '', relation: '', phone: '', address: '' }];
    if (!openSections.contacts) openSections.contacts = true;
  }

  function removeContact(i) {
    profile.emergencyContacts = profile.emergencyContacts.filter((_, idx) => idx !== i);
  }

  // ── Pre-fill name when auth resolves ─────────────────────────────────────
  $: if ($authUser?.displayName && !profile.fullName) {
    profile.fullName = $authUser.displayName;
  }

  // ── Save to localStorage ──────────────────────────────────────────────────
  function save() {
    if (hasErrors) {
      toasts.error('Please fix the errors before saving.');
      return;
    }

    saving = true;
    // Sync legacy single-contact fields from the first contact in the array
    const fc = profile.emergencyContacts?.[0];
    profile.emergencyName  = fc?.name  || '';
    profile.emergencyPhone = fc?.phone || '';

    const updated = { ...profile, updatedAt: new Date().toISOString() };

    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
      profile = updated;
      saveSuccess = true;
      toasts.success('Emergency profile saved.');
      clearTimeout(saveTimer);
      saveTimer = setTimeout(() => { saveSuccess = false; }, 2200);
    } catch {
      toasts.error('Could not save profile. Check device storage.');
    } finally {
      saving = false;
    }
  }

  // ── Auto-resize textarea ──────────────────────────────────────────────────
  function autoResize(node) {
    function resize() {
      node.style.height = 'auto';
      node.style.height = node.scrollHeight + 'px';
    }
    node.addEventListener('input', resize);
    resize();
    return { destroy() { node.removeEventListener('input', resize); } };
  }

  // ── Section toggle ────────────────────────────────────────────────────────
  function toggleSection(key) {
    openSections[key] = !openSections[key];
  }

  // ── Last updated label ────────────────────────────────────────────────────
  function fmtDate(iso) {
    if (!iso) return null;
    try {
      return new Date(iso).toLocaleString([], {
        month: 'short', day: 'numeric', year: 'numeric',
        hour: '2-digit', minute: '2-digit'
      });
    } catch { return null; }
  }

  $: lastUpdatedLabel = fmtDate(profile.updatedAt);

  // ── SOS info text ─────────────────────────────────────────────────────────
  $: shareCode = $authUser?.shareCode ?? '';
</script>

<!-- ════════════════════════════════════════════════════════════════════════════
     TEMPLATE
     ════════════════════════════════════════════════════════════════════════════ -->
<div class="ep-root">

  <!-- ── Header ─────────────────────────────────────────────────────────── -->
  <header class="ep-header" role="banner">
    <button
      class="ep-back-btn"
      aria-label="Back to map"
      on:click={() => push('/')}
    >
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor"
           stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <polyline points="15 18 9 12 15 6"/>
      </svg>
    </button>

    <div class="ep-header-title">
      <span class="ep-header-heading">Emergency Profile</span>
    </div>

    <button
      class="ep-save-btn"
      class:ep-save-btn--success={saveSuccess}
      aria-label="Save emergency profile"
      disabled={saving || hasErrors}
      on:click={save}
    >
      {#if saveSuccess}
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
      {:else}
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/>
          <polyline points="17 21 17 13 7 13 7 21"/>
          <polyline points="7 3 7 8 15 8"/>
        </svg>
        Save
      {/if}
    </button>
  </header>

  <!-- ── Scrollable body ────────────────────────────────────────────────── -->
  <main class="ep-body" role="main">

    <!-- Status badge -->
    <div class="ep-badge-row" aria-live="polite">
      <span class="ep-badge" class:ep-badge--complete={isComplete} class:ep-badge--incomplete={!isComplete}>
        {#if isComplete}
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
          Profile Complete
        {:else}
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
          Incomplete
        {/if}
      </span>

      {#if lastUpdatedLabel}
        <span class="ep-last-updated">Updated {lastUpdatedLabel}</span>
      {/if}
    </div>

    <!-- Progress bar -->
    <div class="ep-progress-wrap" role="group" aria-label="Profile completion">
      <div class="ep-progress-header">
        <span class="ep-progress-label">Profile completion</span>
        <span class="ep-progress-pct">{progress}%</span>
      </div>
      <div class="ep-progress-track" role="progressbar" aria-valuenow={progress} aria-valuemin="0" aria-valuemax="100">
        <div class="ep-progress-fill" style="width: {progress}%"></div>
      </div>
    </div>

    <!-- Warning banner -->
    <div class="ep-warning-card" role="note" aria-label="SOS sharing notice">
      <span class="ep-warning-icon" aria-hidden="true">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="7.86 2 16.14 2 22 7.86 22 16.14 16.14 22 7.86 22 2 16.14 2 7.86 7.86 2"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
      </span>
      <p class="ep-warning-text">
        In an SOS, this profile is shared with your emergency contacts so first responders have your medical information immediately.
      </p>
    </div>

    <!-- ── Section 1: Personal Info ─────────────────────────────────────── -->
    <section class="ep-section" aria-labelledby="section-personal-heading">
      <button
        class="ep-section-header"
        id="section-personal-btn"
        aria-expanded={openSections.personal}
        aria-controls="section-personal-body"
        on:click={() => toggleSection('personal')}
      >
        <span class="ep-section-icon" aria-hidden="true">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
            <circle cx="12" cy="7" r="4"/>
          </svg>
        </span>
        <span id="section-personal-heading" class="ep-section-title">Personal Information</span>
        <span class="ep-section-chevron" class:ep-section-chevron--open={openSections.personal} aria-hidden="true">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </span>
      </button>

      {#if openSections.personal}
        <div class="ep-section-body" id="section-personal-body" role="region" aria-labelledby="section-personal-heading">

          <div class="ep-field">
            <label for="ep-fullname" class="ep-label">
              Full Name <span class="ep-required" aria-label="required">*</span>
            </label>
            <input
              id="ep-fullname"
              type="text"
              class="ep-input"
              bind:value={profile.fullName}
              autocomplete="name"
              aria-required="true"
              aria-describedby="ep-fullname-desc"
              placeholder="Your legal full name"
            />
            <span id="ep-fullname-desc" class="ep-field-hint">As it appears on your ID</span>
          </div>

          <div class="ep-field">
            <label for="ep-dob" class="ep-label">Date of Birth</label>
            <input
              id="ep-dob"
              type="date"
              class="ep-input"
              class:ep-input--error={dobError}
              bind:value={profile.dob}
              max={new Date().toISOString().split('T')[0]}
              aria-describedby={dobError ? 'ep-dob-error' : undefined}
            />
            {#if dobError}
              <span id="ep-dob-error" class="ep-field-error" role="alert">{dobError}</span>
            {/if}
          </div>

          <div class="ep-field">
            <span class="ep-label" id="ep-bloodtype-label">Blood Type</span>
            <div
              class="ep-blood-grid"
              role="group"
              aria-labelledby="ep-bloodtype-label"
            >
              {#each BLOOD_TYPES as bt}
                <button
                  type="button"
                  class="ep-blood-pill"
                  class:ep-blood-pill--selected={profile.bloodType === bt}
                  aria-pressed={profile.bloodType === bt}
                  on:click={() => { profile.bloodType = profile.bloodType === bt ? '' : bt; }}
                >
                  {bt}
                </button>
              {/each}
            </div>
          </div>

        </div>
      {/if}
    </section>

    <!-- ── Section 2: Emergency Contacts ────────────────────────────────── -->
    <section class="ep-section" aria-labelledby="section-contacts-heading">
      <button
        class="ep-section-header"
        id="section-contacts-btn"
        aria-expanded={openSections.contacts}
        aria-controls="section-contacts-body"
        on:click={() => toggleSection('contacts')}
      >
        <span class="ep-section-icon" aria-hidden="true">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07A19.5 19.5 0 013.07 9.81a19.79 19.79 0 01-3.07-8.63A2 2 0 012 .92h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L6.09 8.91A16 16 0 0015.1 17.9l1.27-1.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 16.92z"/>
          </svg>
        </span>
        <span id="section-contacts-heading" class="ep-section-title">Emergency Contacts</span>
        {#if profile.emergencyContacts.length > 0}
          <span class="ep-section-count" aria-label="{profile.emergencyContacts.length} contacts">{profile.emergencyContacts.length}</span>
        {/if}
        <span class="ep-section-chevron" class:ep-section-chevron--open={openSections.contacts} aria-hidden="true">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </span>
      </button>

      {#if openSections.contacts}
        <div class="ep-section-body" id="section-contacts-body" role="region" aria-labelledby="section-contacts-heading">

          {#each profile.emergencyContacts as contact, i}
            <div class="ep-contact-card">
              <div class="ep-contact-header">
                <span class="ep-contact-num">Contact {i + 1}</span>
                <button
                  class="ep-remove-btn"
                  type="button"
                  aria-label="Remove contact {i + 1}"
                  on:click={() => removeContact(i)}
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                       stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                    <line x1="18" y1="6" x2="6" y2="18"/>
                    <line x1="6" y1="6" x2="18" y2="18"/>
                  </svg>
                </button>
              </div>

              <div class="ep-field-row">
                <div class="ep-field ep-field--half">
                  <label for="ep-contact-{i}-name" class="ep-label">Full Name</label>
                  <input
                    id="ep-contact-{i}-name"
                    type="text"
                    class="ep-input"
                    bind:value={contact.name}
                    placeholder="Jane Smith"
                    autocomplete="off"
                  />
                </div>
                <div class="ep-field ep-field--half">
                  <label for="ep-contact-{i}-relation" class="ep-label">Relation</label>
                  <input
                    id="ep-contact-{i}-relation"
                    type="text"
                    class="ep-input"
                    bind:value={contact.relation}
                    placeholder="e.g. Spouse, Parent, Friend"
                    autocomplete="off"
                  />
                </div>
              </div>

              <div class="ep-field-row">
                <div class="ep-field ep-field--half">
                  <label for="ep-contact-{i}-phone" class="ep-label">Phone</label>
                  <input
                    id="ep-contact-{i}-phone"
                    type="tel"
                    class="ep-input"
                    class:ep-input--error={contactPhoneErrors[i]}
                    bind:value={contact.phone}
                    placeholder="+1 555 000 0000"
                    autocomplete="tel"
                    aria-describedby={contactPhoneErrors[i] ? `ep-contact-${i}-phone-error` : undefined}
                  />
                  {#if contactPhoneErrors[i]}
                    <span id="ep-contact-{i}-phone-error" class="ep-field-error" role="alert">{contactPhoneErrors[i]}</span>
                  {/if}
                </div>
                <div class="ep-field ep-field--half">
                  <label for="ep-contact-{i}-address" class="ep-label">
                    Address
                    <span class="ep-optional"> (optional)</span>
                  </label>
                  <input
                    id="ep-contact-{i}-address"
                    type="text"
                    class="ep-input"
                    bind:value={contact.address}
                    placeholder="City, State or full address"
                    autocomplete="off"
                  />
                </div>
              </div>
            </div>
          {/each}

          {#if profile.emergencyContacts.length === 0}
            <p class="ep-empty-hint">
              No emergency contacts added yet. First responders will use these to notify your family immediately.
            </p>
          {/if}

          {#if profile.emergencyContacts.length < MAX_CONTACTS}
            <button class="ep-add-contact-btn" type="button" on:click={addContact}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                   stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <line x1="12" y1="5" x2="12" y2="19"/>
                <line x1="5" y1="12" x2="19" y2="12"/>
              </svg>
              Add Emergency Contact
            </button>
          {/if}

        </div>
      {/if}
    </section>

    <!-- ── Section 3: Medical Info ───────────────────────────────────────── -->
    <section class="ep-section" aria-labelledby="section-medical-heading">
      <button
        class="ep-section-header"
        id="section-medical-btn"
        aria-expanded={openSections.medical}
        aria-controls="section-medical-body"
        on:click={() => toggleSection('medical')}
      >
        <span class="ep-section-icon" aria-hidden="true">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
          </svg>
        </span>
        <span id="section-medical-heading" class="ep-section-title">Medical Information</span>
        <span class="ep-section-chevron" class:ep-section-chevron--open={openSections.medical} aria-hidden="true">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </span>
      </button>

      {#if openSections.medical}
        <div class="ep-section-body" id="section-medical-body" role="region" aria-labelledby="section-medical-heading">

          <div class="ep-field">
            <label for="ep-conditions" class="ep-label">Medical Conditions</label>
            <textarea
              id="ep-conditions"
              class="ep-textarea"
              bind:value={profile.conditions}
              use:autoResize
              placeholder="e.g. Type 2 Diabetes, Hypertension, Asthma..."
              rows="2"
              aria-describedby="ep-conditions-desc"
            ></textarea>
            <span id="ep-conditions-desc" class="ep-field-hint">List all diagnosed conditions</span>
          </div>

          <div class="ep-field">
            <label for="ep-allergies" class="ep-label">Allergies</label>
            <textarea
              id="ep-allergies"
              class="ep-textarea"
              bind:value={profile.allergies}
              use:autoResize
              placeholder="e.g. Penicillin, Peanuts, Latex..."
              rows="2"
              aria-describedby="ep-allergies-desc"
            ></textarea>
            <span id="ep-allergies-desc" class="ep-field-hint">Include drug, food, and environmental allergies</span>
          </div>

          <div class="ep-field">
            <label for="ep-medications" class="ep-label">Current Medications</label>
            <textarea
              id="ep-medications"
              class="ep-textarea"
              bind:value={profile.medications}
              use:autoResize
              placeholder="e.g. Metformin 500mg twice daily, Lisinopril 10mg..."
              rows="2"
              aria-describedby="ep-medications-desc"
            ></textarea>
            <span id="ep-medications-desc" class="ep-field-hint">Include dosage and frequency if known</span>
          </div>

          <div class="ep-field-row">
            <div class="ep-field ep-field--half">
              <label for="ep-doctor-name" class="ep-label">Primary Doctor</label>
              <input
                id="ep-doctor-name"
                type="text"
                class="ep-input"
                bind:value={profile.doctorName}
                autocomplete="off"
                placeholder="Dr. Name"
              />
            </div>
            <div class="ep-field ep-field--half">
              <label for="ep-doctor-phone" class="ep-label">Doctor's Phone</label>
              <input
                id="ep-doctor-phone"
                type="tel"
                class="ep-input"
                class:ep-input--error={doctorPhoneError}
                bind:value={profile.doctorPhone}
                autocomplete="tel"
                placeholder="+1 555 000 0000"
                aria-describedby={doctorPhoneError ? 'ep-doctor-phone-error' : undefined}
              />
              {#if doctorPhoneError}
                <span id="ep-doctor-phone-error" class="ep-field-error" role="alert">{doctorPhoneError}</span>
              {/if}
            </div>
          </div>

        </div>
      {/if}
    </section>

    <!-- ── Section 3: First Responder Notes ─────────────────────────────── -->
    <section class="ep-section" aria-labelledby="section-responder-heading">
      <button
        class="ep-section-header"
        id="section-responder-btn"
        aria-expanded={openSections.responder}
        aria-controls="section-responder-body"
        on:click={() => toggleSection('responder')}
      >
        <span class="ep-section-icon" aria-hidden="true">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="7" width="20" height="14" rx="2" ry="2"/>
            <path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/>
          </svg>
        </span>
        <span id="section-responder-heading" class="ep-section-title">First Responder Notes</span>
        <span class="ep-section-chevron" class:ep-section-chevron--open={openSections.responder} aria-hidden="true">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </span>
      </button>

      {#if openSections.responder}
        <div class="ep-section-body" id="section-responder-body" role="region" aria-labelledby="section-responder-heading">

          <div class="ep-field">
            <label for="ep-language" class="ep-label">Preferred Language</label>
            <input
              id="ep-language"
              type="text"
              class="ep-input"
              bind:value={profile.language}
              placeholder="e.g. Spanish, Mandarin, Hindi"
              aria-describedby="ep-language-desc"
            />
            <span id="ep-language-desc" class="ep-field-hint">Language first responders should speak with you</span>
          </div>

          <div class="ep-field">
            <label for="ep-responder-notes" class="ep-label">Additional Notes</label>
            <textarea
              id="ep-responder-notes"
              class="ep-textarea"
              bind:value={profile.responderNotes}
              use:autoResize
              placeholder="Any other critical information for first responders — implanted devices, prosthetics, mental health notes, do-not-resuscitate preferences, religious considerations..."
              rows="4"
            ></textarea>
          </div>

        </div>
      {/if}
    </section>

    <!-- ── Section 4: QR / Share info ───────────────────────────────────── -->
    <section class="ep-section" aria-labelledby="section-qr-heading">
      <button
        class="ep-section-header"
        id="section-qr-btn"
        aria-expanded={openSections.qr}
        aria-controls="section-qr-body"
        on:click={() => toggleSection('qr')}
      >
        <span class="ep-section-icon" aria-hidden="true">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="3" width="7" height="7"/>
            <rect x="14" y="3" width="7" height="7"/>
            <rect x="3" y="14" width="7" height="7"/>
            <rect x="16" y="16" width="2" height="2"/>
            <rect x="20" y="16" width="2" height="2"/>
            <rect x="16" y="20" width="2" height="2"/>
            <rect x="20" y="20" width="2" height="2"/>
          </svg>
        </span>
        <span id="section-qr-heading" class="ep-section-title">SOS Share Link</span>
        <span class="ep-section-chevron" class:ep-section-chevron--open={openSections.qr} aria-hidden="true">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
               stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </span>
      </button>

      {#if openSections.qr}
        <div class="ep-section-body" id="section-qr-body" role="region" aria-labelledby="section-qr-heading">

          <div class="ep-qr-info-card" role="note">
            <span class="ep-qr-info-icon" aria-hidden="true">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                   stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="10"/>
                <line x1="12" y1="16" x2="12" y2="12"/>
                <line x1="12" y1="8" x2="12.01" y2="8"/>
              </svg>
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

        </div>
      {/if}
    </section>

    <!-- Privacy note -->
    <div class="ep-privacy-note" role="note">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
           stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
        <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
      This information is stored locally on your device only. Kinnect does not upload your medical data to any server.
    </div>

    <!-- Bottom save button (convenience) -->
    <button
      class="ep-save-bottom-btn"
      class:ep-save-bottom-btn--success={saveSuccess}
      disabled={saving || hasErrors}
      aria-label="Save emergency profile"
      on:click={save}
    >
      {#if saveSuccess}
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
        Saved
      {:else}
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/>
          <polyline points="17 21 17 13 7 13 7 21"/>
          <polyline points="7 3 7 8 15 8"/>
        </svg>
        Save Profile
      {/if}
    </button>

  </main>
</div>

<!-- ════════════════════════════════════════════════════════════════════════════
     STYLES
     ════════════════════════════════════════════════════════════════════════════ -->
<style>
  /* ── Root / layout ────────────────────────────────────────────────────── */
  .ep-root {
    min-height: 100dvh;
    display: flex;
    flex-direction: column;
    background: var(--surface-0, #f8fafc);
    font-family: var(--font-sans, 'Inter', system-ui, sans-serif);
    color: var(--text-primary, #0f172a);
  }

  /* ── Header ───────────────────────────────────────────────────────────── */
  .ep-header {
    position: sticky;
    top: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: env(safe-area-inset-top, 0px) 16px 0;
    padding-top: calc(env(safe-area-inset-top, 0px) + 12px);
    padding-bottom: 12px;
    background: var(--surface-2, rgba(255,255,255,0.92));
    backdrop-filter: blur(20px) saturate(1.6);
    -webkit-backdrop-filter: blur(20px) saturate(1.6);
    border-bottom: 1px solid var(--border-default, rgba(15,23,42,0.10));
  }

  .ep-back-btn {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 38px;
    height: 38px;
    border-radius: 10px;
    border: 1px solid var(--border-default, rgba(15,23,42,0.10));
    background: var(--surface-1, rgba(255,255,255,0.80));
    color: var(--text-primary, #0f172a);
    cursor: pointer;
    transition: background 150ms ease, transform 100ms ease;
  }
  .ep-back-btn:hover { background: var(--gray-100, #f1f5f9); }
  .ep-back-btn:active { transform: scale(0.94); }

  .ep-header-title {
    flex: 1;
    min-width: 0;
    text-align: center;
  }
  .ep-header-heading {
    font-size: var(--text-base, 1rem);
    font-weight: 600;
    letter-spacing: -0.01em;
    color: var(--text-primary, #0f172a);
  }

  .ep-save-btn {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 5px;
    height: 36px;
    padding: 0 14px;
    border-radius: 10px;
    border: none;
    background: var(--primary-500, #6366f1);
    color: #fff;
    font-size: var(--text-sm, 0.875rem);
    font-weight: 600;
    cursor: pointer;
    transition: background 150ms ease, transform 120ms ease, box-shadow 150ms ease;
    box-shadow: var(--shadow-primary, 0 4px 14px rgba(79,70,229,0.30));
  }
  .ep-save-btn:hover:not(:disabled) {
    background: var(--primary-600, #4f46e5);
  }
  .ep-save-btn:active:not(:disabled) { transform: scale(0.95); }
  .ep-save-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    box-shadow: none;
  }
  .ep-save-btn--success {
    background: var(--success-500, #10b981) !important;
    box-shadow: 0 4px 14px rgba(16, 185, 129, 0.30) !important;
  }

  /* ── Body ─────────────────────────────────────────────────────────────── */
  .ep-body {
    flex: 1;
    padding: 20px 16px calc(env(safe-area-inset-bottom, 0px) + 32px);
    max-width: 640px;
    width: 100%;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  /* ── Status badge row ─────────────────────────────────────────────────── */
  .ep-badge-row {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }

  .ep-badge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 4px 11px;
    border-radius: 9999px;
    font-size: var(--text-xs, 0.75rem);
    font-weight: 700;
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }
  .ep-badge--complete {
    background: rgba(16, 185, 129, 0.12);
    color: var(--success-600, #059669);
    border: 1px solid rgba(16, 185, 129, 0.25);
  }
  .ep-badge--incomplete {
    background: rgba(245, 158, 11, 0.10);
    color: var(--warning-600, #d97706);
    border: 1px solid rgba(245, 158, 11, 0.22);
  }

  .ep-last-updated {
    font-size: var(--text-xs, 0.75rem);
    color: var(--text-tertiary, #94a3b8);
  }

  /* ── Progress bar ─────────────────────────────────────────────────────── */
  .ep-progress-wrap {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .ep-progress-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .ep-progress-label {
    font-size: var(--text-xs, 0.75rem);
    font-weight: 500;
    color: var(--text-secondary, #475569);
  }
  .ep-progress-pct {
    font-size: var(--text-xs, 0.75rem);
    font-weight: 700;
    color: var(--primary-500, #6366f1);
    font-variant-numeric: tabular-nums;
  }
  .ep-progress-track {
    height: 6px;
    border-radius: 9999px;
    background: var(--gray-200, #e2e8f0);
    overflow: hidden;
  }
  .ep-progress-fill {
    height: 100%;
    border-radius: 9999px;
    background: linear-gradient(90deg, var(--primary-500, #6366f1), var(--primary-400, #818cf8));
    transition: width 400ms cubic-bezier(0.16, 1, 0.3, 1);
  }

  /* ── Warning card ─────────────────────────────────────────────────────── */
  .ep-warning-card {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 12px 14px;
    border-radius: var(--radius-lg, 14px);
    background: rgba(245, 158, 11, 0.08);
    border: 1px solid rgba(245, 158, 11, 0.22);
  }
  .ep-warning-icon {
    flex-shrink: 0;
    color: var(--warning-500, #f59e0b);
    margin-top: 1px;
  }
  .ep-warning-text {
    font-size: var(--text-sm, 0.875rem);
    color: var(--warning-600, #d97706);
    line-height: var(--leading-relaxed, 1.625);
  }

  /* ── Section card ─────────────────────────────────────────────────────── */
  .ep-section {
    border-radius: var(--radius-xl, 20px);
    background: var(--surface-2, rgba(255,255,255,0.92));
    border: 1px solid var(--border-default, rgba(15,23,42,0.10));
    box-shadow: var(--shadow-sm, 0 2px 6px rgba(0,0,0,0.06));
    overflow: hidden;
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
  }

  .ep-section-header {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 15px 16px;
    border: none;
    background: transparent;
    cursor: pointer;
    text-align: left;
    transition: background 120ms ease;
  }
  .ep-section-header:hover {
    background: rgba(99, 102, 241, 0.04);
  }

  .ep-section-icon {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 34px;
    height: 34px;
    border-radius: 9px;
    background: rgba(99, 102, 241, 0.10);
    color: var(--primary-500, #6366f1);
  }

  .ep-section-title {
    flex: 1;
    font-size: var(--text-base, 1rem);
    font-weight: 600;
    color: var(--text-primary, #0f172a);
    letter-spacing: -0.01em;
  }

  .ep-section-chevron {
    flex-shrink: 0;
    color: var(--text-tertiary, #94a3b8);
    display: flex;
    align-items: center;
    transition: transform 220ms cubic-bezier(0.16, 1, 0.3, 1);
  }
  .ep-section-chevron--open {
    transform: rotate(180deg);
  }

  .ep-section-body {
    padding: 4px 16px 18px;
    display: flex;
    flex-direction: column;
    gap: 14px;
    border-top: 1px solid var(--border-subtle, rgba(15,23,42,0.06));
  }

  /* ── Field ────────────────────────────────────────────────────────────── */
  .ep-field {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }
  .ep-field-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  .ep-field--half {
    /* inherits from grid */
  }

  .ep-label {
    font-size: var(--text-sm, 0.875rem);
    font-weight: 500;
    color: var(--text-secondary, #475569);
    display: block;
  }

  .ep-required {
    color: var(--danger-500, #ef4444);
    margin-left: 2px;
  }

  .ep-input {
    width: 100%;
    height: 42px;
    padding: 0 12px;
    border-radius: var(--radius-input, 10px);
    border: 1px solid var(--border-default, rgba(15,23,42,0.10));
    background: var(--surface-3, rgba(255,255,255,0.60));
    font-size: var(--text-base, 1rem);
    color: var(--text-primary, #0f172a);
    font-family: var(--font-sans, 'Inter', system-ui, sans-serif);
    transition: border-color 150ms ease, box-shadow 150ms ease;
    appearance: none;
    -webkit-appearance: none;
  }
  .ep-input::placeholder { color: var(--text-tertiary, #94a3b8); }
  .ep-input:focus {
    outline: none;
    border-color: var(--primary-500, #6366f1);
    box-shadow: 0 0 0 3px rgba(99,102,241,0.12);
  }
  .ep-input--error {
    border-color: var(--danger-500, #ef4444);
  }
  .ep-input--error:focus {
    border-color: var(--danger-500, #ef4444);
    box-shadow: 0 0 0 3px rgba(239,68,68,0.12);
  }

  .ep-textarea {
    width: 100%;
    min-height: 72px;
    padding: 10px 12px;
    border-radius: var(--radius-input, 10px);
    border: 1px solid var(--border-default, rgba(15,23,42,0.10));
    background: var(--surface-3, rgba(255,255,255,0.60));
    font-size: var(--text-base, 1rem);
    color: var(--text-primary, #0f172a);
    font-family: var(--font-sans, 'Inter', system-ui, sans-serif);
    line-height: var(--leading-relaxed, 1.625);
    resize: none;
    overflow: hidden;
    transition: border-color 150ms ease, box-shadow 150ms ease;
  }
  .ep-textarea::placeholder { color: var(--text-tertiary, #94a3b8); }
  .ep-textarea:focus {
    outline: none;
    border-color: var(--primary-500, #6366f1);
    box-shadow: 0 0 0 3px rgba(99,102,241,0.12);
  }

  .ep-field-hint {
    font-size: var(--text-xs, 0.75rem);
    color: var(--text-tertiary, #94a3b8);
  }
  .ep-field-error {
    font-size: var(--text-xs, 0.75rem);
    color: var(--danger-500, #ef4444);
    font-weight: 500;
  }

  /* ── Blood type pill grid ─────────────────────────────────────────────── */
  .ep-blood-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 7px;
    margin-top: 2px;
  }
  .ep-blood-pill {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    height: 36px;
    min-width: 54px;
    padding: 0 12px;
    border-radius: 9999px;
    border: 1.5px solid var(--border-default, rgba(15,23,42,0.10));
    background: var(--surface-3, rgba(255,255,255,0.60));
    font-size: var(--text-sm, 0.875rem);
    font-weight: 600;
    color: var(--text-secondary, #475569);
    cursor: pointer;
    transition: border-color 150ms ease, background 150ms ease, color 150ms ease, transform 100ms ease;
  }
  .ep-blood-pill:hover {
    border-color: var(--primary-400, #818cf8);
    color: var(--primary-500, #6366f1);
  }
  .ep-blood-pill:active { transform: scale(0.93); }
  .ep-blood-pill--selected {
    border-color: var(--danger-500, #ef4444);
    background: rgba(239, 68, 68, 0.08);
    color: var(--danger-600, #dc2626);
  }
  .ep-blood-pill--selected:hover {
    border-color: var(--danger-500, #ef4444);
    color: var(--danger-600, #dc2626);
  }

  /* ── QR / share card ──────────────────────────────────────────────────── */
  .ep-qr-info-card {
    display: flex;
    gap: 12px;
    padding: 14px;
    border-radius: var(--radius-lg, 14px);
    background: rgba(99, 102, 241, 0.05);
    border: 1px solid rgba(99, 102, 241, 0.15);
  }
  .ep-qr-info-icon {
    flex-shrink: 0;
    color: var(--primary-500, #6366f1);
    margin-top: 1px;
  }
  .ep-qr-info-text {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .ep-qr-title {
    font-size: var(--text-sm, 0.875rem);
    font-weight: 600;
    color: var(--text-primary, #0f172a);
  }
  .ep-qr-body {
    font-size: var(--text-sm, 0.875rem);
    color: var(--text-secondary, #475569);
    line-height: var(--leading-relaxed, 1.625);
  }
  .ep-watch-link-block {
    margin-top: 8px;
    padding: 10px 12px;
    border-radius: var(--radius-md, 10px);
    background: var(--surface-inset, #f1f5f9);
    border: 1px solid var(--border-subtle, rgba(15,23,42,0.06));
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .ep-watch-link-label {
    font-size: var(--text-xs, 0.75rem);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-tertiary, #94a3b8);
  }
  .ep-watch-link-code {
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    font-size: var(--text-sm, 0.875rem);
    color: var(--primary-600, #4f46e5);
    background: none;
    padding: 0;
  }
  .ep-watch-link-note {
    font-size: var(--text-xs, 0.75rem);
    color: var(--text-secondary, #475569);
    line-height: var(--leading-relaxed, 1.625);
    margin-top: 2px;
  }

  /* ── Privacy note ─────────────────────────────────────────────────────── */
  .ep-privacy-note {
    display: flex;
    align-items: flex-start;
    gap: 7px;
    font-size: var(--text-xs, 0.75rem);
    color: var(--text-tertiary, #94a3b8);
    padding: 0 4px;
    line-height: var(--leading-relaxed, 1.625);
  }
  .ep-privacy-note svg {
    flex-shrink: 0;
    margin-top: 1px;
  }

  /* ── Bottom save button ───────────────────────────────────────────────── */
  .ep-save-bottom-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    width: 100%;
    height: 52px;
    border-radius: var(--radius-button, 14px);
    border: none;
    background: var(--primary-500, #6366f1);
    color: #fff;
    font-size: var(--text-base, 1rem);
    font-weight: 700;
    font-family: var(--font-sans, 'Inter', system-ui, sans-serif);
    cursor: pointer;
    transition: background 150ms ease, transform 120ms ease, box-shadow 150ms ease;
    box-shadow: var(--shadow-primary, 0 4px 14px rgba(79,70,229,0.30));
    margin-top: 4px;
  }
  .ep-save-bottom-btn:hover:not(:disabled) {
    background: var(--primary-600, #4f46e5);
    box-shadow: 0 6px 20px rgba(79,70,229,0.38);
  }
  .ep-save-bottom-btn:active:not(:disabled) { transform: scale(0.97); }
  .ep-save-bottom-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    box-shadow: none;
  }
  .ep-save-bottom-btn--success {
    background: var(--success-500, #10b981) !important;
    box-shadow: 0 4px 14px rgba(16,185,129,0.30) !important;
  }

  /* ── Section count badge ─────────────────────────────────────────────── */
  .ep-section-count {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 20px;
    height: 20px;
    padding: 0 6px;
    border-radius: 9999px;
    background: rgba(99,102,241,0.12);
    color: var(--primary-500, #6366f1);
    font-size: var(--text-xs, 0.75rem);
    font-weight: 700;
    margin-left: auto;
  }

  /* ── Emergency contact card ───────────────────────────────────────────── */
  .ep-contact-card {
    border: 1px solid var(--border-default, rgba(15,23,42,0.10));
    border-radius: var(--radius-lg, 14px);
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    background: var(--surface-3, rgba(255,255,255,0.60));
  }

  .ep-contact-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .ep-contact-num {
    font-size: var(--text-sm, 0.875rem);
    font-weight: 600;
    color: var(--text-secondary, #475569);
  }

  .ep-remove-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border: 1px solid rgba(239,68,68,0.20);
    border-radius: 7px;
    background: rgba(239,68,68,0.05);
    color: var(--danger-500, #ef4444);
    cursor: pointer;
    transition: background 150ms ease;
  }
  .ep-remove-btn:hover { background: rgba(239,68,68,0.14); }

  .ep-add-contact-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    width: 100%;
    height: 42px;
    border: 1.5px dashed var(--border-default, rgba(15,23,42,0.15));
    border-radius: var(--radius-input, 10px);
    background: transparent;
    color: var(--primary-500, #6366f1);
    font-size: var(--text-sm, 0.875rem);
    font-weight: 600;
    font-family: var(--font-sans, 'Inter', system-ui, sans-serif);
    cursor: pointer;
    transition: background 150ms ease, border-color 150ms ease;
  }
  .ep-add-contact-btn:hover {
    background: rgba(99,102,241,0.06);
    border-color: var(--primary-400, #818cf8);
  }

  .ep-empty-hint {
    font-size: var(--text-sm, 0.875rem);
    color: var(--text-tertiary, #94a3b8);
    text-align: center;
    padding: 8px 4px;
    line-height: 1.6;
  }

  .ep-optional {
    color: var(--text-tertiary, #94a3b8);
    font-weight: 400;
  }

  /* ── Responsive ───────────────────────────────────────────────────────── */
  @media (max-width: 400px) {
    .ep-field-row {
      grid-template-columns: 1fr;
    }
  }

  @media (prefers-color-scheme: dark) {
    .ep-root {
      background: #0b0f1a;
    }
    .ep-header {
      background: rgba(15, 23, 42, 0.92);
      border-color: rgba(255,255,255,0.08);
    }
    .ep-section {
      background: rgba(30, 41, 59, 0.80);
      border-color: rgba(255,255,255,0.08);
    }
    .ep-section-header:hover { background: rgba(99, 102, 241, 0.08); }
    .ep-input,
    .ep-textarea {
      background: rgba(15, 23, 42, 0.60);
      border-color: rgba(255,255,255,0.10);
      color: #f1f5f9;
    }
    .ep-input:focus,
    .ep-textarea:focus {
      border-color: var(--primary-400, #818cf8);
    }
    .ep-blood-pill {
      background: rgba(15, 23, 42, 0.60);
      border-color: rgba(255,255,255,0.12);
      color: #94a3b8;
    }
    .ep-blood-pill--selected {
      background: rgba(239,68,68,0.15);
    }
    .ep-back-btn {
      background: rgba(30,41,59,0.80);
      border-color: rgba(255,255,255,0.10);
      color: #f1f5f9;
    }
    .ep-watch-link-block {
      background: rgba(15, 23, 42, 0.60);
    }
    .ep-warning-card {
      background: rgba(245,158,11,0.10);
    }
    .ep-qr-info-card {
      background: rgba(99,102,241,0.08);
    }
    .ep-header-heading,
    .ep-section-title,
    .ep-qr-title {
      color: #f1f5f9;
    }
    .ep-progress-track {
      background: rgba(255,255,255,0.10);
    }
    .ep-contact-card {
      background: rgba(15,23,42,0.50);
      border-color: rgba(255,255,255,0.08);
    }
    .ep-remove-btn {
      border-color: rgba(239,68,68,0.25);
      background: rgba(239,68,68,0.08);
    }
    .ep-add-contact-btn {
      border-color: rgba(255,255,255,0.12);
    }
  }
</style>
