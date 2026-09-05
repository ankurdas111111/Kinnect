<script>
  /**
   * LandingNav — minimal fixed nav for the landing page.
   *
   * Transparent over the hero, gains a glass floor once the page scrolls
   * (opacity-only crossfade on a ::before layer — never transitions
   * backdrop-filter). Wordmark scrolls back to top; the only links are the
   * two real destinations (login / register). No hash anchors — the SPA is
   * hash-routed, so in-page `#section` hrefs would fight the router.
   */
  import { onMount } from 'svelte';
  import Button from '../primitives/Button.svelte';
  import { navigate } from '../../lib/viewTransition.js';
  import { prefersReducedMotion } from '../../lib/deviceCapability.js';

  let scrolled = $state(false);

  onMount(() => {
    const onScroll = () => {
      const next = window.scrollY > 12;
      if (next !== scrolled) scrolled = next;
    };
    onScroll();
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  });

  function toTop() {
    window.scrollTo({ top: 0, behavior: prefersReducedMotion() ? 'auto' : 'smooth' });
  }
</script>

<nav class="lnav" class:scrolled aria-label="Landing navigation">
  <div class="lnav-inner">
    <button class="lnav-brand" onclick={toTop} aria-label="Kinnect — back to top">
      <span class="lnav-beacon" aria-hidden="true"></span>
      <span class="lnav-wordmark">Kinnect</span>
    </button>

    <div class="lnav-actions">
      <Button variant="ghost" size="md" on:click={() => navigate('/login')}>
        Sign in
      </Button>
      <span class="lnav-cta">
        <Button variant="primary" size="md" on:click={() => navigate('/register')}>
          Start free
        </Button>
      </span>
    </div>
  </div>
</nav>

<style>
  .lnav {
    position: fixed;
    top: 0; left: 0; right: 0;
    z-index: var(--z-nav, 800);
    padding-top: env(safe-area-inset-top, 0px);
    isolation: isolate;
  }

  /* Glass floor — faded in as a layer so backdrop-filter itself never animates */
  .lnav::before {
    content: '';
    position: absolute;
    inset: 0;
    background: color-mix(in oklch, var(--surface-0) 78%, transparent);
    backdrop-filter: blur(16px) saturate(160%);
    -webkit-backdrop-filter: blur(16px) saturate(160%);
    border-bottom: 1px solid var(--border-subtle);
    opacity: 0;
    transition: opacity var(--duration-normal, 250ms) var(--ease-out, ease);
    z-index: -1;
  }
  .lnav.scrolled::before { opacity: 1; }

  .lnav-inner {
    width: min(92vw, 90rem);
    margin-inline: auto;
    padding: var(--space-2) var(--space-6);
    min-height: 60px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
  }

  @media (max-width: 767px) {
    .lnav-inner { width: 100%; padding: var(--space-2) var(--space-4); min-height: 56px; }
  }

  .lnav-brand {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    min-height: 44px;
    padding: var(--space-1) var(--space-2);
    margin-left: calc(-1 * var(--space-2));
    background: none;
    border: none;
    border-radius: var(--radius-md, 12px);
    cursor: pointer;
    color: var(--text-primary);
    -webkit-tap-highlight-color: transparent;
  }
  .lnav-brand:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }

  .lnav-beacon {
    width: 8px; height: 8px;
    border-radius: 50%;
    background: var(--primary-400);
    box-shadow: 0 0 10px color-mix(in oklch, var(--primary-400) 70%, transparent);
    animation: lnav-beacon-breathe 3s ease-in-out infinite;
  }

  @keyframes lnav-beacon-breathe {
    0%, 100% { opacity: 1; transform: scale(1); }
    50%      { opacity: 0.55; transform: scale(0.82); }
  }

  .lnav-wordmark {
    font-family: var(--font-display);
    font-size: var(--text-lg);
    font-weight: 800;
    letter-spacing: -0.02em;
  }

  .lnav-actions {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  @media (max-width: 420px) {
    .lnav-cta { display: none; } /* hero CTA is one glance below */
  }

  @media (prefers-reduced-motion: reduce) {
    .lnav::before { transition: none; }
    .lnav-beacon { animation: none; }
  }
  :global([data-fx='minimal']) .lnav-beacon { animation: none; }
</style>
