<script>
  import { createEventDispatcher, onMount, onDestroy, tick } from 'svelte';

  export let open = false;
  export let title = '';

  const dispatch = createEventDispatcher();

  let sheetEl;
  let dragging = false;
  let startY = 0;
  let currentOffset = 0;
  let snapState = 'peek';
  let lastFocusedEl = null;
  let wasOpen = false;

  const SNAP_PEEK = 0.65;   // 35% of screen visible from bottom
  const SNAP_HALF = 0.45;
  const SNAP_FULL = 0.08;

  $: viewH = typeof window !== 'undefined' ? window.innerHeight : 800;
  $: peekY = viewH * SNAP_PEEK;
  $: halfY = viewH * SNAP_HALF;
  $: fullY = viewH * SNAP_FULL;

  $: if (open && snapState === 'closed') {
    snapState = 'peek';
    currentOffset = peekY;
  }

  $: if (!open) {
    snapState = 'closed';
    currentOffset = viewH;
  }

  $: translateY = snapState === 'closed' ? viewH : currentOffset;

  function onPointerDown(e) {
    if (e.target.closest('.sheet-body')) return;
    dragging = true;
    startY = e.clientY;
    if (sheetEl) sheetEl.style.transition = 'none';
  }

  function applyRubberBand(offset) {
    if (offset < fullY) {
      const overscroll = fullY - offset;
      return fullY - overscroll * 0.3;
    }
    if (offset > viewH) {
      const overscroll = offset - viewH;
      return viewH + overscroll * 0.3;
    }
    return offset;
  }

  function onPointerMove(e) {
    if (!dragging) return;
    const delta = e.clientY - startY;
    const rawOffset = currentOffset + delta;
    const rubber = applyRubberBand(rawOffset);
    if (sheetEl) sheetEl.style.transform = `translateY(${rubber}px)`;
  }

  function onPointerUp(e) {
    if (!dragging) return;
    dragging = false;
    if (sheetEl) sheetEl.style.transition = '';

    const delta = e.clientY - startY;
    const rawOffset = currentOffset + delta;

    if (delta > 80) {
      if (snapState === 'full') { snap('half'); }
      else if (snapState === 'half') { snap('peek'); }
      else { dismiss(); }
    } else if (delta < -80) {
      if (snapState === 'peek') { snap('half'); }
      else if (snapState === 'half') { snap('full'); }
    } else {
      snap(snapState);
    }
  }

  function snap(state) {
    snapState = state;
    if (state === 'peek') currentOffset = peekY;
    else if (state === 'half') currentOffset = halfY;
    else if (state === 'full') currentOffset = fullY;
  }

  function dismiss() {
    snapState = 'closed';
    currentOffset = viewH;
    dispatch('close');
  }

  function onBackdropClick() {
    dismiss();
  }

  function handleResize() {
    viewH = window.innerHeight;
    snap(snapState);
  }

  function getFocusable() {
    if (!sheetEl) return [];
    return Array.from(sheetEl.querySelectorAll(
      'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
    ));
  }

  function onKeydown(e) {
    if (!open) return;
    if (e.key === 'Escape') {
      e.preventDefault();
      dismiss();
      return;
    }
    if (e.key !== 'Tab') return;
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

  onMount(() => {
    window.addEventListener('resize', handleResize);
    window.addEventListener('keydown', onKeydown);
    if (open) {
      snapState = 'peek';
      currentOffset = peekY;
    }
  });

  onDestroy(() => {
    if (typeof window !== 'undefined') window.removeEventListener('resize', handleResize);
    if (typeof window !== 'undefined') window.removeEventListener('keydown', onKeydown);
  });

  $: if (open && !wasOpen) {
    wasOpen = true;
    lastFocusedEl = document.activeElement;
    tick().then(() => {
      var focusable = getFocusable();
      if (focusable.length) focusable[0].focus();
      else sheetEl?.focus();
    });
  }

  $: if (!open && wasOpen) {
    wasOpen = false;
    if (lastFocusedEl && typeof lastFocusedEl.focus === 'function') {
      lastFocusedEl.focus();
    }
  }
</script>

{#if open}
  <div class="sheet-backdrop" class:visible={snapState !== 'closed'} on:click={onBackdropClick} aria-hidden="true"></div>
  <div
    class="sheet"
    bind:this={sheetEl}
    tabindex="-1"
    style="transform: translateY({translateY}px)"
    on:pointerdown={onPointerDown}
    on:pointermove={onPointerMove}
    on:pointerup={onPointerUp}
    role="dialog"
    aria-modal="true"
    aria-label={title || 'Bottom sheet'}
  >
    <div class="sheet-handle-area">
      <div class="sheet-handle"></div>
    </div>
    {#if title}
      <div class="sheet-header">
        <h3>{title}</h3>
        <button class="btn btn-icon btn-ghost" on:click={dismiss} aria-label="Close">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>
    {/if}
    <div class="sheet-body">
      <slot />
    </div>
  </div>
{/if}

<style>
  .sheet-backdrop {
    position: fixed;
    inset: 0;
    z-index: calc(var(--z-panel) - 1);
    background: rgba(0, 0, 0, 0.48);
    opacity: 0;
    transition: opacity var(--duration-normal) var(--ease-out);
    touch-action: none;
  }

  .sheet-backdrop.visible {
    opacity: 1;
  }

  .sheet {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    height: 90vh;
    z-index: var(--z-panel);
    /* 3D glass sheet with depth */
    background: var(--glass-3d);
    border-radius: var(--radius-sheet) var(--radius-sheet) 0 0;
    border: 1px solid var(--glass-3d-border);
    border-bottom: none;
    border-top-color: rgba(255, 255, 255, 0.20);
    box-shadow:
      var(--elevation-5),
      inset 0 1px 0 rgba(255, 255, 255, 0.15),
      inset 0 -1px 0 rgba(0, 0, 0, 0.05);
    backdrop-filter: var(--glass-3d-blur);
    -webkit-backdrop-filter: var(--glass-3d-blur);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    transition: transform 0.42s cubic-bezier(0.34, 1.56, 0.64, 1);
    will-change: transform;
    touch-action: none;
    transform-style: preserve-3d;
  }

  .sheet-handle-area {
    padding: var(--space-2) 0 var(--space-1);
    display: flex;
    justify-content: center;
    cursor: grab;
    flex-shrink: 0;
    touch-action: none;
    min-height: 28px;
  }

  .sheet-handle-area:active {
    cursor: grabbing;
  }

  .sheet-handle {
    width: 48px;
    height: 6px;
    background: var(--gray-300);
    border-radius: 999px;
    /* 3D raised handle */
    box-shadow:
      0 1px 3px rgba(0, 0, 0, 0.12),
      inset 0 1px 0 rgba(255, 255, 255, 0.25);
  }

  :global([data-theme="dark"]) .sheet-handle {
    background: rgba(255, 255, 255, 0.25);
    box-shadow:
      0 1px 3px rgba(0, 0, 0, 0.30),
      inset 0 1px 0 rgba(255, 255, 255, 0.10);
  }

  .sheet-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0 var(--space-4) var(--space-2);
    flex-shrink: 0;
  }

  .sheet-header h3 {
    font-family: var(--font-display);
    font-size: var(--text-lg);
    font-weight: 700;
    margin: 0;
    letter-spacing: -0.015em;
  }

  .sheet-body {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
    overscroll-behavior: contain;
    -webkit-overflow-scrolling: touch;
    touch-action: pan-y;
    padding: 0 var(--space-4) var(--space-4);
    /* Safe area + bottom tab bar height so content isn't hidden behind the tab bar */
    padding-bottom: calc(var(--space-4) + var(--safe-bottom, 0px) + var(--bottom-tab-height, 56px));
  }

  @media (min-width: 768px) {
    .sheet-backdrop, .sheet {
      display: none;
    }
  }
</style>
