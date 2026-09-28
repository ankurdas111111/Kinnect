<script>
  import Router from 'svelte-spa-router';
  import { wrap } from 'svelte-spa-router/wrap';
  import { onMount } from 'svelte';
  import { loadSession } from './lib/stores/auth.js';
  import { familyVerdict } from './lib/stores/verdict.js';
  import Login from './pages/Login.svelte';
  import MainApp from './pages/MainApp.svelte';
  import Toast from './components/primitives/Toast.svelte';

  // Login and MainApp are eager (hot path); every other route lazy-loads.
  const routes = {
    '/': MainApp,
    '/landing': wrap({ asyncComponent: () => import('./pages/Landing.svelte') }),
    '/dashboard': wrap({ asyncComponent: () => import('./pages/FamilyDashboard.svelte') }),
    '/login': Login,
    '/register': wrap({ asyncComponent: () => import('./pages/Register.svelte') }),
    '/monitoring': wrap({ asyncComponent: () => import('./pages/Monitoring.svelte') }),
    '/emergency': wrap({ asyncComponent: () => import('./pages/EmergencyProfile.svelte') }),
    '/replay': wrap({ asyncComponent: () => import('./pages/RoutePlayback.svelte') }),
    '/activity': wrap({ asyncComponent: () => import('./pages/ActivityFeed.svelte') }),
    '/checkins': wrap({ asyncComponent: () => import('./pages/CheckinSchedule.svelte') }),
    '/add-contact/:code': wrap({ asyncComponent: () => import('./pages/AddContact.svelte') }),
    '/live/:token': wrap({ asyncComponent: () => import('./pages/LiveViewer.svelte') }),
    '/watch/:token': wrap({ asyncComponent: () => import('./pages/WatchViewer.svelte') }),
    '/m/:token': wrap({ asyncComponent: () => import('./pages/SecretChatViewer.svelte') })
  };

  onMount(() => {
    // Fire-and-forget: loadSession updates the authUser/authLoading stores internally.
    // The Router renders immediately so Map.svelte can begin initialising in parallel
    // with the auth API call. MainApp.svelte's reactive guard redirects to /login once
    // authLoading resolves and authUser is null.
    loadSession();
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.register('/sw.js').catch(() => {});
    }
    // Verdict tone → <html data-tone> — nav chrome (tab bar, sheet, sidebar,
    // edge lines) tints via CSS custom-property indirection, zero prop-drilling.
    const unsubTone = familyVerdict.subscribe((v) => {
      document.documentElement.setAttribute('data-tone', v?.tone || 'safe');
    });
    return () => unsubTone();
  });

  function conditionsFailed() {
    window.location.hash = '#/login';
  }

  // View Transitions API — wrap hash-based SPA navigation so route
  // changes feel native. Progressive enhancement: falls back to instant replace on
  // unsupported browsers (Firefox, older Safari). Only fires for hash changes that
  // are actual navigations, not anchor scrolls.
  // We intercept clicks on <a href="#/..."> links at the document level so we catch
  // both Router-managed links and any manual window.location.hash assignments that
  // go through anchor clicks.
  // Callers sit behind the onMount support guard below, so no fallback here.
  function wrapWithViewTransition(fn) {
    document.startViewTransition(fn);
  }

  onMount(() => {
    if (!document.startViewTransition) return; // No-op on unsupported browsers

    // Intercept anchor clicks that target hash routes
    function onLinkClick(e) {
      const anchor = e.composedPath().find(
        el => el instanceof HTMLAnchorElement || el instanceof HTMLAreaElement
      );
      if (!anchor) return;
      const href = anchor.getAttribute('href') || '';
      if (!href.startsWith('#/')) return;
      // Same hash — no transition needed
      if (href === window.location.hash) return;

      e.preventDefault();
      wrapWithViewTransition(() => {
        window.location.hash = href.slice(1); // '#/foo' → '/foo' via hash
      });
    }

    document.addEventListener('click', onLinkClick);
    return () => document.removeEventListener('click', onLinkClick);
  });
</script>

<!-- Skip navigation link — renders off-screen, visible on focus -->
<!-- This satisfies WCAG 2.4.1 (Bypass Blocks) for keyboard and screen reader users -->
<a href="#main-content" class="skip-nav">Skip to main content</a>

<main id="main-content">
  <!-- Without a boundary, a single thrown render error anywhere in the tree
       unmounts the whole app and leaves a blank page with no way back. In a
       product whose job is telling you your family is safe, a white screen is
       the worst possible failure mode: it is indistinguishable from "everyone
       is fine". Keep the shell alive and offer a route out. -->
  <svelte:boundary onerror={(e) => console.error('[route boundary]', e)}>
    <Router {routes} on:conditionsFailed={conditionsFailed} />

    {#snippet failed(error, reset)}
      <div class="route-error" role="alert">
        <h1>Something went wrong on this screen.</h1>
        <p>The rest of Kinnect is still running. You can retry this screen, or go back to the map.</p>
        <div class="route-error-actions">
          <button class="route-error-btn" onclick={reset}>Try again</button>
          <a class="route-error-link" href="#/" onclick={() => setTimeout(reset, 0)}>Back to the map</a>
        </div>
        {#if error?.message}
          <p class="route-error-detail">{error.message}</p>
        {/if}
      </div>
    {/snippet}
  </svelte:boundary>
</main>

<Toast />

<style>
  /* Route-level error fallback. Uses HEARTH tokens so a failure still looks
     like Kinnect rather than an unstyled browser error. */
  .route-error {
    max-width: 34rem;
    margin: 0 auto;
    padding: var(--space-8, 32px) var(--space-5, 20px);
    text-align: center;
    color: var(--text-primary, #38332e);
  }
  .route-error h1 {
    font-family: var(--font-serif, Newsreader, serif);
    font-style: italic;
    font-weight: 400;
    font-size: var(--text-2xl, 1.75rem);
    margin: 0 0 var(--space-3, 12px);
  }
  .route-error p {
    color: var(--text-secondary, #6b625a);
    margin: 0 0 var(--space-5, 20px);
  }
  .route-error-actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-3, 12px);
    justify-content: center;
  }
  .route-error-btn,
  .route-error-link {
    min-height: 44px; /* touch floor */
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding-inline: var(--space-5, 20px);
    border-radius: var(--radius-lg, 12px);
    font-size: var(--text-base, 1rem);
    font-weight: 600;
    text-decoration: none;
    cursor: pointer;
  }
  .route-error-btn {
    border: none;
    background: var(--primary-600, #b0511f);
    color: var(--text-on-primary, #fff);
  }
  .route-error-link {
    border: 1px solid var(--border-default, #e6dcd2);
    background: transparent;
    color: var(--text-primary, #38332e);
  }
  .route-error-detail {
    margin-top: var(--space-5, 20px);
    font-size: var(--text-xs, 0.75rem);
    color: var(--text-tertiary, #8a7268);
    word-break: break-word;
  }

  /* Skip nav: visually hidden until focused, then overlays the top of the page */
  .skip-nav {
    position: fixed;
    top: var(--space-3, 12px);
    left: var(--space-3, 12px);
    z-index: var(--z-topmost, 9000);
    padding: var(--space-2, 8px) var(--space-4, 16px);
    /* 44px minimum touch target (WCAG 2.5.5 / Kinnect design rules) */
    min-height: 44px;
    display: inline-flex;
    align-items: center;
    background: var(--primary-500);
    color: var(--text-on-primary, #ffffff);
    font-family: var(--font-display, system-ui, sans-serif);
    font-size: var(--text-sm, 13px);
    font-weight: 700;
    border-radius: var(--radius-md, 8px);
    text-decoration: none;
    /* GPU-only: only transform + opacity change, no layout repaint */
    transform: translateY(-120%);
    opacity: 0;
    transition:
      transform 180ms var(--ease-out, cubic-bezier(0.4, 0, 0.2, 1)),
      opacity 180ms var(--ease-out, cubic-bezier(0.4, 0, 0.2, 1));
    pointer-events: none;
    box-shadow: var(--shadow-primary);
  }

  .skip-nav:focus {
    transform: translateY(0);
    opacity: 1;
    pointer-events: auto;
    outline: 3px solid rgba(255, 255, 255, 0.8);
    outline-offset: 2px;
  }

  /* main wrapper — transparent passthrough, required for skip-nav target */
  main {
    display: contents;
  }

  @media (prefers-reduced-motion: reduce) {
    .skip-nav {
      transition: none;
    }
  }
</style>
