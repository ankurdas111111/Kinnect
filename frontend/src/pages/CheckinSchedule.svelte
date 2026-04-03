<script>
  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { authUser } from '../lib/stores/auth.js';
  import { mySafetyStatus } from '../lib/stores/map.js';
  import { banner } from '../lib/stores/sos.js';
  import { socket } from '../lib/socket.js';

  $: if (!$authUser) push('/login');

  // ── Local config mirrors the server state ──────────────────────────────────
  let enabled = false;
  let intervalMinutes = 15;
  let overdueMinutes = 5;
  let lastCheckInAt = null;
  let dirty = false;
  let saving = false;

  // ── History log (session-scoped) ───────────────────────────────────────────
  let log = [];
  function addLog(type, text) {
    log = [{ type, text, ts: Date.now() }, ...log].slice(0, 50);
  }

  // ── Countdown timer ────────────────────────────────────────────────────────
  let countdownMs = null;
  let countdownInterval = null;

  function updateCountdown() {
    if (!enabled || !lastCheckInAt || intervalMinutes <= 0) {
      countdownMs = null;
      return;
    }
    const nextAt = lastCheckInAt + intervalMinutes * 60 * 1000;
    countdownMs = nextAt - Date.now();
  }

  function formatCountdown(ms) {
    if (ms == null) return '—';
    if (ms <= 0) return 'Due now';
    const totalSec = Math.floor(ms / 1000);
    const m = Math.floor(totalSec / 60);
    const s = totalSec % 60;
    return `${m}m ${s.toString().padStart(2, '0')}s`;
  }

  function formatTime(ts) {
    if (!ts) return '—';
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }

  function formatRelative(ts) {
    const diff = Date.now() - ts;
    if (diff < 60000) return 'just now';
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
    return new Date(ts).toLocaleDateString();
  }

  // ── Load from store ────────────────────────────────────────────────────────
  const unsubStatus = mySafetyStatus.subscribe(s => {
    const ci = s?.checkIn;
    if (!ci) return;
    enabled = ci.enabled ?? false;
    if (!dirty) {
      intervalMinutes = ci.intervalMinutes ?? 15;
      overdueMinutes = ci.overdueMinutes ?? 5;
    }
    lastCheckInAt = ci.lastCheckInAt ?? null;
    updateCountdown();
  });

  // ── Interval / overdueMinutes changes mark dirty ───────────────────────────
  function markDirty() { dirty = true; }

  // ── Save ───────────────────────────────────────────────────────────────────
  function save() {
    if (saving) return;
    saving = true;
    socket.emit('setCheckInRules', { enabled, intervalMinutes, overdueMinutes });
    dirty = false;
    setTimeout(() => { saving = false; }, 1500);
    banner.set({ type: 'info', text: 'Check-in settings saved.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2000);
  }

  // ── Acknowledge check-in ───────────────────────────────────────────────────
  function imOk() {
    socket.emit('checkInAck');
    lastCheckInAt = Date.now();
    mySafetyStatus.update(s => ({ ...s, checkIn: { ...s.checkIn, lastCheckInAt: lastCheckInAt } }));
    addLog('ok', "I'm OK sent");
    updateCountdown();
    banner.set({ type: 'info', text: "Check-in sent — you're all good.", actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2500);
  }

  // ── Socket listeners ───────────────────────────────────────────────────────
  onMount(() => {
    countdownInterval = setInterval(() => updateCountdown(), 1000);

    socket.on('checkInRequest', () => {
      addLog('request', 'Server requested check-in');
    });

    socket.on('checkInUpdate', (data) => {
      if (data?.userId === $authUser?.userId) {
        lastCheckInAt = data.lastCheckInAt;
        mySafetyStatus.update(s => ({ ...s, checkIn: { ...s.checkIn, lastCheckInAt: data.lastCheckInAt } }));
        addLog('ok', `Check-in recorded at ${formatTime(data.lastCheckInAt)}`);
        updateCountdown();
      }
    });

    socket.on('checkInMissed', () => {
      addLog('missed', 'Check-in missed — alert sent to contacts');
    });

    return () => {
      socket.off('checkInRequest');
      socket.off('checkInUpdate');
      socket.off('checkInMissed');
    };
  });

  onDestroy(() => {
    unsubStatus();
    if (countdownInterval) clearInterval(countdownInterval);
  });

  const INTERVALS = [5, 10, 15, 30, 60, 120];
  const OVERDUE   = [2, 5, 10, 15];

  function intervalLabel(m) {
    if (m < 60) return `${m}m`;
    return `${m / 60}h`;
  }

  $: countdownClass = countdownMs != null && countdownMs <= 0 ? 'countdown-due'
    : countdownMs != null && countdownMs < 120000 ? 'countdown-soon' : 'countdown-ok';
</script>

<div class="page-shell page-enter aurora-ambient">

  <!-- ── Top bar ─────────────────────────────────────────────────────────── -->
  <header class="page-header">
    <button class="back-btn" on:click={() => push('/')} aria-label="Back to map">
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M19 12H5"/><path d="m12 5-7 7 7 7"/></svg>
    </button>
    <div class="header-title-block">
      <h1 class="page-title">Check-in</h1>
      <span class="page-subtitle">Auto-safety pulse</span>
    </div>
    {#if enabled}
      <span class="header-status-badge" in:fade={{ duration: 200 }}>Active</span>
    {/if}
  </header>

  <div class="page-content">

    <!-- ── Live countdown card ─────────────────────────────────────────────── -->
    {#if enabled}
      <div class="countdown-card" in:fly={{ y: 12, duration: 240, easing: cubicOut }}>
        <div class="countdown-label">Next check-in in</div>
        <div class="countdown-value {countdownClass}">
          {formatCountdown(countdownMs)}
        </div>
        {#if lastCheckInAt}
          <div class="countdown-meta">Last: {formatRelative(lastCheckInAt)} · {formatTime(lastCheckInAt)}</div>
        {:else}
          <div class="countdown-meta">No check-in recorded yet</div>
        {/if}

        <button class="imok-btn" on:click={imOk} aria-label="Send I'm OK check-in">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
          I'm OK
        </button>
      </div>
    {/if}

    <!-- ── Master toggle ───────────────────────────────────────────────────── -->
    <section class="settings-card">
      <div class="settings-row toggle-row">
        <div class="settings-label-block">
          <span class="settings-label">Enable check-in monitoring</span>
          <span class="settings-hint">Contacts get an alert if you miss your check-in window</span>
        </div>
        <button
          class="toggle-switch"
          class:toggle-on={enabled}
          role="switch"
          aria-checked={enabled}
          aria-label="Enable check-in monitoring"
          on:click={() => { enabled = !enabled; dirty = true; }}
        >
          <span class="toggle-knob"></span>
        </button>
      </div>
    </section>

    <!-- ── Interval ────────────────────────────────────────────────────────── -->
    <section class="settings-card" class:settings-card-dimmed={!enabled}>
      <div class="settings-label-block">
        <span class="settings-label">Check-in every</span>
        <span class="settings-hint">How often you're expected to confirm you're safe</span>
      </div>
      <div class="pill-group" role="group" aria-label="Check-in interval">
        {#each INTERVALS as m}
          <button
            class="pill-btn"
            class:pill-active={intervalMinutes === m}
            disabled={!enabled}
            aria-pressed={intervalMinutes === m}
            on:click={() => { intervalMinutes = m; markDirty(); }}
          >{intervalLabel(m)}</button>
        {/each}
      </div>
    </section>

    <!-- ── Overdue tolerance ───────────────────────────────────────────────── -->
    <section class="settings-card" class:settings-card-dimmed={!enabled}>
      <div class="settings-label-block">
        <span class="settings-label">Alert after missing by</span>
        <span class="settings-hint">Grace period before your contacts are notified</span>
      </div>
      <div class="pill-group" role="group" aria-label="Overdue tolerance">
        {#each OVERDUE as m}
          <button
            class="pill-btn"
            class:pill-active={overdueMinutes === m}
            disabled={!enabled}
            aria-pressed={overdueMinutes === m}
            on:click={() => { overdueMinutes = m; markDirty(); }}
          >{m}m</button>
        {/each}
      </div>
    </section>

    <!-- ── Save ────────────────────────────────────────────────────────────── -->
    {#if dirty}
      <div in:fly={{ y: 8, duration: 200, easing: cubicOut }}>
        <button class="save-btn" on:click={save} disabled={saving} aria-label="Save check-in settings">
          {#if saving}
            <span class="saving-spinner" aria-hidden="true"></span>
            Saving…
          {:else}
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
            Save Settings
          {/if}
        </button>
      </div>
    {/if}

    <!-- ── How it works ────────────────────────────────────────────────────── -->
    <section class="how-card">
      <span class="how-title">How it works</span>
      <ol class="how-list">
        <li>Enable monitoring and set your interval</li>
        <li>Tap <strong>I'm OK</strong> (or the check-in button on the main screen) before time runs out</li>
        <li>If you miss the window by more than the grace period, all your contacts and guardians get an alert</li>
        <li>The alert automatically opens an SOS watch link so they can see your last position</li>
      </ol>
    </section>

    <!-- ── Session log ─────────────────────────────────────────────────────── -->
    {#if log.length > 0}
      <section class="log-section">
        <div class="log-header">
          <span class="settings-label">Session log</span>
          <button class="clear-log-btn" on:click={() => log = []} aria-label="Clear session log">Clear</button>
        </div>
        <div class="log-list" aria-live="polite" aria-label="Check-in event log">
          {#each log as entry (entry.ts)}
            <div
              class="log-entry log-{entry.type}"
              in:fly={{ x: -8, duration: 180, easing: cubicOut }}
            >
              <span class="log-dot" aria-hidden="true"></span>
              <span class="log-text">{entry.text}</span>
              <span class="log-time">{formatRelative(entry.ts)}</span>
            </div>
          {/each}
        </div>
      </section>
    {/if}

  </div>
</div>

<style>
  /* ── Page shell ──────────────────────────────────────────────────────────── */
  .page-shell {
    min-height: 100dvh;
    background: var(--surface-0);
    font-family: var(--font-sans);
    display: flex;
    flex-direction: column;
    padding-bottom: calc(env(safe-area-inset-bottom) + 32px);
  }

  /* ── Header ──────────────────────────────────────────────────────────────── */
  .page-header {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: calc(env(safe-area-inset-top) + 12px) var(--space-4) var(--space-3);
    background: var(--surface-0);
    border-bottom: 1px solid var(--border-subtle);
    position: sticky;
    top: 0;
    z-index: 10;
  }

  .back-btn {
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-lg);
    background: var(--surface-1);
    border: 1px solid var(--border-subtle);
    color: var(--text-primary);
    cursor: pointer;
    flex-shrink: 0;
    transition: background 150ms var(--ease-out), transform 120ms var(--ease-spring);
  }
  .back-btn:hover { background: var(--surface-2); }
  .back-btn:active { transform: scale(0.93); }

  .header-title-block {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .page-title {
    font-family: var(--font-display);
    font-size: var(--text-lg);
    font-weight: 800;
    color: var(--text-primary);
    margin: 0;
    line-height: 1.2;
  }

  .page-subtitle {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }

  .header-status-badge {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: #22c55e;
    background: rgba(34, 197, 94, 0.12);
    border: 1px solid rgba(34, 197, 94, 0.28);
    border-radius: var(--radius-full);
    padding: 3px 10px;
  }

  /* ── Content area ────────────────────────────────────────────────────────── */
  .page-content {
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    max-width: 540px;
    width: 100%;
    margin: 0 auto;
  }

  /* ── Countdown card ──────────────────────────────────────────────────────── */
  .countdown-card {
    background: linear-gradient(135deg, rgba(99,102,241,0.12) 0%, rgba(139,92,246,0.08) 100%);
    border: 1px solid rgba(99,102,241,0.22);
    border-radius: var(--radius-2xl, 20px);
    padding: var(--space-5, 20px) var(--space-4);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-2);
    text-align: center;
  }

  .countdown-label {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  .countdown-value {
    font-family: var(--font-display);
    font-size: clamp(32px, 8vw, 48px);
    font-weight: 900;
    letter-spacing: -0.02em;
    line-height: 1;
    transition: color 300ms var(--ease-out);
  }

  .countdown-ok   { color: var(--primary-400); }
  .countdown-soon { color: #f59e0b; }
  .countdown-due  { color: var(--danger-500); animation: pulse-due 1s ease-in-out infinite; }

  @keyframes pulse-due {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0.55; }
  }

  @media (prefers-reduced-motion: reduce) {
    .countdown-due { animation: none; }
  }

  .countdown-meta {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }

  .imok-btn {
    margin-top: var(--space-2);
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding: 10px 28px;
    background: var(--primary-600);
    color: white;
    border: none;
    border-radius: var(--radius-full);
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    cursor: pointer;
    box-shadow: var(--glow-primary, 0 0 16px rgba(99,102,241,0.35)), 0 2px 8px rgba(0,0,0,0.25);
    transition: transform 150ms var(--ease-spring), box-shadow 200ms var(--ease-out), background 150ms;
  }
  .imok-btn:hover {
    background: var(--primary-500);
    transform: translateY(-1px) scale(1.03);
    box-shadow: var(--glow-primary, 0 0 22px rgba(99,102,241,0.5)), 0 4px 14px rgba(0,0,0,0.3);
  }
  .imok-btn:active { transform: scale(0.97); }

  /* ── Settings card ───────────────────────────────────────────────────────── */
  .settings-card {
    background: var(--surface-1);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-xl);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    transition: opacity 200ms var(--ease-out);
  }

  .settings-card-dimmed {
    opacity: 0.45;
    pointer-events: none;
  }

  .settings-row { display: flex; align-items: center; gap: var(--space-3); }
  .toggle-row   { justify-content: space-between; }

  .settings-label-block {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
  }

  .settings-label {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
  }

  .settings-hint {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    line-height: 1.45;
  }

  /* ── Toggle switch ───────────────────────────────────────────────────────── */
  .toggle-switch {
    width: 48px;
    height: 28px;
    border-radius: 14px;
    border: none;
    cursor: pointer;
    flex-shrink: 0;
    position: relative;
    transition: background 200ms var(--ease-out);
    background: var(--surface-inset);
    outline-offset: 3px;
  }
  .toggle-switch.toggle-on { background: var(--primary-600); }
  .toggle-knob {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: white;
    box-shadow: 0 1px 4px rgba(0,0,0,0.25);
    transition: transform 200ms var(--ease-spring);
  }
  .toggle-on .toggle-knob { transform: translateX(20px); }

  /* ── Pill buttons ────────────────────────────────────────────────────────── */
  .pill-group {
    display: flex;
    gap: var(--space-1-5, 6px);
    flex-wrap: wrap;
  }

  .pill-btn {
    padding: 5px 14px;
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 600;
    border: 1px solid var(--border-default);
    border-radius: var(--radius-full);
    background: var(--surface-2);
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 120ms, color 120ms, transform 120ms var(--ease-spring), box-shadow 150ms;
    min-width: 44px;
    text-align: center;
  }
  .pill-btn:hover:not(:disabled) { background: var(--surface-3, var(--surface-2)); color: var(--text-primary); }
  .pill-btn.pill-active {
    background: var(--primary-600);
    color: white;
    border-color: var(--primary-500);
    box-shadow: var(--glow-primary, 0 0 10px rgba(99,102,241,0.3));
    transform: scale(1.05);
  }
  .pill-btn:disabled { cursor: not-allowed; }

  /* ── Save button ─────────────────────────────────────────────────────────── */
  .save-btn {
    width: 100%;
    padding: 14px;
    background: var(--primary-600);
    color: white;
    border: none;
    border-radius: var(--radius-xl);
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    box-shadow: var(--glow-primary, 0 0 16px rgba(99,102,241,0.3));
    transition: background 150ms, transform 150ms var(--ease-spring), opacity 200ms;
  }
  .save-btn:hover { background: var(--primary-500); transform: translateY(-1px); }
  .save-btn:active { transform: scale(0.98); }
  .save-btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none; }

  .saving-spinner {
    width: 14px;
    height: 14px;
    border: 2px solid rgba(255,255,255,0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
    flex-shrink: 0;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
  @media (prefers-reduced-motion: reduce) { .saving-spinner { animation: none; } }

  /* ── How it works ────────────────────────────────────────────────────────── */
  .how-card {
    background: var(--surface-inset);
    border: 1px dashed var(--border-subtle);
    border-radius: var(--radius-xl);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .how-title {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  .how-list {
    margin: 0;
    padding-left: 20px;
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5, 6px);
    font-size: var(--text-sm);
    color: var(--text-secondary);
    line-height: 1.55;
  }

  /* ── Log ─────────────────────────────────────────────────────────────────── */
  .log-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .log-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .clear-log-btn {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-tertiary);
    background: none;
    border: none;
    cursor: pointer;
    padding: 2px 6px;
    border-radius: var(--radius-sm);
    transition: color 120ms, background 120ms;
  }
  .clear-log-btn:hover { color: var(--danger-400); background: rgba(239,68,68,0.07); }

  .log-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5, 6px);
  }

  .log-entry {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: 8px var(--space-3);
    background: var(--surface-1);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    font-size: var(--text-xs);
  }

  .log-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .log-ok      .log-dot { background: #22c55e; }
  .log-request .log-dot { background: var(--primary-500); }
  .log-missed  .log-dot { background: var(--danger-500); }

  .log-text { flex: 1; color: var(--text-secondary); }
  .log-time {
    font-size: 11px;
    color: var(--text-tertiary);
    white-space: nowrap;
    flex-shrink: 0;
  }
</style>
