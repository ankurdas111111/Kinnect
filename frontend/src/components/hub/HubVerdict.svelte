<script>
  /**
   * HubVerdict — the honest one-sentence family status. The Hub's LCP hero.
   *
   * Replaces the fabricated "safety score". States what a person would say:
   * "Everyone's settled", "Priya is heading to Home", "Dad hasn't updated in a
   * while", "SOS — Aarav needs help". One inline status-colored word carries the
   * only status color; the rest stays calm.
   *
   * Pure presentation — the verdict object is computed by lib/hubStatus.js.
   */
  import { allowMotion } from '../../lib/stores/effects.js';

  /** @type {{ verdict: { tone:'safe'|'caution'|'alert', word:string, sentence:string, detail:string }, onopen?: () => void }} */
  let { verdict, onopen } = $props();

  // Split the sentence so the status word can be tinted inline when it appears.
  let parts = $derived(splitAroundWord(verdict.sentence, verdict.word));

  function splitAroundWord(sentence, word) {
    if (!word || !sentence) return { before: sentence || '', hit: '', after: '' };
    const i = sentence.indexOf(word);
    if (i < 0) return { before: sentence, hit: '', after: '' };
    return { before: sentence.slice(0, i), hit: word, after: sentence.slice(i + word.length) };
  }

  let interactive = $derived(verdict.tone === 'alert' && typeof onopen === 'function');
</script>

{#if interactive}
  <button class="verdict verdict-{verdict.tone}" class:calm={!$allowMotion}
    onclick={onopen} aria-label="{verdict.sentence}. Open the map">
    {@render body()}
  </button>
{:else}
  <section class="verdict verdict-{verdict.tone}" class:calm={!$allowMotion}>
    {@render body()}
  </section>
{/if}

{#snippet body()}
  <div class="verdict-tint" aria-hidden="true"></div>
  <p class="verdict-line" aria-live="polite">
    {parts.before}<span class="verdict-word">{parts.hit}</span>{parts.after}
  </p>
  <p class="verdict-detail">{verdict.detail}</p>
{/snippet}

<style>
  .verdict {
    position: relative;
    display: block; width: 100%; text-align: left;
    padding: var(--space-5); margin: 0;
    border: 1px solid var(--border-default);
    border-left: 3px solid var(--verdict-accent, var(--success-500));
    border-radius: var(--radius-lg, 16px);
    background: var(--glass-bg);
    backdrop-filter: blur(14px); -webkit-backdrop-filter: blur(14px);
    overflow: hidden; color: inherit; font: inherit;
    -webkit-tap-highlight-color: transparent;
  }
  button.verdict { cursor: pointer; }
  button.verdict:focus-visible { outline: 2px solid var(--verdict-accent, var(--primary-400)); outline-offset: 2px; }

  /* Per-tone accent + ambient wash (opacity-only crossfade between two stacked layers) */
  .verdict-safe    { --verdict-accent: var(--success-500); --verdict-wash: var(--success-500-08); }
  .verdict-caution { --verdict-accent: var(--warning-500); --verdict-wash: var(--warning-500-08); }
  .verdict-alert   { --verdict-accent: var(--danger-500); --verdict-wash: var(--danger-500-12); }

  .verdict-tint {
    position: absolute; inset: 0; pointer-events: none;
    background: radial-gradient(ellipse 90% 120% at 0% 0%, var(--verdict-wash) 0%, transparent 70%);
    transition: background var(--duration-slow, 400ms) var(--ease-out, ease);
  }

  .verdict-line {
    position: relative; margin: 0 0 var(--space-1);
    font-family: var(--font-display, system-ui);
    font-size: clamp(1.4rem, 4.5vw, 2rem); font-weight: 700;
    letter-spacing: -0.02em; line-height: 1.2; color: var(--text-primary);
  }
  .verdict-word { color: var(--verdict-accent); }

  .verdict-detail {
    position: relative; margin: 0;
    font-size: var(--text-sm, 13px); color: var(--text-tertiary);
    font-variant-numeric: tabular-nums;
  }

  /* Gentle "breathing" accent for the alert state only — a call for attention,
     never alarming. GPU-only, gated by calm mode + reduced motion. */
  .verdict-alert:not(.calm) .verdict-tint { animation: verdict-breathe 2.4s ease-in-out infinite; }
  @keyframes verdict-breathe { 0%,100% { opacity: 1; } 50% { opacity: 0.55; } }

  @media (prefers-reduced-motion: reduce) {
    .verdict-tint { animation: none !important; transition: none !important; }
  }
</style>
