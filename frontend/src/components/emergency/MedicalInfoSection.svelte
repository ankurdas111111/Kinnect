<script>
  /**
   * MedicalInfoSection — conditions, allergies (critical card), medications,
   * doctor name + phone. The allergy/blood tint is static hierarchy, not
   * motion. Field ids preserved: ep-conditions, ep-allergies, ep-medications,
   * ep-doctor-name, ep-doctor-phone.
   */
  import { autoResize } from './epActions.js';

  /** @type {{ profile: any, doctorPhoneError?: string }} */
  let { profile = $bindable(), doctorPhoneError = '' } = $props();
</script>

<!-- Conditions — medical card -->
<div class="ep-med-card">
  <label for="ep-conditions" class="ep-med-card-title">Medical Conditions</label>
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

<!-- Allergies — critical medical card -->
<div class="ep-med-card ep-med-card--allergy">
  <label for="ep-allergies" class="ep-med-card-title">
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
         stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <polygon points="7.86 2 16.14 2 22 7.86 22 16.14 16.14 22 7.86 22 2 16.14 2 7.86 7.86 2"/>
      <line x1="12" y1="8" x2="12" y2="12"/>
      <line x1="12" y1="16" x2="12.01" y2="16"/>
    </svg>
    Allergies
  </label>
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
      inputmode="tel"
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
