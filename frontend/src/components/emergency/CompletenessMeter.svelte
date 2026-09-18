<script>
  /**
   * CompletenessMeter — quiet record-status line for the emergency profile.
   *
   * Hearth register: the design bans gamified progress mechanics, so this is
   * no longer a percentage ring. Completeness reads as calm reassurance in
   * the reference's crest-chip language — a soft pill ("Emergency record ·
   * ready for responders" / "n of m essentials in place") with the status
   * ladder's dot (sage = settled, ochre = could use a look) and a gentle
   * "Updated …" timestamp. Same inputs, same meaning, no score.
   *
   * Presentational: props only — the counting logic stays in the parent.
   *   progress    — 0–100 integer (drives the complete/in-progress wording)
   *   filledCount — number of filled tracked fields
   *   totalFields — total tracked fields
   *   lastUpdated — formatted "Updated …" string or null
   */

  /** @type {{ progress: number, filledCount: number, totalFields: number, lastUpdated?: string | null }} */
  let { progress, filledCount, totalFields, lastUpdated = null } = $props();

  let isComplete = $derived(progress === 100);
  let statusLabel = $derived(
    isComplete
      ? 'Ready for responders'
      : `${filledCount} of ${totalFields} essentials in place`
  );
</script>

<div class="ep-meter" role="status" aria-live="polite">
  <span class="ep-meter-chip">
    <span
      class="ep-meter-dot"
      class:ep-meter-dot--complete={isComplete}
      aria-hidden="true"
    ></span>
    <span class="ep-meter-label">Emergency record · {statusLabel}</span>
  </span>
  {#if lastUpdated}
    <span class="ep-meter-updated">Updated {lastUpdated}</span>
  {/if}
</div>

<style>
  .ep-meter {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2) var(--space-3);
  }

  .ep-meter-chip {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-1-5) var(--space-3);
    border-radius: var(--radius-full, 9999px);
    background: var(--surface-3);
  }

  /* Status-ladder dot: ochre = the record could use a look; sage = settled.
     Never vermilion — an unfinished form is not an emergency. */
  .ep-meter-dot {
    width: 8px;
    height: 8px;
    border-radius: var(--radius-full, 9999px);
    background: var(--warning-500);
    flex-shrink: 0;
  }
  .ep-meter-dot--complete {
    background: var(--success-500);
  }

  .ep-meter-label {
    font-size: var(--text-xs);
    font-weight: 600;
    letter-spacing: 0.02em;
    color: var(--text-secondary);
  }

  .ep-meter-updated {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }
</style>
