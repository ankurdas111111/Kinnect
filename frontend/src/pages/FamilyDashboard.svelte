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
  import { myContacts } from '../lib/stores/contacts.js';
  import { fmtWhen } from '../lib/hubStatus.js';
  import { socket } from '../lib/socket.js';
  import HomecomingRail from '../components/hub/HomecomingRail.svelte';
  import FamilyRoster from '../components/hub/FamilyRoster.svelte';
  import PulseButton from '../components/hub/PulseButton.svelte';
  import InviteStrip from '../components/hub/InviteStrip.svelte';
  import WeeklyRhythm from '../components/hub/WeeklyRhythm.svelte';
  import CirclesSection from '../components/hub/CirclesSection.svelte';
  import { setMobileTab } from '../lib/stores/uiShell.js';

  // Everyone = the verdict view (how is everyone doing); Circles = the rooms
  // you belong to (which groups, who's in each, anything waiting on you).
  let view = $state('everyone');
  function goManageCircles() {
    // Land on the map with the Connect/sharing surface active — create, join,
    // leave and admin votes all live there.
    setMobileTab('share');
    push('/');
  }

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
  // Zero-member state: nothing to report on, so "Invite" is the one real
  // action — it takes the filled-ember emphasis, "I'm safe" quiets down.
  let zeroMembers = $derived(members.length === 0);

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
  // "Sharing" needs real coordinates — the hub stamps lastUpdate at connect
  // time, so a merely-online member with no position must still read dormant.
  let liveIds = $derived(
    new Set(
      members
        .filter((m) => m.lat != null && m.lastUpdate && nowMs - m.lastUpdate < 300_000)
        .map((m) => m.userId)
    )
  );
  // Hearth 06b: the dormant list comes from myContacts (which includes offline
  // people the live member map never sees), with online/lastUpdate from the hub.
  let notSharing = $derived(($myContacts || []).filter((c) => !liveIds.has(c.userId)));
  let nobodySharing = $derived(
    (members.length > 0 || ($myContacts || []).length > 0) && liveIds.size === 0
  );

  // Nudge — one per person per dashboard visit; the button says what happened.
  let nudged = $state(new Set());
  function nudge(userId) {
    if (nudged.has(userId)) return;
    socket.emit('nudgeUser', { userId });
    nudged = new Set([...nudged, userId]);
  }
  function nudgeAll() {
    notSharing.forEach((c) => nudge(c.userId));
  }
  function lastSharedLabel(c) {
    if (!c.lastUpdate) return "hasn't shared yet";
    const age = nowMs - c.lastUpdate;
    if (age < 86_400_000) return 'last shared ' + fmtWhen(c.lastUpdate, nowMs).replace('since ', 'at ');
    return 'last shared ' + new Date(c.lastUpdate).toLocaleDateString([], { day: 'numeric', month: 'long' });
  }
</script>

<!-- Family Circle: one warm-paper column (max 640px), the verdict spoken at
     the crest, people as pebble rows, everything else quiet prose. -->
<div class="fc fc-{verdict.tone}" class:fc-ready={mounted}>
  <!-- Top app bar — paper, frameless, wordmark in the serif voice -->
  <header class="fc-bar">
    <div class="fc-bar-inner">
      <button class="fc-back" onclick={() => push('/')} aria-label="Back to map">
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="15 18 9 12 15 6"/></svg>
      </button>
      <span class="fc-wordmark verdict-voice">Kinnect</span>
      <div class="fc-bar-right">
        <span class="fc-when">{dateStr} · {timeStr}</span>
        <span class="fc-avatar" aria-hidden="true">{initials}</span>
      </div>
    </div>
  </header>

  <main class="fc-col">
    <!-- Verdict crest: frameless, honest human sentence -->
    <section class="fc-crest" aria-labelledby="fc-crest-head">
      <p class="fc-eyebrow" id="fc-crest-head">
        <span class="fc-eyebrow-dot" aria-hidden="true"></span>
        {familyName} · right now
      </p>
      <h1 class="fc-verdict verdict-voice">{verdict.sentence}</h1>
      {#if nobodySharing}
        <p class="fc-crest-detail">
          Nobody in {familyName} is sharing at the moment. One nudge asks everyone to turn
          sharing back on — or invite someone new.
        </p>
      {:else if verdict.detail}
        <p class="fc-crest-detail">{verdict.detail}</p>
      {/if}

      <!-- One row of real actions, ember reserved for the primary one -->
      <div class="fc-actions">
        <div class="fc-action-pulse"><PulseButton quiet={zeroMembers} /></div>
        <button class="fc-action" onclick={() => push('/')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6"/><line x1="12" y1="2" x2="12" y2="15"/></svg>
          Share live
        </button>
        <button class="fc-action" class:fc-action-filled={zeroMembers} onclick={() => push('/')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          Invite
        </button>
      </div>
    </section>

    <!-- Everyone (the verdict view) vs Circles (the rooms you belong to) -->
    <div class="fc-tabs" role="tablist" aria-label="Dashboard view">
      <button
        class="fc-tab" class:fc-tab-active={view === 'everyone'}
        role="tab" aria-selected={view === 'everyone'}
        onclick={() => (view = 'everyone')}
      >Everyone</button>
      <button
        class="fc-tab" class:fc-tab-active={view === 'circles'}
        role="tab" aria-selected={view === 'circles'}
        onclick={() => (view = 'circles')}
      >Circles{#if $myRooms.length}<span class="fc-tab-count">{$myRooms.length}</span>{/if}</button>
    </div>

    {#if view === 'circles'}
    <section class="fc-section" aria-label="Your circles">
      <CirclesSection
        rooms={$myRooms}
        contacts={$myContacts}
        {members}
        meId={$authUser?.userId || ''}
        meName={$authUser?.displayName || ''}
        onManage={goManageCircles}
      />
    </section>
    {:else}
    <!-- Circle status: continuous quiet paper rows -->
    <section class="fc-section" aria-label="Your circle">
      <div class="fc-section-head">
        <h2 class="fc-label">Your circle{members.length ? ` (${members.length})` : ''}</h2>
      </div>
      <FamilyRoster {members} myLocation={$myLocation} now={nowMs} />
      {#if members.length > 0}
        <button class="fc-rhythm-toggle" role="switch" aria-checked={$rhythmEnabled}
          onclick={() => setRhythmEnabled(!$rhythmEnabled)}>
          <span class="fc-rhythm-dot" class:on={$rhythmEnabled} aria-hidden="true"></span>
          {$rhythmEnabled ? 'Rhythm hints on' : 'Show rhythm hints'}
        </button>
      {/if}
      <InviteStrip {members} />
    </section>

    <!-- Where everyone is — names as pebble chips, dormant people nudgeable -->
    <section class="fc-section" aria-label="Where everyone is">
      <div class="fc-section-head">
        <h2 class="fc-label">Where everyone is</h2>
        <button class="fc-more" onclick={() => push('/')}>Open map →</button>
      </div>
      {#if members.length === 0}
        <p class="fc-empty">Nobody to show yet. Invite someone and they'll appear here.</p>
      {:else}
        <div class="fc-chips">
          {#each members.slice(0, 8) as m (m.userId || m.socketId)}
            <span class="fc-chip" style="--hue: var(--member-{(Math.abs([...(m.displayName || '?')].reduce((a, c) => a + c.charCodeAt(0), 0)) % 4) + 1})">
              <span class="fc-chip-dot" aria-hidden="true"></span>
              {(m.displayName || '?').split(' ')[0]}
            </span>
          {/each}
        </div>
      {/if}
      {#if notSharing.length > 0}
        <!-- Hearth 06b: dormant members are named, dated, and nudgeable -->
        <div class="fc-dormant">
          <div class="fc-dormant-head-row">
            <h3 class="fc-sublabel">Not sharing · {notSharing.length}</h3>
            <button class="fc-more" onclick={nudgeAll}>Nudge all</button>
          </div>
          {#each notSharing as c (c.userId)}
            <div class="fc-dormant-row">
              <span class="fc-dormant-name">{(c.displayName || '?').split(' ')[0]}</span>
              <span class="fc-dormant-when">{lastSharedLabel(c)}</span>
              <button
                class="fc-nudge"
                disabled={nudged.has(c.userId)}
                onclick={() => nudge(c.userId)}
                aria-label="Nudge {c.displayName} to share their location"
              >{nudged.has(c.userId) ? 'Nudged ✓' : 'Nudge'}</button>
            </div>
          {/each}
        </div>
      {/if}
    </section>

    <!-- Coming home -->
    <section class="fc-section" aria-label="Coming home">
      <div class="fc-section-head">
        <h2 class="fc-label">Coming home</h2>
      </div>
      <!-- HomecomingRail self-hides when there is nothing to project, which
           left an empty section. 06b: say what would fill it. -->
      {#if $arrivalProjections && $arrivalProjections.size > 0}
        <HomecomingRail />
      {:else}
        <p class="fc-empty-strong">Nobody's heading home right now.</p>
        <p class="fc-empty">This fills in once someone sets a Home place and starts sharing.</p>
      {/if}
    </section>

    <!-- Today -->
    <section class="fc-section" aria-label="Today">
      <div class="fc-section-head">
        <h2 class="fc-label">Today</h2>
        {#if $activityEvents.length > 0}
          <button class="fc-more" onclick={() => push('/activity')}>All activity →</button>
        {/if}
      </div>
      {#if $activityEvents.length === 0}
        <!-- 06b: an empty day is a sentence, not a blank panel -->
        <p class="fc-empty-strong">Nothing happened today.</p>
        <p class="fc-empty">A quiet day is a good day. Nothing to review.</p>
      {:else}
        <div class="fc-today">
          {#each $activityEvents.slice(0, 3) as ev (ev.id)}
            <button class="fc-today-row" onclick={() => push('/activity')}>
              <span class="fc-today-ts">{formatAge(nowMs - ev.ts)}</span>
              <span class="fc-event-dot fc-event-{ev.type}" aria-hidden="true"></span>
              <span class="fc-today-msg">{ev.message}</span>
            </button>
          {/each}
        </div>
      {/if}
    </section>

    <!-- WeeklyRhythm renders its own "Your week / Private to you" header —
         no wrapper header here, or the section shows the title twice. -->
    <section class="fc-section" aria-label="Your week">
      <WeeklyRhythm />
    </section>
    {/if}

    <!-- Slim text nav, not a wall of tiles -->
    <nav class="fc-nav" aria-label="More">
      <button class="fc-nav-link" onclick={() => visitFeature('activity', '/activity')}>
        Activity{#if !visited.activity}<span class="fc-new" aria-hidden="true"></span><span class="sr-only">(new)</span>{/if}
      </button>
      <button class="fc-nav-link" onclick={() => visitFeature('replay', '/replay')}>
        Routes{#if !visited.replay}<span class="fc-new" aria-hidden="true"></span><span class="sr-only">(new)</span>{/if}
      </button>
      <button class="fc-nav-link" onclick={() => visitFeature('checkins', '/checkins')}>
        Check-ins{#if !visited.checkins}<span class="fc-new" aria-hidden="true"></span><span class="sr-only">(new)</span>{/if}
      </button>
      <button class="fc-nav-link" onclick={() => push('/')}>Places</button>
      <button class="fc-nav-link" class:fc-nav-sos={$mySosActive} onclick={() => visitFeature('emergency', '/emergency')}>
        Emergency{#if !visited.emergency}<span class="fc-new" aria-hidden="true"></span><span class="sr-only">(new)</span>{/if}
      </button>
    </nav>

    <div class="fc-spacer"></div>
  </main>
</div>

<style>
  /* ── Ground: warm paper, one calm column ─────────────────────────────────── */
  .fc {
    height: 100dvh; overflow-y: auto;
    -webkit-overflow-scrolling: touch; overscroll-behavior-y: contain;
    background: var(--paper);
    color: var(--text-primary);
    font-family: var(--font-sans);
    opacity: 0; transition: opacity var(--duration-slow) var(--ease-out);
  }
  .fc.fc-ready { opacity: 1; }

  /* ── Top app bar — sticky paper, no border, no glass ────────────────────── */
  .fc-bar {
    position: sticky; top: 0; z-index: 20;
    background: var(--paper);
    padding-top: var(--safe-top, 0px);
  }
  .fc-bar-inner {
    max-width: 640px; margin-inline: auto;
    display: flex; align-items: center; justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-2) var(--space-4);
    min-height: 64px;
  }
  .fc-back {
    width: 44px; height: 44px; flex-shrink: 0;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none; border-radius: var(--radius-full);
    color: var(--primary-700); cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out), transform var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .fc-back:hover { background: var(--surface-3); }
  .fc-back:active { transform: scale(0.95); }
  .fc-back:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }
  /* Screen-title register — the one other place the serif voice speaks */
  .fc-wordmark {
    font-size: var(--text-xl);
    color: var(--primary-700);
    letter-spacing: -0.01em;
  }
  .fc-bar-right { display: flex; align-items: center; gap: var(--space-3); min-width: 0; }
  .fc-when {
    font-size: var(--text-xs); color: var(--text-tertiary);
    font-variant-numeric: tabular-nums; white-space: nowrap;
  }
  @media (max-width: 640px) {
    .fc-when { display: none; }
  }
  /* Your own pebble */
  .fc-avatar {
    width: 32px; height: 32px; border-radius: var(--radius-full); flex-shrink: 0;
    display: flex; align-items: center; justify-content: center;
    background: var(--primary-100); color: var(--primary-700);
    font-size: var(--text-xs); font-weight: 600; letter-spacing: 0.02em;
  }

  /* ── The column — max 640px, sections separated by whitespace only ──────── */
  .fc-col {
    max-width: 640px; margin-inline: auto;
    padding: var(--space-2) var(--space-4) var(--space-8);
    display: flex; flex-direction: column; gap: var(--space-8);
  }
  /* ── Everyone | Circles — the paper pill switcher (auth's segment idiom) ── */
  .fc-tabs {
    display: flex;
    align-self: flex-start;
    background: var(--surface-3);
    border-radius: var(--radius-full);
    padding: var(--space-1);
    gap: var(--space-1);
  }
  .fc-tab {
    min-height: 44px;
    padding: 0 var(--space-5);
    border: none; cursor: pointer;
    background: none;
    border-radius: var(--radius-full);
    font-family: var(--font-sans);
    font-size: var(--text-sm); font-weight: 600;
    color: var(--text-secondary);
    display: inline-flex; align-items: center; gap: var(--space-2);
    transition: background var(--duration-fast) var(--ease-out),
                color var(--duration-fast) var(--ease-out);
  }
  .fc-tab:hover { color: var(--text-primary); }
  .fc-tab:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }
  .fc-tab-active {
    background: var(--surface-1);
    color: var(--text-primary);
    box-shadow: var(--shadow-xs);
  }
  .fc-tab-count {
    font-size: var(--text-xs); font-weight: 600;
    color: var(--primary-700);
    background: var(--primary-100);
    border-radius: var(--radius-full);
    padding: 1px var(--space-2);
  }
  @media (prefers-reduced-motion: reduce) {
    .fc-tab { transition: none; }
  }

  .fc-section { display: flex; flex-direction: column; gap: var(--space-4); min-width: 0; }

  /* Gentle entrance — opacity/transform only */
  .fc-crest, .fc-section, .fc-nav {
    opacity: 0; transform: translateY(8px);
    transition: opacity var(--duration-slow) var(--ease-out), transform var(--duration-slow) var(--ease-out);
  }
  .fc-ready .fc-crest, .fc-ready .fc-section, .fc-ready .fc-nav { opacity: 1; transform: none; }
  .fc-ready .fc-crest { transition-delay: 40ms; }
  .fc-ready .fc-section { transition-delay: 120ms; }
  .fc-ready .fc-nav { transition-delay: 180ms; }

  /* ── Verdict crest — a frameless typographic statement ──────────────────── */
  .fc-crest { display: flex; flex-direction: column; padding-top: var(--space-2); }
  .fc-eyebrow {
    margin: 0 0 var(--space-2);
    display: flex; align-items: center; gap: var(--space-2);
    font-size: var(--text-2xs); font-weight: 600;
    letter-spacing: 0.1em; text-transform: uppercase;
    color: var(--text-secondary);
  }
  /* Tone dot — sage settled, ochre needs-a-look, vermilion strictly SOS
     (alert only ever means SOS). Words beside it, never color alone. */
  .fc-eyebrow-dot { width: 8px; height: 8px; border-radius: var(--radius-full); flex-shrink: 0; }
  .fc-safe    .fc-eyebrow-dot { background: var(--success-500); }
  .fc-caution .fc-eyebrow-dot { background: var(--warning-500); }
  .fc-alert   .fc-eyebrow-dot { background: var(--danger-500); }

  .fc-verdict {
    margin: 0;
    font-size: clamp(var(--text-2xl), 4vw, var(--text-3xl));
    line-height: 1.4;
    color: var(--text-primary);
  }
  .fc-crest-detail {
    margin: var(--space-2) 0 0;
    font-size: var(--text-base); line-height: 1.5;
    color: var(--text-secondary);
    max-width: 54ch;
  }

  /* ── Companionship triggers — one ember fill, paper outlines beside it ──── */
  .fc-actions {
    display: flex; align-items: stretch; gap: var(--space-3);
    flex-wrap: wrap; margin-top: var(--space-5);
  }
  .fc-action-pulse { flex: 1 1 100%; }
  @media (min-width: 480px) {
    .fc-action-pulse { flex: 1 1 auto; min-width: 0; }
  }
  .fc-action {
    display: inline-flex; align-items: center; justify-content: center; gap: var(--space-2);
    min-height: 48px; padding: 0 var(--space-5);
    background: transparent; color: var(--primary-700);
    border: 1px solid var(--outline-variant); border-radius: var(--radius-md);
    font-size: var(--text-base); font-weight: 500; font-family: inherit; cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out), border-color var(--duration-fast) var(--ease-out), transform var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .fc-action:hover { background: var(--surface-hover); }
  .fc-action:active { transform: scale(0.97); }
  .fc-action:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }
  /* Zero-member state only: Invite becomes the one real action */
  .fc-action-filled {
    background: var(--primary-500);
    border-color: transparent;
    color: var(--text-on-primary);
    font-weight: 600;
    box-shadow: var(--shadow-xs);
  }
  .fc-action-filled:hover { background: var(--primary-600); }

  /* ── Section furniture — quiet labels, text links, honest empty states ──── */
  .fc-section-head {
    display: flex; align-items: baseline; justify-content: space-between;
    gap: var(--space-3);
  }
  .fc-label {
    margin: 0; font-family: var(--font-sans);
    font-size: var(--text-sm); font-weight: 500;
    letter-spacing: 0.01em; color: var(--text-secondary);
  }
  .fc-sublabel {
    margin: 0; font-size: var(--text-xs); font-weight: 600;
    color: var(--text-tertiary); letter-spacing: 0.02em;
  }
  .fc-more {
    background: none; border: none; padding: var(--space-1); margin: calc(-1 * var(--space-1));
    cursor: pointer; font-family: inherit;
    font-size: var(--text-sm); font-weight: 600;
    color: var(--primary-700);
    min-height: 44px; display: inline-flex; align-items: center;
    -webkit-tap-highlight-color: transparent;
  }
  .fc-more:hover { text-decoration: underline; }
  .fc-more:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; border-radius: var(--radius-sm); }

  /* Honest empty states (06b) — a sentence, never a blank panel */
  .fc-empty-strong { margin: 0; font-size: var(--text-base); font-weight: 500; color: var(--text-primary); }
  .fc-empty { margin: 0; font-size: var(--text-base); line-height: 1.5; color: var(--text-tertiary); }

  /* ── Name chips — warm capsules, not coordinates ────────────────────────── */
  .fc-chips { display: flex; flex-wrap: wrap; gap: var(--space-2); }
  .fc-chip {
    display: inline-flex; align-items: center; gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
    background: var(--surface-2);
    border-radius: var(--radius-full);
    font-size: var(--text-sm); font-weight: 500; color: var(--text-secondary);
  }
  .fc-chip-dot { width: 8px; height: 8px; border-radius: var(--radius-full); background: var(--hue, var(--primary-500)); }

  /* ── Dormant members — quiet rows, one Nudge each ───────────────────────── */
  .fc-dormant {
    display: flex; flex-direction: column; gap: var(--space-2);
    padding-top: var(--space-3);
  }
  .fc-dormant-head-row { display: flex; align-items: center; justify-content: space-between; gap: var(--space-2); }
  .fc-dormant-row {
    display: grid; grid-template-columns: auto 1fr auto; align-items: center;
    gap: var(--space-3); min-height: 44px;
  }
  .fc-dormant-name { font-size: var(--text-base); font-weight: 500; color: var(--text-primary); }
  .fc-dormant-when {
    font-size: var(--text-xs); color: var(--text-tertiary);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .fc-nudge {
    min-height: 44px; padding: 0 var(--space-4);
    background: transparent;
    border: 1px solid var(--outline-variant);
    border-radius: var(--radius-md);
    font-family: inherit; font-size: var(--text-sm); font-weight: 600;
    color: var(--primary-700); cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .fc-nudge:hover:not(:disabled) { background: var(--surface-hover); }
  .fc-nudge:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }
  .fc-nudge:disabled {
    cursor: default; color: var(--text-tertiary);
    border-color: transparent; background: transparent;
  }

  /* ── Today — time, dot, sentence; whitespace between rows ───────────────── */
  .fc-today { display: flex; flex-direction: column; gap: var(--space-2); }
  .fc-today-row {
    display: grid; grid-template-columns: auto auto 1fr; align-items: center;
    gap: var(--space-3); min-height: 44px;
    padding: var(--space-1) var(--space-2); margin: 0 calc(-1 * var(--space-2));
    background: none; border: none; border-radius: var(--radius-md);
    text-align: left; cursor: pointer;
    font-family: inherit; color: inherit;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .fc-today-row:hover { background: var(--surface-hover); }
  .fc-today-row:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }
  .fc-today-ts {
    font-size: var(--text-xs); color: var(--text-tertiary);
    font-variant-numeric: tabular-nums; min-width: 4.5ch;
  }
  .fc-event-dot { width: 6px; height: 6px; border-radius: var(--radius-full); flex-shrink: 0; background: var(--primary-500); }
  .fc-event-offline { background: var(--status-offline); }
  /* SOS events are the one sanctioned vermilion outside the SOS screens */
  .fc-event-sos_start { background: var(--danger-500); }
  .fc-event-sos_end { background: var(--success-500); }
  .fc-event-contact { background: var(--success-500); }
  .fc-today-msg { font-size: var(--text-base); line-height: 1.5; color: var(--text-primary); }

  /* ── Rhythm hints opt-in toggle (subtle, non-alarmist) ──────────────────── */
  .fc-rhythm-toggle {
    display: inline-flex; align-items: center; gap: var(--space-2);
    align-self: flex-start; min-height: 44px;
    padding: var(--space-1) var(--space-2); margin-left: calc(-1 * var(--space-2));
    background: none; border: none; cursor: pointer;
    font-family: inherit; font-size: var(--text-xs); font-weight: 500;
    color: var(--text-tertiary);
    -webkit-tap-highlight-color: transparent;
  }
  .fc-rhythm-toggle:hover { color: var(--text-secondary); }
  .fc-rhythm-toggle:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; border-radius: var(--radius-sm); }
  .fc-rhythm-dot {
    width: 8px; height: 8px; border-radius: var(--radius-full);
    background: var(--outline-variant);
    transition: background var(--duration-fast) var(--ease-out);
  }
  .fc-rhythm-dot.on { background: var(--primary-500); }

  /* ── Slim text nav ──────────────────────────────────────────────────────── */
  .fc-nav { display: flex; flex-wrap: wrap; gap: var(--space-5); row-gap: 0; }
  .fc-nav-link {
    position: relative;
    background: none; border: none; padding: var(--space-2) 0; cursor: pointer;
    font-family: inherit; font-size: var(--text-base); font-weight: 500;
    color: var(--text-secondary); min-height: 44px;
    -webkit-tap-highlight-color: transparent;
  }
  .fc-nav-link:hover { color: var(--text-primary); }
  .fc-nav-link:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; border-radius: var(--radius-sm); }
  /* Active SOS is the one place this nav may speak vermilion */
  .fc-nav-sos { color: var(--danger-500); font-weight: 600; }
  /* First-run "new" dot — a quiet ember fleck beside the label */
  .fc-new {
    position: absolute; top: var(--space-2); right: calc(-1 * var(--space-2));
    width: 6px; height: 6px; border-radius: var(--radius-full);
    background: var(--primary-500);
  }

  .fc-spacer { height: calc(var(--safe-bottom, 0px) + var(--space-8)); flex-shrink: 0; }

  @media (prefers-reduced-motion: reduce) {
    .fc, .fc-crest, .fc-section, .fc-nav {
      opacity: 1 !important; transform: none !important;
      transition: none !important; transition-delay: 0ms !important;
    }
    .fc-back:active, .fc-action:active { transform: none; }
  }
</style>
