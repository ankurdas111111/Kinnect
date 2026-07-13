<script>
  /**
   * QuickActionCard — one equal-width quick-action tile in the sharing
   * bento row (Share Ride / On My Way / Invite). Presentational only:
   * the parent owns the click behaviour via `onclick`.
   *
   * Icon is a Snippet (aria-hidden — paired with the visible label text).
   * `active` surfaces a live green tint + pulsing dot (color + shape + text).
   */
  /** @type {{ label: string, tone?: 'ride' | 'omw' | 'invite', active?: boolean, activeLabel?: string, icon: import('svelte').Snippet, onclick?: () => void }} */
  let {
    label,
    tone = 'ride',
    active = false,
    activeLabel = '',
    icon,
    onclick,
  } = $props();
</script>

<button
  class="qa-card tactile"
  class:qa-active={active}
  data-tone={tone}
  onclick={onclick}
>
  <span class="qa-icon" data-tone={tone}>{@render icon()}</span>
  <span class="qa-label">{active && activeLabel ? activeLabel : label}</span>
  {#if active}
    <span class="qa-live-dot fx-ambient" aria-hidden="true"></span>
  {/if}
</button>

<style>
  .qa-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1-5);
    padding: var(--space-3-5) var(--space-2);
    min-height: 44px;
    background: var(--glass-chip-bg, var(--surface-inset));
    border: 1px solid var(--glass-chip-border, var(--border-subtle));
    border-radius: var(--radius-lg);
    cursor: pointer;
    position: relative;
    transition: background var(--duration-fast) var(--ease-out),
                border-color var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }
  .qa-card:hover {
    background: var(--surface-hover);
    border-color: var(--border-default);
  }
  .qa-card:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }

  .qa-active {
    background: var(--success-500-12);
    border-color: color-mix(in oklch, var(--success-500) 35%, transparent);
  }
  .qa-active:hover {
    background: color-mix(in oklch, var(--success-500) 16%, transparent);
  }

  .qa-icon {
    width: 40px;
    height: 40px;
    border-radius: var(--radius-sm2);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .qa-icon[data-tone='ride'] {
    background: var(--primary-500-12);
    border: 1px solid var(--primary-500-20);
    color: var(--primary-400);
  }
  .qa-icon[data-tone='omw'] {
    background: color-mix(in oklch, var(--primary-500) 12%, transparent);
    border: 1px solid var(--primary-500-20);
    color: var(--primary-500);
  }
  .qa-icon[data-tone='invite'] {
    background: color-mix(in oklch, var(--whatsapp-green) 14%, transparent);
    border: 1px solid color-mix(in oklch, var(--whatsapp-green) 22%, transparent);
    color: var(--whatsapp-green);
  }

  .qa-label {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: 0.01em;
    text-align: center;
    line-height: 1.25;
  }

  .qa-live-dot {
    position: absolute;
    top: 8px;
    right: 8px;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--success-500);
    box-shadow: 0 0 6px color-mix(in oklch, var(--success-500) 50%, transparent);
    animation: qa-pulse 2s var(--ease-in-out) infinite;
  }
  @keyframes qa-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }

  @media (prefers-reduced-motion: reduce) {
    .qa-live-dot { animation: none; }
  }
</style>
