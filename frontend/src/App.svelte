<script>
  import Router from 'svelte-spa-router';
  import { wrap } from 'svelte-spa-router/wrap';
  import { onMount } from 'svelte';
  import { authUser, loadSession } from './lib/stores/auth.js';
  import Login from './pages/Login.svelte';
  import Register from './pages/Register.svelte';
  import MainApp from './pages/MainApp.svelte';
  import Monitoring from './pages/Monitoring.svelte';
  import FamilyDashboard from './pages/FamilyDashboard.svelte';
  import EmergencyProfile from './pages/EmergencyProfile.svelte';
  import Toast from './components/primitives/Toast.svelte';

  const routes = {
    '/': MainApp,
    '/dashboard': FamilyDashboard,
    '/login': Login,
    '/register': Register,
    '/monitoring': Monitoring,
    '/emergency': EmergencyProfile,
    '/activity': wrap({ asyncComponent: () => import('./pages/ActivityFeed.svelte') }),
    '/replay': wrap({ asyncComponent: () => import('./pages/RoutePlayback.svelte') }),
    '/checkins': wrap({ asyncComponent: () => import('./pages/CheckinSchedule.svelte') }),
    '/add-contact/:code': wrap({ asyncComponent: () => import('./pages/AddContact.svelte') }),
    '/live/:token': wrap({ asyncComponent: () => import('./pages/LiveViewer.svelte') }),
    '/watch/:token': wrap({ asyncComponent: () => import('./pages/WatchViewer.svelte') }),
    '/m/:token': wrap({ asyncComponent: () => import('./pages/SecretChatViewer.svelte') })
  };

  let loading = true;

  onMount(async () => {
    await loadSession();
    loading = false;
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.register('/sw.js').catch(() => {});
    }
  });

  function conditionsFailed(event) {
    window.location.hash = '#/login';
  }
</script>

{#if loading}
  <div class="app-loading" role="status" aria-live="polite" aria-busy="true">
    <div class="app-loading-spinner"></div>
    <p>Loading Kinnect...</p>
  </div>
{:else}
  <Router {routes} on:conditionsFailed={conditionsFailed} />
{/if}

<Toast />

<style>
  .app-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;
    gap: var(--space-4);
    color: var(--text-secondary);
    font-family: var(--font-sans);
  }
  .app-loading-spinner {
    width: 36px;
    height: 36px;
    border: 3px solid var(--border-default);
    border-top-color: var(--primary-500);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    /* 3D depth spinner */
    box-shadow:
      0 4px 16px rgba(99, 102, 241, 0.20),
      inset 0 1px 2px rgba(255, 255, 255, 0.10);
    transform-style: preserve-3d;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
