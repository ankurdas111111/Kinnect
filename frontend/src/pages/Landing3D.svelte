<script>
  /**
   * Host for the React/R3F landing island.
   *
   * Svelte owns the route, the theme and the auth state; React owns only the
   * 3D page. The bridge is one dynamic import, so nothing about React reaches
   * the app bundle, and the native build aliases it away completely.
   */
  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { landingIsBlocked } from '../lib/entryPoint.js';
  import { supportsWebGL } from '../lib/deviceCapability.js';

  let host = $state();
  let unmount = null;
  let failed = $state(false);

  onMount(() => {
    if (landingIsBlocked()) { push('/login'); return; }
    document.title = 'Kinnect — a quiet map for the people you love';

    // No WebGL means no landing worth rendering: fall through to the real app
    // rather than showing an empty stage.
    if (!supportsWebGL()) { failed = true; return; }

    let cancelled = false;
    import('../landing3d/mount.js')
      .then(({ mountLanding }) => {
        if (cancelled || !host) return;
        return mountLanding(host, { onExit: () => {} });
      })
      .then((fn) => { if (cancelled && fn) fn(); else unmount = fn; })
      .catch(() => { failed = true; });

    return () => { cancelled = true; };
  });

  onDestroy(() => { if (unmount) { unmount(); unmount = null; } });
</script>

<svelte:head>
  <meta name="description" content="Kinnect is family location sharing without the noise: your people as a few pebbles, one honest sentence, and a hold-to-send SOS." />
</svelte:head>

{#if failed}
  <!-- Real content, not a browser-unsupported card: the pitch and both actions
       still work with no WebGL at all. -->
  <main class="l3d-fallback">
    <p class="fb-eyebrow">Family location sharing, without the noise</p>
    <h1 class="fb-h1">A quiet map for the people you love.</h1>
    <p class="fb-sub">
      Your family as a few pebbles, one honest sentence about how everyone’s
      doing, and nothing else. No feeds. No pings. No dashboard.
    </p>
    <div class="fb-row">
      <a class="fb-cta" href="#/register">Create your family — free</a>
      <a class="fb-quiet" href="#/login">Sign in</a>
    </div>
  </main>
{:else}
  <div bind:this={host}></div>
{/if}

<style>
  .l3d-fallback {
    min-height: 100vh;
    display: flex; flex-direction: column; justify-content: center;
    gap: var(--space-5);
    padding: var(--space-10) clamp(var(--space-4), 5vw, var(--space-12));
    max-width: 44rem;
    background: var(--surface-0); color: var(--text-primary);
  }
  .fb-eyebrow {
    margin: 0; font-size: var(--text-xs); font-weight: 600;
    letter-spacing: 0.12em; text-transform: uppercase; color: var(--text-tertiary);
  }
  .fb-h1 {
    margin: 0; font-family: var(--font-serif); font-style: italic; font-weight: 400;
    font-size: clamp(2.4rem, 7vw, 4.4rem); line-height: 1.04; letter-spacing: -0.025em;
  }
  .fb-sub { margin: 0; max-width: 34ch; font-size: var(--text-lg); line-height: 1.6; color: var(--text-secondary); }
  .fb-row { display: flex; flex-wrap: wrap; gap: var(--space-4); align-items: center; }
  .fb-cta {
    min-height: 52px; display: inline-flex; align-items: center; padding: 0 var(--space-6);
    border-radius: var(--radius-full, 999px); background: var(--primary-500);
    color: var(--text-on-primary); text-decoration: none; font-weight: 700;
  }
  .fb-quiet {
    min-height: 44px; display: inline-flex; align-items: center;
    color: var(--text-primary); text-decoration: none; font-weight: 600;
    border-bottom: 1px solid var(--border-strong);
  }
</style>
