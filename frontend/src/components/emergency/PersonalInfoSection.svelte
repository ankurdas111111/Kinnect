<script>
  /**
   * PersonalInfoSection — full name, date of birth, blood type.
   * Presentational: `profile` is bindable; validation strings are passed in.
   * Field ids preserved IDENTICALLY (ep-fullname, ep-dob, ep-bloodtype-label)
   * so screen-reader / autofill muscle memory survives the split.
   */
  import { BLOOD_TYPES } from '../../lib/emergencyProfile.js';
  import { haptics } from '../../lib/haptics.js';

  /** @type {{ profile: any, dobError?: string, todayIso: string }} */
  let { profile = $bindable(), dobError = '', todayIso } = $props();

  function pickBlood(bt) {
    profile.bloodType = profile.bloodType === bt ? '' : bt;
    haptics.tap();
  }
</script>

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
    max={todayIso}
    aria-describedby={dobError ? 'ep-dob-error' : undefined}
  />
  {#if dobError}
    <span id="ep-dob-error" class="ep-field-error" role="alert">{dobError}</span>
  {/if}
</div>

<!-- Blood type — prominent medical card -->
<div class="ep-med-card ep-med-card--blood">
  <span class="ep-med-card-title" id="ep-bloodtype-label">
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
         stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="M12 2.69l5.66 5.66a8 8 0 1 1-11.31 0z"/>
    </svg>
    Blood Type
  </span>
  <div class="ep-blood-grid" role="group" aria-labelledby="ep-bloodtype-label">
    {#each BLOOD_TYPES as bt}
      <button
        type="button"
        class="ep-blood-pill"
        class:ep-blood-pill--selected={profile.bloodType === bt}
        aria-pressed={profile.bloodType === bt}
        onclick={() => pickBlood(bt)}
      >
        {bt}
      </button>
    {/each}
  </div>
</div>
