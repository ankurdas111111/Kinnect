<script>
  import Router from 'svelte-spa-router';
  import { wrap } from 'svelte-spa-router/wrap';
  import { onMount } from 'svelte';
  import { loadSession } from './lib/stores/auth.js';
  import Login from './pages/Login.svelte';
  import Register from './pages/Register.svelte';
  import MainApp from './pages/MainApp.svelte';
  import Monitoring from './pages/Monitoring.svelte';
  import Landing from './pages/Landing.svelte';
  import Toast from './components/primitives/Toast.svelte';

  // Heavy pages are lazy-loaded so the initial bundle only contains
  // what is needed for Login, Register, and the main app shell.
  const routes = {
    '/': MainApp,
    '/landing': Landing,
    '/dashboard': wrap({ asyncComponent: () => import('./pages/FamilyDashboard.svelte') }),
    '/login': Login,
    '/register': Register,
    '/monitoring': Monitoring,
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
  });

  function conditionsFailed() {
    window.location.hash = '#/login';
  }
</script>

<!-- Skip navigation link — renders off-screen, visible on focus -->
<!-- This satisfies WCAG 2.4.1 (Bypass Blocks) for keyboard and screen reader users -->
<a href="#main-content" class="skip-nav">Skip to main content</a>

<main id="main-content">
  <Router {routes} on:conditionsFailed={conditionsFailed} />
</main>

<Toast />

<style>
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
    background: var(--primary-500, #14b8a6);
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
    box-shadow: 0 4px 16px rgba(20, 184, 166, 0.45);
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
