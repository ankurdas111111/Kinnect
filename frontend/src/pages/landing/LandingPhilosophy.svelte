<script>
  /**
   * "Dignity First" — why Kinnect feels like family, not surveillance.
   * Three warm pillar cards on the surface-container-low band.
   */
  import LandingIcon from './LandingIcon.svelte';
</script>

<section class="philosophy" id="philosophy" aria-labelledby="philosophy-title">
  <div class="wrap">
    <div class="intro">
      <span class="eyebrow">Dignity First</span>
      <h2 id="philosophy-title">
        Why Kinnect feels like <em>family</em>, not surveillance.
      </h2>
      <p>
        Traditional tracking apps act like digital parole officers. We designed
        Kinnect for people who love, trust, and care for one another.
      </p>
    </div>

    <div class="pillar-grid">
      <article class="pillar">
        <span class="pillar-ic"><LandingIcon name="quote" size={24} /></span>
        <h3>One honest sentence beats a dashboard</h3>
        <p>
          You don’t need speed telemetry, battery drain warnings, or
          surveillance breadcrumb trails. You just need to glance down and
          know: <em>Everyone is where they want to be.</em>
        </p>
        <span class="pillar-note">Zero KPI charts or vanity alerts</span>
      </article>

      <article class="pillar">
        <span class="pillar-ic"><LandingIcon name="bell-paused" size={24} /></span>
        <h3>Calm by default, loud only for SOS</h3>
        <p>
          No buzzing when a child steps 2 feet outside a geofence. The app
          maintains total quiet, unmuting only when a family member triggers
          the intentional 3-second SOS protocol.
        </p>
        <span class="pillar-note">Eliminating alarm fatigue entirely</span>
      </article>

      <article class="pillar">
        <span class="pillar-ic"><LandingIcon name="lock-open" size={24} /></span>
        <h3>Mutual care, never surveillance</h3>
        <p>
          Built strictly for existing intimacy. Location data stays on your
          devices with local encryption keys. No brokers, no driving scores
          sold to insurers, no ad networks.
        </p>
        <span class="pillar-note">100% peer encrypted circle</span>
      </article>
    </div>
  </div>
</section>

<style>
  .philosophy {
    background: var(--surface-2);
    border-block: 1px solid color-mix(in srgb, var(--border-strong) 30%, transparent);
    padding-block: clamp(var(--space-16), 8vw, 6rem);
    scroll-margin-top: calc(var(--space-16) + var(--space-6)); /* sticky nav clearance */
  }

  .wrap {
    max-width: 72rem; /* design container */
    margin-inline: auto;
    padding-inline: clamp(var(--space-4), 4vw, var(--space-6));
  }

  .intro {
    max-width: 42rem;
    margin: 0 auto var(--space-16);
    text-align: center;
  }

  .eyebrow {
    display: block;
    margin-bottom: var(--space-2);
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.14em;
    color: var(--primary-700);
  }

  h2 {
    margin: 0 0 var(--space-4);
    font-family: var(--font-serif);
    font-weight: 400;
    /* design section headline: 28px → 36px */
    font-size: clamp(var(--text-3xl), 1.4rem + 1.6vw, var(--text-4xl));
    line-height: 1.2;
    letter-spacing: -0.02em;
    color: var(--text-primary);
  }

  h2 em {
    font-style: italic;
    color: var(--primary-700);
  }

  .intro p {
    margin: 0;
    font-size: var(--text-base);
    line-height: var(--leading-normal);
    color: var(--text-secondary);
  }

  .pillar-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--space-8);
  }

  .pillar {
    display: flex;
    flex-direction: column;
    padding: var(--space-8);
    background: var(--paper-warm);
    border: 1px solid color-mix(in srgb, var(--border-strong) 45%, transparent);
    border-radius: var(--radius-card);
    box-shadow: var(--shadow-xs);
    transition: box-shadow var(--duration-normal) var(--ease-out);
  }

  .pillar:hover {
    box-shadow: var(--shadow-md);
  }

  .pillar-ic {
    width: var(--space-12);
    height: var(--space-12);
    margin-bottom: var(--space-6);
    border-radius: var(--radius-md);
    background: var(--surface-3);
    color: var(--primary-700);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .pillar h3 {
    margin: 0 0 var(--space-3);
    font-family: var(--font-sans);
    font-size: var(--text-xl);
    font-weight: 600;
    line-height: var(--leading-tight);
    color: var(--text-primary);
  }

  .pillar p {
    margin: 0 0 var(--space-6);
    font-size: var(--text-base);
    line-height: var(--leading-relaxed);
    color: var(--text-secondary);
    flex-grow: 1;
  }

  .pillar p em {
    font-style: italic;
  }

  .pillar-note {
    padding-top: var(--space-4);
    border-top: 1px solid color-mix(in srgb, var(--border-strong) 20%, transparent);
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--primary-700);
  }

  @media (min-width: 48rem) {
    .pillar-grid {
      grid-template-columns: repeat(3, 1fr);
    }
  }
</style>
