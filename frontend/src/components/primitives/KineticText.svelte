<script>
  /**
   * KineticText — 2026 per-character animated heading.
   *
   * Each character slides + fades in with a staggered spring delay.
   * Gracefully degrades under prefers-reduced-motion.
   *
   * Props:
   *   text     — the heading string to animate
   *   tag      — HTML element to render (default 'h1')
   *   delay    — base delay in ms before animation starts (default 0)
   *   stagger  — ms between each character (default 40)
   *   className — additional CSS class
   *   once     — if true, only animates on first mount (not on text change)
   */
  export let text = '';
  export let tag = 'h1';
  export let delay = 0;
  export let stagger = 40;
  export let className = '';
  export let once = false;

  // Split into characters, preserving spaces as non-breaking
  $: chars = Array.from(text);

  // Key to re-trigger animation on text change (unless once=true)
  let animKey = 0;
  $: if (!once) { text; animKey++; }
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
      transform: translateY(0.6em) rotateX(40deg);
      filter: blur(3px);
    }
    to {
      opacity: 1;
      transform: translateY(0) rotateX(0deg);
      filter: blur(0);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .kt-char {
      animation: none;
      opacity: 1;
      transform: none;
      filter: none;
    }
  }
</style>
