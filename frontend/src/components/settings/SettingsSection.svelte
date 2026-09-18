<script>
  /**
   * SettingsSection — settings group wrapper (title + optional description + children).
   *
   * Hearth register: a quiet paper sheet lifted one tonal step off the panel
   * ground (surface tier + warm shadow) — no glass blur, no boxed glow.
   * The `danger` prop is kept for API stability but no longer paints red:
   * destructive sections speak through explicit copy, not alarm color
   * (vermilion is SOS-only).
   *
   * Props:
   *   title       — string (required) — section heading
   *   description — string            — secondary hint beneath title
   *   danger      — boolean           — accepted, visually identical (see above)
   *   children    — Snippet (required)
   */

  /** @type {{ title: string, description?: string, danger?: boolean, children: import('svelte').Snippet }} */
  let { title, description = '', danger = false, children } = $props();
</script>

<section class="ss-card" class:ss-card--danger={danger}>
  <div class="ss-header">
    <h4 class="ss-title">{title}</h4>
    {#if description}
      <p class="ss-desc">{description}</p>
    {/if}
  </div>
  <div class="ss-body">
    {@render children()}
  </div>
</section>

<style>
  /* Quiet paper sheet — tonal lift, no border-in-border. */
  .ss-card {
    background: var(--surface-1);
    border-radius: var(--radius-card, 20px);
    box-shadow: var(--shadow-xs);
    padding: var(--space-4) var(--space-4) var(--space-4);
  }

  /* Destructive sections look the same on purpose — words carry the weight. */
  .ss-card--danger { background: var(--surface-1); }

  .ss-header {
    margin-bottom: var(--space-3);
  }

  .ss-title {
    margin: 0;
    font-family: var(--font-display);
    font-size: var(--text-xl);
    font-weight: 600;
    color: var(--text-primary);
    letter-spacing: -0.01em;
    line-height: 1.25;
  }

  .ss-desc {
    margin: var(--space-1-5) 0 0;
    font-size: var(--text-sm);
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .ss-body {
    display: flex;
    flex-direction: column;
    gap: var(--space-2-5);
  }

  /* 150ms opacity fade on section entrance — instant feel, not theatrical */
  :global(.settings-section-enter) {
    animation: ss-fade var(--duration-fast, 150ms) var(--ease-out, ease-out) both;
  }

  @keyframes ss-fade {
    from { opacity: 0; }
    to   { opacity: 1; }
  }

  @media (prefers-reduced-motion: reduce) {
    :global(.settings-section-enter) {
      animation: none;
    }
  }
</style>
