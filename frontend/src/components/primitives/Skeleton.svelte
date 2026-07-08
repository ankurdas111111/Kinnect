<script>
  /**
   * Skeleton — shimmer placeholder for loading states.
   * Design rules forbid center-of-card spinners; use these instead.
   *
   * Props:
   *   variant — 'text' | 'title' | 'avatar' | 'card' | 'line'
   *   width   — CSS width (default per-variant)
   *   height  — CSS height (default per-variant)
   *   radius  — CSS border-radius override
   *   count   — render N stacked skeletons (for lists)
   */

  /** @type {{ variant?: string, width?: string, height?: string, radius?: string, count?: number }} */
  let {
    variant = 'text',
    width = undefined,
    height = undefined,
    radius = undefined,
    count = 1,
  } = $props();

  let items = $derived(Array.from({ length: Math.max(1, count) }));
</script>

{#each items as _, i (i)}
  <span
    class="sk sk-{variant} fx-decor-anim"
    aria-hidden="true"
    style:width={width}
    style:height={height}
    style:border-radius={radius}
  ></span>
{/each}

<style>
  .sk {
    display: block;
    position: relative;
    overflow: hidden;
    background: var(--surface-inset, rgba(255, 255, 255, 0.05));
    border-radius: var(--radius-md, 10px);
  }

  .sk + .sk { margin-top: var(--space-2); }

  /* Shimmer sweep — transform-only (GPU). */
  .sk::after {
    content: '';
    position: absolute;
    inset: 0;
    transform: translateX(-100%);
    background: linear-gradient(
      90deg,
      transparent 0%,
      color-mix(in oklch, var(--text-primary, #fff) 8%, transparent) 50%,
      transparent 100%
    );
    animation: sk-shimmer 1.4s var(--ease-in-out, ease-in-out) infinite;
  }

  .sk-text   { width: 100%;  height: 12px; }
  .sk-line   { width: 60%;   height: 10px; }
  .sk-title  { width: 45%;   height: 20px; border-radius: var(--radius-sm, 6px); }
  .sk-avatar { width: 44px;  height: 44px; border-radius: var(--radius-full, 9999px); }
  .sk-card   { width: 100%;  height: 88px; border-radius: var(--radius-card, 20px); }

  @keyframes sk-shimmer {
    to { transform: translateX(100%); }
  }

  @media (prefers-reduced-motion: reduce) {
    .sk::after { animation: none; opacity: 0.4; }
  }
</style>
