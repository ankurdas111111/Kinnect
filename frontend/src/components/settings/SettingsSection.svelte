<script>
  /**
   * SettingsSection — settings group wrapper (title + optional description + children).
   * Renders a glass Card with consistent spacing; supports a danger variant for
   * destructive actions (Delete Account).
   *
   * Props:
   *   title       — string (required) — section heading
   *   description — string            — secondary hint beneath title
   *   danger      — boolean           — red accent border + title colour
   *   children    — Snippet (required)
   */

  import Card from '../primitives/Card.svelte';

  /** @type {{ title: string, description?: string, danger?: boolean, children: import('svelte').Snippet }} */
  let { title, description = '', danger = false, children } = $props();
</script>

<Card variant="glass" hover={false} padding="md" glow={danger ? 'danger' : undefined}>
  <div class="ss-header">
    <h4 class="ss-title" class:ss-title--danger={danger}>{title}</h4>
    {#if description}
      <p class="ss-desc">{description}</p>
    {/if}
  </div>
  <div class="ss-body">
    {@render children()}
  </div>
</Card>

<style>
  .ss-header {
    margin-bottom: var(--space-3);
  }

  .ss-title {
    margin: 0;
    font-family: var(--font-display);
    font-size: var(--text-base);
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1.2;
  }

  .ss-title--danger {
    color: var(--danger-600);
  }

  :global([data-theme='dark']) .ss-title--danger {
    color: var(--danger-400);
  }

  .ss-desc {
    margin: var(--space-1) 0 0;
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    line-height: 1.4;
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
