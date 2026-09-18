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
    background: color-mix(in oklch, var(--ink) 40%, transparent);
    z-index: var(--z-modal, 5000);
    touch-action: none;
  }

  .qa-sheet {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: calc(var(--z-modal, 5000) + 1);
    background: var(--card);
    border-top: 1px solid var(--border-subtle);
    border-radius: var(--radius-sheet) var(--radius-sheet) 0 0;
    box-shadow: var(--sh-up);
    padding: var(--space-2) var(--space-4);
    padding-bottom: max(var(--space-6), env(safe-area-inset-bottom));
    will-change: transform;
  }

  .qa-handle {
    width: 36px;
    height: 4px;
    background: var(--border-strong);
    border-radius: var(--radius-full);
    margin: var(--space-1) auto var(--space-4);
  }

  .qa-cancel-btn {
    display: block;
    width: 100%;
    min-height: 44px;
    padding: var(--space-3-5);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    color: var(--text-secondary);
    font-family: var(--font-sans);
    font-size: var(--text-base);
    font-weight: 600;
    cursor: pointer;
    text-align: center;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    margin-top: var(--space-1);
  }

  .qa-cancel-btn:hover { background: var(--surface-hover); }
  .qa-cancel-btn:active { transform: scale(0.98); transition-duration: var(--duration-fast); }

  /* ── Admin delete confirmation ─────────────────────────────────────────── */
  .delete-confirm-sheet {
    padding-bottom: max(var(--space-5), env(safe-area-inset-bottom));
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

  /* Destructive ≠ vermilion (reserved for SOS). Ochre carries "pause and
     look" weight for the remove confirmation. */
  .delete-confirm-btn {
    padding: var(--space-3-5);
    background: color-mix(in oklch, var(--warning-500) 14%, transparent);
    border: 1px solid color-mix(in oklch, var(--warning-500) 30%, transparent);
    border-radius: var(--radius-md);
    color: var(--warning-700);
    font-family: var(--font-sans);
    font-size: var(--text-base);
    font-weight: 700;
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    min-height: 44px;
  }
  .delete-confirm-btn:hover { background: color-mix(in oklch, var(--warning-500) 22%, transparent); }
  .delete-confirm-btn:active { transform: scale(0.97); transition-duration: var(--duration-fast); }
</style>
