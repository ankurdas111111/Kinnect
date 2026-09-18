<script>
  /**
   * "Walk With Me" — accompany in silence: copy + checklist + testimonial,
   * beside a Live Companion Session card with a three-step timeline.
   */
  import LandingIcon from './LandingIcon.svelte';
</script>

<section class="walk" id="walk-with-me" aria-labelledby="walk-title">
  <div class="split">
    <div class="copy">
      <span class="badge">
        <LandingIcon name="walk" size={14} />
        Accompany in silence
      </span>

      <h2 id="walk-title">
        Walking home alone at night?<br />
        <em>You are never unaccompanied.</em>
      </h2>

      <p class="lede">
        With one tap, activate <strong>Walk With Me</strong>. Kinnect brings
        your chosen loved one onto a gentle, battery-optimized live beacon.
        No need to stay on a phone call in the rain.
      </p>

      <ul class="checklist">
        <li>
          <span class="check-ic"><LandingIcon name="check-circle" size={20} /></span>
          <p>
            <strong>Gentle Warm Pings:</strong> Subtle haptics every 4 minutes
            reassure the walker their companion is actively present.
          </p>
        </li>
        <li>
          <span class="check-ic"><LandingIcon name="check-circle" size={20} /></span>
          <p>
            <strong>Automatic Arrival Confirmation:</strong> Once stepping
            through your front door, the session winds down quietly with a
            gentle signoff.
          </p>
        </li>
      </ul>

      <div class="testimonial">
        <span class="ic-ember"><LandingIcon name="heart" size={20} /></span>
        <p>
          <em>“My daughter doesn’t have to pretend to be on a fake phone call
          when walking from the tram anymore.”</em> — Marcus, Melbourne
        </p>
      </div>
    </div>

    <figure
      class="session-card mock-figure"
      role="img"
      aria-label="A live Walk With Me companion session: Elena is heading to 142 St Georges Road; she left the library at 8:14pm, crossed Scotchmer Street at 8:22pm, and is expected inside her apartment gate at 8:26pm."
    >
      <div class="session-head">
        <div>
          <span class="session-label">Live Companion Session</span>
          <span class="session-title">Elena is heading to 142 St Georges Rd</span>
        </div>
        <span class="live-pill">Live · 3m remain</span>
      </div>

      <ol class="timeline">
        <li class="timeline-step">
          <span class="step-dot">1</span>
          <div>
            <span class="step-time">8:14 PM</span>
            <p class="step-title">Left Fitzroy North Public Library</p>
            <p class="step-sub">Battery at 68% · Route looks normal along tram line</p>
          </div>
        </li>
        <li class="timeline-step is-active">
          <span class="step-dot">2</span>
          <div>
            <span class="step-time">Active now (8:22 PM)</span>
            <p class="step-title">Crossed Scotchmer Street junction</p>
            <p class="step-sub">Meera sent a silent warm heart tap</p>
          </div>
        </li>
        <li class="timeline-step is-future">
          <span class="step-dot">3</span>
          <div>
            <span class="step-time">Estimated 8:26 PM</span>
            <p class="step-title">Arrive inside apartment gate</p>
            <p class="step-sub">Automated safe entry handover</p>
          </div>
        </li>
      </ol>
    </figure>
  </div>
</section>

<style>
  .walk {
    max-width: 72rem; /* design container */
    margin-inline: auto;
    padding: clamp(var(--space-16), 8vw, 6rem) clamp(var(--space-4), 4vw, var(--space-6));
    scroll-margin-top: calc(var(--space-16) + var(--space-6)); /* sticky nav clearance */
  }

  .split {
    display: grid;
    grid-template-columns: 1fr;
    gap: clamp(var(--space-8), 4vw, var(--space-12));
    align-items: center;
  }

  .badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    margin-bottom: var(--space-4);
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-full);
    background: var(--primary-100);
    color: var(--primary-700);
    font-size: var(--text-xs);
    font-weight: 500;
  }

  h2 {
    margin: 0 0 var(--space-6);
    font-family: var(--font-serif);
    font-weight: 400;
    font-size: clamp(var(--text-3xl), 1.4rem + 1.6vw, var(--text-4xl));
    line-height: 1.2;
    letter-spacing: -0.02em;
    color: var(--text-primary);
  }

  h2 em {
    font-style: italic;
    color: var(--primary-700);
  }

  .lede {
    margin: 0 0 var(--space-6);
    font-size: var(--text-base);
    line-height: var(--leading-relaxed);
    color: var(--text-secondary);
  }

  .lede strong {
    font-weight: 600;
    color: var(--text-primary);
  }

  .checklist {
    list-style: none;
    margin: 0 0 var(--space-8);
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .checklist li {
    display: flex;
    align-items: flex-start;
    gap: var(--space-3);
  }

  .check-ic {
    color: var(--success-500);
    flex-shrink: 0;
    margin-top: var(--space-0-5);
    display: inline-flex;
  }

  .checklist p {
    margin: 0;
    font-size: var(--text-base);
    line-height: var(--leading-normal);
    color: var(--text-primary);
  }

  .checklist strong {
    font-weight: 600;
  }

  .testimonial {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-4);
    background: var(--surface-3);
    border: 1px solid color-mix(in srgb, var(--border-strong) 30%, transparent);
    border-radius: var(--radius-lg);
  }

  .ic-ember {
    color: var(--primary-500);
    flex-shrink: 0;
    display: inline-flex;
  }

  .testimonial p {
    margin: 0;
    font-size: var(--text-base);
    line-height: var(--leading-normal);
    color: var(--text-secondary);
  }

  .testimonial em {
    font-style: italic;
  }

  /* ── Live Companion Session card ───────────────────────────────────── */
  .session-card {
    margin: 0;
    padding: clamp(var(--space-6), 3vw, var(--space-8));
    background: var(--paper-warm);
    border: 1px solid color-mix(in srgb, var(--border-strong) 45%, transparent);
    border-radius: var(--radius-card);
    box-shadow: var(--shadow-md);
  }

  .session-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    flex-wrap: wrap;
    padding-bottom: var(--space-4);
    margin-bottom: var(--space-6);
    border-bottom: 1px solid color-mix(in srgb, var(--border-strong) 20%, transparent);
  }

  .session-label {
    display: block;
    font-size: var(--text-xs);
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--primary-700);
  }

  .session-title {
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--text-primary);
  }

  .live-pill {
    padding: var(--space-1) var(--space-2-5);
    background: var(--primary-100);
    color: var(--primary-700);
    font-size: var(--text-xs);
    font-weight: 500;
    border-radius: var(--radius-full);
    white-space: nowrap;
  }

  .timeline {
    list-style: none;
    margin: 0;
    padding: 0;
    position: relative;
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .timeline::before {
    content: '';
    position: absolute;
    top: var(--space-3);
    bottom: var(--space-3);
    left: var(--space-3-5);
    width: var(--space-0-5);
    background: color-mix(in srgb, var(--border-strong) 40%, transparent);
  }

  .timeline-step {
    position: relative;
    display: flex;
    align-items: flex-start;
    gap: var(--space-4);
  }

  .step-dot {
    width: var(--space-7);
    height: var(--space-7);
    border-radius: var(--radius-full);
    background: var(--surface-3);
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: var(--text-xs);
    font-weight: 500;
    z-index: var(--z-base);
    flex-shrink: 0;
  }

  .is-active .step-dot {
    background: var(--primary-500);
    color: var(--text-on-primary);
    box-shadow: 0 0 0 var(--space-1) var(--primary-100);
  }

  .is-future .step-dot {
    background: var(--surface-inset);
    color: var(--text-tertiary);
  }

  .step-time {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-tertiary);
  }

  .is-active .step-time {
    color: var(--primary-700);
  }

  .step-title {
    margin: 0;
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--text-primary);
  }

  .is-future .step-title {
    color: var(--text-secondary);
  }

  .step-sub {
    margin: 0;
    font-size: var(--text-xs);
    color: var(--text-secondary);
  }

  .is-future .step-sub {
    color: var(--text-tertiary);
  }

  @media (min-width: 64rem) {
    .split {
      grid-template-columns: 1fr 1fr;
    }
  }
</style>
