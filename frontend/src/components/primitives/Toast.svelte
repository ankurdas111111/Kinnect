<script>
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { toasts } from '../../lib/stores/toast.js';

  function typeIcon(type) {
    if (type === 'success') return '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>';
    if (type === 'error') return '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>';
    if (type === 'warning') return '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>';
    return '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>';
  }

  // ── Swipe-to-dismiss (horizontal pointer drag) ──────────────────────────
  // Two-phase: no capture until 5px horizontal movement, then capture to
  // track the full swipe even if pointer leaves the element.
  const _drag = new Map(); // pointerId → { id, el, startX, captured }

  function onSwipeDown(e, id) {
    _drag.set(e.pointerId, { id, el: e.currentTarget, startX: e.clientX, captured: false });
    e.currentTarget.style.transition = 'none';
  }

  function onSwipeMove(e) {
    const d = _drag.get(e.pointerId);
    if (!d) return;
    const dx = e.clientX - d.startX;
    if (!d.captured && Math.abs(dx) > 5) {
      d.captured = true;
      d.el.setPointerCapture(e.pointerId);
    }
    if (!d.captured) return;
    d.el.style.transform = `translateX(${dx}px) rotate(${dx * 0.015}deg)`;
    d.el.style.opacity = String(Math.max(0.2, 1 - Math.abs(dx) / 130));
  }

  function onSwipeUp(e, id) {
    const d = _drag.get(e.pointerId);
    _drag.delete(e.pointerId);
    if (!d) return;
    d.el.style.transition = '';
    const dx = e.clientX - d.startX;
    if (d.captured && Math.abs(dx) > 80) {
      // Fly off in swipe direction then remove
      const dir = dx > 0 ? '110%' : '-110%';
      d.el.style.transition = 'transform 180ms cubic-bezier(0.4,0,1,1), opacity 180ms ease-out';
      d.el.style.transform = `translateX(${dir}) rotate(${dx > 0 ? 6 : -6}deg)`;
      d.el.style.opacity = '0';
      setTimeout(() => toasts.remove(id), 180);
    } else {
      d.el.style.transform = '';
      d.el.style.opacity = '';
    }
  }

  function onSwipeCancel(e) {
    const d = _drag.get(e.pointerId);
    _drag.delete(e.pointerId);
    if (!d) return;
    d.el.style.transition = '';
    d.el.style.transform = '';
    d.el.style.opacity = '';
  }
</script>

<div class="toast-container" aria-live="polite" aria-relevant="additions">
  {#each $toasts as toast (toast.id)}
    <div
      class="toast toast-{toast.type}"
      in:fly={{ y: -30, duration: 250, easing: cubicOut }}
      out:fade={{ duration: 150 }}
      role="status"
      on:pointerdown={(e) => onSwipeDown(e, toast.id)}
      on:pointermove={onSwipeMove}
      on:pointerup={(e) => onSwipeUp(e, toast.id)}
      on:pointercancel={onSwipeCancel}
    >
      <span class="toast-icon" aria-hidden="true">{@html typeIcon(toast.type)}</span>
      <span class="toast-message">{toast.message}</span>
      <button class="toast-close" on:click={() => toasts.remove(toast.id)} aria-label="Dismiss">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
      {#if toast.duration > 0}
        <div class="toast-progress" style="animation-duration:{toast.duration}ms"></div>
      {/if}
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    top: calc(var(--safe-top, 0px) + var(--space-4));
    right: var(--space-4);
    z-index: var(--z-toast, 6000);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    max-width: 380px;
    pointer-events: none;
  }

  @media (max-width: 767px) {
    .toast-container {
      top: calc(var(--safe-top, 0px) + var(--space-3));
      left: var(--space-3);
      right: var(--space-3);
      max-width: none;
    }
  }

  /* Premium toast — glass surface with colored left accent bar */
  .toast {
    cursor: grab;
    user-select: none;
    -webkit-user-select: none;
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius-lg);
    /* Richer glass — stronger backdrop, tinted surface */
    background: var(--glass-bg-strong, rgba(12, 12, 28, 0.94));
    border: 1px solid var(--glass-border);
    box-shadow:
      var(--shadow-xl),
      0 0 0 1px rgba(255, 255, 255, 0.04),
      inset 0 1px 0 rgba(255, 255, 255, 0.07);
    backdrop-filter: blur(24px) saturate(1.8);
    -webkit-backdrop-filter: blur(24px) saturate(1.8);
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text-primary);
    pointer-events: auto;
    position: relative;
    overflow: hidden;
    /* Left accent bar — colored glow instead of top progress bar */
    border-left: 3px solid transparent;
    animation: card-rise 220ms var(--ease-out) both;
  }

  /* Type-specific left accent + icon colors */
  .toast-info    { border-left-color: var(--primary-500); }
  .toast-success { border-left-color: var(--success-500); }
  .toast-error   { border-left-color: var(--danger-500); }
  .toast-warning { border-left-color: var(--warning-500); }

  .toast-icon {
    flex-shrink: 0;
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-sm);
  }

  .toast-info .toast-icon    { background: rgba(99,102,241,0.16);  color: var(--primary-400); }
  .toast-success .toast-icon { background: rgba(16,185,129,0.16);  color: var(--success-400); }
  .toast-error .toast-icon   { background: rgba(239,68,68,0.16);   color: var(--danger-400);  }
  .toast-warning .toast-icon { background: rgba(245,158,11,0.16);  color: var(--warning-400); }

  .toast-message {
    flex: 1;
    min-width: 0;
    line-height: var(--leading-normal);
    color: var(--text-primary);
  }

  .toast-close {
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-tertiary);
    padding: 4px;
    min-width: 44px;
    min-height: 44px;
    border-radius: var(--radius-sm);
    flex-shrink: 0;
    transition:
      color var(--duration-fast) var(--ease-out),
      background var(--duration-fast) var(--ease-out);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .toast-close:hover {
    color: var(--text-primary);
    background: var(--surface-hover);
  }

  /* Progress bar — bottom, 3px, matches accent color */
  .toast-progress {
    position: absolute;
    bottom: 0;
    left: 3px; /* start after left accent border */
    right: 0;
    height: 2px;
    transform-origin: left;
    animation: toast-progress linear forwards;
    opacity: 0.6;
    border-radius: 0 0 var(--radius-lg) 0;
  }

  .toast-info .toast-progress    { background: var(--primary-500); }
  .toast-success .toast-progress { background: var(--success-500); }
  .toast-error .toast-progress   { background: var(--danger-500); }
  .toast-warning .toast-progress { background: var(--warning-500); }

  @keyframes toast-progress {
    from { transform: scaleX(1); }
    to { transform: scaleX(0); }
  }
</style>
