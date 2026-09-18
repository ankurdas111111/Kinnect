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
  /* Family Circle: the verdict is a FRAMELESS typographic statement at the
     crest — no card, no border, no glass. The serif italic voice comes from
     the global .verdict-line rule (tokens-hearth.css). */
  .verdict {
    position: relative;
    display: block; width: 100%; text-align: left;
    padding: var(--space-4) 0; margin: 0;
    border: none;
    background: transparent;
    color: inherit; font: inherit;
    -webkit-tap-highlight-color: transparent;
  }
  button.verdict { cursor: pointer; }
  button.verdict:focus-visible { outline: 2px solid var(--verdict-accent, var(--primary-500)); outline-offset: 2px; border-radius: var(--radius-sm); }

  /* Sage settled · ochre needs-a-look · vermilion strictly SOS (alert). */
  .verdict-safe    { --verdict-accent: var(--success-600); --verdict-wash: transparent; }
  .verdict-caution { --verdict-accent: var(--warning-600); --verdict-wash: color-mix(in oklch, var(--warning-500) 7%, transparent); }
  .verdict-alert   { --verdict-accent: var(--danger-500);  --verdict-wash: color-mix(in oklch, var(--danger-500) 9%, transparent); }

  .verdict-tint {
    position: absolute; inset: 0; pointer-events: none;
    background: radial-gradient(ellipse 90% 120% at 0% 0%, var(--verdict-wash) 0%, transparent 70%);
    transition: background var(--duration-slow) var(--ease-out);
  }

  .verdict-line {
    position: relative; margin: 0 0 var(--space-1);
    font-size: clamp(var(--text-2xl), 3vw, var(--text-3xl));
    line-height: 1.4; color: var(--text-primary);
  }
  .verdict-word { color: var(--verdict-accent); font-style: inherit; }

  .verdict-detail {
    position: relative; margin: 0;
    font-size: var(--text-base); line-height: 1.5; color: var(--text-secondary);
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
