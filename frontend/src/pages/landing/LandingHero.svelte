<script>
  /**
   * Hero — status pill, serif headline, subcopy, CTA pair, and the calm
   * product vignette (LandingHeroCard).
   */
  import LandingIcon from './LandingIcon.svelte';
  import LandingHeroCard from './LandingHeroCard.svelte';

  let { goTo } = $props();
</script>

<section class="hero" aria-labelledby="hero-title">
  <div class="hero-copy">
    <div class="hero-status">
      <span class="status-dot" aria-hidden="true"></span>
      Elena, Meera, &amp; Dad are currently safe in Northcote
    </div>

    <h1 class="hero-title" id="hero-title">
      Your family as a few pebbles,<br class="br-wide" />
      <em>one honest sentence</em>, and nothing else.
    </h1>

    <p class="hero-sub">
      A quiet location-sharing and SOS companion for people who already trust
      each other. Calm by default, loud only when something is truly wrong.
      No feeds. No pings. No battery shaming.
    </p>

    <div class="hero-cta-row">
      <a class="hero-start" href="#/register">Start your circle</a>
      <a class="hero-watch" href="#walk-with-me" onclick={(e) => goTo(e, 'walk-with-me')}>
        <span class="watch-ic"><LandingIcon name="play" size={20} /></span>
        <span>Watch how it feels</span>
      </a>
    </div>
  </div>

  <LandingHeroCard />
</section>

<style>
  .hero {
    max-width: 72rem; /* design container */
    margin-inline: auto;
    padding:
      clamp(var(--space-10), 7vw, var(--space-16))
      clamp(var(--space-4), 4vw, var(--space-6))
      clamp(var(--space-12), 8vw, 7rem);
  }

  .hero-copy {
    max-width: 48rem; /* design: hero text measure */
    margin-inline: auto;
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .hero-status {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    margin: 0 0 var(--space-8);
    padding: var(--space-1-5) var(--space-3-5);
    border-radius: var(--radius-full);
    background: var(--surface-3);
    border: 1px solid var(--border-subtle);
    color: var(--text-primary);
    font-size: var(--text-xs);
    font-weight: 500;
    box-shadow: var(--shadow-xs);
  }

  .status-dot {
    width: var(--space-2);
    height: var(--space-2);
    border-radius: var(--radius-full);
    background: var(--success-500);
    animation: dot-breathe 2.4s var(--ease-in-out) infinite; /* GPU-only */
    flex-shrink: 0;
  }

  @keyframes dot-breathe {
    0%, 100% { opacity: 1;    transform: scale(1);   }
    50%      { opacity: 0.55; transform: scale(0.8); }
  }

  .hero-title {
    margin: 0 0 var(--space-6);
    font-family: var(--font-serif);
    font-weight: 400;
    /* design headline-hero scale: 30px mobile → 44px desktop */
    font-size: clamp(1.875rem, 1.1rem + 3vw, 2.75rem);
    line-height: 1.18;
    letter-spacing: -0.02em;
    color: var(--text-primary);
  }

  .hero-title em {
    font-style: italic;
    color: var(--primary-700);
  }

  .br-wide { display: none; }

  .hero-sub {
    margin: 0 0 var(--space-10);
    max-width: 42rem; /* design: subcopy measure */
    font-size: var(--text-lg);
    line-height: var(--leading-relaxed);
    color: var(--text-secondary);
  }

  .hero-cta-row {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    margin-bottom: clamp(var(--space-12), 6vw, var(--space-16));
  }

  .hero-start {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    min-height: calc(var(--space-12) + var(--space-1)); /* 52px primary CTA */
    padding: var(--space-3) var(--space-8);
    background: var(--primary-500);
    color: var(--text-on-primary);
    font-size: var(--text-base);
    font-weight: 600;
    text-decoration: none;
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-md);
    transition:
      background-color var(--duration-normal) var(--ease-out),
      box-shadow var(--duration-normal) var(--ease-out),
      transform var(--duration-fast) var(--ease-out);
  }

  .hero-start:hover {
    background: var(--primary-600);
    box-shadow: var(--shadow-primary);
  }

  .hero-start:active {
    transform: scale(0.96);
  }

  .hero-watch {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    width: 100%;
    min-height: calc(var(--space-12) + var(--space-1));
    padding: var(--space-3) var(--space-6);
    background: var(--paper-warm);
    color: var(--text-primary);
    font-size: var(--text-base);
    font-weight: 500;
    text-decoration: none;
    border: 1px solid color-mix(in srgb, var(--border-strong) 60%, transparent);
    border-radius: var(--radius-md);
    transition: background-color var(--duration-normal) var(--ease-out);
  }

  .hero-watch:hover {
    background: var(--surface-3);
  }

  .watch-ic {
    color: var(--primary-500);
    display: inline-flex;
  }

  @media (min-width: 40rem) {
    .br-wide { display: inline; }

    .hero-cta-row { flex-direction: row; }

    .hero-start,
    .hero-watch { width: auto; }
  }
</style>
