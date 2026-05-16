<svelte:head>
  <title>Kinnect — Secure Note</title>
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
</svelte:head>

<script>
  /**
   * SecretChatViewer — shared-link chat window (full-screen).
   * Accessed via /#/m/:token (no auth required to view, auth required to reply).
   *
   * States: loading → login | gate → messages | error
   *
   * Reuses SecretChatGate, SecretChatMessage, SecretChatCompose for identical UI.
   * Only data-wiring differs (REST API instead of WS/store).
   */
  import { onMount, onDestroy, tick } from 'svelte';
  import { fade } from 'svelte/transition';
  import { decryptMessage, encryptMessage } from '../lib/crypto.js';
  import { apiGet, apiPost, fetchCsrf, clearCsrf } from '../lib/api.js';
  import { compressImage } from '../lib/imageUtils.js';
  import SecretChatGate from '../components/SecretChatGate.svelte';
  import SecretChatMessage from '../components/SecretChatMessage.svelte';
  import SecretChatCompose from '../components/SecretChatCompose.svelte';

  export let params = {};
  $: token = params.token || '';

  // State machine: 'loading' | 'login' | 'gate' | 'messages' | 'error'
  let state = 'loading';
  let errorMsg = '';
  let errorAction = '';

  // Login state
  let loginEmail = '';
  let loginPassword = '';
  let loginError = '';
  let loginLoading = false;

  // Gate state — SecretChatGate owns PIN UI, we own the logic
  let pinError = '';
  let rawMessages = [];
  let decryptedMessages = [];
  let unlocking = false;
  let gatePin = '';
  let gateRef; // bind:this for triggerShake() / triggerSuccess()

  // Reply compose state
  let replySent = false;
  let sending = false;
  let photoSending = false;
  let messagesEl;
  let panicMode = false;

  // Per-message re-lock state
  let lockedSet = new Set();
  let lockCountdowns = {};
  let lockIntervals = {};
  const AUTO_LOCK_SECS = 30;

  // Per-message inline PIN state (one message open at a time)
  let inlineOpenId = null;
  let inlinePin = '';
  let inlineError = '';
  let inlineUnlocking = false;

  // Photo lightbox
  let lightboxSrc = '';
  let lightboxOpen = false;

  // Keyboard avoidance (iOS visualViewport)
  let keyboardOffset = 0;
  let viewEl;

  function onVVChange() {
    if (!viewEl) return;
    const vv = window.visualViewport;
    if (!vv) return;
    const kbH = Math.max(0, window.innerHeight - vv.height);
    keyboardOffset = kbH > 50 ? kbH : 0;
  }

  function restoreFromPanic() { panicMode = false; }

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
      if (!me.ok || !me.userId) { state = 'login'; return; }
      await loadInvite();
    } catch {
      state = 'error';
      errorMsg = 'Could not connect.';
      errorAction = 'Check your connection and reload the page.';
    }
    window.visualViewport?.addEventListener('resize', onVVChange);
    window.visualViewport?.addEventListener('scroll', onVVChange);
  });

  onDestroy(() => {
    for (const id of Object.keys(lockIntervals)) clearInterval(lockIntervals[id]);
    window.visualViewport?.removeEventListener('resize', onVVChange);
    window.visualViewport?.removeEventListener('scroll', onVVChange);
  });

  // Unlock — pin comes from SecretChatGate's 'submit' event
  async function unlock(pin) {
    if (unlocking || pin.length < 4) return;
    pinError = '';
    unlocking = true;

    const validatorMsg = rawMessages.find(m => m.fromOwner);
    if (validatorMsg) {
      try {
        await decryptMessage(validatorMsg.ciphertext, validatorMsg.iv, validatorMsg.salt, pin);
      } catch {
        pinError = 'Incorrect code — check your PIN and try again.';
        unlocking = false;
        gateRef?.triggerShake();
        return;
      }
    }

    const results = rawMessages.map(m => ({
      id: m.createdAt + Math.random(),
      body: null,
      own: m.fromOwner,
      createdAt: m.createdAt,
      raw: !m.fromOwner ? m : null,
      ciphertext: m.ciphertext,
    }));

    lockedSet = new Set(results.filter(m => !m.own && m.raw).map(m => m.id));
    decryptedMessages = results;
    gatePin = pin;
    unlocking = false;
    state = 'messages';
    await tick();
    if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
  }

  $: groupedMessages = decryptedMessages.map((msg, i) => {
    const prev = decryptedMessages[i - 1];
    const next = decryptedMessages[i + 1];
    const GAP = 2 * 60 * 1000;
    const samePrev = prev && prev.own === msg.own &&
      new Date(msg.createdAt) - new Date(prev.createdAt) < GAP;
    const sameNext = next && next.own === msg.own &&
      new Date(next.createdAt) - new Date(msg.createdAt) < GAP;
    return { ...msg, groupFirst: !samePrev, groupLast: !sameNext };
  });

  // Bridge viewer message shape → SecretChatMessage prop shape
  function adaptMsg(msg) {
    return {
      id: msg.id,
      ciphertext: msg.raw?.ciphertext || msg.ciphertext || '',
      iv: msg.raw?.iv || null,
      salt: msg.raw?.salt || null,
      senderId: msg.own ? '_owner_' : '_sender_',
      seenAt: null,
      createdAt: msg.createdAt,
      groupFirst: msg.groupFirst,
      groupLast: msg.groupLast,
    };
  }

  // Inline PIN handlers — one message open at a time
  function handleToggleInline(id) {
    if (inlineOpenId === id) {
      inlineOpenId = null; inlinePin = ''; inlineError = '';
    } else {
      inlineOpenId = id; inlinePin = ''; inlineError = '';
    }
  }

  function handleInlinePinInput({ id, value }) {
    if (id !== inlineOpenId) return;
    inlinePin = value;
    inlineError = '';
  }

  async function handleDecryptOne(adaptedMsg) {
    if (!adaptedMsg || inlineUnlocking || inlinePin.length < 4) return;
    inlineUnlocking = true;
    inlineError = '';
    try {
      const plain = await decryptMessage(
        adaptedMsg.ciphertext, adaptedMsg.iv, adaptedMsg.salt, inlinePin
      );
      decryptedMessages = decryptedMessages.map(m =>
        m.id === adaptedMsg.id ? { ...m, body: plain } : m
      );
      lockedSet.delete(adaptedMsg.id);
      lockedSet = new Set(lockedSet);
      inlineOpenId = null;
      inlinePin = '';
      startAutoLock(adaptedMsg.id);
    } catch {
      inlineError = 'Wrong PIN — try again.';
    } finally {
      inlineUnlocking = false;
    }
  }

  function startAutoLock(id) {
    if (lockIntervals[id]) clearInterval(lockIntervals[id]);
    lockCountdowns = { ...lockCountdowns, [id]: AUTO_LOCK_SECS };
    lockIntervals[id] = setInterval(() => {
      const cur = lockCountdowns[id];
      if (cur == null || cur <= 1) { relockMsg(id); }
      else { lockCountdowns = { ...lockCountdowns, [id]: cur - 1 }; }
    }, 1000);
  }

  function relockMsg(id) {
    if (lockIntervals[id]) { clearInterval(lockIntervals[id]); delete lockIntervals[id]; }
    delete lockCountdowns[id];
    lockCountdowns = { ...lockCountdowns };
    lockedSet = new Set([...lockedSet, id]);
    decryptedMessages = decryptedMessages.map(m => m.id === id ? { ...m, body: null } : m);
  }

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

  function goBack() {
    state = 'gate';
    gatePin = '';
    decryptedMessages = [];
    for (const id of Object.keys(lockIntervals)) clearInterval(lockIntervals[id]);
    lockIntervals = {};
    lockCountdowns = {};
    lockedSet = new Set();
    pinError = '';
    inlineOpenId = null;
    inlinePin = '';
    inlineError = '';
  }

  async function sendDirectReply(text) {
    if (!text || gatePin.length < 4 || sending) return;
    sending = true;
    try {
      const { ciphertext, iv, salt } = await encryptMessage(text, gatePin);
      const data = await apiPost(`/api/m/${token}`, { ciphertext, iv, salt });
      if (!data.ok) return;
      decryptedMessages = [...decryptedMessages, {
        id: Date.now(),
        body: text,
        own: true,
        createdAt: new Date().toISOString(),
        ciphertext,
      }];
      replySent = true;
      await tick();
      if (!userScrolledUp && messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
    } catch { /* silently drop — network error */ }
    finally { sending = false; }
  }

  async function handlePhotoFromCompose(file) {
    if (!file || gatePin.length < 4) return;
    photoSending = true;
    try {
      const dataUrl = await compressImage(file);
      await sendDirectReply(`[photo:${dataUrl}]`);
    } catch { /* silently drop */ }
    finally { photoSending = false; }
  }

  // Formatters (own messages only — their messages use SecretChatMessage's internal formatters)
  function clockTime(ts) {
    if (!ts) return '';
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
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

  const PHOTO_RE = /^\[photo:(data:image\/[^;]+;base64,[^\]]+)\]$/;
  const GIF_RE   = /^\[gif:(https?:\/\/[^\]]+)\]$/;
  function parsePhoto(text) { if (!text) return null; const m = PHOTO_RE.exec(text); return m ? m[1] : null; }
  function parseGif(text)   { const m = GIF_RE.exec(text); return m ? m[1] : null; }

  function ciphertextGibberish(ct) {
    if (!ct) return '···';
    const s = Math.min(4, ct.length);
    const r = ct.slice(s, s + 28);
    return r.length >= 6 ? r + '…' : '···';
  }

  function fakeGibberish(ts) {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
    const seed = new Date(ts).getTime() || Date.now();
    let s = '';
    let x = seed % 2147483647;
    for (let i = 0; i < 28; i++) { x = (x * 16807) % 2147483647; s += chars[x % 64]; }
    return s + '…';
  }
</script>

<!-- ── Panic overlay ─────────────────────────────────────────────── -->
{#if panicMode}
  <div
    class="scv-panic"
    role="alertdialog"
    aria-label="Screen blanked for privacy. Tap to restore."
    aria-live="assertive"
    tabindex="0"
    on:click={restoreFromPanic}
    on:keydown={(e) => (e.key === 'Enter' || e.key === ' ' || e.key === 'Escape') && restoreFromPanic()}
  ></div>
{/if}

<!-- ── Photo lightbox ────────────────────────────────────────────── -->
{#if lightboxOpen}
  <div
    class="scv-lightbox"
    role="dialog"
    aria-label="Photo viewer"
    aria-modal="true"
    tabindex="0"
    on:click={() => lightboxOpen = false}
    on:keydown={(e) => e.key === 'Escape' && (lightboxOpen = false)}
    transition:fade={{ duration: 150 }}
  >
    <img src={lightboxSrc} class="scv-lightbox-img" alt="Expanded photo" />
    <button
      class="scv-lightbox-close"
      on:click|stopPropagation={() => lightboxOpen = false}
      aria-label="Close photo"
      type="button"
    >
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
        <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
      </svg>
    </button>
  </div>
{/if}

<!-- ── Root container ─────────────────────────────────────────────── -->
<div class="scv" class:scv--nocyber={state !== 'messages'}>

  <!-- ── Loading ──────────────────────────────────────────────────── -->
  {#if state === 'loading'}
    <div class="scv-center" role="status" aria-busy="true" aria-label="Loading">
      <div class="scv-spinner" aria-hidden="true"></div>
      <p class="scv-loading-text">Loading…</p>
    </div>

  <!-- ── Error ────────────────────────────────────────────────────── -->
  {:else if state === 'error'}
    <div class="scv-center" role="alert">
      <div class="scv-status-icon scv-status-icon--error" aria-hidden="true">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
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
    <div class="scv-gate-wrap" role="region" aria-label="Sign in to Kinnect">
      <div class="scv-gate-content">
        <div class="scv-gate-icon" aria-hidden="true">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/>
            <polyline points="10 17 15 12 10 7"/>
            <line x1="15" y1="12" x2="3" y2="12"/>
          </svg>
        </div>

        <div class="scv-gate-text">
          <h1 class="scv-gate-title">Sign in to Kinnect</h1>
          <p class="scv-gate-sub">You need an account to read this note.</p>
        </div>

        <form class="scv-login-form" on:submit|preventDefault={doLogin} novalidate>
          <div class="scv-field">
            <label class="scv-sr" for="scv-login-email">Email address</label>
            <input
              id="scv-login-email"
              class="scv-input"
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
          <div class="scv-field">
            <label class="scv-sr" for="scv-login-pass">Password</label>
            <input
              id="scv-login-pass"
              class="scv-input"
              type="password"
              autocomplete="current-password"
              placeholder="Password"
              bind:value={loginPassword}
              disabled={loginLoading}
            />
          </div>

          {#if loginError}
            <p class="scv-field-err" role="alert">{loginError}</p>
          {/if}

          <button
            class="scv-cta-btn"
            class:scv-cta-btn--active={loginEmail.trim().length > 0 && loginPassword.length > 0}
            type="submit"
            disabled={loginLoading || !loginEmail.trim() || !loginPassword}
          >
            {#if loginLoading}
              <span class="scv-btn-ring" aria-hidden="true"></span>
              <span>Signing in…</span>
            {:else}
              <span>Continue</span>
            {/if}
          </button>
        </form>

        <p class="scv-gate-footer" aria-hidden="true">
          <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          End-to-end encrypted note
        </p>
      </div>
    </div>

  <!-- ── Gate (PIN entry) — SecretChatGate fills full height ──────── -->
  {:else if state === 'gate'}
    <div class="scv-gate-outer">
      <SecretChatGate
        bind:this={gateRef}
        peerName="Sender"
        {unlocking}
        error={pinError}
        on:submit={(e) => unlock(e.detail)}
      />
    </div>

  <!-- ── Messages ─────────────────────────────────────────────────── -->
  {:else if state === 'messages'}
    <div
      class="scv-view"
      bind:this={viewEl}
      style="transform: translateY({keyboardOffset > 0 ? `-${keyboardOffset}px` : '0'}); transition: transform 0.2s var(--ease-out, cubic-bezier(0.16,1,0.3,1));"
    >
      <!-- Header -->
      <header class="scv-header">
        <button
          class="scv-header-btn"
          on:click={goBack}
          aria-label="Lock and go back to passcode"
          type="button"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <rect x="3" y="11" width="18" height="11" rx="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
          <span class="scv-header-btn-label">Lock</span>
        </button>

        <div class="scv-header-center" aria-hidden="true">
          <div class="scv-header-status-dot"></div>
          <span class="scv-header-label">Kinnect · Secure Note</span>
        </div>

        <button
          class="scv-panic-btn"
          on:click={() => panicMode = true}
          aria-label="Hide screen — tap anywhere to restore"
          type="button"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
            <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
            <line x1="1" y1="1" x2="23" y2="23"/>
          </svg>
          <span>Hide Screen</span>
        </button>
      </header>

      <!-- Messages list -->
      <main
        class="scv-msgs"
        bind:this={messagesEl}
        on:scroll={handleMessagesScroll}
        role="log"
        aria-live="polite"
        aria-label="Secure messages"
      >
        {#if groupedMessages.length === 0}
          <div class="scv-empty" role="status">
            <div class="scv-empty-icon" aria-hidden="true">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
              </svg>
            </div>
            <p class="scv-empty-title">No messages yet</p>
            <p class="scv-empty-sub">Type a reply below to start the conversation.</p>
          </div>
        {/if}

        {#each groupedMessages as msg, i (msg.id)}
          {@const label = dateLabel(msg.createdAt, i > 0 ? groupedMessages[i-1].createdAt : null)}

          {#if label}
            <div class="scv-date-div" role="separator" aria-label={label}>
              <span>{label}</span>
            </div>
          {/if}

          {#if msg.own}
            <!-- Own (viewer's) messages rendered inline — shows plaintext of replies -->
            <div
              class="scv-msg scv-msg--own"
              class:scv-msg--group-cont={!msg.groupLast}
            >
              <div
                class="scv-bubble scv-bubble--own"
                class:scv-bubble--grp-first={msg.groupFirst}
                class:scv-bubble--grp-last={msg.groupLast}
                class:scv-bubble--grp-mid={!msg.groupFirst && !msg.groupLast}
              >
                {#if msg.body && parsePhoto(msg.body)}
                  <img src={parsePhoto(msg.body)} class="scv-msg-photo" alt="Photo you sent" loading="lazy" />
                {:else if msg.body && parseGif(msg.body)}
                  <img src={parseGif(msg.body)} class="scv-msg-sticker" alt="Sticker" loading="lazy" />
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
            </div>
          {:else}
            <!-- Sender's messages — inline PIN entry per message -->
            <SecretChatMessage
              msg={adaptMsg(msg)}
              plain={msg.body}
              isOwn={false}
              showInline={inlineOpenId === msg.id}
              inlinePin={inlineOpenId === msg.id ? inlinePin : ''}
              inlineError={inlineOpenId === msg.id ? inlineError : ''}
              inlineUnlocking={inlineOpenId === msg.id && inlineUnlocking}
              {lockedSet}
              lockCountdown={lockCountdowns[msg.id] ?? null}
              deletingMsgId={null}
              myId="_owner_"
              peerFirst="Sender"
              seenPulse={false}
              on:toggleInline={(e) => handleToggleInline(e.detail)}
              on:inlinePinInput={(e) => handleInlinePinInput(e.detail)}
              on:decryptOne={(e) => handleDecryptOne(e.detail)}
              on:relock={(e) => relockMsg(e.detail)}
              on:photoExpand={(e) => { lightboxSrc = e.detail; lightboxOpen = true; }}
            />
          {/if}
        {/each}

        {#if replySent}
          <div class="scv-sent-notice" role="status" transition:fade={{ duration: 300 }}>
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

      <!-- Compose — SecretChatCompose owns text/emoji/sticker/photo UI -->
      <SecretChatCompose
        peerFirst="Sender"
        {sending}
        {photoSending}
        hasPin={gatePin.length >= 4}
        on:sendText={(e) => sendDirectReply(e.detail)}
        on:sendPhoto={(e) => handlePhotoFromCompose(e.detail)}
        on:sendSticker={(e) => sendDirectReply(e.detail)}
        on:panic={() => (panicMode = true)}
      />
    </div>
  {/if}
</div>

<style>
  /* ── Chat accent token system ─────────────────────────────────── */
  :root {
    --scv-accent:        var(--primary-500, #14b8a6);
    --scv-accent-dim:    rgba(20, 184, 166, 0.18);
    --scv-accent-subtle: rgba(20, 184, 166, 0.08);
    --scv-accent-glow:   rgba(20, 184, 166, 0.28);
    --scv-bg:            #060610;
    --scv-surface:       #0a0a18;
    --scv-elevated:      #0f0f20;
    --scv-border:        rgba(255, 255, 255, 0.07);
    --scv-border-accent: rgba(20, 184, 166, 0.22);
  }

  /* ── Root shell ───────────────────────────────────────────────── */
  .scv {
    min-height: 100dvh;
    min-height: 100svh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--scv-bg);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    position: relative;
    /* NO overflow:hidden — creates containing block for position:fixed on iOS Chrome */

    /* Bridge --chat-* tokens used by SecretChatGate and SecretChatMessage
       CSS custom properties cascade through Svelte component boundaries */
    --chat-accent:        var(--scv-accent);
    --chat-accent-subtle: var(--scv-accent-subtle);
    --chat-accent-dim:    var(--scv-accent-dim);
    --chat-accent-glow:   var(--scv-accent-glow);
    --chat-border-accent: var(--scv-border-accent);
    --chat-border:        var(--scv-border);
    --chat-bg:            var(--scv-bg);
    --chat-surface:       var(--scv-surface);
    --chat-elevated:      var(--scv-elevated);
  }

  /* Ambient teal mesh — only shown in messages state */
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

  /* Hex grid texture — only in messages state */
  .scv::after {
    content: '';
    position: fixed;
    inset: 0;
    background-image:
      repeating-linear-gradient(
        0deg, transparent, transparent 28px,
        rgba(20, 184, 166, 0.018) 28px, rgba(20, 184, 166, 0.018) 29px
      ),
      repeating-linear-gradient(
        60deg, transparent, transparent 28px,
        rgba(20, 184, 166, 0.012) 28px, rgba(20, 184, 166, 0.012) 29px
      );
    pointer-events: none;
    z-index: 0;
    animation: scv-hex-drift 24s linear infinite;
  }

  @keyframes scv-hex-drift {
    from { background-position: 0 0, 0 0; }
    to   { background-position: 0 56px, 48px 0; }
  }

  .scv--nocyber::before,
  .scv--nocyber::after { display: none; }

  .scv > * { position: relative; z-index: 1; }

  /* ── Loading / Error center ───────────────────────────────────── */
  .scv-center {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-4, 16px);
    text-align: center;
    padding: var(--space-8, 32px);
  }

  .scv-spinner {
    width: 32px;
    height: 32px;
    border-radius: var(--radius-full, 9999px);
    border: 2px solid rgba(255, 255, 255, 0.12);
    border-top-color: rgba(255, 255, 255, 0.55);
    animation: scv-spin 0.8s linear infinite;
  }

  .scv-loading-text {
    margin: 0;
    font-size: var(--text-sm, 0.875rem);
    color: rgba(255, 255, 255, 0.4);
  }

  .scv-status-icon {
    width: 56px;
    height: 56px;
    border-radius: var(--radius-full, 9999px);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .scv-status-icon--error {
    background: rgba(248, 113, 113, 0.08);
    border: 1px solid rgba(248, 113, 113, 0.22);
    color: var(--danger-400, #f87171);
  }

  .scv-err-msg {
    margin: 0;
    font-size: var(--text-base, 1rem);
    font-weight: 600;
    color: rgba(255, 255, 255, 0.75);
    max-width: 260px;
  }

  .scv-err-action {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    color: rgba(255, 255, 255, 0.35);
    max-width: 240px;
    line-height: var(--leading-relaxed, 1.625);
  }

  /* ── Login wrapper (keeps original scv-gate-wrap design) ─────── */
  .scv-gate-wrap {
    width: 100%;
    min-height: 100dvh;
    min-height: 100svh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-8, 32px) var(--space-4, 16px);
    box-sizing: border-box;
    background: rgba(6, 6, 16, 0.94);
  }

  @media (min-width: 768px) {
    .scv-gate-wrap {
      background: rgba(6, 6, 16, 0.72);
      backdrop-filter: blur(20px) saturate(1.4);
      -webkit-backdrop-filter: blur(20px) saturate(1.4);
    }
  }

  .scv-gate-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-5, 20px);
    width: 100%;
    max-width: 320px;
    text-align: center;
    animation: scv-gate-in 0.4s var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
  }

  @keyframes scv-gate-in {
    from { opacity: 0; transform: translateY(20px) scale(0.96); }
    to   { opacity: 1; transform: translateY(0) scale(1); }
  }

  .scv-gate-icon {
    width: 88px;
    height: 88px;
    border-radius: var(--radius-full, 9999px);
    background: var(--scv-accent-subtle);
    border: 1px solid var(--scv-border-accent);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--scv-accent);
    box-shadow:
      0 0 0 14px rgba(20, 184, 166, 0.04),
      0 0 0 28px rgba(20, 184, 166, 0.02),
      0 0 48px rgba(20, 184, 166, 0.24);
    animation: scv-icon-breathe 5s ease-in-out infinite;
    flex-shrink: 0;
  }

  @keyframes scv-icon-breathe {
    0%, 100% { box-shadow: 0 0 0 14px rgba(20,184,166,0.05), 0 0 0 28px rgba(20,184,166,0.02), 0 0 48px rgba(20,184,166,0.24); }
    50%       { box-shadow: 0 0 0 20px rgba(20,184,166,0.03), 0 0 0 38px rgba(20,184,166,0.01), 0 0 72px rgba(20,184,166,0.35); }
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
    color: rgba(255, 255, 255, 0.92);
    letter-spacing: -0.01em;
  }

  .scv-gate-sub {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    color: rgba(255, 255, 255, 0.36);
    line-height: var(--leading-relaxed, 1.625);
  }

  /* ── Login form ───────────────────────────────────────────────── */
  .scv-login-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-3, 12px);
    width: 100%;
  }

  .scv-field { width: 100%; }

  .scv-input {
    width: 100%;
    box-sizing: border-box;
    min-height: 52px;
    padding: 0 var(--space-4, 16px);
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid var(--scv-border-accent);
    border-radius: var(--radius-lg, 14px);
    color: rgba(255, 255, 255, 0.92);
    font-size: 16px; /* 16px min — prevents iOS auto-zoom on focus */
    font-family: var(--font-sans, 'Nunito', sans-serif);
    font-weight: 500;
    caret-color: var(--scv-accent);
    outline: none;
    transition: border-color 0.15s, box-shadow 0.15s;
    -webkit-appearance: none;
    appearance: none;
  }

  .scv-input::placeholder { color: rgba(255, 255, 255, 0.28); }

  .scv-input:focus {
    border-color: var(--scv-accent);
    box-shadow: 0 0 0 3px var(--scv-accent-subtle);
  }

  .scv-input:disabled { opacity: 0.5; }

  .scv-field-err {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    color: var(--danger-400, #f87171);
    font-weight: 500;
    text-align: center;
  }

  /* ── CTA button (login) ───────────────────────────────────────── */
  .scv-cta-btn {
    width: 100%;
    max-width: 280px;
    padding: var(--space-4, 16px);
    border-radius: var(--radius-lg, 14px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    background: rgba(255, 255, 255, 0.06);
    color: rgba(255, 255, 255, 0.45);
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

  .scv-cta-btn:disabled { opacity: 0.3; cursor: not-allowed; }

  .scv-cta-btn:focus-visible {
    outline: 2px solid var(--scv-accent);
    outline-offset: 2px;
  }

  .scv-cta-btn--active {
    background: linear-gradient(135deg, var(--primary-400, #2dd4bf) 0%, var(--primary-600, #0d9488) 100%);
    color: #fff;
    border-color: transparent;
    box-shadow: 0 4px 22px rgba(20, 184, 166, 0.42);
  }

  .scv-cta-btn--active:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 6px 30px rgba(20, 184, 166, 0.58);
  }

  .scv-cta-btn--active:active:not(:disabled) { transform: scale(0.97); }

  .scv-btn-ring {
    width: 16px;
    height: 16px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: #fff;
    border-radius: var(--radius-full, 9999px);
    animation: scv-spin 0.7s linear infinite;
    flex-shrink: 0;
  }

  .scv-gate-footer {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: rgba(255, 255, 255, 0.14);
    letter-spacing: 0.03em;
    margin: 0;
  }

  /* ── Gate outer — full-height flex column for SecretChatGate ──── */
  .scv-gate-outer {
    width: 100%;
    height: 100dvh;
    height: 100svh;
    display: flex;
    flex-direction: column;
    /* SecretChatGate's .gate uses flex:1 — fills this container */
  }

  /* ── Messages view ────────────────────────────────────────────── */
  .scv-view {
    width: 100%;
    max-width: 520px;
    height: 100dvh;
    height: 100svh;
    display: flex;
    flex-direction: column;
    /* NO overflow:hidden — clips position:fixed pickers/panic on iOS Chrome */
  }

  /* ── Header ───────────────────────────────────────────────────── */
  header.scv-header {
    display: flex;
    align-items: center;
    gap: var(--space-2, 8px);
    padding:
      max(var(--space-3, 12px), env(safe-area-inset-top, 0px))
      var(--space-3, 12px)
      var(--space-3, 12px)
      var(--space-3, 12px);
    background: rgba(6, 6, 16, 0.85);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border-bottom: 1px solid var(--scv-border);
    flex-shrink: 0;
    min-height: 56px;
  }

  .scv-header-btn {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    min-width: 44px;
    min-height: 44px;
    padding: 0 var(--space-2, 8px);
    background: none;
    border: none;
    border-radius: var(--radius-sm2, 8px);
    cursor: pointer;
    color: rgba(255, 255, 255, 0.45);
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    font-weight: 600;
    flex-shrink: 0;
    transition: color 0.1s, background 0.1s;
    touch-action: manipulation;
  }

  .scv-header-btn:hover { color: rgba(255, 255, 255, 0.8); background: rgba(255, 255, 255, 0.06); }

  .scv-header-btn:focus-visible { outline: 2px solid var(--scv-accent); outline-offset: 2px; }

  /* Hide Screen panic button — prominent amber pill so anyone can find it */
  .scv-panic-btn {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    min-height: 44px;
    padding: 0 var(--space-3, 12px);
    background: rgba(251, 191, 36, 0.12);
    border: 1px solid rgba(251, 191, 36, 0.28);
    border-radius: var(--radius-full, 9999px);
    cursor: pointer;
    color: rgb(251, 191, 36);
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    font-weight: 700;
    flex-shrink: 0;
    white-space: nowrap;
    transition: background 0.1s, border-color 0.1s, color 0.1s;
    touch-action: manipulation;
  }

  .scv-panic-btn:hover { background: rgba(251, 191, 36, 0.22); border-color: rgba(251, 191, 36, 0.5); }
  .scv-panic-btn:focus-visible { outline: 2px solid rgb(251, 191, 36); outline-offset: 2px; }

  .scv-header-btn-label { font-size: var(--text-xs, 0.75rem); font-weight: 600; }

  .scv-header-center {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-1-5, 6px);
  }

  .scv-header-status-dot {
    width: 6px;
    height: 6px;
    border-radius: var(--radius-full, 9999px);
    background: var(--scv-accent);
    box-shadow: 0 0 0 3px var(--scv-accent-subtle);
    animation: scv-pulse-dot 2.4s ease-in-out infinite;
    flex-shrink: 0;
  }

  @keyframes scv-pulse-dot {
    0%, 100% { box-shadow: 0 0 0 0 var(--scv-accent-glow); }
    50%       { box-shadow: 0 0 0 5px rgba(20, 184, 166, 0); }
  }

  .scv-header-label {
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: rgba(255, 255, 255, 0.3);
    letter-spacing: 0.04em;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* ── Messages area ────────────────────────────────────────────── */
  main.scv-msgs {
    flex: 1;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    padding: var(--space-4, 16px) var(--space-4, 16px) var(--space-2-5, 10px);
    display: flex;
    flex-direction: column;
    gap: var(--space-1, 4px);
    overscroll-behavior: contain;
    min-height: 0; /* allows flex shrink on iOS */
  }

  main.scv-msgs::-webkit-scrollbar { width: 3px; }
  main.scv-msgs::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.08);
    border-radius: var(--radius-full, 9999px);
  }

  /* ── Empty state ──────────────────────────────────────────────── */
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
    width: 64px;
    height: 64px;
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
    color: rgba(255, 255, 255, 0.75);
  }

  .scv-empty-sub {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    color: rgba(255, 255, 255, 0.28);
    line-height: var(--leading-relaxed, 1.625);
  }

  /* ── Date divider ─────────────────────────────────────────────── */
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

  /* ── Own message rows ─────────────────────────────────────────── */
  .scv-msg {
    display: flex;
    flex-direction: column;
    max-width: 76%;
    gap: 2px;
    margin-bottom: var(--space-1, 4px);
    animation: scv-msg-in 0.22s var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
  }

  .scv-msg--group-cont { margin-bottom: 1px; }
  .scv-msg--own { align-self: flex-end; align-items: flex-end; }

  @keyframes scv-msg-in {
    from { opacity: 0; transform: translateY(10px) scale(0.97); }
    to   { opacity: 1; transform: translateY(0) scale(1); }
  }

  .scv-bubble {
    padding: var(--space-2-5, 10px) var(--space-3-5, 14px);
    border-radius: var(--radius-xl, 20px);
    font-size: var(--text-sm, 0.875rem);
    line-height: var(--leading-relaxed, 1.625);
    font-family: var(--font-sans, 'Nunito', sans-serif);
  }

  /* iMessage-style grouping radius — own side */
  .scv-bubble--own.scv-bubble--grp-first { border-bottom-right-radius: var(--radius-sm, 6px); }
  .scv-bubble--own.scv-bubble--grp-last  { border-top-right-radius: var(--radius-sm, 6px); }
  .scv-bubble--own.scv-bubble--grp-mid   { border-top-right-radius: var(--radius-sm, 6px); border-bottom-right-radius: var(--radius-sm, 6px); }

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

  .scv-body { margin: 0; color: rgba(255, 255, 255, 0.92); }

  .scv-cipher-text {
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    font-size: var(--text-2xs, 0.6875rem);
    letter-spacing: 0.04em;
    color: rgba(20, 184, 166, 0.55);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
    user-select: none;
  }

  .scv-lock-icon {
    color: rgba(20, 184, 166, 0.45);
    flex-shrink: 0;
    display: flex;
    align-items: center;
  }

  :global(.scv-msg-photo) {
    max-width: 240px;
    max-height: 280px;
    border-radius: var(--radius-md, 10px);
    display: block;
    object-fit: cover;
  }

  :global(.scv-msg-sticker) {
    max-width: 128px;
    max-height: 128px;
    border-radius: var(--radius-sm2, 8px);
    display: block;
  }

  .scv-time {
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: rgba(255, 255, 255, 0.25);
    font-variant-numeric: tabular-nums;
    padding: 0 2px;
  }

  .scv-sent-notice {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    font-size: var(--text-xs, 0.75rem);
    color: var(--success-400, #34d399);
    align-self: center;
    margin-top: var(--space-1, 4px);
  }

  /* ── Scroll FAB ───────────────────────────────────────────────── */
  .scv-scroll-fab {
    align-self: flex-end;
    margin: calc(-1 * var(--space-2, 8px)) var(--space-3, 12px) 0;
    width: 44px;
    height: 44px;
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
    transition: background 0.1s, transform 0.1s, box-shadow 0.1s;
    z-index: 10;
    flex-shrink: 0;
    position: relative;
  }

  .scv-scroll-fab:hover {
    background: var(--scv-accent-subtle);
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.6), 0 0 20px var(--scv-accent-glow);
  }

  .scv-scroll-fab:focus-visible { outline: 2px solid var(--scv-accent); outline-offset: 2px; }

  /* ── Photo lightbox ───────────────────────────────────────────── */
  .scv-lightbox {
    position: fixed;
    inset: 0;
    z-index: calc(var(--z-topmost, 9000) - 1);
    background: rgba(0, 0, 0, 0.92);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4, 16px);
    cursor: zoom-out;
  }

  .scv-lightbox-img {
    max-width: 100%;
    max-height: 90dvh;
    border-radius: var(--radius-lg, 14px);
    object-fit: contain;
    box-shadow: 0 16px 64px rgba(0, 0, 0, 0.8);
  }

  .scv-lightbox-close {
    position: absolute;
    top: max(var(--space-4, 16px), env(safe-area-inset-top, 16px));
    right: var(--space-4, 16px);
    width: 44px;
    height: 44px;
    border-radius: var(--radius-full, 9999px);
    border: 1px solid rgba(255, 255, 255, 0.15);
    background: rgba(0, 0, 0, 0.6);
    color: rgba(255, 255, 255, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    touch-action: manipulation;
    transition: background 0.1s, color 0.1s;
  }

  .scv-lightbox-close:hover { background: rgba(255, 255, 255, 0.12); color: #fff; }
  .scv-lightbox-close:focus-visible { outline: 2px solid var(--scv-accent); outline-offset: 2px; }

  /* ── Panic overlay ────────────────────────────────────────────── */
  .scv-panic {
    position: fixed;
    inset: 0;
    background: #000;
    z-index: var(--z-topmost, 9000);
    cursor: default;
    animation: scv-panic-on 0.15s ease-out both;
  }

  @keyframes scv-panic-on {
    from { opacity: 0; }
    to   { opacity: 1; }
  }

  /* ── Utility ──────────────────────────────────────────────────── */
  .scv-sr {
    position: absolute;
    width: 1px; height: 1px;
    padding: 0; margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }

  @keyframes scv-spin { to { transform: rotate(360deg); } }

  /* ── prefers-reduced-motion ───────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .scv::before, .scv::after { animation: none; }
    .scv-gate-content { animation: none; }
    .scv-gate-icon { animation: none; }
    .scv-header-status-dot { animation: none; }
    .scv-msg { animation: none; }
    .scv-spinner, .scv-btn-ring { animation: none; }
    .scv-panic { animation: none; }
    .scv-cta-btn { transition: none; }
    .scv-cta-btn--active:hover { transform: none; }
    .scv-scroll-fab:hover { transform: none; }
  }
</style>
