<script>
  /**
   * SecretChatMessage — renders a single message row.
   * Handles: own (ciphertext), received+decrypted, received+locked, photo, sticker.
   * Emits: toggleInline, relockMsg, deleteMsg, decryptPin
   */
  import { createEventDispatcher } from 'svelte';
  import { fade } from 'svelte/transition';

  const dispatch = createEventDispatcher();

  export let msg;          // raw message object from store
  export let plain = null; // decrypted plaintext, or null
  export let isOwn = false;
  export let showInline = false;
  export let inlinePin = '';
  export let inlineError = '';
  export let inlineUnlocking = false;
  export let lockedSet;    // Set<msgId>
  export let lockCountdown = null;
  export let deletingMsgId = null;
  export let myId = '';
  export let peerFirst = '';

  // Tick state for seen animation
  export let seenPulse = false;

  $: decrypted = !isOwn && plain !== null && !lockedSet?.has(msg.id);
  $: isLocked = !isOwn && !decrypted;
  $: unread = msg.senderId !== myId && !msg.seenAt;
  $: likelyPhoto = isLocked && isLikelyPhotoMsg(msg.ciphertext);

  const GIF_RE  = /^\[gif:(https?:\/\/[^\]]+)\]$/;
  const PHOTO_RE = /^\[photo:(data:image\/[^;]+;base64,[^\]]+)\]$/;

  function parseGif(t)   { const m = GIF_RE.exec(t);   return m ? m[1] : null; }
  function parsePhoto(t) { const m = PHOTO_RE.exec(t); return m ? m[1] : null; }
  function isLikelyPhotoMsg(ct) { return ct && ct.length > 5000; }

  function ciphertextGibberish(ct) {
    if (!ct) return '···';
    const s = Math.min(4, ct.length);
    const r = ct.slice(s, s + 30);
    return r.length >= 6 ? r + '…' : '···';
  }

  function clockTime(ts) {
    if (!ts) return '';
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function timeAgo(ts) {
    if (!ts) return '';
    const d = Date.now() - new Date(ts).getTime();
    if (d < 60000) return 'just now';
    if (d < 3600000) return `${Math.floor(d / 60000)}m ago`;
    if (d < 86400000) return `${Math.floor(d / 3600000)}h ago`;
    return new Date(ts).toLocaleDateString([], { month: 'short', day: 'numeric' });
  }
</script>

<div
  class="msg"
  class:msg--own={isOwn}
  class:msg--their={!isOwn}
  class:msg--group-cont={!msg.groupLast}
>

  <!-- ── Own message ────────────────────────────────────────── -->
  {#if isOwn}
    <div
      class="bubble bubble--own"
      class:bubble--grp-first={msg.groupFirst}
      class:bubble--grp-last={msg.groupLast}
      class:bubble--grp-mid={!msg.groupFirst && !msg.groupLast}
      title="End-to-end encrypted — only {peerFirst} can decrypt"
      aria-label="Encrypted message sent"
    >
      <span class="cipher-text" aria-hidden="true">{ciphertextGibberish(msg.ciphertext)}</span>
      <span class="lock-icon" aria-hidden="true">
        <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
      </span>
    </div>

    {#if msg.groupLast}
      <div class="meta meta--own">
        <time class="msg-time" datetime={msg.createdAt}>{clockTime(msg.createdAt)}</time>
        <span
          class="tick"
          class:tick--seen={msg.seenAt}
          class:tick--pulse={seenPulse}
          aria-label={msg.seenAt ? `Read at ${clockTime(msg.seenAt)}` : 'Sent'}
          title={msg.seenAt ? `Read ${clockTime(msg.seenAt)}` : 'Sent'}
        >
          {#if msg.seenAt}
            <!-- Double tick -->
            <svg width="18" height="10" viewBox="0 0 20 11" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <polyline points="1 6 5 10 13 2"/>
              <polyline points="7 10 15 2"/>
            </svg>
          {:else}
            <!-- Single tick -->
            <svg width="12" height="10" viewBox="0 0 13 11" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <polyline points="1 6 5 10 12 2"/>
            </svg>
          {/if}
        </span>
      </div>
    {/if}

    <!-- Delete action — visible on hover -->
    <div class="msg-actions" role="group" aria-label="Message actions">
      <button
        class="delete-btn"
        class:delete-btn--confirm={deletingMsgId === msg.id}
        on:click={() => dispatch('delete', msg.id)}
        aria-label={deletingMsgId === msg.id ? 'Tap again to confirm delete' : 'Delete message'}
        type="button"
      >
        {#if deletingMsgId === msg.id}
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
          Delete?
        {:else}
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>
        {/if}
      </button>
    </div>

  <!-- ── Received + decrypted ───────────────────────────────── -->
  {:else if decrypted}
    <div
      class="bubble bubble--their bubble--decrypted"
      class:bubble--grp-first={msg.groupFirst}
      class:bubble--grp-last={msg.groupLast}
      class:bubble--grp-mid={!msg.groupFirst && !msg.groupLast}
      class:bubble--photo={parsePhoto(plain) !== null}
    >
      {#if parsePhoto(plain)}
        <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-noninteractive-element-interactions -->
        <img
          src={parsePhoto(plain)}
          class="msg-photo"
          alt="Photo from {peerFirst}"
          loading="lazy"
          role="img"
          on:click={() => dispatch('photoExpand', parsePhoto(plain))}
        />
      {:else if parseGif(plain)}
        <img src={parseGif(plain)} class="msg-sticker" alt="Sticker" loading="lazy" />
      {:else}
        <p class="bubble-body">{plain}</p>
      {/if}

      <button
        class="relock-btn"
        on:click={() => dispatch('relock', msg.id)}
        title="Lock message"
        aria-label="Lock this message"
        type="button"
      >
        <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
      </button>
    </div>

    {#if msg.groupLast}
      <div class="meta meta--their">
        {#if lockCountdown != null}
          <span class="countdown" aria-live="polite">
            <span class="countdown-bar" style="--pct: {(lockCountdown / 30) * 100}%"></span>
            Locks in {lockCountdown}s
          </span>
        {:else if unread}
          <span class="unread-dot" aria-label="Unread"></span>
          <time class="msg-time msg-time--unread" datetime={msg.createdAt}>{timeAgo(msg.createdAt)}</time>
        {:else}
          <time class="msg-time" datetime={msg.createdAt}>{clockTime(msg.createdAt)}</time>
        {/if}
      </div>
    {/if}

  <!-- ── Received + locked ──────────────────────────────────── -->
  {:else}
    <button
      class="bubble bubble--locked"
      class:bubble--locked-active={showInline}
      class:bubble--grp-first={msg.groupFirst}
      class:bubble--grp-last={msg.groupLast}
      class:bubble--grp-mid={!msg.groupFirst && !msg.groupLast}
      class:bubble--locked-photo={likelyPhoto}
      on:click={() => dispatch('toggleInline', msg.id)}
      aria-expanded={showInline}
      aria-label={likelyPhoto ? 'Encrypted photo — tap to enter PIN and decrypt' : 'Encrypted message — tap to enter PIN and decrypt'}
      type="button"
    >
      {#if unread}
        <span class="unread-pulse" aria-hidden="true"></span>
      {/if}
      {#if likelyPhoto}
        <span class="photo-placeholder" aria-hidden="true">
          <span class="photo-blur-layer"></span>
          <span class="photo-cam-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
              <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
              <circle cx="12" cy="13" r="4"/>
            </svg>
          </span>
        </span>
        <span class="locked-photo-label">
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          Photo — tap to unlock
        </span>
      {:else}
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
        <span class="cipher-preview" aria-hidden="true">{ciphertextGibberish(msg.ciphertext)}</span>
        {#if unread}
          <span class="msg-ago">{timeAgo(msg.createdAt)}</span>
        {/if}
      {/if}
    </button>

    {#if showInline}
      <div class="inline-decrypt" transition:fade={{ duration: 100 }}>
        <label class="sr-only" for="inline-{msg.id}">Sender's PIN to decrypt</label>
        <input
          id="inline-{msg.id}"
          class="inline-pin"
          type="password"
          inputmode="numeric"
          pattern="[0-9]*"
          maxlength="8"
          placeholder="Sender's PIN"
          value={inlinePin}
          on:input={(e) => dispatch('inlinePinInput', { id: msg.id, value: e.target.value.replace(/\D/g, '') })}
          on:keydown={(e) => e.key === 'Enter' && dispatch('decryptOne', msg)}
          disabled={inlineUnlocking}
          autocomplete="off"
          aria-describedby={inlineError ? `inline-err-${msg.id}` : undefined}
        />
        <button
          class="inline-btn"
          on:click={() => dispatch('decryptOne', msg)}
          disabled={inlineUnlocking || inlinePin.length < 4}
          type="button"
          aria-label="Decrypt message"
        >
          {inlineUnlocking ? '…' : 'Unlock'}
        </button>
        {#if inlineError}
          <span class="inline-err" id="inline-err-{msg.id}" role="alert">{inlineError}</span>
        {/if}
      </div>
    {/if}

    {#if msg.groupLast}
      <div class="meta meta--their">
        {#if unread}
          <span class="unread-dot" aria-label="Unread"></span>
          <time class="msg-time msg-time--unread" datetime={msg.createdAt}>{timeAgo(msg.createdAt)}</time>
        {:else}
          <time class="msg-time" datetime={msg.createdAt}>{clockTime(msg.createdAt)}</time>
        {/if}
      </div>
    {/if}
  {/if}
</div>

<style>
  .msg {
    display: flex;
    flex-direction: column;
    max-width: 76%;
    gap: 2px;
    margin-bottom: var(--space-1, 4px);
    animation: msg-in 0.22s var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
  }

  .msg--group-cont { margin-bottom: 1px; }
  .msg--own  { align-self: flex-end;   align-items: flex-end;   }
  .msg--their { align-self: flex-start; align-items: flex-start; }

  @keyframes msg-in {
    from { opacity: 0; transform: translateY(10px) scale(0.97); }
    to   { opacity: 1; transform: translateY(0)    scale(1);    }
  }

  /* ── Bubble base ── */
  .bubble {
    padding: var(--space-2-5, 10px) var(--space-3-5, 14px);
    border-radius: var(--radius-xl, 20px);
    font-size: var(--text-sm, 0.875rem);
    line-height: var(--leading-relaxed, 1.625);
    font-family: var(--font-sans, 'Nunito', sans-serif);
  }

  /* Grouping radius — iMessage-style */
  .bubble--own.bubble--grp-first  { border-bottom-right-radius: 6px; }
  .bubble--own.bubble--grp-last   { border-top-right-radius: 6px; }
  .bubble--own.bubble--grp-mid    { border-top-right-radius: 6px; border-bottom-right-radius: 6px; }

  .bubble--their.bubble--grp-first,
  .bubble--locked.bubble--grp-first { border-bottom-left-radius: 6px; }
  .bubble--their.bubble--grp-last,
  .bubble--locked.bubble--grp-last  { border-top-left-radius: 6px; }
  .bubble--their.bubble--grp-mid,
  .bubble--locked.bubble--grp-mid   { border-top-left-radius: 6px; border-bottom-left-radius: 6px; }

  /* ── Own bubble ── */
  .bubble--own {
    display: flex;
    align-items: center;
    gap: var(--space-2, 8px);
    background: linear-gradient(135deg, rgba(20, 184, 166, 0.15) 0%, rgba(20, 184, 166, 0.08) 100%);
    border: 1px solid rgba(20, 184, 166, 0.25);
    box-shadow: inset 0 1px 0 rgba(255,255,255,0.06), 0 2px 8px rgba(0,0,0,0.25);
    max-width: 100%;
    overflow: hidden;
  }

  .cipher-text {
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    font-size: var(--text-2xs, 0.6875rem);
    letter-spacing: 0.04em;
    color: rgba(20, 184, 166, 0.55);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
    user-select: none;
  }

  .lock-icon {
    color: rgba(20, 184, 166, 0.45);
    flex-shrink: 0;
    display: flex;
    align-items: center;
  }

  /* ── Their (decrypted) bubble ── */
  .bubble--their {
    background: rgba(255, 255, 255, 0.07);
    border: 1px solid rgba(255, 255, 255, 0.09);
    color: rgba(255, 255, 255, 0.92);
    word-break: break-word;
    box-shadow: inset 0 1px 0 rgba(255,255,255,0.04), 0 1px 4px rgba(0,0,0,0.15);
  }

  .bubble--decrypted {
    position: relative;
    padding-right: var(--space-8, 32px);
  }

  .bubble--decrypted.bubble--photo {
    padding: var(--space-1, 4px);
    overflow: hidden;
  }

  .bubble-body { margin: 0; }

  /* ── Relock button — full 44px touch target ── */
  .relock-btn {
    position: absolute;
    top: 4px; right: 4px;
    width: 44px; height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0,0,0,0.3);
    border: none;
    cursor: pointer;
    color: rgba(255,255,255,0.35);
    border-radius: var(--radius-sm2, 8px);
    transition: color 0.1s, background 0.1s;
    touch-action: manipulation;
  }

  .relock-btn:hover {
    color: var(--chat-accent);
    background: var(--chat-accent-dim);
  }

  .relock-btn:focus-visible {
    outline: 2px solid var(--chat-accent);
    outline-offset: 1px;
  }

  /* ── Photo message ── */
  :global(.msg-photo) {
    max-width: 240px;
    max-height: 280px;
    border-radius: var(--radius-md, 10px);
    display: block;
    object-fit: cover;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  :global(.msg-photo:hover) { opacity: 0.92; }

  :global(.msg-sticker) {
    max-width: 128px;
    max-height: 128px;
    border-radius: var(--radius-sm2, 8px);
    display: block;
  }

  /* ── Locked bubble ── */
  .bubble--locked {
    display: flex;
    align-items: center;
    gap: var(--space-2, 8px);
    background: rgba(255,255,255,0.03);
    border: 1px dashed rgba(255,255,255,0.12);
    color: rgba(255,255,255,0.3);
    font-size: var(--text-xs, 0.75rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    cursor: pointer;
    min-height: 44px;
    transition: background 0.1s, border-color 0.1s;
    position: relative;
    touch-action: manipulation;
    overflow: hidden;
  }

  .bubble--locked:hover {
    background: rgba(255,255,255,0.06);
    border-color: rgba(255,255,255,0.2);
  }

  .bubble--locked:focus-visible {
    outline: 2px solid var(--chat-accent);
    outline-offset: 2px;
  }

  .bubble--locked-active {
    border-color: var(--chat-border-accent);
    background: var(--chat-accent-subtle);
    border-style: solid;
  }

  .bubble--locked-photo {
    flex-direction: column;
    align-items: flex-start;
    padding: 0;
    border-style: solid;
    min-height: 88px;
    overflow: hidden;
    border-radius: var(--radius-lg, 14px);
  }

  .photo-placeholder {
    position: relative;
    width: 190px; height: 130px;
    display: block;
    overflow: hidden;
    flex-shrink: 0;
  }

  .photo-blur-layer {
    position: absolute; inset: 0;
    background:
      radial-gradient(ellipse 60% 50% at 30% 40%, rgba(20,184,166,0.15) 0%, transparent 60%),
      radial-gradient(ellipse 40% 55% at 70% 60%, rgba(6,182,212,0.10) 0%, transparent 55%),
      linear-gradient(135deg, rgba(255,255,255,0.03) 0%, rgba(0,0,0,0.3) 100%);
    filter: blur(10px);
  }

  .photo-cam-icon {
    position: absolute; inset: 0;
    display: flex; align-items: center; justify-content: center;
    color: rgba(255,255,255,0.5);
  }

  .locked-photo-label {
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
    padding: var(--space-2, 8px) var(--space-3, 12px);
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: rgba(255,255,255,0.35);
    border-top: 1px solid rgba(255,255,255,0.06);
    width: 100%;
    box-sizing: border-box;
    flex-shrink: 0;
  }

  .cipher-preview {
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    font-size: 10px;
    letter-spacing: 0.04em;
    color: rgba(255,255,255,0.28);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
    user-select: none;
  }

  .unread-pulse {
    width: 8px; height: 8px;
    border-radius: var(--radius-full, 9999px);
    background: var(--chat-accent);
    box-shadow: 0 0 0 0 var(--chat-accent-glow);
    animation: pulse-accent 1.8s ease-in-out infinite;
    flex-shrink: 0;
  }

  @keyframes pulse-accent {
    0%, 100% { box-shadow: 0 0 0 0 var(--chat-accent-glow); }
    50%       { box-shadow: 0 0 0 5px rgba(20,184,166,0); }
  }

  .msg-ago {
    margin-left: auto;
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: rgba(255,255,255,0.22);
    white-space: nowrap;
    flex-shrink: 0;
  }

  /* ── Meta row ── */
  .meta {
    display: flex;
    align-items: center;
    gap: var(--space-1, 4px);
    padding: 0 2px;
    margin-top: 1px;
  }

  .meta--own   { justify-content: flex-end; }
  .meta--their { justify-content: flex-start; }

  .msg-time {
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: rgba(255,255,255,0.25);
    font-variant-numeric: tabular-nums;
  }

  .msg-time--unread {
    color: var(--chat-accent);
    font-weight: 600;
  }

  .unread-dot {
    width: 6px; height: 6px;
    border-radius: var(--radius-full, 9999px);
    background: var(--chat-accent);
    box-shadow: 0 0 5px var(--chat-accent-glow);
    flex-shrink: 0;
  }

  /* ── Tick (read receipt) ── */
  .tick {
    display: flex;
    align-items: center;
    color: rgba(255,255,255,0.28);
    transition: color 0.3s;
  }

  .tick--seen { color: var(--chat-accent); }

  .tick--pulse {
    animation: tick-seen-pulse 0.6s var(--ease-spring, cubic-bezier(0.34,1.56,0.64,1));
  }

  @keyframes tick-seen-pulse {
    0%   { color: rgba(255,255,255,0.28); transform: scale(1); }
    40%  { color: #fbbf24; transform: scale(1.4); }
    100% { color: var(--chat-accent); transform: scale(1); }
  }

  /* ── Countdown bar ── */
  .countdown {
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    color: var(--chat-accent);
    font-variant-numeric: tabular-nums;
    display: flex;
    align-items: center;
    gap: var(--space-1-5, 6px);
  }

  .countdown-bar {
    display: inline-block;
    width: 40px; height: 2px;
    border-radius: var(--radius-full, 9999px);
    background: rgba(255,255,255,0.12);
    position: relative;
    overflow: hidden;
  }

  .countdown-bar::before {
    content: '';
    position: absolute;
    inset: 0;
    right: calc(100% - var(--pct, 100%));
    background: var(--chat-accent);
    border-radius: var(--radius-full, 9999px);
    transition: right 1s linear;
  }

  /* ── Inline decrypt ── */
  .inline-decrypt {
    display: flex;
    align-items: center;
    gap: var(--space-2, 8px);
    flex-wrap: nowrap;
    margin-top: var(--space-1, 4px);
    padding: var(--space-2, 8px) var(--space-2-5, 10px);
    background: var(--chat-accent-subtle);
    border: 1px solid var(--chat-border-accent);
    border-radius: var(--radius-lg, 14px);
    width: 100%;
    box-sizing: border-box;
  }

  .inline-pin {
    flex: 1;
    min-width: 0;
    padding: var(--space-2-5, 10px);
    border-radius: var(--radius-sm2, 8px);
    border: 1px solid rgba(255,255,255,0.1);
    background: rgba(0,0,0,0.35);
    color: rgba(255,255,255,0.92);
    font-size: var(--text-base, 1rem);
    letter-spacing: 0.25em;
    text-align: center;
    outline: none;
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    -webkit-appearance: none;
    min-height: 44px;
    transition: border-color 0.1s;
    touch-action: manipulation;
  }

  .inline-pin:focus {
    border-color: var(--chat-accent);
    box-shadow: 0 0 0 2px var(--chat-accent-subtle);
  }

  @media (max-width: 767px) { .inline-pin { font-size: 18px; } }

  .inline-btn {
    padding: var(--space-2-5, 10px) var(--space-4, 16px);
    border-radius: var(--radius-sm2, 8px);
    border: none;
    background: var(--chat-accent);
    color: #fff;
    font-size: var(--text-xs, 0.75rem);
    font-weight: 700;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    cursor: pointer;
    min-height: 44px;
    flex-shrink: 0;
    white-space: nowrap;
    touch-action: manipulation;
    transition: background 0.1s, transform 0.1s;
  }

  .inline-btn:hover:not(:disabled) {
    background: var(--primary-400, #2dd4bf);
    transform: scale(1.02);
  }

  .inline-btn:disabled { opacity: 0.3; cursor: not-allowed; }

  .inline-btn:focus-visible {
    outline: 2px solid var(--chat-accent);
    outline-offset: 2px;
  }

  .inline-err {
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: var(--danger-400, #f87171);
    width: 100%;
    font-weight: 500;
  }

  /* ── Delete action row ── */
  .msg-actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 1px;
    opacity: 0;
    transition: opacity 0.1s;
    min-height: 24px;
  }

  .msg--own:hover .msg-actions,
  .msg--own:focus-within .msg-actions { opacity: 1; }

  @media (hover: none) { .msg-actions { opacity: 0.4; } }

  .delete-btn {
    display: flex;
    align-items: center;
    gap: var(--space-1, 4px);
    padding: var(--space-1, 4px) var(--space-2, 8px);
    border-radius: var(--radius-sm2, 8px);
    border: none;
    background: transparent;
    color: rgba(255,255,255,0.22);
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    cursor: pointer;
    transition: color 0.1s, background 0.1s;
    touch-action: manipulation;
    min-height: 44px;
  }

  .delete-btn:hover { color: var(--danger-400, #f87171); background: rgba(248,113,113,0.08); }
  .delete-btn--confirm { color: var(--danger-400, #f87171); background: rgba(248,113,113,0.12); }
  .delete-btn:focus-visible { outline: 2px solid var(--danger-400, #f87171); outline-offset: 2px; }

  .sr-only {
    position: absolute; width: 1px; height: 1px;
    padding: 0; margin: -1px; overflow: hidden;
    clip: rect(0,0,0,0); white-space: nowrap; border: 0;
  }

  @media (prefers-reduced-motion: reduce) {
    .msg { animation: none; }
    .unread-pulse { animation: none; }
    .tick--pulse { animation: none; }
    .relock-btn, .inline-btn, .delete-btn { transition: none; }
  }
</style>
