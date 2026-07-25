<script>
  import { run } from 'svelte/legacy';

  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { authUser } from '../lib/stores/auth.js';
  import { myLocation } from '../lib/stores/map.js';
  import { mySosActive } from '../lib/stores/sos.js';
  import { allowWebGL } from '../lib/stores/effects.js';
  import { clearHubBadge } from '../lib/stores/hubBadge.js';
  import { familyVerdict, familyMembers, verdictNow } from '../lib/stores/verdict.js';
  import { formatAge } from '../lib/presence.js';
  import { activityEvents } from '../lib/activityLog.js';
  import { rhythmEnabled, setRhythmEnabled } from '../lib/presenceRhythm.js';
  import HubVerdict from '../components/hub/HubVerdict.svelte';
  import HomecomingRail from '../components/hub/HomecomingRail.svelte';
  import FamilyRoster from '../components/hub/FamilyRoster.svelte';
  import PulseButton from '../components/hub/PulseButton.svelte';
  import InviteStrip from '../components/hub/InviteStrip.svelte';
  import WeeklyRhythm from '../components/hub/WeeklyRhythm.svelte';
  import GlobeCanvas from '../components/primitives/GlobeCanvas.svelte';

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
    updateGlobeSize();
    window.addEventListener('resize', updateGlobeSize);
  });
  onDestroy(() => {
    window.removeEventListener('resize', updateGlobeSize);
  });

  let mounted = $state(false);
  let nowDate = $derived(new Date(nowMs));
  let timeStr = $derived(nowDate.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }));
  let dateStr = $derived(nowDate.toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric' }));
  function greeting() {
    const h = nowDate.getHours();
    if (h < 5) return 'Up late'; if (h < 12) return 'Good morning';
    if (h < 17) return 'Good afternoon'; return 'Good evening';
  }
  let firstName = $derived(($authUser?.displayName || '').split(' ')[0] || 'there');

  // ── Derived family state — read from the shared verdict store (zero DB) ────
  let members = $derived($familyMembers);
  let verdict = $derived($familyVerdict);

  // ── Globe (demoted): desktop-only ambient flourish, gated by allowWebGL ─────
  let globeSize = $state(420);
  function updateGlobeSize() {
    if (typeof window === 'undefined') return;
    const vw = window.innerWidth, vh = window.innerHeight;
    if (vw < 768) { globeSize = 0; return; }
    const sidebarW = vw >= 1200 ? Math.min(460, vw * 0.35) : Math.min(420, vw * 0.38);
    const leftW = Math.max(400, vw - sidebarW);
    globeSize = Math.round(Math.min(560, Math.max(320, Math.min(vh * 0.6, leftW * 0.62))));
  }
</script>

<div class="d d-{verdict.tone}" class:d-ready={mounted}>
  <!-- ONE ambient tint, driven by the real verdict tone (replaces aurora+noise+glow pile-up) -->
  <div class="d-ambient" aria-hidden="true"></div>

  <!-- Demoted globe — desktop-only living backdrop, skipped in calm/low-end via allowWebGL -->
  {#if $allowWebGL && globeSize > 0}
    <div class="d-globe-flourish" aria-hidden="true">
      <GlobeCanvas size={globeSize} />
    </div>
  {/if}

  <!-- Header -->
  <header class="d-header">
    <button class="d-back tactile" onclick={() => push('/')} aria-label="Back to map">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="15 18 9 12 15 6"/></svg>
      Map
    </button>
    <div class="d-clock">{timeStr}</div>
  </header>

  <!-- Content -->
  <div class="d-content">
    <section class="d-greet">
      <h1 class="d-name"><span class="d-greet-word">{greeting()},&nbsp;</span><span class="d-name-word">{firstName}</span></h1>
      <p class="d-date">{dateStr}</p>
    </section>

    <div class="d-verdict-slot">
      <HubVerdict {verdict} onopen={() => push('/')} />
    </div>

    <div class="d-pulse-slot">
      <PulseButton />
    </div>

    <HomecomingRail />

    <section class="d-family">
      <FamilyRoster {members} myLocation={$myLocation} now={nowMs} />
      {#if members.length > 0}
        <button class="d-rhythm-toggle" role="switch" aria-checked={$rhythmEnabled}
          onclick={() => setRhythmEnabled(!$rhythmEnabled)}>
          <span class="d-rhythm-dot" class:on={$rhythmEnabled}></span>
          {$rhythmEnabled ? 'Rhythm hints on' : 'Show rhythm hints'}
        </button>
      {/if}
    </section>

    <div class="d-invite-slot">
      <InviteStrip {members} />
    </div>

    <div class="d-week-slot">
      <WeeklyRhythm />
    </div>

    {#if $activityEvents.length > 0}
      <section class="d-recent" aria-label="Recent activity">
        <button class="d-recent-head" onclick={() => push('/activity')}>
          <span>Recent</span>
          <span class="d-recent-more">All activity ›</span>
        </button>
        <div class="d-recent-list">
          {#each $activityEvents.slice(0, 2) as ev (ev.id)}
            <button class="d-recent-row" onclick={() => push('/activity')}>
              <span class="d-recent-dot d-recent-{ev.type}" aria-hidden="true"></span>
              <span class="d-recent-msg">{ev.message}</span>
              <span class="d-recent-ts">{formatAge(nowMs - ev.ts)}</span>
            </button>
          {/each}
        </div>
      </section>
    {/if}

    <!-- Quick actions — slim, specific, thumb-reachable -->
    <section class="d-actions" aria-label="Quick actions">
      <button class="d-act act-map tactile" onclick={() => visitFeature(null, '/')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M1 6v16l7-4 8 4 7-4V2l-7 4-8-4-7 4z"/><line x1="8" y1="2" x2="8" y2="18"/><line x1="16" y1="6" x2="16" y2="22"/></svg>
        <span>Live Map</span>
      </button>
      <button class="d-act act-activity tactile" onclick={() => visitFeature('activity', '/activity')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
        <span>Activity</span>
        {#if !visited.activity}<span class="d-dot"></span>{/if}
      </button>
      <button class="d-act act-replay tactile" onclick={() => visitFeature('replay', '/replay')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 .49-4.61"/></svg>
        <span>Routes</span>
        {#if !visited.replay}<span class="d-dot"></span>{/if}
      </button>
      <button class="d-act act-sos tactile" class:act-sos-on={$mySosActive} onclick={() => visitFeature('emergency', '/emergency')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
        <span>Emergency</span>
        {#if !visited.emergency}<span class="d-dot d-dot-red"></span>{/if}
      </button>
      <button class="d-act act-checkin tactile" onclick={() => visitFeature('checkins', '/checkins')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
        <span>Check-ins</span>
        {#if !visited.checkins}<span class="d-dot d-dot-cyan"></span>{/if}
      </button>
    </section>

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
  .d-safe    { --amb-a: var(--success-500-12, rgba(16,185,129,0.10)); --amb-b: var(--primary-500-08, rgba(99,102,241,0.08)); }
  .d-caution { --amb-a: var(--warning-500-12, rgba(245,158,11,0.12)); --amb-b: var(--primary-500-08, rgba(99,102,241,0.06)); }
  .d-alert   { --amb-a: var(--danger-500-12, rgba(239,68,68,0.14));   --amb-b: var(--danger-500-08, rgba(239,68,68,0.06)); }

  /* ── Demoted globe flourish (desktop only, behind content) ────────────────── */
  .d-globe-flourish { display: none; }
  @media (min-width: 768px) {
    .d-globe-flourish {
      display: flex; align-items: center; justify-content: center;
      position: absolute; top: 0; bottom: 0; left: 0;
      right: min(420px, 38vw); z-index: 1; pointer-events: none; opacity: 0.85;
    }
  }
  @media (min-width: 1200px) { .d-globe-flourish { right: min(460px, 35vw); } }

  /* ── Header ───────────────────────────────────────────────────────────────── */
  .d-header {
    position: fixed; top: 0; left: 0; right: 0; z-index: 20;
    display: flex; align-items: center; justify-content: space-between;
    padding: calc(var(--safe-top, 0px) + var(--space-2)) var(--space-5) var(--space-2);
    background: rgba(5,8,18,0.55);
    backdrop-filter: blur(24px) saturate(1.5); -webkit-backdrop-filter: blur(24px) saturate(1.5);
    border-bottom: 1px solid var(--border-default, rgba(255,255,255,0.04));
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
  .d-back:hover { background: rgba(255,255,255,0.10); color: var(--text-primary); }
  .d-back:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .d-clock { font-size: clamp(1.125rem, 1.4vw, 1.25rem); font-weight: 700; letter-spacing: -0.03em; color: var(--text-primary); font-variant-numeric: tabular-nums; font-family: var(--font-display, system-ui); }

  /* ── Content column ───────────────────────────────────────────────────────── */
  .d-content {
    position: relative; z-index: 5;
    height: 100dvh; overflow-y: auto;
    -webkit-overflow-scrolling: touch; overscroll-behavior-y: contain;
    padding: calc(var(--safe-top, 0px) + 52px) 0 0;
    display: flex; flex-direction: column; gap: var(--space-4);
  }
  .d-greet, .d-verdict-slot, .d-pulse-slot, .d-family, .d-invite-slot, .d-week-slot, .d-recent, .d-actions { padding-left: var(--space-5); padding-right: var(--space-5); }

  @media (min-width: 768px) {
    .d-content {
      position: absolute; top: 0; right: 0; bottom: 0;
      width: min(420px, 38vw);
      padding: calc(var(--safe-top, 0px) + 60px) 0 0;
      background: linear-gradient(90deg, transparent 0%, rgba(5,8,18,0.6) 30%, rgba(5,8,18,0.9) 100%);
      backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px);
    }
  }
  @media (min-width: 1200px) { .d-content { width: min(460px, 35vw); } }

  /* Staggered entrance — transform/opacity only */
  .d-greet, .d-verdict-slot, .d-pulse-slot, .d-family, .d-invite-slot, .d-week-slot, .d-recent, .d-actions {
    opacity: 0; transform: translateY(10px);
    transition: opacity var(--duration-slow, 500ms) var(--ease-out), transform var(--duration-slow, 500ms) var(--ease-out);
  }
  .d-ready .d-greet        { opacity: 1; transform: none; transition-delay: 40ms; }
  .d-ready .d-verdict-slot { opacity: 1; transform: none; transition-delay: 90ms; }
  .d-ready .d-pulse-slot   { opacity: 1; transform: none; transition-delay: 130ms; }
  .d-ready .d-family       { opacity: 1; transform: none; transition-delay: 170ms; }
  .d-ready .d-invite-slot  { opacity: 1; transform: none; transition-delay: 210ms; }
  .d-ready .d-week-slot    { opacity: 1; transform: none; transition-delay: 240ms; }
  .d-ready .d-recent       { opacity: 1; transform: none; transition-delay: 270ms; }
  .d-ready .d-actions      { opacity: 1; transform: none; transition-delay: 300ms; }

  .d-greet { margin: 0; }
  .d-name { margin: 0; font-family: var(--font-display, system-ui); font-size: clamp(1.5rem, 4.5vw, 2rem); font-weight: 400; letter-spacing: -0.03em; line-height: 1.15; color: var(--text-secondary); }
  .d-greet-word { font-weight: 400; color: var(--text-secondary); }
  .d-name-word { font-weight: 800; background: linear-gradient(135deg, #fff 30%, var(--primary-300, #c4b5fd) 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }
  .d-date { margin: var(--space-1) 0 0; font-size: var(--text-xs); color: var(--text-tertiary); }

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
