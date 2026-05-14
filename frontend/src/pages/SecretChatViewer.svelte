<svelte:head>
  <title>Note</title>
</svelte:head>

<script>
  import { onMount, onDestroy, tick } from 'svelte';
  import { fade } from 'svelte/transition';
  import { decryptMessage, encryptMessage } from '../lib/crypto.js';
  import { apiGet, apiPost, fetchCsrf, clearCsrf } from '../lib/api.js';
  import EmojiPicker from '../components/primitives/EmojiPicker.svelte';
  import StickerPicker from '../components/primitives/StickerPicker.svelte';

  export let params = {};
  $: token = params.token || '';

  // 'loading' | 'login' | 'gate' | 'messages' | 'error'
  let state = 'loading';
  let errorMsg = '';
  let errorAction = '';

  // Login state (shown when no active session)
  let loginEmail = '';
  let loginPassword = '';
  let loginError = '';
  let loginLoading = false;

  // PIN state
  let pinDigits = [];
  $: pin = pinDigits.join('');
  let pinError = '';
  let rawMessages = [];
  let decrypted = [];
  let unlocking = false;
  let gatePin = '';

  // Reply compose
  let replyText = '';
  let replyError = '';
  let replySent = false;
  let sending = false;
  let messagesEl;
  let replyTextEl;
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
    lockedSet = new Set();
    for (const id of Object.keys(lockIntervals)) clearInterval(lockIntervals[id]);
    lockIntervals = {};
    lockCountdowns = {};
  }

  // PIN input helpers
  let pinInputEl;
  $: if (state === 'gate' && pinInputEl) setTimeout(() => pinInputEl?.focus(), 80);
  $: if (pinInputEl && pinDigits.length === 0) pinInputEl.value = '';

  function handlePinInput(e) {
    const digits = e.target.value.replace(/\D/g, '').slice(0, 8);
    e.target.value = digits;
    pinDigits = digits.split('');
  }

  function handlePinKeydown(e) {
    if (e.key === 'Enter') { e.preventDefault(); unlock(); }
  }

  let lockedSet = new Set();
  let lockCountdowns = {};
  let lockIntervals = {};

  const AUTO_LOCK_SECS = 30;

  async function loadInvite() {
    state = 'loading';
    try {
      const data = await apiGet(`/api/m/${token}`);
      if (!data.ok) {
        state = 'error';
        if (data.expired) {
          errorMsg = 'This link has expired.';
          errorAction = 'Ask the sender to create a new invite.';
        } else if (data.error === 'temporarily_unavailable') {
          errorMsg = 'Temporarily unavailable.';
          errorAction = 'Please try again in a few moments.';
        } else {
          errorMsg = 'Invalid or unrecognised link.';
          errorAction = 'Check the link and try again.';
        }
        return;
      }
      if (data.isParticipant === false) {
        state = 'error';
        errorMsg = 'This note is not for this account.';
        errorAction = "Make sure you're signed in to the right account.";
        return;
      }
      rawMessages = data.messages || [];
      state = 'gate';
    } catch {
      state = 'error';
      errorMsg = 'Could not connect.';
      errorAction = 'Check your connection and reload the page.';
    }
  }

  async function doLogin() {
    if (loginLoading || !loginEmail.trim() || !loginPassword) return;
    loginError = '';
    loginLoading = true;
    try {
      clearCsrf();
      await fetchCsrf();
      const res = await apiPost('/api/login', { email: loginEmail.trim(), password: loginPassword });
      if (!res.ok) {
        loginError = res.error || 'Incorrect email or password.';
        loginLoading = false;
        return;
      }
      // Refresh CSRF for the new session, then load the invite.
      clearCsrf();
      await fetchCsrf();
      loginLoading = false;
      await loadInvite();
    } catch {
      loginError = 'Could not connect — check your connection.';
      loginLoading = false;
    }
  }

  onMount(async () => {
    if (!token) {
      state = 'error';
      errorMsg = 'Invalid link.';
      errorAction = 'Ask the sender to create a new invite.';
      return;
    }
    try {
      const me = await apiGet('/api/me');
      if (!me.ok || !me.userId) {
        state = 'login';
        return;
      }
      await loadInvite();
    } catch {
      state = 'error';
      errorMsg = 'Could not connect.';
      errorAction = 'Check your connection and reload the page.';
    }
  });
  onDestroy(() => {
    if (_pinShakeTimer) clearTimeout(_pinShakeTimer);
    for (const id of Object.keys(lockIntervals)) clearInterval(lockIntervals[id]);
  });


  async function unlock() {
    if (unlocking || pin.length < 4) return;
    pinError = '';
    unlocking = true;

    // Validate PIN against the first owner message only — we never auto-decrypt.
    const validatorMsg = rawMessages.find(m => m.fromOwner);
    if (validatorMsg) {
      try {
        await decryptMessage(validatorMsg.ciphertext, validatorMsg.iv, validatorMsg.salt, pin);
      } catch {
        pinError = 'Incorrect code — try again';
        unlocking = false;
        pinDigits = [];
        triggerShake();
        return;
      }
    }

    // Build message list — all received (fromOwner) messages start locked (body: null).
    const results = rawMessages.map(m => ({
      id: m.createdAt + Math.random(),
      body: null,
      own: !m.fromOwner,
      createdAt: m.createdAt,
      raw: m.fromOwner ? m : null,
      ciphertext: m.ciphertext,
    }));

    // Lock set: all received messages (they must be tapped individually to reveal).
    lockedSet = new Set(results.filter(m => !m.own && m.raw).map(m => m.id));
    decrypted = results;
    gatePin = pin;
    pinDigits = [];
    unlocking = false;
    state = 'messages';
    await tick();
    if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
  }

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

  // ── Per-message tap-to-redecrypt (uses stored gatePin — no re-entry needed) ──
  async function decryptMsg(msg) {
    if (!msg.raw || !gatePin) return;
    try {
      const plain = await decryptMessage(msg.raw.ciphertext, msg.raw.iv, msg.raw.salt, gatePin);
      decrypted = decrypted.map(m => m.id === msg.id ? { ...m, body: plain } : m);
      lockedSet.delete(msg.id);
      lockedSet = new Set(lockedSet);
      startAutoLock(msg.id);
    } catch {
      // PIN changed externally — silently ignore; message stays locked
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
    decrypted = decrypted.map(m => m.id === id ? { ...m, body: null } : m);
  }

  // Sticky scroll
  let userScrolledUp = false;

  function handleMessagesScroll() {
    if (!messagesEl) return;
    const { scrollTop, scrollHeight, clientHeight } = messagesEl;
    userScrolledUp = scrollHeight - scrollTop - clientHeight > 60;
  }

  function jumpToBottom() {
    if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
    userScrolledUp = false;
  }

  // ── Reply ─────────────────────────────────────────────────────
  let photoInputEl;

  async function sendDirectReply(text) {
    if (!text || gatePin.length < 4) return;
    try {
      const { ciphertext, iv, salt } = await encryptMessage(text, gatePin);
      const data = await apiPost(`/api/m/${token}`, { ciphertext, iv, salt });
      if (!data.ok) return;
      decrypted = [...decrypted, {
        id: Date.now(),
        body: text,
        own: true,
        createdAt: new Date().toISOString(),
        ciphertext,
      }];
      replySent = true;
      await tick();
      if (!userScrolledUp && messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
    } catch { /* silently drop */ }
  }

  async function compressImage(file) {
    const MAX_EDGE = 720;
    const MAX_BYTES = 100_000;
    return new Promise((resolve, reject) => {
      const img = new Image();
      const url = URL.createObjectURL(file);
      img.onload = () => {
        URL.revokeObjectURL(url);
        let { width, height } = img;
        if (width > MAX_EDGE || height > MAX_EDGE) {
          const ratio = Math.min(MAX_EDGE / width, MAX_EDGE / height);
          width = Math.round(width * ratio);
          height = Math.round(height * ratio);
        }
        const canvas = document.createElement('canvas');
        canvas.width = width; canvas.height = height;
        const ctx = canvas.getContext('2d');
        ctx.drawImage(img, 0, 0, width, height);
        let quality = 0.82;
        const tryEncode = () => {
          const dataUrl = canvas.toDataURL('image/jpeg', quality);
          const bytes = Math.ceil((dataUrl.length - dataUrl.indexOf(',') - 1) * 0.75);
          if (bytes <= MAX_BYTES || quality <= 0.3) { resolve(dataUrl); return; }
          quality -= 0.15;
          tryEncode();
        };
        tryEncode();
      };
      img.onerror = reject;
      img.src = url;
    });
  }

  async function handlePhotoSelect(e) {
    const file = e.target.files?.[0];
    if (!file) return;
    e.target.value = '';
    try {
      const dataUrl = await compressImage(file);
      await sendDirectReply(`[photo:${dataUrl}]`);
    } catch { replyError = 'Could not attach photo.'; }
  }

  async function sendReply() {
    if (sending || !replyText.trim() || gatePin.length < 4) return;
    replyError = '';
    sending = true;
    const text = replyText.trim();
    try {
      const { ciphertext, iv, salt } = await encryptMessage(text, gatePin);
      const data = await apiPost(`/api/m/${token}`, { ciphertext, iv, salt });
      if (!data.ok) {
        replyError = data.expired
          ? 'Link expired — ask for a new invite.'
          : 'Message failed to send. Check your connection.';
        return;
      }
      decrypted = [...decrypted, {
        id: Date.now(),
        body: text,
        own: true,
        createdAt: new Date().toISOString(),
        ciphertext,
      }];
      replyText = '';
      replySent = true;
      await tick();
      if (!userScrolledUp && messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
    } catch {
      replyError = 'Could not send — check your connection.';
    } finally {
      sending = false;
    }
  }

  function handleReplyKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendReply(); }
  }

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

  function fakeGibberish(ts) {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
    const seed = new Date(ts).getTime() || Date.now();
    let s = '';
    let x = seed % 2147483647;
    for (let i = 0; i < 28; i++) {
      x = (x * 16807) % 2147483647;
      s += chars[x % 64];
    }
    return s + '…';
  }

  function ciphertextGibberish(ciphertext) {
    if (!ciphertext) return '···';
    const start = Math.min(4, ciphertext.length);
    const raw = ciphertext.slice(start, start + 28);
    return raw.length >= 6 ? raw + '…' : '···';
  }

  function isLikelyPhoto(msg) {
    const ct = msg.raw?.ciphertext;
    return ct && ct.length > 5000;
  }

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

  function goBack() {
    state = 'gate';
    gatePin = '';
    decrypted = [];
    for (const id of Object.keys(lockIntervals)) clearInterval(lockIntervals[id]);
    lockIntervals = {};
    lockCountdowns = {};
    lockedSet = new Set();
    pinDigits = [];
    pinError = '';
  }
</script>

{#if panicMode}
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
  <div class="scv-panic" on:click={restoreFromPanic}></div>
{/if}

<div class="scv" class:scv--plain={state !== 'messages'}>
  <!-- ── Loading ──────────────────────────────────────────────────── -->
  {#if state === 'loading'}
    <div class="scv-center" role="status" aria-label="Loading">
      <div class="scv-spinner scv-spinner--solo" aria-hidden="true"></div>
      <p class="scv-loading-text">Loading…</p>
    </div>

  <!-- ── Error ────────────────────────────────────────────────────── -->
  {:else if state === 'error'}
    <div class="scv-center" role="alert">
      <div class="scv-err-icon" aria-hidden="true">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
          <circle cx="12" cy="12" r="10"/>
          <line x1="15" y1="9" x2="9" y2="15"/>
          <line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
      </div>
      <p class="scv-err-msg">{errorMsg}</p>
      {#if errorAction}
        <p class="scv-err-action">{errorAction}</p>
      {/if}
    </div>

  <!-- ── Login ────────────────────────────────────────────────────── -->
  {:else if state === 'login'}
    <div class="scv-gate" role="main">
      <div class="scv-gate-content">

        <div class="scv-gate-text">
          <p class="scv-gate-title">Sign in</p>
        </div>

        <div class="scv-login-form">
          <div class="scv-login-field">
            <label class="scv-sr" for="scv-login-email">Email address</label>
            <input
              id="scv-login-email"
              class="scv-login-input"
              type="email"
              inputmode="email"
              autocomplete="email"
              autocorrect="off"
              autocapitalize="none"
              placeholder="Email"
              bind:value={loginEmail}
              disabled={loginLoading}
              on:keydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); document.getElementById('scv-login-pass')?.focus(); } }}
            />
          </div>
          <div class="scv-login-field">
            <label class="scv-sr" for="scv-login-pass">Password</label>
            <input
              id="scv-login-pass"
              class="scv-login-input"
              type="password"
              autocomplete="current-password"
              placeholder="Password"
              bind:value={loginPassword}
              disabled={loginLoading}
              on:keydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); doLogin(); } }}
            />
          </div>
          {#if loginError}
            <p class="scv-pin-err" role="alert">{loginError}</p>
          {/if}
        </div>

        <button
          class="scv-gate-btn"
          class:scv-gate-btn--ready={loginEmail.trim().length > 0 && loginPassword.length > 0}
          on:click={doLogin}
          disabled={loginLoading || !loginEmail.trim() || !loginPassword}
          type="button"
        >
          {#if loginLoading}
            <div class="scv-gate-spinner" aria-hidden="true"></div>
            <span>Signing in…</span>
          {:else}
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/>
              <polyline points="10 17 15 12 10 7"/>
              <line x1="15" y1="12" x2="3" y2="12"/>
            </svg>
            <span>Continue</span>
          {/if}
        </button>

      </div>
    </div>

  <!-- ── Gate ─────────────────────────────────────────────────────── -->
  {:else if state === 'gate'}
    <div class="scv-gate" role="main">
      <div class="scv-gate-content">

        <div class="scv-gate-text">
          <p class="scv-gate-title">Enter your passcode</p>
        </div>

        <!-- PIN dot visualizer -->
        <div class="scv-pin-dots" aria-hidden="true" role="presentation">
          {#each Array(8) as _, i}
            <span
              class="scv-pin-dot"
              class:scv-pin-dot--filled={i < pinDigits.length}
              class:scv-pin-dot--active={i === pinDigits.length - 1}
            ></span>
          {/each}
        </div>

        <div class="scv-pin-wrap">
          <label class="scv-sr" for="scv-gate-pin">Passcode — minimum 4 digits</label>
          <input
            id="scv-gate-pin"
            bind:this={pinInputEl}
            class="scv-pin-field"
            class:scv-pin-field--shake={pinShake}
            type="password"
            inputmode="numeric"
            pattern="\d*"
            maxlength="8"
            placeholder="Passcode"
            autocomplete="one-time-code"
            autocorrect="off"
            autocapitalize="none"
            on:input={handlePinInput}
            on:keydown={handlePinKeydown}
            aria-describedby={pinError ? 'scv-pin-err' : 'scv-pin-hint'}
          />
          {#if pinError}
            <p class="scv-pin-err" id="scv-pin-err" role="alert">{pinError}</p>
          {:else}
            <p class="scv-pin-hint" id="scv-pin-hint">Minimum 4 digits</p>
          {/if}
        </div>

        <button
          class="scv-gate-btn"
          class:scv-gate-btn--ready={pin.length >= 4}
          on:click={unlock}
          disabled={unlocking || pin.length < 4}
          type="button"
        >
          {#if unlocking}
            <div class="scv-gate-spinner" aria-hidden="true"></div>
            <span>Opening…</span>
          {:else}
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <polyline points="9 18 15 12 9 6"/>
            </svg>
            <span>{pin.length >= 4 ? 'Continue' : 'Enter passcode above'}</span>
          {/if}
        </button>

      </div>
    </div>

  <!-- ── Messages ─────────────────────────────────────────────────── -->
  {:else if state === 'messages'}
    <div class="scv-view">
      <header class="scv-header">
        <button
          class="scv-back-btn"
          on:click={goBack}
          aria-label="Back to access code"
          type="button"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <polyline points="15 18 9 12 15 6"/>
          </svg>
        </button>

        <div class="scv-header-spacer"></div>

        <button
          class="scv-icon-btn scv-panic-btn"
          on:click={() => panicMode = true}
          aria-label="Blank screen (tap to restore)"
          type="button"
        >
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
            <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
            <line x1="1" y1="1" x2="23" y2="23"/>
          </svg>
        </button>
      </header>

      <main
        class="scv-msgs"
        bind:this={messagesEl}
        on:scroll={handleMessagesScroll}
        role="log"
        aria-live="polite"
        aria-label="Messages"
      >
        {#if groupedDecrypted.length === 0}
          <div class="scv-empty" role="status">
            <div class="scv-empty-icon" aria-hidden="true">
              <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
              </svg>
            </div>
            <p class="scv-empty-title">No messages yet</p>
            <p class="scv-empty-sub">Type a reply below to start the conversation</p>
          </div>
        {/if}

        {#each groupedDecrypted as msg, i (msg.id)}
          {@const isDecrypted = !msg.own && msg.body !== null && !lockedSet.has(msg.id)}
          {@const label = dateLabel(msg.createdAt, i > 0 ? groupedDecrypted[i-1].createdAt : null)}
          {@const likelyPhoto = !msg.own && !isDecrypted && isLikelyPhoto(msg)}

          {#if label}
            <div class="scv-date-div" role="separator" aria-label={label}><span>{label}</span></div>
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
                class:scv-bubble--grp-first={msg.groupFirst}
                class:scv-bubble--grp-last={msg.groupLast}
                class:scv-bubble--grp-mid={!msg.groupFirst && !msg.groupLast}
                class:scv-bubble--photo={msg.body && parsePhoto(msg.body) !== null}
              >
                {#if msg.body && parsePhoto(msg.body)}
                  <img src={parsePhoto(msg.body)} class="msg-photo" alt="Photo" loading="lazy" />
                {:else if msg.body && parseGif(msg.body)}
                  <img src={parseGif(msg.body)} class="msg-sticker" alt="Sticker" loading="lazy" />
                {:else if msg.body}
                  <p class="scv-body">{msg.body}</p>
                {:else}
                  <span class="scv-cipher-text" aria-hidden="true">
                    {msg.ciphertext ? ciphertextGibberish(msg.ciphertext) : fakeGibberish(msg.createdAt)}
                  </span>
                  <span class="scv-lock-icon" aria-hidden="true">
                    <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                  </span>
                {/if}
              </div>
              {#if msg.groupLast}
                <time class="scv-time" datetime={msg.createdAt}>{clockTime(msg.createdAt)}</time>
              {/if}

            {:else if isDecrypted}
              <div
                class="scv-bubble scv-bubble--their scv-bubble--decrypted"
                class:scv-bubble--grp-first={msg.groupFirst}
                class:scv-bubble--grp-last={msg.groupLast}
                class:scv-bubble--grp-mid={!msg.groupFirst && !msg.groupLast}
                class:scv-bubble--photo={parsePhoto(msg.body) !== null}
              >
                {#if parsePhoto(msg.body)}
                  <img src={parsePhoto(msg.body)} class="msg-photo" alt="Encrypted photo" loading="lazy" />
                {:else if parseGif(msg.body)}
                  <img src={parseGif(msg.body)} class="msg-sticker" alt="Animated sticker" loading="lazy" />
                {:else}
                  <p class="scv-body">{msg.body}</p>
                {/if}
                <button class="scv-relock-btn" on:click={() => relockMsg(msg.id)} aria-label="Lock this message" type="button">
                  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                </button>
              </div>
              {#if msg.groupLast}
                <div class="scv-meta">
                  {#if lockCountdowns[msg.id] != null}
                    <span class="scv-countdown" aria-live="polite">Locks in {lockCountdowns[msg.id]}s</span>
                  {:else}
                    <time class="scv-time" datetime={msg.createdAt}>{clockTime(msg.createdAt)}</time>
                  {/if}
                </div>
              {/if}

            {:else}
              <!-- Locked received message -->
              <button
                class="scv-bubble scv-bubble--locked"
                class:scv-bubble--grp-first={msg.groupFirst}
                class:scv-bubble--grp-last={msg.groupLast}
                class:scv-bubble--grp-mid={!msg.groupFirst && !msg.groupLast}
                class:scv-bubble--locked-photo={likelyPhoto}
                on:click={() => decryptMsg(msg)}
                aria-label={likelyPhoto ? 'Locked photo — tap to reveal' : 'Locked message — tap to reveal'}
                type="button"
              >
                {#if likelyPhoto}
                  <span class="scv-photo-placeholder" aria-hidden="true">
                    <span class="scv-photo-blur-layer"></span>
                    <span class="scv-photo-cam-icon">
                      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
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

              {#if msg.groupLast}
                <time class="scv-time" datetime={msg.createdAt}>{clockTime(msg.createdAt)}</time>
              {/if}
            {/if}
          </div>
        {/each}

        {#if replySent}
          <div class="scv-sent-notice" role="status">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
            Reply sent securely
          </div>
        {/if}
      </main>

      <!-- Scroll FAB -->
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
      <footer class="scv-compose">
        <div class="scv-compose-inner">
          <button
            class="scv-compose-icon-btn"
            bind:this={emojiAnchor}
            on:click={() => { emojiOpen = !emojiOpen; stickerOpen = false; }}
            aria-label="Open emoji picker"
            aria-expanded={emojiOpen}
            aria-haspopup="true"
            type="button"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <circle cx="12" cy="12" r="10"/>
              <path d="M8 14s1.5 2 4 2 4-2 4-2"/>
              <line x1="9" y1="9" x2="9.01" y2="9"/>
              <line x1="15" y1="9" x2="15.01" y2="9"/>
            </svg>
          </button>

          <button
            class="scv-compose-icon-btn"
            bind:this={stickerAnchor}
            on:click={() => { stickerOpen = !stickerOpen; emojiOpen = false; }}
            aria-label="Open sticker picker"
            aria-expanded={stickerOpen}
            aria-haspopup="true"
            type="button"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
            </svg>
          </button>

          <button
            class="scv-compose-icon-btn"
            on:click={() => photoInputEl?.click()}
            aria-label="Attach photo"
            type="button"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
              <circle cx="12" cy="13" r="4"/>
            </svg>
          </button>
          <input
            bind:this={photoInputEl}
            type="file"
            accept="image/*"
            class="scv-sr"
            tabindex="-1"
            aria-hidden="true"
            on:change={handlePhotoSelect}
          />

          <label class="scv-sr" for="scv-reply">Reply</label>
          <textarea
            id="scv-reply"
            class="scv-compose-text"
            rows="1"
            maxlength="2000"
            placeholder="Write a reply…"
            bind:value={replyText}
            bind:this={replyTextEl}
            on:keydown={handleReplyKeydown}
            disabled={sending}
          ></textarea>
          <button
            class="scv-send-btn"
            class:scv-send-btn--active={replyText.trim().length > 0}
            on:click={sendReply}
            disabled={sending || !replyText.trim()}
            aria-label="Send reply"
            type="button"
          >
            {#if sending}
              <div class="scv-send-ring" aria-hidden="true"></div>
            {:else}
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
            {/if}
          </button>
        </div>
        {#if replyError}
          <p class="scv-reply-err" role="alert">{replyError}</p>
        {/if}
      </footer>

      <!-- Pickers outside compose to avoid backdrop-filter bug on iOS -->
      <EmojiPicker
        open={emojiOpen}
        anchor={emojiAnchor}
        on:pick={(e) => { replyText += e.detail; emojiOpen = false; setTimeout(() => replyTextEl?.focus(), 50); }}
        on:close={() => emojiOpen = false}
      />
      <StickerPicker
        open={stickerOpen}
        anchor={stickerAnchor}
        on:pick={(e) => { stickerOpen = false; sendDirectReply(e.detail); }}
        on:close={() => stickerOpen = false}
      />
    </div>
  {/if}
</div>

<style>
  /* Chat accent tokens — teal system, consistent with SecretChatPanel */
  :root {
    --scv-accent:        var(--primary-500, #14b8a6);
    --scv-accent-dim:    rgba(20, 184, 166, 0.18);
    --scv-accent-subtle: rgba(20, 184, 166, 0.08);
    --scv-accent-glow:   rgba(20, 184, 166, 0.28);
    --scv-bg:            #060610;
    --scv-border:        rgba(255, 255, 255, 0.07);
    --scv-border-accent: rgba(20, 184, 166, 0.22);
  }

  .scv {
    min-height: 100dvh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--scv-bg);
    box-sizing: border-box;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    position: relative;
    overflow: hidden;
  }

  /* Ambient teal mesh */
  .scv::before {
    content: '';
    position: fixed;
    inset: 0;
    background:
      radial-gradient(ellipse 65% 50% at 20% 10%,  rgba(20, 184, 166, 0.07) 0%, transparent 55%),
      radial-gradient(ellipse 50% 40% at 80% 90%,  rgba(20, 184, 166, 0.05) 0%, transparent 50%),
      radial-gradient(ellipse 40% 35% at 60% 45%,  rgba(6, 182, 212, 0.04) 0%, transparent 55%);
    pointer-events: none;
    z-index: 0;
  }

  /* Hex-grid encrypted-space texture */
  .scv::after {
    content: '';
    position: fixed;
    inset: 0;
    background-image:
      repeating-linear-gradient(
        0deg,
        transparent,
        transparent 28px,
        rgba(20, 184, 166, 0.018) 28px,
        rgba(20, 184, 166, 0.018) 29px
      ),
      repeating-linear-gradient(
        60deg,
        transparent,
        transparent 28px,
        rgba(20, 184, 166, 0.012) 28px,
        rgba(20, 184, 166, 0.012) 29px
      );
    pointer-events: none;
    z-index: 0;
    animation: scv-hex-drift 24s linear infinite;
  }
  @keyframes scv-hex-drift {
    from { background-position: 0 0, 0 0; }
    to   { background-position: 0 56px, 48px 0; }
  }

  /* Strip all encryption / cyber tells before PIN entry */
  .scv--plain::before,
  .scv--plain::after { display: none; }

  .scv > * { position: relative; z-index: 1; }

  /* ── Loading / error ───────────────────────────────────────────── */
  .scv-center {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-4, 16px);
    text-align: center;
    padding: var(--space-8, 32px);
  }

  .scv-spinner {
    position: absolute;
    inset: -4px;
    border-radius: var(--radius-full, 9999px);
    border: 2px solid transparent;
    border-top-color: var(--scv-accent);
    animation: scv-spin 0.8s linear infinite;
  }

  /* Standalone spinner — no icon container, no teal circle */
  .scv-spinner--solo {
    position: relative;
    inset: unset;
    width: 32px; height: 32px;
    border-radius: var(--radius-full, 9999px);
    border: 2px solid rgba(255, 255, 255, 0.12);
    border-top-color: rgba(255, 255, 255, 0.55);
    animation: scv-spin 0.8s linear infinite;
  }

  .scv-loading-text {
    margin: 0;
    font-size: var(--text-sm, 0.875rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.4);
  }

  .scv-err-icon {
    width: 56px; height: 56px;
    border-radius: var(--radius-full, 9999px);
    background: rgba(248, 113, 113, 0.08);
    border: 1px solid rgba(248, 113, 113, 0.22);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--danger-400, #f87171);
  }

  .scv-err-msg {
    margin: 0;
    font-size: var(--text-base, 1rem);
    font-weight: 600;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.75);
    max-width: 260px;
  }

  .scv-err-action {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.35);
    max-width: 240px;
  }

  /* ── Gate ─────────────────────────────────────────────────────── */
  .scv-gate {
    position: relative;
    width: 100%;
    min-height: 100dvh;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
  }

  .scv-gate-glow {
    position: absolute;
    inset: 0;
    background:
      radial-gradient(ellipse 60% 45% at 50% 35%, rgba(20, 184, 166, 0.09) 0%, transparent 65%),
      radial-gradient(ellipse 45% 35% at 75% 70%, rgba(6, 182, 212, 0.06) 0%, transparent 55%);
    pointer-events: none;
    animation: scv-glow-breathe 7s ease-in-out infinite;
  }

  @keyframes scv-glow-breathe {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.6; }
  }

  .scv-gate-content {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-5, 20px);
    width: 100%;
    max-width: 300px;
    padding: var(--space-8, 32px) var(--space-4, 16px) var(--space-10, 40px);
    text-align: center;
  }

  .scv-gate-icon {
    width: 80px; height: 80px;
    border-radius: var(--radius-full, 9999px);
    background: var(--scv-accent-subtle);
    border: 1px solid var(--scv-border-accent);
    box-shadow: 0 0 0 14px rgba(20, 184, 166, 0.04), 0 0 44px var(--scv-accent-glow);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--scv-accent);
    animation: scv-icon-breathe 5s ease-in-out infinite;
  }

  @keyframes scv-icon-breathe {
    0%, 100% { box-shadow: 0 0 0 14px rgba(20, 184, 166, 0.04), 0 0 44px rgba(20, 184, 166, 0.22); }
    50% { box-shadow: 0 0 0 22px rgba(20, 184, 166, 0.02), 0 0 64px rgba(20, 184, 166, 0.32); }
  }

  .scv-gate-text {
    display: flex;
    flex-direction: column;
    gap: var(--space-2, 8px);
  }

  .scv-gate-title {
    margin: 0;
    font-size: var(--text-xl, 1.125rem);
    font-weight: 700;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.92);
  }

  .scv-gate-sub {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.35);
    line-height: var(--leading-relaxed, 1.625);
    max-width: 240px;
  }

  /* ── PIN dot visualizer ── */
  .scv-pin-dots {
    display: flex;
    gap: var(--space-2, 8px);
    align-items: center;
    height: 12px;
  }

  .scv-pin-dot {
    width: 8px; height: 8px;
    border-radius: var(--radius-full, 9999px);
    background: rgba(255, 255, 255, 0.12);
    border: 1px solid rgba(255, 255, 255, 0.18);
    transition: background 0.15s, transform 0.15s cubic-bezier(0.34, 1.56, 0.64, 1), box-shadow 0.15s;
    flex-shrink: 0;
  }

  .scv-pin-dot--filled {
    background: var(--scv-accent);
    border-color: transparent;
    box-shadow: 0 0 8px rgba(20, 184, 166, 0.5);
  }

  .scv-pin-dot--active {
    transform: scale(1.25);
  }

  .scv-pin-wrap {
    width: 100%;
    max-width: 260px;
    display: flex;
    flex-direction: column;
    gap: var(--space-2, 8px);
    align-items: center;
  }

  .scv-pin-field {
    width: 100%;
    padding: var(--space-4, 16px) var(--space-4, 16px);
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid var(--scv-border-accent);
    border-radius: var(--radius-lg, 14px);
    color: rgba(255, 255, 255, 0.92);
    font-size: 24px;
    letter-spacing: 0.35em;
    text-align: center;
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    outline: none;
    transition: border-color var(--duration-normal, 200ms), box-shadow var(--duration-normal, 200ms);
    caret-color: var(--scv-accent);
    -webkit-appearance: none;
    appearance: none;
    min-height: 56px;
  }
  .scv-pin-field::placeholder {
    color: rgba(255, 255, 255, 0.2);
    letter-spacing: 0.05em;
    font-size: var(--text-sm, 0.875rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
  }
  .scv-pin-field:focus {
    border-color: var(--scv-accent);
    box-shadow: 0 0 0 3px var(--scv-accent-subtle);
  }

  @keyframes scv-shake {
    0%, 100% { transform: translateX(0); }
    15% { transform: translateX(-7px); }
    35% { transform: translateX(7px); }
    55% { transform: translateX(-5px); }
    75% { transform: translateX(4px); }
    90% { transform: translateX(-2px); }
  }
  .scv-pin-field--shake { animation: scv-shake 0.48s cubic-bezier(0.36, 0.07, 0.19, 0.97) both; }

  .scv-pin-err {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: var(--danger-400, #f87171);
    font-weight: 500;
  }

  .scv-pin-hint {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.2);
  }

  /* ── Gate button ─────────────────────────────────────────────── */
  .scv-gate-btn {
    width: 100%;
    max-width: 260px;
    padding: var(--space-4, 16px);
    border-radius: var(--radius-lg, 14px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    background: rgba(255, 255, 255, 0.06);
    color: rgba(255, 255, 255, 0.5);
    font-size: var(--text-sm, 0.875rem);
    font-weight: 600;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    cursor: pointer;
    min-height: 52px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2, 8px);
    transition: background var(--duration-normal, 200ms), color var(--duration-normal, 200ms), border-color var(--duration-normal, 200ms), transform var(--duration-fast, 100ms), box-shadow var(--duration-normal, 200ms);
    touch-action: manipulation;
  }
  .scv-gate-btn:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.1);
    color: rgba(255, 255, 255, 0.8);
    border-color: rgba(255, 255, 255, 0.18);
  }
  .scv-gate-btn:disabled { opacity: 0.3; cursor: not-allowed; }
  .scv-gate-btn:focus-visible {
    outline: 2px solid var(--scv-accent);
    outline-offset: 2px;
  }

  .scv-gate-btn--ready {
    background: linear-gradient(135deg, var(--primary-400, #2dd4bf) 0%, var(--primary-600, #0d9488) 100%);
    color: #fff;
    border-color: transparent;
    font-weight: 700;
    box-shadow: 0 4px 20px rgba(20, 184, 166, 0.4);
  }
  .scv-gate-btn--ready:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 6px 28px rgba(20, 184, 166, 0.55);
  }
  .scv-gate-btn--ready:active:not(:disabled) {
    transform: scale(0.97);
  }

  .scv-gate-spinner {
    width: 16px; height: 16px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: #fff;
    border-radius: var(--radius-full, 9999px);
    animation: scv-spin 0.7s linear infinite;
    flex-shrink: 0;
  }

  /* ── Messages view ─────────────────────────────────────────────── */
  .scv-view {
    width: 100%;
    max-width: 520px;
    height: 100dvh;
    display: flex;
    flex-direction: column;
    background: transparent;
  }

  header.scv-header {
    display: flex;
    align-items: center;
    gap: var(--space-2-5, 10px);
    padding: var(--space-3, 12px) var(--space-3, 12px) var(--space-3, 12px) var(--space-4, 16px);
    background: rgba(6, 6, 16, 0.75);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border-bottom: 1px solid var(--scv-border);
    flex-shrink: 0;
    min-height: 60px;
    padding-top: max(var(--space-3, 12px), env(safe-area-inset-top, 0px));
  }

  .scv-header-spacer { flex: 1; }

  .scv-back-btn,
  .scv-icon-btn {
    width: 44px; height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: none;
    cursor: pointer;
    border-radius: var(--radius-sm2, 8px);
    flex-shrink: 0;
    transition: color var(--duration-fast, 100ms), background var(--duration-fast, 100ms);
    touch-action: manipulation;
  }

  .scv-back-btn {
    color: rgba(255, 255, 255, 0.45);
  }
  .scv-back-btn:hover {
    color: rgba(255, 255, 255, 0.8);
    background: rgba(255, 255, 255, 0.06);
  }
  .scv-back-btn:focus-visible,
  .scv-icon-btn:focus-visible {
    outline: 2px solid var(--scv-accent);
    outline-offset: 2px;
  }

  .scv-panic-btn {
    color: rgba(255, 255, 255, 0.2);
  }
  .scv-panic-btn:hover {
    color: var(--danger-400, #f87171);
    background: rgba(248, 113, 113, 0.07);
  }

  main.scv-msgs {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-4, 16px) var(--space-4, 16px) var(--space-2-5, 10px);
    display: flex;
    flex-direction: column;
    gap: var(--space-1, 4px);
    overscroll-behavior: contain;
  }
  main.scv-msgs::-webkit-scrollbar { width: 3px; }
  main.scv-msgs::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.08);
    border-radius: var(--radius-full, 9999px);
  }

  /* ── Empty state ───────────────────────────────────────────── */
  .scv-empty {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-3, 12px);
    padding: var(--space-10, 40px) 0;
    text-align: center;
  }

  .scv-empty-icon {
    width: 64px; height: 64px;
    border-radius: var(--radius-full, 9999px);
    background: var(--scv-accent-subtle);
    border: 1px solid var(--scv-border-accent);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--scv-accent);
  }

  .scv-empty-title {
    margin: 0;
    font-size: var(--text-base, 1rem);
    font-weight: 700;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.75);
  }

  .scv-empty-sub {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.28);
    line-height: var(--leading-relaxed, 1.625);
  }

  /* ── Message rows ──────────────────────────────────────────────── */
  .scv-msg {
    display: flex;
    flex-direction: column;
    max-width: 76%;
    gap: var(--space-0-5, 2px);
    margin-bottom: var(--space-1, 4px);
    animation: scv-msg-in 0.22s var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
  }
  .scv-msg--group-cont { margin-bottom: 1px; }
  .scv-msg--own { align-self: flex-end; align-items: flex-end; }
  .scv-msg--their { align-self: flex-start; align-items: flex-start; }

  .scv-bubble {
    padding: var(--space-2-5, 10px) var(--space-3-5, 14px);
    border-radius: var(--radius-xl, 20px);
    font-size: var(--text-sm, 0.875rem);
    line-height: var(--leading-relaxed, 1.625);
    font-family: var(--font-sans, 'Nunito', sans-serif);
  }

  /* Grouping radius */
  .scv-bubble--own.scv-bubble--grp-first  { border-bottom-right-radius: var(--radius-sm, 6px); }
  .scv-bubble--own.scv-bubble--grp-last   { border-top-right-radius: var(--radius-sm, 6px); }
  .scv-bubble--own.scv-bubble--grp-mid    { border-top-right-radius: var(--radius-sm, 6px); border-bottom-right-radius: var(--radius-sm, 6px); }
  .scv-bubble--their.scv-bubble--grp-first,
  .scv-bubble--locked.scv-bubble--grp-first { border-bottom-left-radius: var(--radius-sm, 6px); }
  .scv-bubble--their.scv-bubble--grp-last,
  .scv-bubble--locked.scv-bubble--grp-last  { border-top-left-radius: var(--radius-sm, 6px); }
  .scv-bubble--their.scv-bubble--grp-mid,
  .scv-bubble--locked.scv-bubble--grp-mid   { border-top-left-radius: var(--radius-sm, 6px); border-bottom-left-radius: var(--radius-sm, 6px); }

  .scv-bubble--own {
    display: flex;
    align-items: center;
    gap: var(--space-2, 8px);
    background: linear-gradient(135deg, rgba(20, 184, 166, 0.14) 0%, rgba(20, 184, 166, 0.08) 100%);
    border: 1px solid rgba(20, 184, 166, 0.22);
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.06), 0 2px 8px rgba(0, 0, 0, 0.25);
    max-width: 100%;
    overflow: hidden;
  }

  .scv-bubble--their {
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid rgba(255, 255, 255, 0.09);
    color: rgba(255, 255, 255, 0.92);
    word-break: break-word;
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04), 0 1px 4px rgba(0, 0, 0, 0.15);
  }

  .scv-bubble--decrypted {
    position: relative;
    padding-right: var(--space-8, 32px);
  }

  .scv-bubble--decrypted.scv-bubble--photo {
    padding: var(--space-1, 4px);
    overflow: hidden;
  }

  .scv-bubble--locked {
    display: flex;
    align-items: center;
    gap: var(--space-2, 8px);
    background: rgba(255, 255, 255, 0.03);
    border: 1px dashed rgba(255, 255, 255, 0.12);
    color: rgba(255, 255, 255, 0.3);
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    cursor: pointer;
    min-height: 44px;
    position: relative;
    touch-action: manipulation;
    transition: background var(--duration-fast, 100ms), border-color var(--duration-fast, 100ms);
    overflow: hidden;
  }
  .scv-bubble--locked:hover {
    background: rgba(255, 255, 255, 0.06);
    border-color: rgba(255, 255, 255, 0.2);
  }
  .scv-bubble--locked:focus-visible {
    outline: 2px solid var(--scv-accent);
    outline-offset: 2px;
  }
  .scv-bubble--locked-active {
    border-color: var(--scv-border-accent);
    background: var(--scv-accent-subtle);
  }

  .scv-bubble--locked-photo {
    flex-direction: column;
    align-items: flex-start;
    padding: 0;
    border-style: solid;
    min-height: 88px;
    border-radius: var(--radius-lg, 14px);
    overflow: hidden;
  }

  .scv-photo-placeholder {
    position: relative;
    width: 190px; height: 130px;
    display: block;
    overflow: hidden;
    flex-shrink: 0;
  }

  .scv-photo-blur-layer {
    position: absolute;
    inset: 0;
    background:
      radial-gradient(ellipse 60% 50% at 30% 40%, rgba(20, 184, 166, 0.15) 0%, transparent 60%),
      radial-gradient(ellipse 40% 55% at 70% 60%, rgba(6, 182, 212, 0.10) 0%, transparent 55%),
      linear-gradient(135deg, rgba(255, 255, 255, 0.03) 0%, rgba(0, 0, 0, 0.3) 100%);
    filter: blur(10px);
  }

  .scv-photo-cam-icon {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: rgba(255, 255, 255, 0.5);
  }

  .scv-locked-photo-label {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    padding: var(--space-2, 8px) var(--space-3, 12px);
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.35);
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    width: 100%;
    box-sizing: border-box;
    flex-shrink: 0;
  }

  .scv-cipher-text,
  .scv-cipher-preview {
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    font-size: var(--text-2xs, 0.6875rem);
    letter-spacing: 0.04em;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
    user-select: none;
  }
  .scv-cipher-text { color: rgba(20, 184, 166, 0.55); }
  .scv-cipher-preview { color: rgba(255, 255, 255, 0.28); font-size: 10px; }

  .scv-lock-icon {
    color: rgba(20, 184, 166, 0.45);
    flex-shrink: 0;
    display: flex;
    align-items: center;
  }

  .scv-relock-btn {
    position: absolute;
    top: var(--space-1, 4px);
    right: var(--space-1, 4px);
    width: 44px; height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.35);
    border: none;
    cursor: pointer;
    color: rgba(255, 255, 255, 0.4);
    border-radius: var(--radius-sm2, 8px);
    transition: color var(--duration-fast, 100ms), background var(--duration-fast, 100ms);
    touch-action: manipulation;
  }
  .scv-relock-btn:hover {
    color: var(--scv-accent);
    background: var(--scv-accent-dim);
  }
  .scv-relock-btn:focus-visible {
    outline: 2px solid var(--scv-accent);
    outline-offset: 1px;
  }

  .scv-body { margin: 0; }
  .scv-ago {
    margin-left: auto;
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: rgba(255, 255, 255, 0.22);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .scv-meta {
    display: flex;
    align-items: center;
    gap: var(--space-1, 4px);
    padding: 0 var(--space-0-5, 2px);
    margin-top: 1px;
  }

  .scv-time {
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: rgba(255, 255, 255, 0.25);
    font-variant-numeric: tabular-nums;
    padding: 0 var(--space-0-5, 2px);
  }

  .scv-countdown {
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: var(--scv-accent);
    font-variant-numeric: tabular-nums;
    padding: 0 var(--space-0-5, 2px);
  }

  .scv-sent-notice {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: var(--success-400, #34d399);
    align-self: center;
    margin-top: var(--space-1, 4px);
  }

  /* Compose */
  footer.scv-compose {
    padding: var(--space-2-5, 10px) var(--space-4, 16px) max(var(--space-4, 16px), env(safe-area-inset-bottom, 0px));
    border-top: 1px solid var(--scv-border);
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5, 6px);
    flex-shrink: 0;
    background: rgba(6, 6, 16, 0.85);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
  }

  .scv-compose-inner {
    display: flex;
    align-items: flex-end;
    gap: var(--space-1-5, 6px);
  }

  .scv-compose-icon-btn {
    width: 44px; height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: none;
    color: rgba(255, 255, 255, 0.3);
    cursor: pointer;
    border-radius: var(--radius-sm2, 8px);
    flex-shrink: 0;
    transition: color var(--duration-fast, 100ms), background var(--duration-fast, 100ms);
    touch-action: manipulation;
  }
  .scv-compose-icon-btn:hover {
    color: rgba(255, 255, 255, 0.7);
    background: rgba(255, 255, 255, 0.07);
  }
  .scv-compose-icon-btn:focus-visible {
    outline: 2px solid var(--scv-accent);
    outline-offset: 2px;
  }

  .scv-compose-text {
    flex: 1;
    resize: none;
    padding: var(--space-2-5, 10px) var(--space-3, 12px);
    border-radius: var(--radius-lg, 14px);
    border: 1px solid rgba(255, 255, 255, 0.09);
    background: rgba(255, 255, 255, 0.04);
    color: rgba(255, 255, 255, 0.92);
    font-size: var(--text-sm, 0.875rem);
    line-height: var(--leading-relaxed, 1.625);
    outline: none;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    -webkit-appearance: none;
    max-height: 120px;
    min-height: 44px;
    box-sizing: border-box;
    width: 100%;
    transition: border-color var(--duration-fast, 100ms), box-shadow var(--duration-fast, 100ms);
  }
  .scv-compose-text:focus {
    border-color: var(--scv-border-accent);
    box-shadow: 0 0 0 3px var(--scv-accent-subtle);
  }
  .scv-compose-text::placeholder { color: rgba(255, 255, 255, 0.2); }
  @media (max-width: 767px) { .scv-compose-text { font-size: 16px; } }

  .scv-send-btn {
    width: 44px; height: 44px;
    border-radius: var(--radius-lg, 14px);
    border: none;
    background: rgba(255, 255, 255, 0.07);
    color: rgba(255, 255, 255, 0.28);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: transform var(--duration-fast, 100ms), box-shadow var(--duration-fast, 100ms), background var(--duration-normal, 200ms), color var(--duration-normal, 200ms);
    touch-action: manipulation;
  }
  .scv-send-btn--active {
    background: linear-gradient(135deg, var(--primary-400, #2dd4bf) 0%, var(--primary-600, #0d9488) 100%);
    color: #fff;
    box-shadow: 0 2px 12px rgba(20, 184, 166, 0.4);
  }
  .scv-send-btn--active:hover {
    transform: scale(1.06);
    box-shadow: 0 4px 20px rgba(20, 184, 166, 0.6);
  }
  .scv-send-btn--active:active { transform: scale(0.93); }
  .scv-send-btn:disabled { opacity: 0.28; cursor: not-allowed; box-shadow: none; }
  .scv-send-btn:focus-visible {
    outline: 2px solid var(--scv-accent);
    outline-offset: 2px;
  }

  .scv-send-ring {
    width: 16px; height: 16px;
    border: 2px solid rgba(255, 255, 255, 0.25);
    border-top-color: #fff;
    border-radius: var(--radius-full, 9999px);
    animation: scv-spin 0.7s linear infinite;
  }

  .scv-reply-err {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: var(--danger-400, #f87171);
    font-weight: 500;
  }

  /* Panic overlay — dark, not white */
  .scv-panic {
    position: fixed;
    inset: 0;
    background: #000;
    z-index: 99999;
    cursor: default;
    animation: scv-panic-on 0.15s ease-out both;
  }
  @keyframes scv-panic-on {
    from { opacity: 0; }
    to   { opacity: 1; }
  }

  /* Scroll FAB */
  .scv-scroll-fab {
    align-self: flex-end;
    margin: calc(-1 * var(--space-2, 8px)) var(--space-3, 12px) 0;
    position: relative;
    width: 44px; height: 44px;
    border-radius: var(--radius-full, 9999px);
    border: 1px solid var(--scv-border-accent);
    background: rgba(6, 6, 16, 0.9);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    color: var(--scv-accent);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.6), 0 0 12px var(--scv-accent-glow);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    touch-action: manipulation;
    transition: background var(--duration-fast, 100ms), transform var(--duration-fast, 100ms), box-shadow var(--duration-fast, 100ms);
    z-index: 10;
    flex-shrink: 0;
  }
  .scv-scroll-fab:hover {
    background: var(--scv-accent-subtle);
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.6), 0 0 20px var(--scv-accent-glow);
  }
  .scv-scroll-fab:focus-visible {
    outline: 2px solid var(--scv-accent);
    outline-offset: 2px;
  }

  :global(.msg-sticker) {
    max-width: 128px;
    max-height: 128px;
    border-radius: var(--radius-sm2, 8px);
    display: block;
  }
  :global(.msg-photo) {
    max-width: 240px;
    max-height: 280px;
    border-radius: var(--radius-md, 10px);
    display: block;
    object-fit: cover;
  }

  .scv-date-div {
    display: flex;
    align-items: center;
    gap: var(--space-2-5, 10px);
    margin: var(--space-2-5, 10px) 0 var(--space-1-5, 6px);
    align-self: stretch;
    position: sticky;
    top: var(--space-2, 8px);
    z-index: 2;
    pointer-events: none;
  }
  .scv-date-div::before,
  .scv-date-div::after {
    content: '';
    flex: 1;
    height: 1px;
    background: rgba(255, 255, 255, 0.05);
  }
  .scv-date-div span {
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: rgba(255, 255, 255, 0.3);
    white-space: nowrap;
    letter-spacing: 0.05em;
    padding: 3px var(--space-2, 8px);
    background: rgba(6, 6, 16, 0.9);
    border: 1px solid rgba(255, 255, 255, 0.07);
    border-radius: var(--radius-full, 9999px);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
  }

  .scv-sr {
    position: absolute; width: 1px; height: 1px;
    padding: 0; margin: -1px;
    overflow: hidden; clip: rect(0, 0, 0, 0);
    white-space: nowrap; border: 0;
  }

  @keyframes scv-spin { to { transform: rotate(360deg); } }

  /* Message entry animation — spring, no rotateX */
  @keyframes scv-msg-in {
    from { opacity: 0; transform: translateY(10px) scale(0.97); }
    to   { opacity: 1; transform: translateY(0) scale(1); }
  }

  /* ── Login form ─────────────────────────────────────────────────── */
  .scv-login-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-3, 12px);
    width: 100%;
  }

  .scv-login-field {
    width: 100%;
    position: relative;
  }

  .scv-login-input {
    width: 100%;
    box-sizing: border-box;
    height: 48px;
    padding: 0 var(--space-4, 16px);
    background: rgba(255, 255, 255, 0.05);
    border: 1.5px solid var(--scv-border, rgba(255, 255, 255, 0.07));
    border-radius: var(--radius-lg, 12px);
    color: rgba(255, 255, 255, 0.9);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    font-size: var(--text-sm, 0.875rem);
    font-weight: 500;
    caret-color: var(--scv-accent);
    outline: none;
    transition: border-color 150ms ease, background 150ms ease;
  }
  .scv-login-input::placeholder {
    color: rgba(255, 255, 255, 0.28);
  }
  .scv-login-input:focus {
    border-color: var(--scv-border-accent, rgba(20, 184, 166, 0.4));
    background: rgba(20, 184, 166, 0.05);
  }
  .scv-login-input:disabled {
    opacity: 0.5;
  }

  @media (prefers-reduced-motion: reduce) {
    .scv::before { animation: none; }
    .scv::after  { animation: none; }
    .scv-msg { animation: none; }
    .scv-date-div { animation: none; }
    .scv-spinner { animation: none; }
    .scv-send-ring { animation: none; }
    .scv-gate-spinner { animation: none; }
    .scv-gate-glow { animation: none; }
    .scv-gate-icon { animation: none; }
    .scv-pin-dot { transition: none; }
    .scv-panic { animation: none; }
    .scv-login-input { transition: none; }
  }
</style>
