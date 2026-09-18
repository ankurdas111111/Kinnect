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
  /* ═══ Hearth: quiet paper tiles, one ember icon pebble ═══ */
  .qa-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1-5);
    padding: var(--space-3-5) var(--space-2);
    min-height: 44px;
    background: var(--surface-2);
    border: none;
    border-radius: var(--radius-lg);
    cursor: pointer;
    position: relative;
    transition: background-color var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }
  .qa-card:hover {
    background: var(--surface-hover);
  }
  .qa-card:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }

  .qa-active {
    background: var(--success-500-12);
    box-shadow: inset 0 0 0 1.5px var(--success-500);
  }
  .qa-active:hover {
    background: var(--success-500-20);
  }

  .qa-icon {
    width: 40px;
    height: 40px;
    border-radius: var(--radius-full);
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--primary-100);
    color: var(--primary-700);
  }
  .qa-active .qa-icon {
    background: var(--success-500-20);
    color: var(--success-700);
  }

  .qa-label {
    font-family: var(--font-sans);
    font-size: var(--text-xs);
    font-weight: 600;
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
    background: var(--status-live);
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
