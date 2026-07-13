<script>
  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { authUser, loadSession } from '../lib/stores/auth.js';
  import { socket, setupSocketHandlers } from '../lib/socket.js';
  import { haptics } from '../lib/haptics.js';
  import Skeleton from '../components/primitives/Skeleton.svelte';

  let { params = {} } = $props();

  let status = $state('loading'); // loading | adding | success | error | login-required
  let message = $state('');
  let contactName = '';
  // For zero-CLS crossfade between states
  let visible = $state(true);

  let code = $derived((params.code || '').trim().toUpperCase());

  // ── Calm-tier contract ────────────────────────────────────────────────────
  // AddContact is a deep-link landing surface (recipient-surface).
  // Gate ambient/decoration OFF regardless of stored fx level — same rule as
  // SharedLiveShell. We never start any ambient fx loop here.
  // ─────────────────────────────────────────────────────────────────────────

  onMount(async () => {
    // Ensure session is loaded
    if (!$authUser) {
      await loadSession();
    }

    if (!code || code.length < 4) {
      await crossfadeTo('error');
      message = 'Invalid share code in link.';
      return;
    }

    if (!$authUser) {
      // Not logged in — save the code and redirect to login
      sessionStorage.setItem('kinnect_pending_contact', code);
      await crossfadeTo('login-required');
      // Auto-redirect after a brief moment so user + screen reader see the message
      setTimeout(() => push('/login'), 3000);
      return;
    }

    // User is logged in — attempt to add contact via socket
    addContact();
  });

  /** 180ms opacity crossfade before switching status — prevents layout shift. */
  async function crossfadeTo(nextStatus) {
    visible = false;
    await new Promise(r => setTimeout(r, 180));
    status = nextStatus;
    visible = true;
  }

  function addContact() {
    status = 'adding';
    message = '';
    visible = true;

    // Make sure socket handlers are set up
    if (!socket.connected) {
      setupSocketHandlers();
      socket.connect();
    }

    const onAdded = async (data) => {
      contactName = data?.displayName || 'contact';
      await crossfadeTo('success');
      message = `${contactName} added to your contacts!`;
      haptics.confirm?.();
      cleanup();
      // Redirect to main app after showing success
      setTimeout(() => push('/'), 2000);
    };

    const onError = async (data) => {
      await crossfadeTo('error');
      message = data?.message || 'Could not add contact.';
      cleanup();
    };

    let onConnect = null;

    function cleanup() {
      socket.off('contactAdded', onAdded);
      socket.off('contactError', onError);
      if (onConnect) socket.off('connect', onConnect);
    }

    socket.on('contactAdded', onAdded);
    socket.on('contactError', onError);

    // ── connect-then-emit with 10s timeout (byte-identical handshake) ────────
    if (socket.connected) {
      socket.emit('addContact', { shareCode: code });
    } else {
      onConnect = () => {
        socket.off('connect', onConnect);
        onConnect = null;
        socket.emit('addContact', { shareCode: code });
      };
      socket.on('connect', onConnect);
    }

    // Timeout fallback
    setTimeout(() => {
      if (status === 'adding') {
        crossfadeTo('error').then(() => {
          message = 'Request timed out. Please try again from the app.';
          cleanup();
        });
      }
    }, 10000);
    // ─────────────────────────────────────────────────────────────────────────

    // Store cleanup for onDestroy
    _cleanup = cleanup;
  }

  function handleTryAgain() {
    addContact();
  }

  let _cleanup = null;
  onDestroy(() => { if (_cleanup) _cleanup(); });
</script>

<div class="add-contact-page">
  <!--
    min-height on the card anchors the layout so that loading → success / error / login-required
    state swaps produce zero CLS. The content inside crossfades via the .state-layer wrapper.
  -->
  <div class="add-contact-card" role="main">
    <!-- Kinnect logo -->
    <div class="add-contact-logo" aria-hidden="true">
      <svg width="28" height="34" viewBox="0 0 20 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M10 1C5.029 1 1 5.029 1 10c0 6.938 8.25 13.1 9 14.1.75-1 9-7.162 9-14.1C19 5.029 14.971 1 10 1z" fill="var(--primary-500)" fill-opacity="0.95"/>
        <path d="M7 7v6M7 10l3.5-3M7 10l3.5 3" stroke="white" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    </div>

    <div class="state-layer" class:faded={!visible} aria-live="polite" aria-atomic="true">
      {#if status === 'loading' || status === 'adding'}
        <h2>Adding contact…</h2>
        <div class="add-contact-skeleton" aria-hidden="true">
          <Skeleton variant="title" width="55%" />
          <Skeleton variant="text" count={2} />
        </div>
        <p class="add-contact-sub">Code <span class="code-chip">{code}</span></p>

      {:else if status === 'success'}
        <div class="add-contact-icon success" aria-hidden="true">
          <svg class="check-icon" xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
        </div>
        <h2>{message}</h2>
        <p class="add-contact-sub">Redirecting to Kinnect…</p>
        <a href="#/" class="add-contact-btn tactile" aria-label="Go to Kinnect now">Open Kinnect</a>

      {:else if status === 'login-required'}
        <div class="add-contact-icon info" aria-hidden="true">
          <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><polyline points="10 17 15 12 10 7"/><line x1="15" y1="12" x2="3" y2="12"/></svg>
        </div>
        <h2>Sign in to add contact</h2>
        <p class="add-contact-sub">You need to be logged in to add <span class="code-chip">{code}</span> as a contact.</p>
        <p class="add-contact-sub">Redirecting to login in 3 seconds…</p>
        <!-- Screen-reader affordance: primary CTA so they don't have to wait for redirect -->
        <a href="#/login" class="add-contact-btn tactile">Sign in now</a>

      {:else if status === 'error'}
        <div class="add-contact-icon error" aria-hidden="true">
          <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
        </div>
        <h2>Couldn't add contact</h2>
        <p class="add-contact-sub">{message}</p>
        <div class="btn-row">
          <button class="add-contact-btn tactile" onclick={handleTryAgain} type="button">Try again</button>
          <a href="#/" class="add-contact-btn add-contact-btn--ghost tactile">Open Kinnect</a>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .add-contact-page {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    min-height: 100dvh;
    /* Safe-area padding for notched devices in landscape */
    padding: max(var(--space-4), env(safe-area-inset-top, 0px))
             max(var(--space-4), env(safe-area-inset-right, 0px))
             max(var(--space-4), env(safe-area-inset-bottom, 0px))
             max(var(--space-4), env(safe-area-inset-left, 0px));
    background: var(--bg-base);
    font-family: var(--font-sans);
  }

  /* ── Card — raised-glass tier ─────────────────────────────────────────────
     Replaces the previous: background: var(--surface-raised) + a hardcoded
     drop shadow with the glass-panel tier tokens.
     min-height anchors the layout across all four state transitions (zero CLS).
  ────────────────────────────────────────────────────────────────────────── */
  .add-contact-card {
    background: var(--glass-panel-bg);
    backdrop-filter: var(--glass-panel-blur);
    -webkit-backdrop-filter: var(--glass-panel-blur);
    border: 1px solid var(--glass-panel-border);
    box-shadow: var(--glass-panel-shadow);
    border-radius: var(--radius-xl);
    padding: var(--space-8) var(--space-6);
    max-width: 380px;
    width: 100%;
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    /* Reserve vertical space so state swaps don't shift layout. */
    min-height: 320px;
    justify-content: center;
    /* Card entrance: subtle slide-up + fade. Calm-tier so intentionally muted. */
    animation: card-in 240ms var(--ease-out, cubic-bezier(0.4, 0, 0.2, 1)) both;
  }

  /* ── State layer: 180ms opacity crossfade between status blocks ─────────── */
  .state-layer {
    display: contents;
    /* `display:contents` means the children flow directly into .add-contact-card.
       We use a CSS custom property + transition on each child-block wrapper instead. */
  }

  /* When .faded, all direct children inside the card fade out */
  .add-contact-card.fading-state > *:not(.add-contact-logo) {
    opacity: 0;
  }

  /* Simpler: the wrapper is always rendered, apply opacity on the wrapper. */
  .state-layer {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
    transition: opacity 180ms var(--ease-out, cubic-bezier(0.4, 0, 0.2, 1));
  }

  .state-layer.faded {
    opacity: 0;
    pointer-events: none;
  }

  .add-contact-logo {
    width: 56px;
    height: 56px;
    border-radius: 16px;
    background: var(--surface-inset);
    display: flex;
    align-items: center;
    justify-content: center;
    /* Logo sits outside the crossfading state-layer so it never flickers. */
    flex-shrink: 0;
  }

  .add-contact-card h2 {
    font-family: var(--font-display, var(--font-sans));
    font-size: var(--text-lg);
    font-weight: 700;
    color: var(--text-primary);
    margin: 0;
    line-height: 1.3;
  }

  .add-contact-sub {
    font-size: var(--text-sm);
    color: var(--text-secondary);
    margin: 0;
    line-height: 1.5;
  }

  /* Token-styled share-code chip — monospace, subtle tint + border. */
  .code-chip {
    display: inline-flex;
    align-items: center;
    background: var(--primary-500-12, var(--surface-inset));
    border: 1px solid var(--primary-500-30, var(--border-default));
    padding: 2px 10px;
    border-radius: var(--radius-full, 9999px);
    font-family: var(--font-mono, monospace);
    font-size: var(--text-sm);
    font-weight: 600;
    letter-spacing: 0.08em;
    color: var(--text-primary);
    vertical-align: baseline;
  }

  /* Skeleton block for the loading / adding state. */
  .add-contact-skeleton {
    width: 100%;
    max-width: 240px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) 0;
  }

  .add-contact-icon {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .add-contact-icon.success {
    background: var(--status-live, var(--success-500));
    color: white;
    box-shadow: 0 0 0 0 var(--ring-color-live, var(--status-live, var(--success-500)));
    animation: success-ring-pulse 300ms var(--ease-out, cubic-bezier(0.4,0,0.2,1)) both;
  }

  /* .check-icon — pop-in scale morph (calm-tier: scale only, no spin/confetti) */
  .check-icon {
    animation: check-pop 150ms var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
  }

  .add-contact-icon.error {
    background: var(--status-sos, var(--danger-500));
    color: white;
    box-shadow: var(--glow-sos-sm, var(--shadow-danger, 0 4px 14px oklch(0.55 0.22 27 / 0.25)));
  }

  .add-contact-icon.info {
    background: var(--primary-500);
    color: white;
    box-shadow: var(--shadow-primary, 0 4px 14px oklch(0.55 0.18 275 / 0.30));
  }

  /* Button row: "Try again" (primary) + "Open Kinnect" (ghost) on error state */
  .btn-row {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    width: 100%;
    align-items: center;
    margin-top: var(--space-2);
  }

  .add-contact-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: 44px;
    min-width: 44px;
    padding: var(--space-2) var(--space-5);
    background: var(--primary-500);
    color: white;
    border: none;
    border-radius: var(--radius-lg);
    font-size: var(--text-sm);
    font-weight: 600;
    text-decoration: none;
    margin-top: var(--space-2);
    transition: background 150ms var(--ease-out, cubic-bezier(0.4,0,0.2,1)),
                box-shadow 150ms var(--ease-out, cubic-bezier(0.4,0,0.2,1)),
                transform 100ms var(--ease-out, cubic-bezier(0.4,0,0.2,1));
    cursor: pointer;
    font-family: var(--font-sans);
  }

  .add-contact-btn:hover {
    background: var(--primary-600);
  }

  .add-contact-btn:active {
    transform: scale(0.97);
  }

  .add-contact-btn:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }

  /* Ghost variant for secondary CTA */
  .add-contact-btn--ghost {
    background: transparent;
    color: var(--text-secondary);
    border: 1px solid var(--border-default);
    margin-top: 0;
  }

  .add-contact-btn--ghost:hover {
    background: var(--surface-inset);
    color: var(--text-primary);
  }

  /* ── Card entrance — slide-up + fade, muted for calm-tier ─────────────── */
  @keyframes card-in {
    from { opacity: 0; transform: translateY(8px) scale(0.98); }
    to   { opacity: 1; transform: translateY(0)   scale(1);    }
  }

  /* ── Success checkmark pop ─────────────────────────────────────────────── */
  @keyframes check-pop {
    from { transform: scale(0); }
    50%  { transform: scale(1.1); }
    to   { transform: scale(1); }
  }

  /* ── Single ring pulse on success icon border (calm-green-moment) ───────
     One 300ms pulse then settles. Deliberately not looping.
  ────────────────────────────────────────────────────────────────────────── */
  @keyframes success-ring-pulse {
    0%   { box-shadow: 0 0 0 0   color-mix(in oklch, var(--ring-color-live, var(--status-live)) 60%, transparent); }
    60%  { box-shadow: 0 0 0 12px color-mix(in oklch, var(--ring-color-live, var(--status-live)) 20%, transparent); }
    100% { box-shadow: 0 0 0 0   color-mix(in oklch, var(--ring-color-live, var(--status-live))  0%, transparent); }
  }

  @media (prefers-reduced-motion: reduce) {
    .add-contact-card  { animation: none; }
    .check-icon        { animation: none; }
    .state-layer       { transition: none; }
    .add-contact-btn   { transition: none; }
    .add-contact-icon.success { animation: none; }
  }
</style>
