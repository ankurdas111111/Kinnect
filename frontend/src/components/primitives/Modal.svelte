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
  .modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-modal, 5000);
    background: rgba(0, 0, 0, 0.62);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4);
    backdrop-filter: blur(8px) saturate(1.4);
    -webkit-backdrop-filter: blur(8px) saturate(1.4);
    overscroll-behavior: none;
  }

  .modal-backdrop.urgent {
    background: rgba(185, 28, 28, 0.22);
    backdrop-filter: blur(10px) saturate(1.6);
    -webkit-backdrop-filter: blur(10px) saturate(1.6);
  }

  /* 3D spring entrance — rises from depth plane */
  .modal-card {
    background: var(--glass-3d, rgba(15, 15, 30, 0.85));
    border: 1px solid var(--glass-3d-border, rgba(255,255,255,0.10));
    border-top-color: rgba(255,255,255,0.16);
    border-radius: var(--radius-xl);
    box-shadow:
      var(--elevation-4),
      var(--glass-3d-inner),
      0 0 0 1px rgba(255,255,255,0.04);
    backdrop-filter: var(--glass-3d-blur, blur(28px) saturate(1.8));
    -webkit-backdrop-filter: var(--glass-3d-blur, blur(28px) saturate(1.8));
    overflow: hidden;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    /* 3D spring entrance animation */
    animation: modal-3d-arrive 480ms cubic-bezier(0.34, 1.56, 0.64, 1) both;
    transform-style: preserve-3d;
    position: relative;
  }

  /* Subtle top-edge glow line */
  .modal-card::before {
    content: '';
    position: absolute;
    top: 0; left: 10%; right: 10%;
    height: 1px;
    background: linear-gradient(
      90deg,
      transparent 0%,
      rgba(20, 184, 166, 0.70) 35%,
      rgba(16, 185, 129, 0.60) 65%,
      transparent 100%
    );
    box-shadow: 0 0 8px rgba(20, 184, 166, 0.40);
    border-radius: 0 0 2px 2px;
    pointer-events: none;
  }

  .modal-card.sm { width: 340px; max-width: 100%; }
  .modal-card.md { width: 420px; max-width: 100%; }
  .modal-card.lg { width: 560px; max-width: 100%; }

  .modal-card.urgent {
    border-color: rgba(239, 68, 68, 0.40);
    border-top-color: rgba(239, 68, 68, 0.55);
    animation: modal-3d-arrive 480ms cubic-bezier(0.34, 1.56, 0.64, 1) both,
               urgent-glow-pulse 2.2s ease-in-out 500ms infinite;
  }

  .modal-card.urgent::before {
    background: linear-gradient(
      90deg,
      transparent 0%,
      rgba(239, 68, 68, 0.80) 35%,
      rgba(239, 68, 68, 0.65) 65%,
      transparent 100%
    );
    box-shadow: 0 0 12px rgba(239, 68, 68, 0.55);
  }

  @keyframes urgent-glow-pulse {
    0%, 100% {
      box-shadow: var(--elevation-4), var(--glass-3d-inner), 0 0 0 1px rgba(239,68,68,0.25);
    }
    50% {
      box-shadow: var(--elevation-4), var(--glass-3d-inner),
                  0 0 0 1px rgba(239,68,68,0.45),
                  0 0 24px rgba(239,68,68,0.22),
                  0 0 48px rgba(239,68,68,0.10);
    }
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
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.02em;
    margin: 0;
  }

  .modal-title.urgent {
    color: var(--danger-400);
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
      box-shadow: var(--elevation-4), 0 0 0 2px var(--danger-500);
    }
  }
</style>
