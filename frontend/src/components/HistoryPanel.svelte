<script>
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import { socket } from '../lib/socket.js';
  import { otherUsers } from '../lib/stores/map.js';
  import { authUser } from '../lib/stores/auth.js';
  import { historyPoints, historyLoading, historyTarget, historyDate, historyVisible, historyPlayback } from '../lib/stores/history.js';
  import { myContacts } from '../lib/stores/contacts.js';

  /**
   * @typedef {Object} Props
   * @property {boolean} [embedded]
   */

  /** @type {Props} */
  let { embedded = false } = $props();

  const dispatch = createEventDispatcher();

  let selectedUserId = $state('');
  let dateInput = $state($historyDate);
  let playTimer = null;
  let historyTimeout;


  function buildUserList(others, contacts, auth) {
    const list = [];
    if (auth) list.push({ id: auth.userId, name: 'Me' });
    for (const u of others.values()) {
      if (u.userId) list.push({ id: u.userId, name: u.displayName || u.userId.slice(0, 6) });
    }
    return list;
  }

  function onHistoryData(res) {
    clearTimeout(historyTimeout);
    historyLoading.set(false);
    if (res && res.ok) {
      historyPoints.set(res.points || []);
      historyVisible.set(true);
      historyPlayback.set({ playing: false, index: 0, speed: 1 });
    } else {
      historyPoints.set([]);
    }
  }

  onMount(() => { socket.on('historyData', onHistoryData); });
  onDestroy(() => { stopPlayback(); clearTimeout(historyTimeout); socket.off('historyData', onHistoryData); });

  function fetchHistory() {
    if (!selectedUserId || $historyLoading) return;
    const d = new Date(dateInput);
    const start = d.getTime();
    const end = start + 24 * 60 * 60 * 1000;
    historyLoading.set(true);
    historyTarget.set(selectedUserId);
    historyDate.set(dateInput);
    socket.emit('getHistory', { userId: selectedUserId, start, end });
    clearTimeout(historyTimeout);
    historyTimeout = setTimeout(() => { historyLoading.set(false); historyPoints.set([]); }, 15000);
  }

  function clearHistory() {
    historyVisible.set(false);
    historyPoints.set([]);
    historyPlayback.set({ playing: false, index: 0, speed: 1 });
    stopPlayback();
  }

  function togglePlayback() {
    historyPlayback.update(p => {
      if (p.playing) {
        stopPlayback();
        return { ...p, playing: false };
      }
      startPlayback();
      return { ...p, playing: true };
    });
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
        return { ...p, index: p.index + 1 };
      });
    }, 200);
  }

  function stopPlayback() {
    if (playTimer) { clearInterval(playTimer); playTimer = null; }
  }

  function onSlider(e) {
    const val = parseInt(e.target.value, 10);
    historyPlayback.update(p => ({ ...p, index: val }));
  }

  function formatTime(ts) {
    if (!ts) return '';
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  // Compute total distance from history points
  function calcDistance(pts) {
    if (!pts || pts.length < 2) return null;
    let d = 0;
    for (let i = 1; i < pts.length; i++) {
      const R = 6371;
      const dLat = (pts[i].lat - pts[i-1].lat) * Math.PI / 180;
      const dLng = (pts[i].lng - pts[i-1].lng) * Math.PI / 180;
      const a = Math.sin(dLat/2)**2 + Math.cos(pts[i-1].lat*Math.PI/180) * Math.cos(pts[i].lat*Math.PI/180) * Math.sin(dLng/2)**2;
      d += R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1-a));
    }
    return Math.round(d * 10) / 10;
  }

  let visibleUsers = $derived(buildUserList($otherUsers, $myContacts, $authUser));
  let distKm = $derived(calcDistance($historyPoints));
</script>

{#if embedded}
  <div class="panel-body history-panel">

    <!-- ── Section header ────────────────────────────────────────────── -->
    <div class="section-header">
      <span class="section-label">Route History</span>
    </div>

    <!-- ── Controls ──────────────────────────────────────────────────── -->
    <div class="history-controls">
      <label class="field-label" for="history-user">Person</label>
      <select id="history-user" bind:value={selectedUserId} class="field-input">
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
        {#if $historyVisible}
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

    {:else if $historyVisible && $historyPoints.length > 0}
      <!-- ── Stats ──────────────────────────────────────────────────── -->
      <div class="history-stats" role="region" aria-label="Route statistics">
        <div class="stat-chip">
          <span class="stat-val">{$historyPoints.length}</span>
          <span class="stat-lbl">Points</span>
        </div>
        {#if distKm !== null}
          <div class="stat-chip">
            <span class="stat-val">{distKm} km</span>
            <span class="stat-lbl">Distance</span>
          </div>
        {/if}
        <div class="stat-chip">
          <span class="stat-val">{formatTime($historyPoints[0]?.t)}</span>
          <span class="stat-lbl">Start</span>
        </div>
        <div class="stat-chip">
          <span class="stat-val">{formatTime($historyPoints[$historyPoints.length - 1]?.t)}</span>
          <span class="stat-lbl">End</span>
        </div>
      </div>

      <!-- ── Playback ───────────────────────────────────────────────── -->
      <div class="playback-section" role="region" aria-label="Playback controls">
        <div class="playback-header">
          <span class="section-label">Playback</span>
          <time
            class="playback-time"
            aria-live="polite"
            aria-label="Current position: {formatTime($historyPoints[$historyPlayback.index]?.t)}"
          >
            {formatTime($historyPoints[$historyPlayback.index]?.t)}
          </time>
        </div>

        <input
          type="range"
          min="0"
          max={Math.max(0, $historyPoints.length - 1)}
          value={$historyPlayback.index}
          oninput={onSlider}
          class="playback-slider"
          aria-label="Scrub through route"
          aria-valuemin="0"
          aria-valuemax={$historyPoints.length - 1}
          aria-valuenow={$historyPlayback.index}
        />

        <button
          class="btn btn-sm play-btn"
          class:btn-primary={!$historyPlayback.playing}
          class:btn-secondary={$historyPlayback.playing}
          onclick={togglePlayback}
          aria-label={$historyPlayback.playing ? 'Pause route playback' : 'Play route playback'}
          aria-pressed={$historyPlayback.playing}
        >
          {#if $historyPlayback.playing}
            <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
            Pause
          {:else}
            <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><polygon points="5 3 19 12 5 21 5 3"/></svg>
            Play
          {/if}
        </button>

        {#if $historyPoints[$historyPlayback.index]}
          <p class="playback-speed" aria-live="polite">
            Speed: <strong>{$historyPoints[$historyPlayback.index].speed?.toFixed(1) || '0'} km/h</strong>
          </p>
        {/if}
      </div>

    {:else if $historyVisible && $historyPoints.length === 0}
      <!-- ── Empty state with CTA ──────────────────────────────────── -->
      <div class="empty-state" role="status">
        <div class="empty-icon" aria-hidden="true">
          <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 12h18M12 3l9 9-9 9-9-9 9-9z"/></svg>
        </div>
        <p class="empty-title">No route data for this day</p>
        <p class="empty-sub">Location history is stored for 30 days. Try a different date or person.</p>
        <button class="btn btn-secondary btn-sm empty-cta" onclick={() => { historyVisible.set(false); }}>
          Try another date
        </button>
      </div>

    {:else if !selectedUserId}
      <!-- ── Onboarding hint ────────────────────────────────────────── -->
      <div class="onboard-hint" role="note">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
        <p>Pick a person and a date to replay their route on the map.</p>
      </div>
    {/if}

  </div>
{:else}
  <div class="panel-shell panel-left panel-base">
    <div class="panel-header">
      <h3>Route History</h3>
      <button class="panel-close" onclick={() => dispatch('close')} aria-label="Close route history">&times;</button>
    </div>
    <div class="panel-body">
      <p>View route history from the sidebar.</p>
    </div>
  </div>
{/if}

<style>
  .history-panel { padding: 0; }

  /* ── Section header ────────────────────────────────────────────────────── */
  .section-header {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-4) var(--space-4) var(--space-2);
  }

  .section-label {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.08em;
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
    padding: var(--space-2) var(--space-2-5);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    background: var(--surface-3);
    color: var(--text-primary);
    box-sizing: border-box;
    font-family: var(--font-sans);
    transition: border-color var(--duration-fast) var(--ease-out),
                box-shadow var(--duration-fast) var(--ease-out);
  }

  .field-input:focus {
    outline: none;
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px rgba(20, 184, 166, 0.18);
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
    display: flex;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-4);
    flex-wrap: wrap;
    border-top: 1px solid var(--border-subtle);
  }

  .stat-chip {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: var(--space-2) var(--space-3);
    min-width: 60px;
    flex: 1;
  }

  .stat-val {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .stat-lbl {
    font-size: var(--text-2xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-tertiary);
  }

  /* ── Playback ─────────────────────────────────────────────────────────── */
  .playback-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    border-top: 1px solid var(--border-subtle);
  }

  .playback-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .playback-time {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-secondary);
    font-variant-numeric: tabular-nums;
  }

  .playback-slider {
    width: 100%;
    height: var(--space-1);
    -webkit-appearance: none;
    appearance: none;
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    cursor: pointer;
    accent-color: var(--primary-500);
    outline: none;
  }

  .playback-slider:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 3px;
  }

  .playback-slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: var(--space-5);
    height: var(--space-5);
    border-radius: var(--radius-full);
    background: var(--primary-500);
    border: 2px solid var(--surface-0);
    box-shadow: 0 0 var(--space-2) rgba(20, 184, 166, 0.45);
    cursor: pointer;
    transition: transform var(--duration-fast) var(--ease-spring);
  }

  .playback-slider::-webkit-slider-thumb:hover {
    transform: scale(1.2);
  }

  .playback-slider::-moz-range-thumb {
    width: var(--space-5);
    height: var(--space-5);
    border-radius: var(--radius-full);
    background: var(--primary-500);
    border: 2px solid var(--surface-0);
    cursor: pointer;
  }

  .play-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-1-5);
    width: 100%;
    min-height: 44px;
  }

  .playback-speed {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    margin: 0;
  }

  .playback-speed strong {
    color: var(--text-primary);
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
  }

  /* ── Empty state ─────────────────────────────────────────────────────── */
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-8) var(--space-4);
    text-align: center;
  }

  .empty-icon {
    width: 56px;
    height: 56px;
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    color: var(--text-tertiary);
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: var(--space-1);
  }

  .empty-title {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    margin: 0;
  }

  .empty-sub {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    line-height: var(--leading-normal);
    max-width: 240px;
    margin: 0;
  }

  .empty-cta {
    margin-top: var(--space-2);
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
    .playback-slider::-webkit-slider-thumb:hover { transform: none; }
  }
</style>
