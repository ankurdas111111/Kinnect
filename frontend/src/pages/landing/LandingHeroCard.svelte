<script>
  /**
   * Hero visual — a quiet map vignette with three pebbles, the Reassurance
   * Verdict card, and the quiet-standby row. Pure illustration: role="img",
   * nothing focusable inside.
   *
   * Palette-law note: the design shows the "Hold for SOS" affordance in
   * vermilion, but repo law reserves vermilion for the SOS section alone —
   * here the trigger is dormant (quiet standby), so it stays in ink tones.
   */
  import LandingIcon from './LandingIcon.svelte';
</script>

<figure
  class="hero-card mock-figure"
  id="interactive-preview"
  role="img"
  aria-label="Preview of a Kinnect circle: Dad is home, Meera is at the studio, Elena is walking home. The verdict reads that Elena is eight minutes away and accompanied by Meera, end-to-end encrypted, updated twenty seconds ago."
>
  <div class="card-ground" aria-hidden="true"></div>

  <div class="card-body">
    <div class="card-top">
      <span class="place">
        <LandingIcon name="cottage" size={16} />
        Northcote Sanctuary
      </span>
      <span class="nearby">
        <span class="nearby-dot"></span>
        All 3 nearby
      </span>
    </div>

    <div class="pebble-stage">
      <div class="pebble-group p-dad">
        <span class="pebble">D<span class="presence"></span></span>
        <span class="pebble-tag">Dad · Home</span>
      </div>

      <div class="pebble-group p-meera">
        <span class="pebble">M<span class="presence"></span></span>
        <span class="pebble-tag">Meera · Studio</span>
      </div>

      <div class="pebble-group p-elena">
        <span class="pebble-wrap">
          <span class="pebble-halo"></span>
          <span class="pebble pebble-ember">
            E
            <span class="walk-badge"><LandingIcon name="walk" size={12} /></span>
          </span>
        </span>
        <span class="walking-pill">
          <span class="ping-dot"></span>
          Elena is walking home
        </span>
      </div>
    </div>

    <div class="verdict-card">
      <span class="verdict-label">The Reassurance Verdict</span>
      <p class="verdict-voice">
        “Elena is walking home from Fitzroy Library. She is 8 minutes away and
        accompanied by Meera.”
      </p>
      <div class="verdict-meta">
        <span class="enc">
          <span class="ic-sage"><LandingIcon name="shield-check" size={14} /></span>
          End-to-end encrypted
        </span>
        <span>Updated 20 seconds ago</span>
      </div>
    </div>

    <div class="standby-row">
      <div class="standby-left">
        <span class="back-circle"><LandingIcon name="chevron-left" size={18} /></span>
        <div>
          <span class="standby-title">Quiet Standby Mode</span>
          <p class="standby-sub">All notifications quieted. Instant escalation armed.</p>
        </div>
      </div>
      <div class="standby-right">
        <span class="standby-q">Emergency?</span>
        <span class="standby-hold"><span class="hold-dot"></span>Hold for SOS</span>
      </div>
    </div>
  </div>
</figure>

<style>
  .hero-card {
    position: relative;
    max-width: 56rem; /* design: hero mock width */
    margin: 0 auto;
    padding: clamp(var(--space-4), 3vw, var(--space-8));
    background: var(--paper-warm);
    border: 1px solid color-mix(in srgb, var(--border-strong) 45%, transparent);
    border-radius: var(--radius-card);
    box-shadow: var(--shadow-md);
    overflow: hidden;
  }

  /* Warm ambient map texture: paper tier + soft topographic dot grid */
  .card-ground {
    position: absolute;
    inset: 0;
    background-color: var(--surface-2);
    background-image: radial-gradient(
      color-mix(in srgb, var(--ink-3) 40%, transparent) 1px,
      transparent 1px
    );
    background-size: var(--space-6) var(--space-6);
    opacity: 0.8;
  }

  .card-body {
    position: relative;
    z-index: var(--z-base);
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .card-top {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-2);
    border-bottom: 1px solid color-mix(in srgb, var(--border-strong) 30%, transparent);
    margin-bottom: var(--space-8);
  }

  .place {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text-secondary);
  }

  .nearby {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--success-600);
    background: color-mix(in srgb, var(--success-500) 12%, transparent);
    padding: var(--space-1) var(--space-2-5);
    border-radius: var(--radius-full);
    white-space: nowrap;
  }

  .nearby-dot {
    width: var(--space-1-5);
    height: var(--space-1-5);
    border-radius: var(--radius-full);
    background: var(--success-500);
  }

  /* ── Pebble stage ──────────────────────────────────────────────────── */
  .pebble-stage {
    position: relative;
    width: 100%;
    height: clamp(15rem, 40vw, 20rem);
    margin-block: var(--space-4);
  }

  .pebble-group {
    position: absolute;
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .p-dad   { top: var(--space-8);     left: 25%;  transform: translateX(-50%); }
  .p-meera { top: var(--space-16);    right: 25%; transform: translateX(50%);  }
  .p-elena { bottom: var(--space-10); left: 50%;  transform: translateX(-50%); }

  .pebble {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: calc(var(--space-12) + var(--space-2));  /* 56px pebble */
    height: calc(var(--space-12) + var(--space-2));
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    color: var(--text-primary);
    font-weight: 600;
    font-size: var(--text-sm);
    border: 2px solid var(--card);
    box-shadow: var(--shadow-sm);
  }

  .presence {
    position: absolute;
    bottom: 0;
    right: 0;
    width: var(--space-3-5);
    height: var(--space-3-5);
    border-radius: var(--radius-full);
    background: var(--success-500);
    border: 2px solid var(--card);
  }

  .pebble-tag {
    margin-top: var(--space-2);
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text-secondary);
    background: color-mix(in srgb, var(--paper-warm) 90%, transparent);
    padding: var(--space-0-5) var(--space-2);
    border-radius: var(--radius-sm2);
    box-shadow: var(--shadow-xs);
    white-space: nowrap;
  }

  .pebble-wrap {
    position: relative;
    display: inline-flex;
  }

  /* Breathing halo — GPU-only (transform/opacity) */
  .pebble-halo {
    position: absolute;
    inset: calc(-1 * var(--space-3));
    border-radius: var(--radius-full);
    background: var(--primary-100);
    animation: halo-breathe 3.4s var(--ease-in-out) infinite;
  }

  @keyframes halo-breathe {
    0%, 100% { transform: scale(1);    opacity: 0.55; }
    50%      { transform: scale(1.18); opacity: 0.15; }
  }

  .pebble-ember {
    width: var(--space-16);  /* 64px — the active walker reads larger */
    height: var(--space-16);
    background: var(--primary-500);
    color: var(--text-on-primary);
    font-size: var(--text-base);
    box-shadow: var(--shadow-primary);
    z-index: var(--z-base);
  }

  .walk-badge {
    position: absolute;
    top: calc(-1 * var(--space-1));
    right: calc(-1 * var(--space-1));
    width: var(--space-5);
    height: var(--space-5);
    border-radius: var(--radius-full);
    background: var(--paper-warm);
    color: var(--primary-500);
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: var(--shadow-xs);
  }

  .walking-pill {
    margin-top: var(--space-2-5);
    display: inline-flex;
    align-items: center;
    gap: var(--space-1-5);
    background: var(--primary-100);
    color: var(--primary-700);
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-full);
    font-size: var(--text-xs);
    font-weight: 500;
    box-shadow: var(--shadow-xs);
    white-space: nowrap;
  }

  .ping-dot {
    width: var(--space-2);
    height: var(--space-2);
    border-radius: var(--radius-full);
    background: var(--primary-500);
    animation: ping 1.8s var(--ease-out) infinite;
  }

  @keyframes ping {
    0%        { transform: scale(1);   opacity: 1; }
    70%, 100% { transform: scale(1.8); opacity: 0; }
  }

  /* ── Verdict card ──────────────────────────────────────────────────── */
  .verdict-card {
    width: 100%;
    max-width: 36rem; /* design: verdict banner width */
    margin-block: var(--space-4);
    padding: var(--space-6);
    background: color-mix(in srgb, var(--card) 92%, transparent);
    border: 1px solid color-mix(in srgb, var(--border-strong) 45%, transparent);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-sm);
    text-align: center;
  }

  .verdict-label {
    display: block;
    margin-bottom: var(--space-1);
    font-size: var(--text-2xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.14em;
    color: var(--text-tertiary);
  }

  /* .verdict-voice (global) supplies the Newsreader italic register */
  .verdict-card p {
    margin: 0;
    font-size: clamp(1.25rem, 1rem + 1vw, 1.5rem); /* design verdict scale 20→24px */
    line-height: 1.45;
    color: var(--text-primary);
  }

  .verdict-meta {
    margin-top: var(--space-4);
    padding-top: var(--space-3);
    border-top: 1px solid color-mix(in srgb, var(--border-strong) 30%, transparent);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    flex-wrap: wrap;
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text-secondary);
  }

  .enc {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
  }

  .ic-sage {
    color: var(--success-500);
    display: inline-flex;
  }

  /* ── Quiet standby row ─────────────────────────────────────────────── */
  .standby-row {
    width: 100%;
    max-width: 36rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    flex-wrap: wrap;
    padding: var(--space-4);
    background: color-mix(in srgb, var(--surface-3) 60%, transparent);
    border: 1px solid color-mix(in srgb, var(--border-strong) 30%, transparent);
    border-radius: var(--radius-lg);
  }

  .standby-left {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
  }

  .back-circle {
    width: var(--space-10);
    height: var(--space-10);
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .standby-title {
    display: block;
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--text-primary);
  }

  .standby-sub {
    margin: 0;
    font-size: var(--text-xs);
    color: var(--text-secondary);
  }

  .standby-right {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .standby-q {
    display: none;
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text-secondary);
  }

  /* Dormant SOS affordance — deliberately quiet (vermilion is SOS-section-only) */
  .standby-hold {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1-5);
    padding: var(--space-2) var(--space-3-5);
    background: var(--paper-warm);
    color: var(--text-primary);
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-md);
    font-size: var(--text-xs);
    font-weight: 600;
    white-space: nowrap;
  }

  .hold-dot {
    width: var(--space-2);
    height: var(--space-2);
    border-radius: var(--radius-full);
    background: var(--text-tertiary);
  }

  @media (min-width: 40rem) {
    .standby-q { display: inline; }
  }
</style>
