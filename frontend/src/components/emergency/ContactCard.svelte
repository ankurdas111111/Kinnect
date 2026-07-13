<script>
  /**
   * ContactCard — one emergency contact (name / relation / phone / address).
   * Presentational: `contact` is bindable; index drives the identical field
   * ids (ep-contact-{i}-name, -relation, -phone, -address) and remove callback.
   */
  import { haptics } from '../../lib/haptics.js';

  /** @type {{ contact: any, index: number, phoneError?: string, onremove: (i: number) => void }} */
  let { contact = $bindable(), index, phoneError = '', onremove } = $props();
</script>

<div class="ep-contact-card">
  <div class="ep-contact-header">
    <span class="ep-contact-num">Contact {index + 1}</span>
    <button
      class="ep-remove-btn"
      type="button"
      aria-label="Remove contact {index + 1}"
      onclick={() => { haptics.tap(); onremove(index); }}
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
      <label for="ep-contact-{index}-name" class="ep-label">Full Name</label>
      <input
        id="ep-contact-{index}-name"
        type="text"
        class="ep-input"
        bind:value={contact.name}
        placeholder="Jane Smith"
        autocomplete="off"
      />
    </div>
    <div class="ep-field ep-field--half">
      <label for="ep-contact-{index}-relation" class="ep-label">Relation</label>
      <input
        id="ep-contact-{index}-relation"
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
      <label for="ep-contact-{index}-phone" class="ep-label">Phone</label>
      <input
        id="ep-contact-{index}-phone"
        type="tel"
        inputmode="tel"
        class="ep-input"
        class:ep-input--error={phoneError}
        bind:value={contact.phone}
        placeholder="+1 555 000 0000"
        autocomplete="tel"
        aria-describedby={phoneError ? `ep-contact-${index}-phone-error` : undefined}
      />
      {#if phoneError}
        <span id="ep-contact-{index}-phone-error" class="ep-field-error" role="alert">{phoneError}</span>
      {/if}
    </div>
    <div class="ep-field ep-field--half">
      <label for="ep-contact-{index}-address" class="ep-label">
        Address
        <span class="ep-optional"> (optional)</span>
      </label>
      <input
        id="ep-contact-{index}-address"
        type="text"
        class="ep-input"
        bind:value={contact.address}
        placeholder="City, State or full address"
        autocomplete="off"
      />
    </div>
  </div>
</div>
