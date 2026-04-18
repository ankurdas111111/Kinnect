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

  // Session PIN — set once at gate, used for ALL outgoing messages
  let sessionPin = '';
  let gatePin = '';
  let gateError = '';
  let gateUnlocking = false;
  let gateOpen = false; // true once PIN accepted

  // Per-message inline decryption
  let activeDecryptId = null;       // msgId currently showing inline PIN input
  let inlinePins = {};              // msgId → typed value
  let inlineErrors = {};            // msgId → error string
  let inlineUnlocking = {};         // msgId → boolean

  let composeText = '';
  let sending = false;
  let copyDone = false;
  let messagesEl;

  $: chat = $secretChats.get(peerId) ?? { messages: [], locked: true, decryptedMessages: new Map() };
  $: myId = get(authUser)?.userId;
  $: sortedMsgs = [...chat.messages].sort((a, b) => new Date(a.createdAt) - new Date(b.createdAt));

  // Unread = received messages where seenAt is null
  $: unreadCount = sortedMsgs.filter(m => m.senderId !== myId && !m.seenAt).length;

  // Peer presence
  $: peerUser = $otherUsers ? [...$otherUsers.values()].find(u => u.userId === peerId) : null;
  $: peerOnline = !!peerUser;
  $: peerLastSeen = peerUser?.lastSeen ?? null;

  onMount(() => {
    socket.emit('getSecretMsgs', { peerId, limit: 20 });
  });

  onDestroy(() => {
    lockSecretChat(peerId);
    sessionPin = '';
    gateOpen = false;
  });

  // ── Gate ──────────────────────────────────────────────────────
  function gatePinInput(e) {
    gatePin = e.target.value.replace(/\D/g, '');
  }

  async function submitGate() {
    if (gateUnlocking || gatePin.length < 4) return;
    gateError = '';
    gateUnlocking = true;

    // If there are received messages, validate PIN against first one
    const receivedMsgs = sortedMsgs.filter(m => m.senderId !== myId);
    if (receivedMsgs.length > 0) {
      try {
        await decryptMessage(receivedMsgs[0].ciphertext, receivedMsgs[0].iv, receivedMsgs[0].salt, gatePin);
      } catch {
        gateError = 'Wrong PIN — try again';
        gateUnlocking = false;
        return;
      }
    }

    sessionPin = gatePin;
    gatePin = '';
    gateUnlocking = false;
    gateOpen = true;
    await tick();
    scrollToBottom();
  }

  // ── Per-message inline decrypt ─────────────────────────────────
  function toggleInline(msgId) {
    if (activeDecryptId === msgId) {
      activeDecryptId = null;
    } else {
      activeDecryptId = msgId;
      inlinePins = { ...inlinePins, [msgId]: '' };
      inlineErrors = { ...inlineErrors, [msgId]: '' };
    }
  }

  function inlinePinInput(e, msgId) {
    inlinePins = { ...inlinePins, [msgId]: e.target.value.replace(/\D/g, '') };
  }

  async function decryptOne(msg) {
    const pin = inlinePins[msg.id] ?? '';
    if (!pin || pin.length < 4 || inlineUnlocking[msg.id]) return;
    inlineErrors = { ...inlineErrors, [msg.id]: '' };
    inlineUnlocking = { ...inlineUnlocking, [msg.id]: true };

    try {
      const plain = await decryptMessage(msg.ciphertext, msg.iv, msg.salt, pin);
      storeDecrypted(peerId, msg.id, plain);
      markSecretMsgSeen(msg.id);
      activeDecryptId = null;
    } catch {
      inlineErrors = { ...inlineErrors, [msg.id]: 'Wrong PIN' };
    } finally {
      inlineUnlocking = { ...inlineUnlocking, [msg.id]: false };
    }
  }

  // ── Compose ───────────────────────────────────────────────────
  async function send() {
    const text = composeText.trim();
    if (!text || !sessionPin || sending) return;
    sending = true;
    try {
      const { ciphertext, iv, salt } = await encryptMessage(text, sessionPin);
      socket.emit('sendSecretMsg', { receiverId: peerId, ciphertext, iv, salt });
      composeText = '';
    } catch {
      toasts.error('Failed to send');
    } finally {
      sending = false;
    }
    await tick();
    scrollToBottom();
  }

  function handleComposeKeydown(e) {
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

  // ── Formatters ────────────────────────────────────────────────
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

  function formatLastSeen(ts) {
    if (!ts) return '';
    const ms = typeof ts === 'number' ? ts : new Date(ts).getTime();
    const diff = Date.now() - ms;
    if (diff < 60_000) return 'just now';
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
    return new Date(ms).toLocaleDateString([], { month: 'short', day: 'numeric' });
  }

  function isUnread(msg) {
    return msg.senderId !== myId && !msg.seenAt;
  }

  function getMsgDisplay(msg) {
    if (msg.senderId === myId) return { isOwn: true, plain: null };
    const plain = chat.decryptedMessages.get(msg.id);
    return { isOwn: false, plain: plain ?? null };
  }

  // Peer's first name only (after gate unlocked)
  $: peerFirst = gateOpen ? (peerName || 'Them').split(' ')[0] : '••••••';
</script>

<div class="scp" transition:fade={{ duration: 120 }}>
  <!-- ── Header ─────────────────────────────────────────────── -->
  <div class="scp-header">
    <div class="scp-header-lock" class:scp-header-lock--open={gateOpen}>
      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <rect x="3" y="11" width="18" height="11" rx="2"/>
        <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
    </div>

    <div class="scp-header-info">
      <div class="scp-header-name-row">
        <span class="scp-header-name" class:scp-header-name--hidden={!gateOpen}>
          {gateOpen ? peerName : '••••••'}
        </span>
        {#if unreadCount > 0 && gateOpen}
          <span class="scp-unread-badge">{unreadCount}</span>
        {/if}
      </div>
      <span class="scp-header-sub">
        {#if !gateOpen}
          <span class="scp-subtext">Enter PIN to open</span>
        {:else if peerOnline}
          <span class="scp-dot scp-dot--online"></span><span class="scp-subtext">Online</span>
        {:else if peerLastSeen}
          <span class="scp-dot"></span><span class="scp-subtext">Last seen {formatLastSeen(peerLastSeen)}</span>
        {:else}
          <span class="scp-subtext">End-to-end encrypted</span>
        {/if}
      </span>
    </div>

    {#if gateOpen}
      <button class="scp-icon-btn" on:click={shareLink} aria-label="Share invite link" title="Copy invite link">
        {#if copyDone}
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#4ade80" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
        {:else}
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
        {/if}
      </button>
    {/if}

    <button class="scp-icon-btn" on:click={onClose} aria-label="Close">
      <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
      </svg>
    </button>
  </div>

  <!-- ── Gate ───────────────────────────────────────────────── -->
  {#if !gateOpen}
    <div class="scp-gate">
      <div class="scp-gate-ring">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <rect x="3" y="11" width="18" height="11" rx="2"/>
          <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
      </div>

      <div class="scp-gate-text">
        <p class="scp-gate-title">Secret Chat</p>
        <p class="scp-gate-sub">
          Enter the shared PIN to open this chat.<br>
          The same PIN will encrypt your replies.
        </p>
      </div>

      <label class="scp-sr" for="scp-gate-pin">Chat PIN</label>
      <input
        id="scp-gate-pin"
        class="scp-pin-input"
        type="tel"
        inputmode="numeric"
        pattern="[0-9]*"
        maxlength="8"
        placeholder="• • • •"
        bind:value={gatePin}
        on:input={gatePinInput}
        on:keydown={(e) => e.key === 'Enter' && submitGate()}
        disabled={gateUnlocking}
        autocomplete="off"
        autofocus
      />

      {#if gateError}
        <p class="scp-gate-error" role="alert">{gateError}</p>
      {/if}

      <button
        class="scp-primary-btn"
        on:click={submitGate}
        disabled={gateUnlocking || gatePin.length < 4}
      >
        {gateUnlocking ? 'Checking…' : 'Open Chat'}
      </button>
    </div>

  {:else}
    <!-- ── Messages ──────────────────────────────────────────── -->
    <div class="scp-msgs" bind:this={messagesEl}>
      {#if sortedMsgs.length === 0}
        <div class="scp-empty">
          <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
          <p>No messages yet</p>
          <span>Send the first secret message</span>
        </div>
      {/if}

      {#each sortedMsgs as msg (msg.id)}
        {@const d = getMsgDisplay(msg)}
        {@const unread = isUnread(msg)}
        {@const decrypted = !d.isOwn && d.plain !== null}
        {@const showInline = activeDecryptId === msg.id}

        <div class="scp-msg" class:scp-msg--own={d.isOwn} class:scp-msg--their={!d.isOwn}>

          {#if d.isOwn}
            <!-- Sent: always opaque -->
            <div class="scp-bubble scp-bubble--own">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
              <span>Encrypted</span>
            </div>
            <div class="scp-meta scp-meta--own">
              <span class="scp-time">{clockTime(msg.createdAt)}</span>
              <span class="scp-tick" class:scp-tick--seen={msg.seenAt} title={msg.seenAt ? `Read ${clockTime(msg.seenAt)}` : 'Sent'}>
                {#if msg.seenAt}
                  <svg width="14" height="8" viewBox="0 0 18 10" fill="none" stroke="#53bdeb" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-label="Read"><polyline points="1 5 5 9 13 1"/><polyline points="7 9 15 1"/></svg>
                {:else}
                  <svg width="10" height="8" viewBox="0 0 12 10" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-label="Sent"><polyline points="1 5 5 9 11 1"/></svg>
                {/if}
              </span>
            </div>

          {:else if decrypted}
            <!-- Received + decrypted -->
            <div class="scp-bubble scp-bubble--their">
              <p class="scp-body">{d.plain}</p>
            </div>
            <div class="scp-meta scp-meta--their">
              {#if unread}
                <span class="scp-unread-dot" aria-label="Unread"></span>
                <span class="scp-time scp-time--unread">{timeAgo(msg.createdAt)}</span>
              {:else}
                <span class="scp-time">{clockTime(msg.createdAt)}</span>
              {/if}
            </div>

          {:else}
            <!-- Received + locked -->
            <button
              class="scp-bubble scp-bubble--locked"
              class:scp-bubble--locked-active={showInline}
              on:click={() => toggleInline(msg.id)}
              aria-expanded={showInline}
              aria-label="Tap to enter PIN and decrypt"
            >
              {#if unread}
                <span class="scp-pulse" aria-hidden="true"></span>
              {/if}
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
              <span>Tap to read</span>
              {#if unread}
                <span class="scp-ago">{timeAgo(msg.createdAt)}</span>
              {/if}
            </button>

            {#if showInline}
              <div class="scp-inline-decrypt" transition:fade={{ duration: 100 }}>
                <label class="scp-sr" for="scp-inline-{msg.id}">PIN to decrypt</label>
                <input
                  id="scp-inline-{msg.id}"
                  class="scp-inline-pin"
                  type="tel"
                  inputmode="numeric"
                  pattern="[0-9]*"
                  maxlength="8"
                  placeholder="Enter PIN"
                  value={inlinePins[msg.id] ?? ''}
                  on:input={(e) => inlinePinInput(e, msg.id)}
                  on:keydown={(e) => e.key === 'Enter' && decryptOne(msg)}
                  disabled={inlineUnlocking[msg.id]}
                  autocomplete="off"
                  autofocus
                />
                <button
                  class="scp-inline-btn"
                  on:click={() => decryptOne(msg)}
                  disabled={inlineUnlocking[msg.id] || (inlinePins[msg.id] ?? '').length < 4}
                >
                  {inlineUnlocking[msg.id] ? '…' : 'Read'}
                </button>
                {#if inlineErrors[msg.id]}
                  <span class="scp-inline-err" role="alert">{inlineErrors[msg.id]}</span>
                {/if}
              </div>
            {/if}

            <div class="scp-meta scp-meta--their">
              {#if unread}
                <span class="scp-unread-dot" aria-label="Unread"></span>
                <span class="scp-time scp-time--unread">{timeAgo(msg.createdAt)}</span>
              {:else}
                <span class="scp-time">{clockTime(msg.createdAt)}</span>
              {/if}
            </div>
          {/if}
        </div>
      {/each}
    </div>

    <!-- ── Compose ─────────────────────────────────────────── -->
    <div class="scp-compose">
      <div class="scp-compose-inner">
        <label class="scp-sr" for="scp-compose-text">Secret message</label>
        <textarea
          id="scp-compose-text"
          class="scp-compose-text"
          rows="1"
          maxlength="2000"
          placeholder="Type a secret message…"
          bind:value={composeText}
          on:keydown={handleComposeKeydown}
          disabled={sending}
        ></textarea>
        <button
          class="scp-send-btn"
          on:click={send}
          disabled={sending || !composeText.trim()}
          aria-label="Send"
        >
          {#if sending}
            <div class="scp-send-ring"></div>
          {:else}
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
          {/if}
        </button>
      </div>
      <p class="scp-compose-hint">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
        Encrypted with your session PIN · {peerFirst} needs the same PIN to read
      </p>
    </div>
  {/if}
</div>

<style>
  /* ── Shell ─────────────────────────────────────────────────── */
  .scp {
    display: flex;
    flex-direction: column;
    background: #0f0f17;
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 20px;
    overflow: hidden;
    width: 100%;
    max-width: 420px;
    height: min(85dvh, 640px);
    box-shadow: 0 20px 60px rgba(0,0,0,0.6), 0 0 0 1px rgba(129,140,248,0.07);
  }
  @media (max-width: 480px) {
    .scp { max-width: 100%; border-radius: 24px 24px 0 0; height: 90dvh; }
  }

  /* ── Header ────────────────────────────────────────────────── */
  .scp-header {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 13px 12px 13px 16px;
    background: rgba(255,255,255,0.025);
    border-bottom: 1px solid rgba(255,255,255,0.06);
    flex-shrink: 0;
  }

  .scp-header-lock {
    color: rgba(255,255,255,0.2);
    display: flex;
    align-items: center;
    flex-shrink: 0;
    transition: color 0.3s;
  }
  .scp-header-lock--open { color: rgba(129,140,248,0.65); }

  .scp-header-info { flex: 1; min-width: 0; }

  .scp-header-name-row {
    display: flex;
    align-items: center;
    gap: 7px;
  }

  .scp-header-name {
    font-size: 14px;
    font-weight: 600;
    color: #e2e8f0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: system-ui, sans-serif;
    transition: opacity 0.25s;
  }
  .scp-header-name--hidden { opacity: 0.35; letter-spacing: 0.15em; }

  .scp-unread-badge {
    min-width: 18px;
    height: 18px;
    padding: 0 5px;
    border-radius: 9px;
    background: #818cf8;
    color: #fff;
    font-size: 10px;
    font-weight: 700;
    font-family: system-ui, sans-serif;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .scp-header-sub {
    display: flex;
    align-items: center;
    gap: 5px;
    margin-top: 2px;
  }

  .scp-subtext {
    font-size: 11px;
    color: rgba(255,255,255,0.35);
    font-family: system-ui, sans-serif;
  }

  .scp-dot {
    width: 6px; height: 6px;
    border-radius: 50%;
    background: rgba(255,255,255,0.2);
    flex-shrink: 0;
  }
  .scp-dot--online {
    background: #4ade80;
    box-shadow: 0 0 0 2px rgba(74,222,128,0.25);
    animation: scp-pulse-green 2s ease-in-out infinite;
  }

  .scp-icon-btn {
    width: 32px; height: 32px;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none;
    color: rgba(255,255,255,0.35);
    cursor: pointer;
    border-radius: 8px;
    transition: background 0.15s, color 0.15s;
    flex-shrink: 0;
  }
  .scp-icon-btn:hover { background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.7); }

  /* ── Gate ──────────────────────────────────────────────────── */
  .scp-gate {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    padding: 32px 28px;
    text-align: center;
  }

  .scp-gate-ring {
    width: 64px; height: 64px;
    border-radius: 50%;
    background: rgba(129,140,248,0.08);
    border: 1px solid rgba(129,140,248,0.18);
    display: flex; align-items: center; justify-content: center;
    color: rgba(129,140,248,0.7);
    margin-bottom: 4px;
  }

  .scp-gate-text { display: flex; flex-direction: column; gap: 8px; }

  .scp-gate-title {
    margin: 0;
    font-size: 17px; font-weight: 700;
    color: #e2e8f0;
    font-family: system-ui, sans-serif;
  }

  .scp-gate-sub {
    margin: 0;
    font-size: 12px;
    color: rgba(255,255,255,0.35);
    line-height: 1.65;
    font-family: system-ui, sans-serif;
  }

  .scp-pin-input {
    width: 100%; max-width: 220px;
    padding: 14px 16px;
    border-radius: 14px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(255,255,255,0.04);
    color: #e2e8f0;
    font-size: 24px;
    letter-spacing: 0.35em;
    text-align: center;
    outline: none;
    font-variant-numeric: tabular-nums;
    font-family: system-ui, monospace, sans-serif;
    box-sizing: border-box;
    transition: border-color 0.15s, box-shadow 0.15s;
    -webkit-appearance: none;
  }
  .scp-pin-input:focus {
    border-color: rgba(129,140,248,0.5);
    box-shadow: 0 0 0 3px rgba(129,140,248,0.1);
  }

  .scp-gate-error {
    margin: 0;
    font-size: 12px;
    color: #f87171;
    font-family: system-ui, sans-serif;
  }

  .scp-primary-btn {
    width: 100%; max-width: 220px;
    padding: 13px;
    border-radius: 13px;
    border: none;
    background: rgba(129,140,248,0.85);
    color: #fff;
    font-size: 14px; font-weight: 600;
    cursor: pointer;
    font-family: system-ui, sans-serif;
    min-height: 48px;
    transition: background 0.15s;
  }
  .scp-primary-btn:hover:not(:disabled) { background: #818cf8; }
  .scp-primary-btn:disabled { opacity: 0.3; cursor: not-allowed; }

  /* ── Messages ──────────────────────────────────────────────── */
  .scp-msgs {
    flex: 1;
    overflow-y: auto;
    padding: 14px 14px 10px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    overscroll-behavior: contain;
  }
  .scp-msgs::-webkit-scrollbar { width: 3px; }
  .scp-msgs::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.08); border-radius: 2px; }

  .scp-empty {
    flex: 1;
    display: flex; flex-direction: column;
    align-items: center; justify-content: center;
    gap: 8px;
    color: rgba(255,255,255,0.18);
    padding: 40px 0;
    text-align: center;
  }
  .scp-empty p { margin: 0; font-size: 14px; color: rgba(255,255,255,0.28); font-family: system-ui, sans-serif; }
  .scp-empty span { font-size: 12px; font-family: system-ui, sans-serif; }

  .scp-msg {
    display: flex; flex-direction: column;
    max-width: 78%;
    gap: 3px;
    margin-bottom: 4px;
  }
  .scp-msg--own { align-self: flex-end; align-items: flex-end; }
  .scp-msg--their { align-self: flex-start; align-items: flex-start; }

  /* Bubbles */
  .scp-bubble {
    padding: 9px 13px;
    border-radius: 16px;
    font-size: 14px;
    line-height: 1.55;
    font-family: system-ui, sans-serif;
  }

  .scp-bubble--own {
    display: flex; align-items: center; gap: 7px;
    background: rgba(129,140,248,0.13);
    border: 1px solid rgba(129,140,248,0.18);
    border-bottom-right-radius: 4px;
    color: rgba(129,140,248,0.65);
    font-size: 12px; font-weight: 500;
  }

  .scp-bubble--their {
    background: rgba(255,255,255,0.07);
    border: 1px solid rgba(255,255,255,0.06);
    border-bottom-left-radius: 4px;
    color: #e2e8f0;
    word-break: break-word;
  }

  .scp-bubble--locked {
    display: flex; align-items: center; gap: 8px;
    background: rgba(255,255,255,0.04);
    border: 1px dashed rgba(255,255,255,0.1);
    border-bottom-left-radius: 4px;
    color: rgba(255,255,255,0.3);
    font-size: 13px;
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s;
    position: relative;
  }
  .scp-bubble--locked:hover { background: rgba(255,255,255,0.07); border-color: rgba(255,255,255,0.18); }
  .scp-bubble--locked-active { border-color: rgba(129,140,248,0.35); background: rgba(129,140,248,0.06); }

  .scp-ago {
    margin-left: auto;
    font-size: 10px;
    color: rgba(255,255,255,0.2);
    font-family: system-ui, sans-serif;
    white-space: nowrap;
  }

  /* Glowing pulse dot on locked unread bubbles */
  .scp-pulse {
    width: 7px; height: 7px;
    border-radius: 50%;
    background: #818cf8;
    box-shadow: 0 0 0 0 rgba(129,140,248,0.6);
    animation: scp-glow 1.6s ease-in-out infinite;
    flex-shrink: 0;
  }

  .scp-body { margin: 0; }

  /* Meta row (time + ticks) */
  .scp-meta {
    display: flex; align-items: center; gap: 4px;
    padding: 0 2px;
  }
  .scp-meta--own { justify-content: flex-end; }
  .scp-meta--their { justify-content: flex-start; }

  .scp-time {
    font-size: 10px;
    color: rgba(255,255,255,0.22);
    font-family: system-ui, sans-serif;
  }
  .scp-time--unread {
    color: rgba(129,140,248,0.6);
    font-weight: 500;
  }

  /* Small dot beside time for unread decrypted messages */
  .scp-unread-dot {
    width: 5px; height: 5px;
    border-radius: 50%;
    background: #818cf8;
    box-shadow: 0 0 4px rgba(129,140,248,0.7);
    animation: scp-glow 1.6s ease-in-out infinite;
    flex-shrink: 0;
  }

  .scp-tick { display: flex; align-items: center; color: rgba(255,255,255,0.28); }
  .scp-tick--seen { color: #53bdeb; }

  /* ── Inline decrypt ────────────────────────────────────────── */
  .scp-inline-decrypt {
    display: flex;
    align-items: center;
    gap: 7px;
    flex-wrap: wrap;
    margin-top: 2px;
    padding: 8px 10px;
    background: rgba(129,140,248,0.06);
    border: 1px solid rgba(129,140,248,0.14);
    border-radius: 12px;
    max-width: 260px;
  }

  .scp-inline-pin {
    flex: 1;
    min-width: 90px;
    padding: 7px 10px;
    border-radius: 8px;
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(0,0,0,0.3);
    color: #e2e8f0;
    font-size: 16px;
    letter-spacing: 0.25em;
    text-align: center;
    outline: none;
    font-variant-numeric: tabular-nums;
    font-family: system-ui, monospace, sans-serif;
    -webkit-appearance: none;
    transition: border-color 0.15s;
  }
  .scp-inline-pin:focus { border-color: rgba(129,140,248,0.5); }

  .scp-inline-btn {
    padding: 7px 14px;
    border-radius: 8px;
    border: none;
    background: rgba(129,140,248,0.75);
    color: #fff;
    font-size: 13px; font-weight: 600;
    cursor: pointer;
    font-family: system-ui, sans-serif;
    min-height: 36px;
    transition: background 0.15s;
    white-space: nowrap;
  }
  .scp-inline-btn:hover:not(:disabled) { background: #818cf8; }
  .scp-inline-btn:disabled { opacity: 0.3; cursor: not-allowed; }

  .scp-inline-err {
    font-size: 11px;
    color: #f87171;
    font-family: system-ui, sans-serif;
    width: 100%;
  }

  /* ── Compose ────────────────────────────────────────────────── */
  .scp-compose {
    padding: 10px 14px 14px;
    border-top: 1px solid rgba(255,255,255,0.06);
    display: flex; flex-direction: column;
    gap: 7px;
    flex-shrink: 0;
    background: rgba(0,0,0,0.12);
  }

  .scp-compose-inner {
    display: flex; align-items: flex-end;
    gap: 8px;
  }

  .scp-compose-text {
    flex: 1;
    resize: none;
    padding: 10px 12px;
    border-radius: 14px;
    border: 1px solid rgba(255,255,255,0.09);
    background: rgba(255,255,255,0.04);
    color: #e2e8f0;
    font-size: 14px;
    line-height: 1.5;
    outline: none;
    font-family: system-ui, sans-serif;
    transition: border-color 0.15s;
    -webkit-appearance: none;
    field-sizing: content;
    max-height: 120px;
    overflow-y: auto;
  }
  .scp-compose-text:focus { border-color: rgba(255,255,255,0.18); }
  .scp-compose-text::placeholder { color: rgba(255,255,255,0.2); }

  .scp-send-btn {
    width: 44px; height: 44px;
    border-radius: 13px;
    border: none;
    background: rgba(129,140,248,0.8);
    color: #fff;
    cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
    transition: background 0.15s;
  }
  .scp-send-btn:hover:not(:disabled) { background: #818cf8; }
  .scp-send-btn:disabled { opacity: 0.28; cursor: not-allowed; }

  .scp-send-ring {
    width: 16px; height: 16px;
    border: 2px solid rgba(255,255,255,0.25);
    border-top-color: #fff;
    border-radius: 50%;
    animation: scp-spin 0.8s linear infinite;
  }

  .scp-compose-hint {
    display: flex; align-items: center; gap: 5px;
    margin: 0;
    font-size: 10px;
    color: rgba(255,255,255,0.18);
    font-family: system-ui, sans-serif;
  }

  /* ── Utilities ─────────────────────────────────────────────── */
  .scp-sr {
    position: absolute; width: 1px; height: 1px;
    padding: 0; margin: -1px;
    overflow: hidden; clip: rect(0,0,0,0);
    white-space: nowrap; border: 0;
  }

  @keyframes scp-spin { to { transform: rotate(360deg); } }

  @keyframes scp-glow {
    0%, 100% { box-shadow: 0 0 0 0 rgba(129,140,248,0.55); }
    50%       { box-shadow: 0 0 0 4px rgba(129,140,248,0); }
  }

  @keyframes scp-pulse-green {
    0%, 100% { box-shadow: 0 0 0 0 rgba(74,222,128,0.4); }
    50%       { box-shadow: 0 0 0 4px rgba(74,222,128,0); }
  }
</style>
