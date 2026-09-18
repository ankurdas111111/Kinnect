<script>
  /**
   * Footer — wordmark + origin line, quick links (two section anchors and
   * the web sign-in route).
   */
  let { goTo } = $props();
</script>

<footer class="lp-footer">
  <div class="foot-inner">
    <div class="foot-brand">
      <span class="foot-wordmark">Kinnect</span>
      <span class="foot-sep" aria-hidden="true">·</span>
      <span class="foot-line">
        Built for families and loved ones. Crafted in Northcote, Melbourne.
      </span>
    </div>

    <nav class="foot-links" aria-label="Footer">
      <a href="#emergency" onclick={(e) => goTo(e, 'emergency')}>Emergency Protocol</a>
      <a href="#philosophy" onclick={(e) => goTo(e, 'philosophy')}>Open Code &amp; Encryption</a>
      <a href="#/login">Web App Sign In</a>
    </nav>
  </div>
</footer>

<style>
  .lp-footer {
    background: var(--paper-warm);
    border-top: 1px solid color-mix(in srgb, var(--border-strong) 30%, transparent);
    padding: var(--space-12) clamp(var(--space-4), 4vw, var(--space-6));
  }

  .foot-inner {
    max-width: 72rem; /* design container */
    margin-inline: auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-6);
    text-align: center;
  }

  .foot-brand {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1);
  }

  .foot-wordmark {
    font-family: var(--font-serif);
    font-style: italic;
    font-weight: 400;
    font-size: var(--text-xl);
    color: var(--primary-700);
  }

  .foot-sep {
    display: none;
    color: var(--text-tertiary);
  }

  .foot-line {
    font-size: var(--text-xs);
    color: var(--text-secondary);
  }

  .foot-links {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: var(--space-2) var(--space-6);
  }

  .foot-links a {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text-secondary);
    text-decoration: none;
    min-height: 44px; /* touch floor */
    display: inline-flex;
    align-items: center;
    transition: color var(--duration-normal) var(--ease-out);
  }

  .foot-links a:hover {
    color: var(--primary-700);
  }

  @media (min-width: 40rem) {
    .foot-brand {
      flex-direction: row;
      gap: var(--space-3);
    }

    .foot-sep { display: inline; }
  }

  @media (min-width: 48rem) {
    .foot-inner {
      flex-direction: row;
      justify-content: space-between;
      text-align: left;
    }
  }
</style>
