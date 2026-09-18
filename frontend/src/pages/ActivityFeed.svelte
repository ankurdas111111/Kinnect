<script>
  import { run } from 'svelte/legacy';

  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { authUser } from '../lib/stores/auth.js';
  import { clearHubBadge } from '../lib/stores/hubBadge.js';
  import { activityEvents, clearActivity } from '../lib/activityLog.js';
  import { connectivityStore } from '../lib/stores/connectivity.js';
  import { deriveConnState, formatAge } from '../lib/presence.js';
  import { allowMotion } from '../lib/stores/effects.js';
  import EmptyState from '../components/primitives/EmptyState.svelte';
  import StatusBadge from '../components/primitives/StatusBadge.svelte';
  import AvatarRing from '../components/primitives/AvatarRing.svelte';
  import { getUserColor, getUserColorLight } from '../lib/getUserColor.js';

  run(() => {
    if (!$authUser) push('/login');
  });

  // ── Event log — now device-persistent (survives refresh) via activityLog.js ──
  let activeFilter = $state('all');
  let now = Date.now();
  let clockInterval;

  const FILTERS = [
    { key: 'all',    label: 'All' },
    { key: 'people', label: 'People' },
    { key: 'safety', label: 'Safety' },
    { key: 'me',     label: 'Me' },
  ];

  const TYPE_FILTER_MAP = {
    position: 'people', offline: 'people',
    sos_start: 'safety', sos_end: 'safety',
    contact: 'me', self: 'me',
  };

  let filtered = $derived(activeFilter === 'all'
    ? $activityEvents
    : $activityEvents.filter(e => TYPE_FILTER_MAP[e.type] === activeFilter));

  // ── Day groupings — "Today", "Yesterday", then real dates ─────────────────
  function dayLabel(ts) {
    const d = new Date(ts);
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(today.getDate() - 1);
    const same = (a, b) => a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
    if (same(d, today)) return 'Today';
    if (same(d, yesterday)) return 'Yesterday';
    return d.toLocaleDateString([], { weekday: 'long', month: 'short', day: 'numeric' });
  }

  let grouped = $derived((() => {
    const groups = [];
    for (const ev of filtered) {
      const label = dayLabel(ev.ts);
      const last = groups[groups.length - 1];
      if (last && last.label === label) last.events.push(ev);
      else groups.push({ label, events: [ev] });
    }
    return groups;
  })());

  // ── Relative time — delegates to presence.js formatAge ──────────────────────
  function relTime(ts) {
    return formatAge(now - ts);
  }

  // ── Avatar ──────────────────────────────────────────────────────────────────
  // Colors come from getUserColor() (deterministic per-user hue) — never a
  // hardcoded palette.
  function initials(name) { return (name || '?')[0].toUpperCase(); }

  // ── AvatarRing ring type per event type ────────────────────────────────────
  function ringForType(type) {
    if (type === 'sos_start') return 'sos';
    if (type === 'position') return 'live';
    if (type === 'offline') return 'offline';
    return 'none';
  }

  // ── Connection state (badge-driven, never hardcoded) ─────────────────────
  // Derived from connectivityStore so the badge reflects real socket health.
  // Badge state changes are NOT announced (announce=false, default) — they must
  // not pollute the aria-live="polite" feed region.
  let connState = $derived(
    deriveConnState({
      initialized: $connectivityStore.socketConnected,
      online: $connectivityStore.socketConnected,
      issue: false,
      connecting: !$connectivityStore.socketConnected,
    })
  );

  // Events are captured globally by socketHandlers/activityRecorder.js and
  // persisted in activityLog — this page only renders and clears them, so the
  // feed is populated even when it was never opened.
  onMount(() => {
    clearHubBadge();
    clockInterval = setInterval(() => { now = Date.now(); }, 15000);
  });

  onDestroy(() => clearInterval(clockInterval));

  function clearFeed() { clearActivity(); }
</script>

<div class="activity-page page-enter aurora-ambient">
  <!-- Header -->
  <header class="act-header">
    <button class="icon-btn" onclick={() => push('/')} aria-label="Back to map">
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true">
        <polyline points="15 18 9 12 15 6"/>
      </svg>
    </button>

    <div class="act-title-group">
      <h1 class="act-title verdict-voice">Activity</h1>
      <!-- StatusBadge driven by real socket connection state.
           announce=false: badge transitions must NOT pollute the feed's aria-live region. -->
      <StatusBadge state={connState} announce={false} />
    </div>

    <button class="icon-btn icon-btn-clear" onclick={clearFeed} aria-label="Clear feed" disabled={$activityEvents.length === 0}>
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
        <polyline points="3 6 5 6 21 6"/>
        <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
        <path d="M10 11v6M14 11v6"/>
      </svg>
    </button>
  </header>

  <!-- Filter chips -->
  <div class="filter-row" role="group" aria-label="Filter activity">
    {#each FILTERS as f}
      <button
        class="filter-chip"
        class:active={activeFilter === f.key}
        onclick={() => activeFilter = f.key}
        aria-pressed={activeFilter === f.key}
        style="touch-action: manipulation;"
      >{f.label}</button>
    {/each}
  </div>

  <!-- Feed -->
  <main class="feed" aria-live="polite" aria-relevant="additions">
    {#if filtered.length === 0}
      <div class="empty-wrap" transition:fade={{ duration: 200 }}>
        <EmptyState
          title="Nothing here yet"
          body={connState === 'offline'
            ? 'Events resume when you reconnect'
            : 'Activity shows up here as your people move through their day.'}
        >
          {#snippet icon()}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
            </svg>
          {/snippet}
          {#snippet action()}
            <button class="cta-btn" onclick={() => push('/')}>Open the map</button>
          {/snippet}
        </EmptyState>
      </div>
    {:else}
      {#each grouped as group (group.label)}
        <h2 class="day-label">{group.label}</h2>
        {#each group.events as ev (ev.id)}
          {@const isDanger = ev.type === 'sos_start'}
          {@const isMuted = ev.type === 'offline'}
          <div
            class="feed-item"
            in:fly={$allowMotion ? { y: -18, duration: 240, easing: cubicOut } : { duration: 0 }}
          >
            <div
              class="feed-inner feed-{ev.type}"
              class:is-danger={isDanger}
              class:is-muted={isMuted}
              role="article"
            >
              <!-- Avatar / icon -->
              <div class="feed-left">
                {#if ev.userId}
                  <AvatarRing ring={ringForType(ev.type)} size={44}>
                    {#snippet children()}
                      <div
                        class="feed-avatar"
                        style="background:{getUserColorLight(ev.userId)}; color:{getUserColor(ev.userId)};"
                      >{initials(ev.userName)}</div>
                    {/snippet}
                  </AvatarRing>
                {:else}
                  <div class="feed-icon feed-icon-{ev.type}">
                    {#if ev.type === 'self'}
                      <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="8" r="4"/><path d="M6 20v-2a6 6 0 0 1 12 0v2"/></svg>
                    {:else if ev.type === 'contact'}
                      <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><line x1="19" y1="8" x2="19" y2="14"/><line x1="22" y1="11" x2="16" y2="11"/></svg>
                    {:else}
                      <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><circle cx="12" cy="16" r="0.5" fill="currentColor"/></svg>
                    {/if}
                  </div>
                {/if}
              </div>

              <!-- Body — the sentence first; a whisper of a label only when
                   the words alone can't carry the weight (SOS). -->
              <div class="feed-body">
                <p class="feed-msg">{ev.message}</p>
                <div class="feed-meta-row">
                  <span class="feed-tag feed-tag-{ev.type}">
                    {#if ev.type === 'position'}online
                    {:else if ev.type === 'offline'}offline
                    {:else if ev.type === 'sos_start'}SOS
                    {:else if ev.type === 'sos_end'}SOS resolved
                    {:else if ev.type === 'contact'}contact
                    {:else}note{/if}
                  </span>
                  <time class="feed-ts" datetime={new Date(ev.ts).toISOString()}>{relTime(ev.ts)}</time>
                </div>
              </div>
            </div>
          </div>
        {/each}
      {/each}
    {/if}
  </main>
</div>

<style>
  /* Hearth: what happened, told as sentences on warm paper. */
  .activity-page {
    display: flex;
    flex-direction: column;
    height: 100dvh;
    overflow: hidden;
    background: var(--surface-0);
    color: var(--text-primary);
    font-family: var(--font-sans);
  }

  /* Header — paper bar with a hairline, no glass */
  .act-header {
    position: sticky;
    top: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: calc(var(--safe-top) + var(--space-3)) var(--space-4) var(--space-3);
    background: var(--surface-0);
    border-bottom: 1px solid var(--border-default);
    flex-shrink: 0;
  }

  .icon-btn {
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--surface-1);
    border: 1px solid var(--border-default);
    border-radius: 50%;
    color: var(--text-primary);
    cursor: pointer;
    flex-shrink: 0;
    transition: background 150ms, transform 100ms;
    -webkit-tap-highlight-color: transparent;
  }
  .icon-btn:hover { background: var(--surface-hover); }
  .icon-btn:active { transform: scale(0.88); transition-duration: 60ms; }
  .icon-btn:disabled { opacity: 0.35; cursor: not-allowed; transform: none; }
  /* Clear feed is a secondary action — toned down from the primary back button */
  .icon-btn-clear { color: var(--text-secondary); }

  .act-title-group { flex: 1; display: flex; align-items: center; gap: var(--space-2); }

  /* Screen title — the serif register (verdict-voice supplies the italic) */
  .act-title {
    margin: 0;
    font-size: 22px;
    color: var(--text-primary);
  }

  /* Filter chips */
  .filter-row {
    display: flex;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    overflow-x: auto;
    scrollbar-width: none;
    flex-shrink: 0;
  }
  .filter-row::-webkit-scrollbar { display: none; }

  .filter-chip {
    min-height: 44px;
    padding: 6px 16px;
    border-radius: 999px;
    font-size: 14px;
    font-weight: 500;
    font-family: var(--font-sans);
    background: var(--surface-1);
    border: 1px solid var(--border-default);
    color: var(--text-secondary);
    cursor: pointer;
    white-space: nowrap;
    flex-shrink: 0;
    transition: background 150ms, color 150ms, border-color 150ms;
    -webkit-tap-highlight-color: transparent;
  }
  .filter-chip:hover { background: var(--surface-hover); color: var(--text-primary); }
  .filter-chip.active {
    background: var(--primary-500);
    border-color: transparent;
    color: var(--text-on-primary);
  }

  /* Feed — a single column of quiet rows, one hairline between days */
  .feed {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: var(--space-1) var(--space-4) calc(var(--space-6) + var(--safe-bottom));
    overflow-y: auto;
    max-width: 640px;
    width: 100%;
    margin: 0 auto;
  }

  .day-label {
    margin: var(--space-5) 0 var(--space-1);
    font-family: var(--font-sans);
    font-size: 13px;
    font-weight: 600;
    letter-spacing: 0.02em;
    color: var(--text-tertiary);
  }
  .day-label:first-child { margin-top: var(--space-3); }

  .feed-item { will-change: transform, opacity; }
  .feed-item + .feed-item .feed-inner { border-top: 1px solid var(--border-subtle); }

  /* Quiet row — no cards, no color-coded rails; whitespace does the work */
  .feed-inner {
    display: flex;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-1);
    border-radius: var(--radius-md, 10px);
    transition: opacity 200ms var(--ease-out, cubic-bezier(0.16,1,0.3,1));
  }

  /* SOS is the one event allowed to interrupt the quiet */
  .feed-inner.is-danger {
    background: color-mix(in oklch, var(--danger-500) 8%, transparent);
    padding: var(--space-3);
  }

  /* Offline events de-emphasised, not hidden */
  .feed-inner.is-muted { opacity: 0.6; }

  .feed-left { flex-shrink: 0; }

  /* feed-avatar fills the AvatarRing's content slot — AvatarRing owns the ring. */
  .feed-avatar {
    width: 44px; height: 44px;
    border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    font-family: var(--font-sans);
    font-weight: 600;
    font-size: 16px;
    /* Inline box-shadow removed — AvatarRing supplies the ring via --ring-color-* tokens */
    transition: transform 180ms var(--ease-out, cubic-bezier(0.16,1,0.3,1));
  }
  /* Avatar hover scale: desktop only (coarse pointer = touch device, skip) */
  @media (hover: hover) and (pointer: fine) {
    .feed-inner:hover .feed-avatar { transform: scale(1.08); }
  }

  .feed-icon {
    width: 44px; height: 44px;
    border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    background: var(--primary-100);
    color: var(--primary-700);
    transition: transform 180ms var(--ease-out, cubic-bezier(0.16,1,0.3,1));
  }
  @media (hover: hover) and (pointer: fine) {
    .feed-inner:hover .feed-icon { transform: scale(1.08); }
  }
  .feed-icon-self    { background: var(--surface-inset); color: var(--text-tertiary); }
  .feed-icon-contact { background: var(--primary-100); color: var(--primary-700); }

  .feed-body {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    justify-content: center;
  }

  .feed-meta-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  /* Whisper label — plain lowercase words, no log-style uppercase */
  .feed-tag {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-tertiary);
  }
  .feed-tag-sos_start { color: var(--danger-600); font-weight: 600; }
  .feed-tag-sos_end   { color: var(--success-600); }

  .feed-ts {
    font-size: 13px;
    color: var(--text-tertiary);
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }
  .feed-ts::before { content: '·'; margin-right: var(--space-2); color: var(--text-tertiary); }

  /* The sentence is the row — 16px ink */
  .feed-msg {
    margin: 0;
    font-size: 16px;
    line-height: 1.5;
    color: var(--text-primary);
  }
  /* An SOS sentence reads with weight, not volume */
  .feed-inner.is-danger .feed-msg {
    font-weight: 600;
    color: var(--danger-600);
  }

  /* Empty state wrapper (EmptyState primitive supplies the content) */
  .empty-wrap {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-10) var(--space-6);
  }

  /* CTA inside EmptyState action slot — true secondary button, not a tinted
     ghost that reads as disabled */
  .cta-btn {
    min-height: 44px;
    padding: 0 var(--space-5);
    border-radius: var(--radius-full, 9999px);
    font-size: var(--text-sm);
    font-weight: 600;
    font-family: var(--font-sans);
    background: var(--surface-1);
    border: 1.5px solid color-mix(in oklch, var(--primary-500) 45%, transparent);
    color: var(--primary-700);
    cursor: pointer;
    transition: background 150ms;
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }
  .cta-btn:hover { background: var(--primary-50); }

  @media (prefers-reduced-motion: reduce) {
    .feed-avatar, .feed-icon { transition: none; }
    .feed-inner:hover .feed-avatar,
    .feed-inner:hover .feed-icon { transform: none; }
  }
</style>
