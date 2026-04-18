<script>
  import { get } from 'svelte/store';
  import { onMount, onDestroy, tick } from 'svelte';
  import { fade } from 'svelte/transition';
  import { socket, markSecretMsgSeen, createSecretChatInvite } from '../lib/socket.js';
  import { authUser } from '../lib/stores/auth.js';
  import { secretChats, lockSecretChat, storeDecrypted } from '../lib/stores/secretChat.js';
  import { encryptMessage, decryptMessage } from '../lib/crypto.js';
  import { toasts } from '../lib/stores/toast.js';
  import { otherUsers } from '../lib/stores/map.js';

  export let peerId;
  export let peerName = 'Contact';
  export let onClose = () => {};

  let composeText = '';
  let composePin = '';
  let sending = false;

  let decryptPin = '';
  let decryptError = '';
  let unlocking = false;

  let copyDone = false;
  let messagesEl;

  $: chat = $secretChats.get(peerId) ?? { messages: [], locked: true, decryptedMessages: new Map() };
  $: myId = get(authUser)?.userId;

  // Sorted oldest → newest for display
  $: sortedMsgs = [...chat.messages].sort(
    (a, b) => new Date(a.createdAt) - new Date(b.createdAt)
  );

  // Received messages not yet decrypted
  $: lockedReceived = sortedMsgs.filter(
    m => m.senderId !== myId && !chat.decryptedMessages.has(m.id)
  );
  $: hasLockedReceived = lockedReceived.length > 0;

  // Show PIN gate: first time (locked) AND there are received msgs to decrypt
  $: showGate = chat.locked && hasLockedReceived;

  // Show decrypt banner: already "open" but new locked msgs arrived
  $: showDecryptBanner = !chat.locked && hasLockedReceived;

  // Peer last seen from otherUsers store
  $: peerUser = $otherUsers ? [...$otherUsers.values()].find(u => u.userId === peerId) : null;
  $: peerOnline = !!peerUser;
  $: peerLastSeen = peerUser?.lastSeen ?? null;

  onMount(() => {
    socket.emit('getSecretMsgs', { peerId, limit: 20 });
    // If no received messages exist, auto-open compose without PIN gate
    tick().then(scrollToBottom);
  });

  onDestroy(() => lockSecretChat(peerId));

  function digitsOnly(e, which) {
    const v = e.target.value.replace(/\D/g, '');
    if (which === 'decrypt') decryptPin = v;
    else composePin = v;
  }

  async function unlock() {
    if (unlocking || decryptPin.length < 4) return;
    decryptError = '';
    unlocking = true;

    const toDecrypt = lockedReceived;
    if (toDecrypt.length === 0) {
      // No received messages — just open
      secretChats.update(m => {
        const copy = new Map(m);
        const ch = copy.get(peerId) ?? { messages: [], locked: false, decryptedMessages: new Map() };
        copy.set(peerId, { ...ch, locked: false });
        return copy;
      });
      decryptPin = '';
      unlocking = false;
      return;
    }

    // Validate PIN against first received message
    try {
      await decryptMessage(toDecrypt[0].ciphertext, toDecrypt[0].iv, toDecrypt[0].salt, decryptPin);
    } catch {
      decryptError = 'Wrong PIN';
      unlocking = false;
      return;
    }

    // Decrypt all locked received messages
    for (const msg of toDecrypt) {
      try {
        const plain = await decryptMessage(msg.ciphertext, msg.iv, msg.salt, decryptPin);
        storeDecrypted(peerId, msg.id, plain);
        if (!msg.seenAt) markSecretMsgSeen(msg.id);
      } catch {
        storeDecrypted(peerId, msg.id, null); // null = wrong PIN for this one
      }
    }

    decryptPin = '';
    unlocking = false;
    await tick();
    scrollToBottom();
  }

  async function send() {
    const text = composeText.trim();
    if (!text || composePin.length < 4 || sending) return;
    sending = true;
    try {
      const { ciphertext, iv, salt } = await encryptMessage(text, composePin);
      socket.emit('sendSecretMsg', { receiverId: peerId, ciphertext, iv, salt });
      composeText = '';
      // Keep composePin so user can send multiple msgs with same PIN
    } catch {
      toasts.error('Failed to send');
    } finally {
      sending = false;
    }
    await tick();
    scrollToBottom();
  }

  function handleKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); }
  }

  function scrollToBottom() {
    if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
  }

  async function shareLink() {
    try {
      const token = await createSecretChatInvite(peerId);
      const url = `${window.location.origin}/#/m/${token}`;
      await navigator.clipboard.writeText(url);
      copyDone = true;
      setTimeout(() => { copyDone = false; }, 2500);
    } catch {
      toasts.error('Could not copy link');
    }
  }

  function formatTime(ts) {
    if (!ts) return '';
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function formatLastSeen(ts) {
    if (!ts) return '';
    const diff = Date.now() - (typeof ts === 'number' ? ts : new Date(ts).getTime());
    if (diff < 60_000) return 'just now';
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
    return new Date(ts).toLocaleDateString([], { month: 'short', day: 'numeric' });
  }

  function getMsgDisplay(msg) {
    if (msg.senderId === myId) {
      return { isOwn: true, body: null, locked: true };
    }
    const plain = chat.decryptedMessages.get(msg.id);
    return {
      isOwn: false,
      body: plain ?? null,
      locked: plain === undefined || plain === null,
    };
  }
</script>

<div class="scp" transition:fade={{ duration: 120 }}>
  <!-- Header -->
  <div class="scp-header">
    <div class="scp-header-lock">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <rect x="3" y="11" width="18" height="11" rx="2"/>
        <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
    </div>
    <div class="scp-header-info">
      <span class="scp-header-name">{peerName}</span>
      <span class="scp-header-sub">
        {#if peerOnline}
          <span class="scp-dot scp-dot--online"></span>Online
        {:else if peerLastSeen}
          <span class="scp-dot"></span>Last seen {formatLastSeen(peerLastSeen)}
        {:else}
          End-to-end encrypted
        {/if}
      </span>
    </div>
    <button class="scp-icon-btn" on:click={shareLink} aria-label="Copy invite link" title="Share invite link">
      {#if copyDone}
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#4ade80" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
      {:else}
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
          <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
        </svg>
      {/if}
    </button>
    <button class="scp-icon-btn" on:click={onClose} aria-label="Close chat">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <line x1="18" y1="6" x2="6" y2="18"/>
        <line x1="6" y1="6" x2="18" y2="18"/>
      </svg>
    </button>
  </div>

  {#if showGate}
    <!-- Full PIN gate for first unlock -->
    <div class="scp-gate">
      <div class="scp-gate-icon">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <rect x="3" y="11" width="18" height="11" rx="2"/>
          <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
      </div>
      <p class="scp-gate-title">Enter PIN to read</p>
      <p class="scp-gate-sub">Ask {peerName} for the PIN they used to encrypt the message</p>
      <label class="scp-sr" for="scp-decrypt-pin">Decryption PIN</label>
      <input
        id="scp-decrypt-pin"
        class="scp-pin-input"
        type="tel"
        inputmode="numeric"
        pattern="[0-9]*"
        maxlength="8"
        placeholder="Enter PIN"
        bind:value={decryptPin}
        on:input={(e) => digitsOnly(e, 'decrypt')}
        on:keydown={(e) => e.key === 'Enter' && unlock()}
        disabled={unlocking}
        autocomplete="off"
        autofocus
      />
      {#if decryptError}
        <p class="scp-gate-error" role="alert">{decryptError}</p>
      {/if}
      <button
        class="scp-primary-btn"
        on:click={unlock}
        disabled={unlocking || decryptPin.length < 4}
      >
        {unlocking ? 'Unlocking…' : 'Unlock'}
      </button>
      <button class="scp-ghost-btn" on:click={() => {
        secretChats.update(m => {
          const copy = new Map(m);
          const ch = copy.get(peerId) ?? { messages: [], locked: false, decryptedMessages: new Map() };
          copy.set(peerId, { ...ch, locked: false });
          return copy;
        });
      }}>
        Skip — just compose
      </button>
    </div>

  {:else}
    <!-- Message list -->
    <div class="scp-msgs" bind:this={messagesEl}>
      {#if sortedMsgs.length === 0}
        <div class="scp-empty">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
          <p>No messages yet</p>
          <span>Send the first secret message below</span>
        </div>
      {/if}

      {#each sortedMsgs as msg (msg.id)}
        {@const d = getMsgDisplay(msg)}
        <div class="scp-msg" class:scp-msg--own={d.isOwn} class:scp-msg--their={!d.isOwn}>
          {#if d.isOwn}
            <!-- Own message: never show plaintext or ciphertext — intentionally opaque -->
            <div class="scp-bubble scp-bubble--own">
              <div class="scp-encrypted-badge">
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <rect x="3" y="11" width="18" height="11" rx="2"/>
                  <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                </svg>
                Encrypted
              </div>
            </div>
            <div class="scp-msg-meta">
              <span class="scp-time">{formatTime(msg.createdAt)}</span>
              <span class="scp-tick" class:scp-tick--seen={msg.seenAt} title={msg.seenAt ? `Read ${formatTime(msg.seenAt)}` : 'Sent'}>
                {#if msg.seenAt}
                  <svg width="14" height="8" viewBox="0 0 18 10" fill="none" stroke="#53bdeb" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-label="Read"><polyline points="1 5 5 9 13 1"/><polyline points="7 9 15 1"/></svg>
                {:else}
                  <svg width="10" height="8" viewBox="0 0 12 10" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-label="Sent"><polyline points="1 5 5 9 11 1"/></svg>
                {/if}
              </span>
            </div>

          {:else if d.locked}
            <!-- Received, not yet decrypted -->
            <div class="scp-bubble scp-bubble--locked">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <rect x="3" y="11" width="18" height="11" rx="2"/>
                <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
              <span>Locked</span>
            </div>
            <span class="scp-time scp-time--their">{formatTime(msg.createdAt)}</span>

          {:else}
            <!-- Received + decrypted -->
            <div class="scp-bubble scp-bubble--their">
              <p class="scp-body">{d.body}</p>
            </div>
            <span class="scp-time scp-time--their">{formatTime(msg.createdAt)}</span>
          {/if}
        </div>
      {/each}
    </div>

    <!-- Inline decrypt banner (new locked messages arrived after already open) -->
    {#if showDecryptBanner}
      <div class="scp-decrypt-banner">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <rect x="3" y="11" width="18" height="11" rx="2"/>
          <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
        <span>{lockedReceived.length} locked message{lockedReceived.length > 1 ? 's' : ''}</span>
        <label class="scp-sr" for="scp-banner-pin">Decryption PIN</label>
        <input
          id="scp-banner-pin"
          class="scp-banner-pin"
          type="tel"
          inputmode="numeric"
          pattern="[0-9]*"
          maxlength="8"
          placeholder="PIN"
          bind:value={decryptPin}
          on:input={(e) => digitsOnly(e, 'decrypt')}
          on:keydown={(e) => e.key === 'Enter' && unlock()}
          disabled={unlocking}
          autocomplete="off"
        />
        <button class="scp-banner-btn" on:click={unlock} disabled={unlocking || decryptPin.length < 4}>
          {unlocking ? '…' : 'Decrypt'}
        </button>
        {#if decryptError}
          <span class="scp-banner-err" role="alert">{decryptError}</span>
        {/if}
      </div>
    {/if}

    <!-- Compose area -->
    <div class="scp-compose">
      <label class="scp-sr" for="scp-compose-text">Message</label>
      <textarea
        id="scp-compose-text"
        class="scp-compose-text"
        rows="2"
        maxlength="2000"
        placeholder="Type a secret message…"
        bind:value={composeText}
        on:keydown={handleKeydown}
        disabled={sending}
      ></textarea>
      <div class="scp-compose-row">
        <div class="scp-compose-pin-wrap">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <rect x="3" y="11" width="18" height="11" rx="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
          <label class="scp-sr" for="scp-compose-pin">PIN for this message</label>
          <input
            id="scp-compose-pin"
            class="scp-compose-pin"
            type="tel"
            inputmode="numeric"
            pattern="[0-9]*"
            maxlength="8"
            placeholder="Set PIN (min 4 digits)"
            bind:value={composePin}
            on:input={(e) => digitsOnly(e, 'compose')}
            disabled={sending}
            autocomplete="off"
          />
        </div>
        <button
          class="scp-send-btn"
          on:click={send}
          disabled={sending || !composeText.trim() || composePin.length < 4}
          aria-label="Send secret message"
        >
          {#if sending}
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" class="scp-spin"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
          {:else}
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
          {/if}
        </button>
      </div>
      <p class="scp-compose-hint">Receiver needs your PIN to read — don't share it here</p>
    </div>
  {/if}
</div>

<style>
  /* ---------- Shell ---------- */
  .scp {
    display: flex;
    flex-direction: column;
    background: #111118;
    border: 1px solid rgba(255,255,255,0.09);
    border-radius: 18px;
    overflow: hidden;
    width: 100%;
    max-width: 420px;
    /* On mobile: take most of the screen height */
    height: min(85dvh, 620px);
    box-shadow: 0 16px 48px rgba(0,0,0,0.55), 0 0 0 1px rgba(129,140,248,0.08);
  }

  /* Full-width bottom sheet on narrow screens */
  @media (max-width: 480px) {
    .scp {
      max-width: 100%;
      border-radius: 20px 20px 0 0;
      height: 88dvh;
    }
  }

  /* ---------- Header ---------- */
  .scp-header {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 14px 14px 14px 16px;
    background: rgba(255,255,255,0.03);
    border-bottom: 1px solid rgba(255,255,255,0.07);
    flex-shrink: 0;
  }

  .scp-header-lock {
    color: rgba(129,140,248,0.7);
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }

  .scp-header-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .scp-header-name {
    font-size: 14px;
    font-weight: 600;
    color: #e2e8f0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: system-ui, sans-serif;
  }

  .scp-header-sub {
    font-size: 11px;
    color: rgba(255,255,255,0.38);
    display: flex;
    align-items: center;
    gap: 5px;
    font-family: system-ui, sans-serif;
  }

  .scp-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: rgba(255,255,255,0.2);
    flex-shrink: 0;
  }
  .scp-dot--online { background: #4ade80; box-shadow: 0 0 5px rgba(74,222,128,0.6); }

  .scp-icon-btn {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: none;
    color: rgba(255,255,255,0.4);
    cursor: pointer;
    border-radius: 8px;
    transition: background 0.15s, color 0.15s;
    flex-shrink: 0;
  }
  .scp-icon-btn:hover { background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.75); }

  /* ---------- Gate ---------- */
  .scp-gate {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 14px;
    padding: 32px 24px;
    text-align: center;
  }

  .scp-gate-icon {
    width: 60px;
    height: 60px;
    border-radius: 50%;
    background: rgba(129,140,248,0.1);
    border: 1px solid rgba(129,140,248,0.2);
    display: flex;
    align-items: center;
    justify-content: center;
    color: rgba(129,140,248,0.8);
    margin-bottom: 4px;
  }

  .scp-gate-title {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    color: #e2e8f0;
    font-family: system-ui, sans-serif;
  }

  .scp-gate-sub {
    margin: 0;
    font-size: 12px;
    color: rgba(255,255,255,0.38);
    max-width: 240px;
    line-height: 1.6;
    font-family: system-ui, sans-serif;
  }

  .scp-gate-error {
    margin: 0;
    font-size: 12px;
    color: #f87171;
    font-family: system-ui, sans-serif;
  }

  .scp-pin-input {
    width: 100%;
    max-width: 220px;
    padding: 13px 16px;
    border-radius: 12px;
    border: 1px solid rgba(255,255,255,0.12);
    background: rgba(255,255,255,0.05);
    color: #e2e8f0;
    font-size: 22px;
    letter-spacing: 0.3em;
    text-align: center;
    outline: none;
    font-variant-numeric: tabular-nums;
    font-family: system-ui, monospace, sans-serif;
    transition: border-color 0.15s;
    box-sizing: border-box;
    -webkit-appearance: none;
  }
  .scp-pin-input:focus { border-color: rgba(129,140,248,0.6); box-shadow: 0 0 0 3px rgba(129,140,248,0.12); }

  .scp-primary-btn {
    width: 100%;
    max-width: 220px;
    padding: 13px;
    border-radius: 12px;
    border: none;
    background: rgba(129,140,248,0.85);
    color: #fff;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s;
    font-family: system-ui, sans-serif;
    min-height: 48px;
  }
  .scp-primary-btn:hover:not(:disabled) { background: rgba(129,140,248,1); }
  .scp-primary-btn:disabled { opacity: 0.35; cursor: not-allowed; }

  .scp-ghost-btn {
    background: none;
    border: none;
    color: rgba(255,255,255,0.35);
    font-size: 12px;
    cursor: pointer;
    padding: 6px;
    font-family: system-ui, sans-serif;
    transition: color 0.15s;
  }
  .scp-ghost-btn:hover { color: rgba(255,255,255,0.6); }

  /* ---------- Messages ---------- */
  .scp-msgs {
    flex: 1;
    overflow-y: auto;
    padding: 16px 14px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    overscroll-behavior: contain;
  }
  .scp-msgs::-webkit-scrollbar { width: 4px; }
  .scp-msgs::-webkit-scrollbar-track { background: transparent; }
  .scp-msgs::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.1); border-radius: 2px; }

  .scp-empty {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    color: rgba(255,255,255,0.2);
    text-align: center;
    padding: 40px 0;
  }
  .scp-empty p { margin: 0; font-size: 14px; color: rgba(255,255,255,0.3); font-family: system-ui, sans-serif; }
  .scp-empty span { font-size: 12px; color: rgba(255,255,255,0.2); font-family: system-ui, sans-serif; }

  .scp-msg {
    display: flex;
    flex-direction: column;
    max-width: 78%;
    gap: 3px;
  }
  .scp-msg--own { align-self: flex-end; align-items: flex-end; }
  .scp-msg--their { align-self: flex-start; align-items: flex-start; }

  .scp-bubble {
    padding: 9px 13px;
    border-radius: 16px;
    font-size: 14px;
    line-height: 1.55;
    font-family: system-ui, sans-serif;
  }

  .scp-bubble--own {
    background: rgba(129,140,248,0.15);
    border: 1px solid rgba(129,140,248,0.2);
    border-bottom-right-radius: 4px;
    display: flex;
    align-items: center;
  }

  .scp-encrypted-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    color: rgba(129,140,248,0.65);
    font-size: 12px;
    font-weight: 500;
    letter-spacing: 0.02em;
    font-family: system-ui, sans-serif;
  }

  .scp-bubble--their {
    background: rgba(255,255,255,0.07);
    border: 1px solid rgba(255,255,255,0.06);
    border-bottom-left-radius: 4px;
    color: #e2e8f0;
    word-break: break-word;
  }

  .scp-bubble--locked {
    background: rgba(255,255,255,0.04);
    border: 1px dashed rgba(255,255,255,0.12);
    border-bottom-left-radius: 4px;
    display: flex;
    align-items: center;
    gap: 7px;
    color: rgba(255,255,255,0.25);
    font-size: 12px;
    font-style: italic;
    font-family: system-ui, sans-serif;
  }

  .scp-body { margin: 0; }

  .scp-msg-meta {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .scp-time {
    font-size: 10px;
    color: rgba(255,255,255,0.25);
    font-family: system-ui, sans-serif;
  }
  .scp-time--their { margin-left: 2px; }

  .scp-tick { display: flex; align-items: center; color: rgba(255,255,255,0.3); }
  .scp-tick--seen { color: #53bdeb; }

  /* ---------- Decrypt banner ---------- */
  .scp-decrypt-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 14px;
    background: rgba(129,140,248,0.08);
    border-top: 1px solid rgba(129,140,248,0.12);
    color: rgba(129,140,248,0.7);
    flex-wrap: wrap;
    flex-shrink: 0;
  }
  .scp-decrypt-banner span:first-of-type {
    font-size: 12px;
    font-family: system-ui, sans-serif;
    flex: 1;
    min-width: 120px;
  }

  .scp-banner-pin {
    width: 110px;
    padding: 7px 10px;
    border-radius: 8px;
    border: 1px solid rgba(129,140,248,0.25);
    background: rgba(0,0,0,0.3);
    color: #e2e8f0;
    font-size: 15px;
    letter-spacing: 0.2em;
    text-align: center;
    outline: none;
    font-family: system-ui, monospace, sans-serif;
    -webkit-appearance: none;
  }
  .scp-banner-pin:focus { border-color: rgba(129,140,248,0.6); }

  .scp-banner-btn {
    padding: 7px 14px;
    border-radius: 8px;
    border: none;
    background: rgba(129,140,248,0.7);
    color: #fff;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    font-family: system-ui, sans-serif;
    min-height: 36px;
    transition: background 0.15s;
  }
  .scp-banner-btn:hover:not(:disabled) { background: rgba(129,140,248,0.9); }
  .scp-banner-btn:disabled { opacity: 0.35; cursor: not-allowed; }

  .scp-banner-err {
    font-size: 11px;
    color: #f87171;
    font-family: system-ui, sans-serif;
    width: 100%;
  }

  /* ---------- Compose ---------- */
  .scp-compose {
    padding: 10px 14px 14px;
    border-top: 1px solid rgba(255,255,255,0.07);
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex-shrink: 0;
    background: rgba(0,0,0,0.15);
  }

  .scp-compose-text {
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
  .scp-compose-text:focus { border-color: rgba(255,255,255,0.22); }
  .scp-compose-text::placeholder { color: rgba(255,255,255,0.25); }

  .scp-compose-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .scp-compose-pin-wrap {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 9px 12px;
    border-radius: 12px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(255,255,255,0.04);
    color: rgba(255,255,255,0.35);
    transition: border-color 0.15s;
    min-width: 0;
  }
  .scp-compose-pin-wrap:focus-within { border-color: rgba(255,255,255,0.2); }

  .scp-compose-pin {
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
  .scp-compose-pin::placeholder { color: rgba(255,255,255,0.22); letter-spacing: normal; font-size: 13px; }

  .scp-send-btn {
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
  .scp-send-btn:hover:not(:disabled) { background: rgba(129,140,248,1); }
  .scp-send-btn:disabled { opacity: 0.3; cursor: not-allowed; }

  .scp-compose-hint {
    margin: 0;
    font-size: 10px;
    color: rgba(255,255,255,0.2);
    font-family: system-ui, sans-serif;
  }

  /* ---------- Utilities ---------- */
  .scp-sr {
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

  @keyframes scp-spin { to { transform: rotate(360deg); } }
  .scp-spin { animation: scp-spin 1s linear infinite; }
</style>
