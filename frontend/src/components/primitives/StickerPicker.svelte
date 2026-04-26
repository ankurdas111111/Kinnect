<script>
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  export let open = false;
  export let anchor = null;

  let wrapEl;
  let activeTab = 'flirty';

  const CDN = 'https://fonts.gstatic.com/s/e/notoemoji/latest';

  const TABS = [
    { id: 'flirty', label: '❤️ Flirty' },
    { id: 'spicy',  label: '🔥 Spicy'  },
    { id: 'fun',    label: '😂 Fun'    },
  ];

  const STICKERS = {
    flirty: [
      { hex: '1f970', alt: '🥰' },
      { hex: '1f60d', alt: '😍' },
      { hex: '1f618', alt: '😘' },
      { hex: '1f48b', alt: '💋' },
      { hex: '2764_fe0f', alt: '❤️' },
      { hex: '1f495', alt: '💕' },
      { hex: '1f49e', alt: '💞' },
      { hex: '1f497', alt: '💗' },
      { hex: '1fae6', alt: '🫦' },
      { hex: '1f609', alt: '😉' },
      { hex: '1f979', alt: '🥹' },
      { hex: '1f48c', alt: '💌' },
      { hex: '1f60f', alt: '😏' },
      { hex: '1f61c', alt: '😜' },
      { hex: '1fae0', alt: '🫠' },
      { hex: '1f491', alt: '💑' },
      { hex: '1f48f', alt: '💏' },
      { hex: '1f444', alt: '👄' },
      { hex: '1f493', alt: '💓' },
      { hex: '1fac0', alt: '🫀' },
    ],
    spicy: [
      { hex: '1f525', alt: '🔥' },
      { hex: '1f975', alt: '🥵' },
      { hex: '1f4a6', alt: '💦' },
      { hex: '1f346', alt: '🍆' },
      { hex: '1f351', alt: '🍑' },
      { hex: '1f353', alt: '🍓' },
      { hex: '1f60b', alt: '😋' },
      { hex: '1f924', alt: '🤤' },
      { hex: '1f608', alt: '😈' },
      { hex: '1f445', alt: '👅' },
      { hex: '1fae6', alt: '🫦' },
      { hex: '1f336', alt: '🌶️' },
      { hex: '1f36d', alt: '🍭' },
      { hex: '1f36f', alt: '🍯' },
      { hex: '1f352', alt: '🍒' },
      { hex: '1f9e8', alt: '🧨' },
      { hex: '1f4a5', alt: '💥' },
      { hex: '1f386', alt: '🎆' },
      { hex: '1f624', alt: '😤' },
      { hex: '1f929', alt: '🤩' },
    ],
    fun: [
      { hex: '1f602', alt: '😂' },
      { hex: '1f973', alt: '🥳' },
      { hex: '1f389', alt: '🎉' },
      { hex: '1f648', alt: '🙈' },
      { hex: '1f649', alt: '🙉' },
      { hex: '1f64a', alt: '🙊' },
      { hex: '1f974', alt: '🥴' },
      { hex: '1f62d', alt: '😭' },
      { hex: '1f923', alt: '🤣' },
      { hex: '1f92f', alt: '🤯' },
      { hex: '1fae3', alt: '🫣' },
      { hex: '1f92d', alt: '🤭' },
      { hex: '1f631', alt: '😱' },
      { hex: '1fae1', alt: '🫡' },
      { hex: '1f978', alt: '🥸' },
      { hex: '1f480', alt: '💀' },
      { hex: '1faf6', alt: '🫶' },
      { hex: '1f918', alt: '🤘' },
      { hex: '1fac2', alt: '🫂' },
      { hex: '1f3ad', alt: '🎭' },
    ],
  };

  function gifUrl(hex) {
    return `${CDN}/${hex}/512.gif`;
  }

  function pick(hex) {
    const url = gifUrl(hex);
    dispatch('pick', `[gif:${url}]`);
  }

  function reposition() {
    if (!wrapEl || !anchor) return;
    const ab = anchor.getBoundingClientRect();
    const pw = wrapEl.offsetWidth  || 280;
    const ph = wrapEl.offsetHeight || 300;
    const vw = window.innerWidth;

    // Prefer above anchor
    let top = ab.top - ph - 8;
    if (top < 8) top = ab.bottom + 8;

    let left = ab.right - pw;
    if (left < 8) left = 8;
    if (left + pw > vw - 8) left = vw - pw - 8;

    wrapEl.style.top  = top  + 'px';
    wrapEl.style.left = left + 'px';
  }

  $: if (open && wrapEl) {
    setTimeout(reposition, 10);
  }

  function onDocPointer(e) {
    if (!open) return;
    if (wrapEl && !wrapEl.contains(e.target) && anchor && !anchor.contains(e.target)) {
      dispatch('close');
    }
  }

  onMount(() => { document.addEventListener('pointerdown', onDocPointer, true); });
  onDestroy(() => { document.removeEventListener('pointerdown', onDocPointer, true); });
</script>

{#if open}
  <div class="sp-wrap" bind:this={wrapEl}>
    <div class="sp-tabs">
      {#each TABS as tab}
        <button
          class="sp-tab"
          class:sp-tab--active={activeTab === tab.id}
          on:click={() => activeTab = tab.id}
          type="button"
        >{tab.label}</button>
      {/each}
    </div>
    <div class="sp-grid">
      {#each STICKERS[activeTab] as s (s.hex)}
        <button
          class="sp-sticker-btn"
          on:click={() => pick(s.hex)}
          type="button"
          title={s.alt}
          aria-label={s.alt}
        >
          <img
            src={gifUrl(s.hex)}
            alt={s.alt}
            loading="lazy"
            class="sp-img"
          />
        </button>
      {/each}
    </div>
  </div>
{/if}

<style>
  .sp-wrap {
    position: fixed;
    z-index: 9999;
    width: 280px;
    height: 300px;
    background: #111118;
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 16px;
    box-shadow: 0 16px 48px rgba(0,0,0,0.7), 0 0 0 1px rgba(129,140,248,0.07);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    animation: sp-pop 0.18s cubic-bezier(0.34,1.56,0.64,1) both;
  }

  .sp-tabs {
    display: flex;
    border-bottom: 1px solid rgba(255,255,255,0.07);
    flex-shrink: 0;
  }

  .sp-tab {
    flex: 1;
    padding: 9px 4px;
    background: none;
    border: none;
    color: rgba(255,255,255,0.35);
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    font-family: system-ui, sans-serif;
    transition: color 0.15s, background 0.15s;
    border-bottom: 2px solid transparent;
    white-space: nowrap;
  }
  .sp-tab:hover { color: rgba(255,255,255,0.65); background: rgba(255,255,255,0.04); }
  .sp-tab--active {
    color: #818cf8;
    border-bottom-color: #818cf8;
  }

  .sp-grid {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: 4px;
    align-content: start;
  }
  .sp-grid::-webkit-scrollbar { width: 3px; }
  .sp-grid::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.08); border-radius: 2px; }

  .sp-sticker-btn {
    width: 100%;
    aspect-ratio: 1;
    background: none;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    padding: 3px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.12s;
    touch-action: manipulation;
  }
  .sp-sticker-btn:hover { background: rgba(255,255,255,0.08); }
  .sp-sticker-btn:active { background: rgba(129,140,248,0.18); }

  .sp-img {
    width: 48px;
    height: 48px;
    object-fit: contain;
    display: block;
    border-radius: 4px;
  }

  @keyframes sp-pop {
    from { opacity: 0; transform: scale(0.88) translateY(8px); }
    to   { opacity: 1; transform: scale(1)    translateY(0);    }
  }
</style>
