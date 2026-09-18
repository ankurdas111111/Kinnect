<script>
  /**
   * Landing — the Stitch "Kinnect Hearth" editorial landing.
   *
   * Structure (design order, reproduced faithfully):
   *   sticky nav → hero (headline + calm product vignette) → Dignity First
   *   pillars → Walk With Me split → Emergency SOS path → invitation callout
   *   → footer.
   *
   * All colour comes from the Hearth semantic tokens, so night mode follows
   * automatically. Vermilion is confined to the SOS section; everything else
   * speaks in paper, ink, ember and sage.
   *
   * Section navigation scrolls via scrollIntoView instead of changing
   * location.hash — the hash belongs to svelte-spa-router.
   */
  import { onMount } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { landingIsBlocked } from '../lib/entryPoint.js';
  import { prefersReducedMotion } from '../lib/deviceCapability.js';
  import LandingNav from './landing/LandingNav.svelte';
  import LandingHero from './landing/LandingHero.svelte';
  import LandingPhilosophy from './landing/LandingPhilosophy.svelte';
  import LandingWalk from './landing/LandingWalk.svelte';
  import LandingSos from './landing/LandingSos.svelte';
  import LandingInvite from './landing/LandingInvite.svelte';
  import LandingFooter from './landing/LandingFooter.svelte';

  onMount(() => {
    // The native app must never show marketing. Someone who installed
    // Kinnect does not need to be sold it, and a store reviewer landing on
    // a pitch page instead of a sign-in is a rejection risk.
    if (landingIsBlocked()) { push('/login'); return; }
    document.title = 'Kinnect — a quiet map for the people you love';
    return () => { document.title = 'Kinnect'; };
  });

  /** Scroll to a landing section without touching the hash router. */
  function goTo(event, id) {
    event.preventDefault();
    const el = document.getElementById(id);
    if (!el) return;
    el.scrollIntoView({
      behavior: prefersReducedMotion() ? 'auto' : 'smooth',
      block: 'start',
    });
  }
</script>

<svelte:head>
  <meta name="description" content="Kinnect is family location sharing without the noise: your people as a few pebbles, one honest sentence, and a hold-to-send SOS." />
</svelte:head>

<div class="landing" id="top">
  <LandingNav {goTo} />
  <LandingHero {goTo} />
  <LandingPhilosophy />
  <LandingWalk />
  <LandingSos />
  <LandingInvite />
  <LandingFooter {goTo} />
</div>

<style>
  .landing {
    min-height: 100vh;
    background: var(--paper);
    color: var(--text-primary);
    font-family: var(--font-sans);
    font-size: var(--text-base);
    line-height: var(--leading-normal);
    /* Belt-and-braces: nothing inside may create horizontal scroll */
    overflow-x: clip;
  }

  .landing :global(::selection) {
    background: var(--primary-100);
    color: var(--primary-700);
  }

  .landing :global(a:focus-visible),
  .landing :global(button:focus-visible),
  .landing :global(input:focus-visible) {
    outline: 2px solid var(--primary-500);
    outline-offset: 2px;
  }

  @media (prefers-reduced-motion: reduce) {
    .landing :global(*),
    .landing :global(*::before),
    .landing :global(*::after) {
      animation-duration: 0.01ms !important;
      animation-iteration-count: 1 !important;
      transition-duration: 0.01ms !important;
    }
  }
</style>
