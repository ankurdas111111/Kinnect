<script>
  import { get } from 'svelte/store';
  import { onMount, onDestroy, tick } from 'svelte';
  import { fade } from 'svelte/transition';
  import { socket, markSecretMsgSeen, createSecretChatInvite } from '../lib/socket.js';
  import { authUser } from '../lib/stores/auth.js';
  import { secretChats, lockSecretChat, storeDecrypted, secretChatPresence } from '../lib/stores/secretChat.js';
  import { otherUsers } from '../lib/stores/map.js';
  import { encryptMessage, decryptMessage } from '../lib/crypto.js';
  import { toasts } from '../lib/stores/toast.js';
  import EmojiPicker from './primitives/EmojiPicker.svelte';
  import StickerPicker from './primitives/StickerPicker.svelte';

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

  // Re-lock state — messages can be locked again after reading
  let lockedSet = new Set();        // msgId → manually re-locked
  let lockCountdowns = {};          // msgId → seconds remaining before auto-lock
  let lockIntervals = {};           // msgId → setInterval handle

  const AUTO_LOCK_SECS = 30;

  function relockMsg(msgId) {
    // Clear any running countdown
    if (lockIntervals[msgId]) {
      clearInterval(lockIntervals[msgId]);
      delete lockIntervals[msgId];
    }
    delete lockCountdowns[msgId];
    lockCountdowns = { ...lockCountdowns };
    // Add to locked set — triggers re-render to locked bubble
    lockedSet = new Set([...lockedSet, msgId]);
    // Reset inline state so PIN input is fresh next time
    inlinePins = { ...inlinePins, [msgId]: '' };
    inlineErrors = { ...inlineErrors, [msgId]: '' };
    activeDecryptId = null;
  }

  function startAutoLock(msgId) {
    if (lockIntervals[msgId]) clearInterval(lockIntervals[msgId]);
    lockCountdowns = { ...lockCountdowns, [msgId]: AUTO_LOCK_SECS };
    lockIntervals[msgId] = setInterval(() => {
      const cur = lockCountdowns[msgId];
      if (cur == null || cur <= 1) {
        relockMsg(msgId);
      } else {
        lockCountdowns = { ...lockCountdowns, [msgId]: cur - 1 };
      }
    }, 1000);
  }

  let composeText = '';
  let sending = false;
  let copyDone = false;
  let messagesEl;
  let emojiOpen = false;
  let emojiAnchor;
  let stickerOpen = false;
  let stickerAnchor;
  let panicMode = false;

  function restoreFromPanic() {
    panicMode = false;
    // Re-lock everything — force PIN re-entry
    gateOpen = false;
    sessionPin = '';
    gatePin = '';
    // Clear all auto-lock timers and decrypted state
    for (const id of Object.keys(lockIntervals)) clearInterval(lockIntervals[id]);
    lockIntervals = {};
    lockCountdowns = {};
    lockedSet = new Set();
    activeDecryptId = null;
    inlinePins = {};
    inlineErrors = {};
  }

  $: chat = $secretChats.get(peerId) ?? { messages: [], locked: true, decryptedMessages: new Map() };
  $: myId = get(authUser)?.userId;
  $: sortedMsgs = [...chat.messages].sort((a, b) => new Date(a.createdAt) - new Date(b.createdAt));

  // Unread = received messages where seenAt is null
  $: unreadCount = sortedMsgs.filter(m => m.senderId !== myId && !m.seenAt).length;

  // Peer presence — chat-specific (did they open THIS chat)
  $: peerPresence = $secretChatPresence.get(peerId) ?? null;
  $: peerChatOpen = peerPresence?.open ?? false;
  $: peerLastOpenedAt = (peerPresence && !peerPresence.open) ? peerPresence.at : null;

  // General online status — is this peer connected to Kinnect at all?
  $: peerKinnectOnline = Array.from($otherUsers.values()).some(u => u.userId === peerId && u.online !== false);

  function emitPresence(open) {
    socket.emit('secretChatPresence', { peerId, open });
  }

  onMount(() => {
    socket.emit('getSecretMsgs', { peerId, limit: 20 });
    emitPresence(true);
  });

  onDestroy(() => {
    emitPresence(false);
    lockSecretChat(peerId);
    sessionPin = '';
    gateOpen = false;
    // Clear all running auto-lock timers
    for (const id of Object.values(lockIntervals)) clearInterval(id);
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
      // Remove from lockedSet if it was manually re-locked before
      lockedSet.delete(msg.id);
      lockedSet = new Set(lockedSet);
      // Start auto-lock countdown
      startAutoLock(msg.id);
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

  function getMsgDisplay(msg, locked, decryptedMsgs) {
    if (msg.senderId === myId) return { isOwn: true, plain: null };
    // If manually or auto re-locked, treat as still locked
    if (locked.has(msg.id)) return { isOwn: false, plain: null };
    const plain = decryptedMsgs.get(msg.id);
    return { isOwn: false, plain: plain ?? null };
  }

  // Peer's first name only (after gate unlocked)
  $: peerFirst = gateOpen ? (peerName || 'Them').split(' ')[0] : '••••••';

  const GIF_RE = /^\[gif:(https?:\/\/[^\]]+)\]$/;
  function parseGif(text) {
    const m = GIF_RE.exec(text);
    return m ? m[1] : null;
  }
</script>

{#if panicMode}
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
  <div class="scp-panic" on:click={restoreFromPanic}></div>
{/if}

<div class="scp-backdrop" transition:fade={{ duration: 180 }} on:click|self={onClose}>
<div class="scp">
  <!-- Mobile drag handle affordance -->
  <div class="scp-drag-handle" aria-hidden="true"></div>

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
        {:else if peerChatOpen}
          <span class="scp-dot scp-dot--online"></span><span class="scp-subtext">In this chat</span>
        {:else if peerKinnectOnline}
          <span class="scp-dot scp-dot--online" style="background:#60a5fa;box-shadow:0 0 0 2px rgba(96,165,250,0.25)"></span><span class="scp-subtext">Online</span>
        {:else if peerLastOpenedAt}
          <span class="scp-dot"></span><span class="scp-subtext">Last seen {formatLastSeen(peerLastOpenedAt)}</span>
        {:else}
          <span class="scp-dot" style="background:rgba(255,255,255,0.12)"></span><span class="scp-subtext">Offline</span>
        {/if}
      </span>
    </div>

    {#if gateOpen}
      <button
        class="scp-share-pill"
        class:scp-share-pill--copied={copyDone}
        on:click={shareLink}
        aria-label={copyDone ? 'Link copied' : 'Copy invite link'}
      >
        {#if copyDone}
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
          <span>Copied!</span>
        {:else}
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
          <span>Share</span>
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
        {@const d = getMsgDisplay(msg, lockedSet, chat.decryptedMessages)}
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
            <div class="scp-bubble scp-bubble--their scp-bubble--decrypted">
              {#if parseGif(d.plain)}
                <img src={parseGif(d.plain)} class="msg-sticker" alt="sticker" loading="lazy" />
              {:else}
                <p class="scp-body">{d.plain}</p>
              {/if}
              <button
                class="scp-relock-btn"
                on:click={() => relockMsg(msg.id)}
                title="Lock message"
                aria-label="Lock message"
              >
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
              </button>
            </div>
            <div class="scp-meta scp-meta--their">
              {#if lockCountdowns[msg.id] != null}
                <span class="scp-countdown">Locks in {lockCountdowns[msg.id]}s</span>
              {:else if unread}
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
        <!-- Panic / blank button -->
        <button
          class="scp-compose-icon-btn"
          on:click={() => panicMode = true}
          aria-label="Blank screen"
          title="Blank screen (tap screen to restore)"
          type="button"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
            <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
            <line x1="1" y1="1" x2="23" y2="23"/>
          </svg>
        </button>

        <button
          class="scp-emoji-btn"
          bind:this={emojiAnchor}
          on:click={() => { emojiOpen = !emojiOpen; stickerOpen = false; }}
          aria-label="Emoji picker"
          aria-expanded={emojiOpen}
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
          class="scp-compose-icon-btn"
          bind:this={stickerAnchor}
          on:click={() => { stickerOpen = !stickerOpen; emojiOpen = false; }}
          aria-label="Sticker picker"
          aria-expanded={stickerOpen}
          title="Stickers"
          type="button"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <circle cx="12" cy="12" r="10"/>
            <path d="M8 13s1.5 2 4 2 4-2 4-2"/>
            <line x1="9" y1="9" x2="9.01" y2="9"/>
            <line x1="15" y1="9" x2="15.01" y2="9"/>
            <path d="M12 2v2M12 20v2M2 12h2M20 12h2"/>
          </svg>
        </button>

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

      <EmojiPicker
        open={emojiOpen}
        anchor={emojiAnchor}
        on:pick={(e) => { composeText += e.detail; }}
        on:close={() => emojiOpen = false}
      />
      <StickerPicker
        open={stickerOpen}
        anchor={stickerAnchor}
        on:pick={(e) => { composeText += e.detail; stickerOpen = false; }}
        on:close={() => stickerOpen = false}
      />
      <p class="scp-compose-hint">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
        Encrypted with your session PIN · {peerFirst} needs the same PIN to read
      </p>
    </div>
  {/if}
</div>
</div>

<style>
  /* ── Backdrop / overlay shell ──────────────────────────────── */
  .scp-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-modal, 5000);
    background: rgba(0, 0, 0, 0.55);
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 16px;
    /* Mobile: bottom-sheet pattern */
  }
  @media (max-width: 767px) {
    .scp-backdrop {
      align-items: flex-end;
      padding: 0;
    }
  }

  /* ── Panel shell ───────────────────────────────────────────── */
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
  @media (max-width: 767px) {
    .scp {
      max-width: 100%;
      border-radius: 20px 20px 0 0;
      height: 88dvh;
      /* Respect notch / home-bar on iOS */
      padding-bottom: env(safe-area-inset-bottom, 0px);
      animation: scp-slide-up 0.28s cubic-bezier(0.32, 0.72, 0, 1) both;
    }
  }

  @keyframes scp-slide-up {
    from { transform: translateY(100%); }
    to   { transform: translateY(0);    }
  }

  /* ── Drag handle (mobile only) ────────────────────────────── */
  .scp-drag-handle {
    display: none;
    width: 36px;
    height: 4px;
    border-radius: 2px;
    background: rgba(255,255,255,0.18);
    margin: 10px auto 4px;
    flex-shrink: 0;
  }
  @media (max-width: 767px) {
    .scp-drag-handle { display: block; }
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
    width: 36px; height: 36px;
    min-width: 44px; min-height: 44px;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none;
    color: rgba(255,255,255,0.35);
    cursor: pointer;
    border-radius: 8px;
    transition: background 0.15s, color 0.15s;
    flex-shrink: 0;
    touch-action: manipulation;
  }
  .scp-icon-btn:hover { background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.7); }

  /* ── Share pill ─────────────────────────────────────────────── */
  .scp-share-pill {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 0 12px 0 9px;
    height: 32px;
    /* Ensure ≥44px touch area via margin compensation */
    min-height: 44px;
    border-radius: 20px;
    border: 1px solid rgba(129,140,248,0.28);
    background: rgba(129,140,248,0.09);
    color: rgba(129,140,248,0.9);
    font-size: 12px;
    font-weight: 600;
    font-family: system-ui, sans-serif;
    cursor: pointer;
    flex-shrink: 0;
    white-space: nowrap;
    transition: background 0.15s, border-color 0.15s, color 0.15s;
    touch-action: manipulation;
  }
  .scp-share-pill:hover { background: rgba(129,140,248,0.16); border-color: rgba(129,140,248,0.45); }
  .scp-share-pill--copied {
    border-color: rgba(74,222,128,0.35);
    background: rgba(74,222,128,0.09);
    color: rgba(74,222,128,0.95);
  }

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
    touch-action: manipulation;
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

  .scp-bubble--decrypted {
    position: relative;
    padding-right: 30px; /* room for lock button */
  }

  .scp-relock-btn {
    position: absolute;
    top: 4px; right: 4px;
    /* Visual size stays small; touch area expanded via padding (11px → 22+22=44px total) */
    width: 22px; height: 22px;
    padding: 11px;
    margin: -11px;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none; cursor: pointer;
    color: rgba(255,255,255,0.18);
    border-radius: 8px;
    transition: color 0.15s, background 0.15s;
    touch-action: manipulation;
    /* Ensure bubble can fit the expanded hit-area */
    box-sizing: content-box;
  }
  .scp-relock-btn:hover { color: rgba(129,140,248,0.7); background: rgba(129,140,248,0.08); }

  .scp-countdown {
    font-size: 10px;
    color: rgba(129,140,248,0.5);
    font-family: system-ui, sans-serif;
    font-variant-numeric: tabular-nums;
  }

  .scp-bubble--locked {
    display: flex; align-items: center; gap: 8px;
    background: rgba(255,255,255,0.04);
    border: 1px dashed rgba(255,255,255,0.1);
    border-bottom-left-radius: 4px;
    color: rgba(255,255,255,0.3);
    font-size: 13px;
    cursor: pointer;
    min-height: 44px;
    transition: background 0.15s, border-color 0.15s;
    position: relative;
    touch-action: manipulation;
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
    gap: 8px;
    flex-wrap: nowrap;
    margin-top: 4px;
    padding: 8px 10px;
    background: rgba(129,140,248,0.06);
    border: 1px solid rgba(129,140,248,0.14);
    border-radius: 12px;
    /* Fill the message bubble width, never overflow */
    width: 100%;
    max-width: 100%;
    box-sizing: border-box;
  }

  .scp-inline-pin {
    flex: 1;
    min-width: 0;
    padding: 10px 10px;
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
    /* 44px minimum touch height */
    min-height: 44px;
    transition: border-color 0.15s;
    touch-action: manipulation;
  }
  .scp-inline-pin:focus { border-color: rgba(129,140,248,0.5); }
  /* Prevent iOS zoom on focus */
  @media (max-width: 767px) {
    .scp-inline-pin { font-size: 18px; }
  }

  .scp-inline-btn {
    padding: 10px 16px;
    border-radius: 8px;
    border: none;
    background: rgba(129,140,248,0.75);
    color: #fff;
    font-size: 13px; font-weight: 600;
    cursor: pointer;
    font-family: system-ui, sans-serif;
    /* 44px touch target */
    min-height: 44px;
    flex-shrink: 0;
    transition: background 0.15s;
    white-space: nowrap;
    touch-action: manipulation;
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

  /* Panic overlay */
  .scp-panic {
    position: fixed;
    inset: 0;
    background: #fff;
    z-index: 99999;
    cursor: default;
  }

  .scp-emoji-btn {
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
  .scp-emoji-btn:hover { color: rgba(255,255,255,0.65); background: rgba(255,255,255,0.06); }

  .scp-compose-icon-btn {
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
  .scp-compose-icon-btn:hover { color: rgba(255,255,255,0.65); background: rgba(255,255,255,0.06); }

  :global(.msg-sticker) {
    max-width: 120px;
    max-height: 120px;
    border-radius: 8px;
    display: block;
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
  /* Prevent iOS from zooming in when textarea is focused */
  @media (max-width: 767px) {
    .scp-compose-text { font-size: 16px; }
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
    touch-action: manipulation;
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
