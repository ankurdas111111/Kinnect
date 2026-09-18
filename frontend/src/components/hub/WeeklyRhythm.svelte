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

<!-- Family Circle: the design bans dashboards, so the week is spoken first as
     one quiet sentence; beneath it, seven small day-dots trace the rhythm —
     a murmur, never a KPI chart. Numbers stay reachable per-day (tooltip +
     screen-reader text). -->
<section class="week" aria-label="Your week">
  <div class="week-head">
    <h3 class="week-title">Your week</h3>
    <span class="week-sub">Private to you</span>
  </div>

  {#if loading && !hasData}
    <div class="week-dots" role="status" aria-label="Loading your week" aria-busy="true">
      {#each Array(7) as _, i (i)}
        <span class="day-cell"><span class="dot dot-skel"></span><span class="day-lbl">·</span></span>
      {/each}
    </div>
  {:else if days.length === 0}
    <p class="week-empty">Move around with tracking on and your week fills in here.</p>
  {:else}
    <p class="week-caption">{caption}</p>
    <div class="week-dots">
      {#each days as d (d.date)}
        {@const ratio = (d.activeMinutes || 0) / maxMin}
        <span class="day-cell" title="{d.activeMinutes || 0} active min · {((d.distanceM || 0) / 1000).toFixed(1)} km">
          <span
            class="dot"
            class:dot-quiet={(d.activeMinutes || 0) === 0}
            class:dot-today={isToday(d.date)}
            style="--w: {(0.25 + ratio * 0.75).toFixed(2)}"
            aria-hidden="true"
          ></span>
          <span class="day-lbl" class:today-lbl={isToday(d.date)} aria-hidden="true">{dayInitial(d.date)}</span>
          <span class="sr-only">{d.date}: {d.activeMinutes || 0} active minutes, {((d.distanceM || 0) / 1000).toFixed(1)} kilometres</span>
        </span>
      {/each}
    </div>
  {/if}
</section>

<style>
  .week { display: flex; flex-direction: column; gap: var(--space-3); }
  .week-head { display: flex; align-items: baseline; justify-content: space-between; gap: var(--space-2); }
  /* Quiet label register — matches the page's other section labels. */
  .week-title { margin: 0; font-size: var(--text-sm); font-weight: 500; color: var(--text-secondary); }
  .week-sub { font-size: var(--text-xs); color: var(--text-tertiary); }

  /* The sentence carries the meaning; body register, ink. */
  .week-caption { margin: 0; font-size: var(--text-base); line-height: 1.5; color: var(--text-primary); }
  .week-empty { margin: 0; font-size: var(--text-base); line-height: 1.5; color: var(--text-tertiary); }

  /* Seven small pebble-dots, left→right. Weight is opacity (--w), so the row
     reads as a soft rhythm, not a bar chart. */
  .week-dots { display: flex; align-items: center; gap: var(--space-4); }
  .day-cell { display: flex; flex-direction: column; align-items: center; gap: var(--space-1); }
  .dot {
    width: 10px; height: 10px; border-radius: var(--radius-full);
    background: var(--primary-500);
    opacity: var(--w, 0.25);
  }
  /* A day with no movement is an open circle — shape, not just color. */
  .dot-quiet {
    background: transparent;
    border: 1px solid var(--outline-variant);
    opacity: 1;
  }
  .dot-today {
    background: var(--primary-500); opacity: 1;
    box-shadow: 0 0 0 var(--space-1) color-mix(in oklch, var(--primary-500) 18%, transparent);
  }
  .dot-skel {
    background: var(--surface-inset); opacity: 1;
    animation: shimmer 1.4s ease-in-out infinite;
  }
  @keyframes shimmer { 0%,100% { opacity: 0.5; } 50% { opacity: 0.9; } }
  .day-lbl { font-size: var(--text-2xs); font-weight: 500; color: var(--text-tertiary); }
  .today-lbl { color: var(--primary-700); font-weight: 600; }

  @media (prefers-reduced-motion: reduce) {
    .dot-skel { animation: none !important; }
  }
</style>
