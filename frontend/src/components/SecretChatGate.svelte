<script>
  /**
   * SecretChatGate — PIN entry screen before secret chat is accessible.
   *
   * Props:
   *   peerName   — display name of the chat partner
   *   unlocking  — true while parent is processing the PIN
   *   error      — error string from parent (cleared on next submit)
   *
   * Events:
   *   submit(pin: string) — dispatched when user hits Open / Enter
   *
   * Imperative API (via bind:this):
   *   triggerShake()   — animate the input on wrong PIN
   *   triggerSuccess() — animate the icon on correct PIN
   *
   * CSS classes:
   *   .scv-pin-dot, .scv-pin-dot--filled, .scv-pin-dot--active — used by
   *   Playwright tests (scoped in :global so they survive Svelte scoping).
   *   .scv-cta-btn, .scv-cta-btn--active — used by tests as stable selectors.
   *
   * iOS notes:
   *   - Input font-size is fixed at 26px (above the 16px iOS auto-zoom threshold).
   *   - onMount focuses after 120ms to allow the animation to settle first.
   *   - _shakeTimer cleaned up in onDestroy.
   */
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import { haptics } from '../lib/haptics.js';

  const dispatch = createEventDispatcher();

  /**
   * @typedef {Object} Props
   * @property {string} [peerName]
   * @property {boolean} [unlocking]
   * @property {string} [error]
   */

  /** @type {Props} */
  let { peerName = 'Contact', unlocking = false, error = '' } = $props();

  let pinDigits = $state([]);
  let pinShake = $state(false);
  let pinInputEl = $state();
  let unlockSuccess = $state(false);
  let _shakeTimer = null;

  let pin = $derived(pinDigits.join(''));
  let pinReady = $derived(pin.length >= 4);
  // Placeholder names ('Sender' from the public viewer, 'Contact' default) get
  // generic copy instead of reading "Messages with Sender are…".
  let knownPeer = $derived(!!peerName && peerName !== 'Sender' && peerName !== 'Contact');

  onMount(() => {
    setTimeout(() => pinInputEl?.focus(), 120);
  });

  onDestroy(() => {
    clearTimeout(_shakeTimer);
  });

  function handleInput(e) {
    const digits = e.target.value.replace(/\D/g, '').slice(0, 8);
    e.target.value = digits;
    pinDigits = digits.split('');
  }

  function handleKeydown(e) {
    if (e.key === 'Enter') { e.preventDefault(); submit(); }
  }

  export function triggerShake() {
    pinShake = true;
    haptics.error?.();
    pinDigits = [];
    if (pinInputEl) pinInputEl.value = '';
    clearTimeout(_shakeTimer);
    _shakeTimer = setTimeout(() => { pinShake = false; }, 520);
  }

  export function triggerSuccess() {
    unlockSuccess = true;
    haptics.success?.();
  }

  function submit() {
    if (unlocking || pin.length < 4) return;
    dispatch('submit', pin);
  }
</script>

<div class="gate" role="region" aria-label="Enter PIN to read this note">
  <div class="gate-content" class:gate-content--success={unlockSuccess}>
    <!-- Lock pebble — quiet ember presence, no alarm chrome -->
    <div class="gate-icon-wrap" aria-hidden="true">
      <div class="gate-icon" class:gate-icon--success={unlockSuccess}>
        {#if unlockSuccess}
          <svg width="30" height="30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="11" width="18" height="11" rx="2"/>
            <path d="M7 11V7a5 5 0 0 1 9.9-1"/>
          </svg>
        {:else}
          <svg width="30" height="30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="11" width="18" height="11" rx="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
        {/if}
      </div>
    </div>

    <!-- Title + plain-words encryption explanation -->
    <div class="gate-text">
      <h2 class="gate-title verdict-voice">This conversation is sealed.</h2>
      <p class="gate-sub">
        {#if knownPeer}
          Messages with <strong>{peerName}</strong> are locked on this device.
        {:else}
          These messages are locked on this device.
        {/if}
        Enter the PIN you were given to open them — Kinnect can't read them, and can't recover the PIN.
      </p>
    </div>

    <!-- PIN dot visualizer — stable class names for Playwright tests -->
    <div class="gate-dots" aria-hidden="true" role="presentation">
      {#each Array(8) as _, i}
        <span
          class="gate-dot scv-pin-dot"
          class:gate-dot--filled={i < pinDigits.length}
          class:scv-pin-dot--filled={i < pinDigits.length}
          class:gate-dot--active={i === pinDigits.length - 1 && pinDigits.length > 0}
          class:scv-pin-dot--active={i === pinDigits.length - 1 && pinDigits.length > 0}
        ></span>
      {/each}
    </div>

    <!-- Hidden PIN input -->
    <div class="gate-input-wrap">
      <label class="sr-only" for="scv-gate-pin">Enter PIN — minimum 4 digits</label>
      <input
        id="scv-gate-pin"
        bind:this={pinInputEl}
        class="gate-pin-input"
        class:gate-pin-input--shake={pinShake}
        type="password"
        inputmode="numeric"
        pattern="\d*"
        maxlength="8"
        placeholder="••••"
        autocomplete="one-time-code"
        autocorrect="off"
        autocapitalize="none"
        oninput={handleInput}
        onkeydown={handleKeydown}
        aria-describedby={error ? 'gate-pin-err' : 'gate-pin-hint'}
        disabled={unlocking}
      />

      {#if error}
        <p class="gate-field-err" id="gate-pin-err" role="alert">{error}</p>
      {:else}
        <p class="gate-hint" id="gate-pin-hint">4–8 digits</p>
      {/if}
    </div>

    <!-- Open button — stable class names for Playwright tests -->
    <button
      class="gate-btn scv-cta-btn"
      class:gate-btn--ready={pinReady}
      class:scv-cta-btn--active={pinReady}
      onclick={submit}
      disabled={unlocking || !pinReady}
      type="button"
      aria-label={pinReady ? 'Open note' : 'Enter at least 4 digits'}
    >
      {#if unlocking}
        <span class="gate-btn-ring" aria-hidden="true"></span>
        <span>Opening…</span>
      {:else}
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          {#if pinReady}
            <rect x="3" y="11" width="18" height="11" rx="2"/>
            <path d="M7 11V7a5 5 0 0 1 9.9-1"/>
          {:else}
            <rect x="3" y="11" width="18" height="11" rx="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          {/if}
        </svg>
        <span>{pinReady ? 'Open Chat' : 'Enter PIN above'}</span>
      {/if}
    </button>

    <!-- Plain-words trust note, no jargon -->
    <p class="gate-footer" aria-hidden="true">
      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <rect x="3" y="11" width="18" height="11" rx="2"/>
        <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
      Unlocked only on this screen — never on a server.
    </p>
  </div>
</div>

<style>
  /* ── Gate shell — quiet warm paper, no vault theatrics ────────── */
  .gate {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    position: relative;
    overflow: hidden;
    min-height: 0;
    background: var(--chat-bg, var(--surface-0));
  }

  /* ── Content block ───────────────────────────────────────────── */
  .gate-content {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-5);
    width: 100%;
    max-width: 340px;
    padding: var(--space-6) var(--space-4) var(--space-8);
    text-align: center;
    animation: gate-content-in 0.3s var(--ease-out) both;
  }

  @keyframes gate-content-in {
    from { opacity: 0; transform: translateY(16px); }
    to   { opacity: 1; transform: translateY(0); }
  }

  .gate-content--success {
    animation: gate-content-out 0.3s var(--ease-out) both;
  }

  @keyframes gate-content-out {
    to { opacity: 0; transform: translateY(-16px); }
  }

  /* ── Lock pebble ─────────────────────────────────────────────── */
  .gate-icon-wrap {
    position: relative;
    width: 88px;
    height: 88px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .gate-icon {
    width: 88px;
    height: 88px;
    border-radius: var(--radius-full);
    background: var(--primary-100);
    border: 1px solid color-mix(in oklch, var(--primary-500) 24%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--primary-700);
    position: relative;
    z-index: 1;
    box-shadow: var(--shadow-sm);
    transition: color var(--duration-slow) var(--ease-out), background var(--duration-slow) var(--ease-out), border-color var(--duration-slow) var(--ease-out);
    flex-shrink: 0;
  }

  .gate-icon--success {
    color: var(--success-700);
    border-color: color-mix(in oklch, var(--success-500) 35%, transparent);
    background: color-mix(in oklch, var(--success-500) 10%, transparent);
    animation: gate-icon-success 0.5s var(--ease-spring);
  }

  @keyframes gate-icon-success {
    0%   { transform: scale(1); }
    40%  { transform: scale(1.12); }
    100% { transform: scale(1); }
  }

  /* ── Text ────────────────────────────────────────────────────── */
  .gate-text {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  /* The one serif moment on this screen — the verdict sentence. */
  .gate-title {
    margin: 0;
    font-size: var(--text-2xl);
    line-height: var(--leading-tight);
    color: var(--text-primary);
  }

  .gate-sub {
    margin: 0;
    font-size: var(--text-base);
    font-family: var(--font-sans);
    color: var(--text-secondary);
    line-height: var(--leading-relaxed);
  }

  .gate-sub strong {
    color: var(--text-primary);
    font-weight: 600;
  }

  /* ── PIN dot visualizer ──────────────────────────────────────── */
  .gate-dots {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    height: 16px;
  }

  .gate-dot {
    width: 8px;
    height: 8px;
    border-radius: var(--radius-full);
    background: transparent;
    border: 1px solid var(--border-strong);
    transition:
      background var(--duration-normal) var(--ease-out),
      transform  var(--duration-normal) var(--ease-spring);
    flex-shrink: 0;
  }

  .gate-dot--filled,
  :global(.scv-pin-dot--filled) {
    background: var(--primary-500);
    border-color: transparent;
  }

  .gate-dot--active,
  :global(.scv-pin-dot--active) {
    transform: scale(1.3);
  }

  /* ── PIN input — warm inset sheet, ember focus ring ──────────── */
  .gate-input-wrap {
    width: 100%;
    max-width: 280px;
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5);
    align-items: center;
  }

  .gate-pin-input {
    width: 100%;
    padding: var(--space-3-5) var(--space-4);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-input);
    color: var(--text-primary);
    /* 26px > 16px iOS threshold — prevents auto-zoom */
    font-size: 26px;
    letter-spacing: 0.4em;
    text-align: center;
    font-family: var(--font-sans);
    font-variant-numeric: tabular-nums;
    outline: none;
    transition: border-color var(--duration-normal) var(--ease-out), box-shadow var(--duration-normal) var(--ease-out);
    caret-color: var(--primary-500);
    -webkit-appearance: none;
    appearance: none;
    min-height: 64px;
    box-sizing: border-box;
  }

  .gate-pin-input::placeholder {
    color: var(--text-tertiary);
    letter-spacing: 0.3em;
    font-size: 20px;
    font-family: var(--font-sans);
  }

  .gate-pin-input:focus {
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px color-mix(in oklch, var(--primary-500) 16%, transparent);
  }

  @keyframes gate-shake {
    0%, 100% { transform: translateX(0); }
    15%  { transform: translateX(-8px); }
    35%  { transform: translateX(8px); }
    55%  { transform: translateX(-6px); }
    75%  { transform: translateX(5px); }
    90%  { transform: translateX(-2px); }
  }

  /* Wrong PIN is a stumble, not an emergency — ochre, never vermilion. */
  .gate-pin-input--shake {
    animation: gate-shake 0.48s cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
    border-color: var(--warning-500);
    box-shadow: 0 0 0 3px color-mix(in oklch, var(--warning-500) 16%, transparent);
  }

  .gate-field-err {
    margin: 0;
    font-size: var(--text-sm);
    font-family: var(--font-sans);
    color: var(--warning-700);
    font-weight: 500;
  }

  .gate-hint {
    margin: 0;
    font-size: var(--text-sm);
    font-family: var(--font-sans);
    color: var(--text-tertiary);
  }

  /* ── Open button — quiet until ready, then the one ember ─────── */
  .gate-btn {
    width: 100%;
    max-width: 280px;
    padding: var(--space-3-5) var(--space-4);
    border-radius: var(--radius-button);
    border: 1px solid var(--border-default);
    background: var(--surface-2);
    color: var(--text-tertiary);
    font-size: var(--text-base);
    font-weight: 600;
    font-family: var(--font-sans);
    cursor: pointer;
    min-height: 52px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    transition: background var(--duration-normal) var(--ease-out), color var(--duration-normal) var(--ease-out), border-color var(--duration-normal) var(--ease-out), transform var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }

  .gate-btn:disabled { opacity: 0.45; cursor: not-allowed; }

  .gate-btn:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 2px;
  }

  .gate-btn--ready {
    background: var(--primary-500);
    color: var(--text-on-primary);
    border-color: transparent;
    box-shadow: var(--shadow-primary);
  }

  .gate-btn--ready:hover:not(:disabled) {
    background: var(--primary-600);
  }

  .gate-btn--ready:active:not(:disabled) {
    transform: scale(0.98);
  }

  /* ── Button spinner ──────────────────────────────────────────── */
  .gate-btn-ring {
    width: 16px;
    height: 16px;
    border: 2px solid color-mix(in oklch, currentColor 30%, transparent);
    border-top-color: currentColor;
    border-radius: var(--radius-full);
    animation: gate-spin 0.7s linear infinite;
    flex-shrink: 0;
  }

  @keyframes gate-spin { to { transform: rotate(360deg); } }

  /* ── Footer note — plain words, quiet ink ────────────────────── */
  .gate-footer {
    margin: 0;
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    font-size: var(--text-xs);
    font-family: var(--font-sans);
    color: var(--text-tertiary);
  }
  .gate-footer svg { color: var(--primary-700); flex-shrink: 0; }

  /* ── Responsive — compact phones (≤ 380px) ──────────────────── */
  @media (max-width: 380px) {
    .gate-content {
      gap: var(--space-4);
      padding: var(--space-4) var(--space-3) var(--space-6);
    }

    .gate-icon-wrap,
    .gate-icon {
      width: 72px;
      height: 72px;
    }

    .gate-title {
      font-size: var(--text-xl);
    }

    .gate-pin-input {
      min-height: 56px;
      font-size: 22px;
    }

    .gate-btn {
      min-height: 48px;
      padding: var(--space-3);
    }
  }

  /* ── Responsive — landscape short screens ────────────────────── */
  @media (max-height: 600px) and (orientation: landscape) {
    .gate-content {
      gap: var(--space-3);
      padding: var(--space-2) var(--space-4) var(--space-4);
    }

    .gate-icon-wrap,
    .gate-icon {
      width: 56px;
      height: 56px;
    }
  }

  /* ── Accessibility ───────────────────────────────────────────── */
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }

  /* ── Reduced motion ──────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .gate-content   { animation: none; }
    .gate-content--success { animation: none; }
    .gate-icon      { animation: none; }
    .gate-icon--success { animation: none; }
    .gate-dot       { transition: none; }
    .gate-pin-input--shake { animation: none; }
    .gate-btn       { transition: none; }
    .gate-btn-ring  { animation: none; }
  }
</style>
