<script>
  /**
   * Sticky top bar — wordmark + "Hearth v1.4" tag, section links, auth CTAs.
   * `goTo` scrolls to a section without touching the hash router.
   */
  import LandingIcon from './LandingIcon.svelte';

  let { goTo } = $props();
</script>

<header class="lp-nav">
  <div class="nav-inner">
    <div class="brand">
      <span class="brand-dot" aria-hidden="true"></span>
      <a class="lp-wordmark" href="#top" onclick={(e) => goTo(e, 'top')}>Kinnect</a>
      <span class="lp-version">Hearth v1.4</span>
    </div>

    <nav class="nav-links" aria-label="Landing sections">
      <a href="#philosophy" onclick={(e) => goTo(e, 'philosophy')}>The Philosophy</a>
      <a href="#walk-with-me" onclick={(e) => goTo(e, 'walk-with-me')}>Walk With Me</a>
      <a href="#emergency" onclick={(e) => goTo(e, 'emergency')}>Emergency Path</a>
      <a href="#circle" onclick={(e) => goTo(e, 'circle')}>Android &amp; Web</a>
    </nav>

    <div class="nav-cta">
      <a class="lp-signin" href="#/login">Sign In</a>
      <a class="lp-create" href="#/register">
        <span>Create your Circle</span>
        <LandingIcon name="arrow-forward" size={16} />
      </a>
    </div>
  </div>
</header>

<style>
  .lp-nav {
    position: sticky;
    top: 0;
    z-index: var(--z-navbar);
    background: var(--paper-warm);
  }

  .nav-inner {
    max-width: 72rem; /* design container */
    margin-inline: auto;
    padding: var(--space-4) clamp(var(--space-4), 4vw, var(--space-6));
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
  }

  .brand {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
  }

  .brand-dot {
    width: var(--space-2-5);
    height: var(--space-2-5);
    border-radius: var(--radius-full);
    background: var(--primary-500);
    box-shadow: 0 0 0 var(--space-1) var(--primary-100);
    flex-shrink: 0;
  }

  .lp-wordmark {
    font-family: var(--font-serif);
    font-style: italic;
    font-weight: 400;
    font-size: var(--text-3xl);
    line-height: 1;
    letter-spacing: -0.01em;
    color: var(--primary-700);
    text-decoration: none;
    min-height: 44px; /* touch floor */
    display: inline-flex;
    align-items: center;
  }

  .lp-version {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text-secondary);
    background: var(--surface-3);
    padding: var(--space-0-5) var(--space-2);
    border-radius: var(--radius-full);
    white-space: nowrap;
    display: none;
  }

  .nav-links {
    display: none;
    align-items: center;
    gap: var(--space-8);
  }

  .nav-links a {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text-secondary);
    text-decoration: none;
    min-height: 44px; /* touch floor */
    display: inline-flex;
    align-items: center;
    transition: color var(--duration-normal) var(--ease-out);
  }

  .nav-links a:hover {
    color: var(--primary-700);
  }

  .nav-cta {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .lp-signin {
    display: none;
    align-items: center;
    justify-content: center;
    min-height: 44px; /* touch floor */
    padding-inline: var(--space-4);
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text-primary);
    text-decoration: none;
    border-radius: var(--radius-md);
    transition: color var(--duration-normal) var(--ease-out);
  }

  .lp-signin:hover {
    color: var(--primary-700);
  }

  .lp-create {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    min-height: var(--space-12); /* 48px accent-fill floor */
    padding: 0 var(--space-5);
    background: var(--primary-500);
    color: var(--text-on-primary);
    font-size: var(--text-sm);
    font-weight: 600;
    text-decoration: none;
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-sm);
    transition:
      background-color var(--duration-normal) var(--ease-out),
      box-shadow var(--duration-normal) var(--ease-out),
      transform var(--duration-fast) var(--ease-out);
  }

  .lp-create:hover {
    background: var(--primary-600);
    box-shadow: var(--shadow-md);
  }

  .lp-create:active {
    transform: scale(0.96);
  }

  @media (min-width: 40rem) {
    .lp-version { display: inline-block; }
    .lp-signin  { display: inline-flex; }
  }

  @media (min-width: 60rem) {
    .nav-links { display: flex; }
  }
</style>
