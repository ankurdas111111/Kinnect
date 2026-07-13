<script>
  /**
   * ResponderNotesSection — preferred language + free-text responder notes.
   * Field ids preserved: ep-language, ep-responder-notes.
   */
  import { autoResize } from './epActions.js';

  /** @type {{ profile: any }} */
  let { profile = $bindable() } = $props();
</script>

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
