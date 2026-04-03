<script>
  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { authUser, loadSession } from '../lib/stores/auth.js';
  import { socket, setupSocketHandlers } from '../lib/socket.js';

  export let params = {};

  let status = 'loading'; // loading | adding | success | error | login-required
  let message = '';
  let contactName = '';

  $: code = (params.code || '').trim().toUpperCase();

  onMount(async () => {
    // Ensure session is loaded
    if (!$authUser) {
      await loadSession();
    }

    if (!code || code.length < 4) {
      status = 'error';
      message = 'Invalid share code in link.';
      return;
    }

    if (!$authUser) {
      // Not logged in — save the code and redirect to login
      sessionStorage.setItem('kinnect_pending_contact', code);
      status = 'login-required';
      // Auto-redirect after a brief moment so user sees the message
      setTimeout(() => push('/login'), 2000);
      return;
    }

    // User is logged in — attempt to add contact via socket
    addContact();
  });

  function addContact() {
    status = 'adding';
    message = '';

    // Make sure socket handlers are set up
    if (!socket.connected) {
      setupSocketHandlers();
      socket.connect();
    }

    const onAdded = (data) => {
      contactName = data?.displayName || 'contact';
      status = 'success';
      message = `${contactName} added to your contacts!`;
      cleanup();
      // Redirect to main app after showing success
      setTimeout(() => push('/'), 2000);
    };

    const onError = (data) => {
      status = 'error';
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

    // Wait for socket to be connected, then emit
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
        status = 'error';
        message = 'Request timed out. Please try again from the app.';
        cleanup();
      }
    }, 10000);

    // Store cleanup for onDestroy
    _cleanup = cleanup;
  }

  let _cleanup = null;
  onDestroy(() => { if (_cleanup) _cleanup(); });
</script>

<div class="add-contact-page">
  <div class="add-contact-card">
    <!-- Kinnect logo -->
    <div class="add-contact-logo">
      <svg width="28" height="34" viewBox="0 0 20 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M10 1C5.029 1 1 5.029 1 10c0 6.938 8.25 13.1 9 14.1.75-1 9-7.162 9-14.1C19 5.029 14.971 1 10 1z" fill="var(--primary-500)" fill-opacity="0.95"/>
        <path d="M7 7v6M7 10l3.5-3M7 10l3.5 3" stroke="white" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    </div>

    {#if status === 'loading' || status === 'adding'}
      <div class="add-contact-spinner"></div>
      <h2>Adding contact...</h2>
      <p class="add-contact-sub">Code: <code>{code}</code></p>

    {:else if status === 'success'}
      <div class="add-contact-icon success">
        <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
      </div>
      <h2>{message}</h2>
      <p class="add-contact-sub">Redirecting to Kinnect...</p>

    {:else if status === 'login-required'}
      <div class="add-contact-icon info">
        <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><polyline points="10 17 15 12 10 7"/><line x1="15" y1="12" x2="3" y2="12"/></svg>
      </div>
      <h2>Sign in to add contact</h2>
      <p class="add-contact-sub">You need to be logged in to add <code>{code}</code> as a contact.</p>
      <p class="add-contact-sub">Redirecting to login...</p>
      <a href="#/login" class="add-contact-btn">Sign in now</a>

    {:else if status === 'error'}
      <div class="add-contact-icon error">
        <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
      </div>
      <h2>Oops</h2>
      <p class="add-contact-sub">{message}</p>
      <a href="#/" class="add-contact-btn">Open Kinnect</a>
    {/if}
  </div>
</div>

<style>
  .add-contact-page {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    min-height: 100dvh;
    padding: var(--space-4);
    background: var(--bg-base);
    font-family: var(--font-sans);
  }

  .add-contact-card {
    background: var(--surface-raised);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-xl);
    padding: var(--space-8) var(--space-6);
    max-width: 380px;
    width: 100%;
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
  }

  .add-contact-logo {
    width: 56px;
    height: 56px;
    border-radius: 16px;
    background: var(--surface-inset);
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: var(--space-2);
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

  .add-contact-sub code {
    background: var(--surface-inset);
    padding: 2px 8px;
    border-radius: var(--radius-sm);
    font-size: var(--text-sm);
    font-weight: 600;
    letter-spacing: 0.05em;
    color: var(--text-primary);
  }

  .add-contact-spinner {
    width: 36px;
    height: 36px;
    border: 3px solid var(--border-default);
    border-top-color: var(--primary-500);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  .add-contact-icon {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .add-contact-icon.success {
    background: var(--success-500, #10b981);
    color: white;
  }

  .add-contact-icon.error {
    background: var(--danger-500, #ef4444);
    color: white;
  }

  .add-contact-icon.info {
    background: var(--primary-500);
    color: white;
  }

  .add-contact-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-2) var(--space-5);
    background: var(--primary-500);
    color: white;
    border-radius: var(--radius-lg);
    font-size: var(--text-sm);
    font-weight: 600;
    text-decoration: none;
    margin-top: var(--space-2);
    transition: background 0.15s;
    cursor: pointer;
  }

  .add-contact-btn:hover {
    background: var(--primary-600);
  }

  @keyframes spin { to { transform: rotate(360deg); } }
</style>
