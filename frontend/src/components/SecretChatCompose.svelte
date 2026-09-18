<script>
  /**
   * SecretChatCompose — compose bar, emoji/sticker pickers, and attach menu.
   * Owns all local UI state for composition; dispatches semantic events to parent.
   *
   * Props:
   *   peerFirst    — first name used in placeholder
   *   sending      — parent is mid-send (disables send button)
   *   photoSending — parent is mid-upload (shows spinner on camera button)
   *   hasPin       — whether a session PIN is active (guards photo/sticker sends)
   *
   * Events dispatched:
   *   sendText   (text: string)   — user hit Send or Enter
   *   sendPhoto  (file: File)     — user chose a photo from gallery/camera
   *   sendSticker(tag: string)    — user picked a sticker
   *   panic                       — user pressed blank-screen button
   *   typing     (active: bool)   — typing indicator changed
   *
   * iOS notes:
   *   - field-sizing:content is used natively where supported (Chrome/Edge 123+,
   *     Safari 26+); on iOS Safari versions without it we fall back to the JS
   *     auto-resize approach on the textarea (autoResize skips itself when the
   *     browser supports field-sizing).
   *   - The compose footer uses padding-bottom: max(space-4, safe-area-inset-bottom)
   *     so it always clears the home indicator on iPhone.
   *   - When rendered inside SecretChatViewer, --keyboard-offset CSS var is set
   *     by the parent's VisualViewport listener and consumed here.
   *
   * aria-label note:
   *   The panic button here uses aria-label="Blank screen" (not "Blank screen for
   *   privacy") to avoid a strict-mode locator conflict with the header panic button
   *   in SecretChatViewer which carries the full label. Tests use the header button
   *   as the canonical panic trigger selector.
   */
  import { onDestroy } from 'svelte';
  import { createEventDispatcher } from 'svelte';
  import { toasts } from '../lib/stores/toast.js';
  import { haptics } from '../lib/haptics.js';
  import EmojiPicker from './primitives/EmojiPicker.svelte';
  import StickerPicker from './primitives/StickerPicker.svelte';

  /**
   * @typedef {Object} Props
   * @property {string} [peerFirst]
   * @property {boolean} [sending]
   * @property {boolean} [photoSending]
   * @property {boolean} [hasPin]
   */

  /** @type {Props} */
  let {
    peerFirst = 'them',
    sending = false,
    photoSending = false,
    hasPin = false
  } = $props();

  const dispatch = createEventDispatcher();

  let composeText = $state('');
  let composeTextEl = $state();
  let emojiOpen = $state(false);
  let emojiAnchor = $state();
  let stickerOpen = $state(false);
  let stickerAnchor = $state();
  let attachMenuOpen = $state(false);
  let photoInputEl = $state();
  let cameraInputEl = $state();
  let isTyping = false;
  let _typingTimer = null;

  onDestroy(() => { clearTimeout(_typingTimer); });

  // JS auto-resize fallback for browsers without CSS field-sizing:content
  // (notably older iOS Safari). When field-sizing is supported the CSS
  // @supports block below handles growth natively and this is a no-op.
  function autoResize(el) {
    if (!el) return;
    if ('fieldSizing' in el.style) return; // native field-sizing:content active
    el.style.height = 'auto';
    const maxH = 120;
    const scrollH = el.scrollHeight;
    el.style.height = Math.min(scrollH, maxH) + 'px';
    el.style.overflowY = scrollH > maxH ? 'auto' : 'hidden';
  }

  function send() {
    const text = composeText.trim();
    if (!text || sending) return;
    dispatch('sendText', text);
    composeText = '';
    // Reset height after clearing
    if (composeTextEl) {
      composeTextEl.style.height = '';
      composeTextEl.style.overflowY = '';
    }
    isTyping = false;
  }

  function handleKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); }
    dispatchTyping();
  }

  function handleInput(e) {
    autoResize(e.target);
    dispatchTyping();
  }

  function dispatchTyping() {
    if (!isTyping) {
      isTyping = true;
      dispatch('typing', true);
    }
    clearTimeout(_typingTimer);
    _typingTimer = setTimeout(() => {
      isTyping = false;
      dispatch('typing', false);
    }, 2000);
  }

  function toggleAttachMenu() {
    attachMenuOpen = !attachMenuOpen;
    if (attachMenuOpen) {
      const close = (e) => {
        if (!e.target.closest?.('.scc-attach-wrap')) attachMenuOpen = false;
        document.removeEventListener('click', close, true);
      };
      setTimeout(() => document.addEventListener('click', close, true), 0);
    }
  }

  function handlePhotoSelect(e) {
    const file = e.target.files?.[0];
    if (!file) return;
    e.target.value = '';
    if (!hasPin) { toasts.error('Enter your PIN before sending a photo'); return; }
    if (file.size > 15 * 1024 * 1024) { toasts.error('Photo too large — max 15 MB'); return; }
    haptics.tap?.();
    dispatch('sendPhoto', file);
  }

  function handleStickerPick(e) {
    const tag = e.detail;
    stickerOpen = false;
    if (!hasPin) { toasts.error('Enter your PIN before sending a sticker'); return; }
    haptics.tap?.();
    dispatch('sendSticker', tag);
  }

  function focusTextarea() {
    setTimeout(() => composeTextEl?.focus(), 50);
  }
</script>

<!-- ── Compose footer ───────────────────────────────────────── -->
<footer class="scc-compose scv-compose fx-glass">
  <div class="scc-compose-inner">

    <!-- Panic/blank-screen button.
         aria-label="Blank screen" (NOT "Blank screen for privacy") — the header
         panic button in SecretChatViewer carries the full label and is the canonical
         selector used by Playwright tests. Using the same label here causes a
         strict-mode locator conflict (2 elements match). -->
    <button
      class="scc-icon-btn scc-icon-btn--panic"
      onclick={() => dispatch('panic')}
      aria-label="Blank screen"
      title="Blank screen for privacy"
      type="button"
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
        <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
        <line x1="1" y1="1" x2="23" y2="23"/>
      </svg>
    </button>

    <!-- Emoji picker trigger -->
    <button
      class="scc-icon-btn"
      bind:this={emojiAnchor}
      onclick={() => { emojiOpen = !emojiOpen; stickerOpen = false; haptics.tap?.(); }}
      aria-label="Open emoji picker"
      aria-expanded={emojiOpen}
      aria-haspopup="true"
      type="button"
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <circle cx="12" cy="12" r="10"/>
        <path d="M8 14s1.5 2 4 2 4-2 4-2"/>
        <line x1="9" y1="9" x2="9.01" y2="9"/>
        <line x1="15" y1="9" x2="15.01" y2="9"/>
      </svg>
    </button>

    <!-- Sticker picker trigger -->
    <button
      class="scc-icon-btn"
      bind:this={stickerAnchor}
      onclick={() => { stickerOpen = !stickerOpen; emojiOpen = false; haptics.tap?.(); }}
      aria-label="Open sticker picker"
      aria-expanded={stickerOpen}
      aria-haspopup="true"
      type="button"
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
      </svg>
    </button>

    <!-- Hidden file inputs -->
    <input bind:this={photoInputEl} type="file" accept="image/*" style="display:none"
           onchange={handlePhotoSelect} aria-hidden="true" tabindex="-1" />
    <input bind:this={cameraInputEl} type="file" accept="image/*" capture="environment"
           style="display:none" onchange={handlePhotoSelect} aria-hidden="true" tabindex="-1" />

    <!-- Attach / camera menu -->
    <div class="scc-attach-wrap">
      <button
        class="scc-icon-btn"
        class:scc-icon-btn--loading={photoSending}
        class:scc-icon-btn--active={attachMenuOpen}
        onclick={toggleAttachMenu}
        aria-label="Send encrypted photo"
        aria-expanded={attachMenuOpen}
        aria-haspopup="menu"
        type="button"
        disabled={photoSending || !hasPin}
      >
        {#if photoSending}
          <div class="scc-mini-spinner" aria-hidden="true"></div>
        {:else}
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
            <circle cx="12" cy="13" r="4"/>
          </svg>
        {/if}
      </button>
      {#if attachMenuOpen}
        <div class="scc-attach-menu" role="menu" aria-label="Photo source">
          <button class="scc-attach-item" type="button" role="menuitem"
                  onclick={() => { attachMenuOpen = false; cameraInputEl?.click(); }}>
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
              <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
              <circle cx="12" cy="13" r="4"/>
            </svg>
            Take Photo
          </button>
          <button class="scc-attach-item" type="button" role="menuitem"
                  onclick={() => { attachMenuOpen = false; photoInputEl?.click(); }}>
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
              <circle cx="8.5" cy="8.5" r="1.5"/>
              <polyline points="21 15 16 10 5 21"/>
            </svg>
            Choose from Gallery
          </button>
        </div>
      {/if}
    </div>

    <label class="scc-sr" for="scv-reply">Secret message to {peerFirst}</label>
    <textarea
      id="scv-reply"
      class="scc-compose-text"
      rows="1"
      maxlength="2000"
      placeholder="Message {peerFirst}…"
      bind:value={composeText}
      bind:this={composeTextEl}
      onkeydown={handleKeydown}
      oninput={handleInput}
      disabled={sending}
    ></textarea>

    <button
      class="scc-send-btn scv-send-btn tactile"
      class:scc-send-btn--active={composeText.trim().length > 0}
      class:scv-send-btn--active={composeText.trim().length > 0}
      onclick={send}
      disabled={sending || !composeText.trim()}
      aria-label="Send encrypted message"
      type="button"
    >
      {#if sending}
        <div class="scc-send-ring" aria-hidden="true"></div>
      {:else}
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
          <line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
        </svg>
      {/if}
    </button>
  </div>

  <div class="scc-compose-meta">
    <p class="scc-compose-hint" aria-hidden="true">
      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
      End-to-end encrypted
    </p>
    {#if composeText.length > 1800}
      <span class="scc-char-count" class:scc-char-count--warn={composeText.length > 1950}
            aria-live="polite">{2000 - composeText.length} remaining</span>
    {/if}
  </div>
</footer>

<!-- Pickers rendered outside footer to avoid backdrop-filter containing-block bug on iOS -->
<EmojiPicker
  open={emojiOpen}
  anchor={emojiAnchor}
  on:pick={(e) => { composeText += e.detail; emojiOpen = false; focusTextarea(); }}
  on:close={() => (emojiOpen = false)}
/>
<StickerPicker
  open={stickerOpen}
  anchor={stickerAnchor}
  on:pick={handleStickerPick}
  on:close={() => (stickerOpen = false)}
/>

<style>
  /* ── Compose footer — warm paper sheet (.fx-glass resolves to the
     hearth paper tiers via tokens; no local re-tinting needed) ───── */
  footer.scc-compose {
    /* Consume keyboard-offset CSS var set by parent's VisualViewport listener.
       This moves the compose bar up by the keyboard height on iOS Chrome/Safari
       without changing the stacking context (no translateY on parent). */
    padding:
      var(--space-2-5)
      var(--space-4)
      max(var(--space-4), env(safe-area-inset-bottom, 0px));
    border-top: 1px solid var(--border-subtle);
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5);
    flex-shrink: 0;
  }

  .scc-compose-inner {
    display: flex;
    align-items: flex-end;
    gap: var(--space-1-5);
  }

  /* ── Icon buttons ─────────────────────────────────────────────── */
  .scc-icon-btn {
    width: 44px; height: 44px;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none;
    color: var(--text-tertiary);
    cursor: pointer;
    border-radius: var(--radius-input);
    flex-shrink: 0;
    transition: color var(--duration-fast) var(--ease-out), background var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }
  .scc-icon-btn:hover { color: var(--text-primary); background: var(--surface-hover); }
  .scc-icon-btn:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }
  .scc-icon-btn:disabled { opacity: 0.35; cursor: not-allowed; }
  /* Hide-screen is a privacy action, not an emergency — ochre, never red. */
  .scc-icon-btn--panic { color: var(--text-tertiary); }
  .scc-icon-btn--panic:hover { color: var(--warning-700); background: color-mix(in oklch, var(--warning-500) 10%, transparent); }
  .scc-icon-btn--loading { cursor: wait; }
  .scc-icon-btn--active { color: var(--primary-700); background: var(--primary-100); }

  .scc-mini-spinner {
    width: 15px; height: 15px;
    border: 2px solid var(--border-default);
    border-top-color: var(--primary-500);
    border-radius: var(--radius-full);
    animation: scc-spin 0.7s linear infinite;
  }

  /* ── Compose textarea — warm inset well, ember focus ring ─────── */
  .scc-compose-text {
    flex: 1;
    resize: none;
    padding: var(--space-2-5) var(--space-3);
    border-radius: var(--radius-input);
    border: 1px solid var(--border-subtle);
    background: var(--surface-inset);
    color: var(--text-primary);
    /* 16px unconditionally — iOS checks at pointerdown before media queries fire.
       Below 16px iOS Safari auto-zooms the viewport on focus, which is a poor UX. */
    font-size: 16px;
    line-height: var(--leading-relaxed);
    outline: none;
    font-family: var(--font-sans);
    transition: border-color var(--duration-fast) var(--ease-out), box-shadow var(--duration-fast) var(--ease-out);
    -webkit-appearance: none;
    /* Browsers without field-sizing (older iOS Safari): JS autoResize() handles growth */
    max-height: 120px;
    overflow-y: hidden;
    min-height: 44px;
    height: 44px;
    box-sizing: border-box;
    display: block;
  }

  /* Native content-sized textarea where supported (Baseline 2026) —
     grows with input up to max-height, no JS measurement needed. */
  @supports (field-sizing: content) {
    .scc-compose-text {
      field-sizing: content;
      height: auto;
      overflow-y: auto;
    }
  }
  .scc-compose-text:focus {
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px color-mix(in oklch, var(--primary-500) 14%, transparent);
  }
  .scc-compose-text::placeholder { color: var(--text-tertiary); }

  /* ── Send button — the one ember on this row ──────────────────── */
  .scc-send-btn {
    width: 44px; height: 44px;
    border-radius: var(--radius-input);
    border: none;
    background: var(--surface-inset);
    color: var(--text-tertiary);
    cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
    transition: transform var(--duration-fast) var(--ease-out), background var(--duration-normal) var(--ease-out), color var(--duration-normal) var(--ease-out);
    touch-action: manipulation;
  }
  .scc-send-btn--active {
    background: var(--primary-500);
    color: var(--text-on-primary);
    box-shadow: var(--shadow-primary);
  }
  .scc-send-btn--active:hover { background: var(--primary-600); }
  .scc-send-btn--active:active { transform: scale(0.93); }
  .scc-send-btn:disabled { opacity: 0.4; cursor: not-allowed; box-shadow: none; }
  .scc-send-btn:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }

  .scc-send-ring {
    width: 16px; height: 16px;
    border: 2px solid color-mix(in oklch, currentColor 30%, transparent);
    border-top-color: currentColor;
    border-radius: var(--radius-full);
    animation: scc-spin 0.7s linear infinite;
  }

  /* ── Attach menu — small paper sheet ──────────────────────────── */
  .scc-attach-wrap { position: relative; }

  .scc-attach-menu {
    position: absolute;
    bottom: calc(100% + var(--space-1-5));
    left: 50%;
    transform: translateX(-50%);
    background: var(--surface-1);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-button);
    padding: var(--space-1);
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 168px;
    box-shadow: var(--shadow-lg);
    animation: scc-pop 0.15s var(--ease-spring) both;
    z-index: 20;
  }

  .scc-attach-item {
    display: flex; align-items: center;
    gap: var(--space-2);
    padding: var(--space-2-5) var(--space-3);
    background: none; border: none;
    border-radius: var(--radius-input);
    color: var(--text-primary);
    font-size: var(--text-sm);
    font-weight: 500;
    font-family: var(--font-sans);
    cursor: pointer;
    white-space: nowrap;
    min-height: 44px;
    transition: background var(--duration-fast) var(--ease-out);
  }
  .scc-attach-item:hover { background: var(--surface-hover); }
  .scc-attach-item:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }

  /* ── Meta row ─────────────────────────────────────────────────── */
  .scc-compose-meta {
    display: flex; align-items: center; justify-content: space-between;
  }

  /* Encryption trust cue — quiet ink line, ember lock */
  .scc-compose-hint {
    display: flex; align-items: center;
    gap: var(--space-1-5);
    margin: 0;
    font-size: var(--text-xs);
    font-family: var(--font-sans);
    color: var(--text-tertiary);
  }
  .scc-compose-hint svg { color: var(--primary-700); }

  .scc-char-count {
    font-size: var(--text-xs);
    font-family: var(--font-sans);
    color: var(--text-tertiary);
    font-variant-numeric: tabular-nums;
    transition: color var(--duration-normal) var(--ease-out);
  }
  /* Running out of room needs a look, not an alarm — ochre. */
  .scc-char-count--warn { color: var(--warning-700); font-weight: 600; }

  /* ── Screen-reader only ───────────────────────────────────────── */
  .scc-sr {
    position: absolute; width: 1px; height: 1px;
    padding: 0; margin: -1px; overflow: hidden;
    clip: rect(0,0,0,0); white-space: nowrap; border: 0;
  }

  /* ── Keyframes ────────────────────────────────────────────────── */
  @keyframes scc-spin { to { transform: rotate(360deg); } }
  @keyframes scc-pop {
    from { opacity: 0; transform: translateX(-50%) scale(0.88) translateY(6px); }
    to   { opacity: 1; transform: translateX(-50%) scale(1)    translateY(0);   }
  }

  /* ── Reduced motion ───────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .scc-mini-spinner { animation: none; }
    .scc-send-ring    { animation: none; }
    .scc-send-btn--active:active { transform: none; }
    .scc-attach-menu { animation: none; }
  }
</style>
