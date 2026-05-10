<script>
  import { onMount, onDestroy, tick, afterUpdate } from 'svelte';
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

  // Session PIN — entered via PIN pad at gate, used for ALL outgoing messages
  let sessionPin = '';
  let pinDigits = [];
  let gateError = '';
  let gateUnlocking = false;
  let gateOpen = false;

  $: gatePin = pinDigits.join('');

  // Per-message inline decryption
  let activeDecryptId = null;
  let inlinePins = {};
  let inlineErrors = {};
  let inlineUnlocking = {};

  // Re-lock state
  let lockedSet = new Set();
  let lockCountdowns = {};
  let lockIntervals = {};

  const AUTO_LOCK_SECS = 30;

  function relockMsg(msgId) {
    if (lockIntervals[msgId]) {
      clearInterval(lockIntervals[msgId]);
      delete lockIntervals[msgId];
    }
    delete lockCountdowns[msgId];
    lockCountdowns = { ...lockCountdowns };
    lockedSet = new Set([...lockedSet, msgId]);
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

  // ── iOS keyboard avoidance ───────────────────────────────────
  let scpEl;

  function onVVChange() {
    if (!scpEl) return;
    const vv = window.visualViewport;
    if (!vv) return;
    const kbH = Math.max(0, window.innerHeight - vv.height - vv.offsetTop);
    scpEl.style.marginBottom = kbH > 50 ? `${kbH}px` : '';
    scpEl.style.transition = 'margin-bottom 0.2s ease';
  }

  $: if (gateOpen && scpEl) { scpEl.style.marginBottom = ''; }

  // ── PIN input helpers ────────────────────────────────────────
  let pinInputEl;
  $: if (!gateOpen && pinInputEl) setTimeout(() => pinInputEl?.focus(), 80);
  $: if (pinInputEl && pinDigits.length === 0) pinInputEl.value = '';

  function handlePinInput(e) {
    const digits = e.target.value.replace(/\D/g, '').slice(0, 8);
    e.target.value = digits;
    pinDigits = digits.split('');
  }

  function handlePinKeydown(e) {
    if (e.key === 'Enter') { e.preventDefault(); submitGate(); }
  }

  let composeText = '';
  let sending = false;
  let copyDone = false;
  let messagesEl;
  let composeTextEl;
  let emojiOpen = false;
  let emojiAnchor;
  let stickerOpen = false;
  let stickerAnchor;
  let panicMode = false;

  // Delete confirmation — two-tap
  let deletingMsgId = null;

  // PIN shake feedback
  let pinShake = false;
  let _pinShakeTimer = null;
  function triggerShake() {
    pinShake = true;
    clearTimeout(_pinShakeTimer);
    _pinShakeTimer = setTimeout(() => { pinShake = false; }, 520);
  }

  function deleteMsg(msgId) {
    if (deletingMsgId === msgId) {
      socket.emit('deleteSecretMsg', { id: msgId });
      deletingMsgId = null;
    } else {
      deletingMsgId = msgId;
      setTimeout(() => { if (deletingMsgId === msgId) deletingMsgId = null; }, 2500);
    }
  }

  function restoreFromPanic() {
    panicMode = false;
    gateOpen = false;
    sessionPin = '';
    pinDigits = [];
    for (const id of Object.keys(lockIntervals)) clearInterval(lockIntervals[id]);
    lockIntervals = {};
    lockCountdowns = {};
    lockedSet = new Set();
    activeDecryptId = null;
    inlinePins = {};
    inlineErrors = {};
  }

  $: chat = $secretChats.get(peerId) ?? { messages: [], locked: true, decryptedMessages: new Map() };
  $: myId = $authUser?.userId;
  $: sortedMsgs = [...chat.messages].sort((a, b) => new Date(a.createdAt) - new Date(b.createdAt));

  $: groupedMsgs = sortedMsgs.map((msg, i) => {
    const prev = sortedMsgs[i - 1];
    const next = sortedMsgs[i + 1];
    const GAP = 2 * 60 * 1000;
    const samePrev = prev && prev.senderId === msg.senderId &&
      new Date(msg.createdAt) - new Date(prev.createdAt) < GAP;
    const sameNext = next && next.senderId === msg.senderId &&
      new Date(next.createdAt) - new Date(msg.createdAt) < GAP;
    return { ...msg, groupFirst: !samePrev, groupLast: !sameNext };
  });

  $: unreadCount = sortedMsgs.filter(m => m.senderId !== myId && !m.seenAt).length;

  $: peerPresence = $secretChatPresence.get(peerId) ?? null;
  $: peerChatOpen = peerPresence?.open ?? false;
  $: peerLastOpenedAt = (peerPresence && !peerPresence.open) ? peerPresence.at : null;
  $: peerKinnectOnline = Array.from($otherUsers.values()).some(u => u.userId === peerId && u.online !== false);
  $: peerLastMsgAt = sortedMsgs.filter(m => m.senderId === peerId).slice(-1)[0]?.createdAt ?? null;
  $: peerLastSeenAt = peerLastOpenedAt ?? peerLastMsgAt;

  function emitPresence(open) {
    socket.emit('secretChatPresence', { peerId, open });
  }

  let loadingMessages = true;
  $: if ($secretChats.has(peerId) && loadingMessages) loadingMessages = false;

  onMount(() => {
    socket.emit('getSecretMsgs', { peerId, limit: 20 });
    emitPresence(true);
    setTimeout(() => { loadingMessages = false; }, 4000);
    window.visualViewport?.addEventListener('resize', onVVChange);
    window.visualViewport?.addEventListener('scroll', onVVChange);
  });

  onDestroy(() => {
    emitPresence(false);
    lockSecretChat(peerId);
    sessionPin = '';
    gateOpen = false;
    for (const id of Object.values(lockIntervals)) clearInterval(id);
    window.visualViewport?.removeEventListener('resize', onVVChange);
    window.visualViewport?.removeEventListener('scroll', onVVChange);
  });

  // ── Gate ──────────────────────────────────────────────────────
  async function submitGate() {
    if (gateUnlocking || pinDigits.length < 4) return;
    gateError = '';
    gateUnlocking = true;
    sessionPin = pinDigits.join('');
    pinDigits = [];
    gateOpen = true;
    await tick();
    scrollToBottom();
    gateUnlocking = false;
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
      lockedSet.delete(msg.id);
      lockedSet = new Set(lockedSet);
      startAutoLock(msg.id);
    } catch {
      inlineErrors = { ...inlineErrors, [msg.id]: 'Wrong PIN' };
      triggerShake();
    } finally {
      inlineUnlocking = { ...inlineUnlocking, [msg.id]: false };
    }
  }

  // ── Photo sending ─────────────────────────────────────────────
  // Images are compressed client-side (WebP → JPEG fallback, multi-pass size reduction),
  // then encrypted with the session PIN before being stored as opaque ciphertext blobs.
  // Target: ≤100 KB binary (≤137 KB base64) per photo to fit within the 1 GB DB budget.
  let photoInputEl;
  let cameraInputEl;
  let photoSending = false;
  let attachMenuOpen = false;

  // Downscale longest edge to this before encoding
  const MAX_PHOTO_EDGE = 720;
  // Binary budget per photo — 100 KB ensures base64 stays under ~137 KB
  const MAX_BINARY_BYTES = 100_000;

  /**
   * Ultra-compress an image File via multi-pass Canvas encoding.
   * Returns the compressed DataURL string.
   * Rejects with { tooLarge: true, sizeKB: N } if all passes fail to reach target.
   */
  function compressImage(file) {
    return new Promise((resolve, reject) => {
      const img = new Image();
      const url = URL.createObjectURL(file);
      img.onload = () => {
        URL.revokeObjectURL(url);
        let { width, height } = img;
        if (width > MAX_PHOTO_EDGE || height > MAX_PHOTO_EDGE) {
          if (width >= height) {
            height = Math.round((height / width) * MAX_PHOTO_EDGE);
            width = MAX_PHOTO_EDGE;
          } else {
            width = Math.round((width / height) * MAX_PHOTO_EDGE);
            height = MAX_PHOTO_EDGE;
          }
        }

        // Returns WebP if supported, JPEG otherwise
        function encode(cvs, q) {
          const webp = cvs.toDataURL('image/webp', q);
          if (webp.startsWith('data:image/webp')) return webp;
          return cvs.toDataURL('image/jpeg', q);
        }

        const c0 = document.createElement('canvas');
        c0.width = width; c0.height = height;
        c0.getContext('2d').drawImage(img, 0, 0, width, height);
        let result = encode(c0, 0.80);

        // Multi-pass: shrink until under budget (result.length * 0.75 ≈ binary bytes)
        let w = width, h = height;
        for (let pass = 0; pass < 4 && result.length * 0.75 > MAX_BINARY_BYTES; pass++) {
          const scale = Math.sqrt(MAX_BINARY_BYTES / (result.length * 0.75)) * 0.88;
          w = Math.max(120, Math.round(w * scale));
          h = Math.max(90, Math.round(h * scale));
          const c = document.createElement('canvas');
          c.width = w; c.height = h;
          c.getContext('2d').drawImage(img, 0, 0, w, h);
          result = encode(c, 0.72 - pass * 0.05);
        }

        const finalBinaryKB = Math.round((result.length * 0.75) / 1024);
        if (result.length * 0.75 > MAX_BINARY_BYTES * 1.5) {
          // After 4 passes still too large — reject with size info so the toast is specific
          reject({ tooLarge: true, sizeKB: finalBinaryKB });
          return;
        }

        resolve(result);
      };
      img.onerror = () => { URL.revokeObjectURL(url); reject(new Error('Image load failed')); };
      img.src = url;
    });
  }

  function toggleAttachMenu() {
    attachMenuOpen = !attachMenuOpen;
    if (attachMenuOpen) {
      const close = (e) => {
        if (!e.target.closest?.('.scp-attach-wrap')) attachMenuOpen = false;
        document.removeEventListener('click', close, true);
      };
      setTimeout(() => document.addEventListener('click', close, true), 0);
    }
  }

  async function handlePhotoSelect(e) {
    const file = e.target.files?.[0];
    if (!file) return;
    e.target.value = '';

    if (!sessionPin) {
      toasts.error('Enter your PIN before sending a photo');
      return;
    }
    if (file.size > 15 * 1024 * 1024) {
      toasts.error('Photo too large to process — max 15 MB');
      return;
    }

    photoSending = true;
    try {
      const dataUrl = await compressImage(file);
      const payload = `[photo:${dataUrl}]`;
      const { ciphertext, iv, salt } = await encryptMessage(payload, sessionPin);
      socket.emit('sendSecretMsg', { receiverId: peerId, ciphertext, iv, salt });
    } catch (err) {
      if (err && err.tooLarge) {
        toasts.error(`Photo still too large after maximum compression (${err.sizeKB} KB). Use a screenshot or a simpler image.`);
      } else {
        toasts.error('Could not send photo');
      }
    } finally {
      photoSending = false;
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
  }

  function handleComposeKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); }
  }

  function scrollToBottom() {
    if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
  }

  let userScrolledUp = false;
  let unreadWhileScrolledUp = 0;

  function handleMessagesScroll() {
    if (!messagesEl) return;
    const { scrollTop, scrollHeight, clientHeight } = messagesEl;
    userScrolledUp = scrollHeight - scrollTop - clientHeight > 60;
    if (!userScrolledUp) unreadWhileScrolledUp = 0;
  }

  function jumpToBottom() {
    scrollToBottom();
    userScrolledUp = false;
    unreadWhileScrolledUp = 0;
  }

  let _prevSortedLength = 0;
  afterUpdate(() => {
    if (gateOpen && sortedMsgs.length > _prevSortedLength) {
      const newCount = sortedMsgs.length - _prevSortedLength;
      _prevSortedLength = sortedMsgs.length;
      if (userScrolledUp) {
        unreadWhileScrolledUp += newCount;
      } else {
        scrollToBottom();
      }
    }
  });

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
    if (locked.has(msg.id)) return { isOwn: false, plain: null };
    const plain = decryptedMsgs.get(msg.id);
    return { isOwn: false, plain: plain ?? null };
  }

  $: peerFirst = gateOpen ? (peerName || 'Them').split(' ')[0] : '••••••';

  const GIF_RE = /^\[gif:(https?:\/\/[^\]]+)\]$/;
  function parseGif(text) {
    const m = GIF_RE.exec(text);
    return m ? m[1] : null;
  }

  const PHOTO_RE = /^\[photo:(data:image\/[^;]+;base64,[^\]]+)\]$/;
  function parsePhoto(text) {
    const m = PHOTO_RE.exec(text);
    return m ? m[1] : null;
  }

  /**
   * Detect if a message is a photo type from ciphertext prefix analysis.
   * Since we can't decrypt locked messages, we use a heuristic: photo payloads
   * produce longer ciphertext due to base64 image data (typically >5000 chars).
   * This is imprecise but purely cosmetic — it drives the locked photo placeholder.
   */
  function isLikelyPhoto(ciphertext) {
    return ciphertext && ciphertext.length > 5000;
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

  /**
   * Return a short excerpt of the actual base64 ciphertext for visual "gibberish" display.
   * Uses the real ciphertext characters — reproducible, not random.
   * Length varies by position in string to create visual rhythm: 20–36 chars.
   * Ends with ellipsis to signal truncation.
   */
  function ciphertextGibberish(ciphertext) {
    if (!ciphertext) return '···';
    // Offset into the ciphertext by 4 chars to skip the leading AAAA PBKDF2 prefix
    // that's present on all AES-GCM outputs — makes the displayed chars more varied
    const start = Math.min(4, ciphertext.length);
    const raw = ciphertext.slice(start, start + 30);
    return raw.length >= 6 ? raw + '…' : '···';
  }
</script>

{#if panicMode}
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
  <div class="scp-panic" on:click={restoreFromPanic}></div>
{/if}

<div class="scp-backdrop" transition:fade={{ duration: 180 }} on:click|self={onClose}>
<div class="scp" bind:this={scpEl}>
  <!-- Mobile drag handle -->
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
        {:else if peerLastSeenAt}
          <span class="scp-dot"></span><span class="scp-subtext">Last seen {formatLastSeen(peerLastSeenAt)}</span>
        {:else}
          <span class="scp-dot" style="background:rgba(255,255,255,0.12)"></span><span class="scp-subtext">Offline</span>
        {/if}
      </span>
    </div>

    {#if gateOpen}
      <button
        class="scp-invite-pill"
        class:scp-invite-pill--copied={copyDone}
        on:click={shareLink}
        aria-label={copyDone ? 'Link copied' : 'Copy invite link'}
      >
        {#if copyDone}
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
          <span>Copied!</span>
        {:else}
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
          <span>Invite</span>
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
          Your PIN encrypts messages you send.<br>
          Enter the sender's PIN to read received messages.
        </p>
      </div>

      <input
        bind:this={pinInputEl}
        class="scp-pin-field"
        class:scp-pin-field--shake={pinShake}
        type="password"
        inputmode="numeric"
        pattern="\d*"
        maxlength="8"
        placeholder="PIN"
        autocomplete="one-time-code"
        autocorrect="off"
        autocapitalize="none"
        on:input={handlePinInput}
        on:keydown={handlePinKeydown}
        aria-label="Enter PIN — minimum 4 digits"
        aria-describedby={gateError ? 'scp-pin-err' : undefined}
      />

      {#if gateError}
        <p class="scp-gate-error" id="scp-pin-err" role="alert">{gateError}</p>
      {/if}

      <button
        class="scp-primary-btn"
        on:click={submitGate}
        disabled={gateUnlocking || pinDigits.length < 4}
      >
        {gateUnlocking ? '…' : 'Open Chat'}
      </button>
    </div>

  {:else}
    <!-- ── Messages ──────────────────────────────────────────── -->
    <div class="scp-msgs" bind:this={messagesEl} on:scroll={handleMessagesScroll} role="log" aria-live="polite" aria-label="Secret chat messages">
      {#if loadingMessages}
        <div class="scp-loading-shimmer" aria-label="Loading messages" aria-busy="true">
          <div class="scp-shimmer-row scp-shimmer-row--their"><div class="scp-shimmer-bubble"></div></div>
          <div class="scp-shimmer-row scp-shimmer-row--own"><div class="scp-shimmer-bubble scp-shimmer-bubble--short"></div></div>
          <div class="scp-shimmer-row scp-shimmer-row--their"><div class="scp-shimmer-bubble scp-shimmer-bubble--long"></div></div>
        </div>
      {:else if groupedMsgs.length === 0}
        <div class="scp-empty">
          <div class="scp-empty-ring" aria-hidden="true">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
            </svg>
          </div>
          <p>No messages yet</p>
          <span>Messages are end-to-end encrypted</span>
        </div>
      {/if}

      {#each groupedMsgs as msg, i (msg.id)}
        {@const d = getMsgDisplay(msg, lockedSet, chat.decryptedMessages)}
        {@const unread = isUnread(msg)}
        {@const decrypted = !d.isOwn && d.plain !== null}
        {@const showInline = activeDecryptId === msg.id}
        {@const label = dateLabel(msg.createdAt, i > 0 ? groupedMsgs[i-1].createdAt : null)}
        {@const likelyPhoto = !d.isOwn && !decrypted && isLikelyPhoto(msg.ciphertext)}

        {#if label}
          <div class="scp-date-div"><span>{label}</span></div>
        {/if}

        <div
          class="scp-msg"
          class:scp-msg--own={d.isOwn}
          class:scp-msg--their={!d.isOwn}
          class:scp-msg--group-cont={!msg.groupLast}
        >

          {#if d.isOwn}
            <!-- Sent: show actual ciphertext excerpt as visual gibberish — confirms encryption is active -->
            <div
              class="scp-bubble scp-bubble--own"
              class:scp-bubble--grp-notfirst={!msg.groupFirst}
              aria-label="Encrypted message sent"
              title="End-to-end encrypted — only your recipient can decrypt"
            >
              <span class="scp-cipher-text" aria-hidden="true">{ciphertextGibberish(msg.ciphertext)}</span>
              <span class="scp-lock-icon" aria-hidden="true">
                <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
              </span>
            </div>
            {#if msg.groupLast}
              <div class="scp-meta scp-meta--own">
                <time class="scp-time" datetime={msg.createdAt}>{clockTime(msg.createdAt)}</time>
                <span class="scp-tick" class:scp-tick--seen={msg.seenAt} title={msg.seenAt ? `Read ${clockTime(msg.seenAt)}` : 'Sent'}>
                  {#if msg.seenAt}
                    <svg width="14" height="8" viewBox="0 0 18 10" fill="none" stroke="#53bdeb" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-label="Read"><polyline points="1 5 5 9 13 1"/><polyline points="7 9 15 1"/></svg>
                  {:else}
                    <svg width="10" height="8" viewBox="0 0 12 10" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-label="Sent"><polyline points="1 5 5 9 11 1"/></svg>
                  {/if}
                </span>
              </div>
            {/if}

            <div class="scp-msg-actions">
              <button
                class="scp-delete-btn"
                class:scp-delete-btn--confirm={deletingMsgId === msg.id}
                on:click={() => deleteMsg(msg.id)}
                aria-label={deletingMsgId === msg.id ? 'Tap again to confirm delete' : 'Delete message'}
                title={deletingMsgId === msg.id ? 'Tap again to delete' : 'Delete'}
                type="button"
              >
                {#if deletingMsgId === msg.id}
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
                  Delete?
                {:else}
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>
                {/if}
              </button>
            </div>

          {:else if decrypted}
            <!-- Received + decrypted -->
            <div
              class="scp-bubble scp-bubble--their scp-bubble--decrypted"
              class:scp-bubble--grp-notfirst={!msg.groupFirst}
              class:scp-bubble--photo={parsePhoto(d.plain) !== null}
            >
              {#if parsePhoto(d.plain)}
                <img
                  src={parsePhoto(d.plain)}
                  class="msg-photo"
                  alt="Encrypted photo"
                  loading="lazy"
                />
              {:else if parseGif(d.plain)}
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
            {#if msg.groupLast}
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
            {/if}

          {:else}
            <!-- Received + locked -->
            <button
              class="scp-bubble scp-bubble--locked"
              class:scp-bubble--locked-active={showInline}
              class:scp-bubble--grp-notfirst={!msg.groupFirst}
              class:scp-bubble--locked-photo={likelyPhoto}
              on:click={() => toggleInline(msg.id)}
              aria-expanded={showInline}
              aria-label={likelyPhoto ? 'Tap to enter PIN and decrypt photo' : 'Tap to enter PIN and decrypt'}
            >
              {#if unread}
                <span class="scp-pulse" aria-hidden="true"></span>
              {/if}
              {#if likelyPhoto}
                <!-- Photo placeholder: blurred noise frame + camera icon -->
                <span class="scp-photo-placeholder" aria-hidden="true">
                  <span class="scp-photo-blur-layer"></span>
                  <span class="scp-photo-cam-icon">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                      <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
                      <circle cx="12" cy="13" r="4"/>
                    </svg>
                  </span>
                </span>
                <span class="scp-locked-photo-label">
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                  Photo — tap to unlock
                </span>
              {:else}
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                <span class="scp-cipher-preview" aria-hidden="true">{ciphertextGibberish(msg.ciphertext)}</span>
                {#if unread}
                  <span class="scp-ago">{timeAgo(msg.createdAt)}</span>
                {/if}
              {/if}
            </button>

            {#if showInline}
              <div class="scp-inline-decrypt" transition:fade={{ duration: 100 }}>
                <label class="scp-sr" for="scp-inline-{msg.id}">PIN to decrypt</label>
                <input
                  id="scp-inline-{msg.id}"
                  class="scp-inline-pin"
                  type="password"
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

            {#if msg.groupLast}
              <div class="scp-meta scp-meta--their">
                {#if unread}
                  <span class="scp-unread-dot" aria-label="Unread"></span>
                  <time class="scp-time scp-time--unread" datetime={msg.createdAt}>{timeAgo(msg.createdAt)}</time>
                {:else}
                  <time class="scp-time" datetime={msg.createdAt}>{clockTime(msg.createdAt)}</time>
                {/if}
              </div>
            {/if}
          {/if}
        </div>
      {/each}
    </div>

    <!-- Scroll-to-bottom FAB -->
    {#if userScrolledUp}
      <button
        class="scp-scroll-fab"
        on:click={jumpToBottom}
        aria-label="Jump to latest messages"
        type="button"
        transition:fade={{ duration: 120 }}
      >
        {#if unreadWhileScrolledUp > 0}
          <span class="scp-scroll-fab-badge">{unreadWhileScrolledUp}</span>
        {/if}
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <polyline points="6 9 12 15 18 9"/>
        </svg>
      </button>
    {/if}

    <!-- ── Compose ─────────────────────────────────────────── -->
    <div class="scp-compose">
      <div class="scp-compose-inner">
        <!-- Panic / blank button -->
        <button
          class="scp-compose-icon-btn scp-compose-icon-btn--panic"
          on:click={() => { panicMode = true; document.activeElement?.blur(); }}
          aria-label="Blank screen"
          title="Blank screen (tap to restore)"
          type="button"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
            <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
            <line x1="1" y1="1" x2="23" y2="23"/>
          </svg>
        </button>

        <button
          class="scp-compose-icon-btn"
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

        <!-- Sticker / animated GIF button -->
        <button
          class="scp-compose-icon-btn"
          bind:this={stickerAnchor}
          on:click={() => { stickerOpen = !stickerOpen; emojiOpen = false; }}
          aria-label="Animated sticker picker"
          aria-expanded={stickerOpen}
          title="Animated stickers"
          type="button"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
          </svg>
        </button>

        <!-- Attachment menu (camera + gallery) -->
        <input bind:this={photoInputEl} type="file" accept="image/*" style="display:none" on:change={handlePhotoSelect} aria-hidden="true" tabindex="-1" />
        <input bind:this={cameraInputEl} type="file" accept="image/*" capture="environment" style="display:none" on:change={handlePhotoSelect} aria-hidden="true" tabindex="-1" />
        <div class="scp-attach-wrap">
          <button
            class="scp-compose-icon-btn"
            class:scp-compose-icon-btn--loading={photoSending}
            class:scp-compose-icon-btn--attach-active={attachMenuOpen}
            on:click={toggleAttachMenu}
            aria-label="Send photo (encrypted)"
            aria-expanded={attachMenuOpen}
            title="Send encrypted photo"
            type="button"
            disabled={photoSending || !sessionPin}
          >
            {#if photoSending}
              <div class="scp-mini-spinner" aria-hidden="true"></div>
            {:else}
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
                <circle cx="12" cy="13" r="4"/>
              </svg>
            {/if}
          </button>
          {#if attachMenuOpen}
            <div class="scp-attach-menu" role="menu" aria-label="Attach photo">
              <button class="scp-attach-item" type="button" role="menuitem" on:click={() => { attachMenuOpen = false; cameraInputEl?.click(); }}>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
                  <circle cx="12" cy="13" r="4"/>
                </svg>
                Camera
              </button>
              <button class="scp-attach-item" type="button" role="menuitem" on:click={() => { attachMenuOpen = false; photoInputEl?.click(); }}>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                  <circle cx="8.5" cy="8.5" r="1.5"/>
                  <polyline points="21 15 16 10 5 21"/>
                </svg>
                Gallery
              </button>
            </div>
          {/if}
        </div>

        <label class="scp-sr" for="scp-compose-text">Secret message</label>
        <textarea
          id="scp-compose-text"
          class="scp-compose-text"
          rows="1"
          maxlength="2000"
          placeholder="Type a secret message…"
          bind:value={composeText}
          bind:this={composeTextEl}
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

      <div class="scp-compose-meta">
        <p class="scp-compose-hint">
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          End-to-end encrypted
        </p>
        {#if composeText.length > 1800}
          <span class="scp-char-count" class:scp-char-count--warn={composeText.length > 1950}>{2000 - composeText.length}</span>
        {/if}
      </div>
    </div>
    <!-- Pickers sit outside .scp-compose to avoid backdrop-filter containing-block bug on iOS/Safari -->
    <EmojiPicker
      open={emojiOpen}
      anchor={emojiAnchor}
      on:pick={(e) => { composeText += e.detail; emojiOpen = false; setTimeout(() => composeTextEl?.focus(), 50); }}
      on:close={() => emojiOpen = false}
    />
    <StickerPicker
      open={stickerOpen}
      anchor={stickerAnchor}
      on:pick={(e) => { composeText += e.detail; stickerOpen = false; }}
      on:close={() => stickerOpen = false}
    />
  {/if}
</div>
</div>

<style>
  /* ── Backdrop / overlay shell ──────────────────────────────── */
  .scp-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-modal, 5000);
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(6px);
    -webkit-backdrop-filter: blur(6px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 16px;
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
    background: #080810;
    border: 1px solid rgba(129,140,248,0.18);
    border-radius: 20px;
    overflow: hidden;
    width: 100%;
    max-width: 420px;
    height: min(85dvh, 640px);
    box-shadow:
      0 32px 96px rgba(0,0,0,0.85),
      0 0 0 1px rgba(129,140,248,0.12),
      inset 0 1px 0 rgba(255,255,255,0.07);
    position: relative;
  }
  /* Subtle ambient light mesh — cyberpunk atmosphere without garish colors */
  .scp::before {
    content: '';
    position: absolute;
    inset: 0;
    background:
      radial-gradient(ellipse 80% 60% at 20% 0%, rgba(99,102,241,0.14) 0%, transparent 60%),
      radial-gradient(ellipse 60% 50% at 80% 100%, rgba(168,85,247,0.11) 0%, transparent 55%),
      radial-gradient(ellipse 40% 40% at 50% 50%, rgba(59,130,246,0.06) 0%, transparent 60%);
    pointer-events: none;
    z-index: 0;
    animation: scp-mesh-shift 12s ease-in-out infinite alternate;
  }
  .scp > * { position: relative; z-index: 1; }
  @media (max-width: 767px) {
    .scp {
      max-width: 100%;
      border-radius: 20px 20px 0 0;
      height: 88dvh;
      padding-bottom: env(safe-area-inset-bottom, 0px);
      animation: scp-slide-up 0.28s cubic-bezier(0.32, 0.72, 0, 1) both;
    }
  }

  @keyframes scp-slide-up {
    from { transform: translateY(100%); }
    to   { transform: translateY(0);    }
  }

  @keyframes scp-mesh-shift {
    0%   { opacity: 1; }
    50%  { opacity: 0.65; }
    100% { opacity: 1; }
  }

  /* ── Drag handle ──────────────────────────────────────────── */
  .scp-drag-handle {
    display: none;
    width: 36px; height: 4px;
    border-radius: 2px;
    background: rgba(255,255,255,0.16);
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
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border-bottom: 1px solid rgba(129,140,248,0.1);
    flex-shrink: 0;
  }

  .scp-header-lock {
    color: rgba(255,255,255,0.18);
    display: flex; align-items: center;
    flex-shrink: 0;
    transition: color 0.3s;
  }
  .scp-header-lock--open { color: rgba(129,140,248,0.7); }

  .scp-header-info { flex: 1; min-width: 0; }

  .scp-header-name-row {
    display: flex; align-items: center; gap: 7px;
  }

  .scp-header-name {
    font-size: 14px; font-weight: 700;
    background: linear-gradient(110deg, #c7d2fe 0%, #a5b4fc 40%, #818cf8 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
    font-family: system-ui, sans-serif;
    transition: opacity 0.25s;
  }
  .scp-header-name--hidden {
    opacity: 0.3;
    letter-spacing: 0.15em;
    background: none;
    -webkit-background-clip: unset;
    -webkit-text-fill-color: rgba(255,255,255,0.3);
    background-clip: unset;
  }

  .scp-unread-badge {
    min-width: 18px; height: 18px;
    padding: 0 5px;
    border-radius: 9px;
    background: #818cf8;
    color: #fff;
    font-size: 10px; font-weight: 700;
    font-family: system-ui, sans-serif;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
  }

  .scp-header-sub {
    display: flex; align-items: center; gap: 5px;
    margin-top: 2px;
  }

  .scp-subtext {
    font-size: 11px;
    color: rgba(255,255,255,0.3);
    font-family: system-ui, sans-serif;
  }

  .scp-dot {
    width: 6px; height: 6px;
    border-radius: 50%;
    background: rgba(255,255,255,0.18);
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
    color: rgba(255,255,255,0.3);
    cursor: pointer;
    border-radius: 8px;
    transition: background 0.15s, color 0.15s;
    flex-shrink: 0;
    touch-action: manipulation;
  }
  .scp-icon-btn:hover { background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.7); }

  /* ── Invite pill ────────────────────────────────────────────── */
  .scp-invite-pill {
    display: flex; align-items: center; gap: 5px;
    padding: 0 12px 0 9px;
    height: 32px;
    min-height: 44px;
    border-radius: 20px;
    border: 1px solid rgba(129,140,248,0.3);
    background: rgba(99,102,241,0.1);
    color: rgba(165,180,252,0.95);
    font-size: 12px; font-weight: 600;
    font-family: system-ui, sans-serif;
    cursor: pointer;
    flex-shrink: 0;
    white-space: nowrap;
    transition: background 0.15s, border-color 0.15s, color 0.15s, box-shadow 0.15s;
    touch-action: manipulation;
    position: relative;
    overflow: hidden;
  }
  .scp-invite-pill::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(90deg, transparent 0%, rgba(165,180,252,0.12) 50%, transparent 100%);
    transform: translateX(-100%);
    transition: transform 0.4s ease;
    pointer-events: none;
  }
  .scp-invite-pill:hover::after { transform: translateX(100%); }
  .scp-invite-pill:hover {
    background: rgba(99,102,241,0.18);
    border-color: rgba(129,140,248,0.5);
    box-shadow: 0 0 16px rgba(99,102,241,0.2);
  }
  .scp-invite-pill--copied {
    border-color: rgba(74,222,128,0.4);
    background: rgba(74,222,128,0.1);
    color: rgba(74,222,128,0.95);
  }

  /* ── PIN field ──────────────────────────────────────────────── */
  .scp-pin-field {
    width: 100%; max-width: 240px;
    padding: 14px 18px;
    background: rgba(255,255,255,0.05);
    border: 1px solid rgba(129,140,248,0.25);
    border-radius: 13px;
    color: #e2e8f0;
    font-size: 22px;
    letter-spacing: 0.3em;
    text-align: center;
    font-family: system-ui, sans-serif;
    outline: none;
    transition: border-color 0.15s, box-shadow 0.15s;
    caret-color: #818cf8;
    -webkit-appearance: none;
    appearance: none;
  }
  .scp-pin-field::placeholder { color: rgba(255,255,255,0.2); letter-spacing: 0.05em; font-size: 15px; }
  .scp-pin-field:focus {
    border-color: rgba(129,140,248,0.6);
    box-shadow: 0 0 0 3px rgba(99,102,241,0.18);
  }
  @keyframes scp-shake {
    0%, 100% { transform: translateX(0); }
    15%      { transform: translateX(-7px); }
    35%      { transform: translateX(7px); }
    55%      { transform: translateX(-5px); }
    75%      { transform: translateX(4px); }
    90%      { transform: translateX(-2px); }
  }
  .scp-pin-field--shake { animation: scp-shake 0.48s cubic-bezier(.36,.07,.19,.97) both; }

  /* ── Gate ──────────────────────────────────────────────────── */
  .scp-gate {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 20px;
    padding: 24px 28px 32px;
    text-align: center;
    background: radial-gradient(ellipse at 50% 25%, rgba(99,102,241,0.1) 0%, transparent 65%);
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    overscroll-behavior: contain;
    position: relative;
  }
  .scp-gate::before {
    content: '';
    position: absolute;
    width: 180px; height: 180px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(99,102,241,0.12) 0%, transparent 70%);
    top: -40px; left: -30px;
    animation: scp-orb-float 8s ease-in-out infinite;
    pointer-events: none;
  }
  .scp-gate::after {
    content: '';
    position: absolute;
    width: 140px; height: 140px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(168,85,247,0.1) 0%, transparent 70%);
    bottom: 10px; right: -20px;
    animation: scp-orb-float 11s ease-in-out infinite reverse;
    pointer-events: none;
  }
  @keyframes scp-orb-float {
    0%, 100% { transform: translateY(0) scale(1); }
    50%       { transform: translateY(-18px) scale(1.07); }
  }

  .scp-gate-ring {
    width: 72px; height: 72px;
    border-radius: 50%;
    background: rgba(99,102,241,0.1);
    border: 1px solid rgba(129,140,248,0.25);
    box-shadow: 0 0 0 8px rgba(99,102,241,0.05), 0 0 32px rgba(99,102,241,0.15);
    display: flex; align-items: center; justify-content: center;
    color: rgba(129,140,248,0.85);
    position: relative; z-index: 1;
    animation: scp-ring-breathe 4s ease-in-out infinite;
  }
  @keyframes scp-ring-breathe {
    0%, 100% { box-shadow: 0 0 0 8px rgba(99,102,241,0.05), 0 0 32px rgba(99,102,241,0.15); }
    50%       { box-shadow: 0 0 0 14px rgba(99,102,241,0.03), 0 0 48px rgba(99,102,241,0.22); }
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
    color: rgba(255,255,255,0.32);
    line-height: 1.65;
    font-family: system-ui, sans-serif;
  }

  .scp-gate-error {
    margin: 0;
    font-size: 12px;
    color: #f87171;
    font-family: system-ui, sans-serif;
  }

  .scp-primary-btn {
    width: 100%; max-width: 240px;
    padding: 13px;
    border-radius: 13px;
    border: none;
    background: linear-gradient(135deg, #818cf8 0%, #6366f1 60%, #7c3aed 100%);
    color: #fff;
    font-size: 14px; font-weight: 600;
    cursor: pointer;
    font-family: system-ui, sans-serif;
    min-height: 48px;
    transition: transform 0.12s, box-shadow 0.15s, opacity 0.15s;
    touch-action: manipulation;
    box-shadow: 0 4px 20px rgba(99,102,241,0.35);
    position: relative; z-index: 1;
  }
  .scp-primary-btn:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 6px 28px rgba(99,102,241,0.5);
  }
  .scp-primary-btn:active:not(:disabled) {
    transform: scale(0.97);
    box-shadow: 0 2px 12px rgba(99,102,241,0.3);
  }
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
    gap: 12px;
    padding: 40px 0;
    text-align: center;
  }

  .scp-empty-ring {
    width: 52px; height: 52px;
    border-radius: 50%;
    background: rgba(129,140,248,0.07);
    border: 1px solid rgba(129,140,248,0.14);
    display: flex; align-items: center; justify-content: center;
    color: rgba(129,140,248,0.45);
  }

  .scp-empty p { margin: 0; font-size: 14px; color: rgba(255,255,255,0.28); font-family: system-ui, sans-serif; }
  .scp-empty span { font-size: 12px; color: rgba(255,255,255,0.18); font-family: system-ui, sans-serif; }

  .scp-msg {
    display: flex; flex-direction: column;
    max-width: 78%;
    gap: 2px;
    margin-bottom: 4px;
  }
  .scp-msg--group-cont { margin-bottom: 1px; }
  .scp-msg--own { align-self: flex-end; align-items: flex-end; }
  .scp-msg--their { align-self: flex-start; align-items: flex-start; }

  /* ── Bubbles ─────────────────────────────────────────────────── */
  .scp-bubble {
    padding: 9px 13px;
    border-radius: 18px;
    font-size: 14px;
    line-height: 1.55;
    font-family: system-ui, sans-serif;
  }

  /* Sent bubble: dark indigo glass, ciphertext rendered as visible noise */
  .scp-bubble--own {
    display: flex; align-items: center; gap: 8px;
    background: linear-gradient(135deg, rgba(99,102,241,0.22) 0%, rgba(129,140,248,0.14) 100%);
    border: 1px solid rgba(129,140,248,0.28);
    border-bottom-right-radius: 5px;
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    box-shadow:
      inset 0 1px 0 rgba(255,255,255,0.08),
      0 2px 8px rgba(0,0,0,0.2),
      0 0 0 1px rgba(99,102,241,0.08);
    max-width: 100%;
    overflow: hidden;
  }
  .scp-bubble--own.scp-bubble--grp-notfirst {
    border-top-right-radius: 5px;
  }

  /* The ciphertext gibberish — monospace, truncated, muted violet */
  .scp-cipher-text {
    font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace;
    font-size: 11px;
    letter-spacing: 0.05em;
    color: rgba(165,180,252,0.5);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
    user-select: none;
  }

  /* Ciphertext preview inside locked received bubble */
  .scp-cipher-preview {
    font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace;
    font-size: 10px;
    letter-spacing: 0.04em;
    color: rgba(165,180,252,0.35);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
    user-select: none;
  }

  /* Small lock icon pinned at the right of sent bubbles */
  .scp-lock-icon {
    color: rgba(129,140,248,0.4);
    flex-shrink: 0;
    display: flex;
    align-items: center;
  }

  .scp-bubble--their {
    background: rgba(255,255,255,0.055);
    border: 1px solid rgba(255,255,255,0.08);
    border-bottom-left-radius: 5px;
    color: #e2e8f0;
    word-break: break-word;
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
    box-shadow: inset 0 1px 0 rgba(255,255,255,0.05), 0 1px 6px rgba(0,0,0,0.12);
  }
  .scp-bubble--their.scp-bubble--grp-notfirst,
  .scp-bubble--locked.scp-bubble--grp-notfirst {
    border-top-left-radius: 5px;
  }

  .scp-bubble--decrypted {
    position: relative;
    padding-right: 30px;
  }

  /* Photo bubble: remove extra padding, let image fill edge-to-edge */
  .scp-bubble--decrypted.scp-bubble--photo {
    padding: 4px;
    overflow: hidden;
  }

  .scp-relock-btn {
    position: absolute;
    top: 4px; right: 4px;
    width: 22px; height: 22px;
    padding: 11px; margin: -11px;
    display: flex; align-items: center; justify-content: center;
    background: rgba(0,0,0,0.35);
    border: none; cursor: pointer;
    color: rgba(255,255,255,0.45);
    border-radius: 8px;
    transition: color 0.15s, background 0.15s;
    touch-action: manipulation;
    box-sizing: content-box;
  }
  .scp-relock-btn:hover { color: rgba(129,140,248,0.9); background: rgba(129,140,248,0.18); }

  .scp-countdown {
    font-size: 10px;
    color: rgba(129,140,248,0.55);
    font-family: system-ui, sans-serif;
    font-variant-numeric: tabular-nums;
  }

  /* ── Locked bubble ───────────────────────────────────────────── */
  .scp-bubble--locked {
    display: flex; align-items: center; gap: 8px;
    background: rgba(255,255,255,0.03);
    border: 1px dashed rgba(255,255,255,0.1);
    border-bottom-left-radius: 4px;
    color: rgba(255,255,255,0.28);
    font-size: 13px;
    cursor: pointer;
    min-height: 44px;
    transition: background 0.15s, border-color 0.15s;
    position: relative;
    touch-action: manipulation;
    overflow: hidden;
  }
  .scp-bubble--locked:hover { background: rgba(255,255,255,0.06); border-color: rgba(255,255,255,0.18); }
  .scp-bubble--locked-active { border-color: rgba(129,140,248,0.35); background: rgba(129,140,248,0.06); }

  /* Photo locked state — larger bubble with blurred placeholder */
  .scp-bubble--locked-photo {
    flex-direction: column;
    align-items: flex-start;
    padding: 0;
    border-style: solid;
    min-height: 80px;
    overflow: hidden;
    border-radius: 12px;
    border-bottom-left-radius: 4px;
  }

  /* Blurred noise layer simulating a censored photo */
  .scp-photo-placeholder {
    position: relative;
    width: 180px;
    height: 120px;
    display: block;
    overflow: hidden;
    flex-shrink: 0;
  }

  .scp-photo-blur-layer {
    position: absolute;
    inset: 0;
    /* Pseudo-random gradient noise pattern — suggests a blurred image */
    background:
      radial-gradient(ellipse 60% 50% at 30% 40%, rgba(99,102,241,0.18) 0%, transparent 60%),
      radial-gradient(ellipse 40% 55% at 70% 60%, rgba(168,85,247,0.12) 0%, transparent 55%),
      radial-gradient(ellipse 80% 40% at 50% 50%, rgba(59,130,246,0.1) 0%, transparent 70%),
      linear-gradient(135deg, rgba(255,255,255,0.04) 0%, rgba(0,0,0,0.25) 100%);
    filter: blur(8px);
  }

  .scp-photo-cam-icon {
    position: absolute;
    inset: 0;
    display: flex; align-items: center; justify-content: center;
    color: rgba(255,255,255,0.55);
  }

  .scp-locked-photo-label {
    display: flex; align-items: center; gap: 6px;
    padding: 7px 12px;
    font-size: 11px;
    color: rgba(255,255,255,0.35);
    font-family: system-ui, sans-serif;
    border-top: 1px solid rgba(255,255,255,0.06);
    width: 100%;
    box-sizing: border-box;
    flex-shrink: 0;
  }

  .scp-ago {
    margin-left: auto;
    font-size: 10px;
    color: rgba(255,255,255,0.2);
    font-family: system-ui, sans-serif;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .scp-pulse {
    width: 7px; height: 7px;
    border-radius: 50%;
    background: #818cf8;
    box-shadow: 0 0 0 0 rgba(129,140,248,0.6);
    animation: scp-glow 1.6s ease-in-out infinite;
    flex-shrink: 0;
  }

  .scp-body { margin: 0; }

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

  /* ── Photo message ───────────────────────────────────────────── */
  :global(.msg-photo) {
    max-width: 220px;
    max-height: 260px;
    border-radius: 10px;
    display: block;
    object-fit: cover;
  }

  /* ── Sticker / GIF in chat ───────────────────────────────────── */
  :global(.msg-sticker) {
    max-width: 120px; max-height: 120px;
    border-radius: 8px;
    display: block;
  }

  /* ── Inline decrypt ────────────────────────────────────────── */
  .scp-inline-decrypt {
    display: flex; align-items: center; gap: 8px;
    flex-wrap: nowrap;
    margin-top: 4px;
    padding: 8px 10px;
    background: rgba(129,140,248,0.06);
    border: 1px solid rgba(129,140,248,0.14);
    border-radius: 12px;
    width: 100%; max-width: 100%;
    box-sizing: border-box;
  }

  /* P0 fix: type="password" with inputmode="numeric" — consistent with gate PIN,
     prevents digits appearing in plain text on keyboards */
  .scp-inline-pin {
    flex: 1; min-width: 0;
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
    min-height: 44px;
    transition: border-color 0.15s;
    touch-action: manipulation;
  }
  .scp-inline-pin:focus { border-color: rgba(129,140,248,0.5); }
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
    border-top: 1px solid rgba(129,140,248,0.08);
    display: flex; flex-direction: column;
    gap: 7px;
    flex-shrink: 0;
    background: rgba(0,0,0,0.25);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
  }

  .scp-compose-inner {
    display: flex; align-items: flex-end; gap: 6px;
  }

  /* Panic overlay */
  .scp-panic {
    position: fixed;
    inset: 0;
    background: #fff;
    z-index: 99999;
    cursor: default;
  }

  .scp-compose-icon-btn {
    width: 36px; height: 36px;
    min-width: 44px; min-height: 44px;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none;
    color: rgba(255,255,255,0.28);
    cursor: pointer;
    border-radius: 10px;
    flex-shrink: 0;
    transition: color 0.15s, background 0.15s;
    touch-action: manipulation;
  }
  .scp-compose-icon-btn:hover { color: rgba(255,255,255,0.65); background: rgba(255,255,255,0.06); }
  .scp-compose-icon-btn:disabled { opacity: 0.22; cursor: not-allowed; }

  .scp-compose-icon-btn--panic { color: rgba(255,255,255,0.16); }
  .scp-compose-icon-btn--panic:hover { color: rgba(248,113,113,0.6); background: rgba(248,113,113,0.06); }

  .scp-compose-icon-btn--loading { cursor: wait; }

  /* Mini spinner for photo upload */
  .scp-mini-spinner {
    width: 14px; height: 14px;
    border: 2px solid rgba(255,255,255,0.15);
    border-top-color: rgba(165,180,252,0.7);
    border-radius: 50%;
    animation: scp-spin 0.8s linear infinite;
  }

  .scp-compose-text {
    flex: 1;
    resize: none;
    padding: 10px 12px;
    border-radius: 14px;
    border: 1px solid rgba(255,255,255,0.09);
    background: rgba(255,255,255,0.04);
    color: #e2e8f0;
    font-size: 14px; line-height: 1.5;
    outline: none;
    font-family: system-ui, sans-serif;
    transition: border-color 0.15s;
    -webkit-appearance: none;
    field-sizing: content;
    max-height: 120px;
    overflow-y: auto;
  }
  @media (max-width: 767px) {
    .scp-compose-text { font-size: 16px; }
  }
  .scp-compose-text:focus {
    border-color: rgba(129,140,248,0.4);
    box-shadow: 0 0 0 3px rgba(99,102,241,0.1);
  }
  .scp-compose-text::placeholder { color: rgba(255,255,255,0.18); }

  .scp-send-btn {
    width: 44px; height: 44px;
    border-radius: 13px;
    border: none;
    background: linear-gradient(135deg, #818cf8 0%, #6366f1 100%);
    color: #fff;
    cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
    transition: transform 0.12s, box-shadow 0.15s, background 0.15s;
    touch-action: manipulation;
    box-shadow: 0 2px 12px rgba(99,102,241,0.4);
  }
  .scp-send-btn:hover:not(:disabled) {
    transform: scale(1.06);
    box-shadow: 0 4px 20px rgba(99,102,241,0.6);
  }
  .scp-send-btn:active:not(:disabled) {
    transform: scale(0.93);
    box-shadow: 0 1px 6px rgba(99,102,241,0.3);
  }
  .scp-send-btn:disabled { opacity: 0.28; cursor: not-allowed; box-shadow: none; }

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
    color: rgba(255,255,255,0.16);
    font-family: system-ui, sans-serif;
  }

  /* ── Attachment menu ────────────────────────────────────────── */
  .scp-attach-wrap {
    position: relative;
  }

  .scp-attach-menu {
    position: absolute;
    bottom: calc(100% + 6px);
    left: 50%;
    transform: translateX(-50%);
    background: #14141e;
    border: 1px solid rgba(255,255,255,0.1);
    border-radius: 12px;
    padding: 4px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 130px;
    box-shadow: 0 8px 24px rgba(0,0,0,0.6);
    animation: scp-pop 0.14s cubic-bezier(0.34,1.56,0.64,1) both;
    z-index: 10;
  }

  .scp-attach-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 9px 12px;
    background: none;
    border: none;
    border-radius: 8px;
    color: rgba(255,255,255,0.8);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    white-space: nowrap;
    min-height: 44px;
    font-family: system-ui, sans-serif;
    transition: background 0.12s, color 0.12s;
  }
  .scp-attach-item:hover { background: rgba(255,255,255,0.07); color: #fff; }
  .scp-attach-item:active { background: rgba(129,140,248,0.15); }

  .scp-compose-icon-btn--attach-active {
    color: #818cf8;
    background: rgba(129,140,248,0.12);
  }

  @keyframes scp-pop {
    from { opacity: 0; transform: translateX(-50%) scale(0.88) translateY(6px); }
    to   { opacity: 1; transform: translateX(-50%) scale(1)    translateY(0);   }
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

  /* ── Message slide-in animation ─────────────────────────────── */
  @keyframes scp-msg-in {
    from {
      opacity: 0;
      transform: perspective(500px) rotateX(9deg) translateY(10px) scale(0.97);
    }
    to {
      opacity: 1;
      transform: perspective(500px) rotateX(0deg) translateY(0) scale(1);
    }
  }
  .scp-msg {
    animation: scp-msg-in 0.22s cubic-bezier(0.2, 0.8, 0.3, 1) both;
    transform-origin: bottom center;
  }

  /* ── Date divider ───────────────────────────────────────────── */
  .scp-date-div {
    display: flex;
    align-items: center;
    gap: 10px;
    margin: 10px 0 6px;
    align-self: stretch;
    animation: scp-msg-in 0.18s ease-out both;
  }
  .scp-date-div::before,
  .scp-date-div::after {
    content: '';
    flex: 1;
    height: 1px;
    background: rgba(255,255,255,0.06);
  }
  .scp-date-div span {
    font-size: 10px;
    color: rgba(255,255,255,0.22);
    white-space: nowrap;
    font-family: system-ui, sans-serif;
    letter-spacing: 0.03em;
    padding: 2px 4px;
  }

  /* ── Delete button + actions row ─────────────────────────────── */
  .scp-msg-actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 2px;
    opacity: 0;
    transition: opacity 0.15s;
    min-height: 22px;
  }
  .scp-msg--own:hover .scp-msg-actions,
  .scp-msg--own:focus-within .scp-msg-actions {
    opacity: 1;
  }
  @media (hover: none) {
    .scp-msg-actions { opacity: 0.4; }
  }

  .scp-delete-btn {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 3px 8px;
    border-radius: 8px;
    border: none;
    background: transparent;
    color: rgba(255,255,255,0.2);
    font-size: 10px;
    font-family: system-ui, sans-serif;
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
    touch-action: manipulation;
    min-height: 28px;
  }
  .scp-delete-btn:hover {
    color: rgba(248,113,113,0.8);
    background: rgba(248,113,113,0.08);
  }
  .scp-delete-btn--confirm {
    color: #f87171;
    background: rgba(248,113,113,0.12);
    border-radius: 8px;
  }

  /* ── Compose meta row ───────────────────────────────────────── */
  .scp-compose-meta {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .scp-char-count {
    font-size: 10px;
    color: rgba(255,255,255,0.28);
    font-family: system-ui, sans-serif;
    font-variant-numeric: tabular-nums;
    transition: color 0.2s;
  }
  .scp-char-count--warn { color: rgba(248,113,113,0.75); }

  /* ── Scroll-to-bottom FAB ───────────────────────────────────── */
  .scp-scroll-fab {
    align-self: flex-end;
    margin: -8px 12px 0;
    position: relative;
    width: 36px; height: 36px;
    border-radius: 50%;
    border: 1px solid rgba(129,140,248,0.35);
    background: rgba(8,8,16,0.88);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    color: rgba(165,180,252,0.9);
    box-shadow: 0 4px 16px rgba(0,0,0,0.5), 0 0 12px rgba(99,102,241,0.12);
    display: flex; align-items: center; justify-content: center;
    cursor: pointer;
    touch-action: manipulation;
    transition: background 0.15s, transform 0.15s, box-shadow 0.15s;
    z-index: 10;
    flex-shrink: 0;
  }
  .scp-scroll-fab:hover {
    background: rgba(99,102,241,0.15);
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(0,0,0,0.5), 0 0 20px rgba(99,102,241,0.2);
  }
  .scp-scroll-fab-badge {
    position: absolute;
    top: -5px; right: -5px;
    min-width: 18px; height: 18px;
    padding: 0 4px;
    border-radius: 9px;
    background: #818cf8;
    color: #fff;
    font-size: 10px; font-weight: 700;
    display: flex; align-items: center; justify-content: center;
    font-family: system-ui, sans-serif;
  }

  /* ── Loading shimmer ────────────────────────────────────────── */
  @keyframes scp-shimmer {
    0% { background-position: -200% 0; }
    100% { background-position: 200% 0; }
  }
  .scp-loading-shimmer {
    display: flex; flex-direction: column; gap: 12px;
    padding: 8px 0;
    flex: 1;
  }
  .scp-shimmer-row { display: flex; }
  .scp-shimmer-row--own { justify-content: flex-end; }
  .scp-shimmer-row--their { justify-content: flex-start; }
  .scp-shimmer-bubble {
    height: 36px; width: 52%;
    border-radius: 16px;
    background: linear-gradient(90deg, rgba(255,255,255,0.03) 25%, rgba(255,255,255,0.07) 50%, rgba(255,255,255,0.03) 75%);
    background-size: 200% 100%;
    animation: scp-shimmer 1.4s ease infinite;
  }
  .scp-shimmer-bubble--short { width: 36%; }
  .scp-shimmer-bubble--long { width: 65%; }

  @media (prefers-reduced-motion: reduce) {
    /* Disable all animations — critical for users with vestibular disorders */
    .scp::before { animation: none; }
    .scp-shimmer-bubble { animation: none; background: rgba(255,255,255,0.04); }
    .scp-msg { animation: none; }
    .scp-date-div { animation: none; }
    .scp-gate::before, .scp-gate::after { animation: none; }
    .scp-gate-ring { animation: none; }
    .scp-dot--online { animation: none; }
    .scp-pulse { animation: none; }
    .scp-unread-dot { animation: none; }
    .scp-attach-menu { animation: none; }
  }
</style>
