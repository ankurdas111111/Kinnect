<script>
  /**
   * SecretChatPanel — orchestrates the secret chat experience.
   * Split into sub-components: SecretChatGate, SecretChatMessage.
   * All encryption/socket logic lives here; UI delegates to sub-components.
   */
  import { onMount, onDestroy, tick, afterUpdate } from 'svelte';
  import { fade } from 'svelte/transition';
  import { socket, markSecretMsgSeen, createSecretChatInvite } from '../lib/socket.js';
  import { authUser } from '../lib/stores/auth.js';
  import { secretChats, lockSecretChat, storeDecrypted, secretChatPresence } from '../lib/stores/secretChat.js';
  import { otherUsers } from '../lib/stores/map.js';
  import { encryptMessage, decryptMessage } from '../lib/crypto.js';
  import { toasts } from '../lib/stores/toast.js';
  import { haptics } from '../lib/haptics.js';
  import EmojiPicker from './primitives/EmojiPicker.svelte';
  import StickerPicker from './primitives/StickerPicker.svelte';
  import SecretChatGate from './SecretChatGate.svelte';
  import SecretChatMessage from './SecretChatMessage.svelte';

  export let peerId;
  export let peerName = 'Contact';
  export let onClose = () => {};

  // Session PIN
  let sessionPin = '';
  let gateOpen = false;
  let gateError = '';
  let gateUnlocking = false;
  let gateRef;     // bind:this on SecretChatGate for imperative shake/success

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

  // Auto-lock progress bar — drains top of chat
  let autoLockProgress = 100;    // 0–100%
  let autoLockTimerId = null;
  let autoLockTotalSecs = 120;   // 2 min inactivity auto-lock
  let autoLockRemaining = autoLockTotalSecs;

  function resetAutoLockBar() {
    autoLockRemaining = autoLockTotalSecs;
    autoLockProgress = 100;
  }

  function startAutoLockBar() {
    clearInterval(autoLockTimerId);
    resetAutoLockBar();
    autoLockTimerId = setInterval(() => {
      autoLockRemaining--;
      autoLockProgress = (autoLockRemaining / autoLockTotalSecs) * 100;
      if (autoLockRemaining <= 0) {
        clearInterval(autoLockTimerId);
        // Auto-lock the whole session
        gateOpen = false;
        sessionPin = '';
        lockedSet = new Set();
        lockCountdowns = {};
        for (const id of Object.keys(lockIntervals)) clearInterval(lockIntervals[id]);
        lockIntervals = {};
      }
    }, 1000);
  }

  function touchAutoLock() {
    if (gateOpen) resetAutoLockBar();
  }

  function relockMsg(msgId) {
    if (lockIntervals[msgId]) { clearInterval(lockIntervals[msgId]); delete lockIntervals[msgId]; }
    delete lockCountdowns[msgId];
    lockCountdowns = { ...lockCountdowns };
    lockedSet = new Set([...lockedSet, msgId]);
    inlinePins = { ...inlinePins, [msgId]: '' };
    inlineErrors = { ...inlineErrors, [msgId]: '' };
    activeDecryptId = null;
  }

  function startAutoLockMsg(msgId) {
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

  // iOS keyboard avoidance
  let scpEl;
  let keyboardOffset = 0;

  function onVVChange() {
    if (!scpEl) return;
    const vv = window.visualViewport;
    if (!vv) return;
    const kbH = Math.max(0, window.innerHeight - vv.height - vv.offsetTop);
    keyboardOffset = kbH > 50 ? kbH : 0;
  }

  $: if (gateOpen && scpEl) { keyboardOffset = 0; }

  // Compose state
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
  let panicGlitching = false;

  // Delete confirmation — two-tap
  let deletingMsgId = null;

  // Typing indicator
  let isTyping = false;
  let _typingTimer = null;
  let peerTyping = false; // derived from presence + compose activity heuristic

  // Seen receipt pulse — tracks which msg IDs recently got seen
  let seenPulseIds = new Set();

  // Photo lightbox
  let lightboxSrc = null;
  let lightboxOpen = false;

  // Delete message handler
  function deleteMsg(msgId) {
    if (deletingMsgId === msgId) {
      socket.emit('deleteSecretMsg', { id: msgId });
      deletingMsgId = null;
      haptics.tap?.();
    } else {
      deletingMsgId = msgId;
      haptics.tap?.();
      setTimeout(() => { if (deletingMsgId === msgId) deletingMsgId = null; }, 2500);
    }
  }

  function restoreFromPanic() {
    panicMode = false;
    panicGlitching = false;
    gateOpen = false;
    sessionPin = '';
    for (const id of Object.keys(lockIntervals)) clearInterval(lockIntervals[id]);
    lockIntervals = {};
    lockCountdowns = {};
    lockedSet = new Set();
    activeDecryptId = null;
    inlinePins = {};
    inlineErrors = {};
  }

  function triggerPanic() {
    panicGlitching = true;
    haptics.heavy?.();
    setTimeout(() => {
      panicGlitching = false;
      panicMode = true;
      document.activeElement?.blur();
    }, 220);
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
  $: peerFirst = gateOpen ? (peerName || 'Them').split(' ')[0] : '••••••';

  // Heuristic: peer is "typing" if they have the chat open and last activity was <15s ago
  $: peerTyping = peerChatOpen && peerPresence && (Date.now() - (peerPresence.at ?? 0)) < 15000;

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
    // Watch for seen events to trigger pulse
    socket.on('secretMsgSeen', onSecretMsgSeen);
  });

  onDestroy(() => {
    emitPresence(false);
    lockSecretChat(peerId);
    sessionPin = '';
    gateOpen = false;
    clearInterval(autoLockTimerId);
    for (const id of Object.values(lockIntervals)) clearInterval(id);
    window.visualViewport?.removeEventListener('resize', onVVChange);
    window.visualViewport?.removeEventListener('scroll', onVVChange);
    socket.off('secretMsgSeen', onSecretMsgSeen);
  });

  function onSecretMsgSeen(data) {
    if (!data?.id) return;
    seenPulseIds = new Set([...seenPulseIds, data.id]);
    setTimeout(() => {
      seenPulseIds = new Set([...seenPulseIds].filter(id => id !== data.id));
    }, 800);
  }

  // ── Gate ──────────────────────────────────────────────────────
  async function submitGate(pin) {
    if (gateUnlocking || pin.length < 4) return;
    gateError = '';
    gateUnlocking = true;
    sessionPin = pin;
    gateOpen = true;
    gateUnlocking = false;
    gateRef?.triggerSuccess?.();
    await tick();
    scrollToBottom();
    startAutoLockBar();
    haptics.success?.();
  }

  // ── Per-message inline decrypt ─────────────────────────────────
  function toggleInline(msgId) {
    activeDecryptId = activeDecryptId === msgId ? null : msgId;
    inlinePins = { ...inlinePins, [msgId]: '' };
    inlineErrors = { ...inlineErrors, [msgId]: '' };
    touchAutoLock();
  }

  function handleInlinePinInput(e) {
    const { id, value } = e.detail;
    inlinePins = { ...inlinePins, [id]: value };
    touchAutoLock();
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
      startAutoLockMsg(msg.id);
      haptics.tap?.();
      touchAutoLock();
    } catch {
      inlineErrors = { ...inlineErrors, [msg.id]: 'Wrong PIN — try again' };
      haptics.error?.();
    } finally {
      inlineUnlocking = { ...inlineUnlocking, [msg.id]: false };
    }
  }

  // ── Photo sending ─────────────────────────────────────────────
  let photoInputEl;
  let cameraInputEl;
  let photoSending = false;
  let attachMenuOpen = false;

  const MAX_PHOTO_EDGE = 720;
  const MAX_BINARY_BYTES = 100_000;

  function compressImage(file) {
    return new Promise((resolve, reject) => {
      const img = new Image();
      const url = URL.createObjectURL(file);
      img.onload = () => {
        URL.revokeObjectURL(url);
        let { width, height } = img;
        if (width > MAX_PHOTO_EDGE || height > MAX_PHOTO_EDGE) {
          if (width >= height) { height = Math.round((height / width) * MAX_PHOTO_EDGE); width = MAX_PHOTO_EDGE; }
          else { width = Math.round((width / height) * MAX_PHOTO_EDGE); height = MAX_PHOTO_EDGE; }
        }
        function encode(cvs, q) {
          const webp = cvs.toDataURL('image/webp', q);
          return webp.startsWith('data:image/webp') ? webp : cvs.toDataURL('image/jpeg', q);
        }
        const c0 = document.createElement('canvas');
        c0.width = width; c0.height = height;
        c0.getContext('2d').drawImage(img, 0, 0, width, height);
        let result = encode(c0, 0.80);
        let w = width, h = height;
        for (let pass = 0; pass < 4 && result.length * 0.75 > MAX_BINARY_BYTES; pass++) {
          const scale = Math.sqrt(MAX_BINARY_BYTES / (result.length * 0.75)) * 0.88;
          w = Math.max(120, Math.round(w * scale)); h = Math.max(90, Math.round(h * scale));
          const c = document.createElement('canvas');
          c.width = w; c.height = h;
          c.getContext('2d').drawImage(img, 0, 0, w, h);
          result = encode(c, 0.72 - pass * 0.05);
        }
        const finalBinaryKB = Math.round((result.length * 0.75) / 1024);
        if (result.length * 0.75 > MAX_BINARY_BYTES * 1.5) { reject({ tooLarge: true, sizeKB: finalBinaryKB }); return; }
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
    if (!sessionPin) { toasts.error('Enter your PIN before sending a photo'); return; }
    if (file.size > 15 * 1024 * 1024) { toasts.error('Photo too large — max 15 MB'); return; }
    photoSending = true;
    haptics.tap?.();
    try {
      const dataUrl = await compressImage(file);
      const payload = `[photo:${dataUrl}]`;
      const { ciphertext, iv, salt } = await encryptMessage(payload, sessionPin);
      socket.emit('sendSecretMsg', { receiverId: peerId, ciphertext, iv, salt });
      haptics.confirm?.();
    } catch (err) {
      if (err && err.tooLarge) {
        toasts.error(`Photo still too large (${err.sizeKB} KB). Use a simpler image.`);
      } else {
        toasts.error('Could not send photo — check your connection');
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
    haptics.tap?.();
    touchAutoLock();
    try {
      const { ciphertext, iv, salt } = await encryptMessage(text, sessionPin);
      socket.emit('sendSecretMsg', { receiverId: peerId, ciphertext, iv, salt });
      composeText = '';
      await tick();
      scrollToBottom();
      isTyping = false;
    } catch {
      toasts.error('Failed to send — check your connection');
    } finally {
      sending = false;
    }
  }

  function handleComposeKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); }
    touchAutoLock();
    // Typing indicator heuristic
    if (!isTyping) {
      isTyping = true;
      emitPresence(true);
    }
    clearTimeout(_typingTimer);
    _typingTimer = setTimeout(() => { isTyping = false; }, 2000);
  }

  function handleComposeInput() {
    touchAutoLock();
    if (!isTyping) { isTyping = true; emitPresence(true); }
    clearTimeout(_typingTimer);
    _typingTimer = setTimeout(() => { isTyping = false; }, 2000);
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
    touchAutoLock();
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
      haptics.confirm?.();
      setTimeout(() => { copyDone = false; }, 2500);
    } catch {
      toasts.error('Could not copy invite link — try again');
    }
  }

  // ── Formatters ────────────────────────────────────────────────
  function formatLastSeen(ts) {
    if (!ts) return '';
    const ms = typeof ts === 'number' ? ts : new Date(ts).getTime();
    const diff = Date.now() - ms;
    if (diff < 60_000) return 'just now';
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
    return new Date(ms).toLocaleDateString([], { month: 'short', day: 'numeric' });
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

  // Photo lightbox
  function openLightbox(src) {
    lightboxSrc = src;
    lightboxOpen = true;
    haptics.tap?.();
  }

  function closeLightbox(e) {
    if (e && e.target !== e.currentTarget && !e.target.classList.contains('lightbox-backdrop')) return;
    lightboxOpen = false;
    lightboxSrc = null;
  }

  function handleLightboxKey(e) {
    if (e.key === 'Escape') closeLightbox();
  }
</script>

<!-- ── Panic glitch overlay ──────────────────────────────────── -->
{#if panicGlitching}
  <div class="scp-glitch" aria-hidden="true"></div>
{/if}

<!-- ── Panic blank screen ─────────────────────────────────────── -->
{#if panicMode}
  <div
    class="scp-panic"
    on:click={restoreFromPanic}
    on:keydown={(e) => (e.key === 'Enter' || e.key === ' ' || e.key === 'Escape') && restoreFromPanic()}
    role="alertdialog"
    aria-label="Screen blanked for privacy. Press Enter or tap to restore."
    aria-live="assertive"
    tabindex="0"
  ></div>
{/if}

<!-- ── Photo lightbox ─────────────────────────────────────────── -->
{#if lightboxOpen && lightboxSrc}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div
    class="lightbox-backdrop"
    on:click={closeLightbox}
    on:keydown={handleLightboxKey}
    role="dialog"
    aria-modal="true"
    aria-label="Photo viewer"
    tabindex="-1"
    transition:fade={{ duration: 180 }}
  >
    <button class="lightbox-close" on:click={closeLightbox} aria-label="Close photo" type="button">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
    </button>
    <img src={lightboxSrc} class="lightbox-img" alt="Encrypted photo" />
    <p class="lightbox-hint" aria-hidden="true">
      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
      End-to-end encrypted
    </p>
  </div>
{/if}

<!-- ── Main panel backdrop ───────────────────────────────────── -->
<div
  class="scp-backdrop"
  transition:fade={{ duration: 180 }}
  on:click|self={onClose}
  role="dialog"
  aria-modal="true"
  aria-label="Secret Chat with {peerName}"
>
<div
  class="scp"
  bind:this={scpEl}
  style="transform: translateY({keyboardOffset > 0 ? `-${keyboardOffset}px` : '0'}); transition: transform 0.2s var(--ease-out, cubic-bezier(0.16,1,0.3,1));"
  on:click={touchAutoLock}
  on:keydown={touchAutoLock}
  role="presentation"
>
  <!-- Mobile drag handle -->
  <div class="scp-drag-handle" aria-hidden="true"></div>

  <!-- ── Auto-lock progress bar ─────────────────────────────── -->
  {#if gateOpen}
    <div
      class="scp-autolock-bar"
      style="width: {autoLockProgress}%; opacity: {autoLockProgress < 30 ? 1 : 0.35}"
      aria-hidden="true"
    ></div>
  {/if}

  <!-- ── Header ─────────────────────────────────────────────── -->
  <header class="scp-header">
    <div class="scp-header-lock" class:scp-header-lock--open={gateOpen} aria-hidden="true">
      {#if gateOpen}
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="11" width="18" height="11" rx="2"/>
          <path d="M7 11V7a5 5 0 0 1 9.9-1"/>
        </svg>
      {:else}
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="11" width="18" height="11" rx="2"/>
          <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
      {/if}
    </div>

    <div class="scp-header-info">
      <div class="scp-header-name-row">
        <h2 class="scp-header-name" class:scp-header-name--hidden={!gateOpen}>
          {gateOpen ? peerName : '••••••'}
        </h2>
        {#if unreadCount > 0 && gateOpen}
          <span class="scp-unread-badge" aria-label="{unreadCount} unread">{unreadCount}</span>
        {/if}
      </div>

      <div class="scp-header-sub" aria-live="polite" aria-atomic="true">
        {#if !gateOpen}
          <span class="scp-subtext">Enter PIN to open</span>
        {:else if peerTyping}
          <span class="scp-typing-indicator" aria-label="{peerFirst} is active">
            <span class="typing-dot"></span>
            <span class="typing-dot"></span>
            <span class="typing-dot"></span>
          </span>
          <span class="scp-subtext">Active</span>
        {:else if peerChatOpen}
          <span class="scp-presence-dot scp-presence-dot--active" aria-hidden="true"></span>
          <span class="scp-subtext">In this chat</span>
        {:else if peerKinnectOnline}
          <span class="scp-presence-dot scp-presence-dot--online" aria-hidden="true"></span>
          <span class="scp-subtext">Online</span>
        {:else if peerLastSeenAt}
          <span class="scp-presence-dot" aria-hidden="true"></span>
          <span class="scp-subtext">Last seen {formatLastSeen(peerLastSeenAt)}</span>
        {:else}
          <span class="scp-presence-dot" aria-hidden="true"></span>
          <span class="scp-subtext">Offline</span>
        {/if}
      </div>
    </div>

    {#if gateOpen}
      <button
        class="scp-invite-btn"
        class:scp-invite-btn--copied={copyDone}
        on:click={shareLink}
        aria-label={copyDone ? 'Invite link copied' : 'Copy invite link'}
        type="button"
      >
        {#if copyDone}
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg>
          <span>Copied</span>
        {:else}
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
          <span>Invite</span>
        {/if}
      </button>
    {/if}

    <button class="scp-icon-btn" on:click={onClose} aria-label="Close secret chat" type="button">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
      </svg>
    </button>
  </header>

  <!-- ── Gate ───────────────────────────────────────────────── -->
  {#if !gateOpen}
    <SecretChatGate
      bind:this={gateRef}
      {peerName}
      unlocking={gateUnlocking}
      error={gateError}
      on:submit={(e) => submitGate(e.detail)}
    />

  {:else}
    <!-- ── Messages ──────────────────────────────────────────── -->
    <main
      class="scp-msgs"
      bind:this={messagesEl}
      on:scroll={handleMessagesScroll}
      role="log"
      aria-live="polite"
      aria-label="Secret chat messages with {peerFirst}"
    >
      {#if loadingMessages}
        <div class="scp-skeleton" aria-label="Loading messages" aria-busy="true" role="status">
          <div class="scp-skel-row scp-skel-row--their"><div class="scp-skel-bubble"></div></div>
          <div class="scp-skel-row scp-skel-row--own"><div class="scp-skel-bubble scp-skel-bubble--short"></div></div>
          <div class="scp-skel-row scp-skel-row--their"><div class="scp-skel-bubble scp-skel-bubble--long"></div></div>
          <div class="scp-skel-row scp-skel-row--own"><div class="scp-skel-bubble"></div></div>
        </div>

      {:else if groupedMsgs.length === 0}
        <!-- Beautiful empty state -->
        <div class="scp-empty" role="status">
          <div class="scp-empty-lock-art" aria-hidden="true">
            <div class="scp-empty-lock-ring scp-empty-lock-ring--outer"></div>
            <div class="scp-empty-lock-ring scp-empty-lock-ring--inner"></div>
            <div class="scp-empty-lock-icon">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="11" width="18" height="11" rx="2"/>
                <path d="M7 11V7a5 5 0 0 1 9.9-1"/>
              </svg>
            </div>
          </div>
          <p class="scp-empty-title">Say something only you two can read</p>
          <p class="scp-empty-sub">
            Messages between you and {peerFirst} are end-to-end encrypted.<br>
            Only you can read them. Not Kinnect, not anyone.
          </p>
          <p class="scp-empty-cta">Type a message below</p>
        </div>
      {/if}

      {#each groupedMsgs as msg, i (msg.id)}
        {@const plain = chat.decryptedMessages.get(msg.id) ?? null}
        {@const isOwn = msg.senderId === myId}
        {@const label = dateLabel(msg.createdAt, i > 0 ? groupedMsgs[i-1].createdAt : null)}

        {#if label}
          <div class="scp-date-div" role="separator" aria-label={label}>
            <span>{label}</span>
          </div>
        {/if}

        <SecretChatMessage
          {msg}
          {plain}
          {isOwn}
          showInline={activeDecryptId === msg.id}
          inlinePin={inlinePins[msg.id] ?? ''}
          inlineError={inlineErrors[msg.id] ?? ''}
          inlineUnlocking={inlineUnlocking[msg.id] ?? false}
          {lockedSet}
          lockCountdown={lockCountdowns[msg.id] ?? null}
          {deletingMsgId}
          {myId}
          {peerFirst}
          seenPulse={seenPulseIds.has(msg.id)}
          on:toggleInline={(e) => toggleInline(e.detail)}
          on:relock={(e) => relockMsg(e.detail)}
          on:delete={(e) => deleteMsg(e.detail)}
          on:decryptOne={(e) => decryptOne(e.detail)}
          on:inlinePinInput={handleInlinePinInput}
          on:photoExpand={(e) => openLightbox(e.detail)}
        />
      {/each}
    </main>

    <!-- Scroll-to-bottom FAB -->
    {#if userScrolledUp}
      <button
        class="scp-scroll-fab"
        on:click={jumpToBottom}
        aria-label="Jump to latest messages{unreadWhileScrolledUp > 0 ? ` — ${unreadWhileScrolledUp} new` : ''}"
        type="button"
        transition:fade={{ duration: 120 }}
      >
        {#if unreadWhileScrolledUp > 0}
          <span class="scp-scroll-fab-badge" aria-hidden="true">{unreadWhileScrolledUp}</span>
        {/if}
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
          <polyline points="6 9 12 15 18 9"/>
        </svg>
      </button>
    {/if}

    <!-- ── Compose ─────────────────────────────────────────── -->
    <footer class="scp-compose">
      <div class="scp-compose-inner">
        <!-- Panic button -->
        <button
          class="scp-compose-icon-btn scp-compose-icon-btn--panic"
          on:click={triggerPanic}
          aria-label="Blank screen for privacy"
          title="Blank screen"
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
          on:click={() => { emojiOpen = !emojiOpen; stickerOpen = false; haptics.tap?.(); }}
          aria-label="Open emoji picker"
          aria-expanded={emojiOpen}
          aria-haspopup="true"
          type="button"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <circle cx="12" cy="12" r="10"/>
            <path d="M8 14s1.5 2 4 2 4-2 4-2"/>
            <line x1="9" y1="9" x2="9.01" y2="9"/>
            <line x1="15" y1="9" x2="15.01" y2="9"/>
          </svg>
        </button>

        <button
          class="scp-compose-icon-btn"
          bind:this={stickerAnchor}
          on:click={() => { stickerOpen = !stickerOpen; emojiOpen = false; haptics.tap?.(); }}
          aria-label="Open sticker picker"
          aria-expanded={stickerOpen}
          aria-haspopup="true"
          type="button"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
          </svg>
        </button>

        <!-- Attachment menu -->
        <input bind:this={photoInputEl} type="file" accept="image/*" style="display:none" on:change={handlePhotoSelect} aria-hidden="true" tabindex="-1" />
        <input bind:this={cameraInputEl} type="file" accept="image/*" capture="environment" style="display:none" on:change={handlePhotoSelect} aria-hidden="true" tabindex="-1" />
        <div class="scp-attach-wrap">
          <button
            class="scp-compose-icon-btn"
            class:scp-compose-icon-btn--loading={photoSending}
            class:scp-compose-icon-btn--active={attachMenuOpen}
            on:click={toggleAttachMenu}
            aria-label="Send encrypted photo"
            aria-expanded={attachMenuOpen}
            aria-haspopup="menu"
            type="button"
            disabled={photoSending || !sessionPin}
          >
            {#if photoSending}
              <div class="scp-mini-spinner" aria-hidden="true"></div>
            {:else}
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
                <circle cx="12" cy="13" r="4"/>
              </svg>
            {/if}
          </button>
          {#if attachMenuOpen}
            <div class="scp-attach-menu" role="menu" aria-label="Photo source">
              <button class="scp-attach-item" type="button" role="menuitem" on:click={() => { attachMenuOpen = false; cameraInputEl?.click(); }}>
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/><circle cx="12" cy="13" r="4"/></svg>
                Take Photo
              </button>
              <button class="scp-attach-item" type="button" role="menuitem" on:click={() => { attachMenuOpen = false; photoInputEl?.click(); }}>
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>
                Choose from Gallery
              </button>
            </div>
          {/if}
        </div>

        <label class="scp-sr" for="scp-compose-text">Secret message to {peerFirst}</label>
        <textarea
          id="scp-compose-text"
          class="scp-compose-text"
          rows="1"
          maxlength="2000"
          placeholder="Message {peerFirst}…"
          bind:value={composeText}
          bind:this={composeTextEl}
          on:keydown={handleComposeKeydown}
          on:input={handleComposeInput}
          disabled={sending}
        ></textarea>

        <!-- Voice note placeholder -->
        <button
          class="scp-compose-icon-btn scp-compose-icon-btn--voice"
          title="Voice notes coming soon"
          aria-label="Voice notes — coming soon"
          type="button"
          disabled
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/>
            <path d="M19 10v2a7 7 0 0 1-14 0v-2"/>
            <line x1="12" y1="19" x2="12" y2="23"/>
            <line x1="8" y1="23" x2="16" y2="23"/>
          </svg>
        </button>

        <button
          class="scp-send-btn"
          class:scp-send-btn--active={composeText.trim().length > 0}
          on:click={send}
          disabled={sending || !composeText.trim()}
          aria-label="Send encrypted message"
          type="button"
        >
          {#if sending}
            <div class="scp-send-ring" aria-hidden="true"></div>
          {:else}
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
          {/if}
        </button>
      </div>

      <div class="scp-compose-meta">
        <p class="scp-compose-hint" aria-hidden="true">
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          End-to-end encrypted
        </p>
        {#if composeText.length > 1800}
          <span class="scp-char-count" class:scp-char-count--warn={composeText.length > 1950} aria-live="polite">{2000 - composeText.length} remaining</span>
        {/if}
      </div>
    </footer>

    <!-- Pickers outside compose — avoids backdrop-filter containing-block iOS bug -->
    <EmojiPicker
      open={emojiOpen}
      anchor={emojiAnchor}
      on:pick={(e) => { composeText += e.detail; emojiOpen = false; setTimeout(() => composeTextEl?.focus(), 50); }}
      on:close={() => emojiOpen = false}
    />
    <StickerPicker
      open={stickerOpen}
      anchor={stickerAnchor}
      on:pick={async (e) => {
        const tag = e.detail;
        stickerOpen = false;
        if (!sessionPin) { toasts.error('Enter your PIN before sending a sticker'); return; }
        haptics.tap?.();
        try {
          const { ciphertext, iv, salt } = await encryptMessage(tag, sessionPin);
          socket.emit('sendSecretMsg', { receiverId: peerId, ciphertext, iv, salt });
        } catch {
          toasts.error('Failed to send sticker');
        }
      }}
      on:close={() => stickerOpen = false}
    />
  {/if}
</div>
</div>

<style>
  /* ─────────────────────────────────────────────────────────────
     Chat-specific design tokens — teal system
     ───────────────────────────────────────────────────────────── */
  .scp-backdrop {
    --chat-accent:        var(--primary-500, #14b8a6);
    --chat-accent-dim:    rgba(20, 184, 166, 0.18);
    --chat-accent-subtle: rgba(20, 184, 166, 0.08);
    --chat-accent-glow:   rgba(20, 184, 166, 0.28);
    --chat-bg:            #060610;
    --chat-surface:       #0a0a18;
    --chat-elevated:      #0f0f20;
    --chat-border:        rgba(255, 255, 255, 0.07);
    --chat-border-accent: rgba(20, 184, 166, 0.22);
  }

  /* ── Panic effects ──────────────────────────────────────────── */
  .scp-glitch {
    position: fixed;
    inset: 0;
    z-index: 99998;
    animation: scp-glitch 0.22s linear forwards;
    pointer-events: none;
  }

  @keyframes scp-glitch {
    0%   { background: transparent; clip-path: inset(0 0 100% 0); }
    10%  { background: rgba(20,184,166,0.3); clip-path: inset(10% 0 60% 0); transform: translateX(4px); }
    20%  { background: rgba(248,113,113,0.2); clip-path: inset(40% 0 20% 0); transform: translateX(-3px); }
    30%  { background: rgba(255,255,255,0.15); clip-path: inset(60% 0 10% 0); transform: translateX(5px); }
    40%  { background: rgba(20,184,166,0.2); clip-path: inset(0 0 80% 0); transform: translateX(-2px); }
    50%  { background: rgba(0,0,0,0.9); clip-path: inset(0 0 0 0); transform: translateX(0); }
    60%  { background: rgba(0,0,0,0.95); clip-path: inset(0 0 0 0); }
    80%  { background: rgba(0,0,0,0.98); }
    100% { background: #000; }
  }

  .scp-panic {
    position: fixed;
    inset: 0;
    background: #000;
    z-index: 99999;
    cursor: default;
    animation: scp-panic-on 0.15s var(--ease-out) both;
  }

  @keyframes scp-panic-on {
    from { opacity: 0; }
    to   { opacity: 1; }
  }

  /* ── Backdrop overlay ───────────────────────────────────────── */
  .scp-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-modal, 5000);
    background: rgba(0, 0, 0, 0.65);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4, 16px);
  }

  @media (max-width: 767px) {
    .scp-backdrop { align-items: flex-end; padding: 0; }
  }

  /* ── Panel shell ───────────────────────────────────────────── */
  .scp {
    display: flex;
    flex-direction: column;
    background: var(--chat-bg);
    border: 1px solid var(--chat-border-accent);
    border-radius: var(--radius-xl, 20px);
    overflow: hidden;
    width: 100%;
    max-width: 440px;
    height: min(86dvh, 660px);
    box-shadow:
      0 32px 80px rgba(0, 0, 0, 0.9),
      0 0 0 1px rgba(20, 184, 166, 0.08),
      inset 0 1px 0 rgba(255, 255, 255, 0.06);
    position: relative;
    will-change: transform;
  }

  /* Ambient teal mesh */
  .scp::before {
    content: '';
    position: absolute;
    inset: 0;
    background:
      radial-gradient(ellipse 70% 50% at 15% 0%,   rgba(20, 184, 166, 0.07) 0%, transparent 55%),
      radial-gradient(ellipse 50% 40% at 85% 100%,  rgba(20, 184, 166, 0.05) 0%, transparent 50%),
      radial-gradient(ellipse 40% 35% at 50% 50%,   rgba(6, 182, 212, 0.03)  0%, transparent 55%);
    pointer-events: none;
    z-index: 0;
  }
  .scp > * { position: relative; z-index: 1; }

  @media (max-width: 767px) {
    .scp {
      max-width: 100%;
      border-radius: var(--radius-2xl, 24px) var(--radius-2xl, 24px) 0 0;
      height: 90dvh;
      padding-bottom: max(var(--space-4, 16px), env(safe-area-inset-bottom, 0px));
      animation: scp-slide-up 0.28s cubic-bezier(0.32, 0.72, 0, 1) both;
    }
  }

  @keyframes scp-slide-up {
    from { transform: translateY(100%); }
    to   { transform: translateY(0); }
  }

  /* ── Drag handle ─────────────────────────────────────────── */
  .scp-drag-handle {
    display: none;
    width: 40px; height: 4px;
    border-radius: var(--radius-full, 9999px);
    background: rgba(255, 255, 255, 0.12);
    margin: var(--space-2-5, 10px) auto var(--space-1, 4px);
    flex-shrink: 0;
  }
  @media (max-width: 767px) { .scp-drag-handle { display: block; } }

  /* ── Auto-lock progress bar ──────────────────────────────── */
  .scp-autolock-bar {
    position: absolute;
    top: 0; left: 0;
    height: 2px;
    background: linear-gradient(90deg, var(--chat-accent) 0%, rgba(6, 182, 212, 0.7) 100%);
    transition: width 1s linear, opacity 0.5s;
    z-index: 10;
    border-radius: 0 2px 2px 0;
    pointer-events: none;
  }

  /* ── Header ──────────────────────────────────────────────── */
  header.scp-header {
    display: flex;
    align-items: center;
    gap: var(--space-2-5, 10px);
    padding: var(--space-3, 12px) var(--space-3, 12px) var(--space-3, 12px) var(--space-4, 16px);
    background: rgba(255, 255, 255, 0.02);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border-bottom: 1px solid var(--chat-border);
    flex-shrink: 0;
    min-height: 60px;
  }

  .scp-header-lock {
    color: rgba(255, 255, 255, 0.2);
    display: flex;
    align-items: center;
    flex-shrink: 0;
    transition: color 0.2s;
  }
  .scp-header-lock--open { color: var(--chat-accent); }

  .scp-header-info { flex: 1; min-width: 0; }

  .scp-header-name-row {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
  }

  .scp-header-name {
    font-size: var(--text-base, 1rem);
    font-weight: 700;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.92);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    margin: 0;
    transition: opacity 0.2s;
  }
  .scp-header-name--hidden {
    opacity: 0.3;
    letter-spacing: 0.15em;
    color: rgba(255, 255, 255, 0.3);
  }

  .scp-unread-badge {
    min-width: 20px; height: 20px;
    padding: 0 var(--space-1-5, 6px);
    border-radius: var(--radius-full, 9999px);
    background: var(--chat-accent);
    color: #fff;
    font-size: var(--text-2xs, 0.6875rem);
    font-weight: 700;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .scp-header-sub {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    margin-top: 2px;
    min-height: 16px;
  }

  .scp-subtext {
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.35);
  }

  /* ── Typing indicator (CSS-only dots) ─────────────────────── */
  .scp-typing-indicator {
    display: flex;
    align-items: center;
    gap: 3px;
    padding: 2px 0;
  }

  .typing-dot {
    width: 5px; height: 5px;
    border-radius: var(--radius-full, 9999px);
    background: var(--chat-accent);
    animation: typing-bounce 1.4s ease-in-out infinite;
    flex-shrink: 0;
  }

  .typing-dot:nth-child(2) { animation-delay: 0.2s; }
  .typing-dot:nth-child(3) { animation-delay: 0.4s; }

  @keyframes typing-bounce {
    0%, 60%, 100% { transform: translateY(0);     opacity: 0.4; }
    30%           { transform: translateY(-5px);  opacity: 1;   }
  }

  .scp-presence-dot {
    width: 6px; height: 6px;
    border-radius: var(--radius-full, 9999px);
    background: rgba(255, 255, 255, 0.2);
    flex-shrink: 0;
  }
  .scp-presence-dot--active {
    background: var(--chat-accent);
    box-shadow: 0 0 0 3px var(--chat-accent-subtle);
    animation: scp-pulse-accent 2.4s ease-in-out infinite;
  }
  .scp-presence-dot--online {
    background: var(--success-400, #34d399);
    box-shadow: 0 0 0 3px rgba(52, 211, 153, 0.2);
    animation: scp-pulse-green 2.4s ease-in-out infinite;
  }

  @keyframes scp-pulse-accent {
    0%, 100% { box-shadow: 0 0 0 0 var(--chat-accent-glow); }
    50%       { box-shadow: 0 0 0 5px rgba(20, 184, 166, 0); }
  }
  @keyframes scp-pulse-green {
    0%, 100% { box-shadow: 0 0 0 0 rgba(52, 211, 153, 0.5); }
    50%       { box-shadow: 0 0 0 5px rgba(52, 211, 153, 0); }
  }

  /* ── Header buttons ──────────────────────────────────────── */
  .scp-icon-btn {
    width: 44px; height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: none;
    color: rgba(255, 255, 255, 0.3);
    cursor: pointer;
    border-radius: var(--radius-sm2, 8px);
    transition: background 0.1s, color 0.1s;
    flex-shrink: 0;
    touch-action: manipulation;
  }
  .scp-icon-btn:hover { background: rgba(255, 255, 255, 0.07); color: rgba(255, 255, 255, 0.75); }
  .scp-icon-btn:focus-visible { outline: 2px solid var(--chat-accent); outline-offset: 2px; }

  .scp-invite-btn {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    padding: 0 var(--space-3, 12px);
    height: 32px;
    min-height: 44px;
    border-radius: var(--radius-full, 9999px);
    border: 1px solid var(--chat-border-accent);
    background: var(--chat-accent-subtle);
    color: var(--chat-accent);
    font-size: var(--text-xs, 0.75rem);
    font-weight: 600;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    cursor: pointer;
    flex-shrink: 0;
    white-space: nowrap;
    transition: background 0.1s, box-shadow 0.1s;
    touch-action: manipulation;
  }
  .scp-invite-btn:hover { background: var(--chat-accent-dim); box-shadow: 0 0 16px var(--chat-accent-glow); }
  .scp-invite-btn:focus-visible { outline: 2px solid var(--chat-accent); outline-offset: 2px; }
  .scp-invite-btn--copied { border-color: rgba(52, 211, 153, 0.4); background: rgba(52, 211, 153, 0.08); color: var(--success-400, #34d399); }

  /* ── Messages area ───────────────────────────────────────── */
  main.scp-msgs {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-4, 16px) var(--space-4, 16px) var(--space-2-5, 10px);
    display: flex;
    flex-direction: column;
    gap: var(--space-1, 4px);
    overscroll-behavior: contain;
  }
  main.scp-msgs::-webkit-scrollbar { width: 3px; }
  main.scp-msgs::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.08);
    border-radius: var(--radius-full, 9999px);
  }

  /* ── Skeleton loading ────────────────────────────────────── */
  .scp-skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--space-2, 8px);
    padding: var(--space-4, 16px) 0;
    flex: 1;
  }
  .scp-skel-row {
    display: flex;
  }
  .scp-skel-row--own  { justify-content: flex-end; }
  .scp-skel-row--their { justify-content: flex-start; }

  .scp-skel-bubble {
    height: 38px; width: 160px;
    border-radius: var(--radius-xl, 20px);
    background: rgba(255, 255, 255, 0.04);
    animation: skel-shimmer 1.6s ease-in-out infinite;
  }
  .scp-skel-bubble--short { width: 100px; }
  .scp-skel-bubble--long  { width: 220px; }

  @keyframes skel-shimmer {
    0%, 100% { opacity: 0.5; }
    50%       { opacity: 1;   }
  }

  /* ── Empty state ─────────────────────────────────────────── */
  .scp-empty {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-3, 12px);
    padding: var(--space-10, 40px) 0;
    text-align: center;
  }

  /* CSS lock art with concentric pulsing rings */
  .scp-empty-lock-art {
    position: relative;
    width: 96px; height: 96px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: var(--space-2, 8px);
  }

  .scp-empty-lock-ring {
    position: absolute;
    border-radius: var(--radius-full, 9999px);
    border: 1px solid var(--chat-border-accent);
  }

  .scp-empty-lock-ring--outer {
    inset: 0;
    background: rgba(20, 184, 166, 0.04);
    animation: empty-ring-pulse 3s ease-in-out infinite;
  }

  .scp-empty-lock-ring--inner {
    inset: 12px;
    background: rgba(20, 184, 166, 0.07);
    animation: empty-ring-pulse 3s ease-in-out infinite 0.5s;
  }

  .scp-empty-lock-icon {
    width: 52px; height: 52px;
    border-radius: var(--radius-full, 9999px);
    background: var(--chat-accent-subtle);
    border: 1px solid var(--chat-border-accent);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--chat-accent);
    position: relative;
    z-index: 1;
    box-shadow: 0 0 32px rgba(20, 184, 166, 0.2);
  }

  @keyframes empty-ring-pulse {
    0%, 100% { transform: scale(1); opacity: 0.8; }
    50%       { transform: scale(1.05); opacity: 0.5; }
  }

  .scp-empty-title {
    margin: 0;
    font-size: var(--text-base, 1rem);
    font-weight: 700;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.8);
  }

  .scp-empty-sub {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.28);
    line-height: var(--leading-relaxed, 1.625);
    max-width: 260px;
  }

  .scp-empty-cta {
    margin: 0;
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: var(--chat-accent);
    font-weight: 600;
  }

  /* ── Date divider ─────────────────────────────────────────── */
  .scp-date-div {
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

  .scp-date-div::before,
  .scp-date-div::after {
    content: '';
    flex: 1;
    height: 1px;
    background: rgba(255, 255, 255, 0.05);
  }

  .scp-date-div span {
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

  /* ── Scroll-to-bottom FAB ────────────────────────────────── */
  .scp-scroll-fab {
    align-self: flex-end;
    margin: calc(-1 * var(--space-2, 8px)) var(--space-3, 12px) 0;
    position: relative;
    width: 44px; height: 44px;
    border-radius: var(--radius-full, 9999px);
    border: 1px solid var(--chat-border-accent);
    background: rgba(6, 6, 16, 0.9);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    color: var(--chat-accent);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.6), 0 0 12px var(--chat-accent-glow);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    touch-action: manipulation;
    transition: background 0.1s, transform 0.1s, box-shadow 0.1s;
    z-index: 10;
    flex-shrink: 0;
  }
  .scp-scroll-fab:hover { background: var(--chat-accent-subtle); transform: translateY(-2px); box-shadow: 0 6px 20px rgba(0, 0, 0, 0.6), 0 0 20px var(--chat-accent-glow); }
  .scp-scroll-fab:focus-visible { outline: 2px solid var(--chat-accent); outline-offset: 2px; }

  .scp-scroll-fab-badge {
    position: absolute;
    top: -5px; right: -5px;
    min-width: 18px; height: 18px;
    padding: 0 var(--space-1, 4px);
    border-radius: var(--radius-full, 9999px);
    background: var(--chat-accent);
    color: #fff;
    font-size: 10px;
    font-weight: 700;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.5);
  }

  /* ── Compose ─────────────────────────────────────────────── */
  footer.scp-compose {
    padding: var(--space-2-5, 10px) var(--space-4, 16px) max(var(--space-4, 16px), env(safe-area-inset-bottom, 0px));
    border-top: 1px solid var(--chat-border);
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5, 6px);
    flex-shrink: 0;
    background: rgba(0, 0, 0, 0.3);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
  }

  .scp-compose-inner {
    display: flex;
    align-items: flex-end;
    gap: var(--space-1-5, 6px);
  }

  .scp-compose-icon-btn {
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
    transition: color 0.1s, background 0.1s;
    touch-action: manipulation;
  }
  .scp-compose-icon-btn:hover { color: rgba(255, 255, 255, 0.7); background: rgba(255, 255, 255, 0.07); }
  .scp-compose-icon-btn:focus-visible { outline: 2px solid var(--chat-accent); outline-offset: 2px; }
  .scp-compose-icon-btn:disabled { opacity: 0.22; cursor: not-allowed; }
  .scp-compose-icon-btn--panic:hover { color: var(--danger-400, #f87171); background: rgba(248, 113, 113, 0.07); }
  .scp-compose-icon-btn--panic { color: rgba(255, 255, 255, 0.16); }
  .scp-compose-icon-btn--voice { color: rgba(255, 255, 255, 0.14); cursor: not-allowed; }
  .scp-compose-icon-btn--loading { cursor: wait; }
  .scp-compose-icon-btn--active { color: var(--chat-accent); background: var(--chat-accent-subtle); }

  .scp-mini-spinner {
    width: 15px; height: 15px;
    border: 2px solid rgba(255, 255, 255, 0.15);
    border-top-color: var(--chat-accent);
    border-radius: var(--radius-full, 9999px);
    animation: scp-spin 0.7s linear infinite;
  }

  .scp-compose-text {
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
    transition: border-color 0.1s, box-shadow 0.1s;
    -webkit-appearance: none;
    field-sizing: content;
    max-height: 120px;
    overflow-y: auto;
    min-height: 44px;
    box-sizing: border-box;
  }
  @media (max-width: 767px) { .scp-compose-text { font-size: 16px; } }
  .scp-compose-text:focus { border-color: var(--chat-border-accent); box-shadow: 0 0 0 3px var(--chat-accent-subtle); }
  .scp-compose-text::placeholder { color: rgba(255, 255, 255, 0.2); }

  .scp-send-btn {
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
    transition: transform 0.1s, box-shadow 0.2s, background 0.2s, color 0.2s;
    touch-action: manipulation;
  }
  .scp-send-btn--active {
    background: linear-gradient(135deg, var(--primary-400, #2dd4bf) 0%, var(--primary-600, #0d9488) 100%);
    color: #fff;
    box-shadow: 0 2px 12px rgba(20, 184, 166, 0.4);
  }
  .scp-send-btn--active:hover { transform: scale(1.06); box-shadow: 0 4px 20px rgba(20, 184, 166, 0.6); }
  .scp-send-btn--active:active { transform: scale(0.93); }
  .scp-send-btn:disabled { opacity: 0.28; cursor: not-allowed; box-shadow: none; }
  .scp-send-btn:focus-visible { outline: 2px solid var(--chat-accent); outline-offset: 2px; }

  .scp-send-ring {
    width: 16px; height: 16px;
    border: 2px solid rgba(255, 255, 255, 0.25);
    border-top-color: #fff;
    border-radius: var(--radius-full, 9999px);
    animation: scp-spin 0.7s linear infinite;
  }

  /* ── Attach menu ─────────────────────────────────────────── */
  .scp-attach-wrap { position: relative; }

  .scp-attach-menu {
    position: absolute;
    bottom: calc(100% + 6px);
    left: 50%;
    transform: translateX(-50%);
    background: var(--chat-elevated, #0f0f20);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: var(--radius-lg, 14px);
    padding: var(--space-1, 4px);
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 160px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.7);
    animation: scp-pop 0.15s var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
    z-index: 10;
  }

  .scp-attach-item {
    display: flex;
    align-items: center;
    gap: var(--space-2, 8px);
    padding: var(--space-2-5, 10px) var(--space-3, 12px);
    background: none;
    border: none;
    border-radius: var(--radius-sm2, 8px);
    color: rgba(255, 255, 255, 0.8);
    font-size: var(--text-xs, 0.75rem);
    font-weight: 500;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    cursor: pointer;
    white-space: nowrap;
    min-height: 44px;
    transition: background 0.1s, color 0.1s;
  }
  .scp-attach-item:hover { background: rgba(255, 255, 255, 0.07); color: #fff; }
  .scp-attach-item:focus-visible { outline: 2px solid var(--chat-accent); outline-offset: 2px; }

  /* ── Compose meta ────────────────────────────────────────── */
  .scp-compose-meta {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .scp-compose-hint {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    margin: 0;
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.14);
  }

  .scp-char-count {
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: rgba(255, 255, 255, 0.3);
    font-variant-numeric: tabular-nums;
    transition: color 0.2s;
  }
  .scp-char-count--warn { color: var(--danger-400, #f87171); }

  /* ── Photo lightbox ──────────────────────────────────────── */
  .lightbox-backdrop {
    position: fixed;
    inset: 0;
    z-index: calc(var(--z-modal, 5000) + 100);
    background: rgba(0, 0, 0, 0.88);
    backdrop-filter: blur(24px) saturate(0.6);
    -webkit-backdrop-filter: blur(24px) saturate(0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-8, 32px);
    cursor: zoom-out;
  }

  .lightbox-img {
    max-width: 90vw;
    max-height: 80dvh;
    border-radius: var(--radius-xl, 20px);
    object-fit: contain;
    box-shadow: 0 24px 80px rgba(0, 0, 0, 0.8);
    animation: lightbox-in 0.25s var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
    cursor: default;
  }

  @keyframes lightbox-in {
    from { opacity: 0; transform: scale(0.88); }
    to   { opacity: 1; transform: scale(1);    }
  }

  .lightbox-close {
    position: absolute;
    top: var(--space-4, 16px);
    right: var(--space-4, 16px);
    width: 44px; height: 44px;
    border-radius: var(--radius-full, 9999px);
    border: 1px solid rgba(255, 255, 255, 0.15);
    background: rgba(255, 255, 255, 0.08);
    color: rgba(255, 255, 255, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: background 0.1s, color 0.1s;
    top: max(var(--space-4, 16px), env(safe-area-inset-top, 0px));
  }
  .lightbox-close:hover { background: rgba(255, 255, 255, 0.14); color: #fff; }
  .lightbox-close:focus-visible { outline: 2px solid var(--chat-accent); outline-offset: 2px; }

  .lightbox-hint {
    position: absolute;
    bottom: max(var(--space-6, 24px), env(safe-area-inset-bottom, 0px));
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    font-size: 11px;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255, 255, 255, 0.25);
  }

  /* ── Utility ─────────────────────────────────────────────── */
  .scp-sr {
    position: absolute; width: 1px; height: 1px;
    padding: 0; margin: -1px; overflow: hidden;
    clip: rect(0,0,0,0); white-space: nowrap; border: 0;
  }

  @keyframes scp-spin  { to { transform: rotate(360deg); } }
  @keyframes scp-pop {
    from { opacity: 0; transform: translateX(-50%) scale(0.88) translateY(6px); }
    to   { opacity: 1; transform: translateX(-50%) scale(1)    translateY(0);   }
  }

  /* ── prefers-reduced-motion ──────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .scp-glitch { animation: none; }
    .scp-panic  { animation: none; }
    .scp { animation: none; }
    .scp-empty-lock-ring { animation: none; }
    .scp-presence-dot--active  { animation: none; }
    .scp-presence-dot--online  { animation: none; }
    .typing-dot { animation: none; }
    .scp-skel-bubble { animation: none; }
    .scp-scroll-fab:hover { transform: none; }
    .scp-send-btn--active:hover { transform: none; }
    .lightbox-img { animation: none; }
  }
</style>
