<script>
  import { createEventDispatcher } from 'svelte';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';

  
  /**
   * @typedef {Object} Props
   * @property {any} [user] - User pending delete confirmation; null hides the sheet.
   */

  /** @type {Props} */
  let { user = null } = $props();

  const dispatch = createEventDispatcher();
</script>

<!-- ── Admin delete confirmation dialog (non-blocking, accessible) ─────── -->
{#if user}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="qa-backdrop" onclick={() => dispatch('cancel')} aria-hidden="true" transition:fade={{ duration: 150 }}></div>
  <div
    class="qa-sheet delete-confirm-sheet"
    role="alertdialog"
    aria-label="Confirm delete {user.displayName}"
    aria-modal="true"
    transition:fly={{ y: 120, duration: 250, easing: cubicOut }}
  >
    <div class="qa-handle" aria-hidden="true"></div>
    <p class="delete-confirm-text">
      Remove <strong>{user.displayName}</strong> from this session? This will disconnect them.
    </p>
    <div class="delete-confirm-actions">
      <button class="qa-cancel-btn" onclick={() => dispatch('cancel')}>Cancel</button>
      <button class="delete-confirm-btn" onclick={() => dispatch('confirm')}>Remove</button>
    </div>
  </div>
{/if}

<style>
  .qa-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    backdrop-filter: blur(3px);
    -webkit-backdrop-filter: blur(3px);
    z-index: var(--z-modal, 5000);
    touch-action: none;
  }

  .qa-sheet {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: calc(var(--z-modal, 5000) + 1);
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

  /* ── Admin delete confirmation ─────────────────────────────────────────── */
  .delete-confirm-sheet {
    padding-bottom: calc(20px + env(safe-area-inset-bottom, 0px));
  }

  .delete-confirm-text {
    font-size: var(--text-base);
    color: var(--text-primary);
    text-align: center;
    margin: 0 0 var(--space-4);
    line-height: var(--leading-relaxed);
    padding: 0 var(--space-2);
  }

  .delete-confirm-actions {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-3);
  }

  .delete-confirm-btn {
    padding: 14px;
    background: rgba(239, 68, 68, 0.12);
    border: 1px solid rgba(239, 68, 68, 0.28);
    border-radius: var(--radius-lg);
    color: var(--danger-400, #f87171);
    font-family: var(--font-sans);
    font-size: var(--text-base);
    font-weight: 700;
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    min-height: 44px;
  }
  .delete-confirm-btn:hover { background: rgba(239, 68, 68, 0.20); }
  .delete-confirm-btn:active { transform: scale(0.97); transition-duration: 60ms; }
</style>
