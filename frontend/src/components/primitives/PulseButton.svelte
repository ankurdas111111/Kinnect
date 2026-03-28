<script>
  import { socket } from '../../lib/socket.js';
  import { haptics } from '../../lib/haptics.js';

  let holdTimer = null;
  let holding = false;
  let feedback = null; // 'ok' | 'callme' | null

  function clearFeedback() {
    setTimeout(() => { feedback = null; }, 1500);
  }

  function onPointerDown(e) {
    e.preventDefault();
    holding = true;
    holdTimer = setTimeout(() => {
      if (holding) {
        sendPulse('callme');
      }
    }, 2000);
  }

  function onPointerUp(e) {
    e.preventDefault();
    if (!holding) return;
    holding = false;
    if (holdTimer) {
      clearTimeout(holdTimer);
      holdTimer = null;
      sendPulse('ok');
    }
  }

  function onPointerCancel() {
    holding = false;
    if (holdTimer) { clearTimeout(holdTimer); holdTimer = null; }
  }

  function sendPulse(type) {
    socket.emit('sendPulse', { type });
    feedback = type;
    haptics[type === 'callme' ? 'sos' : 'light']?.();
    clearFeedback();
  }
</script>

<button
  class="pulse-fab"
  class:feedback-ok={feedback === 'ok'}
  class:feedback-callme={feedback === 'callme'}
  aria-label={holding ? 'Hold for Call Me pulse' : "Tap for I'm OK pulse"}
  on:pointerdown={onPointerDown}
  on:pointerup={onPointerUp}
  on:pointercancel={onPointerCancel}
  on:contextmenu|preventDefault
>
  {#if feedback === 'ok'}
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
      <polyline points="20 6 9 17 4 12"/>
    </svg>
  {:else if feedback === 'callme'}
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true">
      <path d="M6.62 10.79a15.05 15.05 0 0 0 6.59 6.59l2.2-2.2a1 1 0 0 1 1.01-.24 11.36 11.36 0 0 0 3.56.57 1 1 0 0 1 1 1V20a1 1 0 0 1-1 1A17 17 0 0 1 3 4a1 1 0 0 1 1-1h3.5a1 1 0 0 1 1 1c0 1.25.2 2.45.57 3.56a1 1 0 0 1-.25 1.02z"/>
    </svg>
  {:else}
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07A19.5 19.5 0 0 1 4.69 12 19.79 19.79 0 0 1 1.61 3.38 2 2 0 0 1 3.6 1.2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L7.91 9.91a16 16 0 0 0 6.09 6.09l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/>
    </svg>
  {/if}
  {#if feedback}
    <span class="pulse-label">{feedback === 'ok' ? "I'm OK" : 'Call me'}</span>
  {/if}
</button>

<style>
  .pulse-fab {
    position: fixed;
    right: var(--space-4);
    bottom: calc(64px + var(--space-4) + env(safe-area-inset-bottom, 0px));
    z-index: 400;
    width: 48px;
    height: 48px;
    border-radius: var(--radius-full);
    background: rgba(16, 185, 129, 0.90);
    border: 1.5px solid rgba(16, 185, 129, 0.50);
    color: white;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    touch-action: none;
    box-shadow: 0 4px 20px rgba(16, 185, 129, 0.35);
    transition: transform 150ms var(--ease-out), background 150ms, box-shadow 150ms;
    -webkit-user-select: none;
    user-select: none;
    overflow: visible;
  }

  .pulse-fab:active {
    transform: scale(0.90);
  }

  .pulse-fab.feedback-ok {
    background: rgba(16, 185, 129, 1);
    box-shadow: 0 0 28px rgba(16, 185, 129, 0.65);
  }

  .pulse-fab.feedback-callme {
    background: rgba(245, 158, 11, 0.95);
    border-color: rgba(245, 158, 11, 0.5);
    box-shadow: 0 0 28px rgba(245, 158, 11, 0.55);
  }

  .pulse-label {
    position: absolute;
    right: calc(100% + 8px);
    top: 50%;
    transform: translateY(-50%);
    white-space: nowrap;
    font-size: 11px;
    font-weight: 700;
    background: rgba(8, 8, 16, 0.88);
    color: white;
    padding: 4px 9px;
    border-radius: var(--radius-full);
    pointer-events: none;
    animation: pulse-label-pop 0.15s var(--ease-out);
  }

  @keyframes pulse-label-pop {
    from { opacity: 0; transform: translateY(-50%) scale(0.85); }
    to   { opacity: 1; transform: translateY(-50%) scale(1); }
  }
</style>
