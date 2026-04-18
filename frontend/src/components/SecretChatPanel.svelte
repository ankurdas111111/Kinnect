<script>
  import { get } from 'svelte/store';
  import { onMount, onDestroy } from 'svelte';
  import { fade } from 'svelte/transition';
  import { socket, markSecretMsgSeen, createSecretChatInvite } from '../lib/socket.js';
  import { authUser } from '../lib/stores/auth.js';
  import { secretChats, lockSecretChat, addSecretMessage, storeDecrypted } from '../lib/stores/secretChat.js';
  import { encryptMessage, decryptMessage } from '../lib/crypto.js';
  import { toasts } from '../lib/stores/toast.js';

  /** Contact's userId */
  export let peerId;
  /** Contact's display name */
  export let peerName = 'Contact';
  /** Callback when the panel should close */
  export let onClose = () => {};

  let pin = '';
  let composePin = '';
  let copyDone = false;
  let composeText = '';
  let pinError = '';
  let sending = false;
  let unlocking = false;

  $: chat = $secretChats.get(peerId) ?? { messages: [], locked: true, decryptedMessages: new Map() };
  $: isLocked = chat.locked;
  $: myId = get(authUser)?.userId;

  onMount(() => {
    socket.emit('getSecretMsgs', { peerId, limit: 20 });
  });

  onDestroy(() => {
    // Wipe all decrypted plaintext from memory when panel closes
    lockSecretChat(peerId);
  });

  /** Strip any non-digit character as the user types */
  function digitsOnly(e, target) {
    const cleaned = e.target.value.replace(/\D/g, '');
    if (target === 'unlock') pin = cleaned;
    else composePin = cleaned;
  }

  async function unlock() {
    if (unlocking) return;
    pinError = '';
    const msgs = chat.messages;

    if (msgs.length === 0) {
      // No messages yet — just open the compose area
      secretChats.update((m) => {
        const copy = new Map(m);
        const ch = copy.get(peerId) ?? { messages: [], locked: false, decryptedMessages: new Map() };
        copy.set(peerId, { ...ch, locked: false });
        return copy;
      });
      pin = '';
      return;
    }

    unlocking = true;
    // Validate PIN against the first message — if that decrypts, the PIN is correct
    const testMsg = msgs[0];
    try {
      await decryptMessage(testMsg.ciphertext, testMsg.iv, testMsg.salt, pin);
    } catch {
      pinError = 'Incorrect PIN';
      unlocking = false;
      return;
    }

    // Decrypt all messages with this PIN; mark received messages as seen
    for (const msg of msgs) {
      try {
        const plain = await decryptMessage(msg.ciphertext, msg.iv, msg.salt, pin);
        storeDecrypted(peerId, msg.id, plain);
        // Notify sender that we decrypted their message (seen = actually read)
        if (msg.senderId !== myId && !msg.seenAt) {
          markSecretMsgSeen(msg.id);
        }
      } catch {
        storeDecrypted(peerId, msg.id, '[encrypted with different PIN]');
      }
    }

    pin = '';
    unlocking = false;
  }

  async function send() {
    const text = composeText.trim();
    if (!text || composePin.length < 4 || !/^\d+$/.test(composePin) || sending) return;
    sending = true;
    try {
      const { ciphertext, iv, salt } = await encryptMessage(text, composePin);
      socket.emit('sendSecretMsg', { receiverId: peerId, ciphertext, iv, salt });
      composeText = '';
      composePin = '';
    } catch (err) {
      toasts.error('Failed to encrypt message');
    } finally {
      sending = false;
    }
  }

  function handleComposeKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      send();
    }
  }

  function getDisplay(msg) {
    if (msg.senderId === myId) {
      // Sender always sees ciphertext — intentional, cannot read without the PIN they set
      return {
        label: 'You (encrypted)',
        body: msg.ciphertext ? msg.ciphertext.slice(0, 48) + '…' : '…',
        isOwn: true,
        locked: true,
      };
    }
    const plain = chat.decryptedMessages.get(msg.id);
    return {
      label: peerName,
      body: plain ?? '[locked — enter PIN to read]',
      isOwn: false,
      locked: !plain,
    };
  }

  async function shareLink() {
    try {
      const token = await createSecretChatInvite(peerId);
      const url = `${window.location.origin}/#/m/${token}`;
      await navigator.clipboard.writeText(url);
      copyDone = true;
      setTimeout(() => { copyDone = false; }, 2000);
    } catch {
      toasts.error('Could not copy link');
    }
  }

  function formatTime(ts) {
    if (!ts) return '';
    const d = new Date(ts);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
</script>

<div class="secret-panel" transition:fade={{ duration: 150 }}>
  <!-- Header -->
  <div class="sp-header">
    <span class="sp-lock-icon">🔒</span>
    <span class="sp-title">Secret chat with {peerName}</span>
    <button class="sp-share-btn" on:click={shareLink} title="Copy invite link" aria-label="Copy invite link">
      {#if copyDone}
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
      {:else}
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
          <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
        </svg>
      {/if}
    </button>
    <button class="sp-close" on:click={onClose} aria-label="Close">✕</button>
  </div>

  {#if isLocked}
    <!-- PIN gate -->
    <div class="sp-pin-gate">
      <p class="sp-pin-hint">Enter the PIN to read messages</p>
      <input
        class="sp-pin-input"
        type="tel"
        inputmode="numeric"
        pattern="[0-9]*"
        maxlength="8"
        placeholder="PIN (numbers only)"
        bind:value={pin}
        on:input={(e) => digitsOnly(e, 'unlock')}
        on:keydown={(e) => e.key === 'Enter' && unlock()}
        disabled={unlocking}
        autocomplete="off"
      />
      {#if pinError}
        <p class="sp-pin-error">{pinError}</p>
      {/if}
      <button
        class="sp-btn sp-btn-primary"
        on:click={unlock}
        disabled={unlocking || pin.length < 4}
      >
        {unlocking ? 'Unlocking…' : 'Unlock'}
      </button>
    </div>
  {:else}
    <!-- Message list -->
    <div class="sp-messages">
      {#if chat.messages.length === 0}
        <p class="sp-empty">No messages yet. Send the first one.</p>
      {/if}
      {#each chat.messages as msg (msg.id)}
        {@const d = getDisplay(msg)}
        <div class="sp-msg" class:sp-msg-own={d.isOwn} class:sp-msg-locked={d.locked && !d.isOwn}>
          <span class="sp-msg-label">{d.label}</span>
          <p class="sp-msg-body">{d.body}</p>
          <div class="sp-msg-footer">
            <span class="sp-msg-time">{formatTime(msg.createdAt)}</span>
            {#if d.isOwn}
              <span
                class="sp-seen-tick"
                class:sp-seen-tick--seen={msg.seenAt}
                title={msg.seenAt ? `Seen ${formatTime(msg.seenAt)}` : 'Sent'}
              >{msg.seenAt ? '✓✓' : '✓'}</span>
            {/if}
          </div>
        </div>
      {/each}
    </div>

    <!-- Compose area -->
    <div class="sp-compose">
      <textarea
        class="sp-compose-text"
        rows="2"
        maxlength="2000"
        placeholder="Type secret message…"
        bind:value={composeText}
        on:keydown={handleComposeKeydown}
        disabled={sending}
      ></textarea>
      <input
        class="sp-pin-input sp-compose-pin"
        type="tel"
        inputmode="numeric"
        pattern="[0-9]*"
        maxlength="8"
        placeholder="Set PIN — numbers only"
        bind:value={composePin}
        on:input={(e) => digitsOnly(e, 'compose')}
        disabled={sending}
        autocomplete="off"
      />
      <p class="sp-compose-hint">Receiver must enter this PIN to read the message</p>
      <button
        class="sp-btn sp-btn-primary sp-send-btn"
        on:click={send}
        disabled={sending || !composeText.trim() || composePin.length < 4 || !/^\d+$/.test(composePin)}
      >
        {sending ? 'Sending…' : 'Send Secret'}
      </button>
    </div>
  {/if}
</div>

<style>
  .secret-panel {
    display: flex;
    flex-direction: column;
    background: var(--surface, #1a1a2e);
    border: 1px solid var(--border, rgba(255,255,255,0.12));
    border-radius: 16px;
    overflow: hidden;
    max-width: 420px;
    width: 100%;
    max-height: 80vh;
    box-shadow: 0 8px 32px rgba(0,0,0,0.4);
  }

  .sp-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 14px 16px;
    background: var(--surface-2, rgba(255,255,255,0.05));
    border-bottom: 1px solid var(--border, rgba(255,255,255,0.08));
  }
  .sp-lock-icon { font-size: 16px; }
  .sp-title {
    flex: 1;
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary, #e2e8f0);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sp-share-btn {
    background: none;
    border: none;
    color: var(--text-secondary, #94a3b8);
    cursor: pointer;
    padding: 4px;
    line-height: 1;
    border-radius: 4px;
    display: flex;
    align-items: center;
  }
  .sp-share-btn:hover { color: var(--accent, #818cf8); }

  .sp-close {
    background: none;
    border: none;
    color: var(--text-secondary, #94a3b8);
    cursor: pointer;
    font-size: 16px;
    padding: 4px;
    line-height: 1;
    border-radius: 4px;
  }
  .sp-close:hover { color: var(--text-primary, #e2e8f0); }

  /* PIN gate */
  .sp-pin-gate {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    padding: 32px 24px;
  }
  .sp-pin-hint {
    color: var(--text-secondary, #94a3b8);
    font-size: 13px;
    margin: 0;
  }
  .sp-pin-error {
    color: var(--danger, #f87171);
    font-size: 12px;
    margin: 0;
  }

  /* Shared input style */
  .sp-pin-input {
    width: 100%;
    max-width: 200px;
    padding: 10px 14px;
    border-radius: 10px;
    border: 1px solid var(--border, rgba(255,255,255,0.15));
    background: var(--input-bg, rgba(255,255,255,0.06));
    color: var(--text-primary, #e2e8f0);
    font-size: 18px;
    letter-spacing: 0.2em;
    text-align: center;
    outline: none;
    font-variant-numeric: tabular-nums;
  }
  .sp-pin-input:focus {
    border-color: var(--accent, #818cf8);
  }

  /* Buttons */
  .sp-btn {
    padding: 10px 24px;
    border-radius: 10px;
    border: none;
    cursor: pointer;
    font-size: 14px;
    font-weight: 600;
    transition: opacity 0.15s;
  }
  .sp-btn:disabled { opacity: 0.4; cursor: not-allowed; }
  .sp-btn-primary {
    background: var(--accent, #818cf8);
    color: #fff;
  }
  .sp-btn-primary:not(:disabled):hover { opacity: 0.85; }

  /* Messages */
  .sp-messages {
    flex: 1;
    overflow-y: auto;
    padding: 12px 16px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .sp-empty {
    text-align: center;
    color: var(--text-secondary, #94a3b8);
    font-size: 13px;
    margin: auto;
  }
  .sp-msg {
    display: flex;
    flex-direction: column;
    max-width: 80%;
    gap: 2px;
  }
  .sp-msg-own { align-self: flex-end; align-items: flex-end; }
  .sp-msg:not(.sp-msg-own) { align-self: flex-start; align-items: flex-start; }
  .sp-msg-label {
    font-size: 10px;
    color: var(--text-secondary, #94a3b8);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .sp-msg-body {
    margin: 0;
    padding: 8px 12px;
    border-radius: 12px;
    font-size: 13px;
    line-height: 1.5;
    word-break: break-word;
    background: var(--surface-2, rgba(255,255,255,0.07));
    color: var(--text-primary, #e2e8f0);
  }
  .sp-msg-own .sp-msg-body {
    background: var(--accent-dim, rgba(129,140,248,0.2));
    font-family: monospace;
    font-size: 11px;
    color: var(--text-secondary, #94a3b8);
  }
  .sp-msg-locked .sp-msg-body {
    opacity: 0.5;
    font-style: italic;
  }
  .sp-msg-footer {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 2px;
  }

  .sp-msg-time {
    font-size: 10px;
    color: var(--text-secondary, #94a3b8);
  }

  .sp-seen-tick {
    font-size: 11px;
    color: #8696a0; /* grey = sent, delivered */
    line-height: 1;
  }
  .sp-seen-tick--seen {
    color: #53bdeb; /* blue = seen (receiver decrypted) — WhatsApp convention */
  }

  /* Compose */
  .sp-compose {
    padding: 12px 16px;
    border-top: 1px solid var(--border, rgba(255,255,255,0.08));
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .sp-compose-text {
    width: 100%;
    resize: none;
    padding: 10px 12px;
    border-radius: 10px;
    border: 1px solid var(--border, rgba(255,255,255,0.15));
    background: var(--input-bg, rgba(255,255,255,0.06));
    color: var(--text-primary, #e2e8f0);
    font-size: 13px;
    line-height: 1.5;
    outline: none;
    box-sizing: border-box;
  }
  .sp-compose-text:focus { border-color: var(--accent, #818cf8); }
  .sp-compose-pin {
    max-width: 100%;
    text-align: left;
    letter-spacing: normal;
    font-size: 14px;
  }
  .sp-compose-hint {
    font-size: 11px;
    color: var(--text-secondary, #94a3b8);
    margin: 0;
  }
  .sp-send-btn { align-self: flex-end; }
</style>
