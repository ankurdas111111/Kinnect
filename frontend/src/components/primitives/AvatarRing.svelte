<script>
  /**
   * AvatarRing — one presence-ring grammar (CONTRACTS.md §4).
   * Wraps ANY avatar content (image, initials div, emoji) — owns the ring,
   * never the avatar itself.
   *
   * Ring semantics — color + SHAPE, never color alone (grayscale-safe):
   *   live    — solid --ring-width-md ring, --ring-color-live
   *   sos     — widest ring + pre-layered pulse halo (fx-ambient, gated)
   *   offline — thinner --ring-width-sm ring, --ring-color-offline, dimmed
   *   none    — no ring
   *
   * CONSUMPTION RULE: ring state must co-exist with visible text in the
   * consumer (StatusBadge, FreshnessChip, sublines) — never the only signal.
   * Non-interactive; never focusable.
   */
  import { allowMotion } from '../../lib/stores/effects.js';

  /** @type {{ ring?: 'live' | 'sos' | 'offline' | 'none', size?: number, label?: string, children: import('svelte').Snippet }} */
  let {
    ring = 'none',
    size = 44,
    label = '',
    children,
  } = $props();

  // SOS pulse loop is JS-gated on the effects store (CSS additionally gates
  // via prefers-reduced-motion). The static sos ring survives either gate.
  let pulse = $derived(ring === 'sos' && $allowMotion);
</script>

<span
  class="avatar-ring"
  data-ring={ring}
  class:has-pulse={pulse}
  style="--ar-size: {size}px"
  role={label ? 'img' : undefined}
  aria-label={label || undefined}
  aria-hidden={label ? undefined : 'true'}
>
  {#if ring === 'sos'}
    <!-- Pre-layered halo: static paint, animated by transform/opacity only. -->
    <i class="ring-pulse fx-ambient" aria-hidden="true"></i>
  {/if}
  <span class="ar-content">{@render children()}</span>
</span>

<style>
  .avatar-ring {
    position: relative;
    display: inline-flex;
    width: var(--ar-size);
    height: var(--ar-size);
    flex-shrink: 0;
    border-radius: var(--radius-full, 9999px);
  }

  .ar-content {
    display: flex;
    width: 100%;
    height: 100%;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    border-radius: inherit;
  }

  .avatar-ring[data-ring='live'] .ar-content {
    box-shadow: 0 0 0 var(--ring-width-md) var(--ring-color-live);
  }

  /* Widest ring = the sos shape cue (plus halo). calc keeps it token-derived. */
  .avatar-ring[data-ring='sos'] .ar-content {
    box-shadow: 0 0 0 calc(var(--ring-width-md) * 1.6) var(--ring-color-sos);
  }

  .avatar-ring[data-ring='offline'] .ar-content {
    box-shadow: 0 0 0 var(--ring-width-sm) var(--ring-color-offline);
    opacity: 0.75;
  }

  /* SOS pulse halo — pre-layered, transform/opacity only, dormant until
     .has-pulse (JS allowMotion gate) switches it on. */
  .ring-pulse {
    position: absolute;
    inset: 0;
    border-radius: inherit;
    box-shadow: 0 0 0 var(--ring-width-md) var(--ring-color-sos);
    opacity: 0;
    pointer-events: none;
  }
  .avatar-ring.has-pulse .ring-pulse {
    animation: ar-sos-pulse 1.2s var(--ease-out, cubic-bezier(0.4, 0, 0.2, 1)) infinite;
  }

  @keyframes ar-sos-pulse {
    0%   { transform: scale(1);    opacity: 0.7; }
    70%  { transform: scale(1.35); opacity: 0; }
    100% { transform: scale(1.35); opacity: 0; }
  }

  /* Motion dies, meaning survives: static sos ring remains. */
  @media (prefers-reduced-motion: reduce) {
    .ring-pulse { animation: none !important; }
  }
</style>
