<script>
  /**
   * EpSection — the collapsible section shell for the Emergency Profile.
   *
   * Hearth register: a quiet paper sheet lifted one tonal step off the paper
   * ground (surface tier + warm shadow, no boxed outline, no glass blur).
   * The header keeps a bare ember icon beside plain Work Sans words — the
   * reference's section-crest treatment — and the whole shell stays a real
   * disclosure so an anxious user can collapse what they don't need.
   *
   * Disclosure grammar (infoPanel pattern): chevron rotate 200ms + content
   * opacity 150ms, DISPLAY-GATED (`{#if open}` unmounts, so no reflow cost and
   * no hidden focusable elements). Reduced-motion → instant.
   *
   * Presentational: owns no profile state. `open` is bindable; the parent
   * persists openSections. Header is a real <button> with aria-expanded /
   * aria-controls wired to a stable id so screen-reader muscle memory survives.
   */

  /**
   * @type {{
   *   id: string,
   *   title: string,
   *   open?: boolean,
   *   count?: number | null,
   *   icon?: import('svelte').Snippet,
   *   children: import('svelte').Snippet,
   *   ontoggle?: () => void,
   * }}
   */
  let {
    id,
    title,
    open = $bindable(false),
    count = null,
    icon = undefined,
    children,
    ontoggle,
  } = $props();

  const bodyId = `${id}-body`;
  const headingId = `${id}-heading`;

  function toggle() {
    open = !open;
    ontoggle?.();
  }
</script>

<section class="ep-section" aria-labelledby={headingId}>
  <button
    class="ep-section-header"
    id={`${id}-btn`}
    aria-expanded={open}
    aria-controls={bodyId}
    onclick={toggle}
  >
    {#if icon}
      <span class="ep-section-icon" aria-hidden="true">{@render icon()}</span>
    {/if}
    <span id={headingId} class="ep-section-title">{title}</span>
    {#if count != null && count > 0}
      <span class="ep-section-count" aria-label="{count} items">{count}</span>
    {/if}
    <span class="ep-section-chevron" class:ep-section-chevron--open={open} aria-hidden="true">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
           stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
        <polyline points="6 9 12 15 18 9"/>
      </svg>
    </span>
  </button>

  {#if open}
    <div
      class="ep-section-body ep-reveal"
      id={bodyId}
      role="region"
      aria-labelledby={headingId}
    >
      {@render children()}
    </div>
  {/if}
</section>

<style>
  /* ── Section sheet — tonal lift off the paper ground, no boxed border ──── */
  .ep-section {
    border-radius: var(--radius-card, 20px);
    background: var(--surface-1);
    box-shadow: var(--shadow-xs);
    overflow: hidden;
  }

  .ep-section-header {
    width: 100%;
    min-height: 44px;
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
    padding: var(--space-4) var(--space-4);
    border: none;
    background: transparent;
    cursor: pointer;
    text-align: left;
    font-family: var(--font-sans);
    transition: background 120ms var(--ease-out);
  }
  .ep-section-header:hover {
    background: var(--surface-hover);
  }
  .ep-section-header:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: -2px;
  }

  /* Bare ember glyph beside the words — no icon chip, no badge box. */
  .ep-section-icon {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--primary-700);
  }

  .ep-section-title {
    flex: 1;
    font-size: var(--text-xl);
    font-weight: 600;
    color: var(--text-primary);
    letter-spacing: -0.01em;
  }

  .ep-section-count {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 22px;
    height: 22px;
    padding: 0 var(--space-2);
    border-radius: var(--radius-full, 9999px);
    background: var(--surface-3);
    color: var(--text-secondary);
    font-size: var(--text-xs);
    font-weight: 600;
    margin-left: auto;
  }

  .ep-section-chevron {
    flex-shrink: 0;
    color: var(--text-tertiary);
    display: flex;
    align-items: center;
    transition: transform 200ms var(--ease-out);
  }
  .ep-section-chevron--open {
    transform: rotate(180deg);
  }

  /* Body separated from the header by whitespace alone — no divider rule. */
  .ep-section-body {
    padding: var(--space-1) var(--space-4) var(--space-5);
    display: flex;
    flex-direction: column;
    gap: var(--space-3-5);
  }

  /* Disclosure reveal — opacity only (GPU-safe), display-gated by {#if}. */
  .ep-reveal {
    animation: ep-reveal-fade 150ms var(--ease-out) both;
  }
  @keyframes ep-reveal-fade {
    from { opacity: 0; }
    to   { opacity: 1; }
  }

  @media (prefers-reduced-motion: reduce) {
    .ep-section-chevron,
    .ep-section-header { transition: none; }
    .ep-reveal { animation: none; }
  }
</style>
