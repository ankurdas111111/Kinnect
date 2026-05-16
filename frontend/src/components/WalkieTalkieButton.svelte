<script>
  import { onMount, onDestroy } from 'svelte';
  import { callState, callPeer } from '../lib/stores/webrtc.js';
  import {
    startCall,
    hangup,
    startTransmitting,
    stopTransmitting,
    initWebRTCSocketHandlers,
  } from '../lib/webrtc.js';

  export let user;

  onMount(() => { initWebRTCSocketHandlers(); });

  $: pttActive =
    ($callState === 'calling' || $callState === 'connected') &&
    $callPeer?.userID === user?.userId;

  $: callDisabled = !user?.online || ($callState !== 'idle' && !pttActive);

  function handleTalk() {
    startCall(user.userId, user.displayName || user.name || user.userId);
  }

  function handlePointerDown(e) {
    e.preventDefault();
    startTransmitting();
  }

  function handlePointerUp() {
    stopTransmitting();
  }
</script>

{#if pttActive && $callState === 'calling'}
  <!-- Outgoing call bar -->
  <div class="call-bar calling" role="status" aria-live="polite">
    <span class="call-label">
      Calling
      <span class="dots"><span>.</span><span>.</span><span>.</span></span>
    </span>
    <button class="btn-cancel" on:click={hangup} aria-label="Cancel call">End</button>
  </div>

{:else if pttActive && $callState === 'connected'}
  <!-- PTT connected bar -->
  <div class="call-bar connected" role="status" aria-live="polite">
    <button
      class="btn-ptt"
      on:pointerdown={handlePointerDown}
      on:pointerup={handlePointerUp}
      on:pointerleave={handlePointerUp}
      aria-label="Hold to talk"
      style="touch-action: none; user-select: none;"
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
        <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/>
        <path d="M19 10v2a7 7 0 0 1-14 0v-2"/>
        <line x1="12" y1="19" x2="12" y2="23"/>
        <line x1="8" y1="23" x2="16" y2="23"/>
      </svg>
      Hold to Talk
    </button>
    <button class="btn-cancel" on:click={hangup} aria-label="End call">End</button>
  </div>

{:else}
  <!-- Idle — compact Talk button for card-actions row -->
  <button
    class="btn btn-secondary btn-sm action-btn talk-btn"
    on:click={handleTalk}
    disabled={callDisabled}
    aria-label="Start walkie-talkie with {user?.displayName || user?.name || 'this person'}"
    title={user?.online ? 'Walkie-talkie' : 'User is offline'}
  >
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
      <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/>
      <path d="M19 10v2a7 7 0 0 1-14 0v-2"/>
      <line x1="12" y1="19" x2="12" y2="23"/>
      <line x1="8" y1="23" x2="16" y2="23"/>
    </svg>
    Talk
  </button>
{/if}

<style>
  /* ── Call bars ─────────────────────────────────────────────────── */
  .call-bar {
    display: flex;
    align-items: center;
    gap: var(--space-3, 12px);
    padding: var(--space-2, 8px) var(--space-4, 16px);
    border-radius: var(--radius-md, 8px);
    width: 100%;
  }

  .call-bar.calling {
    background: var(--bg-secondary, rgba(255,255,255,0.06));
    border: 1px solid var(--border-subtle, rgba(255,255,255,0.1));
  }

  .call-bar.connected {
    background: rgba(20, 184, 166, 0.1);
    border: 1px solid rgba(20, 184, 166, 0.3);
  }

  .call-label {
    flex: 1;
    font-size: var(--text-sm, 13px);
    color: var(--text-secondary, rgba(255,255,255,0.6));
  }

  /* Animated dots for "Calling..." */
  .dots span {
    animation: blink 1.2s infinite;
    opacity: 0;
  }
  .dots span:nth-child(2) { animation-delay: 0.2s; }
  .dots span:nth-child(3) { animation-delay: 0.4s; }

  @keyframes blink {
    0%, 80%, 100% { opacity: 0; }
    40% { opacity: 1; }
  }

  /* ── PTT button ───────────────────────────────────────────────── */
  .btn-ptt {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2, 8px);
    min-height: 44px;
    padding: var(--space-2, 8px) var(--space-4, 16px);
    background: var(--brand-primary, #14b8a6);
    color: var(--text-on-primary);
    border: none;
    border-radius: var(--radius-md, 8px);
    font-size: var(--text-sm, 13px);
    font-weight: 600;
    cursor: pointer;
    transition: background 120ms, box-shadow 120ms;
  }

  .btn-ptt:active {
    background: var(--brand-primary-dark, #0d9488);
    box-shadow: 0 0 0 4px rgba(20, 184, 166, 0.35);
  }

  /* ── Cancel / End button ──────────────────────────────────────── */
  .btn-cancel {
    min-width: 48px;
    min-height: 44px;
    padding: var(--space-2, 8px) var(--space-3, 12px);
    background: rgba(239, 68, 68, 0.15);
    color: var(--status-danger, #ef4444);
    border: 1px solid rgba(239, 68, 68, 0.3);
    border-radius: var(--radius-md, 8px);
    font-size: var(--text-sm, 13px);
    font-weight: 600;
    cursor: pointer;
    transition: background 120ms;
  }

  .btn-cancel:hover {
    background: rgba(239, 68, 68, 0.25);
  }

  /* ── Idle Talk button (fits inside card-actions row) ─────────── */
  .talk-btn {
    display: flex;
    align-items: center;
    gap: var(--space-1, 4px);
  }

  @media (prefers-reduced-motion: reduce) {
    .dots span { animation: none; opacity: 1; }
    .btn-ptt, .btn-cancel { transition: none; }
  }
</style>
