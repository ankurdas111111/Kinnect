<script>
  /**
   * VerdictStrip — compact one-line family verdict for nav surfaces
   * (FamilyPanel header, desktop sidebar header). Reads the shared verdict
   * store directly; tone is shown as BOTH a colored dot and words (never
   * color alone). Optionally clickable (sidebar: jumps to the Family tab).
   */
  import { familyVerdict } from '../../lib/stores/verdict.js';

  /** @type {{ onopen?: () => void, compact?: boolean }} */
  let { onopen, compact = false } = $props();

  let v = $derived($familyVerdict);
  let interactive = $derived(typeof onopen === 'function');
</script>

{#if interactive}
  <button class="strip tone-{v.tone}" class:compact onclick={onopen}
    aria-label="Family status: {v.sentence}. Open Family">
    {@render body()}
  </button>
{:else}
  <div class="strip tone-{v.tone}" class:compact role="status" aria-label="Family status: {v.sentence}">
    {@render body()}
  </div>
{/if}

{#snippet body()}
  <span class="strip-dot" aria-hidden="true"></span>
  {#if !compact}
    <span class="strip-text">
      <span class="strip-sentence verdict-voice">{v.sentence}</span>
      {#if v.detail}<span class="strip-detail">{v.detail}</span>{/if}
    </span>
  {/if}
{/snippet}

<style>
  /* Family Circle language: a quiet paper tier separated tonally, not by
     borders. Tone reads as BOTH the dot color and the sentence itself. */
  .strip {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    width: 100%;
    min-height: 44px;
    padding: var(--space-2) var(--space-3);
    background: var(--surface-2);
    border: none;
    border-radius: var(--radius-lg);
    text-align: left;
    color: inherit;
    font: inherit;
    -webkit-tap-highlight-color: transparent;
  }
  button.strip { cursor: pointer; }
  button.strip:hover { background: var(--surface-3); }
  button.strip:focus-visible { outline: 2px solid var(--strip-accent); outline-offset: 2px; }

  /* Sage = settled, ochre = needs a look, vermilion = SOS only (the alert
     tone is exclusively SOS in computeVerdict's ladder). */
  .tone-safe    { --strip-accent: var(--success-500); }
  .tone-caution { --strip-accent: var(--warning-500); }
  .tone-alert   { --strip-accent: var(--danger-500); }

  .strip-dot {
    width: 8px; height: 8px; border-radius: var(--radius-full); flex-shrink: 0;
    background: var(--strip-accent);
  }
  .strip-text { display: flex; flex-direction: column; gap: 1px; min-width: 0; }
  /* The one-line verdict speaks in the serif voice (via .verdict-voice). */
  .strip-sentence {
    font-size: var(--text-base);
    color: var(--text-primary);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .strip-detail {
    font-size: var(--text-xs); color: var(--text-tertiary);
    font-variant-numeric: tabular-nums;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }

  .strip.compact { justify-content: center; padding: var(--space-2); }
</style>
