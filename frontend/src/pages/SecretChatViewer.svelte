<svelte:head>
  <title>Messages</title>
</svelte:head>

<script>
  import { onMount } from 'svelte';
  import { decryptMessage } from '../lib/crypto.js';

  export let params = {};
  $: token = params.token || '';

  // state: 'loading' | 'passcode' | 'decrypting' | 'messages' | 'error'
  let state = 'loading';
  let errorMsg = '';
  let pin = '';
  let pinError = '';
  let rawMessages = []; // { ciphertext, iv, salt, fromOwner, createdAt }
  let decrypted = [];   // { body, fromOwner, createdAt }
  let decrypting = false;

  onMount(async () => {
    if (!token) {
      state = 'error';
      errorMsg = 'Invalid link.';
      return;
    }
    try {
      const res = await fetch(`/api/m/${token}`);
      const data = await res.json();
      if (!data.ok) {
        state = 'error';
        if (data.expired) {
          errorMsg = 'This link has expired. Ask for a new one.';
        } else if (data.error === 'temporarily_unavailable') {
          errorMsg = 'Something went wrong. Please try again later.';
        } else {
          errorMsg = 'Invalid link.';
        }
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
    if (decrypting || pin.length < 4) return;
    pinError = '';

    if (rawMessages.length === 0) {
      // No messages yet — nothing to decrypt, show empty state
      decrypted = [];
      state = 'messages';
      return;
    }

    decrypting = true;
    // Validate PIN against first message
    try {
      const m = rawMessages[0];
      await decryptMessage(m.ciphertext, m.iv, m.salt, pin);
    } catch {
      pinError = 'Incorrect passcode';
      decrypting = false;
      return;
    }

    // Decrypt all messages
    const results = [];
    for (const m of rawMessages) {
      try {
        const body = await decryptMessage(m.ciphertext, m.iv, m.salt, pin);
        results.push({ body, fromOwner: m.fromOwner, createdAt: m.createdAt });
      } catch {
        results.push({ body: '[encrypted with different passcode]', fromOwner: m.fromOwner, createdAt: m.createdAt });
      }
    }
    decrypted = results;
    decrypting = false;
    state = 'messages';
  }

  function handleKeydown(e) {
    if (e.key === 'Enter') unlock();
  }

  function digitsOnly(e) {
    pin = e.target.value.replace(/\D/g, '');
  }

  function formatTime(ts) {
    if (!ts) return '';
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
</script>

<div class="scv-shell">
  {#if state === 'loading'}
    <div class="scv-card">
      <div class="scv-spinner"></div>
    </div>

  {:else if state === 'error'}
    <div class="scv-card">
      <div class="scv-icon scv-icon--error">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="12" r="10"/>
          <line x1="15" y1="9" x2="9" y2="15"/>
          <line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
      </div>
      <p class="scv-hint">{errorMsg}</p>
    </div>

  {:else if state === 'passcode'}
    <div class="scv-card">
      <div class="scv-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
          <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
      </div>
      <p class="scv-hint">Enter passcode to continue</p>
      <input
        class="scv-pin-input"
        type="tel"
        inputmode="numeric"
        pattern="[0-9]*"
        maxlength="8"
        placeholder="Passcode"
        bind:value={pin}
        on:input={digitsOnly}
        on:keydown={handleKeydown}
        disabled={decrypting}
        autocomplete="off"
        autofocus
      />
      {#if pinError}
        <p class="scv-pin-error">{pinError}</p>
      {/if}
      <button
        class="scv-btn"
        on:click={unlock}
        disabled={decrypting || pin.length < 4}
      >
        {decrypting ? 'Verifying…' : 'Continue'}
      </button>
    </div>

  {:else if state === 'messages'}
    <div class="scv-view">
      <div class="scv-messages">
        {#if decrypted.length === 0}
          <p class="scv-empty">No messages.</p>
        {/if}
        {#each decrypted as msg}
          <div class="scv-msg" class:scv-msg-theirs={!msg.fromOwner} class:scv-msg-mine={msg.fromOwner}>
            <p class="scv-msg-body">{msg.body}</p>
            <span class="scv-msg-time">{formatTime(msg.createdAt)}</span>
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .scv-shell {
    min-height: 100dvh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #0d0d14;
    padding: 24px;
  }

  .scv-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    width: 100%;
    max-width: 340px;
  }

  .scv-icon {
    color: rgba(255,255,255,0.35);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .scv-icon--error { color: #f87171; }

  .scv-spinner {
    width: 28px;
    height: 28px;
    border: 2px solid rgba(255,255,255,0.1);
    border-top-color: rgba(255,255,255,0.4);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .scv-hint {
    color: rgba(255,255,255,0.45);
    font-size: 13px;
    margin: 0;
    text-align: center;
    font-family: system-ui, sans-serif;
  }

  .scv-pin-input {
    width: 100%;
    padding: 12px 16px;
    border-radius: 10px;
    border: 1px solid rgba(255,255,255,0.12);
    background: rgba(255,255,255,0.05);
    color: #e2e8f0;
    font-size: 20px;
    letter-spacing: 0.25em;
    text-align: center;
    outline: none;
    font-variant-numeric: tabular-nums;
    font-family: system-ui, sans-serif;
    box-sizing: border-box;
  }
  .scv-pin-input:focus {
    border-color: rgba(255,255,255,0.3);
  }

  .scv-pin-error {
    color: #f87171;
    font-size: 12px;
    margin: 0;
    font-family: system-ui, sans-serif;
  }

  .scv-btn {
    width: 100%;
    padding: 12px;
    border-radius: 10px;
    border: none;
    background: rgba(255,255,255,0.1);
    color: rgba(255,255,255,0.8);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
    font-family: system-ui, sans-serif;
  }
  .scv-btn:hover:not(:disabled) { background: rgba(255,255,255,0.16); }
  .scv-btn:disabled { opacity: 0.4; cursor: not-allowed; }

  /* Message view */
  .scv-view {
    width: 100%;
    max-width: 500px;
    height: 100dvh;
    display: flex;
    flex-direction: column;
  }

  .scv-messages {
    flex: 1;
    overflow-y: auto;
    padding: 20px 16px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .scv-empty {
    text-align: center;
    color: rgba(255,255,255,0.3);
    font-size: 13px;
    margin: auto;
    font-family: system-ui, sans-serif;
  }

  .scv-msg {
    display: flex;
    flex-direction: column;
    max-width: 78%;
    gap: 3px;
  }
  .scv-msg-mine { align-self: flex-end; align-items: flex-end; }
  .scv-msg-theirs { align-self: flex-start; align-items: flex-start; }

  .scv-msg-body {
    margin: 0;
    padding: 9px 13px;
    border-radius: 14px;
    font-size: 14px;
    line-height: 1.5;
    word-break: break-word;
    font-family: system-ui, sans-serif;
  }
  .scv-msg-mine .scv-msg-body {
    background: rgba(129,140,248,0.2);
    color: #e2e8f0;
    border-bottom-right-radius: 4px;
  }
  .scv-msg-theirs .scv-msg-body {
    background: rgba(255,255,255,0.08);
    color: #e2e8f0;
    border-bottom-left-radius: 4px;
  }

  .scv-msg-time {
    font-size: 10px;
    color: rgba(255,255,255,0.3);
    font-family: system-ui, sans-serif;
  }
</style>
