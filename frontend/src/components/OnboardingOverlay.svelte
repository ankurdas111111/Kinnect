<script>
  import { self } from 'svelte/legacy';

  import { createEventDispatcher, onMount } from 'svelte';
  import { authUser } from '../lib/stores/auth.js';
  import { myRooms } from '../lib/stores/rooms.js';
  import { socket } from '../lib/socket.js';

  /**
   * @typedef {Object} Props
   * @property {boolean} [visible]
   */

  /** @type {Props} */
  let { visible = false } = $props();

  const dispatch = createEventDispatcher();

  let step = $state(1);
  let contactCode = $state('');
  let adding = $state(false);
  let addError = $state('');
  let addSuccess = $state(false);

  // Hearth 03a — "Give your family a name" step
  let familyName = $state('');
  let creating = $state(false);
  let createError = $state('');

  let shareCode = $derived($authUser?.shareCode || '');

  // Returning users on a fresh device already belong to a family — don't ask
  // them to create another one; go straight to connecting.
  function nextAfterPermission() {
    step = ($myRooms?.length || 0) > 0 ? 3 : 2;
  }

  // An SOS can only reach a phone that is in a pocket via push, so ask here,
  // where the safety reason is on screen, rather than leaving it in Settings.
  // Fire-and-forget: a declined prompt must not block onboarding.
  function askForAlerts() {
    import('../lib/push.js').then(m => m.ensurePushSubscription({ prompt: true })).catch(() => {});
  }

  function handleCreateFamily() {
    if (!familyName.trim() || creating) return;
    creating = true;
    createError = '';
    socket.emit('createRoom', { name: familyName.trim() });
  }

  function handleAddContact() {
    if (!contactCode.trim() || adding || addSuccess) return;
    adding = true;
    addError = '';
    // WsCompatSocket does not support ack callbacks — use socket event listeners instead.
    socket.emit('addContact', { shareCode: contactCode.trim().toUpperCase() });
  }

  onMount(() => {
    const onContactAdded = () => {
      adding = false;
      addSuccess = true;
      setTimeout(() => dispatch('dismiss'), 1200);
    };
    const onContactError = (data) => {
      adding = false;
      addError = data?.message || 'Could not find user with that code.';
    };
    const onRoomCreated = () => {
      if (step !== 2) return;
      creating = false;
      step = 3;
    };
    const onRoomError = (data) => {
      if (step !== 2) return;
      creating = false;
      createError = data?.message || 'Could not create your family. Try again.';
    };
    socket.on('contactAdded', onContactAdded);
    socket.on('contactError', onContactError);
    socket.on('roomCreated', onRoomCreated);
    socket.on('roomError', onRoomError);
    return () => {
      socket.off('contactAdded', onContactAdded);
      socket.off('contactError', onContactError);
      socket.off('roomCreated', onRoomCreated);
      socket.off('roomError', onRoomError);
    };
  });

  function copyCode() {
    navigator.clipboard?.writeText(shareCode).catch(() => {});
    if (navigator.share) {
      navigator.share({ title: 'Join me on Kinnect', text: `Add me on Kinnect with code: ${shareCode}` }).catch(() => {});
    }
  }
</script>

{#if visible}
  <div class="onboarding-backdrop" onclick={self(() => dispatch('dismiss'))} onkeydown={(e) => { if (e.key === 'Escape') dispatch('dismiss'); }} role="dialog" aria-modal="true" aria-label="Get started" tabindex="-1">
    <div class="onboarding-card">
      <!-- Step indicators -->
      <div class="step-indicators" aria-label="Step {step} of 3">
        <span class="step-dot" class:active={step === 1}></span>
        <span class="step-dot" class:active={step === 2}></span>
        <span class="step-dot" class:active={step === 3}></span>
      </div>

      {#if step === 1}
        <!-- Step 1: Permission education -->
        <div class="onboarding-step" role="tabpanel" aria-label="Step 1: Enable location">
          <div class="brand-icon" aria-hidden="true">
            <svg width="40" height="48" viewBox="0 0 20 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M10 1C5.029 1 1 5.029 1 10c0 6.938 8.25 13.1 9 14.1.75-1 9-7.162 9-14.1C19 5.029 14.971 1 10 1z" fill="var(--text-on-primary)" fill-opacity="0.95"/>
              <path d="M7 7v6M7 10l3.5-3M7 10l3.5 3" stroke="var(--primary-500)" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <h2 class="onboarding-title verdict-voice">Welcome to Kinnect</h2>
          <!-- Earn the permission with one plain sentence before asking. -->
          <p class="onboarding-desc">Kinnect only works if the people you love can see where you are — so the next step asks for your location.</p>
          <div class="privacy-note">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
            Your location is private. Only your family can see it.
          </div>
          <div class="onboarding-actions">
            <button class="btn-primary-full" onclick={() => { dispatch('requestPermission'); askForAlerts(); nextAfterPermission(); }}>
              Turn on location
            </button>
            <button class="btn-ghost-sm" onclick={nextAfterPermission}>Maybe later</button>
          </div>
        </div>
      {:else if step === 2}
        <!-- Step 2 (Hearth 03a): name the family — the room IS the family -->
        <div class="onboarding-step" role="tabpanel" aria-label="Step 2: Name your family">
          <h2 class="onboarding-title verdict-voice">Give your family a name</h2>
          <p class="onboarding-desc">This is what everyone sees at the top of the map — "The Sharmas", "Home", whatever feels right.</p>

          <div class="input-row">
            <input
              class="code-input family-input"
              placeholder="Your family's name"
              bind:value={familyName}
              maxlength="50"
              onkeydown={(e) => e.key === 'Enter' && handleCreateFamily()}
            />
            <button class="add-btn" onclick={handleCreateFamily} disabled={creating || !familyName.trim()}>
              {#if creating}
                <span class="mini-spinner"></span>
              {:else}
                Create
              {/if}
            </button>
          </div>
          {#if createError}
            <span class="add-error">{createError}</span>
          {/if}

          <button class="btn-ghost-sm" style="margin-top: var(--space-3)" onclick={() => (step = 3)}>
            Have an invite code? Join a family instead
          </button>
        </div>
      {:else}
        <!-- Step 3: Add first person -->
        <div class="onboarding-step" role="tabpanel" aria-label="Step 3: Add a contact">
          <h2 class="onboarding-title verdict-voice">Connect with family</h2>
          <p class="onboarding-desc">Send your code to a family member, or type in theirs to start sharing locations.</p>

          <!-- Your share code -->
          <div class="code-block">
            <span class="code-label">Your family code</span>
            <div class="code-display">
              <span class="code-value">{shareCode || '—'}</span>
              <button class="copy-btn" onclick={copyCode} aria-label="Copy or share code" disabled={!shareCode}>
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
              </button>
            </div>
          </div>

          <!-- Enter their code -->
          <div class="input-row">
            <input
              class="code-input"
              placeholder="Their family code"
              bind:value={contactCode}
              maxlength="10"
              style="text-transform:uppercase"
              onkeydown={(e) => e.key === 'Enter' && handleAddContact()}
            />
            <button class="add-btn" onclick={handleAddContact} disabled={adding || !contactCode.trim() || addSuccess}>
              {#if adding}
                <span class="mini-spinner"></span>
              {:else if addSuccess}
                ✓
              {:else}
                Add
              {/if}
            </button>
          </div>
          {#if addError}
            <span class="add-error">{addError}</span>
          {/if}
          {#if addSuccess}
            <span class="add-success">You're connected!</span>
          {/if}

          <button class="btn-ghost-sm" style="margin-top: var(--space-4)" onclick={() => dispatch('dismiss')}>
            I'm all set
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  /* Ink veil over the room — warm, not black */
  .onboarding-backdrop {
    position: fixed;
    inset: 0;
    background: color-mix(in oklch, var(--ink) 45%, transparent);
    backdrop-filter: blur(6px);
    -webkit-backdrop-filter: blur(6px);
    z-index: var(--z-topmost, 9000);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4);
    animation: fade-in 0.2s ease;
  }

  /* Warm paper sheet — no glass, no flip theatrics */
  .onboarding-card {
    background: var(--surface-1);
    border: 1px solid var(--border-default);
    box-shadow: var(--shadow-xl);
    border-radius: 24px;
    /* Viewport-proportional: 22.5rem (360px, previous fixed max) is the floor
       so mobile renders identically; grows with 32vw and caps at 32rem. */
    width: min(92vw, clamp(22.5rem, 32vw, 32rem));
    padding: clamp(28px, 2.2vw, 40px) clamp(24px, 1.9vw, 36px);
    animation: card-in 320ms var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1)) both;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-4);
  }

  .step-indicators {
    display: flex;
    gap: 6px;
  }

  .step-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--border-default);
    transition: width 0.25s cubic-bezier(0.34,1.56,0.64,1), border-radius 0.25s, background 0.2s;
  }

  .step-dot.active {
    width: 20px;
    border-radius: 3px;
    background: var(--primary-500);
  }

  .onboarding-step {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    text-align: center;
  }

  .brand-icon {
    width: clamp(76px, 5.5vw, 88px);
    height: clamp(76px, 5.5vw, 88px);
    /* Canonical mark — white pin on a flat ember squircle (same mark the
       auth screens carry; keep the two in step if the mark ever changes). */
    background: var(--primary-500);
    border-radius: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: var(--shadow-md);
  }

  /* Serif register (verdict-voice supplies the italic) */
  .onboarding-title {
    font-size: clamp(22px, 1.6vw, 27px);
    color: var(--text-primary);
    margin: 0;
  }

  .onboarding-desc {
    font-size: 16px;
    color: var(--text-secondary);
    line-height: 1.55;
    margin: 0;
    max-width: min(100%, clamp(280px, 22vw, 400px));
  }

  .privacy-note {
    display: flex;
    align-items: center;
    gap: 6px;
    font-family: var(--font-sans);
    font-size: clamp(12px, 0.85vw, 13px);
    font-weight: 500;
    color: var(--text-secondary);
    background: var(--surface-3);
    border: 1px solid var(--border-default);
    border-radius: 10px;
    padding: 7px 14px;
  }

  .onboarding-actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 100%;
    margin-top: 4px;
  }

  .btn-primary-full {
    width: 100%;
    min-height: 48px;
    padding: 15px;
    border-radius: 14px;
    background: var(--primary-500);
    color: var(--text-on-primary);
    font-family: var(--font-sans);
    font-size: 16px;
    font-weight: 600;
    border: none;
    cursor: pointer;
    box-shadow: var(--shadow-sm);
    transition: transform 140ms cubic-bezier(0.34, 1.56, 0.64, 1), background 150ms var(--ease-out);
    letter-spacing: -0.01em;
  }

  .btn-primary-full:hover {
    background: var(--primary-600);
  }

  .btn-primary-full:active {
    transform: scale(0.95);
  }

  .btn-ghost-sm {
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-secondary);
    font-family: var(--font-sans);
    font-size: 14px;
    font-weight: 500;
    padding: 10px 12px;
    min-height: 44px;
    border-radius: 8px;
    transition: color 0.15s, background 0.15s;
  }

  .btn-ghost-sm:hover {
    color: var(--text-secondary);
    background: var(--surface-hover);
  }

  /* Code block */
  .code-block {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .code-label {
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.02em;
    color: var(--text-tertiary);
  }

  .code-display {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    background: var(--surface-3);
    border-radius: 10px;
    padding: 10px 12px;
    border: 1px solid var(--border-default);
  }

  .code-value {
    font-family: var(--font-sans);
    font-variant-numeric: tabular-nums;
    font-size: clamp(18px, 1.2vw, 21px);
    font-weight: 600;
    letter-spacing: 0.12em;
    color: var(--primary-700);
    flex: 1;
    text-align: center;
  }

  .copy-btn {
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-tertiary);
    padding: 4px;
    border-radius: 6px;
    display: flex;
    align-items: center;
    transition: color 0.15s;
  }

  .copy-btn:hover { color: var(--primary-700); }

  .copy-btn:disabled { opacity: 0.4; cursor: default; }

  /* Input row */
  .input-row {
    display: flex;
    gap: 8px;
    width: 100%;
  }

  .code-input {
    flex: 1;
    padding: 12px;
    min-height: 44px;
    border-radius: 10px;
    border: 1.5px solid var(--border-default);
    background: var(--surface-1);
    font-size: 16px;
    font-weight: 500;
    letter-spacing: 0.08em;
    color: var(--text-primary);
    outline: none;
    transition: border-color 0.15s;
  }

  .code-input:focus {
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px color-mix(in oklch, var(--primary-500) 15%, transparent);
  }

  /* Family-name input reads as a name, not a code */
  .family-input {
    letter-spacing: normal;
    text-transform: none;
  }

  .add-btn {
    padding: 10px 18px;
    min-height: 44px;
    border-radius: 10px;
    background: var(--primary-500);
    color: var(--text-on-primary);
    font-size: 14px;
    font-weight: 600;
    border: none;
    cursor: pointer;
    transition: background 0.15s, transform 0.1s;
    flex-shrink: 0;
    min-width: 60px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .add-btn:hover:not(:disabled) { background: var(--primary-600); }
  .add-btn:active:not(:disabled) { transform: scale(0.95); }
  .add-btn:disabled { opacity: 0.35; cursor: default; pointer-events: none; }

  /* A wrong code is a form mistake, not an emergency — ochre words. */
  .add-error {
    font-size: 13px;
    color: var(--warning-700);
    align-self: flex-start;
  }

  .add-success {
    font-size: 14px;
    font-weight: 600;
    color: var(--success-600);
  }

  .mini-spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid color-mix(in oklch, var(--text-on-primary) 30%, transparent);
    border-top-color: var(--text-on-primary);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin { to { transform: rotate(360deg); } }
  @keyframes fade-in { from { opacity: 0; } to { opacity: 1; } }
  @keyframes card-in {
    from { opacity: 0; transform: scale(0.90) translateY(24px); }
    to { opacity: 1; transform: scale(1) translateY(0); }
  }

  /* Reduced motion: instant overlay, no flip, no float — spinner stays
     (essential loading state). */
  @media (prefers-reduced-motion: reduce) {
    .onboarding-backdrop, .onboarding-card, .brand-icon { animation: none; }
  }
</style>
