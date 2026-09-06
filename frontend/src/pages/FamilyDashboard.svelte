<script>
  import { run } from 'svelte/legacy';

  import { onMount } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { authUser } from '../lib/stores/auth.js';
  import { myLocation } from '../lib/stores/map.js';
  import { mySosActive } from '../lib/stores/sos.js';
  import { clearHubBadge } from '../lib/stores/hubBadge.js';
  import { familyVerdict, familyMembers, verdictNow } from '../lib/stores/verdict.js';
  import { arrivalProjections } from '../lib/stores/arrivals.js';
  import { formatAge } from '../lib/presence.js';
  import { activityEvents } from '../lib/activityLog.js';
  import { rhythmEnabled, setRhythmEnabled } from '../lib/presenceRhythm.js';
  import { myRooms } from '../lib/stores/rooms.js';
  import HomecomingRail from '../components/hub/HomecomingRail.svelte';
  import FamilyRoster from '../components/hub/FamilyRoster.svelte';
  import PulseButton from '../components/hub/PulseButton.svelte';
  import InviteStrip from '../components/hub/InviteStrip.svelte';
  import WeeklyRhythm from '../components/hub/WeeklyRhythm.svelte';

  run(() => {
    if (!$authUser) push('/login');
  });

  // ── Shared "now" heartbeat — verdict store owns the 10s tick ───────────────
  let nowMs = $derived($verdictNow);

  // First-run "new feature" dots — client-only, localStorage (zero DB).
  const VIS_KEYS = {
    activity: 'kinnect_vis_activity', replay: 'kinnect_vis_replay',
    emergency: 'kinnect_vis_emergency', checkins: 'kinnect_vis_checkins',
  };
  let visited = $state({ activity: true, replay: true, emergency: true, checkins: true });
  function visitFeature(key, route) {
    if (key) { localStorage.setItem(VIS_KEYS[key], '1'); visited = { ...visited, [key]: true }; }
    push(route);
  }

  onMount(() => {
    clearHubBadge();
    visited = Object.fromEntries(Object.entries(VIS_KEYS).map(([k, v]) => [k, !!localStorage.getItem(v)]));
    requestAnimationFrame(() => { mounted = true; });
  });

  let mounted = $state(false);
  let nowDate = $derived(new Date(nowMs));
  let timeStr = $derived(nowDate.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' }));
  let dateStr = $derived(nowDate.toLocaleDateString([], { weekday: 'long', day: 'numeric', month: 'long' }));

  // ── Derived family state — read from the shared verdict store (zero DB) ────
  let members = $derived($familyMembers);
  let verdict = $derived($familyVerdict);

  // Hearth 06: rooms are this app's families, so the room name titles the page.
  let familyName = $derived($myRooms?.[0]?.name || 'Your family');
  let initials = $derived(
    ($authUser?.displayName || '?')
      .split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]).join('').toUpperCase()
  );

  /**
   * Hearth 06b — when nobody has shared in a long time the page must SAY so and
   * offer one useful thing to do, instead of rendering empty widgets. Derived
   * from real state only; no invented numbers.
   */
  let sharingNow = $derived(members.filter((m) => m.online && m.lastUpdate));
  let dormant = $derived(members.filter((m) => !m.online || !m.lastUpdate));
  let nobodySharing = $derived(members.length > 0 && sharingNow.length === 0);
</script>

<div class="d d-{verdict.tone}" class:d-ready={mounted}>
  <!-- ONE ambient tint, driven by the real verdict tone (replaces aurora+noise+glow pile-up) -->
  <div class="d-ambient" aria-hidden="true"></div>

  <!-- Header (Hearth 06): back to map · wordmark · family · date + you -->
  <header class="d-header">
    <button class="d-back tactile" onclick={() => push('/')} aria-label="Back to map">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="15 18 9 12 15 6"/></svg>
      Map
    </button>
    <span class="d-wordmark">Kinnect</span>
    <span class="d-family-name">{familyName}</span>
    <div class="d-head-right">
      <span class="d-when">{dateStr} · {timeStr}</span>
      <span class="d-avatar" aria-hidden="true">{initials}</span>
    </div>
  </header>

  <!-- Content — two columns on desktop, one on phone (06a / 06c) -->
  <div class="d-content">
    <div class="d-main">
      <section class="d-now" aria-labelledby="d-now-head">
        <p class="d-eyebrow" id="d-now-head">Right now</p>
        <h1 class="d-verdict verdict-voice">{verdict.sentence}</h1>
        {#if nobodySharing}
          <p class="d-verdict-detail">
            Nobody in {familyName} is sharing at the moment. One nudge asks everyone to turn
            sharing back on — or invite someone new.
          </p>
        {:else if verdict.detail}
          <p class="d-verdict-detail">{verdict.detail}</p>
        {/if}

        <!-- One row of real actions, ember reserved for the primary one -->
        <div class="d-cta-row">
          <div class="d-cta-pulse"><PulseButton /></div>
          <button class="d-cta" onclick={() => push('/')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6"/><line x1="12" y1="2" x2="12" y2="15"/></svg>
            Share live
          </button>
          <button class="d-cta d-cta-accent" onclick={() => push('/')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            Invite
          </button>
        </div>
      </section>

      <section class="d-family card">
        <FamilyRoster {members} myLocation={$myLocation} now={nowMs} />
        {#if members.length > 0}
          <button class="d-rhythm-toggle" role="switch" aria-checked={$rhythmEnabled}
            onclick={() => setRhythmEnabled(!$rhythmEnabled)}>
            <span class="d-rhythm-dot" class:on={$rhythmEnabled}></span>
            {$rhythmEnabled ? 'Rhythm hints on' : 'Show rhythm hints'}
          </button>
        {/if}
        <div class="d-invite-slot"><InviteStrip {members} /></div>
      </section>

      <!-- Slim text nav, not a wall of tiles (06a footer) -->
      <nav class="d-nav" aria-label="More">
        <button class="d-nav-link" onclick={() => visitFeature('activity', '/activity')}>Activity{#if !visited.activity}<span class="d-dot"></span>{/if}</button>
        <button class="d-nav-link" onclick={() => visitFeature('replay', '/replay')}>Routes{#if !visited.replay}<span class="d-dot"></span>{/if}</button>
        <button class="d-nav-link" onclick={() => visitFeature('checkins', '/checkins')}>Check-ins{#if !visited.checkins}<span class="d-dot"></span>{/if}</button>
        <button class="d-nav-link" onclick={() => push('/')}>Places</button>
        <button class="d-nav-link" class:d-nav-sos={$mySosActive} onclick={() => visitFeature('emergency', '/emergency')}>Emergency{#if !visited.emergency}<span class="d-dot d-dot-red"></span>{/if}</button>
      </nav>
    </div>

    <!-- Right column: where everyone is, coming home, today, your week -->
    <aside class="d-side" aria-label="Family overview">
      <section class="card d-card">
        <header class="d-card-head">
          <h2 class="d-card-title">Where everyone is</h2>
          <button class="d-card-more" onclick={() => push('/')}>Open map →</button>
        </header>
        {#if members.length === 0}
          <p class="d-empty">Nobody to show yet. Invite someone and they'll appear here.</p>
        {:else}
          <div class="d-pebbles">
            {#each members.slice(0, 8) as m (m.userId || m.socketId)}
              <span class="d-pebble" style="--hue: var(--member-{(Math.abs([...(m.displayName || '?')].reduce((a, c) => a + c.charCodeAt(0), 0)) % 4) + 1})">
                <span class="d-pebble-dot" aria-hidden="true"></span>
                {(m.displayName || '?').split(' ')[0]}
              </span>
            {/each}
          </div>
        {/if}
      </section>

      <section class="card d-card">
        <h2 class="d-card-title">Coming home</h2>
        <!-- HomecomingRail self-hides when there is nothing to project, which
             left an empty card. 06b: say what would fill it. -->
        {#if $arrivalProjections && $arrivalProjections.size > 0}
          <HomecomingRail />
        {:else}
          <p class="d-empty-strong">Nobody's heading home right now.</p>
          <p class="d-empty">This fills in once someone sets a Home place and starts sharing.</p>
        {/if}
      </section>

      <section class="card d-card">
        <header class="d-card-head">
          <h2 class="d-card-title">Today</h2>
          {#if $activityEvents.length > 0}
            <button class="d-card-more" onclick={() => push('/activity')}>All activity →</button>
          {/if}
        </header>
        {#if $activityEvents.length === 0}
          <!-- 06b: an empty day is a sentence, not a blank panel -->
          <p class="d-empty-strong">Nothing happened today.</p>
          <p class="d-empty">A quiet day is a good day. Nothing to review.</p>
        {:else}
          <div class="d-today">
            {#each $activityEvents.slice(0, 3) as ev (ev.id)}
              <button class="d-today-row" onclick={() => push('/activity')}>
                <span class="d-today-ts">{formatAge(nowMs - ev.ts)}</span>
                <span class="d-recent-dot d-recent-{ev.type}" aria-hidden="true"></span>
                <span class="d-today-msg">{ev.message}</span>
              </button>
            {/each}
          </div>
        {/if}
      </section>

      <!-- WeeklyRhythm renders its own "Your week / Private to you" header —
           no wrapper header here, or the card shows the title twice. -->
      <section class="card d-card">
        <WeeklyRhythm />
      </section>
    </aside>

    <div class="d-spacer"></div>
  </div>
</div>

<style>
  .d {
    position: relative;
    height: 100dvh; overflow: hidden;
    background: var(--surface-0, #050812);
    color: var(--text-primary, #fff);
    font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
    opacity: 0; transition: opacity var(--duration-slow, 500ms) var(--ease-out, ease);
  }
  .d.d-ready { opacity: 1; }

  /* ── Single ambient tint (opacity-only crossfade between tones) ───────────── */
  .d-ambient {
    position: fixed; inset: 0; pointer-events: none; z-index: 0;
    background:
      radial-gradient(ellipse 70% 55% at 20% 12%, var(--amb-a) 0%, transparent 62%),
      radial-gradient(ellipse 60% 45% at 85% 80%, var(--amb-b) 0%, transparent 60%);
    transition: background var(--duration-slow, 600ms) var(--ease-out, ease);
  }
  /* Hearth: the page is warm paper, so the tone wash is a whisper of the tone
     colour — the old emerald/indigo pair read as a green stain on cream. */
  .d-safe    { --amb-a: color-mix(in oklch, var(--sage) 5%, transparent);      --amb-b: transparent; }
  .d-caution { --amb-a: color-mix(in oklch, var(--ochre) 7%, transparent);     --amb-b: transparent; }
  .d-alert   { --amb-a: color-mix(in oklch, var(--vermilion) 9%, transparent); --amb-b: transparent; }

  /* ── Header ───────────────────────────────────────────────────────────────── */
  .d-header {
    position: fixed; top: 0; left: 0; right: 0; z-index: 20;
    display: flex; align-items: center; gap: var(--space-3);
    padding: calc(var(--safe-top, 0px) + var(--space-2)) var(--space-5) var(--space-2);
    background: color-mix(in oklch, var(--surface-0) 88%, transparent);
    backdrop-filter: blur(24px) saturate(1.5); -webkit-backdrop-filter: blur(24px) saturate(1.5);
    border-bottom: 1px solid var(--border-subtle);
  }
  .d-wordmark {
    font-family: var(--font-serif, var(--font-display));
    font-style: italic;
    font-size: var(--text-lg);
    color: var(--text-primary);
  }
  .d-family-name { font-size: var(--text-sm); font-weight: 600; color: var(--text-secondary); }
  .d-head-right { margin-left: auto; display: flex; align-items: center; gap: var(--space-3); }
  .d-when {
    font-size: var(--text-xs); color: var(--text-tertiary);
    font-variant-numeric: tabular-nums; white-space: nowrap;
  }
  .d-avatar {
    width: 30px; height: 30px; border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    background: var(--primary-500); color: var(--text-on-primary);
    font-size: 11px; font-weight: 700; letter-spacing: 0.02em; flex-shrink: 0;
  }
  @media (max-width: 640px) {
    .d-wordmark, .d-when { display: none; }
  }
  .d-back {
    display: flex; align-items: center; gap: 4px; min-height: 44px;
    padding: 0 var(--space-3) 0 var(--space-2);
    background: var(--surface-inset, rgba(255,255,255,0.06)); border: 1px solid var(--border-default, rgba(255,255,255,0.08));
    border-radius: var(--radius-full, 22px);
    color: var(--text-secondary); font-size: var(--text-xs); font-weight: 600; cursor: pointer;
    transition: background var(--duration-fast, 150ms) var(--ease-out), color var(--duration-fast, 150ms) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .d-back:hover { background: var(--surface-hover); color: var(--text-primary); }
  .d-back:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }

  /* ── Content: two columns on desktop (06a), one on phone (06c) ───────────── */
  .d-content {
    position: relative; z-index: 5;
    height: 100dvh; overflow-y: auto;
    -webkit-overflow-scrolling: touch; overscroll-behavior-y: contain;
    padding: calc(var(--safe-top, 0px) + 60px) var(--space-5) var(--space-8);
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--space-5);
    align-content: start;
  }
  @media (min-width: 1024px) {
    .d-content {
      grid-template-columns: minmax(0, 1.35fr) minmax(320px, 0.65fr);
      gap: var(--space-6);
      max-width: 1360px;
      margin-inline: auto;
      width: 100%;
    }
  }
  .d-main, .d-side {
    display: flex; flex-direction: column; gap: var(--space-4);
    min-width: 0;
  }

  /* Staggered entrance — transform/opacity only */
  .d-main, .d-side {
    opacity: 0; transform: translateY(10px);
    transition: opacity var(--duration-slow, 500ms) var(--ease-out), transform var(--duration-slow, 500ms) var(--ease-out);
  }
  .d-ready .d-main { opacity: 1; transform: none; transition-delay: 40ms; }
  .d-ready .d-side { opacity: 1; transform: none; transition-delay: 140ms; }

  /* ── "Right now" — the page answers before it shows anything ─────────────── */
  .d-eyebrow {
    margin: 0 0 var(--space-2);
    font-size: var(--text-xs); font-weight: 600;
    letter-spacing: 0.08em; text-transform: uppercase;
    color: var(--text-tertiary);
  }
  .d-verdict {
    margin: 0;
    font-size: clamp(2rem, 4.4vw, 3.25rem);
    line-height: 1.05;
    color: var(--text-primary);
  }
  .d-verdict-detail {
    margin: var(--space-3) 0 0;
    font-size: var(--text-base);
    line-height: 1.55;
    color: var(--text-secondary);
    max-width: 54ch;
  }
  .d-cta-row {
    display: flex; align-items: center; gap: var(--space-2);
    flex-wrap: wrap; margin-top: var(--space-4);
  }
  .d-cta {
    display: inline-flex; align-items: center; gap: var(--space-2);
    min-height: 44px; padding: 0 var(--space-4);
    background: var(--surface-1); color: var(--text-primary);
    border: 1px solid var(--border-default); border-radius: var(--radius-full, 999px);
    font-size: var(--text-sm); font-weight: 600; font-family: inherit; cursor: pointer;
    transition: background var(--duration-fast, 150ms) var(--ease-out), border-color var(--duration-fast, 150ms) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .d-cta:hover { background: var(--surface-hover); }
  .d-cta:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  /* Ember stays reserved for the one action worth taking */
  .d-cta-accent {
    background: color-mix(in oklch, var(--primary-500) 12%, transparent);
    border-color: color-mix(in oklch, var(--primary-500) 30%, transparent);
    color: var(--primary-600);
  }
  .d-cta-accent:hover { background: color-mix(in oklch, var(--primary-500) 18%, transparent); }
  .d-cta-pulse :global(button) { min-height: 44px; }

  /* ── Cards ───────────────────────────────────────────────────────────────── */
  .card {
    background: var(--surface-1);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-lg, 16px);
    padding: var(--space-4);
    box-shadow: var(--shadow-sm);
  }
  .d-card { display: flex; flex-direction: column; gap: var(--space-3); }

  /* The roster's ghost-constellation empty state is sized for a whole panel;
     inside the dashboard card it swallowed the left column. Scale the stage
     down here only — the panel version is untouched. */
  .d-family :global(.gc-stage) { transform: scale(0.68); transform-origin: center; margin: calc(-1 * var(--space-6)) 0; }
  @media (prefers-reduced-motion: reduce) {
    .d-family :global(.gc-stage) { transform: scale(0.68); }
  }
  .d-card-head { display: flex; align-items: baseline; justify-content: space-between; gap: var(--space-3); }
  .d-card-title {
    margin: 0; font-family: var(--font-display);
    font-size: var(--text-sm); font-weight: 700;
    letter-spacing: 0.02em; color: var(--text-primary);
  }
  .d-card-more {
    background: none; border: none; padding: 0; cursor: pointer;
    font-family: inherit; font-size: var(--text-xs); font-weight: 600;
    color: var(--primary-600);
  }
  .d-card-more:hover { text-decoration: underline; }
  .d-card-more:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .d-card-note { font-size: var(--text-xs); color: var(--text-tertiary); }

  /* Honest empty states (06b) — a sentence, never a blank panel */
  .d-empty-strong { margin: 0; font-size: var(--text-sm); font-weight: 600; color: var(--text-primary); }
  .d-empty { margin: 0; font-size: var(--text-sm); line-height: 1.5; color: var(--text-tertiary); }

  /* Pebbles — name tags, not coordinates */
  .d-pebbles { display: flex; flex-wrap: wrap; gap: var(--space-2); }
  .d-pebble {
    display: inline-flex; align-items: center; gap: 6px;
    padding: 5px var(--space-3);
    background: var(--surface-3); border: 1px solid var(--border-subtle);
    border-radius: var(--radius-full, 999px);
    font-size: var(--text-xs); font-weight: 600; color: var(--text-secondary);
  }
  .d-pebble-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--hue, var(--primary-500)); }

  /* Today timeline */
  .d-today { display: flex; flex-direction: column; }
  .d-today-row {
    display: grid; grid-template-columns: auto auto 1fr; align-items: center;
    gap: var(--space-2); padding: var(--space-2) 0;
    background: none; border: none; text-align: left; cursor: pointer;
    font-family: inherit; color: inherit;
    border-bottom: 1px solid var(--border-subtle);
  }
  .d-today-row:last-child { border-bottom: none; }
  .d-today-row:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .d-today-ts {
    font-size: var(--text-xs); color: var(--text-tertiary);
    font-variant-numeric: tabular-nums; min-width: 4.5ch;
  }
  .d-today-msg { font-size: var(--text-sm); color: var(--text-secondary); }

  /* Slim text nav (06a footer) */
  .d-nav { display: flex; flex-wrap: wrap; gap: var(--space-4); padding-top: var(--space-2); }
  .d-nav-link {
    position: relative;
    background: none; border: none; padding: var(--space-2) 0; cursor: pointer;
    font-family: inherit; font-size: var(--text-sm); font-weight: 600;
    color: var(--text-secondary); min-height: 44px;
  }
  .d-nav-link:hover { color: var(--text-primary); }
  .d-nav-link:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; border-radius: var(--radius-sm, 6px); }
  .d-nav-sos { color: var(--danger-500); }

  /* ── Quick actions ────────────────────────────────────────────────────────── */
  .d-actions { display: grid; grid-template-columns: repeat(5, 1fr); gap: var(--space-2); }
  .d-act {
    position: relative; min-height: 64px;
    display: flex; flex-direction: column; align-items: center; justify-content: center; gap: var(--space-1);
    padding: var(--space-2) var(--space-1);
    background: var(--glass-bg, rgba(255,255,255,0.03)); border: 1px solid var(--border-default, rgba(255,255,255,0.06));
    border-radius: var(--radius-md, 12px);
    color: var(--text-secondary); font-size: 10px; font-weight: 600; cursor: pointer;
    transition: border-color var(--duration-fast, 150ms) var(--ease-out), background var(--duration-fast, 150ms) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .d-act:hover { border-color: rgba(255,255,255,0.14); background: rgba(255,255,255,0.05); }
  .d-act:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .act-map { color: var(--primary-400); }
  .act-activity { color: var(--success-300, #34d399); }
  .act-replay { color: var(--warning-300, #fbbf24); }
  .act-sos { color: var(--danger-400); }
  .act-checkin { color: var(--info-300, #22d3ee); }
  .act-sos-on { border-color: var(--danger-500-20, rgba(239,68,68,0.25)); animation: sos-b 2s ease-in-out infinite; }
  @keyframes sos-b { 0%,100% { border-color: rgba(239,68,68,0.15); } 50% { border-color: rgba(239,68,68,0.45); } }
  .d-dot { position: absolute; top: 6px; right: 6px; width: 6px; height: 6px; border-radius: 50%; background: var(--warning-400); }
  .d-dot-red { background: var(--danger-500); }
  .d-dot-cyan { background: var(--info-400, #22d3ee); }

  /* ── Rhythm hints opt-in toggle (subtle, non-alarmist) ────────────────────── */
  .d-rhythm-toggle {
    display: inline-flex; align-items: center; gap: var(--space-2);
    align-self: flex-start; margin-top: var(--space-3); min-height: 32px;
    padding: var(--space-1) var(--space-2);
    background: none; border: none; cursor: pointer;
    font-size: 11px; font-weight: 600; color: var(--text-tertiary);
    -webkit-tap-highlight-color: transparent;
  }
  .d-rhythm-toggle:hover { color: var(--text-secondary); }
  .d-rhythm-toggle:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; border-radius: var(--radius-sm, 6px); }
  .d-rhythm-dot {
    width: 8px; height: 8px; border-radius: 50%;
    background: var(--border-strong, rgba(255,255,255,0.22));
    transition: background var(--duration-fast, 150ms) var(--ease-out), box-shadow var(--duration-fast, 150ms) var(--ease-out);
  }
  .d-rhythm-dot.on { background: var(--primary-400); box-shadow: 0 0 6px var(--primary-500-30, rgba(99,102,241,0.5)); }
  .d-family { display: flex; flex-direction: column; }

  /* ── Recent activity peek ─────────────────────────────────────────────────── */
  .d-recent { display: flex; flex-direction: column; gap: var(--space-2); }
  .d-recent-head {
    display: flex; align-items: center; justify-content: space-between;
    background: none; border: none; padding: 0; min-height: 24px; cursor: pointer;
    font-size: var(--text-2xs, 10px); font-weight: 700; text-transform: uppercase;
    letter-spacing: 0.08em; color: var(--text-tertiary);
    -webkit-tap-highlight-color: transparent;
  }
  .d-recent-head:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; border-radius: var(--radius-sm, 6px); }
  .d-recent-more { color: var(--primary-300, #c4b5fd); font-weight: 600; text-transform: none; letter-spacing: 0; }
  .d-recent-list { display: flex; flex-direction: column; gap: var(--space-1); }
  .d-recent-row {
    display: flex; align-items: center; gap: var(--space-2); min-height: 40px;
    padding: var(--space-2) var(--space-3);
    background: var(--glass-bg, rgba(255,255,255,0.03));
    border: 1px solid var(--border-default, rgba(255,255,255,0.06));
    border-radius: var(--radius-md, 12px);
    cursor: pointer; text-align: left; color: inherit; font: inherit;
    -webkit-tap-highlight-color: transparent;
    transition: border-color var(--duration-fast, 150ms) var(--ease-out), background var(--duration-fast, 150ms) var(--ease-out);
  }
  .d-recent-row:hover { border-color: rgba(255,255,255,0.12); background: rgba(255,255,255,0.05); }
  .d-recent-row:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .d-recent-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; background: var(--primary-400); }
  .d-recent-dot.d-recent-offline { background: var(--text-tertiary); }
  .d-recent-dot.d-recent-sos_start { background: var(--danger-500); }
  .d-recent-dot.d-recent-sos_end { background: var(--success-500); }
  .d-recent-dot.d-recent-contact { background: var(--success-400, #34d399); }
  .d-recent-msg { flex: 1; min-width: 0; font-size: var(--text-xs, 12px); color: var(--text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .d-recent-ts { font-size: 10px; color: var(--text-tertiary); font-variant-numeric: tabular-nums; flex-shrink: 0; }

  .d-spacer { height: calc(var(--safe-bottom, 0px) + var(--space-8)); flex-shrink: 0; }

  @media (prefers-reduced-motion: reduce) {
    .d-ambient { transition: none !important; }
    .act-sos-on { animation: none !important; }
    .d-greet, .d-verdict-slot, .d-pulse-slot, .d-family, .d-invite-slot, .d-week-slot, .d-recent, .d-actions {
      opacity: 1 !important; transform: none !important; transition: none !important; transition-delay: 0ms !important;
    }
  }
</style>
