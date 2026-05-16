<script>
  /**
   * SecretChatPanel — orchestrates the secret chat experience.
   * Split into sub-components: SecretChatGate, SecretChatMessage, SecretChatCompose.
   * All encryption/socket logic lives here; UI delegates to sub-components.
   *
   * iOS keyboard avoidance:
   *   We set --keyboard-offset on the .scp-backdrop root element via
   *   panelRootEl.style.setProperty() instead of using inline transform on .scp.
   *   A transform on .scp creates a new CSS stacking context, which on iOS Chrome
   *   makes .scp a containing block for all position:fixed descendants (lightbox,
   *   panic overlay, glitch). Those then clip to the panel rect instead of the
   *   viewport. Using a CSS custom property avoids any stacking context promotion.
   *
   * iOS overflow/contain:
   *   overflow:clip on .scp is removed. It + any compositing hint (will-change,
   *   backdrop-filter) causes the same containing-block bug. The panel uses
   *   clip-path on .scp for border-radius clipping so the scroll area can still
   *   use overflow:auto without .scp being a containing block.
   *   NOTE: clip-path also creates a stacking context but DOES NOT make the element
   *   a containing block for position:fixed children on iOS — only overflow:*
   *   (other than "visible") and transform do.
   */
  import { onMount, onDestroy, tick, afterUpdate } from 'svelte';
  import { fade } from 'svelte/transition';
  import { socket, markSecretMsgSeen, createSecretChatInvite } from '../lib/socket.js';
  import { authUser } from '../lib/stores/auth.js';
  import { secretChats, lockSecretChat, storeDecrypted, secretChatPresence, addOptimisticMessage, failOptimisticMessage, removeSecretMessageByTempId } from '../lib/stores/secretChat.js';
  import { otherUsers } from '../lib/stores/map.js';
  import { encryptMessage, decryptMessage } from '../lib/crypto.js';
  import { compressImage } from '../lib/imageUtils.js';
  import { toasts } from '../lib/stores/toast.js';
  import { haptics } from '../lib/haptics.js';
  import SecretChatGate from './SecretChatGate.svelte';
  import SecretChatMessage from './SecretChatMessage.svelte';
  import SecretChatCompose from './SecretChatCompose.svelte';

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

  // ── iOS keyboard avoidance ─────────────────────────────────────
  // We track the keyboard height and expose it as --keyboard-offset on the root
  // element so the panel shell can translateY via CSS without creating a stacking
  // context on the JS-driven element itself.
  let panelRootEl;   // bind:this on .scp-backdrop

  function onVVChange() {
    if (!panelRootEl) return;
    const vv = window.visualViewport;
    if (!vv) return;
    // On iOS Chrome/Safari vv.height already excludes the on-screen keyboard.
    // vv.offsetTop is page-scroll, NOT keyboard — do not subtract it here.
    const kbH = Math.max(0, window.innerHeight - vv.height);
    const offset = kbH > 50 ? kbH : 0;
    panelRootEl.style.setProperty('--keyboard-offset', `${offset}px`);
  }

  // Compose state (compose UI lives in SecretChatCompose; parent owns crypto + socket)
  let sending = false;
  let photoSending = false;
  let copyDone = false;
  let messagesEl;
  let panicMode = false;
  let panicGlitching = false;

  // Delete confirmation — two-tap
  let deletingMsgId = null;

  // Peer typing indicator (derived from presence)
  let peerTyping = false;

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
  $: myId = $authUser?.userId ?? '';
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
  async function toggleInline(msgId) {
    activeDecryptId = activeDecryptId === msgId ? null : msgId;
    inlinePins = { ...inlinePins, [msgId]: '' };
    inlineErrors = { ...inlineErrors, [msgId]: '' };
    touchAutoLock();
    if (activeDecryptId === msgId) {
      // Focus the PIN input imperatively after Svelte renders it.
      // autofocus on the element causes a double keyboard-flash on iOS
      // Chrome/Safari because the virtual keyboard fires twice (element insert
      // + focus event). Waiting one tick and calling focus() once is correct.
      await tick();
      const el = document.getElementById(`inline-${msgId}`);
      el?.focus();
    }
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

  // ── Helpers ───────────────────────────────────────────────────
  function makeOptimisticMsg(plaintext) {
    return {
      id: -Date.now(),
      senderId: myId,
      receiverId: peerId,
      ciphertext: '',
      iv: '',
      salt: '',
      createdAt: new Date().toISOString(),
      pending: true,
      _plaintext: plaintext,
    };
  }

  // ── Photo sending (called by SecretChatCompose dispatch) ──────
  async function handlePhotoFromCompose(file) {
    if (!sessionPin) { toasts.error('Enter your PIN before sending a photo'); return; }
    if (file.size > 15 * 1024 * 1024) { toasts.error('Photo too large — max 15 MB'); return; }
    photoSending = true;
    haptics.tap?.();
    touchAutoLock();
    const tempMsg = makeOptimisticMsg('[photo]');
    addOptimisticMessage(peerId, tempMsg);
    // Store a sentinel so retryFailedMsg finds non-null plaintext and gives a
    // helpful "cannot retry photo" message rather than hitting the null fallback.
    storeDecrypted(peerId, tempMsg.id, '[photo]');
    await tick();
    scrollToBottom();
    try {
      const dataUrl = await compressImage(file);
      const payload = `[photo:${dataUrl}]`;
      const { ciphertext, iv, salt } = await encryptMessage(payload, sessionPin);
      socket.emit('sendSecretMsg', { receiverId: peerId, ciphertext, iv, salt });
      haptics.confirm?.();
    } catch (err) {
      failOptimisticMessage(peerId, tempMsg.id);
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
  async function sendFromCompose(text) {
    if (!text || !sessionPin || sending) return;
    sending = true;
    haptics.tap?.();
    touchAutoLock();
    // Show the message immediately — before the expensive PBKDF2 encryption runs.
    const tempMsg = makeOptimisticMsg(text);
    addOptimisticMessage(peerId, tempMsg);
    // Store plaintext keyed to tempId so retry can re-encrypt if needed.
    storeDecrypted(peerId, tempMsg.id, text);
    await tick();
    scrollToBottom();
    try {
      const { ciphertext, iv, salt } = await encryptMessage(text, sessionPin);
      socket.emit('sendSecretMsg', { receiverId: peerId, ciphertext, iv, salt });
    } catch {
      failOptimisticMessage(peerId, tempMsg.id);
      toasts.error('Failed to send — check your connection');
    } finally {
      sending = false;
    }
  }

  async function sendStickerFromCompose(tag) {
    if (!sessionPin || sending) return;
    sending = true;
    haptics.tap?.();
    touchAutoLock();
    const payload = `[sticker:${tag}]`;
    const tempMsg = makeOptimisticMsg(payload);
    addOptimisticMessage(peerId, tempMsg);
    // Store plaintext keyed to tempId so retry can re-encrypt if the send fails.
    storeDecrypted(peerId, tempMsg.id, payload);
    await tick();
    scrollToBottom();
    try {
      const { ciphertext, iv, salt } = await encryptMessage(payload, sessionPin);
      socket.emit('sendSecretMsg', { receiverId: peerId, ciphertext, iv, salt });
    } catch {
      failOptimisticMessage(peerId, tempMsg.id);
      toasts.error('Failed to send sticker — check your connection');
    } finally {
      sending = false;
    }
  }

  /** Re-send a failed message using its stored plaintext. */
  async function retryFailedMsg(tempId) {
    if (!sessionPin || sending) return;
    const failedMsg = chat.messages.find((m) => m.id === tempId);
    if (!failedMsg) return;
    // Read plaintext from decryptedMessages (stored at optimistic-add time)
    const plaintext = chat.decryptedMessages.get(tempId) ?? failedMsg._plaintext;
    if (!plaintext) {
      // No plaintext to re-encrypt — just remove the ghost bubble
      removeSecretMessageByTempId(peerId, tempId);
      toasts.error('Cannot retry — original message lost. Please type it again.');
      return;
    }
    // Photos cannot be retried — the raw data URL is not persisted (only the
    // '[photo]' sentinel is stored). Remove the ghost bubble and ask user to
    // re-attach. Stickers with a [sticker:tag] payload CAN be retried normally.
    if (plaintext === '[photo]') {
      removeSecretMessageByTempId(peerId, tempId);
      toasts.error('Photo send failed — please re-attach the photo to retry.');
      return;
    }
    // Flip bubble back to pending state while we re-encrypt
    secretChats.update((m) => {
      const copy = new Map(m);
      const c = copy.get(peerId);
      if (!c) return copy;
      const msgs = c.messages.map((msg) =>
        msg.id === tempId ? { ...msg, failed: false, pending: true } : msg
      );
      copy.set(peerId, { ...c, messages: msgs });
      return copy;
    });
    sending = true;
    await tick();
    scrollToBottom();
    try {
      const { ciphertext, iv, salt } = await encryptMessage(plaintext, sessionPin);
      socket.emit('sendSecretMsg', { receiverId: peerId, ciphertext, iv, salt });
    } catch {
      failOptimisticMessage(peerId, tempId);
      toasts.error('Retry failed — check your connection');
    } finally {
      sending = false;
    }
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
<!--
  bind:this={panelRootEl} — the keyboard avoidance JS sets --keyboard-offset
  on this element. The .scp panel uses that CSS var via translateY in CSS,
  keeping the JS completely separate from stacking context promotion.
-->
<div
  class="scp-backdrop"
  bind:this={panelRootEl}
  style="--keyboard-offset: 0px"
  transition:fade={{ duration: 180 }}
  on:click|self={onClose}
  role="dialog"
  aria-modal="true"
  aria-label="Secret Chat with {peerName}"
>
<div
  class="scp"
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
      style="transform: scaleX({autoLockProgress / 100}); opacity: {autoLockProgress < 30 ? 1 : 0.35}"
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
          on:retry={(e) => retryFailedMsg(e.detail)}
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
    <SecretChatCompose
      peerFirst={peerFirst}
      sending={sending}
      photoSending={photoSending}
      hasPin={!!sessionPin}
      on:sendText={(e) => sendFromCompose(e.detail)}
      on:sendPhoto={(e) => handlePhotoFromCompose(e.detail)}
      on:sendSticker={(e) => sendStickerFromCompose(e.detail)}
      on:panic={triggerPanic}
      on:typing={(e) => { if (e.detail) emitPresence(true); touchAutoLock(); }}
    />
  {/if}
</div>
</div>

<style>
  /* ─────────────────────────────────────────────────────────────
     Chat-specific design tokens — teal system
     Defined on .scp-backdrop so they cascade into all child
     components (SecretChatGate, SecretChatMessage, SecretChatCompose).
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
    z-index: calc(var(--z-topmost, 9000) - 1);
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
    z-index: var(--z-topmost, 9000);
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
    .scp-backdrop {
      align-items: flex-end;
      padding: 0;
      /* Disable backdrop-filter on mobile: on iOS Chrome a position:fixed element
         with backdrop-filter becomes a containing block for ALL position:fixed
         descendants (lightbox, emoji picker, panic overlay). Those then clip to
         the panel rect instead of the viewport.
         The rgba background provides the scrim without the containing-block bug. */
      backdrop-filter: none;
      -webkit-backdrop-filter: none;
    }
  }

  /* ── Panel shell ───────────────────────────────────────────── */
  .scp {
    display: flex;
    flex-direction: column;
    background: var(--chat-bg);
    border: 1px solid var(--chat-border-accent);
    border-radius: var(--radius-xl, 20px);
    /*
     * overflow:clip REMOVED — on iOS Chrome/Safari, overflow:clip (or overflow:hidden)
     * combined with any compositing hint makes this element a containing block for
     * position:fixed descendants. The lightbox, emoji picker, and panic overlay would
     * then be clipped to the panel rect instead of the viewport.
     *
     * Border-radius clipping is achieved via clip-path on the children instead,
     * or we accept the border-radius not clipping child content visually (which is
     * acceptable since the header/footer/messages have their own bg fills).
     *
     * clip-path on THIS element also creates a stacking context but does NOT make it
     * a containing block for position:fixed children — so it is safe to use for
     * visual rounding if needed. We leave it off here to keep the stacking context
     * list minimal and let child backgrounds provide the visual boundary.
     */
    width: 100%;
    max-width: 440px;
    height: min(86dvh, 660px);
    box-shadow:
      0 32px 80px rgba(0, 0, 0, 0.9),
      0 0 0 1px rgba(20, 184, 166, 0.08),
      inset 0 1px 0 rgba(255, 255, 255, 0.06);
    position: relative;
    /*
     * will-change:transform REMOVED — it promoted the panel to a compositing layer
     * which on iOS made it a containing block for position:fixed children, causing
     * the same clipping as overflow:hidden.
     *
     * Keyboard avoidance (previously translateY on this element) is now handled via
     * --keyboard-offset CSS custom property set on the parent .scp-backdrop.
     * The CSS transition below consumes it without any JS touching .scp's transform.
     */
    transform: translateY(calc(-1 * var(--keyboard-offset, 0px)));
    transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  }

  /* Ambient teal mesh — purely decorative, pointer-events:none, z-index:0 */
  .scp::before {
    content: '';
    position: absolute;
    inset: 0;
    background:
      radial-gradient(ellipse 70% 50% at 15% 0%,   rgba(20, 184, 166, 0.07) 0%, transparent 55%),
      radial-gradient(ellipse 50% 40% at 85% 100%,  rgba(20, 184, 166, 0.05) 0%, transparent 50%),
      radial-gradient(ellipse 40% 35% at 50% 50%,   rgba(6, 182, 212, 0.03)  0%, transparent 55%);
    pointer-events: none;
    border-radius: inherit;
    z-index: 0;
  }
  .scp > * { position: relative; z-index: 1; }

  @media (max-width: 767px) {
    .scp {
      max-width: 100%;
      border-radius: var(--radius-2xl, 24px) var(--radius-2xl, 24px) 0 0;
      /*
       * svh = small viewport height (stable — excludes browser chrome + keyboard at all times).
       * dvh updates continuously when the keyboard opens on iOS Chrome, causing layout thrash.
       * We declare dvh first (broader support), then svh overrides it in supporting browsers.
       */
      height: 90dvh;
      height: 90svh;
      animation: scp-slide-up 0.28s cubic-bezier(0.32, 0.72, 0, 1) both;
    }
  }

  @keyframes scp-slide-up {
    from { transform: translateY(100%); }
    to   { transform: translateY(calc(-1 * var(--keyboard-offset, 0px))); }
  }

  /* ── Drag handle ─────────────────────────────────────────── */
  .scp-drag-handle {
    display: none;
    width: 40px; height: 4px;
    border-radius: var(--radius-full, 9999px);
    background: rgba(255, 255, 255, 0.12);
    margin: 10px auto var(--space-1, 4px);
    flex-shrink: 0;
  }
  @media (max-width: 767px) { .scp-drag-handle { display: block; } }

  /* ── Auto-lock progress bar ──────────────────────────────── */
  .scp-autolock-bar {
    position: absolute;
    top: 0; left: 0;
    width: 100%;
    height: 2px;
    background: linear-gradient(90deg, var(--chat-accent) 0%, rgba(6, 182, 212, 0.7) 100%);
    transform-origin: left center;
    transition: transform 1s linear, opacity 0.5s;
    z-index: 10;
    border-radius: 0 2px 2px 0;
    pointer-events: none;
  }

  /* ── Header ──────────────────────────────────────────────── */
  header.scp-header {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: var(--space-3, 12px) var(--space-3, 12px) var(--space-3, 12px) var(--space-4, 16px);
    background: rgba(255, 255, 255, 0.02);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border-bottom: 1px solid var(--chat-border);
    flex-shrink: 0;
    min-height: 60px;
    /* border-radius top corners match panel */
    border-radius: var(--radius-xl, 20px) var(--radius-xl, 20px) 0 0;
  }

  @media (max-width: 767px) {
    header.scp-header {
      border-radius: var(--radius-2xl, 24px) var(--radius-2xl, 24px) 0 0;
      padding-top: max(var(--space-3, 12px), env(safe-area-inset-top, 0px));
    }
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
    opacity: 0.50;
    letter-spacing: 0.15em;
    color: rgba(255, 255, 255, 0.50);
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
    color: rgba(255, 255, 255, 0.52);
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
    0%, 60%, 100% { transform: translateY(0);    opacity: 0.4; }
    30%           { transform: translateY(-5px); opacity: 1;   }
  }

  .scp-presence-dot {
    width: 6px; height: 6px;
    border-radius: var(--radius-full, 9999px);
    background: rgba(255, 255, 255, 0.35);
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
    -webkit-overflow-scrolling: touch;
    padding: var(--space-4, 16px) var(--space-4, 16px) var(--space-2-5, 10px);
    display: flex;
    flex-direction: column;
    gap: var(--space-1, 4px);
    overscroll-behavior: contain;
    min-height: 0; /* flex child must be able to shrink below content height on iOS */
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
  .scp-skel-row--own   { justify-content: flex-end; }
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
    0%, 100% { transform: scale(1);    opacity: 0.8; }
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
    gap: 10px;
    margin: 10px 0 var(--space-1-5, 6px);
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
    top: max(var(--space-4, 16px), env(safe-area-inset-top, 0px));
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

  /* ── prefers-reduced-motion ──────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .scp-glitch                { animation: none; }
    .scp-panic                 { animation: none; }
    .scp                       { animation: none; transition: none; }
    .scp-empty-lock-ring       { animation: none; }
    .scp-presence-dot--active  { animation: none; }
    .scp-presence-dot--online  { animation: none; }
    .typing-dot                { animation: none; }
    .scp-skel-bubble           { animation: none; }
    .scp-scroll-fab:hover      { transform: none; }
    .lightbox-img              { animation: none; }
  }
</style>
