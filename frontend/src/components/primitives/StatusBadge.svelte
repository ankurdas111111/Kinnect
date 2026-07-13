<script>
  /**
   * StatusBadge — the connection badge, unified (CONTRACTS.md §5).
   * Extracted grammar from LiveViewer .status-badge + WatchViewer
   * .watch-status-badge. Feed `state` from deriveConnState() (lib/presence.js).
   *
   * Color + SHAPE + text, never color alone:
   *   live       — filled dot          — --status-live
   *   connecting — hollow dot          — --text-tertiary
   *   issue      — diamond (45° square)— --warning-500
   *   offline    — hollow dot, dimmed  — --status-offline
   *
   * `label` overrides the default vocab — server-health consumers (Monitoring)
   * pass "OK" / "Degraded" / "Down", never "live/offline".
   *
   * `announce` default false: churny connections must not spam live regions.
   */

  /** @type {{ state?: 'live' | 'connecting' | 'issue' | 'offline', label?: string, announce?: boolean }} */
  let {
    state: badgeState = 'connecting',
    label = '',
    announce = false,
  } = $props();

  const VOCAB = {
    live: 'Live',
    connecting: 'Connecting…',
    issue: 'Connection issue',
    offline: 'Offline',
  };

  let text = $derived(label || VOCAB[badgeState] || badgeState);
</script>

<span
  class="status-badge"
  data-state={badgeState}
  role={announce ? 'status' : undefined}
  aria-live={announce ? 'polite' : undefined}
>
  <i class="sb-dot" class:fx-ambient={badgeState === 'live' || badgeState === 'issue'} aria-hidden="true"></i>
  <span class="sb-label">{text}</span>
</span>

<style>
  .status-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-full, 9999px);
    font-size: var(--text-sm);
    font-weight: 600;
    transition:
      background var(--duration-slow, 320ms) var(--ease-out, cubic-bezier(0.4, 0, 0.2, 1)),
      color var(--duration-slow, 320ms) var(--ease-out, cubic-bezier(0.4, 0, 0.2, 1));
  }

  .sb-label {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sb-dot {
    width: var(--space-2);
    height: var(--space-2);
    flex-shrink: 0;
    border-radius: var(--radius-full, 9999px);
    background: currentColor;
  }

  .status-badge[data-state='live'] {
    color: var(--status-live);
    background: color-mix(in oklch, var(--status-live) 12%, transparent);
  }

  .status-badge[data-state='connecting'] {
    color: var(--text-tertiary);
    background: var(--primary-500-08);
  }
  .status-badge[data-state='connecting'] .sb-dot {
    background: transparent;
    border: var(--ring-width-sm) solid currentColor;
  }

  .status-badge[data-state='issue'] {
    color: var(--warning-500);
    background: color-mix(in oklch, var(--warning-500) 12%, transparent);
  }
  /* Diamond: rotated square. Static transform — the blink animates opacity
     only, so shape and loop never fight over the same property. */
  .status-badge[data-state='issue'] .sb-dot {
    border-radius: var(--radius-xs, 2px);
    transform: rotate(45deg);
  }

  .status-badge[data-state='offline'] {
    color: var(--status-offline);
    background: transparent;
    opacity: 0.7;
  }
  .status-badge[data-state='offline'] .sb-dot {
    background: transparent;
    border: var(--ring-width-sm) solid currentColor;
    opacity: 0.5;
  }

  .status-badge[data-state='live'] .sb-dot {
    animation: sb-pulse 1.8s var(--ease-in-out, ease-in-out) infinite;
  }
  /* 1Hz blink is WCAG 2.3.1-safe (flash threshold is 3/s) — NEVER speed up. */
  .status-badge[data-state='connecting'] .sb-dot {
    animation: sb-blink 1s var(--ease-in-out, ease-in-out) infinite;
  }
  .status-badge[data-state='issue'] .sb-dot {
    animation: sb-blink 0.85s var(--ease-in-out, ease-in-out) infinite;
  }

  @keyframes sb-pulse {
    0%, 100% { opacity: 1;    transform: scale(1); }
    50%      { opacity: 0.55; transform: scale(0.82); }
  }
  @keyframes sb-blink {
    0%, 100% { opacity: 1; }
    50%      { opacity: 0.25; }
  }

  /* State stays readable via dot shape + color + text. */
  @media (prefers-reduced-motion: reduce) {
    .sb-dot { animation: none !important; }
  }
</style>
