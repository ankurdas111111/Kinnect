<svelte:head>
  <title>Kinnect</title>
</svelte:head>

<script>
  import { onMount, tick } from 'svelte';
  import { decryptMessage, encryptMessage } from '../lib/crypto.js';

  export let params = {};
  $: token = params.token || '';

  // 'loading' | 'gate' | 'messages' | 'error'
  let state = 'loading';
  let errorMsg = '';
  let pin = '';
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

  // Per-message inline decrypt (messages locked by default after gate)
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

    if (rawMessages.length === 0) {
      gatePin = pin;
      decrypted = [];
      state = 'messages';
      return;
    }

    unlocking = true;
    const first = rawMessages.find(m => m.fromOwner) ?? rawMessages[0];
    try {
      await decryptMessage(first.ciphertext, first.iv, first.salt, pin);
    } catch {
      pinError = 'Wrong code';
      unlocking = false;
      return;
    }

    // Build message list — owner messages stay locked (inline decrypt), own replies opaque
    const results = [];
    for (const m of rawMessages) {
      if (!m.fromOwner) {
        results.push({ id: m.createdAt + Math.random(), body: null, own: true, createdAt: m.createdAt });
      } else {
        results.push({ id: m.createdAt + Math.random(), body: null, own: false, createdAt: m.createdAt, raw: m });
      }
    }
    decrypted = results;
    gatePin = pin;
    unlocking = false;
    state = 'messages';
    await tick();
    if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
  }

  // ── Per-message inline decrypt (mirrors SecretChatPanel) ──────────
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

  // ── Reply (uses gate PIN) ─────────────────────────────────────────
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

  function handlePinKeydown(e) { if (e.key === 'Enter') unlock(); }
  function handleReplyKeydown(e) { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendReply(); } }
  function digitsOnly(e) { pin = e.target.value.replace(/\D/g, ''); }

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
</script>

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

  <!-- ── Gate — decoy: looks like a generic access screen ────────── -->
  {:else if state === 'gate'}
    <div class="scv-gate">
      <div class="scv-gate-icon" aria-hidden="true">
        <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
        </svg>
      </div>
      <label class="scv-sr" for="scv-pin">Access code</label>
      <input
        id="scv-pin"
        class="scv-gate-pin"
        type="tel"
        inputmode="numeric"
        pattern="[0-9]*"
        maxlength="8"
        placeholder="• • • •"
        bind:value={pin}
        on:input={digitsOnly}
        on:keydown={handlePinKeydown}
        disabled={unlocking}
        autocomplete="off"
        autofocus
      />
      {#if pinError}
        <p class="scv-gate-err" role="alert">{pinError}</p>
      {/if}
      <button class="scv-gate-btn" on:click={unlock} disabled={unlocking || pin.length < 4}>
        {unlocking ? '…' : 'Continue'}
      </button>
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
      </div>

      <div class="scv-msgs" bind:this={messagesEl}>
        {#if decrypted.length === 0}
          <div class="scv-empty">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" aria-hidden="true"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
            <p>No messages yet</p>
            <span>Send a reply below</span>
          </div>
        {/if}

        {#each decrypted as msg (msg.id)}
          {@const showInline = activeDecryptId === msg.id}
          {@const isDecrypted = !msg.own && msg.body !== null && !lockedSet.has(msg.id)}

          <div class="scv-msg" class:scv-msg--own={msg.own} class:scv-msg--their={!msg.own}>

            {#if msg.own}
              <!-- Sent reply — opaque, can't decrypt own -->
              <div class="scv-bubble scv-bubble--own">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                <span>Sent · encrypted</span>
              </div>
              <span class="scv-time">{clockTime(msg.createdAt)}</span>

            {:else if isDecrypted}
              <!-- Received + decrypted -->
              <div class="scv-bubble scv-bubble--their scv-bubble--decrypted">
                <p class="scv-body">{msg.body}</p>
                <button class="scv-relock-btn" on:click={() => relockMsg(msg.id)} aria-label="Lock message">
                  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                </button>
              </div>
              <div class="scv-meta">
                {#if lockCountdowns[msg.id] != null}
                  <span class="scv-countdown">Locks in {lockCountdowns[msg.id]}s</span>
                {:else}
                  <span class="scv-time">{clockTime(msg.createdAt)}</span>
                {/if}
              </div>

            {:else}
              <!-- Received + locked -->
              <button
                class="scv-bubble scv-bubble--locked"
                class:scv-bubble--locked-active={showInline}
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
                    placeholder="PIN"
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

              <span class="scv-time">{clockTime(msg.createdAt)}</span>
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
        {#if replyError}
          <p class="scv-reply-err" role="alert">{replyError}</p>
        {/if}
        <p class="scv-compose-hint">
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          Encrypted with your PIN · share PIN separately
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
    padding: 16px;
    box-sizing: border-box;
    font-family: system-ui, sans-serif;
  }

  /* ── Loading / error ───────────────────────────────────────────── */
  .scv-center { display: flex; flex-direction: column; align-items: center; gap: 16px; }
  .scv-spinner { width: 28px; height: 28px; border: 2px solid rgba(255,255,255,0.07); border-top-color: rgba(129,140,248,0.7); border-radius: 50%; animation: scv-spin 0.8s linear infinite; }
  .scv-icon-ring--err { width: 52px; height: 52px; border-radius: 50%; background: rgba(248,113,113,0.08); border: 1px solid rgba(248,113,113,0.2); display: flex; align-items: center; justify-content: center; color: #f87171; }
  .scv-err-txt { color: rgba(255,255,255,0.45); font-size: 14px; text-align: center; margin: 0; max-width: 240px; }

  /* ── Gate — deliberately neutral, no chat hints ────────────────── */
  .scv-gate {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    width: 100%;
    max-width: 280px;
  }
  .scv-gate-icon {
    width: 52px; height: 52px;
    border-radius: 50%;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.08);
    display: flex; align-items: center; justify-content: center;
    color: rgba(255,255,255,0.2);
    margin-bottom: 8px;
  }
  .scv-gate-pin {
    width: 100%;
    padding: 14px 16px;
    border-radius: 14px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(255,255,255,0.04);
    color: #e2e8f0;
    font-size: 26px;
    letter-spacing: 0.35em;
    text-align: center;
    outline: none;
    font-variant-numeric: tabular-nums;
    font-family: system-ui, monospace, sans-serif;
    box-sizing: border-box;
    -webkit-appearance: none;
    transition: border-color 0.15s;
  }
  .scv-gate-pin:focus { border-color: rgba(129,140,248,0.4); }
  .scv-gate-err { color: #f87171; font-size: 12px; margin: 0; }
  .scv-gate-btn {
    width: 100%;
    padding: 13px;
    border-radius: 13px;
    border: none;
    background: rgba(255,255,255,0.07);
    color: rgba(255,255,255,0.6);
    font-size: 14px; font-weight: 500;
    cursor: pointer;
    min-height: 48px;
    transition: background 0.15s, color 0.15s;
    touch-action: manipulation;
  }
  .scv-gate-btn:hover:not(:disabled) { background: rgba(255,255,255,0.1); color: rgba(255,255,255,0.85); }
  .scv-gate-btn:disabled { opacity: 0.3; cursor: not-allowed; }

  /* ── Messages view ─────────────────────────────────────────────── */
  .scv-view { width: 100%; max-width: 500px; height: 100dvh; display: flex; flex-direction: column; background: #0d0d14; }

  .scv-header { display: flex; align-items: center; gap: 10px; padding: 13px 16px; background: rgba(255,255,255,0.025); border-bottom: 1px solid rgba(255,255,255,0.06); flex-shrink: 0; }
  .scv-header-lock { color: rgba(129,140,248,0.65); display: flex; align-items: center; flex-shrink: 0; }
  .scv-header-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
  .scv-header-title { font-size: 14px; font-weight: 600; color: #e2e8f0; }
  .scv-header-sub { font-size: 11px; color: rgba(255,255,255,0.35); }
  .scv-e2e-badge { font-size: 9px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; color: rgba(129,140,248,0.6); background: rgba(129,140,248,0.08); border: 1px solid rgba(129,140,248,0.15); padding: 2px 7px; border-radius: 20px; white-space: nowrap; flex-shrink: 0; }

  .scv-msgs { flex: 1; overflow-y: auto; padding: 14px 14px 10px; display: flex; flex-direction: column; gap: 4px; overscroll-behavior: contain; }
  .scv-msgs::-webkit-scrollbar { width: 3px; }
  .scv-msgs::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.08); border-radius: 2px; }

  .scv-empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; color: rgba(255,255,255,0.18); padding: 40px 0; text-align: center; }
  .scv-empty p { margin: 0; font-size: 14px; color: rgba(255,255,255,0.28); }
  .scv-empty span { font-size: 12px; }

  .scv-msg { display: flex; flex-direction: column; max-width: 78%; gap: 3px; margin-bottom: 4px; }
  .scv-msg--own { align-self: flex-end; align-items: flex-end; }
  .scv-msg--their { align-self: flex-start; align-items: flex-start; }

  .scv-bubble { padding: 9px 13px; border-radius: 16px; font-size: 14px; line-height: 1.55; }
  .scv-bubble--own { display: flex; align-items: center; gap: 7px; background: rgba(129,140,248,0.13); border: 1px solid rgba(129,140,248,0.18); border-bottom-right-radius: 4px; color: rgba(129,140,248,0.65); font-size: 12px; font-weight: 500; }
  .scv-bubble--their { background: rgba(255,255,255,0.07); border: 1px solid rgba(255,255,255,0.06); border-bottom-left-radius: 4px; color: #e2e8f0; word-break: break-word; }
  .scv-bubble--decrypted { position: relative; padding-right: 30px; }
  .scv-bubble--locked { display: flex; align-items: center; gap: 8px; background: rgba(255,255,255,0.04); border: 1px dashed rgba(255,255,255,0.1); border-bottom-left-radius: 4px; color: rgba(255,255,255,0.3); font-size: 13px; cursor: pointer; min-height: 44px; position: relative; touch-action: manipulation; transition: background 0.15s, border-color 0.15s; }
  .scv-bubble--locked:hover { background: rgba(255,255,255,0.07); border-color: rgba(255,255,255,0.18); }
  .scv-bubble--locked-active { border-color: rgba(129,140,248,0.35); background: rgba(129,140,248,0.06); }

  .scv-relock-btn { position: absolute; top: 4px; right: 4px; width: 22px; height: 22px; padding: 11px; margin: -11px; display: flex; align-items: center; justify-content: center; background: none; border: none; cursor: pointer; color: rgba(255,255,255,0.18); border-radius: 8px; touch-action: manipulation; box-sizing: content-box; }
  .scv-relock-btn:hover { color: rgba(129,140,248,0.7); }

  .scv-body { margin: 0; }
  .scv-ago { margin-left: auto; font-size: 10px; color: rgba(255,255,255,0.2); white-space: nowrap; }
  .scv-meta { display: flex; align-items: center; gap: 4px; padding: 0 2px; }
  .scv-time { font-size: 10px; color: rgba(255,255,255,0.22); padding: 0 2px; }
  .scv-countdown { font-size: 10px; color: rgba(129,140,248,0.5); font-variant-numeric: tabular-nums; padding: 0 2px; }

  /* Inline decrypt */
  .scv-inline-decrypt { display: flex; align-items: center; gap: 8px; flex-wrap: nowrap; margin-top: 4px; padding: 8px 10px; background: rgba(129,140,248,0.06); border: 1px solid rgba(129,140,248,0.14); border-radius: 12px; width: 100%; max-width: 100%; box-sizing: border-box; }
  .scv-inline-pin { flex: 1; min-width: 0; padding: 10px; border-radius: 8px; border: 1px solid rgba(255,255,255,0.1); background: rgba(0,0,0,0.3); color: #e2e8f0; font-size: 16px; letter-spacing: 0.25em; text-align: center; outline: none; -webkit-appearance: none; min-height: 44px; touch-action: manipulation; }
  .scv-inline-pin:focus { border-color: rgba(129,140,248,0.5); }
  .scv-inline-btn { padding: 10px 16px; border-radius: 8px; border: none; background: rgba(129,140,248,0.75); color: #fff; font-size: 13px; font-weight: 600; cursor: pointer; min-height: 44px; flex-shrink: 0; white-space: nowrap; touch-action: manipulation; }
  .scv-inline-btn:disabled { opacity: 0.3; cursor: not-allowed; }
  .scv-inline-err { font-size: 11px; color: #f87171; width: 100%; }

  .scv-sent-notice { display: flex; align-items: center; gap: 6px; font-size: 11px; color: #4ade80; align-self: center; margin-top: 4px; }

  /* Compose */
  .scv-compose { padding: 10px 14px 14px; border-top: 1px solid rgba(255,255,255,0.06); display: flex; flex-direction: column; gap: 7px; flex-shrink: 0; background: rgba(0,0,0,0.12); }
  .scv-compose-inner { display: flex; align-items: flex-end; gap: 8px; }
  .scv-compose-text { flex: 1; resize: none; padding: 10px 12px; border-radius: 14px; border: 1px solid rgba(255,255,255,0.09); background: rgba(255,255,255,0.04); color: #e2e8f0; font-size: 14px; line-height: 1.5; outline: none; font-family: system-ui, sans-serif; -webkit-appearance: none; max-height: 120px; min-height: 44px; box-sizing: border-box; width: 100%; transition: border-color 0.15s; }
  .scv-compose-text:focus { border-color: rgba(255,255,255,0.18); }
  .scv-compose-text::placeholder { color: rgba(255,255,255,0.2); }
  @media (max-width: 767px) { .scv-compose-text { font-size: 16px; } }
  .scv-send-btn { width: 44px; height: 44px; border-radius: 13px; border: none; background: rgba(129,140,248,0.8); color: #fff; cursor: pointer; display: flex; align-items: center; justify-content: center; flex-shrink: 0; transition: background 0.15s; touch-action: manipulation; }
  .scv-send-btn:hover:not(:disabled) { background: #818cf8; }
  .scv-send-btn:disabled { opacity: 0.28; cursor: not-allowed; }
  .scv-send-ring { width: 16px; height: 16px; border: 2px solid rgba(255,255,255,0.25); border-top-color: #fff; border-radius: 50%; animation: scv-spin 0.8s linear infinite; }
  .scv-compose-hint { display: flex; align-items: center; gap: 5px; margin: 0; font-size: 10px; color: rgba(255,255,255,0.18); }
  .scv-reply-err { margin: 0; font-size: 12px; color: #f87171; }

  /* Utils */
  .scv-sr { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
  @keyframes scv-spin { to { transform: rotate(360deg); } }
</style>
