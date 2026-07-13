<script>
  /**
   * StoryBeat — one beat of the Landing scrollytelling spine.
   *
   * Owns its own IntersectionObserver (the PRIMARY scene driver — must work
   * on every WebView): when the beat crosses the viewport's center band it
   * calls `onactive(index + 1)` so the parent switches the sticky mockup's
   * scene. CSS scroll-driven animation is a progressive enhancement only,
   * behind @supports — never load-bearing.
   *
   * Props:
   *   beat     — { time, title, desc } from landingStory.js BEATS
   *   index    — 0-based beat index (scene = index + 1)
   *   active   — parent-confirmed active state (drives emphasis styling)
   *   onactive — (sceneIndex) => void
   */
  import { onMount } from 'svelte';

  let { beat, index = 0, active = false, onactive } = $props();

  let el;

  onMount(() => {
    const io = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.isIntersecting) onactive?.(index + 1);
        }
      },
      // Fire when the beat enters the middle 30% band of the viewport.
      { rootMargin: '-35% 0px -35% 0px', threshold: 0 }
    );
    io.observe(el);
    return () => io.disconnect();
  });
</script>

<li class="beat" class:active bind:this={el}>
  <div class="beat-inner">
    <span class="beat-time font-tabular">{beat.time}</span>
    <h3 class="beat-title">{beat.title}</h3>
    <p class="beat-desc">{beat.desc}</p>
  </div>
</li>

<style>
  .beat {
    min-height: 72svh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    list-style: none;
  }

  @media (max-width: 900px) {
    .beat { min-height: 64svh; justify-content: flex-end; padding-bottom: var(--space-12); }
  }

  /* Dimmed rest state → active emphasis. Transform/opacity only. */
  .beat-inner {
    max-width: 46ch;
    opacity: 0.35;
    transform: translateY(12px);
    transition:
      opacity var(--duration-slow) var(--ease-out),
      transform var(--duration-slow) var(--ease-out);
  }
  .beat.active .beat-inner {
    opacity: 1;
    transform: translateY(0);
  }

  .beat-time {
    display: inline-block;
    font-family: var(--font-mono, monospace);
    font-size: var(--text-xs);
    font-weight: 700;
    letter-spacing: 0.08em;
    color: var(--primary-400);
    border: 1px solid color-mix(in oklch, var(--primary-400) 30%, transparent);
    background: color-mix(in oklch, var(--primary-400) 8%, transparent);
    border-radius: var(--radius-full, 9999px);
    padding: 3px 10px;
    margin-bottom: var(--space-3);
    font-variant-numeric: tabular-nums;
  }

  .beat-title {
    font-family: var(--font-display);
    font-size: clamp(var(--text-xl), 2.4vw, var(--text-3xl));
    font-weight: 800;
    letter-spacing: -0.02em;
    line-height: 1.15;
    color: var(--text-primary);
    margin-bottom: var(--space-3);
  }

  .beat-desc {
    font-size: var(--text-base);
    color: var(--text-secondary);
    line-height: var(--leading-relaxed);
  }

  /* Progressive enhancement: scroll-linked entrance where view() exists. */
  @supports (animation-timeline: view()) {
    .beat-inner {
      animation: beat-rise linear both;
      animation-timeline: view();
      animation-range: entry 10% entry 45%;
    }
    @keyframes beat-rise {
      from { opacity: 0.15; transform: translateY(28px); }
      to   { opacity: 0.35; transform: translateY(12px); }
    }
    .beat.active .beat-inner { animation: none; }
  }

  /* Reduced motion / minimal FX: every beat rests fully legible, no travel. */
  @media (prefers-reduced-motion: reduce) {
    .beat-inner {
      opacity: 1 !important;
      transform: none !important;
      transition: none;
      animation: none !important;
    }
  }
  :global([data-fx='minimal']) .beat-inner {
    opacity: 1;
    transform: none;
    transition: none;
    animation: none;
  }
</style>
