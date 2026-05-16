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

  export let peerName = 'Contact';
  export let unlocking = false;
  export let error = '';

  let pinDigits = [];
  let pinShake = false;
  let pinInputEl;
  let unlockSuccess = false;
  let _shakeTimer = null;

  $: pin = pinDigits.join('');
  $: pinReady = pin.length >= 4;

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
  <!-- Encrypted-space background texture -->
  <div class="gate-hex-bg" aria-hidden="true"></div>
  <div class="gate-glow" aria-hidden="true"></div>

  <div class="gate-content" class:gate-content--success={unlockSuccess}>
    <!-- Animated lock icon with concentric glow rings -->
    <div class="gate-icon-wrap" aria-hidden="true">
      <div class="gate-icon-ring gate-icon-ring--outer"></div>
      <div class="gate-icon-ring gate-icon-ring--inner"></div>
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

    <!-- Title + description -->
    <div class="gate-text">
      <h2 class="gate-title">Secret Chat</h2>
      <p class="gate-sub">
        Encrypted with <strong>{peerName}</strong>.<br>
        Only your PIN unlocks this conversation.
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
        on:input={handleInput}
        on:keydown={handleKeydown}
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
      on:click={submit}
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

    <!-- Crypto footer note -->
    <p class="gate-footer" aria-hidden="true">
      <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <rect x="3" y="11" width="18" height="11" rx="2"/>
        <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
      AES-GCM encrypted · PBKDF2 key derivation
    </p>
  </div>
</div>

<style>
  /* ── Gate shell ──────────────────────────────────────────────── */
  .gate {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    position: relative;
    overflow: hidden;
    min-height: 0;
  }

  /* ── Hex texture background ─────────────────────────────────── */
  .gate-hex-bg {
    position: absolute;
    inset: 0;
    background-image:
      repeating-linear-gradient(
        0deg,
        transparent,
        transparent 28px,
        rgba(20, 184, 166, 0.025) 28px,
        rgba(20, 184, 166, 0.025) 29px
      ),
      repeating-linear-gradient(
        60deg,
        transparent,
        transparent 28px,
        rgba(20, 184, 166, 0.015) 28px,
        rgba(20, 184, 166, 0.015) 29px
      ),
      repeating-linear-gradient(
        120deg,
        transparent,
        transparent 28px,
        rgba(20, 184, 166, 0.015) 28px,
        rgba(20, 184, 166, 0.015) 29px
      );
    animation: gate-hex-drift 20s linear infinite;
    pointer-events: none;
  }

  @keyframes gate-hex-drift {
    from { background-position: 0 0, 0 0, 0 0; }
    to   { background-position: 0 56px, 48px 0, 0 56px; }
  }

  /* ── Ambient glow ────────────────────────────────────────────── */
  .gate-glow {
    position: absolute;
    inset: 0;
    background:
      radial-gradient(ellipse 55% 45% at 50% 40%, rgba(20, 184, 166, 0.11) 0%, transparent 65%),
      radial-gradient(ellipse 40% 30% at 80% 75%, rgba(6, 182, 212, 0.07) 0%, transparent 55%);
    pointer-events: none;
    animation: gate-glow-breathe 7s ease-in-out infinite;
  }

  @keyframes gate-glow-breathe {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0.55; }
  }

  /* ── Content block ───────────────────────────────────────────── */
  .gate-content {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-5, 20px);
    width: 100%;
    max-width: 320px;
    padding: var(--space-6, 24px) var(--space-4, 16px) var(--space-8, 32px);
    text-align: center;
    animation: gate-content-in 0.4s var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
  }

  @keyframes gate-content-in {
    from { opacity: 0; transform: translateY(24px) scale(0.95); }
    to   { opacity: 1; transform: translateY(0) scale(1); }
  }

  .gate-content--success {
    animation: gate-content-out 0.3s var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1)) both;
  }

  @keyframes gate-content-out {
    to { opacity: 0; transform: translateY(-20px) scale(0.96); }
  }

  /* ── Lock icon with concentric rings ─────────────────────────── */
  .gate-icon-wrap {
    position: relative;
    width: 96px;
    height: 96px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .gate-icon-ring {
    position: absolute;
    border-radius: var(--radius-full, 9999px);
    border: 1px solid var(--chat-border-accent, rgba(20, 184, 166, 0.22));
  }

  .gate-icon-ring--outer {
    inset: 0;
    background: rgba(20, 184, 166, 0.04);
    animation: gate-ring-pulse 3s ease-in-out infinite;
  }

  .gate-icon-ring--inner {
    inset: 12px;
    background: rgba(20, 184, 166, 0.07);
    animation: gate-ring-pulse 3s ease-in-out infinite 0.5s;
  }

  @keyframes gate-ring-pulse {
    0%, 100% { transform: scale(1);    opacity: 0.8; }
    50%       { transform: scale(1.06); opacity: 0.5; }
  }

  .gate-icon {
    width: 56px;
    height: 56px;
    border-radius: var(--radius-full, 9999px);
    background: var(--chat-accent-subtle, rgba(20, 184, 166, 0.08));
    border: 1px solid var(--chat-border-accent, rgba(20, 184, 166, 0.22));
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--chat-accent, #14b8a6);
    position: relative;
    z-index: 1;
    box-shadow: 0 0 32px rgba(20, 184, 166, 0.2);
    animation: gate-icon-breathe 5s ease-in-out infinite;
    transition: box-shadow 0.4s var(--ease-out), color 0.3s, background 0.3s, border-color 0.3s;
    flex-shrink: 0;
  }

  @keyframes gate-icon-breathe {
    0%, 100% { box-shadow: 0 0 24px rgba(20, 184, 166, 0.20); }
    50%       { box-shadow: 0 0 48px rgba(20, 184, 166, 0.36); }
  }

  .gate-icon--success {
    color: var(--success-400, #34d399);
    border-color: rgba(52, 211, 153, 0.35);
    background: rgba(52, 211, 153, 0.08);
    animation: gate-icon-success 0.5s var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1));
  }

  @keyframes gate-icon-success {
    0%   { transform: scale(1); }
    40%  { transform: scale(1.2); }
    100% { transform: scale(1); }
  }

  /* ── Text ────────────────────────────────────────────────────── */
  .gate-text {
    display: flex;
    flex-direction: column;
    gap: var(--space-2, 8px);
  }

  .gate-title {
    margin: 0;
    font-size: var(--text-xl, 1.125rem);
    font-weight: 700;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.92);
    letter-spacing: -0.01em;
  }

  .gate-sub {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.52);
    line-height: var(--leading-relaxed, 1.625);
  }

  .gate-sub strong {
    color: rgba(255, 255, 255, 0.80);
    font-weight: 600;
  }

  /* ── PIN dot visualizer ──────────────────────────────────────── */
  .gate-dots {
    display: flex;
    gap: var(--space-2, 8px);
    align-items: center;
    height: 16px;
  }

  .gate-dot {
    width: 8px;
    height: 8px;
    border-radius: var(--radius-full, 9999px);
    background: rgba(255, 255, 255, 0.12);
    border: 1px solid rgba(255, 255, 255, 0.18);
    transition:
      background 0.15s var(--ease-out),
      transform  0.15s var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)),
      box-shadow 0.15s;
    flex-shrink: 0;
  }

  .gate-dot--filled,
  :global(.scv-pin-dot--filled) {
    background: var(--chat-accent, #14b8a6);
    border-color: transparent;
    box-shadow: 0 0 8px rgba(20, 184, 166, 0.5);
  }

  .gate-dot--active,
  :global(.scv-pin-dot--active) {
    transform: scale(1.3);
  }

  /* ── PIN input ───────────────────────────────────────────────── */
  .gate-input-wrap {
    width: 100%;
    max-width: 280px;
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5, 6px);
    align-items: center;
  }

  .gate-pin-input {
    width: 100%;
    padding: var(--space-4, 16px);
    background: rgba(255, 255, 255, 0.07);
    border: 1px solid var(--chat-border-accent, rgba(20, 184, 166, 0.30));
    border-radius: var(--radius-lg, 14px);
    color: rgba(255, 255, 255, 0.92);
    /* 26px > 16px iOS threshold — prevents auto-zoom, signals vault-weight */
    font-size: 26px;
    letter-spacing: 0.4em;
    text-align: center;
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    outline: none;
    transition: border-color 0.2s, box-shadow 0.2s;
    caret-color: var(--chat-accent, #14b8a6);
    -webkit-appearance: none;
    appearance: none;
    min-height: 64px;
    box-sizing: border-box;
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
  }

  .gate-pin-input::placeholder {
    color: rgba(255, 255, 255, 0.40);
    letter-spacing: 0.3em;
    font-size: 20px;
    font-family: var(--font-sans, 'Nunito', sans-serif);
  }

  .gate-pin-input:focus {
    border-color: var(--chat-accent, #14b8a6);
    box-shadow: 0 0 0 3px rgba(20, 184, 166, 0.12), inset 0 1px 0 rgba(255, 255, 255, 0.04);
  }

  @keyframes gate-shake {
    0%, 100% { transform: translateX(0); }
    15%  { transform: translateX(-8px); }
    35%  { transform: translateX(8px); }
    55%  { transform: translateX(-6px); }
    75%  { transform: translateX(5px); }
    90%  { transform: translateX(-2px); }
  }

  .gate-pin-input--shake {
    animation: gate-shake 0.48s cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
    border-color: var(--danger-400, #f87171);
    box-shadow: 0 0 0 3px rgba(248, 113, 113, 0.12);
  }

  .gate-field-err {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: var(--danger-400, #f87171);
    font-weight: 500;
  }

  .gate-hint {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.38);
  }

  /* ── Open button ─────────────────────────────────────────────── */
  .gate-btn {
    width: 100%;
    max-width: 280px;
    padding: var(--space-4, 16px);
    border-radius: var(--radius-lg, 14px);
    border: 1px solid rgba(255, 255, 255, 0.14);
    background: rgba(255, 255, 255, 0.08);
    color: rgba(255, 255, 255, 0.52);
    font-size: var(--text-base, 1rem);
    font-weight: 700;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    cursor: pointer;
    min-height: 52px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2, 8px);
    transition: background 0.2s, color 0.2s, border-color 0.2s, transform 0.1s, box-shadow 0.2s;
    touch-action: manipulation;
  }

  .gate-btn:disabled { opacity: 0.45; cursor: not-allowed; }

  .gate-btn:focus-visible {
    outline: 2px solid var(--chat-accent, #14b8a6);
    outline-offset: 2px;
  }

  .gate-btn--ready {
    background: linear-gradient(135deg, var(--primary-400, #2dd4bf) 0%, var(--primary-600, #0d9488) 100%);
    color: #fff;
    border-color: transparent;
    box-shadow: 0 4px 22px rgba(20, 184, 166, 0.42);
  }

  .gate-btn--ready:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 6px 30px rgba(20, 184, 166, 0.58);
  }

  .gate-btn--ready:active:not(:disabled) {
    transform: scale(0.97);
  }

  /* ── Button spinner ──────────────────────────────────────────── */
  .gate-btn-ring {
    width: 16px;
    height: 16px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: #fff;
    border-radius: var(--radius-full, 9999px);
    animation: gate-spin 0.7s linear infinite;
    flex-shrink: 0;
  }

  @keyframes gate-spin { to { transform: rotate(360deg); } }

  /* ── Footer note ─────────────────────────────────────────────── */
  .gate-footer {
    margin: 0;
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: rgba(255, 255, 255, 0.14);
    letter-spacing: 0.03em;
  }

  /* ── Responsive — compact phones (≤ 380px) ──────────────────── */
  @media (max-width: 380px) {
    .gate-content {
      gap: var(--space-4, 16px);
      padding: var(--space-4, 16px) var(--space-3, 12px) var(--space-6, 24px);
    }

    .gate-icon-wrap {
      width: 80px;
      height: 80px;
    }

    .gate-icon {
      width: 46px;
      height: 46px;
    }

    .gate-icon-ring--inner {
      inset: 10px;
    }

    .gate-title {
      font-size: var(--text-lg, 1rem);
    }

    .gate-pin-input {
      min-height: 56px;
      font-size: 22px;
    }

    .gate-btn {
      min-height: 48px;
      padding: var(--space-3, 12px);
    }
  }

  /* ── Responsive — landscape short screens ────────────────────── */
  @media (max-height: 600px) and (orientation: landscape) {
    .gate-content {
      gap: var(--space-3, 12px);
      padding: var(--space-2, 8px) var(--space-4, 16px) var(--space-4, 16px);
    }

    .gate-icon-wrap {
      width: 64px;
      height: 64px;
    }

    .gate-icon {
      width: 40px;
      height: 40px;
    }

    .gate-icon-ring--inner {
      inset: 8px;
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
    .gate-hex-bg    { animation: none; }
    .gate-glow      { animation: none; }
    .gate-content   { animation: none; }
    .gate-content--success { animation: none; }
    .gate-icon-ring { animation: none; }
    .gate-icon      { animation: none; }
    .gate-icon--success { animation: none; }
    .gate-dot       { transition: none; }
    .gate-pin-input--shake { animation: none; }
    .gate-btn       { transition: none; }
    .gate-btn--ready:hover { transform: none; }
    .gate-btn-ring  { animation: none; }
  }
</style>
