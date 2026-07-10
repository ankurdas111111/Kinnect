<script>
  import { fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { toasts } from '../../lib/stores/toast.js';

  function typeIcon(type) {
    if (type === 'success') return '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>';
    if (type === 'error') return '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>';
    if (type === 'warning') return '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>';
    return '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>';
  }

  // ── Swipe-to-dismiss (horizontal pointer drag) ──────────────────────────
  const _drag = new Map();

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
      out:fade={{ duration: 150 }}
      role="status"
      onpointerdown={(e) => onSwipeDown(e, toast.id)}
      onpointermove={onSwipeMove}
      onpointerup={(e) => onSwipeUp(e, toast.id)}
      onpointercancel={onSwipeCancel}
    >
      <span class="toast-icon" aria-hidden="true">{@html typeIcon(toast.type)}</span>
      <span class="toast-message">{toast.message}</span>
      <button class="toast-close" onclick={() => toasts.remove(toast.id)} aria-label="Dismiss">
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
    max-width: min(85vw, 380px);
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

  /* 2026 premium toast — glass surface + neon accent + spring entrance */
  .toast {
    cursor: grab;
    user-select: none;
    -webkit-user-select: none;
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius-lg);
    /* Deep glass with richer backdrop */
    background: var(--glass-bg-strong, rgba(9, 10, 22, 0.95));
    border: 1px solid var(--glass-3d-border, rgba(255,255,255,0.10));
    border-top-color: rgba(255,255,255,0.14);
    box-shadow:
      var(--elevation-4),
      inset 0 1px 0 rgba(255, 255, 255, 0.08),
      inset 0 -1px 0 rgba(0, 0, 0, 0.18);
    backdrop-filter: blur(28px) saturate(1.9);
    -webkit-backdrop-filter: blur(28px) saturate(1.9);
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text-primary);
    pointer-events: auto;
    position: relative;
    overflow: hidden;
    /* Neon left accent bar */
    border-left: 3px solid transparent;
    /* 2026 spring entrance */
    animation: toast-spring-in 360ms cubic-bezier(0.34, 1.56, 0.64, 1) both;
  }

  /* Shimmer sweep on new toast */
  .toast::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(
      105deg,
      transparent 20%,
      rgba(255,255,255,0.07) 45%,
      rgba(255,255,255,0.05) 55%,
      transparent 80%
    );
    transform: translateX(-120%);
    pointer-events: none;
    animation: holo-travel 1.8s ease-out 200ms both;
    border-radius: inherit;
  }

  /* Type-specific neon left accent + glow */
  .toast-info {
    border-left-color: var(--primary-500);
    box-shadow:
      var(--elevation-4),
      inset 0 1px 0 rgba(255,255,255,0.08),
      -2px 0 12px rgba(20,184,166,0.25);
  }
  .toast-success {
    border-left-color: var(--success-500);
    box-shadow:
      var(--elevation-4),
      inset 0 1px 0 rgba(255,255,255,0.08),
      -2px 0 14px rgba(16,185,129,0.30);
  }
  .toast-error {
    border-left-color: var(--danger-500);
    box-shadow:
      var(--elevation-4),
      inset 0 1px 0 rgba(255,255,255,0.08),
      -2px 0 14px rgba(239,68,68,0.32);
  }
  .toast-warning {
    border-left-color: var(--warning-500);
    box-shadow:
      var(--elevation-4),
      inset 0 1px 0 rgba(255,255,255,0.08),
      -2px 0 12px rgba(245,158,11,0.28);
  }

  .toast-icon {
    flex-shrink: 0;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-sm);
  }

  .toast-info .toast-icon    {
    background: rgba(20,184,166,0.16);
    color: var(--primary-400);
    box-shadow: 0 0 8px rgba(20,184,166,0.20);
  }
  .toast-success .toast-icon {
    background: rgba(16,185,129,0.16);
    color: var(--success-400);
    box-shadow: 0 0 8px rgba(16,185,129,0.22);
  }
  .toast-error .toast-icon   {
    background: rgba(239,68,68,0.16);
    color: var(--danger-400);
    box-shadow: 0 0 8px rgba(239,68,68,0.24);
  }
  .toast-warning .toast-icon {
    background: rgba(245,158,11,0.16);
    color: var(--warning-400);
    box-shadow: 0 0 8px rgba(245,158,11,0.20);
  }

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

  /* Neon progress bar at bottom */
  .toast-progress {
    position: absolute;
    bottom: 0;
    left: 3px;
    right: 0;
    height: 2px;
    transform-origin: left;
    animation: toast-progress linear forwards;
    border-radius: 0 0 var(--radius-lg) 0;
  }

  .toast-info .toast-progress    {
    background: linear-gradient(90deg, var(--primary-500), var(--primary-400));
    box-shadow: 0 0 4px rgba(20,184,166,0.50);
  }
  .toast-success .toast-progress {
    background: linear-gradient(90deg, var(--success-500), var(--success-400));
    box-shadow: 0 0 4px rgba(16,185,129,0.50);
  }
  .toast-error .toast-progress   {
    background: linear-gradient(90deg, var(--danger-500), var(--danger-400));
    box-shadow: 0 0 4px rgba(239,68,68,0.50);
  }
  .toast-warning .toast-progress {
    background: linear-gradient(90deg, var(--warning-500), var(--warning-400));
    box-shadow: 0 0 4px rgba(245,158,11,0.50);
  }

  @keyframes toast-progress {
    from { transform: scaleX(1); }
    to   { transform: scaleX(0); }
  }

  @media (prefers-reduced-motion: reduce) {
    .toast {
      animation: none;
    }
    .toast::after {
      animation: none;
    }
  }
</style>
