<script>
  /**
   * SectionHeader — the canonical section header (CONTRACTS.md §7).
   * Kills the 11+ duplicated header patterns (InfoPanel ×12, Monitoring ×6,
   * EmergencyProfile ×6, SavedPlaces, HistoryPanel, SearchResults).
   *
   * Structure only: no animation, and the component owns no interactive
   * elements — the action snippet's focusability belongs to the consumer.
   * The action area guarantees a 44px alignment box so consumer buttons meet
   * touch targets without per-callsite CSS.
   */

  /** @type {{ title: string, subtitle?: string, level?: 2 | 3 | 4, icon?: import('svelte').Snippet, action?: import('svelte').Snippet }} */
  let {
    title,
    subtitle = '',
    level = 3,
    icon = undefined,
    action = undefined,
  } = $props();
</script>

<header class="section-header" data-level={level}>
  {#if icon}
    <span class="sh-icon" aria-hidden="true">{@render icon()}</span>
  {/if}
  <div class="sh-text">
    <svelte:element this={`h${level}`} class="sh-title">{title}</svelte:element>
    {#if subtitle}
      <p class="sh-subtitle">{subtitle}</p>
    {/if}
  </div>
  {#if action}
    <div class="sh-action">{@render action()}</div>
  {/if}
</header>

<style>
  .section-header {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
  }

  .sh-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    color: var(--text-secondary);
  }

  .sh-text {
    flex: 1;
    min-width: 0;
  }

  .sh-title {
    margin: 0;
    font-family: var(--font-display);
    font-size: var(--text-lg);
    font-weight: 700;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .section-header[data-level='2'] .sh-title {
    font-size: var(--text-xl);
  }

  .sh-subtitle {
    margin: var(--space-1) 0 0;
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }

  .sh-action {
    display: flex;
    align-items: center;
    flex-shrink: 0;
    min-height: 44px;
  }
</style>
