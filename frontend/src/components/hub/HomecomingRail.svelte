<script>
  /**
   * HomecomingRail — live "who's almost home" strip.
   *
   * Renders one card per member currently converging on a saved place, with a
   * client-side ETA countdown and a progress ribbon that fills as they close in.
   * When the server-side arrival monitor drops a member from arrivalProjections
   * (i.e. they arrived), the card plays a one-shot "landed" settle + soft haptic,
   * then self-dismisses. Collapses to nothing when nobody is en route.
   *
   * DB load: ZERO. Reads only the in-memory arrivalProjections store (WS-fed).
   */
  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { arrivalProjections } from '../../lib/stores/arrivals.js';
  import { focusUser } from '../../lib/stores/map.js';
  import { allowMotion } from '../../lib/stores/effects.js';
  import { getUserColor } from '../../lib/getUserColor.js';
  import { fmtEta } from '../../lib/hubStatus.js';
  import { haptics } from '../../lib/haptics.js';
  import AvatarRing from '../primitives/AvatarRing.svelte';

  const LANDED_TTL = 60_000; // keep a "got home" chip for 60s

  // Non-reactive per-user tracking: initial distance (for progress) + seenAt.
  const tracked = new Map(); // userId → { firstDist, seenAt, etaSeconds }

  let nowTick = $state(Date.now());
  let landed = $state([]); // [{ userId, displayName, placeName, at }]
  let tickInterval;

  // Live projections as a plain array (reactive on store change).
  let projections = $derived(
    [...$arrivalProjections.values()]
      .filter((a) => a && a.userId && a.etaSeconds != null)
      .sort((a, b) => a.etaSeconds - b.etaSeconds)
  );

  // Diff the store against `tracked` to (a) seed progress baselines and
  // (b) detect arrivals (a tracked user that left the projection set).
  $effect(() => {
    const current = new Set();
    for (const a of projections) {
      current.add(a.userId);
      const t = tracked.get(a.userId);
      if (!t) {
        tracked.set(a.userId, {
          firstDist: a.distanceM ?? null,
          seenAt: Date.now(),
          etaSeconds: a.etaSeconds,
        });
      } else {
        // resync countdown + grow the baseline if they started farther out
        t.seenAt = Date.now();
        t.etaSeconds = a.etaSeconds;
        if (a.distanceM != null && (t.firstDist == null || a.distanceM > t.firstDist)) t.firstDist = a.distanceM;
      }
    }
    // Anyone we were tracking who vanished → arrived.
    for (const [userId, t] of tracked) {
      if (!current.has(userId)) {
        markLanded(userId, t);
        tracked.delete(userId);
      }
    }
  });

  function markLanded(userId, t) {
    // Recover a display name from the last projection snapshot if we have it.
    const name = t.displayName || nameFor(userId) || 'Someone';
    landed = [...landed.filter((l) => l.userId !== userId), { userId, displayName: name, placeName: t.placeName, at: Date.now() }];
    if ($allowMotion) haptics.success();
  }

  // Cache display metadata so a landed chip can still name the person/place.
  let nameCache = new Map();
  $effect(() => {
    for (const a of projections) nameCache.set(a.userId, { displayName: a.displayName, placeName: a.placeName });
    // stash onto tracked so markLanded can read it
    for (const a of projections) {
      const t = tracked.get(a.userId);
      if (t) { t.displayName = a.displayName; t.placeName = a.placeName; }
    }
  });
  function nameFor(userId) { return nameCache.get(userId)?.displayName; }

  function etaFor(a) {
    const t = tracked.get(a.userId);
    if (!t) return a.etaSeconds;
    const elapsed = (nowTick - t.seenAt) / 1000;
    return Math.max(0, (t.etaSeconds ?? 0) - elapsed);
  }

  function progressFor(a) {
    const t = tracked.get(a.userId);
    if (!t || !t.firstDist || a.distanceM == null) return 0;
    const p = 1 - a.distanceM / t.firstDist;
    return Math.min(1, Math.max(0, p));
  }

  function openMember(userId) {
    focusUser.set(userId);
    push('/');
  }

  onMount(() => {
    tickInterval = setInterval(() => {
      nowTick = Date.now();
      // prune expired landed chips
      const cutoff = Date.now() - LANDED_TTL;
      if (landed.some((l) => l.at < cutoff)) landed = landed.filter((l) => l.at >= cutoff);
    }, 1000);
  });
  onDestroy(() => clearInterval(tickInterval));

  let show = $derived(projections.length > 0 || landed.length > 0);
</script>

{#if show}
  <section class="rail" aria-label="Coming home">
    <h2 class="rail-title">Coming home</h2>
    <div class="rail-track">
      {#each projections as a (a.userId)}
        {@const color = getUserColor(a.userId)}
        <button class="rail-card" style="--mc:{color}" onclick={() => openMember(a.userId)}
          aria-label="{a.displayName || 'Someone'} heading to {a.placeName || 'a saved place'}, about {fmtEta(etaFor(a))} away — view on map">
          <div class="rail-top">
            <AvatarRing ring="live" size={36}>
              <span class="rail-init">{(a.displayName || '?')[0].toUpperCase()}</span>
            </AvatarRing>
            <div class="rail-meta">
              <span class="rail-name">{a.displayName || 'Someone'}</span>
              <span class="rail-dest">→ {a.placeName || 'saved place'}</span>
            </div>
            <span class="rail-eta">{fmtEta(etaFor(a))}</span>
          </div>
          <div class="rail-bar" aria-hidden="true">
            <span class="rail-fill" class:animate={$allowMotion} style="transform:scaleX({progressFor(a)})"></span>
          </div>
        </button>
      {/each}

      {#each landed as l (l.userId)}
        <div class="rail-card rail-landed" aria-label="{l.displayName} arrived at {l.placeName || 'a saved place'}">
          <div class="rail-top">
            <AvatarRing ring="live" size={36}>
              <span class="rail-init">{(l.displayName || '?')[0].toUpperCase()}</span>
            </AvatarRing>
            <div class="rail-meta">
              <span class="rail-name">{l.displayName}</span>
              <span class="rail-dest rail-arrived">Arrived{l.placeName ? ` · ${l.placeName}` : ''}</span>
            </div>
            <span class="rail-check" aria-hidden="true">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
            </span>
          </div>
          <div class="rail-bar" aria-hidden="true"><span class="rail-fill rail-fill-done"></span></div>
        </div>
      {/each}
    </div>
  </section>
{/if}

<style>
  .rail { padding: 0 var(--space-5); margin-top: var(--space-3); }
  .rail-title {
    font-size: var(--text-2xs, 10px); font-weight: 700; text-transform: uppercase;
    letter-spacing: 0.08em; color: var(--text-tertiary); margin: 0 0 var(--space-2);
  }
  .rail-track {
    display: flex; gap: var(--space-2); overflow-x: auto;
    scroll-snap-type: x proximity; scrollbar-width: none;
    padding-bottom: var(--space-1);
  }
  .rail-track::-webkit-scrollbar { display: none; }

  .rail-card {
    flex: 0 0 auto; width: 232px; min-height: 44px;
    scroll-snap-align: start;
    display: flex; flex-direction: column; gap: var(--space-2);
    padding: var(--space-3);
    background: var(--glass-bg);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-lg, 14px);
    backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px);
    cursor: pointer; text-align: left; color: inherit; font: inherit;
    transition: border-color var(--duration-fast, 150ms) var(--ease-out, ease),
                background var(--duration-fast, 150ms) var(--ease-out, ease);
    -webkit-tap-highlight-color: transparent;
  }
  .rail-card:hover { border-color: color-mix(in srgb, var(--mc) 40%, transparent); background: var(--surface-hover); }
  .rail-card:focus-visible { outline: 2px solid var(--mc, var(--primary-400)); outline-offset: 2px; }
  .rail-landed { cursor: default; border-color: var(--success-500-20); background: var(--success-500-08); }

  .rail-top { display: flex; align-items: center; gap: var(--space-2); }
  .rail-init { font-size: 13px; font-weight: 800; color: var(--mc, var(--primary-400)); line-height: 1; user-select: none; }
  .rail-meta { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1px; }
  .rail-name { font-size: var(--text-sm, 13px); font-weight: 700; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .rail-dest { font-size: 11px; color: var(--text-tertiary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .rail-arrived { color: var(--success-400); font-weight: 600; }
  .rail-eta { font-size: var(--text-sm, 14px); font-weight: 700; font-family: var(--font-mono, monospace); font-variant-numeric: tabular-nums; color: var(--text-secondary); white-space: nowrap; }
  .rail-check { color: var(--success-400); display: flex; }

  .rail-bar { height: 3px; border-radius: var(--radius-full, 999px); background: var(--surface-2); overflow: hidden; }
  .rail-fill {
    display: block; height: 100%; width: 100%; border-radius: 999px;
    transform-origin: left center; transform: scaleX(0);
    background: linear-gradient(90deg, color-mix(in srgb, var(--mc) 60%, transparent), var(--mc));
  }
  .rail-fill.animate { transition: transform 1s var(--ease-out, cubic-bezier(0.16,1,0.3,1)); }
  .rail-fill-done { transform: scaleX(1); background: var(--success-500); }

  @media (prefers-reduced-motion: reduce) {
    .rail-fill { transition: none !important; }
  }
</style>
