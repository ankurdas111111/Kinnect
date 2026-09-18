<script>
  /**
   * Emergency SOS path — the ONLY place vermilion (var(--danger-*)) may
   * appear on the landing. First-responder handover card + the 3-second
   * hold copy.
   */
  import LandingIcon from './LandingIcon.svelte';
</script>

<section class="sos" id="emergency" aria-labelledby="sos-title">
  <div class="split">
    <div class="copy">
      <span class="badge-sos">
        <LandingIcon name="sos-star" size={14} />
        Reserved strictly for true emergencies
      </span>

      <h2 id="sos-title">
        A 3-second hold that<br />
        <em>cuts through absolute silence.</em>
      </h2>

      <p class="lede">
        Because we ban vermilion and shrill alarms from routine everyday use,
        when an emergency trigger occurs, your circle answers immediately.
        No accidental triggers, no false alarms.
      </p>

      <ul class="sos-list">
        <li>
          <span class="sos-ic"><LandingIcon name="radial-hold" size={20} /></span>
          <p>
            <strong>Radial 3-Second Hold:</strong> Eliminates accidental pocket
            dials with progressive tactile vibration feedback.
          </p>
        </li>
        <li>
          <span class="sos-ic"><LandingIcon name="volume" size={20} /></span>
          <p>
            <strong>Critical Quiet-Hours Bypass:</strong> Rings unmuted through
            Android Focus and Bedtime modes, and lands as an urgent web push
            on every signed-in browser.
          </p>
        </li>
      </ul>
    </div>

    <figure
      class="emergency-card mock-figure"
      role="img"
      aria-label="First responder emergency handover card for Elena Vance: emergency contact Daniel Vance, blood type O positive, severe penicillin allergy, currently near 204 St Georges Road."
    >
      <div class="em-head">
        <span class="em-head-label">
          <span class="sos-dot"></span>
          First Responder Emergency Handover
        </span>
        <span class="em-armed">Armed &amp; Decrypted</span>
      </div>

      <div class="em-body">
        <div class="em-person">
          <div>
            <span class="em-name">Elena Vance</span>
            <p class="em-contact">Emergency Contact: Daniel Vance (+61 412 889 201)</p>
          </div>
          <span class="em-blood">Blood: O Pos</span>
        </div>

        <div class="em-grid">
          <div class="em-cell">
            <span class="em-cell-label">Known allergies</span>
            <span class="em-cell-val">Penicillin (Severe)</span>
          </div>
          <div class="em-cell">
            <span class="em-cell-label">Current location</span>
            <span class="em-cell-val">Near 204 St Georges Rd</span>
          </div>
        </div>

        <p class="em-note">
          <span class="em-note-ic"><LandingIcon name="sos-cross" size={16} /></span>
          Overrides Do-Not-Disturb on circle members’ phones instantly.
        </p>
      </div>
    </figure>
  </div>
</section>

<style>
  .sos {
    max-width: 72rem; /* design container */
    margin-inline: auto;
    padding: 0 clamp(var(--space-4), 4vw, var(--space-6)) clamp(var(--space-16), 8vw, 6rem);
    scroll-margin-top: calc(var(--space-16) + var(--space-6)); /* sticky nav clearance */
  }

  .split {
    display: grid;
    grid-template-columns: 1fr;
    gap: clamp(var(--space-8), 4vw, var(--space-12));
    align-items: center;
  }

  .badge-sos {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    margin-bottom: var(--space-4);
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-full);
    background: color-mix(in srgb, var(--danger-500) 10%, transparent);
    color: var(--danger-600);
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
    color: var(--danger-500); /* large display text — SOS voice */
  }

  .lede {
    margin: 0 0 var(--space-6);
    font-size: var(--text-base);
    line-height: var(--leading-relaxed);
    color: var(--text-secondary);
  }

  .sos-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .sos-list li {
    display: flex;
    align-items: flex-start;
    gap: var(--space-3);
  }

  .sos-ic {
    color: var(--danger-500);
    flex-shrink: 0;
    margin-top: var(--space-0-5);
    display: inline-flex;
  }

  .sos-list p {
    margin: 0;
    font-size: var(--text-base);
    line-height: var(--leading-normal);
    color: var(--text-primary);
  }

  .sos-list strong {
    font-weight: 600;
  }

  /* ── First-responder handover card ─────────────────────────────────── */
  .emergency-card {
    margin: 0;
    padding: clamp(var(--space-6), 3vw, var(--space-8));
    background: var(--paper-warm);
    border: 1px solid color-mix(in srgb, var(--border-strong) 45%, transparent);
    border-radius: var(--radius-card);
    box-shadow: var(--shadow-md);
  }

  .em-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    flex-wrap: wrap;
    padding-bottom: var(--space-4);
    margin-bottom: var(--space-6);
    border-bottom: 1px solid color-mix(in srgb, var(--border-strong) 30%, transparent);
  }

  .em-head-label {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--text-primary);
  }

  .sos-dot {
    width: var(--space-3);
    height: var(--space-3);
    border-radius: var(--radius-full);
    background: var(--danger-500);
    flex-shrink: 0;
  }

  .em-armed {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--danger-600);
    background: color-mix(in srgb, var(--danger-500) 12%, transparent);
    padding: var(--space-0-5) var(--space-2);
    border-radius: var(--radius-xs);
    white-space: nowrap;
  }

  .em-body {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    padding: var(--space-5);
    background: var(--surface-3);
    border: 1px solid color-mix(in srgb, var(--border-strong) 30%, transparent);
    border-radius: var(--radius-lg);
  }

  .em-person {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-3);
  }

  .em-name {
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--text-primary);
  }

  .em-contact {
    margin: 0;
    font-size: var(--text-xs);
    color: var(--text-secondary);
  }

  .em-blood {
    background: var(--paper-warm);
    color: var(--danger-600);
    font-size: var(--text-xs);
    font-weight: 600;
    padding: var(--space-1) var(--space-2-5);
    border-radius: var(--radius-sm2);
    box-shadow: var(--shadow-xs);
    white-space: nowrap;
  }

  .em-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
    gap: var(--space-3);
  }

  .em-cell {
    display: flex;
    flex-direction: column;
    gap: var(--space-0-5);
    padding: var(--space-2-5);
    background: var(--paper-warm);
    border: 1px solid color-mix(in srgb, var(--border-strong) 20%, transparent);
    border-radius: var(--radius-sm2);
  }

  .em-cell-label {
    font-size: var(--text-2xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-tertiary);
  }

  .em-cell-val {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text-primary);
  }

  .em-note {
    margin: 0;
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-3);
    background: color-mix(in srgb, var(--danger-500) 8%, transparent);
    border-radius: var(--radius-sm2);
    font-size: var(--text-xs);
    color: var(--danger-700);
  }

  .em-note-ic {
    color: var(--danger-500);
    flex-shrink: 0;
    display: inline-flex;
  }

  @media (min-width: 64rem) {
    .split {
      grid-template-columns: 1fr 1fr;
    }

    /* Design order on wide screens: card left, copy right */
    .emergency-card { order: -1; }
  }
</style>
