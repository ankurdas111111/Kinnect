<script>
  /**
   * WeeklyRhythm — YOUR OWN last-7-days movement rhythm. Self-only, never
   * surveils anyone else — a gentle, private reason to open the Hub daily.
   *
   * DB load: the ONE sanctioned low-DB read on the Hub. It fires
   * emitGetDailyActivity() at most once per session (guarded by the store cache,
   * which already holds the result for the rest of the session) — never on
   * re-render, scroll, or WS event.
   */
  import { onMount } from 'svelte';
  import { authUser } from '../../lib/stores/auth.js';
  import { dailyActivity } from '../../lib/stores/activity.js';
  import { emitGetDailyActivity } from '../../lib/socket.js';
  import { allowMotion } from '../../lib/stores/effects.js';

  let myUserId = $derived($authUser?.userId);
  let hasData = $derived(myUserId ? $dailyActivity.has(myUserId) : false);
  let loading = $state(false);
  let loadTimer;

  onMount(() => {
    if (myUserId && !$dailyActivity.has(myUserId)) {
      emitGetDailyActivity(myUserId);        // one read; result cached in the store
      loading = true;
      loadTimer = setTimeout(() => { loading = false; }, 2500);
    }
    return () => clearTimeout(loadTimer);
  });

  // Response arrived — drop the skeleton immediately.
  $effect(() => {
    if (loading && hasData) { loading = false; clearTimeout(loadTimer); }
  });

  // Chronological left→right (store gives most-recent-first, ≤7).
  let days = $derived(myUserId ? [...($dailyActivity.get(myUserId) || [])].reverse() : []);
  let maxMin = $derived(Math.max(1, ...days.map((d) => d.activeMinutes || 0)));

  function dayInitial(dateStr) {
    if (!dateStr) return '';
    const d = new Date(dateStr + 'T00:00:00');
    return d.toLocaleDateString([], { weekday: 'narrow' });
  }
  function isToday(dateStr) {
    if (!dateStr) return false;
    const d = new Date(dateStr + 'T00:00:00'), n = new Date();
    return d.getFullYear() === n.getFullYear() && d.getMonth() === n.getMonth() && d.getDate() === n.getDate();
  }

  let caption = $derived(buildCaption(days));
  function buildCaption(list) {
    if (list.length === 0) return '';
    const today = list[list.length - 1];
    const rest = list.slice(0, -1).filter((d) => (d.activeMinutes || 0) > 0);
    const km = ((today.distanceM || 0) / 1000).toFixed(1);
    if (rest.length === 0) return `${today.activeMinutes || 0} active min today`;
    const avg = rest.reduce((s, d) => s + (d.activeMinutes || 0), 0) / rest.length;
    if ((today.activeMinutes || 0) > avg * 1.3) return `More active than usual — ${km} km so far`;
    if ((today.activeMinutes || 0) < avg * 0.5) return `Quiet day so far — ${km} km`;
    return `A steady week — ${km} km today`;
  }
</script>

<section class="week" aria-label="Your week">
  <div class="week-head">
    <h3 class="week-title">Your week</h3>
    <span class="week-sub">Private to you</span>
  </div>

  {#if loading && !hasData}
    <div class="week-bars" role="status" aria-label="Loading your week" aria-busy="true">
      {#each Array(7) as _, i (i)}<span class="bar bar-skel" style="height:{30 + (i % 4) * 14}%"></span>{/each}
    </div>
  {:else if days.length === 0}
    <p class="week-empty">Move around with tracking on and your week fills in here.</p>
  {:else}
    <div class="week-bars">
      {#each days as d (d.date)}
        {@const h = Math.round(((d.activeMinutes || 0) / maxMin) * 100)}
        <span class="bar-cell" title="{d.activeMinutes || 0} active min · {((d.distanceM || 0) / 1000).toFixed(1)} km">
          <span class="bar-track">
            <span class="bar bar-fill" class:today={isToday(d.date)} class:animate={$allowMotion} style="height:{Math.max(6, h)}%"></span>
          </span>
          <span class="bar-lbl" class:today-lbl={isToday(d.date)}>{dayInitial(d.date)}</span>
        </span>
      {/each}
    </div>
    <p class="week-caption">{caption}</p>
  {/if}
</section>

<style>
  .week { display: flex; flex-direction: column; gap: var(--space-2); }
  .week-head { display: flex; align-items: baseline; justify-content: space-between; }
  .week-title { margin: 0; font-size: var(--text-2xs, 10px); font-weight: 700; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-tertiary); }
  .week-sub { font-size: 9px; color: var(--text-tertiary); opacity: 0.7; }

  .week-bars { display: flex; align-items: flex-end; gap: var(--space-2); height: 72px; }
  .bar-cell { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 4px; height: 100%; }
  .bar-track { flex: 1; width: 100%; display: flex; align-items: flex-end; justify-content: center; }
  .bar {
    width: 100%; max-width: 22px; border-radius: var(--radius-sm, 5px);
    background: var(--primary-500-30);
    transform-origin: bottom center;
  }
  .bar-fill.animate { animation: grow var(--duration-slow, 500ms) var(--ease-out, cubic-bezier(0.16,1,0.3,1)) both; }
  @keyframes grow { from { transform: scaleY(0); } to { transform: scaleY(1); } }
  .bar.today { background: var(--primary-500); box-shadow: 0 0 8px var(--primary-500-30); }
  .bar-skel { background: var(--surface-inset); animation: shimmer 1.4s ease-in-out infinite; }
  @keyframes shimmer { 0%,100% { opacity: 0.5; } 50% { opacity: 0.85; } }
  .bar-lbl { font-size: 9px; font-weight: 600; color: var(--text-tertiary); }
  .today-lbl { color: var(--primary-300); font-weight: 800; }

  .week-caption { margin: 0; font-size: var(--text-xs, 12px); color: var(--text-secondary); }
  .week-empty { margin: 0; font-size: var(--text-xs, 12px); color: var(--text-tertiary); }

  @media (prefers-reduced-motion: reduce) {
    .bar-fill.animate { animation: none !important; }
    .bar-skel { animation: none !important; }
  }
</style>
