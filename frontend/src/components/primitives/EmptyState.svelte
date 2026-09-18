<script>
  /**
   * EmptyState — standard empty view. Design rules forbid empty states with no
   * next step, so `title` + a CTA (via the `action` snippet) are the norm.
   *
   * Props:
   *   title  — headline (required, specific & action-oriented — not "No data")
   *   body   — supporting sentence
   *   icon   — snippet for a glyph (paired with text, never decorative alone)
   *   action — snippet for the CTA (button/link)
   *   tone   — 'neutral' | 'primary' | 'danger' (tints the icon halo)
   */

  /** @type {{ title: string, body?: string, tone?: string, icon?: import('svelte').Snippet, action?: import('svelte').Snippet }} */
  let { title, body = '', tone = 'primary', icon, action } = $props();
</script>

<div class="empty" role="status">
  {#if icon}
    <div class="empty-icon tone-{tone} fx-decor-anim" aria-hidden="true">
      {@render icon()}
    </div>
  {/if}
  <h3 class="empty-title">{title}</h3>
  {#if body}
    <p class="empty-body">{body}</p>
  {/if}
  {#if action}
    <div class="empty-action">{@render action()}</div>
  {/if}
</div>

<style>
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    gap: var(--space-3);
    padding: var(--space-8) var(--space-5);
    max-width: 340px;
    margin-inline: auto;
  }

  .empty-icon {
    display: grid;
    place-items: center;
    width: 64px;
    height: 64px;
    border-radius: var(--radius-full, 9999px);
    color: var(--text-primary);
    background: var(--surface-inset);
    animation: empty-breathe 4s var(--ease-in-out, ease-in-out) infinite;
  }
  .empty-icon :global(svg) { width: 28px; height: 28px; }

  .tone-primary { background: var(--primary-100); color: var(--primary-700); }
  /* "danger" empty states are error notices, not SOS — deep ochre. */
  .tone-danger  { background: color-mix(in oklch, var(--warning-500) 12%, transparent); color: var(--warning-700); }

  .empty-title {
    margin: 0;
    font-family: var(--font-display);
    font-size: var(--text-lg, 1.125rem);
    font-weight: 600;
    color: var(--text-primary);
  }

  .empty-body {
    margin: 0;
    font-size: var(--text-base, 1rem);
    line-height: 1.5;
    color: var(--text-secondary);
  }

  .empty-action { margin-top: var(--space-2); }

  @keyframes empty-breathe {
    0%, 100% { transform: scale(1); }
    50%      { transform: scale(1.05); }
  }

  @media (prefers-reduced-motion: reduce) {
    .empty-icon { animation: none; }
  }
</style>
