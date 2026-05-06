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

  // PIN pad helpers
  function addPinDigit(d) {
    if (pinDigits.length >= 8 || unlocking) return;
    pinDigits = [...pinDigits, d];
  }

  function removePinDigit() {
    if (!pinDigits.length) return;
    pinDigits = pinDigits.slice(0, -1);
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
    // Without this check, any 4+ digit PIN would silently pass the gate and
    // replies would be encrypted with the wrong PIN.
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
      decrypted = [...decrypted, { id: Date.now(), body: null, own: true, createdAt: new Date().toISOString() }];
      replyText = '';
      replySent = true;
      await tick();
      if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
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

        <!-- PIN dot indicators -->
        <div class="scv-pin-dots" class:scv-pin-dots--shake={pinShake} aria-live="polite" aria-label="{pinDigits.length} digit{pinDigits.length === 1 ? '' : 's'} entered">
          {#each {length: Math.max(4, pinDigits.length)} as _, i}
            <div class="scv-pin-dot" class:scv-pin-dot--filled={i < pinDigits.length}></div>
          {/each}
        </div>

        {#if pinError}
          <p class="scv-gate-err" role="alert">{pinError}</p>
        {/if}

        <!-- Number pad -->
        <div class="scv-numpad" role="group" aria-label="PIN keypad">
          {#each [1,2,3,4,5,6,7,8,9] as d}
            <button
              class="scv-numpad-key"
              on:click={() => addPinDigit(String(d))}
              type="button"
              disabled={unlocking}
              aria-label={String(d)}
            >{d}</button>
          {/each}
          <div class="scv-numpad-spacer" aria-hidden="true"></div>
          <button
            class="scv-numpad-key"
            on:click={() => addPinDigit('0')}
            type="button"
            disabled={unlocking}
            aria-label="0"
          >0</button>
          <button
            class="scv-numpad-key scv-numpad-key--back"
            on:click={removePinDigit}
            type="button"
            disabled={unlocking || pinDigits.length === 0}
            aria-label="Backspace"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M21 4H8l-7 8 7 8h13a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2z"/>
              <line x1="18" y1="9" x2="12" y2="15"/><line x1="12" y1="9" x2="18" y2="15"/>
            </svg>
          </button>
        </div>

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

      <div class="scv-msgs" bind:this={messagesEl}>
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
              >
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                <span>Sent · encrypted</span>
              </div>
              {#if msg.groupLast}
                <span class="scv-time">{clockTime(msg.createdAt)}</span>
              {/if}

            {:else if isDecrypted}
              <div
                class="scv-bubble scv-bubble--their scv-bubble--decrypted"
                class:scv-bubble--grp-notfirst={!msg.groupFirst}
              >
                {#if parseGif(msg.body)}
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
                    <span class="scv-time">{clockTime(msg.createdAt)}</span>
                  {/if}
                </div>
              {/if}

            {:else}
              <button
                class="scv-bubble scv-bubble--locked"
                class:scv-bubble--locked-active={showInline}
                class:scv-bubble--grp-notfirst={!msg.groupFirst}
                on:click={() => toggleInline(msg.id)}
                aria-expanded={showInline}
                aria-label="Tap to enter PIN and decrypt"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                <span>Tap to read</span>
                <span class="scv-ago">{timeAgo(msg.createdAt)}</span>
              </button>

              {#if showInline}
                <div class="scv-inline-decrypt">
                  <label class="scv-sr" for="scv-ipin-{msg.id}">PIN</label>
                  <input
                    id="scv-ipin-{msg.id}"
                    class="scv-inline-pin"
                    type="tel"
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
                <span class="scv-time">{clockTime(msg.createdAt)}</span>
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

          <!-- Sticker button — distinct star icon -->
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
    background: #0d0d14;
    box-sizing: border-box;
    font-family: system-ui, sans-serif;
  }

  /* ── Loading / error ───────────────────────────────────────────── */
  .scv-center { display: flex; flex-direction: column; align-items: center; gap: 16px; }
  .scv-spinner { width: 28px; height: 28px; border: 2px solid rgba(255,255,255,0.07); border-top-color: rgba(129,140,248,0.7); border-radius: 50%; animation: scv-spin 0.8s linear infinite; }
  .scv-icon-ring--err { width: 52px; height: 52px; border-radius: 50%; background: rgba(248,113,113,0.08); border: 1px solid rgba(248,113,113,0.2); display: flex; align-items: center; justify-content: center; color: #f87171; }
  .scv-err-txt { color: rgba(255,255,255,0.45); font-size: 14px; text-align: center; margin: 0; max-width: 240px; }

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

  /* Subtle ambient glow — makes page feel intentional, not broken */
  .scv-gate-glow {
    position: absolute;
    inset: 0;
    background:
      radial-gradient(ellipse 60% 40% at 50% 35%, rgba(129,140,248,0.06) 0%, transparent 70%),
      radial-gradient(ellipse 40% 30% at 70% 70%, rgba(99,102,241,0.04) 0%, transparent 60%);
    pointer-events: none;
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
    width: 56px; height: 56px;
    border-radius: 50%;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.09);
    display: flex; align-items: center; justify-content: center;
    color: rgba(255,255,255,0.25);
  }

  .scv-gate-label {
    margin: 0;
    font-size: 13px;
    color: rgba(255,255,255,0.3);
    letter-spacing: 0.04em;
    font-family: system-ui, sans-serif;
  }

  /* ── PIN dots (viewer) ──────────────────────────────────────────── */
  .scv-pin-dots {
    display: flex;
    gap: 14px;
    justify-content: center;
    height: 20px;
    align-items: center;
  }

  .scv-pin-dot {
    width: 13px; height: 13px;
    border-radius: 50%;
    border: 2px solid rgba(255,255,255,0.18);
    background: transparent;
    transition: background 0.12s, border-color 0.12s, transform 0.1s;
    flex-shrink: 0;
  }

  .scv-pin-dot--filled {
    background: rgba(255,255,255,0.5);
    border-color: rgba(255,255,255,0.5);
    transform: scale(1.1);
  }

  /* ── Number pad (viewer) ────────────────────────────────────────── */
  .scv-numpad {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 10px;
    width: 100%;
    max-width: 240px;
  }

  .scv-numpad-spacer { height: 52px; }

  .scv-numpad-key {
    height: 52px;
    border-radius: 14px;
    border: 1px solid rgba(255,255,255,0.07);
    background: rgba(255,255,255,0.04);
    color: rgba(255,255,255,0.7);
    font-size: 20px;
    font-weight: 400;
    cursor: pointer;
    font-family: system-ui, sans-serif;
    transition: background 0.1s, transform 0.08s;
    touch-action: manipulation;
    display: flex; align-items: center; justify-content: center;
    user-select: none;
    -webkit-user-select: none;
  }
  .scv-numpad-key:hover:not(:disabled) { background: rgba(255,255,255,0.08); }
  .scv-numpad-key:active:not(:disabled) { background: rgba(255,255,255,0.12); transform: scale(0.94); }
  .scv-numpad-key:disabled { opacity: 0.25; cursor: not-allowed; }

  .scv-numpad-key--back {
    background: transparent;
    border-color: transparent;
    color: rgba(255,255,255,0.35);
  }
  .scv-numpad-key--back:hover:not(:disabled) { background: rgba(255,255,255,0.05); color: rgba(255,255,255,0.6); }

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
  .scv-view { width: 100%; max-width: 500px; height: 100dvh; display: flex; flex-direction: column; background: #0d0d14; }

  .scv-header {
    display: flex; align-items: center; gap: 10px;
    padding: 13px 12px 13px 16px;
    background: rgba(255,255,255,0.025);
    border-bottom: 1px solid rgba(255,255,255,0.06);
    flex-shrink: 0;
  }
  .scv-header-lock { color: rgba(129,140,248,0.65); display: flex; align-items: center; flex-shrink: 0; }
  .scv-header-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
  .scv-header-title { font-size: 14px; font-weight: 600; color: #e2e8f0; }
  .scv-header-sub { font-size: 11px; color: rgba(255,255,255,0.35); }
  .scv-e2e-badge { font-size: 9px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; color: rgba(129,140,248,0.6); background: rgba(129,140,248,0.08); border: 1px solid rgba(129,140,248,0.15); padding: 2px 7px; border-radius: 20px; white-space: nowrap; flex-shrink: 0; }

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

  .scv-bubble { padding: 9px 13px; border-radius: 16px; font-size: 14px; line-height: 1.55; }
  .scv-bubble--own {
    display: flex; align-items: center; gap: 7px;
    background: rgba(129,140,248,0.13);
    border: 1px solid rgba(129,140,248,0.18);
    border-bottom-right-radius: 4px;
    color: rgba(129,140,248,0.65);
    font-size: 12px; font-weight: 500;
  }
  .scv-bubble--own.scv-bubble--grp-notfirst { border-top-right-radius: 4px; }

  .scv-bubble--their {
    background: rgba(255,255,255,0.07);
    border: 1px solid rgba(255,255,255,0.06);
    border-bottom-left-radius: 4px;
    color: #e2e8f0;
    word-break: break-word;
  }
  .scv-bubble--their.scv-bubble--grp-notfirst,
  .scv-bubble--locked.scv-bubble--grp-notfirst {
    border-top-left-radius: 4px;
  }

  .scv-bubble--decrypted { position: relative; padding-right: 30px; }

  .scv-bubble--locked {
    display: flex; align-items: center; gap: 8px;
    background: rgba(255,255,255,0.04);
    border: 1px dashed rgba(255,255,255,0.1);
    border-bottom-left-radius: 4px;
    color: rgba(255,255,255,0.3);
    font-size: 13px;
    cursor: pointer;
    min-height: 44px;
    position: relative;
    touch-action: manipulation;
    transition: background 0.15s, border-color 0.15s;
  }
  .scv-bubble--locked:hover { background: rgba(255,255,255,0.07); border-color: rgba(255,255,255,0.18); }
  .scv-bubble--locked-active { border-color: rgba(129,140,248,0.35); background: rgba(129,140,248,0.06); }

  .scv-relock-btn {
    position: absolute; top: 4px; right: 4px;
    width: 22px; height: 22px;
    padding: 11px; margin: -11px;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none; cursor: pointer;
    color: rgba(255,255,255,0.18);
    border-radius: 8px;
    touch-action: manipulation;
    box-sizing: content-box;
    transition: color 0.15s;
  }
  .scv-relock-btn:hover { color: rgba(129,140,248,0.7); }

  .scv-body { margin: 0; }
  .scv-ago { margin-left: auto; font-size: 10px; color: rgba(255,255,255,0.2); white-space: nowrap; }
  .scv-meta { display: flex; align-items: center; gap: 4px; padding: 0 2px; }
  .scv-time { font-size: 10px; color: rgba(255,255,255,0.22); padding: 0 2px; }
  .scv-countdown { font-size: 10px; color: rgba(129,140,248,0.5); font-variant-numeric: tabular-nums; padding: 0 2px; }

  /* Inline decrypt */
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
    border-top: 1px solid rgba(255,255,255,0.06);
    display: flex; flex-direction: column;
    gap: 7px;
    flex-shrink: 0;
    background: rgba(0,0,0,0.12);
    padding-bottom: calc(14px + env(safe-area-inset-bottom, 0px));
  }
  .scv-compose-inner { display: flex; align-items: flex-end; gap: 8px; }

  .scv-compose-icon-btn {
    width: 36px; height: 36px;
    min-width: 44px; min-height: 44px;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none;
    color: rgba(255,255,255,0.3);
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
  .scv-compose-text:focus { border-color: rgba(255,255,255,0.18); }
  .scv-compose-text::placeholder { color: rgba(255,255,255,0.2); }
  @media (max-width: 767px) { .scv-compose-text { font-size: 16px; } }

  .scv-send-btn {
    width: 44px; height: 44px;
    border-radius: 13px; border: none;
    background: rgba(129,140,248,0.8);
    color: #fff;
    cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
    transition: background 0.15s;
    touch-action: manipulation;
  }
  .scv-send-btn:hover:not(:disabled) { background: #818cf8; }
  .scv-send-btn:disabled { opacity: 0.28; cursor: not-allowed; }
  .scv-send-ring { width: 16px; height: 16px; border: 2px solid rgba(255,255,255,0.25); border-top-color: #fff; border-radius: 50%; animation: scv-spin 0.8s linear infinite; }

  .scv-compose-hint { display: flex; align-items: center; gap: 5px; margin: 0; font-size: 10px; color: rgba(255,255,255,0.18); }
  .scv-reply-err { margin: 0; font-size: 12px; color: #f87171; }

  /* Panic overlay */
  .scv-panic {
    position: fixed; inset: 0;
    background: #fff;
    z-index: 99999;
    cursor: default;
  }

  :global(.msg-sticker) { max-width: 120px; max-height: 120px; border-radius: 8px; display: block; }

  .scv-sr { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
  @keyframes scv-spin { to { transform: rotate(360deg); } }

  /* ── PIN shake animation ────────────────────────────────────── */
  @keyframes scv-shake {
    0%, 100% { transform: translateX(0); }
    15%      { transform: translateX(-7px); }
    35%      { transform: translateX(7px); }
    55%      { transform: translateX(-5px); }
    75%      { transform: translateX(4px); }
    90%      { transform: translateX(-2px); }
  }
  .scv-pin-dots--shake { animation: scv-shake 0.48s cubic-bezier(.36,.07,.19,.97) both; }

  /* ── Gate button CTA state (4+ digits entered) ──────────────── */
  .scv-gate-btn--ready {
    background: rgba(129,140,248,0.85);
    color: #fff;
    border-color: transparent;
    font-weight: 600;
  }
  .scv-gate-btn--ready:hover:not(:disabled) {
    background: #818cf8;
    color: #fff;
    border-color: transparent;
  }
  .scv-gate-spinner {
    width: 16px; height: 16px;
    border: 2px solid rgba(255,255,255,0.25);
    border-top-color: #fff;
    border-radius: 50%;
    animation: scv-spin 0.8s linear infinite;
    margin: 0 auto;
  }

  /* ── Message slide-in animation ─────────────────────────────── */
  @keyframes scv-msg-in {
    from { opacity: 0; transform: translateY(6px); }
    to   { opacity: 1; transform: translateY(0); }
  }
  .scv-msg { animation: scv-msg-in 0.18s ease-out both; }

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
  .scv-date-div span {
    font-size: 10px;
    color: rgba(255,255,255,0.22);
    white-space: nowrap;
    font-family: system-ui, sans-serif;
    letter-spacing: 0.03em;
    padding: 2px 4px;
  }
</style>
