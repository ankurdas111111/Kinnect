<script>
  /**
   * HomecomingRail — live "who's almost home", spoken as quiet paper rows
   * (Stitch "Family Circle"): a warm pebble plus one plain-language sentence
   * ("Elena is heading to Home — about 4 min away."), with a hairline
   * reassurance ribbon that fills as they close in. No cards, no borders.
   *
   * When the server-side arrival monitor drops a member from
   * arrivalProjections (i.e. they arrived), the row settles into a sage
   * "arrived" sentence + soft haptic, then self-dismisses after 60s.
   * Collapses to nothing when nobody is en route.
   *
   * The section heading lives in the parent (FamilyDashboard) — this renders
   * only the rows, so the title never appears twice.
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

  const LANDED_TTL = 60_000; // keep a "got home" row for 60s

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

  // Cache display metadata so a landed row can still name the person/place.
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
      // prune expired landed rows
      const cutoff = Date.now() - LANDED_TTL;
      if (landed.some((l) => l.at < cutoff)) landed = landed.filter((l) => l.at >= cutoff);
    }, 1000);
  });
  onDestroy(() => clearInterval(tickInterval));

  let show = $derived(projections.length > 0 || landed.length > 0);
</script>

{#if show}
  <!-- The "Coming home" heading + landmark live in FamilyDashboard's section. -->
  <div class="hc">
    {#each projections as a (a.userId)}
      {@const color = getUserColor(a.userId)}
      <button class="hc-row" style="--mc:{color}" onclick={() => openMember(a.userId)}
        aria-label="{a.displayName || 'Someone'} heading to {a.placeName || 'a saved place'}, about {fmtEta(etaFor(a))} away — view on map">
        <span class="hc-pebble">
          {#if $allowMotion}<span class="hc-halo" aria-hidden="true"></span>{/if}
          <span class="hc-init">{(a.displayName || '?')[0].toUpperCase()}</span>
        </span>
        <span class="hc-body">
          <span class="hc-sentence">
            {a.displayName || 'Someone'} is heading to {a.placeName || 'a saved place'} —
            about <span class="hc-eta">{fmtEta(etaFor(a))}</span> away.
          </span>
          <span class="hc-bar" aria-hidden="true">
            <span class="hc-fill" class:animate={$allowMotion} style="transform:scaleX({progressFor(a)})"></span>
          </span>
        </span>
      </button>
    {/each}

    {#each landed as l (l.userId)}
      <div class="hc-row hc-landed" aria-label="{l.displayName} arrived at {l.placeName || 'a saved place'}">
        <span class="hc-pebble hc-pebble-landed">
          <span class="hc-init">{(l.displayName || '?')[0].toUpperCase()}</span>
        </span>
        <span class="hc-body">
          <span class="hc-sentence hc-sentence-landed">
            {l.displayName} arrived{l.placeName ? ` at ${l.placeName}` : ''} safely.
            <span class="hc-check" aria-hidden="true">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
            </span>
          </span>
          <span class="hc-bar" aria-hidden="true"><span class="hc-fill hc-fill-done"></span></span>
        </span>
      </div>
    {/each}
  </div>
{/if}

<style>
  /* Quiet paper rows separated by whitespace — never bordered cards. */
  .hc { display: flex; flex-direction: column; gap: var(--space-4); }

  .hc-row {
    display: flex; align-items: flex-start; gap: var(--space-4);
    min-height: 44px;
    padding: var(--space-2);
    margin: calc(-1 * var(--space-2));
    background: transparent; border: none;
    border-radius: var(--radius-md);
    cursor: pointer; text-align: left; color: inherit; font: inherit;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  button.hc-row:hover { background: var(--surface-hover); }
  button.hc-row:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }
  .hc-landed { cursor: default; }

  /* Warm pebble with the live breathing halo (transform/opacity only). */
  .hc-pebble {
    position: relative; flex-shrink: 0; margin-top: 2px;
    width: 40px; height: 40px;
    display: flex; align-items: center; justify-content: center;
    border-radius: var(--radius-full);
    background: var(--primary-100);
  }
  .hc-pebble-landed { background: color-mix(in oklch, var(--success-500) 12%, transparent); }
  .hc-init {
    position: relative;
    font-size: var(--text-sm); font-weight: 600; line-height: 1;
    color: var(--mc, var(--primary-700)); user-select: none;
  }
  .hc-pebble-landed .hc-init { color: var(--success-600); }
  .hc-halo {
    position: absolute; inset: 0; border-radius: inherit;
    box-shadow: 0 0 0 var(--space-1) color-mix(in oklch, var(--primary-500) 28%, transparent);
    opacity: 0; transform: scale(0.94);
    animation: hc-breathe 3.6s ease-in-out infinite;
    pointer-events: none;
  }
  @keyframes hc-breathe {
    0%, 100% { opacity: 0; transform: scale(0.94); }
    50%      { opacity: 1; transform: scale(1.06); }
  }

  .hc-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: var(--space-2); }
  /* One plain-language sentence — body register, soft numbers inline. */
  .hc-sentence { font-size: var(--text-base); line-height: 1.5; color: var(--text-primary); }
  .hc-eta { font-variant-numeric: tabular-nums; color: var(--primary-700); font-weight: 500; }
  .hc-sentence-landed { color: var(--success-600); }
  .hc-check { display: inline-flex; vertical-align: middle; color: var(--success-500); margin-left: var(--space-1); }

  /* Hairline reassurance ribbon — quiet, subordinate, GPU-only fill. */
  .hc-bar {
    height: 3px; border-radius: var(--radius-full);
    background: var(--surface-inset); overflow: hidden;
  }
  .hc-fill {
    display: block; height: 100%; width: 100%; border-radius: inherit;
    transform-origin: left center; transform: scaleX(0);
    background: var(--primary-500);
  }
  .hc-fill.animate { transition: transform 1s var(--ease-out); }
  .hc-fill-done { transform: scaleX(1); background: var(--success-500); }

  @media (prefers-reduced-motion: reduce) {
    .hc-fill { transition: none !important; }
    .hc-halo { animation: none !important; opacity: 0 !important; }
  }
</style>
