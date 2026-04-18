<svelte:head>
  <title>Secret Message</title>
</svelte:head>

<script>
  import { onMount, tick } from 'svelte';
  import { decryptMessage, encryptMessage } from '../lib/crypto.js';

  export let params = {};
  $: token = params.token || '';

  // 'loading' | 'passcode' | 'messages' | 'error'
  let state = 'loading';
  let errorMsg = '';
  let pin = '';
  let pinError = '';
  let rawMessages = [];
  let decrypted = [];
  let unlocking = false;

  // Reply compose
  let replyText = '';
  let replyPin = '';
  let replyError = '';
  let replySent = false;
  let sending = false;
  let messagesEl;

  onMount(async () => {
    if (!token) { state = 'error'; errorMsg = 'Invalid link.'; return; }
    try {
      const res = await fetch(`/api/m/${token}`);
      const data = await res.json();
      if (!data.ok) {
        state = 'error';
        errorMsg = data.expired
          ? 'This link has expired. Ask for a new one.'
          : data.error === 'temporarily_unavailable'
            ? 'Something went wrong. Please try again later.'
            : 'Invalid link.';
        return;
      }
      rawMessages = data.messages || [];
      state = 'passcode';
    } catch {
      state = 'error';
      errorMsg = 'Could not load messages.';
    }
  });

  async function unlock() {
    if (unlocking || pin.length < 4) return;
    pinError = '';

    if (rawMessages.length === 0) {
      decrypted = [];
      state = 'messages';
      return;
    }

    unlocking = true;

    // Find first received message (fromOwner = true means owner sent it, we are the peer)
    const first = rawMessages.find(m => m.fromOwner) ?? rawMessages[0];
    try {
      await decryptMessage(first.ciphertext, first.iv, first.salt, pin);
    } catch {
      pinError = 'Wrong PIN — try again';
      unlocking = false;
      return;
    }

    const results = [];
    for (const m of rawMessages) {
      try {
        const body = m.fromOwner
          ? await decryptMessage(m.ciphertext, m.iv, m.salt, pin)
          : '[your reply — encrypted]';
        results.push({ body, fromOwner: m.fromOwner, createdAt: m.createdAt, own: !m.fromOwner });
      } catch {
        results.push({ body: '[encrypted with a different PIN]', fromOwner: m.fromOwner, createdAt: m.createdAt, own: !m.fromOwner });
      }
    }
    decrypted = results;
    unlocking = false;
    state = 'messages';
    await tick();
    if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
  }

  async function sendReply() {
    if (sending || !replyText.trim() || replyPin.length < 4) return;
    replyError = '';
    sending = true;
    try {
      const { ciphertext, iv, salt } = await encryptMessage(replyText.trim(), replyPin);
      const res = await fetch(`/api/m/${token}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ciphertext, iv, salt }),
      });
      const data = await res.json();
      if (!data.ok) {
        if (data.expired) {
          replyError = 'Link expired — cannot send.';
        } else {
          replyError = 'Send failed. Try again.';
        }
        return;
      }
      // Show sent reply as own bubble (they can't decrypt their own — by design)
      decrypted = [...decrypted, { body: null, fromOwner: false, createdAt: new Date().toISOString(), own: true }];
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

  function handlePinKeydown(e) { if (e.key === 'Enter') unlock(); }
  function handleReplyKeydown(e) { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendReply(); } }

  function digitsOnly(e, which) {
    const v = e.target.value.replace(/\D/g, '');
    if (which === 'pin') pin = v;
    else replyPin = v;
  }

  function formatTime(ts) {
    if (!ts) return '';
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
</script>

<div class="scv">
  {#if state === 'loading'}
    <div class="scv-center">
      <div class="scv-spinner"></div>
    </div>

  {:else if state === 'error'}
    <div class="scv-center">
      <div class="scv-error-icon">
        <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <circle cx="12" cy="12" r="10"/>
          <line x1="15" y1="9" x2="9" y2="15"/>
          <line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
      </div>
      <p class="scv-error-text">{errorMsg}</p>
    </div>

  {:else if state === 'passcode'}
    <div class="scv-card">
      <!-- Lock icon -->
      <div class="scv-lock-icon">
        <svg width="30" height="30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <rect x="3" y="11" width="18" height="11" rx="2"/>
          <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
      </div>
      <h1 class="scv-card-title">Secret Message</h1>
      <p class="scv-card-sub">
        {rawMessages.length > 0
          ? `${rawMessages.length} encrypted message${rawMessages.length > 1 ? 's' : ''}. Enter the PIN to read.`
          : 'Enter the PIN shared with you to continue.'}
      </p>

      <label class="scv-label" for="scv-pin">PIN</label>
      <input
        id="scv-pin"
        class="scv-pin"
        type="tel"
        inputmode="numeric"
        pattern="[0-9]*"
        maxlength="8"
        placeholder="• • • •"
        bind:value={pin}
        on:input={(e) => digitsOnly(e, 'pin')}
        on:keydown={handlePinKeydown}
        disabled={unlocking}
        autocomplete="off"
        autofocus
      />
      {#if pinError}
        <p class="scv-pin-error" role="alert">{pinError}</p>
      {/if}
      <button
        class="scv-btn"
        on:click={unlock}
        disabled={unlocking || pin.length < 4}
      >
        {unlocking ? 'Verifying…' : 'Open Messages'}
      </button>
    </div>

  {:else if state === 'messages'}
    <div class="scv-view">
      <!-- Sticky header -->
      <div class="scv-view-header">
        <div class="scv-view-lock">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <rect x="3" y="11" width="18" height="11" rx="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
        </div>
        <span class="scv-view-title">Secret Chat</span>
        <span class="scv-view-badge">E2E Encrypted</span>
      </div>

      <!-- Messages -->
      <div class="scv-msgs" bind:this={messagesEl}>
        {#if decrypted.length === 0}
          <div class="scv-empty">
            <p>No messages yet.</p>
            <span>Send a reply below</span>
          </div>
        {:else}
          {#each decrypted as msg}
            <div class="scv-msg" class:scv-msg--own={msg.own} class:scv-msg--their={!msg.own}>
              {#if msg.own}
                <div class="scv-bubble scv-bubble--own">
                  <div class="scv-encrypted-badge">
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                    Sent · encrypted
                  </div>
                </div>
              {:else}
                <div class="scv-bubble scv-bubble--their">
                  <p class="scv-body">{msg.body}</p>
                </div>
              {/if}
              <span class="scv-time">{formatTime(msg.createdAt)}</span>
            </div>
          {/each}
        {/if}

        {#if replySent}
          <div class="scv-sent-notice">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#4ade80" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
            Reply sent securely
          </div>
        {/if}
      </div>

      <!-- Reply compose -->
      <div class="scv-compose">
        <p class="scv-compose-label">Send a reply</p>
        <label class="scv-sr" for="scv-reply-text">Your reply</label>
        <textarea
          id="scv-reply-text"
          class="scv-compose-text"
          rows="2"
          maxlength="2000"
          placeholder="Type a message…"
          bind:value={replyText}
          on:keydown={handleReplyKeydown}
          disabled={sending}
        ></textarea>
        <div class="scv-compose-row">
          <div class="scv-pin-wrap">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
            <label class="scv-sr" for="scv-reply-pin">PIN for your reply</label>
            <input
              id="scv-reply-pin"
              class="scv-compose-pin"
              type="tel"
              inputmode="numeric"
              pattern="[0-9]*"
              maxlength="8"
              placeholder="Set a PIN"
              bind:value={replyPin}
              on:input={(e) => digitsOnly(e, 'reply')}
              disabled={sending}
              autocomplete="off"
            />
          </div>
          <button
            class="scv-send-btn"
            on:click={sendReply}
            disabled={sending || !replyText.trim() || replyPin.length < 4}
            aria-label="Send reply"
          >
            {#if sending}
              <div class="scv-send-spinner"></div>
            {:else}
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
            {/if}
          </button>
        </div>
        {#if replyError}
          <p class="scv-reply-error" role="alert">{replyError}</p>
        {/if}
        <p class="scv-compose-hint">The receiver needs your PIN to read this — share it separately</p>
      </div>
    </div>
  {/if}
</div>

<style>
  /* ---------- Shell ---------- */
  .scv {
    min-height: 100dvh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #0d0d14;
    padding: 16px;
    box-sizing: border-box;
  }

  /* ---------- Loading / error states ---------- */
  .scv-center {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
  }

  .scv-spinner {
    width: 32px;
    height: 32px;
    border: 2px solid rgba(255,255,255,0.08);
    border-top-color: rgba(129,140,248,0.7);
    border-radius: 50%;
    animation: scv-spin 0.8s linear infinite;
  }

  .scv-error-icon {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: rgba(248,113,113,0.1);
    border: 1px solid rgba(248,113,113,0.2);
    display: flex;
    align-items: center;
    justify-content: center;
    color: #f87171;
  }

  .scv-error-text {
    color: rgba(255,255,255,0.5);
    font-size: 14px;
    text-align: center;
    font-family: system-ui, sans-serif;
    margin: 0;
    max-width: 260px;
  }

  /* ---------- Passcode card ---------- */
  .scv-card {
    width: 100%;
    max-width: 340px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 14px;
    padding: 32px 24px;
    background: #111118;
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 20px;
    box-shadow: 0 20px 60px rgba(0,0,0,0.5);
    text-align: center;
  }

  .scv-lock-icon {
    width: 64px;
    height: 64px;
    border-radius: 50%;
    background: rgba(129,140,248,0.1);
    border: 1px solid rgba(129,140,248,0.18);
    display: flex;
    align-items: center;
    justify-content: center;
    color: rgba(129,140,248,0.75);
    margin-bottom: 4px;
  }

  .scv-card-title {
    margin: 0;
    font-size: 18px;
    font-weight: 700;
    color: #e2e8f0;
    font-family: system-ui, sans-serif;
  }

  .scv-card-sub {
    margin: 0;
    font-size: 13px;
    color: rgba(255,255,255,0.38);
    line-height: 1.6;
    font-family: system-ui, sans-serif;
    max-width: 240px;
  }

  .scv-label {
    display: block;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: rgba(255,255,255,0.35);
    font-family: system-ui, sans-serif;
    align-self: flex-start;
    padding-left: 4px;
  }

  .scv-pin {
    width: 100%;
    padding: 14px 16px;
    border-radius: 14px;
    border: 1px solid rgba(255,255,255,0.12);
    background: rgba(255,255,255,0.05);
    color: #e2e8f0;
    font-size: 26px;
    letter-spacing: 0.35em;
    text-align: center;
    outline: none;
    font-variant-numeric: tabular-nums;
    font-family: system-ui, monospace, sans-serif;
    box-sizing: border-box;
    transition: border-color 0.15s, box-shadow 0.15s;
    -webkit-appearance: none;
  }
  .scv-pin:focus {
    border-color: rgba(129,140,248,0.5);
    box-shadow: 0 0 0 3px rgba(129,140,248,0.12);
  }

  .scv-pin-error {
    color: #f87171;
    font-size: 12px;
    margin: 0;
    font-family: system-ui, sans-serif;
  }

  .scv-btn {
    width: 100%;
    padding: 14px;
    border-radius: 14px;
    border: none;
    background: rgba(129,140,248,0.85);
    color: #fff;
    font-size: 15px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s;
    font-family: system-ui, sans-serif;
    min-height: 52px;
  }
  .scv-btn:hover:not(:disabled) { background: rgba(129,140,248,1); }
  .scv-btn:disabled { opacity: 0.35; cursor: not-allowed; }

  /* ---------- Messages view ---------- */
  .scv-view {
    width: 100%;
    max-width: 500px;
    height: 100dvh;
    display: flex;
    flex-direction: column;
    background: #0d0d14;
  }

  .scv-view-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 14px 16px;
    background: #111118;
    border-bottom: 1px solid rgba(255,255,255,0.07);
    flex-shrink: 0;
  }

  .scv-view-lock {
    color: rgba(129,140,248,0.65);
    display: flex;
    align-items: center;
  }

  .scv-view-title {
    flex: 1;
    font-size: 14px;
    font-weight: 600;
    color: #e2e8f0;
    font-family: system-ui, sans-serif;
  }

  .scv-view-badge {
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: rgba(129,140,248,0.6);
    background: rgba(129,140,248,0.08);
    border: 1px solid rgba(129,140,248,0.15);
    padding: 3px 8px;
    border-radius: 20px;
    font-family: system-ui, sans-serif;
  }

  .scv-msgs {
    flex: 1;
    overflow-y: auto;
    padding: 16px 14px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    overscroll-behavior: contain;
  }
  .scv-msgs::-webkit-scrollbar { width: 4px; }
  .scv-msgs::-webkit-scrollbar-track { background: transparent; }
  .scv-msgs::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.1); border-radius: 2px; }

  .scv-empty {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    text-align: center;
    padding: 40px 0;
  }
  .scv-empty p { margin: 0; font-size: 14px; color: rgba(255,255,255,0.3); font-family: system-ui, sans-serif; }
  .scv-empty span { font-size: 12px; color: rgba(255,255,255,0.2); font-family: system-ui, sans-serif; }

  .scv-msg {
    display: flex;
    flex-direction: column;
    max-width: 78%;
    gap: 3px;
  }
  .scv-msg--own { align-self: flex-end; align-items: flex-end; }
  .scv-msg--their { align-self: flex-start; align-items: flex-start; }

  .scv-bubble {
    padding: 9px 13px;
    border-radius: 16px;
    font-size: 14px;
    line-height: 1.55;
    font-family: system-ui, sans-serif;
  }

  .scv-bubble--their {
    background: rgba(255,255,255,0.07);
    border: 1px solid rgba(255,255,255,0.06);
    border-bottom-left-radius: 4px;
    color: #e2e8f0;
    word-break: break-word;
  }

  .scv-bubble--own {
    background: rgba(129,140,248,0.13);
    border: 1px solid rgba(129,140,248,0.18);
    border-bottom-right-radius: 4px;
    display: flex;
    align-items: center;
  }

  .scv-encrypted-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    color: rgba(129,140,248,0.6);
    font-size: 12px;
    font-weight: 500;
    font-family: system-ui, sans-serif;
  }

  .scv-body { margin: 0; }

  .scv-time {
    font-size: 10px;
    color: rgba(255,255,255,0.25);
    font-family: system-ui, sans-serif;
    padding: 0 2px;
  }

  .scv-sent-notice {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: #4ade80;
    font-family: system-ui, sans-serif;
    align-self: center;
    margin-top: 4px;
  }

  /* ---------- Compose ---------- */
  .scv-compose {
    padding: 12px 14px 20px;
    border-top: 1px solid rgba(255,255,255,0.07);
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex-shrink: 0;
    background: rgba(0,0,0,0.2);
  }

  .scv-compose-label {
    margin: 0;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: rgba(255,255,255,0.3);
    font-family: system-ui, sans-serif;
  }

  .scv-compose-text {
    width: 100%;
    resize: none;
    padding: 10px 12px;
    border-radius: 12px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(255,255,255,0.05);
    color: #e2e8f0;
    font-size: 14px;
    line-height: 1.5;
    outline: none;
    box-sizing: border-box;
    font-family: system-ui, sans-serif;
    transition: border-color 0.15s;
    -webkit-appearance: none;
  }
  .scv-compose-text:focus { border-color: rgba(255,255,255,0.22); }
  .scv-compose-text::placeholder { color: rgba(255,255,255,0.25); }

  .scv-compose-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .scv-pin-wrap {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 9px 12px;
    border-radius: 12px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(255,255,255,0.04);
    color: rgba(255,255,255,0.3);
    transition: border-color 0.15s;
    min-width: 0;
  }
  .scv-pin-wrap:focus-within { border-color: rgba(255,255,255,0.2); }

  .scv-compose-pin {
    flex: 1;
    border: none;
    background: none;
    color: #e2e8f0;
    font-size: 15px;
    letter-spacing: 0.25em;
    outline: none;
    font-variant-numeric: tabular-nums;
    font-family: system-ui, monospace, sans-serif;
    min-width: 0;
    -webkit-appearance: none;
  }
  .scv-compose-pin::placeholder { color: rgba(255,255,255,0.22); letter-spacing: normal; font-size: 13px; }

  .scv-send-btn {
    width: 44px;
    height: 44px;
    border-radius: 12px;
    border: none;
    background: rgba(129,140,248,0.85);
    color: #fff;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: background 0.15s;
  }
  .scv-send-btn:hover:not(:disabled) { background: rgba(129,140,248,1); }
  .scv-send-btn:disabled { opacity: 0.3; cursor: not-allowed; }

  .scv-send-spinner {
    width: 16px;
    height: 16px;
    border: 2px solid rgba(255,255,255,0.2);
    border-top-color: #fff;
    border-radius: 50%;
    animation: scv-spin 0.8s linear infinite;
  }

  .scv-reply-error {
    margin: 0;
    font-size: 12px;
    color: #f87171;
    font-family: system-ui, sans-serif;
  }

  .scv-compose-hint {
    margin: 0;
    font-size: 10px;
    color: rgba(255,255,255,0.2);
    font-family: system-ui, sans-serif;
  }

  /* ---------- Utilities ---------- */
  .scv-sr {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0,0,0,0);
    white-space: nowrap;
    border: 0;
  }

  @keyframes scv-spin { to { transform: rotate(360deg); } }
</style>
