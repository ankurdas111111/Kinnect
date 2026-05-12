<script>
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  export let open = false;
  export let anchor = null; // element to position near

  let pickerEl;
  let wrapEl;
  let pickerReady = false;

  // Position the picker above/near the anchor button
  function reposition() {
    if (!wrapEl || !anchor) return;
    const ab = anchor.getBoundingClientRect();
    const pw = wrapEl.offsetWidth  || 320;
    const ph = wrapEl.offsetHeight || 400;
    const vw = window.innerWidth;

    // Prefer above the anchor
    let top = ab.top - ph - 8;
    if (top < 8) top = ab.bottom + 8; // flip below if not enough space

    let left = ab.right - pw;
    if (left < 8) left = 8;
    if (left + pw > vw - 8) left = vw - pw - 8;

    wrapEl.style.top  = top  + 'px';
    wrapEl.style.left = left + 'px';
  }

  onMount(async () => {
    await import('emoji-picker-element');
    pickerReady = true;
  });

  $: if (open && pickerReady && wrapEl) {
    setTimeout(reposition, 10);
  }

  function handlePick(e) {
    const emoji = e.detail?.emoji?.unicode;
    if (emoji) dispatch('pick', emoji);
  }

  // Close on outside click
  function onDocClick(e) {
    if (!open) return;
    if (wrapEl && !wrapEl.contains(e.target) && anchor && !anchor.contains(e.target)) {
      dispatch('close');
    }
  }

  onMount(() => { document.addEventListener('pointerdown', onDocClick, true); });
  onDestroy(() => { document.removeEventListener('pointerdown', onDocClick, true); });
</script>

{#if open && pickerReady}
  <div class="ep-wrap" bind:this={wrapEl}>
    <emoji-picker
      bind:this={pickerEl}
      class="ep-picker"
      on:emoji-click={handlePick}
    ></emoji-picker>
  </div>
{/if}

<style>
  .ep-wrap {
    position: fixed;
    z-index: 9999;
    border-radius: var(--radius-xl, 20px);
    overflow: hidden;
    box-shadow:
      0 16px 48px rgba(0, 0, 0, 0.75),
      0 0 0 1px rgba(20, 184, 166, 0.12),
      inset 0 1px 0 rgba(255, 255, 255, 0.05);
    animation: ep-pop var(--duration-normal, 200ms) var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
  }

  /* Override emoji-picker-element CSS variables to match teal dark theme */
  .ep-wrap :global(emoji-picker) {
    --background:               #0a0a18;
    --border-color:             rgba(255, 255, 255, 0.07);
    --border-size:              1px;
    --button-active-background: rgba(20, 184, 166, 0.18);
    --button-hover-background:  rgba(255, 255, 255, 0.07);
    --category-emoji-padding:   4px;
    --category-emoji-size:      1.6rem;
    --category-font-color:      rgba(255, 255, 255, 0.35);
    --category-font-size:       0.7rem;
    --emoji-padding:            5px;
    --emoji-size:               1.5rem;
    --indicator-color:          #14b8a6;
    --indicator-height:         2px;
    --input-border-color:       rgba(255, 255, 255, 0.10);
    --input-border-radius:      var(--radius-md, 10px);
    --input-border-size:        1px;
    --input-font-color:         rgba(255, 255, 255, 0.90);
    --input-font-size:          14px;
    --input-line-height:        1.5;
    --input-padding:            8px 12px;
    --input-placeholder-color:  rgba(255, 255, 255, 0.28);
    --num-columns:              8;
    --outline-color:            rgba(20, 184, 166, 0.45);
    --outline-size:             2px;
    --skintone-border-radius:   50%;
    --category-padding:         4px 8px;
    width: 320px;
    height: 380px;
  }

  @media (max-width: 400px) {
    .ep-wrap :global(emoji-picker) {
      width: calc(100vw - 32px);
      --num-columns: 7;
    }
  }

  @keyframes ep-pop {
    from { opacity: 0; transform: scale(0.88) translateY(8px); }
    to   { opacity: 1; transform: scale(1)    translateY(0);   }
  }

  @media (prefers-reduced-motion: reduce) {
    .ep-wrap { animation: none; }
  }
</style>
