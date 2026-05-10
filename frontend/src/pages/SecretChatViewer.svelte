<svelte:head>
  <title>Kinnect</title>
</svelte:head>

<script>
  import { onMount, tick } from 'svelte';
  import { fade } from 'svelte/transition';
  import { decryptMessage, encryptMessage } from '../lib/crypto.js';
  import EmojiPicker from '../components/primitives/EmojiPicker.svelte';
  import StickerPicker from '../components/primitives/StickerPicker.svelte';

  export let params = {};
  $: token = params.token || '';

  // 'loading' | 'gate' | 'messages' | 'error'
  let state = 'loading';
  let errorMsg = '';

  // PIN pad state
  let pinDigits = [];
  $: pin = pinDigits.join('');

  let pinError = '';
  let rawMessages = [];
  let decrypted = [];
  let unlocking = false;
  let gatePin = '';   // saved after unlock — reused for replies

  // Reply compose
  let replyText = '';
  let replyError = '';
  let replySent = false;
  let sending = false;
  let messagesEl;
  let emojiOpen = false;
  let emojiAnchor;
  let stickerOpen = false;
  let stickerAnchor;
  let panicMode = false;

  // PIN shake feedback
  let pinShake = false;
  let _pinShakeTimer = null;
  function triggerShake() {
    pinShake = true;
    clearTimeout(_pinShakeTimer);
    _pinShakeTimer = setTimeout(() => { pinShake = false; }, 520);
  }

  function restoreFromPanic() {
    panicMode = false;
    state = 'gate';
    pinDigits = [];
    pinError = '';
    gatePin = '';
    decrypted = [];
    activeDecryptId = null;
    inlinePins = {};
    inlineErrors = {};
    lockedSet = new Set();
    for (const id of Object.keys(lockIntervals)) clearInterval(lockIntervals[id]);
    lockIntervals = {};
    lockCountdowns = {};
  }

  // PIN input helpers
  let pinInputEl;

  $: if (state === 'gate' && pinInputEl) setTimeout(() => pinInputEl?.focus(), 80);
  // Sync DOM value when pinDigits is cleared from code (failed unlock, restore, etc.)
  $: if (pinInputEl && pinDigits.length === 0) pinInputEl.value = '';

  function handlePinInput(e) {
    const digits = e.target.value.replace(/\D/g, '').slice(0, 8);
    e.target.value = digits;
    pinDigits = digits.split('');
  }

  function handlePinKeydown(e) {
    if (e.key === 'Enter') { e.preventDefault(); unlock(); }
  }

  // Per-message inline decrypt
  let activeDecryptId = null;
  let inlinePins = {};
  let inlineErrors = {};
  let inlineUnlocking = {};
  let lockedSet = new Set();
  let lockCountdowns = {};
  let lockIntervals = {};

  const AUTO_LOCK_SECS = 30;

  onMount(async () => {
    if (!token) { state = 'error'; errorMsg = 'Invalid link.'; return; }
    try {
      const res = await fetch(`/api/m/${token}`);
      const data = await res.json();
      if (!data.ok) {
        state = 'error';
        errorMsg = data.expired
          ? 'This link has expired.'
          : data.error === 'temporarily_unavailable'
            ? 'Something went wrong. Try again later.'
            : 'Invalid link.';
        return;
      }
      rawMessages = data.messages || [];
      state = 'gate';
    } catch {
      state = 'error';
      errorMsg = 'Could not connect. Check your connection.';
    }
  });

  async function unlock() {
    if (unlocking || pin.length < 4) return;
    pinError = '';
    unlocking = true;

    // Validate PIN by attempting to decrypt the first owner-side message.
    const firstOwnerMsg = rawMessages.find(m => m.fromOwner);
    if (firstOwnerMsg) {
      try {
        await decryptMessage(firstOwnerMsg.ciphertext, firstOwnerMsg.iv, firstOwnerMsg.salt, pin);
      } catch {
        pinError = 'Incorrect PIN';
        unlocking = false;
        pinDigits = [];
        triggerShake();
        return;
      }
    }

    const results = rawMessages.map(m => ({
      id: m.createdAt + Math.random(),
      body: null,
      own: !m.fromOwner,
      createdAt: m.createdAt,
      raw: m.fromOwner ? m : null,
      // Store ciphertext for received messages so we can show gibberish preview
      ciphertext: m.fromOwner ? m.ciphertext : null,
    }));
    decrypted = results;
    gatePin = pin;
    pinDigits = [];
    unlocking = false;
    state = 'messages';
    await tick();
    if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
  }

  // Message grouping for decrypted list
  $: groupedDecrypted = decrypted.map((msg, i) => {
    const prev = decrypted[i - 1];
    const next = decrypted[i + 1];
    const GAP = 2 * 60 * 1000;
    const samePrev = prev && prev.own === msg.own &&
      new Date(msg.createdAt) - new Date(prev.createdAt) < GAP;
    const sameNext = next && next.own === msg.own &&
      new Date(next.createdAt) - new Date(msg.createdAt) < GAP;
    return { ...msg, groupFirst: !samePrev, groupLast: !sameNext };
  });

  // ── Per-message inline decrypt ────────────────────────────────
  function toggleInline(id) {
    activeDecryptId = activeDecryptId === id ? null : id;
    inlinePins = { ...inlinePins, [id]: '' };
    inlineErrors = { ...inlineErrors, [id]: '' };
  }

  function inlinePinInput(e, id) {
    inlinePins = { ...inlinePins, [id]: e.target.value.replace(/\D/g, '') };
  }

  async function decryptOne(msg) {
    const p = inlinePins[msg.id] ?? '';
    if (!p || p.length < 4 || inlineUnlocking[msg.id]) return;
    inlineErrors = { ...inlineErrors, [msg.id]: '' };
    inlineUnlocking = { ...inlineUnlocking, [msg.id]: true };
    try {
      const plain = await decryptMessage(msg.raw.ciphertext, msg.raw.iv, msg.raw.salt, p);
      decrypted = decrypted.map(m => m.id === msg.id ? { ...m, body: plain } : m);
      activeDecryptId = null;
      lockedSet.delete(msg.id);
      lockedSet = new Set(lockedSet);
      startAutoLock(msg.id);
    } catch {
      inlineErrors = { ...inlineErrors, [msg.id]: 'Wrong PIN' };
    } finally {
      inlineUnlocking = { ...inlineUnlocking, [msg.id]: false };
    }
  }

  function startAutoLock(id) {
    if (lockIntervals[id]) clearInterval(lockIntervals[id]);
    lockCountdowns = { ...lockCountdowns, [id]: AUTO_LOCK_SECS };
    lockIntervals[id] = setInterval(() => {
      const cur = lockCountdowns[id];
      if (cur == null || cur <= 1) {
        relockMsg(id);
      } else {
        lockCountdowns = { ...lockCountdowns, [id]: cur - 1 };
      }
    }, 1000);
  }

  function relockMsg(id) {
    if (lockIntervals[id]) { clearInterval(lockIntervals[id]); delete lockIntervals[id]; }
    delete lockCountdowns[id];
    lockCountdowns = { ...lockCountdowns };
    lockedSet = new Set([...lockedSet, id]);
    inlinePins = { ...inlinePins, [id]: '' };
    inlineErrors = { ...inlineErrors, [id]: '' };
    activeDecryptId = null;
    decrypted = decrypted.map(m => m.id === id ? { ...m, body: null } : m);
  }

  // Sticky scroll — only auto-scroll when user is near the bottom
  let userScrolledUp = false;
  let unreadWhileScrolledUp = 0;

  function handleMessagesScroll() {
    if (!messagesEl) return;
    const { scrollTop, scrollHeight, clientHeight } = messagesEl;
    userScrolledUp = scrollHeight - scrollTop - clientHeight > 60;
    if (!userScrolledUp) unreadWhileScrolledUp = 0;
  }

  function jumpToBottom() {
    if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
    userScrolledUp = false;
    unreadWhileScrolledUp = 0;
  }

  // ── Reply ─────────────────────────────────────────────────────
  async function sendReply() {
    if (sending || !replyText.trim() || gatePin.length < 4) return;
    replyError = '';
    sending = true;
    try {
      const { ciphertext, iv, salt } = await encryptMessage(replyText.trim(), gatePin);
      const res = await fetch(`/api/m/${token}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ciphertext, iv, salt }),
      });
      const data = await res.json();
      if (!data.ok) {
        replyError = data.expired ? 'Link expired.' : 'Send failed. Try again.';
        return;
      }
      decrypted = [...decrypted, { id: Date.now(), body: null, own: true, createdAt: new Date().toISOString(), ciphertext }];
      replyText = '';
      replySent = true;
      await tick();
      if (!userScrolledUp && messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
    } catch {
      replyError = 'Send failed. Check your connection.';
    } finally {
      sending = false;
    }
  }

  function handleReplyKeydown(e) { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendReply(); } }

  function clockTime(ts) {
    if (!ts) return '';
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function timeAgo(ts) {
    if (!ts) return '';
    const diff = Date.now() - new Date(ts).getTime();
    if (diff < 60_000) return 'just now';
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
    return new Date(ts).toLocaleDateString([], { month: 'short', day: 'numeric' });
  }

  const GIF_RE = /^\[gif:(https?:\/\/[^\]]+)\]$/;
  function parseGif(text) {
    if (!text) return null;
    const m = GIF_RE.exec(text);
    return m ? m[1] : null;
  }

  const PHOTO_RE = /^\[photo:(data:image\/[^;]+;base64,[^\]]+)\]$/;
  function parsePhoto(text) {
    if (!text) return null;
    const m = PHOTO_RE.exec(text);
    return m ? m[1] : null;
  }

  /**
   * Generate a plausible ciphertext noise string from a timestamp.
   * Used in the viewer where we don't always have the raw ciphertext on own messages.
   * The LCG is seeded by timestamp so the display is reproducible — same as SecretChatPanel.
   */
  function fakeGibberish(ts) {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
    const seed = new Date(ts).getTime() || Date.now();
    let s = '';
    let x = seed % 2147483647;
    for (let i = 0; i < 30; i++) {
      x = (x * 16807) % 2147483647;
      s += chars[x % 64];
    }
    return s + '…';
  }

  /**
   * Return a short excerpt of the actual base64 ciphertext for locked received messages.
   * Skips 4 chars to avoid the leading AAAA PBKDF2 prefix.
   */
  function ciphertextGibberish(ciphertext) {
    if (!ciphertext) return '···';
    const start = Math.min(4, ciphertext.length);
    const raw = ciphertext.slice(start, start + 28);
    return raw.length >= 6 ? raw + '…' : '···';
  }

  /**
   * Detect if a locked message is likely a photo — longer ciphertext due to base64 image.
   * This is a heuristic for the locked placeholder UI only.
   */
  function isLikelyPhoto(msg) {
    const ct = msg.raw?.ciphertext;
    return ct && ct.length > 5000;
  }

  // Date divider helper
  function dateLabel(ts, prevTs) {
    const d = new Date(ts);
    const p = prevTs ? new Date(prevTs) : null;
    if (p && d.toDateString() === p.toDateString()) return null;
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(today.getDate() - 1);
    if (d.toDateString() === today.toDateString()) return 'Today';
    if (d.toDateString() === yesterday.toDateString()) return 'Yesterday';
    return d.toLocaleDateString([], {
      month: 'short', day: 'numeric',
      year: d.getFullYear() !== today.getFullYear() ? 'numeric' : undefined,
    });
  }
</script>

{#if panicMode}
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
  <div class="scv-panic" on:click={restoreFromPanic}></div>
{/if}

<div class="scv">
  <!-- ── Loading ──────────────────────────────────────────────────── -->
  {#if state === 'loading'}
    <div class="scv-center">
      <div class="scv-spinner"></div>
    </div>

  <!-- ── Error ────────────────────────────────────────────────────── -->
  {:else if state === 'error'}
    <div class="scv-center">
      <div class="scv-icon-ring scv-icon-ring--err">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
          <circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
      </div>
      <p class="scv-err-txt">{errorMsg}</p>
    </div>

  <!-- ── Gate — deliberately neutral, no chat references ─────────── -->
  {:else if state === 'gate'}
    <div class="scv-gate">
      <!-- Subtle ambient glow -->
      <div class="scv-gate-glow" aria-hidden="true"></div>

      <div class="scv-gate-content">
        <div class="scv-gate-icon" aria-hidden="true">
          <!-- Neutral: key/access icon, no "chat" hint -->
          <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
          </svg>
        </div>

        <p class="scv-gate-label">Enter access code</p>

        <input
          bind:this={pinInputEl}
          class="scv-pin-field"
          class:scv-pin-field--shake={pinShake}
          type="password"
          inputmode="numeric"
          pattern="\d*"
          maxlength="8"
          placeholder="PIN"
          autocomplete="one-time-code"
          autocorrect="off"
          autocapitalize="none"
          on:input={handlePinInput}
          on:keydown={handlePinKeydown}
          aria-label="Enter access code — minimum 4 digits"
          aria-describedby={pinError ? 'scv-pin-err' : undefined}
        />

        {#if pinError}
          <p class="scv-gate-err" id="scv-pin-err" role="alert">{pinError}</p>
        {/if}

        <button
          class="scv-gate-btn"
          class:scv-gate-btn--ready={pin.length >= 4}
          on:click={unlock}
          disabled={unlocking || pin.length < 4}
        >
          {#if unlocking}
            <div class="scv-gate-spinner"></div>
          {:else}
            {pin.length >= 4 ? 'Open' : 'Continue'}
          {/if}
        </button>
      </div>
    </div>

  <!-- ── Messages ─────────────────────────────────────────────────── -->
  {:else if state === 'messages'}
    <div class="scv-view">
      <div class="scv-header">
        <!-- Back — returns to gate, keeps the invite link alive -->
        <button
          class="scv-back-btn"
          on:click={() => { state = 'gate'; gatePin = ''; decrypted = []; activeDecryptId = null; for (const id of Object.keys(lockIntervals)) clearInterval(lockIntervals[id]); lockIntervals = {}; lockCountdowns = {}; lockedSet = new Set(); pinDigits = []; pinError = ''; }}
          aria-label="Back to access code"
          type="button"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <polyline points="15 18 9 12 15 6"/>
          </svg>
        </button>
        <div class="scv-header-lock">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
        </div>
        <div class="scv-header-info">
          <span class="scv-header-title">Secret Chat</span>
          <span class="scv-header-sub">End-to-end encrypted</span>
        </div>
        <span class="scv-e2e-badge">E2E</span>
        <!-- Quick panic from header -->
        <button
          class="scv-header-panic"
          on:click={() => panicMode = true}
          aria-label="Blank screen"
          title="Blank screen"
          type="button"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
            <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
            <line x1="1" y1="1" x2="23" y2="23"/>
          </svg>
        </button>
      </div>

      <div class="scv-msgs" bind:this={messagesEl} on:scroll={handleMessagesScroll} role="log" aria-live="polite" aria-label="Secret chat messages">
        {#if groupedDecrypted.length === 0}
          <div class="scv-empty">
            <div class="scv-empty-ring" aria-hidden="true">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
              </svg>
            </div>
            <p>No messages yet</p>
            <span>Send a reply below</span>
          </div>
        {/if}

        {#each groupedDecrypted as msg, i (msg.id)}
          {@const showInline = activeDecryptId === msg.id}
          {@const isDecrypted = !msg.own && msg.body !== null && !lockedSet.has(msg.id)}
          {@const label = dateLabel(msg.createdAt, i > 0 ? groupedDecrypted[i-1].createdAt : null)}
          {@const likelyPhoto = !msg.own && !isDecrypted && isLikelyPhoto(msg)}

          {#if label}
            <div class="scv-date-div"><span>{label}</span></div>
          {/if}

          <div
            class="scv-msg"
            class:scv-msg--own={msg.own}
            class:scv-msg--their={!msg.own}
            class:scv-msg--group-cont={!msg.groupLast}
          >

            {#if msg.own}
              <div
                class="scv-bubble scv-bubble--own"
                class:scv-bubble--grp-notfirst={!msg.groupFirst}
                aria-label="Encrypted message sent"
                title="End-to-end encrypted"
              >
                <!-- Use real ciphertext if available (reply messages), else seeded gibberish -->
                <span class="scv-cipher-text" aria-hidden="true">
                  {msg.ciphertext ? ciphertextGibberish(msg.ciphertext) : fakeGibberish(msg.createdAt)}
                </span>
                <span class="scv-lock-icon" aria-hidden="true">
                  <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                </span>
              </div>
              {#if msg.groupLast}
                <time class="scv-time" datetime={msg.createdAt}>{clockTime(msg.createdAt)}</time>
              {/if}

            {:else if isDecrypted}
              <div
                class="scv-bubble scv-bubble--their scv-bubble--decrypted"
                class:scv-bubble--grp-notfirst={!msg.groupFirst}
                class:scv-bubble--photo={parsePhoto(msg.body) !== null}
              >
                {#if parsePhoto(msg.body)}
                  <img src={parsePhoto(msg.body)} class="msg-photo" alt="Encrypted photo" loading="lazy" />
                {:else if parseGif(msg.body)}
                  <img src={parseGif(msg.body)} class="msg-sticker" alt="sticker" loading="lazy" />
                {:else}
                  <p class="scv-body">{msg.body}</p>
                {/if}
                <button class="scv-relock-btn" on:click={() => relockMsg(msg.id)} aria-label="Lock message">
                  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                </button>
              </div>
              {#if msg.groupLast}
                <div class="scv-meta">
                  {#if lockCountdowns[msg.id] != null}
                    <span class="scv-countdown">Locks in {lockCountdowns[msg.id]}s</span>
                  {:else}
                    <time class="scv-time" datetime={msg.createdAt}>{clockTime(msg.createdAt)}</time>
                  {/if}
                </div>
              {/if}

            {:else}
              <button
                class="scv-bubble scv-bubble--locked"
                class:scv-bubble--locked-active={showInline}
                class:scv-bubble--grp-notfirst={!msg.groupFirst}
                class:scv-bubble--locked-photo={likelyPhoto}
                on:click={() => toggleInline(msg.id)}
                aria-expanded={showInline}
                aria-label={likelyPhoto ? 'Tap to enter PIN and decrypt photo' : 'Tap to enter PIN and decrypt'}
              >
                {#if likelyPhoto}
                  <span class="scv-photo-placeholder" aria-hidden="true">
                    <span class="scv-photo-blur-layer"></span>
                    <span class="scv-photo-cam-icon">
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                        <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
                        <circle cx="12" cy="13" r="4"/>
                      </svg>
                    </span>
                  </span>
                  <span class="scv-locked-photo-label">
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                    Photo — tap to unlock
                  </span>
                {:else}
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                  {#if msg.raw?.ciphertext}
                    <span class="scv-cipher-preview" aria-hidden="true">{ciphertextGibberish(msg.raw.ciphertext)}</span>
                  {:else}
                    <span>Tap to read</span>
                  {/if}
                  <span class="scv-ago">{timeAgo(msg.createdAt)}</span>
                {/if}
              </button>

              {#if showInline}
                <!-- transition:fade matches SecretChatPanel behavior — smooth expand/collapse -->
                <div class="scv-inline-decrypt" transition:fade={{ duration: 100 }}>
                  <label class="scv-sr" for="scv-ipin-{msg.id}">PIN</label>
                  <input
                    id="scv-ipin-{msg.id}"
                    class="scv-inline-pin"
                    type="password"
                    inputmode="numeric"
                    pattern="[0-9]*"
                    maxlength="8"
                    placeholder="Sender's PIN"
                    value={inlinePins[msg.id] ?? ''}
                    on:input={(e) => inlinePinInput(e, msg.id)}
                    on:keydown={(e) => e.key === 'Enter' && decryptOne(msg)}
                    disabled={inlineUnlocking[msg.id]}
                    autocomplete="off"
                    autofocus
                  />
                  <button
                    class="scv-inline-btn"
                    on:click={() => decryptOne(msg)}
                    disabled={inlineUnlocking[msg.id] || (inlinePins[msg.id] ?? '').length < 4}
                  >
                    {inlineUnlocking[msg.id] ? '…' : 'Read'}
                  </button>
                  {#if inlineErrors[msg.id]}
                    <span class="scv-inline-err" role="alert">{inlineErrors[msg.id]}</span>
                  {/if}
                </div>
              {/if}

              {#if msg.groupLast}
                <time class="scv-time" datetime={msg.createdAt}>{clockTime(msg.createdAt)}</time>
              {/if}
            {/if}
          </div>
        {/each}

        {#if replySent}
          <div class="scv-sent-notice">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#4ade80" stroke-width="2.5" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
            Reply sent securely
          </div>
        {/if}
      </div>

      <!-- Scroll-to-bottom FAB -->
      {#if userScrolledUp}
        <button
          class="scv-scroll-fab"
          on:click={jumpToBottom}
          aria-label="Jump to latest messages"
          type="button"
          transition:fade={{ duration: 120 }}
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </button>
      {/if}

      <!-- Compose -->
      <div class="scv-compose">
        <div class="scv-compose-inner">
          <button
            class="scv-compose-icon-btn"
            bind:this={emojiAnchor}
            on:click={() => { emojiOpen = !emojiOpen; stickerOpen = false; }}
            aria-label="Emoji picker"
            type="button"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <circle cx="12" cy="12" r="10"/>
              <path d="M8 14s1.5 2 4 2 4-2 4-2"/>
              <line x1="9" y1="9" x2="9.01" y2="9"/>
              <line x1="15" y1="9" x2="15.01" y2="9"/>
            </svg>
          </button>

          <!-- Sticker button -->
          <button
            class="scv-compose-icon-btn"
            bind:this={stickerAnchor}
            on:click={() => { stickerOpen = !stickerOpen; emojiOpen = false; }}
            aria-label="Sticker picker"
            aria-expanded={stickerOpen}
            title="Stickers"
            type="button"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
            </svg>
          </button>

          <label class="scv-sr" for="scv-reply">Reply</label>
          <textarea
            id="scv-reply"
            class="scv-compose-text"
            rows="1"
            maxlength="2000"
            placeholder="Type a secret reply…"
            bind:value={replyText}
            on:keydown={handleReplyKeydown}
            disabled={sending}
          ></textarea>
          <button
            class="scv-send-btn"
            on:click={sendReply}
            disabled={sending || !replyText.trim()}
            aria-label="Send reply"
          >
            {#if sending}
              <div class="scv-send-ring"></div>
            {:else}
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
            {/if}
          </button>
        </div>
        <EmojiPicker
          open={emojiOpen}
          anchor={emojiAnchor}
          on:pick={(e) => { replyText += e.detail; }}
          on:close={() => emojiOpen = false}
        />
        <StickerPicker
          open={stickerOpen}
          anchor={stickerAnchor}
          on:pick={(e) => { replyText += e.detail; stickerOpen = false; }}
          on:close={() => stickerOpen = false}
        />

        {#if replyError}
          <p class="scv-reply-err" role="alert">{replyError}</p>
        {/if}
        <p class="scv-compose-hint">
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          End-to-end encrypted
        </p>
      </div>
    </div>
  {/if}
</div>

<style>
  .scv {
    min-height: 100dvh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #07070f;
    box-sizing: border-box;
    font-family: system-ui, sans-serif;
    position: relative;
    overflow: hidden;
  }
  /* Animated gradient mesh behind all content */
  .scv::before {
    content: '';
    position: fixed;
    inset: 0;
    background:
      radial-gradient(ellipse 70% 55% at 20% 10%, rgba(99,102,241,0.1) 0%, transparent 60%),
      radial-gradient(ellipse 55% 45% at 80% 90%, rgba(168,85,247,0.08) 0%, transparent 55%),
      radial-gradient(ellipse 45% 40% at 60% 40%, rgba(59,130,246,0.04) 0%, transparent 55%);
    pointer-events: none;
    z-index: 0;
    animation: scv-mesh-shift 14s ease-in-out infinite alternate;
  }
  @keyframes scv-mesh-shift {
    0%   { opacity: 1; }
    50%  { opacity: 0.6; }
    100% { opacity: 1; }
  }
  /* Ensure all children sit above mesh */
  .scv > * { position: relative; z-index: 1; }

  /* ── Loading / error ───────────────────────────────────────────── */
  .scv-center { display: flex; flex-direction: column; align-items: center; gap: 16px; }
  .scv-spinner { width: 28px; height: 28px; border: 2px solid rgba(255,255,255,0.07); border-top-color: rgba(129,140,248,0.7); border-radius: 50%; animation: scv-spin 0.8s linear infinite; }
  .scv-icon-ring--err { width: 52px; height: 52px; border-radius: 50%; background: rgba(248,113,113,0.08); border: 1px solid rgba(248,113,113,0.2); display: flex; align-items: center; justify-content: center; color: #f87171; }
  .scv-err-txt { color: rgba(255,255,255,0.45); font-size: 14px; text-align: center; margin: 0; max-width: 240px; }

  /* ── PIN field ───────────────────────────────────────────────────── */
  .scv-pin-field {
    width: 100%;
    padding: 14px 18px;
    background: rgba(255,255,255,0.05);
    border: 1px solid rgba(129,140,248,0.25);
    border-radius: 13px;
    color: #e2e8f0;
    font-size: 22px;
    letter-spacing: 0.3em;
    text-align: center;
    font-family: system-ui, sans-serif;
    outline: none;
    transition: border-color 0.15s, box-shadow 0.15s;
    caret-color: #818cf8;
    -webkit-appearance: none;
    appearance: none;
  }
  .scv-pin-field::placeholder { color: rgba(255,255,255,0.2); letter-spacing: 0.05em; font-size: 15px; }
  .scv-pin-field:focus {
    border-color: rgba(129,140,248,0.6);
    box-shadow: 0 0 0 3px rgba(99,102,241,0.18);
  }
  @keyframes scv-shake {
    0%, 100% { transform: translateX(0); }
    15%      { transform: translateX(-7px); }
    35%      { transform: translateX(7px); }
    55%      { transform: translateX(-5px); }
    75%      { transform: translateX(4px); }
    90%      { transform: translateX(-2px); }
  }
  .scv-pin-field--shake { animation: scv-shake 0.48s cubic-bezier(.36,.07,.19,.97) both; }

  /* ── Gate — full-screen, atmospheric, deliberately neutral ─────── */
  .scv-gate {
    position: relative;
    width: 100%;
    min-height: 100dvh;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
  }

  /* Ambient glow — makes page feel alive */
  .scv-gate-glow {
    position: absolute;
    inset: 0;
    background:
      radial-gradient(ellipse 65% 45% at 50% 30%, rgba(99,102,241,0.1) 0%, transparent 70%),
      radial-gradient(ellipse 50% 40% at 75% 75%, rgba(168,85,247,0.07) 0%, transparent 60%);
    pointer-events: none;
    animation: scv-glow-breathe 8s ease-in-out infinite;
  }
  @keyframes scv-glow-breathe {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0.65; }
  }

  .scv-gate-content {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 20px;
    width: 100%;
    max-width: 280px;
    padding: 32px 16px;
  }

  .scv-gate-icon {
    width: 68px; height: 68px;
    border-radius: 50%;
    background: rgba(99,102,241,0.09);
    border: 1px solid rgba(129,140,248,0.22);
    box-shadow: 0 0 0 10px rgba(99,102,241,0.04), 0 0 36px rgba(99,102,241,0.14);
    display: flex; align-items: center; justify-content: center;
    color: rgba(165,180,252,0.8);
    animation: scv-icon-breathe 5s ease-in-out infinite;
  }
  @keyframes scv-icon-breathe {
    0%, 100% { box-shadow: 0 0 0 10px rgba(99,102,241,0.04), 0 0 36px rgba(99,102,241,0.14); }
    50%       { box-shadow: 0 0 0 16px rgba(99,102,241,0.02), 0 0 52px rgba(99,102,241,0.2); }
  }

  .scv-gate-label {
    margin: 0;
    font-size: 13px;
    color: rgba(255,255,255,0.3);
    letter-spacing: 0.04em;
    font-family: system-ui, sans-serif;
  }

  .scv-gate-err { color: #f87171; font-size: 12px; margin: 0; }

  .scv-gate-btn {
    width: 100%;
    padding: 13px;
    border-radius: 13px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(255,255,255,0.06);
    color: rgba(255,255,255,0.6);
    font-size: 14px; font-weight: 500;
    cursor: pointer;
    min-height: 48px;
    transition: background 0.15s, color 0.15s, border-color 0.15s;
    touch-action: manipulation;
  }
  .scv-gate-btn:hover:not(:disabled) { background: rgba(255,255,255,0.1); color: rgba(255,255,255,0.85); border-color: rgba(255,255,255,0.18); }
  .scv-gate-btn:disabled { opacity: 0.3; cursor: not-allowed; }

  /* ── Messages view ─────────────────────────────────────────────── */
  .scv-view { width: 100%; max-width: 500px; height: 100dvh; display: flex; flex-direction: column; background: transparent; }

  .scv-header {
    display: flex; align-items: center; gap: 10px;
    padding: 13px 12px 13px 16px;
    background: rgba(10,10,18,0.7);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border-bottom: 1px solid rgba(129,140,248,0.1);
    flex-shrink: 0;
  }
  .scv-header-lock { color: rgba(129,140,248,0.75); display: flex; align-items: center; flex-shrink: 0; }
  .scv-header-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
  .scv-header-title {
    font-size: 14px; font-weight: 700;
    background: linear-gradient(110deg, #c7d2fe 0%, #a5b4fc 40%, #818cf8 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }
  .scv-header-sub { font-size: 11px; color: rgba(255,255,255,0.35); }
  .scv-e2e-badge {
    font-size: 9px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase;
    color: rgba(165,180,252,0.75);
    background: rgba(99,102,241,0.1);
    border: 1px solid rgba(129,140,248,0.22);
    padding: 2px 7px; border-radius: 20px; white-space: nowrap; flex-shrink: 0;
    box-shadow: 0 0 10px rgba(99,102,241,0.08);
  }

  .scv-back-btn {
    width: 36px; height: 36px;
    min-width: 44px; min-height: 44px;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none;
    color: rgba(255,255,255,0.45);
    cursor: pointer;
    border-radius: 8px;
    flex-shrink: 0;
    transition: color 0.15s, background 0.15s;
    touch-action: manipulation;
  }
  .scv-back-btn:hover { color: rgba(255,255,255,0.8); background: rgba(255,255,255,0.06); }

  .scv-header-panic {
    width: 36px; height: 36px;
    min-width: 44px; min-height: 44px;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none;
    color: rgba(255,255,255,0.18);
    cursor: pointer;
    border-radius: 8px;
    flex-shrink: 0;
    transition: color 0.15s, background 0.15s;
    touch-action: manipulation;
  }
  .scv-header-panic:hover { color: rgba(248,113,113,0.55); background: rgba(248,113,113,0.06); }

  .scv-msgs { flex: 1; overflow-y: auto; padding: 14px 14px 10px; display: flex; flex-direction: column; gap: 4px; overscroll-behavior: contain; }
  .scv-msgs::-webkit-scrollbar { width: 3px; }
  .scv-msgs::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.08); border-radius: 2px; }

  .scv-empty {
    flex: 1;
    display: flex; flex-direction: column;
    align-items: center; justify-content: center;
    gap: 12px;
    padding: 40px 0;
    text-align: center;
  }

  .scv-empty-ring {
    width: 52px; height: 52px;
    border-radius: 50%;
    background: rgba(129,140,248,0.07);
    border: 1px solid rgba(129,140,248,0.14);
    display: flex; align-items: center; justify-content: center;
    color: rgba(129,140,248,0.45);
  }

  .scv-empty p { margin: 0; font-size: 14px; color: rgba(255,255,255,0.28); }
  .scv-empty span { font-size: 12px; color: rgba(255,255,255,0.18); }

  /* ── Message rows ──────────────────────────────────────────────── */
  .scv-msg { display: flex; flex-direction: column; max-width: 78%; gap: 2px; margin-bottom: 4px; }
  .scv-msg--group-cont { margin-bottom: 1px; }
  .scv-msg--own { align-self: flex-end; align-items: flex-end; }
  .scv-msg--their { align-self: flex-start; align-items: flex-start; }

  .scv-bubble { padding: 9px 13px; border-radius: 18px; font-size: 14px; line-height: 1.55; }
  .scv-bubble--own {
    display: flex; align-items: center; gap: 8px;
    background: linear-gradient(135deg, rgba(99,102,241,0.22) 0%, rgba(129,140,248,0.14) 100%);
    border: 1px solid rgba(129,140,248,0.28);
    border-bottom-right-radius: 5px;
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    box-shadow:
      inset 0 1px 0 rgba(255,255,255,0.08),
      0 2px 8px rgba(0,0,0,0.2),
      0 0 0 1px rgba(99,102,241,0.08);
    max-width: 100%;
    overflow: hidden;
  }
  .scv-bubble--own.scv-bubble--grp-notfirst { border-top-right-radius: 5px; }

  .scv-bubble--their {
    background: rgba(255,255,255,0.055);
    border: 1px solid rgba(255,255,255,0.08);
    border-bottom-left-radius: 5px;
    color: #e2e8f0;
    word-break: break-word;
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
    box-shadow: inset 0 1px 0 rgba(255,255,255,0.05), 0 1px 6px rgba(0,0,0,0.12);
  }
  .scv-bubble--their.scv-bubble--grp-notfirst,
  .scv-bubble--locked.scv-bubble--grp-notfirst {
    border-top-left-radius: 5px;
  }

  .scv-bubble--decrypted { position: relative; padding-right: 30px; }

  /* Photo decrypted bubble — edge-to-edge image */
  .scv-bubble--decrypted.scv-bubble--photo {
    padding: 4px;
    overflow: hidden;
  }

  .scv-bubble--locked {
    display: flex; align-items: center; gap: 8px;
    background: rgba(255,255,255,0.03);
    border: 1px dashed rgba(255,255,255,0.1);
    border-bottom-left-radius: 4px;
    color: rgba(255,255,255,0.28);
    font-size: 13px;
    cursor: pointer;
    min-height: 44px;
    position: relative;
    touch-action: manipulation;
    transition: background 0.15s, border-color 0.15s;
    overflow: hidden;
  }
  .scv-bubble--locked:hover { background: rgba(255,255,255,0.06); border-color: rgba(255,255,255,0.18); }
  .scv-bubble--locked-active { border-color: rgba(129,140,248,0.35); background: rgba(129,140,248,0.06); }

  /* Photo locked state */
  .scv-bubble--locked-photo {
    flex-direction: column;
    align-items: flex-start;
    padding: 0;
    border-style: solid;
    min-height: 80px;
    border-radius: 12px;
    border-bottom-left-radius: 4px;
  }

  .scv-photo-placeholder {
    position: relative;
    width: 180px;
    height: 120px;
    display: block;
    overflow: hidden;
    flex-shrink: 0;
  }

  .scv-photo-blur-layer {
    position: absolute;
    inset: 0;
    background:
      radial-gradient(ellipse 60% 50% at 30% 40%, rgba(99,102,241,0.18) 0%, transparent 60%),
      radial-gradient(ellipse 40% 55% at 70% 60%, rgba(168,85,247,0.12) 0%, transparent 55%),
      radial-gradient(ellipse 80% 40% at 50% 50%, rgba(59,130,246,0.1) 0%, transparent 70%),
      linear-gradient(135deg, rgba(255,255,255,0.04) 0%, rgba(0,0,0,0.25) 100%);
    filter: blur(8px);
  }

  .scv-photo-cam-icon {
    position: absolute;
    inset: 0;
    display: flex; align-items: center; justify-content: center;
    color: rgba(255,255,255,0.55);
  }

  .scv-locked-photo-label {
    display: flex; align-items: center; gap: 6px;
    padding: 7px 12px;
    font-size: 11px;
    color: rgba(255,255,255,0.35);
    font-family: system-ui, sans-serif;
    border-top: 1px solid rgba(255,255,255,0.06);
    width: 100%;
    box-sizing: border-box;
    flex-shrink: 0;
  }

  /* Ciphertext preview in locked received bubbles */
  .scv-cipher-preview {
    font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace;
    font-size: 10px;
    letter-spacing: 0.04em;
    color: rgba(165,180,252,0.32);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
    user-select: none;
  }

  .scv-relock-btn {
    position: absolute; top: 4px; right: 4px;
    width: 22px; height: 22px;
    padding: 11px; margin: -11px;
    display: flex; align-items: center; justify-content: center;
    background: rgba(0,0,0,0.35);
    border: none; cursor: pointer;
    color: rgba(255,255,255,0.45);
    border-radius: 8px;
    touch-action: manipulation;
    box-sizing: content-box;
    transition: color 0.15s, background 0.15s;
  }
  .scv-relock-btn:hover { color: rgba(129,140,248,0.9); background: rgba(129,140,248,0.18); }

  .scv-body { margin: 0; }
  .scv-ago { margin-left: auto; font-size: 10px; color: rgba(255,255,255,0.2); white-space: nowrap; flex-shrink: 0; }
  .scv-meta { display: flex; align-items: center; gap: 4px; padding: 0 2px; }
  .scv-time { font-size: 10px; color: rgba(255,255,255,0.22); padding: 0 2px; }
  .scv-countdown { font-size: 10px; color: rgba(129,140,248,0.5); font-variant-numeric: tabular-nums; padding: 0 2px; }

  /* Cipher text on own bubbles */
  .scv-cipher-text {
    font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace;
    font-size: 11px;
    letter-spacing: 0.05em;
    color: rgba(165,180,252,0.5);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
    user-select: none;
  }
  .scv-lock-icon {
    color: rgba(129,140,248,0.4);
    flex-shrink: 0;
    display: flex;
    align-items: center;
  }

  /* Inline decrypt — P0 fix: type="password" with inputmode="numeric" */
  .scv-inline-decrypt {
    display: flex; align-items: center; gap: 8px; flex-wrap: nowrap;
    margin-top: 4px;
    padding: 8px 10px;
    background: rgba(129,140,248,0.06);
    border: 1px solid rgba(129,140,248,0.14);
    border-radius: 12px;
    width: 100%; max-width: 100%;
    box-sizing: border-box;
  }
  .scv-inline-pin {
    flex: 1; min-width: 0;
    padding: 10px;
    border-radius: 8px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(0,0,0,0.3);
    color: #e2e8f0;
    font-size: 16px;
    letter-spacing: 0.25em;
    text-align: center;
    outline: none;
    -webkit-appearance: none;
    min-height: 44px;
    touch-action: manipulation;
    transition: border-color 0.15s;
  }
  .scv-inline-pin:focus { border-color: rgba(129,140,248,0.5); }
  @media (max-width: 767px) { .scv-inline-pin { font-size: 18px; } }
  .scv-inline-btn {
    padding: 10px 16px;
    border-radius: 8px; border: none;
    background: rgba(129,140,248,0.75);
    color: #fff;
    font-size: 13px; font-weight: 600;
    cursor: pointer;
    min-height: 44px;
    flex-shrink: 0;
    white-space: nowrap;
    touch-action: manipulation;
    transition: background 0.15s;
  }
  .scv-inline-btn:hover:not(:disabled) { background: #818cf8; }
  .scv-inline-btn:disabled { opacity: 0.3; cursor: not-allowed; }
  .scv-inline-err { font-size: 11px; color: #f87171; width: 100%; }

  .scv-sent-notice { display: flex; align-items: center; gap: 6px; font-size: 11px; color: #4ade80; align-self: center; margin-top: 4px; }

  /* Compose */
  .scv-compose {
    padding: 10px 14px 14px;
    border-top: 1px solid rgba(129,140,248,0.08);
    display: flex; flex-direction: column;
    gap: 7px;
    flex-shrink: 0;
    background: rgba(7,7,15,0.8);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    padding-bottom: calc(14px + env(safe-area-inset-bottom, 0px));
  }
  .scv-compose-inner { display: flex; align-items: flex-end; gap: 8px; }

  .scv-compose-icon-btn {
    width: 36px; height: 36px;
    min-width: 44px; min-height: 44px;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none;
    color: rgba(255,255,255,0.28);
    cursor: pointer;
    border-radius: 10px;
    flex-shrink: 0;
    transition: color 0.15s, background 0.15s;
    touch-action: manipulation;
  }
  .scv-compose-icon-btn:hover { color: rgba(255,255,255,0.65); background: rgba(255,255,255,0.06); }

  .scv-compose-text {
    flex: 1; resize: none;
    padding: 10px 12px;
    border-radius: 14px;
    border: 1px solid rgba(255,255,255,0.09);
    background: rgba(255,255,255,0.04);
    color: #e2e8f0;
    font-size: 14px; line-height: 1.5;
    outline: none;
    font-family: system-ui, sans-serif;
    -webkit-appearance: none;
    max-height: 120px; min-height: 44px;
    box-sizing: border-box; width: 100%;
    transition: border-color 0.15s;
  }
  .scv-compose-text:focus {
    border-color: rgba(129,140,248,0.4);
    box-shadow: 0 0 0 3px rgba(99,102,241,0.1);
  }
  .scv-compose-text::placeholder { color: rgba(255,255,255,0.18); }
  @media (max-width: 767px) { .scv-compose-text { font-size: 16px; } }

  .scv-send-btn {
    width: 44px; height: 44px;
    border-radius: 13px; border: none;
    background: linear-gradient(135deg, #818cf8 0%, #6366f1 100%);
    color: #fff;
    cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
    transition: transform 0.12s, box-shadow 0.15s;
    touch-action: manipulation;
    box-shadow: 0 2px 12px rgba(99,102,241,0.4);
  }
  .scv-send-btn:hover:not(:disabled) {
    transform: scale(1.06);
    box-shadow: 0 4px 20px rgba(99,102,241,0.6);
  }
  .scv-send-btn:active:not(:disabled) {
    transform: scale(0.93);
    box-shadow: 0 1px 6px rgba(99,102,241,0.3);
  }
  .scv-send-btn:disabled { opacity: 0.28; cursor: not-allowed; box-shadow: none; }
  .scv-send-ring { width: 16px; height: 16px; border: 2px solid rgba(255,255,255,0.25); border-top-color: #fff; border-radius: 50%; animation: scv-spin 0.8s linear infinite; }

  .scv-compose-hint { display: flex; align-items: center; gap: 5px; margin: 0; font-size: 10px; color: rgba(255,255,255,0.16); }
  .scv-reply-err { margin: 0; font-size: 12px; color: #f87171; }

  /* Panic overlay */
  .scv-panic {
    position: fixed; inset: 0;
    background: #fff;
    z-index: 99999;
    cursor: default;
  }

  :global(.msg-sticker) { max-width: 120px; max-height: 120px; border-radius: 8px; display: block; }
  :global(.msg-photo) { max-width: 220px; max-height: 260px; border-radius: 10px; display: block; object-fit: cover; }

  .scv-sr { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
  @keyframes scv-spin { to { transform: rotate(360deg); } }

  /* ── Gate button CTA state (4+ digits entered) ──────────────── */
  .scv-gate-btn--ready {
    background: linear-gradient(135deg, #818cf8 0%, #6366f1 60%, #7c3aed 100%);
    color: #fff;
    border-color: transparent;
    font-weight: 600;
    box-shadow: 0 4px 20px rgba(99,102,241,0.4);
  }
  .scv-gate-btn--ready:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 6px 28px rgba(99,102,241,0.55);
  }
  .scv-gate-btn--ready:active:not(:disabled) {
    transform: scale(0.97);
    box-shadow: 0 2px 12px rgba(99,102,241,0.3);
  }
  .scv-gate-spinner {
    width: 16px; height: 16px;
    border: 2px solid rgba(255,255,255,0.25);
    border-top-color: #fff;
    border-radius: 50%;
    animation: scv-spin 0.8s linear infinite;
    margin: 0 auto;
  }

  /* ── Message slide-in animation (3D tilt entry) ─────────────── */
  @keyframes scv-msg-in {
    from {
      opacity: 0;
      transform: perspective(500px) rotateX(9deg) translateY(10px) scale(0.97);
    }
    to {
      opacity: 1;
      transform: perspective(500px) rotateX(0deg) translateY(0) scale(1);
    }
  }
  .scv-msg {
    animation: scv-msg-in 0.22s cubic-bezier(0.2, 0.8, 0.3, 1) both;
    transform-origin: bottom center;
  }

  /* ── Date divider ───────────────────────────────────────────── */
  .scv-date-div {
    display: flex;
    align-items: center;
    gap: 10px;
    margin: 10px 0 6px;
    align-self: stretch;
    animation: scv-msg-in 0.18s ease-out both;
  }
  .scv-date-div::before,
  .scv-date-div::after {
    content: '';
    flex: 1;
    height: 1px;
    background: rgba(255,255,255,0.06);
  }
  /* ── Scroll-to-bottom FAB ───────────────────────────────────── */
  .scv-scroll-fab {
    align-self: flex-end;
    margin: -8px 12px 0;
    position: relative;
    width: 36px; height: 36px;
    border-radius: 50%;
    border: 1px solid rgba(129,140,248,0.35);
    background: rgba(15,15,28,0.85);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    color: rgba(165,180,252,0.9);
    box-shadow: 0 4px 16px rgba(0,0,0,0.5), 0 0 12px rgba(99,102,241,0.12);
    display: flex; align-items: center; justify-content: center;
    cursor: pointer;
    touch-action: manipulation;
    transition: background 0.15s, transform 0.15s, box-shadow 0.15s;
    z-index: 10;
    flex-shrink: 0;
  }
  .scv-scroll-fab:hover {
    background: rgba(99,102,241,0.15);
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(0,0,0,0.5), 0 0 20px rgba(99,102,241,0.2);
  }

  .scv-date-div span {
    font-size: 10px;
    color: rgba(255,255,255,0.22);
    white-space: nowrap;
    font-family: system-ui, sans-serif;
    letter-spacing: 0.03em;
    padding: 2px 4px;
  }

  @media (prefers-reduced-motion: reduce) {
    /* Disable all animations — includes background mesh, gate icon, message entry */
    .scv::before { animation: none; }
    .scv-msg { animation: none; }
    .scv-date-div { animation: none; }
    .scv-spinner { animation: none; }
    .scv-send-ring { animation: none; }
    .scv-gate-spinner { animation: none; }
    .scv-gate-glow { animation: none; }
    .scv-gate-icon { animation: none; }
  }
</style>
