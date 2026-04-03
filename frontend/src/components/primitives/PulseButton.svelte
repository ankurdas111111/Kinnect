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
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
    </svg>
  {/if}
  {#if feedback}
    <span class="pulse-label">{feedback === 'ok' ? "I'm OK" : 'Call me'}</span>
  {/if}
</button>

<style>
  .pulse-fab {
    position: fixed;
    left: var(--space-4);
    bottom: calc(var(--space-4) + 52px + 10px);
    z-index: calc(var(--z-panel, 1000) + 2);
    width: 48px;
    height: 48px;
    border-radius: var(--radius-full);
    background: linear-gradient(135deg, rgba(16, 185, 129, 0.95) 0%, rgba(5, 150, 105, 0.95) 100%);
    border: 1.5px solid rgba(16, 185, 129, 0.55);
    color: white;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    touch-action: none;
    /* 3D raised button */
    box-shadow:
      0 6px 20px rgba(16, 185, 129, 0.40),
      0 2px 6px rgba(16, 185, 129, 0.25),
      inset 0 2px 4px rgba(255, 255, 255, 0.18),
      inset 0 -2px 4px rgba(0, 0, 0, 0.15);
    transform-style: preserve-3d;
    transition:
      transform var(--duration-3d, 250ms) cubic-bezier(0.34, 1.56, 0.64, 1),
      background 150ms,
      box-shadow var(--duration-3d, 250ms) ease;
    -webkit-user-select: none;
    user-select: none;
    overflow: visible;
  }

  /* Mobile: SOS FAB is raised above tabs, match that offset */
  @media (max-width: 767px) {
    .pulse-fab {
      bottom: calc(var(--bottom-tab-height, 56px) + var(--safe-bottom, 0px) + var(--space-4) + 52px + 10px);
    }
  }

  .pulse-fab:active {
    transform: perspective(600px) translateZ(-6px) scale(0.88);
    box-shadow:
      0 1px 6px rgba(16, 185, 129, 0.30),
      inset 0 3px 6px rgba(0, 0, 0, 0.20);
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
    left: calc(100% + 8px);
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

  /* Desktop: slide with sidebar, matching SOS FAB offsets */
  :global(.app-layout.sidebar-open:not(.mobile)) .pulse-fab {
    left: calc(var(--sidebar-width, 400px) + var(--space-4));
  }
  :global(.app-layout.tablet.sidebar-open) .pulse-fab {
    left: calc(var(--sidebar-tablet, 320px) + var(--space-4));
  }
  :global(.app-layout.sidebar-closed:not(.mobile)) .pulse-fab {
    left: calc(var(--sidebar-collapsed, 56px) + var(--space-4));
  }
</style>
