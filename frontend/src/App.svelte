<script>
  import Router from 'svelte-spa-router';
  import { wrap } from 'svelte-spa-router/wrap';
  import { onMount } from 'svelte';
  import { loadSession } from './lib/stores/auth.js';
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

<Router {routes} on:conditionsFailed={conditionsFailed} />

<Toast />
