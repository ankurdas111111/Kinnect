<script>
  import { run } from 'svelte/legacy';

  import { createEventDispatcher, onMount, onDestroy, tick } from 'svelte';
  import { fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';

  /**
   * @typedef {Object} Props
   * @property {boolean} [open]
   * @property {boolean} [urgent]
   * @property {string} [title]
   * @property {string} [size]
   * @property {import('svelte').Snippet} [children]
   * @property {import('svelte').Snippet} [footer]
   */

  /** @type {Props} */
  let {
    open = false,
    urgent = false,
    title = '',
    size = 'md',
    children,
    footer
  } = $props();

  // Stable ID for aria-labelledby — derived from title, ASCII-safe fallback
  let titleId = $derived(title
    ? 'modal-title-' + title.toLowerCase().replace(/[^a-z0-9]+/g, '-').slice(0, 32)
    : null);

  const dispatch = createEventDispatcher();
  let dialogEl = $state();
  let lastFocusedEl = $state(null);
  let wasOpen = $state(false);

  function dismiss() {
    dispatch('close');
  }

  function onKeydown(e) {
    if (!open) return;
    if (e.key === 'Escape') {
      e.preventDefault();
      dismiss();
      return;
    }
    if (e.key !== 'Tab' || !dialogEl) return;
    var focusable = getFocusable();
    if (!focusable.length) return;
    var first = focusable[0];
    var last = focusable[focusable.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }

  function onBackdropClick(e) {
    if (e.target === e.currentTarget) dismiss();
  }

  onMount(() => {
    window.addEventListener('keydown', onKeydown);
    if (open) document.body.style.overflow = 'hidden';
  });

  onDestroy(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('keydown', onKeydown);
      document.body.style.overflow = '';
    }
  });

  run(() => {
    if (typeof document !== 'undefined') {
      document.body.style.overflow = open ? 'hidden' : '';
    }
  });

  function getFocusable() {
    if (!dialogEl) return [];
    return Array.from(dialogEl.querySelectorAll(
      'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
    ));
  }

  run(() => {
    if (open && !wasOpen) {
      wasOpen = true;
      lastFocusedEl = document.activeElement;
      tick().then(() => {
        var focusable = getFocusable();
        if (focusable.length) focusable[0].focus();
        else dialogEl?.focus();
      });
    }
  });

  run(() => {
    if (!open && wasOpen) {
      wasOpen = false;
      if (lastFocusedEl && typeof lastFocusedEl.focus === 'function') {
        lastFocusedEl.focus();
      }
    }
  });
</script>

{#if open}
  <div
    class="modal-backdrop"
    class:urgent
    onclick={onBackdropClick}
    transition:fade={{ duration: 180 }}
    role="presentation"
  >
    <div
      class="modal-card {size}"
      class:urgent
      bind:this={dialogEl}
      tabindex="-1"
      role="dialog"
      aria-modal="true"
      aria-labelledby={title ? titleId : undefined}
      aria-label={title ? undefined : 'Modal dialog'}
    >
      {#if title}
        <div class="modal-header">
          <h3 id={titleId} class="modal-title" class:urgent>{title}</h3>
          <button class="btn btn-icon btn-ghost modal-close" onclick={dismiss} aria-label="Close">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>
      {/if}
      <div class="modal-body">
        {@render children?.()}
      </div>
      {#if footer}
        <div class="modal-footer">
          {@render footer?.()}
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  /* Hearth: an ink veil over the paper room — warm, never pure black. */
  .modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-modal, 5000);
    background: color-mix(in oklch, var(--ink) 52%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4);
    backdrop-filter: blur(6px);
    -webkit-backdrop-filter: blur(6px);
    overscroll-behavior: none;
  }

  /* An urgent modal is an SOS. It outranks EVERYTHING, including the
     first-run onboarding overlay at --z-topmost, which was covering "someone
     needs help" with "Give your family a name" for any new user. */
  /* Urgent = SOS: the veil warms toward vermilion — the one red in the app. */
  .modal-backdrop.urgent {
    z-index: calc(var(--z-topmost, 9000) + 10);
    background: color-mix(in oklch, var(--danger-700) 22%, transparent);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
  }

  /* Warm paper sheet — spring entrance kept (modal-3d-arrive). */
  .modal-card {
    background: var(--surface-1);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-xl);
    box-shadow: var(--shadow-xl);
    overflow-y: auto;
    max-height: min(85dvh, 40rem);
    display: flex;
    flex-direction: column;
    animation: modal-3d-arrive 480ms cubic-bezier(0.34, 1.56, 0.64, 1) both;
    transform-style: preserve-3d;
    position: relative;
  }

  .modal-card.sm { width: min(90vw, 340px); max-width: 100%; }
  .modal-card.md { width: min(90vw, 420px); max-width: 100%; }
  .modal-card.lg { width: min(90vw, 560px); max-width: 100%; }

  .modal-card.urgent {
    border-color: color-mix(in oklch, var(--danger-500) 40%, transparent);
    box-shadow: var(--shadow-xl), var(--shadow-danger);
  }

  /* SOS top edge — quiet vermilion hairline, no neon sweep. */
  .modal-card.urgent::before {
    content: '';
    position: absolute;
    top: 0; left: 10%; right: 10%;
    height: 2px;
    background: var(--danger-500);
    border-radius: 0 0 2px 2px;
    pointer-events: none;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-4) var(--space-6) 0;
  }

  .modal-title {
    font-family: var(--font-display);
    font-size: var(--text-xl);
    font-weight: 600;
    color: var(--text-primary);
    letter-spacing: -0.01em;
    margin: 0;
  }

  .modal-title.urgent {
    color: var(--danger-600);
  }

  .modal-close {
    flex-shrink: 0;
  }

  .modal-body {
    padding: var(--space-4) var(--space-6);
    overflow-y: auto;
    flex: 1;
    -webkit-overflow-scrolling: touch;
  }

  .modal-footer {
    display: flex;
    gap: var(--space-3);
    justify-content: flex-end;
    padding: 0 var(--space-6) var(--space-4);
    flex-wrap: wrap;
  }

  @media (max-width: 767px) {
    .modal-card {
      width: calc(100% - var(--space-4));
      max-width: 100%;
    }
    .modal-body {
      padding: var(--space-3) var(--space-4);
    }
    .modal-header {
      padding: var(--space-3) var(--space-4) 0;
    }
    .modal-footer {
      padding: 0 var(--space-4) var(--space-3);
      justify-content: stretch;
    }
    .modal-footer :global(.btn) {
      flex: 1;
      min-height: 48px;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .modal-card {
      animation: none;
    }
    .modal-card.urgent {
      animation: none;
      box-shadow: var(--shadow-xl), 0 0 0 2px var(--danger-500);
    }
  }
</style>
