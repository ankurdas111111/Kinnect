<script>
  /**
   * @typedef {Object} Props
   * @property {number} [count] - Total people count (self + others); header hidden when zero.
   * @property {(() => void) | null} [onAddPeople] - Optional quiet "Add person"
   *   action (routes to the existing Connect/sharing flow). Hidden when null.
   */

  /** @type {Props} */
  let { count = 0, onAddPeople = null } = $props();
</script>

<!-- Sheet crest — the serif screen title with a quiet add action beside it -->
{#if count > 0}
  <div class="people-section-header">
    <h2 class="people-section-title">
      Circle Presence
      <span class="people-section-count font-tabular" aria-label="{count} people">{count}</span>
    </h2>
    {#if onAddPeople}
      <button class="add-person-btn" type="button" onclick={onAddPeople} aria-label="Add or invite a person">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/></svg>
        <span>Add person</span>
      </button>
    {/if}
  </div>
{/if}

<style>
  /* ── Sheet header — title crest + quiet action, no metric chrome ───────── */
  .people-section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4) var(--space-2);
  }

  /* Screen-title register: Newsreader, upright, calm */
  .people-section-title {
    display: inline-flex;
    align-items: baseline;
    gap: var(--space-2);
    margin: 0;
    font-family: var(--font-serif);
    font-size: var(--heading-section);
    font-weight: 400;
    letter-spacing: -0.01em;
    color: var(--text-primary);
    line-height: var(--leading-tight);
  }

  .people-section-count {
    font-family: var(--font-sans);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-tertiary);
    background: var(--surface-inset);
    border-radius: var(--radius-full);
    padding: var(--space-0-5) var(--space-2);
    min-width: 20px;
    text-align: center;
    font-variant-numeric: tabular-nums;
  }

  /* Quiet ember text-button — small accent text rides --primary-700 for AA */
  .add-person-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    min-height: 44px;
    padding: var(--space-1) var(--space-2);
    border: none;
    background: transparent;
    border-radius: var(--radius-md);
    font-family: var(--font-sans);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--primary-700);
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out), color var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .add-person-btn:hover {
    background: color-mix(in oklch, var(--primary-500) 10%, transparent);
  }
  .add-person-btn:active {
    background: color-mix(in oklch, var(--primary-500) 16%, transparent);
  }
  .add-person-btn:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 2px;
  }
</style>
