<script>
  import { run } from 'svelte/legacy';

  
  /**
   * @typedef {Object} Props
   * @property {string} [text] - KineticText — 2026 per-character animated heading.
Each character slides + fades in with a staggered spring delay.
Gracefully degrades under prefers-reduced-motion.
Props:
text      — the heading string to animate
tag       — HTML element to render (default 'h1')
delay     — base delay in ms before animation starts (default 0)
stagger   — ms between each character (default 40)
className — additional CSS class applied to the root element
once      — if true, only animates on first mount (not on text change)
Bug fix: removed filter:blur() from the kt-rise keyframe.
filter is NOT GPU-composited — it forces the browser to create a new
stacking context and triggers repaint on every frame, causing jank on
iOS Safari especially when 20+ characters animate simultaneously.
The entrance now uses transform+opacity only (GPU compositor path).
   * @property {string} [tag]
   * @property {number} [delay]
   * @property {number} [stagger]
   * @property {string} [className]
   * @property {boolean} [once]
   */

  /** @type {Props} */
  let {
    text = '',
    tag = 'h1',
    delay = 0,
    stagger = 40,
    className = '',
    once = false
  } = $props();

  // Split into characters, preserving spaces as non-breaking
  let chars = $derived(Array.from(text));

  // Key to re-trigger animation on text change (unless once=true)
  let animKey = $state(0);
  run(() => {
    if (!once) { text; animKey++; }
  });
</script>

<!-- svelte:element used to render dynamic tag -->
{#key animKey}
  <svelte:element this={tag} class="kt-root {className}" aria-label={text}>
    {#each chars as char, i}
      <span
        class="kt-char"
        class:kt-space={char === ' '}
        aria-hidden="true"
        style="animation-delay: {delay + i * stagger}ms"
      >{char === ' ' ? '\u00A0' : char}</span>
    {/each}
  </svelte:element>
{/key}

<style>
  .kt-root {
    display: block;
    /* Characters use inline-block so they each animate independently */
  }

  .kt-char {
    display: inline-block;
    /* GPU-only: transform + opacity only. No filter — filter is not composited
       and causes repaint on every frame when many characters animate in parallel,
       particularly on iOS Safari. */
    animation: kt-rise 500ms cubic-bezier(0.34, 1.56, 0.64, 1) both;
    will-change: transform, opacity;
  }

  .kt-space {
    /* Preserve space width — non-breaking space handles it, but add min-width */
    min-width: 0.28em;
  }

  @keyframes kt-rise {
    from {
      opacity: 0;
      transform: translateY(0.5em) scale(0.9);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .kt-char {
      animation: none;
      opacity: 1;
      transform: none;
      will-change: auto;
    }
  }
</style>
