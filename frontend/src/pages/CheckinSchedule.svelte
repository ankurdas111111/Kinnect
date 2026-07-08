<script>
  import { run } from 'svelte/legacy';

  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { authUser } from '../lib/stores/auth.js';
  import { mySafetyStatus } from '../lib/stores/map.js';
  import { banner } from '../lib/stores/sos.js';
  import { socket } from '../lib/socket.js';
  import Card from '../components/primitives/Card.svelte';
  import EmptyState from '../components/primitives/EmptyState.svelte';

  run(() => {
    if (!$authUser) push('/login');
  });

  // ── Local config mirrors the server state ──────────────────────────────────
  let enabled = $state(false);
  let intervalMinutes = $state(15);
  let overdueMinutes = $state(5);
  let lastCheckInAt = $state(null);
  let dirty = $state(false);
  let saving = $state(false);

  // ── History log (session-scoped) ───────────────────────────────────────────
  let log = $state([]);
  function addLog(type, text) {
    log = [{ type, text, ts: Date.now() }, ...log].slice(0, 50);
  }

  // ── Countdown timer ────────────────────────────────────────────────────────
  let countdownMs = $state(null);
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
    if (ms <= 0) return 'Check in now';
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
    banner.set({ type: 'info', text: "Check-in sent — your family knows you're safe.", actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 2500);
  }

  // ── Socket listeners ───────────────────────────────────────────────────────
  // Named handler refs are required so socket.off() removes only THIS component's
  // listeners. Without refs, socket.off('eventName') strips ALL handlers for that
  // event — including the global ones registered in socket.js — silently breaking
  // check-in alerts for the entire session.
  const _onCheckInRequest = () => {
    addLog('request', 'Check-in reminder sent');
  };
  const _onCheckInUpdate = (data) => {
    if (data?.userId === $authUser?.userId) {
      lastCheckInAt = data.lastCheckInAt;
      mySafetyStatus.update(s => ({ ...s, checkIn: { ...s.checkIn, lastCheckInAt: data.lastCheckInAt } }));
      addLog('ok', `Check-in recorded at ${formatTime(data.lastCheckInAt)}`);
      updateCountdown();
    }
  };
  const _onCheckInMissed = () => {
    addLog('missed', 'Missed check-in — your family was notified');
  };

  onMount(() => {
    countdownInterval = setInterval(() => updateCountdown(), 1000);
    socket.on('checkInRequest', _onCheckInRequest);
    socket.on('checkInUpdate', _onCheckInUpdate);
    socket.on('checkInMissed', _onCheckInMissed);
  });

  onDestroy(() => {
    socket.off('checkInRequest', _onCheckInRequest);
    socket.off('checkInUpdate', _onCheckInUpdate);
    socket.off('checkInMissed', _onCheckInMissed);
    unsubStatus();
    if (countdownInterval) clearInterval(countdownInterval);
  });

  const INTERVALS = [5, 10, 15, 30, 60, 120];
  const OVERDUE   = [2, 5, 10, 15];

  function intervalLabel(m) {
    if (m < 60) return `${m}m`;
    return `${m / 60}h`;
  }

  let countdownClass = $derived(countdownMs != null && countdownMs <= 0 ? 'countdown-due'
    : countdownMs != null && countdownMs < 120000 ? 'countdown-soon' : 'countdown-ok');

  // Ring state mirrors the countdown urgency (drives conic color + glow)
  let ringClass = $derived(countdownMs != null && countdownMs <= 0 ? 'ring-due'
    : countdownMs != null && countdownMs < 120000 ? 'ring-soon' : 'ring-ok');

  let isDue = $derived(countdownMs != null && countdownMs <= 0);

  // Fraction of the interval still remaining → arc fill %. 0 when overdue/idle.
  let ringPct = $derived(
    countdownMs == null || intervalMinutes <= 0
      ? 0
      : Math.max(0, Math.min(100, Math.round((countdownMs / (intervalMinutes * 60 * 1000)) * 100)))
  );

  // Glow color for history Cards, keyed on event type.
  function logGlow(type) {
    if (type === 'ok') return 'success';
    if (type === 'missed') return 'danger';
    return 'primary';
  }
</script>

<div class="page-shell page-enter aurora-ambient">

  <!-- ── Top bar ─────────────────────────────────────────────────────────── -->
  <header class="page-header">
    <button class="back-btn" onclick={() => push('/')} aria-label="Back to map">
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

    <!-- ── Live countdown ring ─────────────────────────────────────────────── -->
    {#if enabled}
      <div class="countdown-card" class:countdown-card-urgent={isDue} in:fly={{ y: 12, duration: 240, easing: cubicOut }}>
        <span class="countdown-heading">Next check-in</span>

        <div
          class="ring {ringClass}"
          style="--ring-pct: {ringPct}%"
          role="img"
          aria-label="Time until next check-in: {formatCountdown(countdownMs)}"
        >
          <div class="ring-inner">
            <span class="ring-value {countdownClass}">{formatCountdown(countdownMs)}</span>
            {#if countdownMs != null && countdownMs > 0}
              <span class="ring-caption">remaining</span>
            {/if}
          </div>
        </div>

        {#if lastCheckInAt}
          <div class="countdown-meta">Last: {formatRelative(lastCheckInAt)} · {formatTime(lastCheckInAt)}</div>
        {:else}
          <div class="countdown-meta">No check-in recorded yet</div>
        {/if}

        <button class="imok-btn tactile" onclick={imOk} aria-label="Send I'm OK check-in">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
          I'm OK
        </button>
      </div>
    {:else}
      <div in:fade={{ duration: 200 }}>
        <EmptyState
          title="No check-in schedule yet"
          body="Turn on monitoring below and we'll pulse a reminder on your interval — your family is alerted only if you miss it."
          tone="primary"
        >
          {#snippet icon()}
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="9"/><polyline points="12 7 12 12 15 14"/></svg>
          {/snippet}
        </EmptyState>
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
          onclick={() => { enabled = !enabled; dirty = true; }}
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
            class="pill-btn tactile"
            class:pill-active={intervalMinutes === m}
            disabled={!enabled}
            aria-pressed={intervalMinutes === m}
            onclick={() => { intervalMinutes = m; markDirty(); }}
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
            class="pill-btn tactile"
            class:pill-active={overdueMinutes === m}
            disabled={!enabled}
            aria-pressed={overdueMinutes === m}
            onclick={() => { overdueMinutes = m; markDirty(); }}
          >{m}m</button>
        {/each}
      </div>
    </section>

    <!-- ── Save ────────────────────────────────────────────────────────────── -->
    {#if dirty}
      <div in:fly={{ y: 8, duration: 200, easing: cubicOut }}>
        <button class="save-btn" onclick={save} disabled={saving} aria-label="Save check-in settings">
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
          <span class="section-heading">Session log</span>
          <button class="clear-log-btn" onclick={() => log = []} aria-label="Clear session log">Clear</button>
        </div>
        <div class="log-list" aria-live="polite" aria-label="Check-in event log">
          {#each log as entry (entry.ts)}
            <div class="log-wrap" in:fly={{ x: -8, duration: 180, easing: cubicOut }}>
              <Card variant="glass" padding="sm" hover={false} glow={logGlow(entry.type)}>
                <div class="log-entry log-{entry.type}">
                  <span class="log-dot" aria-hidden="true"></span>
                  <span class="log-text">{entry.text}</span>
                  <span class="log-time">{formatRelative(entry.ts)}</span>
                </div>
              </Card>
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
    width: 44px;
    height: 44px;
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
    font-size: var(--text-xl);
    font-weight: 800;
    color: var(--text-primary);
    margin: 0;
    line-height: 1.2;
    letter-spacing: -0.02em;
  }

  .page-subtitle {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }

  .header-status-badge {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--success-500);
    background: var(--success-500-12);
    border: 1px solid var(--success-500-20);
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
    position: relative;
    background: linear-gradient(135deg, var(--primary-500-12) 0%, var(--primary-500-08) 100%);
    border: 1px solid var(--primary-500-20);
    border-radius: var(--radius-2xl, 20px);
    padding: var(--space-5, 20px) var(--space-4);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-2);
    text-align: center;
    transition: border-color 300ms var(--ease-out), background 300ms var(--ease-out);
  }

  .countdown-card-urgent {
    border-color: var(--danger-500-20);
    background: linear-gradient(135deg, var(--danger-500-12) 0%, var(--danger-500-20) 100%);
  }

  .countdown-heading {
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  /* ── Conic-gradient progress ring ────────────────────────────────────────── */
  .ring {
    position: relative;
    width: 184px;
    height: 184px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    margin: var(--space-2) 0;
    background: conic-gradient(
      from -90deg,
      var(--ring-color, var(--primary-500)) var(--ring-pct, 0%),
      var(--surface-inset) 0
    );
  }
  .ring-ok   { --ring-color: var(--primary-500); }
  .ring-soon { --ring-color: var(--warning-500); }
  .ring-due  { --ring-color: var(--danger-500); }

  /* Pulsing danger glow when overdue — opacity-only (GPU safe) */
  .ring-due::before {
    content: '';
    position: absolute;
    inset: -8px;
    border-radius: 50%;
    box-shadow: 0 0 30px 4px var(--danger-500-20);
    pointer-events: none;
    animation: ring-glow 1.4s ease-in-out infinite;
  }

  @keyframes ring-glow {
    0%, 100% { opacity: 0.35; }
    50%       { opacity: 1; }
  }

  .ring-inner {
    width: calc(100% - 26px);
    height: calc(100% - 26px);
    border-radius: 50%;
    background: var(--surface-1);
    border: 1px solid var(--border-subtle);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
  }

  .ring-value {
    font-family: var(--font-display);
    font-size: clamp(22px, 6vw, 30px);
    font-weight: 900;
    letter-spacing: -0.02em;
    line-height: 1.05;
    text-align: center;
    padding: 0 var(--space-2);
    transition: color 300ms var(--ease-out);
  }

  .ring-caption {
    font-size: var(--text-2xs, 10px);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--text-tertiary);
  }

  .countdown-ok   { color: var(--primary-400); }
  .countdown-soon { color: var(--warning-500); }
  .countdown-due  { color: var(--danger-500); animation: pulse-due 1s ease-in-out infinite; }

  @keyframes pulse-due {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0.55; }
  }

  @media (prefers-reduced-motion: reduce) {
    .countdown-due { animation: none; }
    .ring-due::before { animation: none; opacity: 0.6; }
  }

  .countdown-meta {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }

  .imok-btn {
    margin-top: var(--space-2);
    min-height: 44px;
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding: 10px 28px;
    background: var(--primary-600);
    color: var(--text-inverse, white);
    border: none;
    border-radius: var(--radius-full);
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    cursor: pointer;
    box-shadow: var(--glow-primary), 0 2px 8px var(--shadow-color, rgba(0,0,0,0.25));
    transition: transform 150ms var(--ease-spring), box-shadow 200ms var(--ease-out), background 150ms;
  }
  .imok-btn:hover {
    background: var(--primary-500);
    transform: translateY(-1px) scale(1.03);
    box-shadow: var(--glow-primary), 0 4px 14px var(--shadow-color, rgba(0,0,0,0.3));
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
    font-size: var(--text-base);
    font-weight: 800;
    letter-spacing: -0.01em;
    color: var(--text-primary);
  }

  .settings-hint {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    line-height: 1.45;
  }

  .section-heading {
    font-family: var(--font-display);
    font-size: var(--text-base);
    font-weight: 800;
    letter-spacing: -0.01em;
    color: var(--text-primary);
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
    background: var(--text-inverse, white);
    box-shadow: 0 1px 4px var(--shadow-color, rgba(0,0,0,0.25));
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
    min-height: 44px;
    font-family: var(--font-display);
    font-size: var(--text-xs);
    font-weight: 600;
    border: 1px solid var(--border-default);
    border-radius: var(--radius-full);
    background: var(--surface-2);
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 120ms, color 120ms, box-shadow 150ms;
    min-width: 44px;
    text-align: center;
  }
  .pill-btn:hover:not(:disabled) { background: var(--surface-3, var(--surface-2)); color: var(--text-primary); }
  .pill-btn.pill-active {
    background: var(--primary-600);
    color: var(--text-inverse, white);
    border-color: var(--primary-500);
    box-shadow: var(--glow-primary);
  }
  .pill-btn:disabled { cursor: not-allowed; }

  /* ── Save button ─────────────────────────────────────────────────────────── */
  .save-btn {
    width: 100%;
    min-height: 44px;
    padding: 14px;
    background: var(--primary-600);
    color: var(--text-inverse, white);
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
    box-shadow: var(--glow-primary);
    transition: background 150ms, transform 150ms var(--ease-spring), opacity 200ms;
  }
  .save-btn:hover { background: var(--primary-500); transform: translateY(-1px); }
  .save-btn:active { transform: scale(0.98); }
  .save-btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none; }

  .saving-spinner {
    width: 14px;
    height: 14px;
    border: 2px solid var(--primary-500-30);
    border-top-color: var(--text-inverse, white);
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
    padding: 10px 12px;
    min-height: 44px;
    border-radius: var(--radius-sm);
    transition: color 120ms, background 120ms;
  }
  .clear-log-btn:hover { color: var(--danger-400); background: var(--danger-500-12); }

  .log-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5, 6px);
  }

  .log-entry {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-xs);
  }

  .log-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .log-ok      .log-dot { background: var(--success-500); }
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
