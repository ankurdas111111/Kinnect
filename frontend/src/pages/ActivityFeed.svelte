<script>
  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { get } from 'svelte/store';
  import { authUser } from '../lib/stores/auth.js';
  import { otherUsers } from '../lib/stores/map.js';
  import { socket } from '../lib/socket.js';
  import { clearHubBadge } from '../lib/stores/hubBadge.js';

  $: if (!$authUser) push('/login');

  // ── Event log ──────────────────────────────────────────────────────────────
  let events = [];
  let nextId = 1;
  let activeFilter = 'all';
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

  $: filtered = activeFilter === 'all'
    ? events
    : events.filter(e => TYPE_FILTER_MAP[e.type] === activeFilter);

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

  // ── Avatar color ────────────────────────────────────────────────────────────
  const PALETTE = ['#818cf8','#34d399','#fbbf24','#f87171','#a78bfa','#22d3ee','#fb7185'];
  const PALETTE_BG = [
    'rgba(99,102,241,0.18)', 'rgba(16,185,129,0.18)', 'rgba(245,158,11,0.18)',
    'rgba(239,68,68,0.16)',  'rgba(139,92,246,0.18)', 'rgba(6,182,212,0.16)',
    'rgba(251,113,133,0.16)',
  ];
  function avatarPal(name) {
    const i = (name || '?').charCodeAt(0) % PALETTE.length;
    return { color: PALETTE[i], bg: PALETTE_BG[i] };
  }
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
    <button class="icon-btn" on:click={() => push('/')} aria-label="Back to map">
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

    <button class="icon-btn" on:click={clearFeed} aria-label="Clear feed" disabled={events.length === 0}>
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
        on:click={() => activeFilter = f.key}
        aria-pressed={activeFilter === f.key}
      >{f.label}</button>
    {/each}
  </div>

  <!-- Feed -->
  <main class="feed" aria-live="polite" aria-relevant="additions">
    {#if filtered.length === 0}
      <div class="empty-state" transition:fade={{ duration: 200 }}>
        <div class="empty-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
            <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
          </svg>
        </div>
        <p class="empty-title">Nothing here yet</p>
        <p class="empty-desc">Activity appears in real time as events happen on your network</p>
      </div>
    {:else}
      {#each filtered as ev (ev.id)}
        {@const pal = ev.userId ? avatarPal(ev.userName) : null}
        <div
          class="feed-card feed-{ev.type}"
          in:fly={{ y: -18, duration: 240, easing: cubicOut }}
          role="article"
        >
          <!-- Avatar / icon -->
          <div class="feed-left">
            {#if ev.userId}
              <div class="feed-avatar" style="background:{pal.bg}; color:{pal.color};">{initials(ev.userName)}</div>
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
    gap: 12px;
    padding: calc(env(safe-area-inset-top, 0px) + 10px) 16px 10px;
    background: rgba(5, 5, 18, 0.92);
    backdrop-filter: blur(24px) saturate(1.6);
    -webkit-backdrop-filter: blur(24px) saturate(1.6);
    border-bottom: 1px solid rgba(255,255,255,0.06);
    flex-shrink: 0;
  }

  .icon-btn {
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255,255,255,0.06);
    border: 1px solid rgba(255,255,255,0.09);
    border-radius: 50%;
    color: var(--text-primary);
    cursor: pointer;
    flex-shrink: 0;
    transition: background 150ms, transform 100ms;
    -webkit-tap-highlight-color: transparent;
  }
  .icon-btn:hover { background: rgba(255,255,255,0.11); }
  .icon-btn:active { transform: scale(0.88); transition-duration: 60ms; }
  .icon-btn:disabled { opacity: 0.35; cursor: not-allowed; transform: none; }

  .act-title-group { flex: 1; display: flex; align-items: center; gap: 8px; }

  .act-title {
    margin: 0;
    font-family: var(--font-display);
    font-size: 20px;
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
    color: var(--success-400, #34d399);
    background: rgba(16,185,129,0.12);
    border: 1px solid rgba(16,185,129,0.22);
    border-radius: 999px;
    padding: 2px 8px 2px 6px;
  }
  .live-dot {
    width: 6px; height: 6px;
    border-radius: 50%;
    background: var(--success-400, #34d399);
    animation: live-pulse 2s ease-in-out infinite;
  }
  @keyframes live-pulse { 0%,100% { opacity:1; } 50% { opacity:0.45; } }

  /* Filter chips */
  .filter-row {
    display: flex;
    gap: 8px;
    padding: 12px 16px;
    overflow-x: auto;
    scrollbar-width: none;
    flex-shrink: 0;
  }
  .filter-row::-webkit-scrollbar { display: none; }

  .filter-chip {
    padding: 6px 14px;
    border-radius: 999px;
    font-size: 13px;
    font-weight: 600;
    font-family: var(--font-display);
    background: rgba(255,255,255,0.05);
    border: 1px solid rgba(255,255,255,0.09);
    color: var(--text-secondary);
    cursor: pointer;
    white-space: nowrap;
    flex-shrink: 0;
    transition: all 150ms;
    -webkit-tap-highlight-color: transparent;
  }
  .filter-chip:hover { background: rgba(255,255,255,0.09); color: var(--text-primary); }
  .filter-chip.active {
    background: rgba(99,102,241,0.18);
    border-color: rgba(99,102,241,0.38);
    color: var(--primary-300, #a5b4fc);
    box-shadow: 0 0 14px rgba(99,102,241,0.18);
  }

  /* Feed */
  .feed {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 4px 16px calc(24px + env(safe-area-inset-bottom, 0px));
    overflow-y: auto;
  }

  /* Feed card */
  .feed-card {
    display: flex;
    gap: 12px;
    padding: 13px 14px;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.07);
    border-left: 3px solid var(--primary-500, #6366f1);
    border-radius: 14px;
  }
  .feed-offline  { border-left-color: rgba(255,255,255,0.18); }
  .feed-sos_start { border-left-color: var(--danger-500, #ef4444); background: rgba(239,68,68,0.05); }
  .feed-sos_end   { border-left-color: var(--success-500, #10b981); }
  .feed-self      { border-left-color: rgba(255,255,255,0.14); }

  .feed-left { flex-shrink: 0; }

  .feed-avatar {
    width: 40px; height: 40px;
    border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 16px;
  }

  .feed-icon {
    width: 40px; height: 40px;
    border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    background: rgba(99,102,241,0.13);
    color: var(--primary-400, #818cf8);
  }
  .feed-icon-self    { background: rgba(255,255,255,0.06); color: var(--text-tertiary); }
  .feed-icon-contact { background: rgba(99,102,241,0.13); color: var(--primary-400, #818cf8); }

  .feed-body {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .feed-meta-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .feed-tag {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--primary-400, #818cf8);
  }
  .feed-tag-offline   { color: var(--text-tertiary); }
  .feed-tag-sos_start { color: var(--danger-400, #f87171); }
  .feed-tag-sos_end   { color: var(--success-400, #34d399); }
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

  /* Empty state */
  .empty-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    padding: 60px 24px;
    text-align: center;
  }
  .empty-icon {
    width: 64px; height: 64px;
    border-radius: 20px;
    background: rgba(99,102,241,0.10);
    border: 1px solid rgba(99,102,241,0.18);
    display: flex; align-items: center; justify-content: center;
    color: var(--primary-400, #818cf8);
  }
  .empty-title {
    font-family: var(--font-display);
    font-size: 18px;
    font-weight: 700;
    color: var(--text-primary);
    margin: 0;
  }
  .empty-desc {
    font-size: 14px;
    color: var(--text-tertiary);
    max-width: 220px;
    line-height: 1.55;
    margin: 0;
  }

  @media (prefers-reduced-motion: reduce) {
    .live-dot { animation: none; }
  }
</style>
