<script>
  import { createEventDispatcher } from 'svelte';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { haptics } from '../../lib/haptics.js';
  import { getAvatarStyle } from './avatarPalette.js';

  
  /**
   * @typedef {Object} Props
   * @property {any} [user] - User shown in the sheet; null hides it.
   */

  /** @type {Props} */
  let { user = null } = $props();

  const dispatch = createEventDispatcher();

  function qaCopy() {
    if (!user?.latitude) return;
    navigator.clipboard?.writeText(`${user.latitude.toFixed(6)}, ${user.longitude.toFixed(6)}`).catch(() => {});
    haptics.success?.();
    dispatch('close');
  }
</script>

<!-- ── Long-press quick actions sheet ─────────────────────────────────── -->
{#if user}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div
    class="qa-backdrop"
    onclick={() => dispatch('close')}
    aria-hidden="true"
    transition:fade={{ duration: 200 }}
  ></div>
  <div
    class="qa-sheet"
    role="dialog"
    aria-label="Quick actions for {user.displayName}"
    aria-modal="true"
    transition:fly={{ y: 180, duration: 300, easing: cubicOut }}
  >
    <div class="qa-handle" aria-hidden="true"></div>

    <!-- User identity -->
    <div class="qa-user-header">
      <div class="qa-avatar" style="{getAvatarStyle(user.displayName)}">
        {(user.displayName || 'U')[0].toUpperCase()}
        {#if user.sos?.active}
          <span class="qa-sos-ring" aria-hidden="true"></span>
        {/if}
      </div>
      <div class="qa-user-info">
        <strong class="qa-user-name">{user.displayName}</strong>
        <span class="qa-user-status" class:qa-status-sos={user.sos?.active} class:qa-status-offline={user.online === false}>
          {user.sos?.active ? 'SOS Active' : user.online === false ? 'Offline' : 'Online'}
        </span>
      </div>
    </div>

    <!-- Actions -->
    <div class="qa-actions" role="group" aria-label="Quick actions">
      <button class="qa-action-btn" onclick={() => dispatch('locate', user.socketId)} disabled={!user.latitude}>
        <div class="qa-action-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
        </div>
        <span>Locate on Map</span>
      </button>

      {#if user.userId}
        <button class="qa-action-btn" onclick={() => dispatch('secretChat', { id: user.userId, name: user.displayName })}>
          <div class="qa-action-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
          </div>
          <span>Chat</span>
        </button>
      {/if}

      <button class="qa-action-btn" onclick={qaCopy} disabled={!user.latitude}>
        <div class="qa-action-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
        </div>
        <span>Copy Coordinates</span>
      </button>
    </div>

    <button class="qa-cancel-btn" onclick={() => dispatch('close')}>Cancel</button>
  </div>
{/if}

<style>
  /* ── Quick actions sheet (long-press) ───────────────────────────────────── */
  .qa-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    backdrop-filter: blur(3px);
    -webkit-backdrop-filter: blur(3px);
    /* Was z:5400 hardcoded. Using --z-modal so sheet layer is tokenized. */
    z-index: var(--z-modal, 5000);
    touch-action: none;
  }

  .qa-sheet {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    /* Was z:5401 hardcoded. Using --z-modal + 1 to sit above its backdrop. */
    z-index: calc(var(--z-modal, 5000) + 1);
    /* TECHNIQUE 6: Liquid Glass 2.0 — stronger blur + saturation + brightness */
    background: var(--surface-2, rgba(12, 12, 28, 0.88));
    backdrop-filter: blur(32px) saturate(180%) brightness(1.06);
    -webkit-backdrop-filter: blur(32px) saturate(180%) brightness(1.06);
    border-top: 1px solid rgba(255, 255, 255, 0.14);
    border-radius: 20px 20px 0 0;
    box-shadow:
      0 -8px 48px rgba(0, 0, 0, 0.40),
      0 -1px 0 rgba(255, 255, 255, 0.10),
      inset 0 1px 0 rgba(255, 255, 255, 0.08),
      inset 0 -1px 0 rgba(0, 0, 0, 0.10);
    padding: 8px 16px calc(24px + env(safe-area-inset-bottom, 0px));
    will-change: transform;
  }

  .qa-handle {
    width: 40px;
    height: 5px;
    background: var(--gray-400, rgba(255,255,255,0.22));
    border-radius: 999px;
    margin: 4px auto 16px;
  }

  .qa-user-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 4px 16px;
    border-bottom: 1px solid var(--border-subtle, rgba(255,255,255,0.07));
    margin-bottom: 8px;
  }

  .qa-avatar {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 1.125rem;
    flex-shrink: 0;
    position: relative;
  }

  .qa-sos-ring {
    position: absolute;
    inset: -3px;
    border-radius: 50%;
    box-shadow: 0 0 0 2.5px var(--danger-500), 0 0 12px rgba(239,68,68,0.5);
    animation: sos-urgent-pulse 1s ease-in-out infinite;
    pointer-events: none;
  }

  .qa-user-info {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }

  .qa-user-name {
    font-family: var(--font-display);
    font-size: var(--text-base, 16px);
    font-weight: 700;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .qa-user-status {
    font-size: var(--text-xs, 12px);
    color: var(--success-500, #10b981);
    font-weight: 600;
  }

  .qa-status-offline { color: var(--text-tertiary); }
  .qa-status-sos { color: var(--danger-500, #ef4444); }

  .qa-actions {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 8px;
  }

  .qa-action-btn {
    display: flex;
    align-items: center;
    gap: 14px;
    width: 100%;
    padding: 14px 16px;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-primary);
    font-family: var(--font-sans);
    font-size: var(--text-base, 16px);
    font-weight: 500;
    text-align: left;
    border-radius: var(--radius-lg, 12px);
    transition: background var(--duration-fast, 120ms) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }

  .qa-action-btn:hover {
    background: var(--surface-hover, rgba(255,255,255,0.06));
  }

  .qa-action-btn:active {
    background: var(--surface-active, rgba(255,255,255,0.10));
    transform: scale(0.98);
    transition-duration: 60ms;
  }

  .qa-action-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .qa-action-icon {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: rgba(99,102,241,0.14);
    border: 1px solid rgba(99,102,241,0.22);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--primary-400, #818cf8);
    flex-shrink: 0;
  }

  .qa-cancel-btn {
    display: block;
    width: 100%;
    padding: 15px;
    background: var(--surface-inset, rgba(255,255,255,0.04));
    border: 1px solid var(--border-subtle, rgba(255,255,255,0.08));
    border-radius: var(--radius-lg, 12px);
    color: var(--text-secondary);
    font-family: var(--font-sans);
    font-size: var(--text-base, 16px);
    font-weight: 600;
    cursor: pointer;
    text-align: center;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    margin-top: 4px;
  }

  .qa-cancel-btn:hover { background: var(--surface-hover); }
  .qa-cancel-btn:active { transform: scale(0.98); transition-duration: 60ms; }

  @media (prefers-reduced-motion: reduce) {
    .qa-sos-ring { animation: none; }
  }
</style>
