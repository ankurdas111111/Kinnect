<script>
  
  /**
   * @typedef {Object} Props
   * @property {number} [count] - Total people count (self + others); header hidden when zero.
   */

  /** @type {Props} */
  let { count = 0 } = $props();
</script>

<!-- Section label -->
{#if count > 0}
  <div class="people-section-header">
    <span class="people-section-label">
      <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><circle cx="12" cy="8" r="4"/><path d="M20 21a8 8 0 1 0-16 0"/></svg>
      People
    </span>
    <span class="people-section-count">{count}</span>
  </div>
{/if}

<style>
  /* ── Section header ────────────────────────────────────────────────────── */
  .people-section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-3) var(--space-4) var(--space-2);
  }

  .people-section-label {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  .people-section-count {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    background: var(--surface-inset);
    border-radius: var(--radius-full);
    padding: 1px 7px;
    min-width: 20px;
    text-align: center;
  }
</style>
