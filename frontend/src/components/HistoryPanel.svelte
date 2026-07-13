<script>
  import { onMount, onDestroy } from 'svelte';
  import { socket } from '../lib/socket.js';
  import { otherUsers } from '../lib/stores/map.js';
  import { authUser } from '../lib/stores/auth.js';
  import {
    historyPoints,
    historyLoading,
    historyTarget,
    historyDate,
    historyVisible,
    historyPlayback,
  } from '../lib/stores/history.js';
  import { myContacts } from '../lib/stores/contacts.js';
  import PlaybackControls from './primitives/PlaybackControls.svelte';
  import StatCard        from './primitives/StatCard.svelte';
  import EmptyState      from './primitives/EmptyState.svelte';
  import SectionHeader   from './primitives/SectionHeader.svelte';

  /**
   * @typedef {Object} Props
   * @property {boolean} [embedded]
   */

  /** @type {Props} */
  let { embedded = false } = $props();

  // ── Local state ───────────────────────────────────────────────────────────────
  let selectedUserId = $state('');
  let dateInput      = $state($historyDate);
  let historyError   = $state(false);  // true after 15 s timeout fires
  let playTimer      = null;
  let historyTimeout;

  // ── Helpers ───────────────────────────────────────────────────────────────────
  function buildUserList(others, contacts, auth) {
    const list = [];
    if (auth) list.push({ id: auth.userId, name: 'Me' });
    for (const u of others.values()) {
      if (u.userId) list.push({ id: u.userId, name: u.displayName || u.userId.slice(0, 6) });
    }
    return list;
  }

  function formatTime(ts) {
    if (!ts) return null;
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function calcDistance(pts) {
    if (!pts || pts.length < 2) return null;
    let d = 0;
    for (let i = 1; i < pts.length; i++) {
      const R    = 6371;
      const dLat = (pts[i].lat - pts[i - 1].lat) * Math.PI / 180;
      const dLng = (pts[i].lng - pts[i - 1].lng) * Math.PI / 180;
      const a    =
        Math.sin(dLat / 2) ** 2 +
        Math.cos(pts[i - 1].lat * Math.PI / 180) *
        Math.cos(pts[i].lat     * Math.PI / 180) *
        Math.sin(dLng / 2) ** 2;
      d += R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    }
    return Math.round(d * 10) / 10;
  }

  // ── Socket ────────────────────────────────────────────────────────────────────
  function onHistoryData(res) {
    clearTimeout(historyTimeout);
    historyLoading.set(false);
    historyError = false;
    if (res && res.ok) {
      historyPoints.set(res.points || []);
      historyVisible.set(true);
      historyPlayback.set({ playing: false, index: 0, speed: 1 });
    } else {
      historyPoints.set([]);
      historyVisible.set(true);
    }
  }

  onMount(()    => { socket.on('historyData', onHistoryData); });
  onDestroy(()  => {
    stopPlayback();
    clearTimeout(historyTimeout);
    socket.off('historyData', onHistoryData);
  });

  // ── Data fetch ────────────────────────────────────────────────────────────────
  function fetchHistory() {
    if (!selectedUserId || $historyLoading) return;
    const d     = new Date(dateInput);
    const start = d.getTime();
    const end   = start + 24 * 60 * 60 * 1000;
    historyLoading.set(true);
    historyError = false;
    historyVisible.set(false);
    historyTarget.set(selectedUserId);
    historyDate.set(dateInput);
    socket.emit('getHistory', { userId: selectedUserId, start, end });
    clearTimeout(historyTimeout);
    historyTimeout = setTimeout(() => {
      historyLoading.set(false);
      historyError = true;
    }, 15000);
  }

  function clearHistory() {
    historyVisible.set(false);
    historyPoints.set([]);
    historyPlayback.set({ playing: false, index: 0, speed: 1 });
    historyError = false;
    stopPlayback();
  }

  // ── Playback engine ───────────────────────────────────────────────────────────
  // Tick interval (ms) — hard constant; JS timing does not belong in CSS tokens
  const PLAYBACK_TICK_MS = 200;

  function stopPlayback() {
    if (playTimer) { clearInterval(playTimer); playTimer = null; }
  }

  function startPlayback() {
    stopPlayback();
    playTimer = setInterval(() => {
      historyPlayback.update(p => {
        const pts = $historyPoints;
        if (p.index >= pts.length - 1) {
          stopPlayback();
          return { ...p, playing: false, index: pts.length - 1 };
        }
        const newSpeed = p.speed ?? 1;
        return { ...p, index: Math.min(p.index + newSpeed, pts.length - 1) };
      });
    }, PLAYBACK_TICK_MS);
  }

  // ── PlaybackControls callbacks (CONTRACTS.md §11 — names frozen) ──────────────
  function handlePlay() {
    historyPlayback.update(p => ({ ...p, playing: true }));
    startPlayback();
  }

  function handlePause() {
    stopPlayback();
    historyPlayback.update(p => ({ ...p, playing: false }));
  }

  function handleScrub({ progress }) {
    const pts = $historyPoints;
    if (!pts.length) return;
    const newIndex = Math.round(progress * (pts.length - 1));
    historyPlayback.update(p => ({ ...p, index: newIndex }));
  }

  function handleSpeedChange({ speed }) {
    historyPlayback.update(p => ({ ...p, speed }));
    // Re-start if currently playing so the new speed takes effect immediately
    if ($historyPlayback.playing) {
      stopPlayback();
      startPlayback();
    }
  }

  // ── Derived display values ────────────────────────────────────────────────────
  let visibleUsers = $derived(buildUserList($otherUsers, $myContacts, $authUser));
  let distKm       = $derived(calcDistance($historyPoints));

  // progress 0–1 for PlaybackControls
  let playbackProgress = $derived(
    $historyPoints.length > 1
      ? $historyPlayback.index / ($historyPoints.length - 1)
      : 0
  );

  let currentTs = $derived($historyPoints[$historyPlayback.index]?.t ?? null);
  let endTs     = $derived($historyPoints[$historyPoints.length - 1]?.t ?? null);
</script>

{#if embedded}
  <div class="panel-body history-panel">

    <!-- ── Section header ────────────────────────────────────────────── -->
    <div class="section-header-wrap">
      <SectionHeader title="Route History" level={3} />
    </div>

    <!-- ── Controls ──────────────────────────────────────────────────── -->
    <div class="history-controls">
      <label class="field-label" for="history-user">Person</label>
      <select
        id="history-user"
        bind:value={selectedUserId}
        class="field-input"
      >
        <option value="">Choose someone</option>
        {#each visibleUsers as u}
          <option value={u.id}>{u.name}</option>
        {/each}
      </select>

      <label class="field-label" for="history-date">Date</label>
      <input
        id="history-date"
        type="date"
        bind:value={dateInput}
        class="field-input"
        aria-label="Select date for route history"
      />

      <div class="history-actions">
        <button
          class="btn btn-primary btn-sm"
          onclick={fetchHistory}
          disabled={!selectedUserId || $historyLoading}
          aria-busy={$historyLoading}
        >
          {$historyLoading ? 'Loading…' : 'Show Route'}
        </button>
        {#if $historyVisible || historyError}
          <button class="btn btn-secondary btn-sm" onclick={clearHistory}>Clear</button>
        {/if}
      </div>
    </div>

    <!-- ── Loading skeleton ───────────────────────────────────────────── -->
    {#if $historyLoading}
      <div class="skeleton-block" role="status" aria-label="Loading route history" aria-busy="true">
        <div class="skel-row skel-wide"></div>
        <div class="skel-row skel-mid"></div>
        <div class="skel-row skel-short"></div>
      </div>

    {:else if historyError}
      <!-- ── Error state with retry ───────────────────────────────────── -->
      <EmptyState
        title="Route not loaded"
        body="The server took too long to respond. Check your connection and try again."
        tone="neutral"
      >
        {#snippet icon()}
          <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
        {/snippet}
        {#snippet action()}
          <button class="btn btn-primary btn-sm" onclick={fetchHistory}>
            Try again
          </button>
        {/snippet}
      </EmptyState>

    {:else if $historyVisible && $historyPoints.length > 0}
      <!-- ── Stats ──────────────────────────────────────────────────── -->
      <div class="history-stats" role="region" aria-label="Route statistics">
        <StatCard label="Points"   value={$historyPoints.length} tint="primary" />
        {#if distKm !== null}
          <StatCard label="Distance" value={distKm} unit="km" tint="primary" />
        {/if}
        {#if formatTime($historyPoints[0]?.t)}
          <StatCard label="Start"   value={formatTime($historyPoints[0].t)} tint="neutral" />
        {/if}
        {#if formatTime($historyPoints[$historyPoints.length - 1]?.t)}
          <StatCard label="End"     value={formatTime($historyPoints[$historyPoints.length - 1].t)} tint="neutral" />
        {/if}
      </div>

      <!-- ── Playback ───────────────────────────────────────────────── -->
      <div class="playback-section" role="region" aria-label="Playback controls">
        <div class="playback-section-label" aria-hidden="true">Playback</div>
        <PlaybackControls
          playing={$historyPlayback.playing}
          progress={playbackProgress}
          speed={$historyPlayback.speed ?? 1}
          timestamps={{ current: currentTs, end: endTs }}
          onplay={handlePlay}
          onpause={handlePause}
          onscrub={handleScrub}
          onspeedchange={handleSpeedChange}
        />
      </div>

    {:else if $historyVisible && $historyPoints.length === 0}
      <!-- ── Empty state with CTA ──────────────────────────────────── -->
      <EmptyState
        title="No route data for this day"
        body="Location history is stored for 30 days. Try a different date or person."
        tone="neutral"
      >
        {#snippet icon()}
          <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
            <path d="M3 12h18M12 3l9 9-9 9-9-9 9-9z"/>
          </svg>
        {/snippet}
        {#snippet action()}
          <button class="btn btn-secondary btn-sm" onclick={() => { historyVisible.set(false); }}>
            Try another date
          </button>
        {/snippet}
      </EmptyState>

    {:else if !selectedUserId}
      <!-- ── Onboarding hint ────────────────────────────────────────── -->
      <div class="onboard-hint" role="note">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
          <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
        </svg>
        <p>Pick a person and a date to replay their route on the map.</p>
      </div>
    {/if}

  </div>
{:else}
  <div class="panel-shell panel-left panel-base">
    <div class="panel-header">
      <h3>Route History</h3>
      <button class="panel-close" onclick={() => {}} aria-label="Close route history">&times;</button>
    </div>
    <div class="panel-body">
      <p>View route history from the sidebar.</p>
    </div>
  </div>
{/if}

<style>
  .history-panel { padding: 0; }

  /* ── Section header ────────────────────────────────────────────────────── */
  .section-header-wrap {
    padding: var(--space-4) var(--space-4) var(--space-2);
  }

  /* ── Controls ──────────────────────────────────────────────────────────── */
  .history-controls {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: 0 var(--space-4) var(--space-3);
  }

  .field-label {
    display: block;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: var(--space-1);
  }

  .field-input {
    width: 100%;
    /* 16px minimum — prevents iOS Safari auto-zoom on focus */
    font-size: max(16px, var(--text-base));
    padding: var(--space-2) var(--space-2-5);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
    background: var(--surface-3);
    color: var(--text-primary);
    box-sizing: border-box;
    font-family: var(--font-sans);
    touch-action: manipulation;
    transition:
      border-color var(--duration-fast) var(--ease-out),
      box-shadow   var(--duration-fast) var(--ease-out);
  }

  .field-input:focus {
    outline: none;
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px color-mix(in oklch, var(--primary-500) 18%, transparent);
  }

  .field-input option { background: var(--surface-2); color: var(--text-primary); }

  .history-actions {
    display: flex;
    gap: var(--space-2);
    padding-top: var(--space-1);
  }

  /* ── Loading skeleton ─────────────────────────────────────────────────── */
  .skeleton-block {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
  }

  .skel-row {
    height: var(--space-4);
    border-radius: var(--radius-sm);
    background: var(--skeleton-base, rgba(255,255,255,0.05));
    animation: skel-pulse 1.6s ease-in-out infinite;
  }

  .skel-wide  { width: 100%; }
  .skel-mid   { width: 70%; }
  .skel-short { width: 45%; }

  @keyframes skel-pulse {
    0%, 100% { opacity: 0.5; }
    50%       { opacity: 1; }
  }

  /* ── Stats ─────────────────────────────────────────────────────────────── */
  .history-stats {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    border-top: 1px solid var(--border-subtle);
  }

  /* ── Playback ─────────────────────────────────────────────────────────── */
  .playback-section {
    border-top: 1px solid var(--border-subtle);
    padding-top: var(--space-3);
  }

  .playback-section-label {
    padding: 0 var(--space-4) var(--space-1);
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  /* ── Onboarding hint ─────────────────────────────────────────────────── */
  .onboard-hint {
    display: flex;
    align-items: flex-start;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-4);
    margin: 0 var(--space-4);
    background: var(--surface-inset);
    border: 1px dashed var(--border-subtle);
    border-radius: var(--radius-lg);
    color: var(--text-tertiary);
  }

  .onboard-hint svg { flex-shrink: 0; margin-top: 2px; }

  .onboard-hint p {
    font-size: var(--text-xs);
    line-height: var(--leading-normal);
    margin: 0;
  }

  /* ── Reduced motion ──────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .skel-row { animation: none; }
  }
</style>
