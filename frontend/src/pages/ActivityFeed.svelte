<script>
  import { run } from 'svelte/legacy';

  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { get } from 'svelte/store';
  import { authUser } from '../lib/stores/auth.js';
  import { otherUsers } from '../lib/stores/map.js';
  import { socket } from '../lib/socket.js';
  import { clearHubBadge } from '../lib/stores/hubBadge.js';
  import Card from '../components/primitives/Card.svelte';
  import EmptyState from '../components/primitives/EmptyState.svelte';
  import { getUserColor, getUserColorLight } from '../lib/getUserColor.js';

  run(() => {
    if (!$authUser) push('/login');
  });

  // ── Event log ──────────────────────────────────────────────────────────────
  let events = $state([]);
  let nextId = 1;
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
    ? events
    : events.filter(e => TYPE_FILTER_MAP[e.type] === activeFilter));

  function addEvent(type, userId, userName, message, severity = 'normal') {
    events = [
      { id: nextId++, type, userId, userName, message, severity, ts: Date.now() },
      ...events,
    ].slice(0, 120);
  }

  // ── Relative time ───────────────────────────────────────────────────────────
  function relTime(ts) {
    const diff = now - ts;
    if (diff < 10000) return 'just now';
    if (diff < 60000) return `${Math.floor(diff / 1000)}s ago`;
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    const d = new Date(ts);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  // ── Avatar ──────────────────────────────────────────────────────────────────
  // Colors come from getUserColor() (deterministic per-user hue) — never a
  // hardcoded palette.
  function initials(name) { return (name || '?')[0].toUpperCase(); }

  // ── Socket listeners ────────────────────────────────────────────────────────
  // Track which userIds currently have an active SOS so we only log start/end once
  const sosActiveSet = new Set();

  function makeHandlers() {
    return {
      userConnected: (d) => addEvent('position', d.userId, d.displayName, `${d.displayName || 'Someone'} came online`),
      userOffline:   (d) => addEvent('offline',  d.userId, d.displayName, `${d.displayName || 'Someone'} went offline`),
      // userDisconnect sends a raw socketId string — look up name from otherUsers
      userDisconnect: (socketId) => {
        const uid = typeof socketId === 'string' ? socketId : socketId?.socketId;
        const user = get(otherUsers).get(uid);
        const name = user?.displayName || 'Someone';
        addEvent('offline', user?.userId || null, name, `${name} left`);
      },
      // Backend emits sosUpdate for both SOS start AND end; diff with sosActiveSet
      sosUpdate: (d) => {
        if (!d || !d.userId) return;
        const sos = d.sos || {};
        const name = d.displayName || 'Someone';
        if (sos.active && !sosActiveSet.has(d.userId)) {
          sosActiveSet.add(d.userId);
          addEvent('sos_start', d.userId, name, `SOS — ${sos.reason || 'Emergency'}`, 'danger');
        } else if (!sos.active && sosActiveSet.has(d.userId)) {
          sosActiveSet.delete(d.userId);
          addEvent('sos_end', d.userId, name, `${name} cancelled SOS`);
        }
      },
      contactAdded:  ()  => addEvent('contact', null, null, 'New contact added to your network'),
      roomJoined:    (d) => addEvent('self', null, null, `Joined room: ${d.name || d.roomId || 'Unknown'}`),
    };
  }

  let handlers = {};

  onMount(() => {
    clearHubBadge();
    clockInterval = setInterval(() => { now = Date.now(); }, 15000);
    handlers = makeHandlers();
    for (const [event, fn] of Object.entries(handlers)) socket.on(event, fn);
    addEvent('self', null, null, 'Activity feed started');
  });

  onDestroy(() => {
    clearInterval(clockInterval);
    for (const [event, fn] of Object.entries(handlers)) socket.off(event, fn);
  });

  function clearFeed() { events = []; }
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
      <h1 class="act-title">Activity</h1>
      <span class="live-badge">
        <span class="live-dot" aria-hidden="true"></span>
        Live
      </span>
    </div>

    <button class="icon-btn" onclick={clearFeed} aria-label="Clear feed" disabled={events.length === 0}>
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
      >{f.label}</button>
    {/each}
  </div>

  <!-- Feed -->
  <main class="feed" aria-live="polite" aria-relevant="additions">
    {#if filtered.length === 0}
      <div class="empty-wrap" transition:fade={{ duration: 200 }}>
        <EmptyState
          title="Nothing here yet"
          body="Activity appears in real time as events happen on your network"
        >
          {#snippet icon()}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
            </svg>
          {/snippet}
        </EmptyState>
      </div>
    {:else}
      {#each filtered as ev (ev.id)}
        {@const isDanger = ev.type === 'sos_start'}
        {@const isMuted = ev.type === 'offline'}
        <div class="feed-item" in:fly={{ y: -18, duration: 240, easing: cubicOut }}>
          <Card
            variant="glass"
            padding="none"
            hover={false}
            noise={false}
            glow={isDanger ? 'danger' : null}
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
                  <div
                    class="feed-avatar"
                    style="background:{getUserColorLight(ev.userId)}; color:{getUserColor(ev.userId)}; box-shadow:0 0 10px {getUserColorLight(ev.userId)};"
                  >{initials(ev.userName)}</div>
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

              <!-- Body -->
              <div class="feed-body">
                <div class="feed-meta-row">
                  <span class="feed-tag feed-tag-{ev.type}">
                    {#if ev.type === 'position'}Online
                    {:else if ev.type === 'offline'}Offline
                    {:else if ev.type === 'sos_start'}SOS Alert
                    {:else if ev.type === 'sos_end'}SOS Resolved
                    {:else if ev.type === 'contact'}Contact
                    {:else}System{/if}
                  </span>
                  <time class="feed-ts" datetime={new Date(ev.ts).toISOString()}>{relTime(ev.ts)}</time>
                </div>
                <p class="feed-msg">{ev.message}</p>
              </div>
            </div>
          </Card>
        </div>
      {/each}
    {/if}
  </main>
</div>

<style>
  .activity-page {
    display: flex;
    flex-direction: column;
    height: 100dvh;
    overflow: hidden;
    background: var(--bg-base, #0a0a14);
    color: var(--text-primary);
    font-family: var(--font-sans);
  }

  /* Header */
  .act-header {
    position: sticky;
    top: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: calc(env(safe-area-inset-top, 0px) + var(--space-3)) var(--space-4) var(--space-3);
    background: var(--glass-bg, rgba(5, 5, 18, 0.92));
    backdrop-filter: blur(24px) saturate(1.6);
    -webkit-backdrop-filter: blur(24px) saturate(1.6);
    border-bottom: 1px solid var(--border-default);
    flex-shrink: 0;
  }

  .icon-btn {
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--surface-inset, rgba(255,255,255,0.06));
    border: 1px solid var(--border-default);
    border-radius: 50%;
    color: var(--text-primary);
    cursor: pointer;
    flex-shrink: 0;
    transition: background 150ms, transform 100ms;
    -webkit-tap-highlight-color: transparent;
  }
  .icon-btn:hover { background: var(--surface-hover, rgba(255,255,255,0.11)); }
  .icon-btn:active { transform: scale(0.88); transition-duration: 60ms; }
  .icon-btn:disabled { opacity: 0.35; cursor: not-allowed; transform: none; }

  .act-title-group { flex: 1; display: flex; align-items: center; gap: var(--space-2); }

  .act-title {
    margin: 0;
    font-family: var(--font-display);
    font-size: var(--text-xl, 20px);
    font-weight: 700;
    letter-spacing: -0.02em;
  }

  .live-badge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--success-500);
    background: var(--success-500-12);
    border: 1px solid var(--success-500-20);
    border-radius: 999px;
    padding: 2px 8px 2px 6px;
  }
  .live-dot {
    width: 6px; height: 6px;
    border-radius: 50%;
    background: var(--success-500);
    animation: live-pulse 2s ease-in-out infinite;
  }
  @keyframes live-pulse { 0%,100% { opacity:1; } 50% { opacity:0.45; } }

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
    font-size: 13px;
    font-weight: 600;
    font-family: var(--font-display);
    background: var(--surface-inset, rgba(255,255,255,0.05));
    border: 1px solid var(--border-default);
    color: var(--text-secondary);
    cursor: pointer;
    white-space: nowrap;
    flex-shrink: 0;
    transition: background 150ms, color 150ms, border-color 150ms, box-shadow 150ms;
    -webkit-tap-highlight-color: transparent;
  }
  .filter-chip:hover { background: var(--surface-hover, rgba(255,255,255,0.09)); color: var(--text-primary); }
  .filter-chip.active {
    background: var(--primary-500-20);
    border-color: var(--primary-500-20);
    color: var(--primary-300, var(--primary-400));
    box-shadow: var(--glow-primary-sm);
  }

  /* Feed */
  .feed {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-1) var(--space-4) calc(var(--space-6) + env(safe-area-inset-bottom, 0px));
    overflow-y: auto;
  }

  .feed-item { will-change: transform, opacity; }

  /* Feed card inner (Card primitive supplies the glass surface + edge line) */
  .feed-inner {
    display: flex;
    gap: var(--space-3);
    padding: var(--space-3);
    border-left: 3px solid var(--primary-500);
    transition: opacity 200ms var(--ease-out, cubic-bezier(0.16,1,0.3,1));
  }
  .feed-offline   .feed-avatar,
  .feed-offline { border-left-color: var(--border-strong, rgba(255,255,255,0.18)); }
  .feed-sos_start { border-left-color: var(--danger-500); }
  .feed-sos_end   { border-left-color: var(--success-500); }
  .feed-self      { border-left-color: var(--border-strong, rgba(255,255,255,0.14)); }
  .feed-contact   { border-left-color: var(--primary-500); }

  /* Danger events carry extra visual weight: tint + roomier padding */
  .feed-inner.is-danger {
    background: var(--danger-500-12);
    padding: var(--space-4) var(--space-3);
  }

  /* Offline events de-emphasised, not hidden */
  .feed-inner.is-muted { opacity: 0.6; }

  .feed-left { flex-shrink: 0; }

  .feed-avatar {
    width: 44px; height: 44px;
    border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 16px;
    transition: transform 180ms var(--ease-out, cubic-bezier(0.16,1,0.3,1));
  }
  .feed-inner:hover .feed-avatar { transform: scale(1.08); }

  .feed-icon {
    width: 44px; height: 44px;
    border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    background: var(--primary-500-12);
    color: var(--primary-400);
    transition: transform 180ms var(--ease-out, cubic-bezier(0.16,1,0.3,1));
  }
  .feed-inner:hover .feed-icon { transform: scale(1.08); }
  .feed-icon-self    { background: var(--surface-inset, rgba(255,255,255,0.06)); color: var(--text-tertiary); }
  .feed-icon-contact { background: var(--primary-500-12); color: var(--primary-400); }

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
    justify-content: space-between;
    gap: var(--space-2);
  }

  .feed-tag {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--primary-400);
  }
  .feed-tag-offline   { color: var(--text-tertiary); }
  .feed-tag-sos_start { color: var(--danger-500); }
  .feed-tag-sos_end   { color: var(--success-500); }
  .feed-tag-self      { color: var(--text-tertiary); }

  .feed-ts {
    font-size: 11px;
    color: var(--text-tertiary);
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }

  .feed-msg {
    margin: 0;
    font-size: 14px;
    line-height: 1.45;
    color: var(--text-secondary);
  }
  /* Danger message reads louder */
  .feed-inner.is-danger .feed-msg {
    font-size: 15px;
    font-weight: 600;
    color: var(--text-primary);
  }

  /* Empty state wrapper (EmptyState primitive supplies the content) */
  .empty-wrap {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-10) var(--space-6);
  }

  @media (prefers-reduced-motion: reduce) {
    .live-dot { animation: none; }
    .feed-avatar, .feed-icon { transition: none; }
    .feed-inner:hover .feed-avatar,
    .feed-inner:hover .feed-icon { transform: none; }
  }
</style>
