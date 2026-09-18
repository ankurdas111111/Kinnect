<script>
  /**
   * ContactCard — one emergency contact (name / relation / phone / address).
   * Presentational: `contact` is bindable; index drives the identical field
   * ids (ep-contact-{i}-name, -relation, -phone, -address) and remove callback.
   *
   * Hearth register: the card leads with a warm identity pebble (the person's
   * initial) rather than a numbered register row, and "Remove" is a plain-ink
   * worded action — removing a contact is quiet, never red.
   */
  import { haptics } from '../../lib/haptics.js';

  /** @type {{ contact: any, index: number, phoneError?: string, onremove: (i: number) => void }} */
  let { contact = $bindable(), index, phoneError = '', onremove } = $props();

  let initial = $derived((contact.name || '').trim().charAt(0).toUpperCase() || '?');
  let displayLabel = $derived((contact.name || '').trim() || `Contact ${index + 1}`);
</script>

<div class="ep-contact-card">
  <div class="ep-contact-header">
    <span class="ep-contact-id">
      <span class="ep-contact-pebble" aria-hidden="true">{initial}</span>
      <span class="ep-contact-num">{displayLabel}</span>
    </span>
    <button
      class="ep-remove-btn"
      type="button"
      aria-label="Remove contact {index + 1}"
      onclick={() => { haptics.tap(); onremove(index); }}
    >
      Remove
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
