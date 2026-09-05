<script>
  /**
   * LandingFooter — compact closing footer.
   *
   * Only links to destinations that actually exist (login / register) — a
   * hash-routed SPA can't do `#section` anchors, and dead footer links read
   * as template filler. Everything else is honest, plain text.
   */
  import { navigate } from '../../lib/viewTransition.js';

  const PRODUCT_FACTS = ['Live family map', 'One-tap SOS', '30-day route replay', 'Secret chat'];
  const TRUST = ['End-to-end encrypted', 'No ads, ever', 'GDPR compliant'];
</script>

<footer class="lfoot" aria-label="Landing footer">
  <div class="lfoot-inner">
    <div class="lfoot-top">
      <div class="lfoot-brand">
        <p class="lfoot-wordmark">
          <span class="lfoot-beacon" aria-hidden="true"></span>Kinnect
        </p>
        <p class="lfoot-promise">
          The calm way to know your family is safe — a live map, a quiet
          watch, and one tap when it matters.
        </p>
      </div>

      <div class="lfoot-cols">
        <div class="lfoot-col">
          <p class="lfoot-col-title">Product</p>
          <ul class="lfoot-list">
            {#each PRODUCT_FACTS as fact}
              <li>{fact}</li>
            {/each}
          </ul>
        </div>

        <nav class="lfoot-col" aria-label="Account">
          <p class="lfoot-col-title">Get started</p>
          <ul class="lfoot-list">
            <li>
              <button class="lfoot-link" onclick={() => navigate('/register')}>
                Create account
              </button>
            </li>
            <li>
              <button class="lfoot-link" onclick={() => navigate('/login')}>
                Sign in
              </button>
            </li>
          </ul>
        </nav>
      </div>
    </div>

    <div class="lfoot-bottom">
      <p class="lfoot-copy">© 2026 Kinnect · Built for families, not advertisers</p>
      <ul class="lfoot-trust" aria-label="Privacy commitments">
        {#each TRUST as t}
          <li class="lfoot-trust-item">
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
              <circle cx="6" cy="6" r="5.5" stroke="currentColor" stroke-opacity="0.5"/>
              <polyline points="3.5,6 5,7.5 8.5,4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            {t}
          </li>
        {/each}
      </ul>
    </div>
  </div>
</footer>

<style>
  .lfoot {
    border-top: 1px solid var(--border-subtle);
    background: var(--surface-1);
    padding: clamp(var(--space-10), 4vw, 4rem) 0 var(--space-8);
    padding-bottom: max(var(--space-8), env(safe-area-inset-bottom));
  }

  .lfoot-inner {
    width: min(92vw, 90rem);
    margin-inline: auto;
    padding: 0 var(--space-6);
  }

  @media (max-width: 767px) {
    .lfoot-inner { width: 100%; padding: 0 var(--space-4); }
  }

  .lfoot-top {
    display: flex;
    justify-content: space-between;
    gap: var(--space-10);
    flex-wrap: wrap;
    margin-bottom: var(--space-8);
  }

  .lfoot-brand { max-width: 40ch; }

  .lfoot-wordmark {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-family: var(--font-display);
    font-size: var(--text-lg);
    font-weight: 800;
    letter-spacing: -0.02em;
    color: var(--text-primary);
    margin: 0 0 var(--space-3);
  }

  .lfoot-beacon {
    width: 7px; height: 7px;
    border-radius: 50%;
    background: var(--primary-400);
    box-shadow: 0 0 8px color-mix(in oklch, var(--primary-400) 60%, transparent);
  }

  .lfoot-promise {
    font-size: var(--text-sm);
    color: var(--text-tertiary);
    line-height: var(--leading-relaxed);
    margin: 0;
  }

  .lfoot-cols {
    display: flex;
    gap: clamp(var(--space-10), 6vw, 6rem);
    flex-wrap: wrap;
  }

  .lfoot-col-title {
    font-size: var(--text-xs);
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-secondary);
    margin: 0 0 var(--space-3);
  }

  .lfoot-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    font-size: var(--text-sm);
    color: var(--text-tertiary);
  }

  .lfoot-link {
    background: none;
    border: none;
    padding: var(--space-1) 0;
    min-height: 32px;
    font: inherit;
    color: var(--text-secondary);
    cursor: pointer;
    text-align: left;
    border-radius: var(--radius-sm, 6px);
    transition: color var(--duration-fast, 150ms) var(--ease-out, ease);
    -webkit-tap-highlight-color: transparent;
  }
  .lfoot-link:hover { color: var(--primary-400); }
  .lfoot-link:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }

  .lfoot-bottom {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--space-4);
    flex-wrap: wrap;
    padding-top: var(--space-5);
    border-top: 1px solid var(--border-subtle);
  }

  .lfoot-copy {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    margin: 0;
  }

  .lfoot-trust {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    gap: var(--space-5);
    flex-wrap: wrap;
  }

  .lfoot-trust-item {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    font-weight: 500;
  }
</style>
