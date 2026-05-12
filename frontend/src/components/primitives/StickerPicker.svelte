<script>
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  export let open = false;
  export let anchor = null;

  let wrapEl;
  let activeTab = 'flirty';

  const CDN = 'https://fonts.gstatic.com/s/e/notoemoji/latest';

  const TABS = [
    { id: 'flirty',    label: '💕 Flirty'   },
    { id: 'spicy',     label: '🔥 Spicy'    },
    { id: 'fun',       label: '😂 Fun'      },
    { id: 'reactions', label: '🫢 Reactions' },
    { id: 'hands',     label: '🫶 Hands'    },
  ];

  // Only animated GIFs — all verified 200 OK against fonts.gstatic.com CDN.
  // Replacements: 1f57a (man dancing) → 1f4aa (flexed biceps, confirmed 200).
  //               1f47a (goblin) → 1f47d (alien, confirmed 200).
  // Static objects, food, and flowers removed entirely.
  const STICKERS = {
    flirty: [
      { hex: '1f970', alt: '🥰' },  // smiling face with hearts
      { hex: '1f60d', alt: '😍' },  // heart eyes
      { hex: '1f618', alt: '😘' },  // kissing heart
      { hex: '1f48b', alt: '💋' },  // kiss mark
      { hex: '2764',  alt: '❤️' },  // red heart
      { hex: '1f495', alt: '💕' },  // two hearts
      { hex: '1f49e', alt: '💞' },  // revolving hearts
      { hex: '1f497', alt: '💗' },  // growing heart
      { hex: '1fae6', alt: '🫦' },  // biting lip
      { hex: '1f609', alt: '😉' },  // winking face
      { hex: '1f979', alt: '🥹' },  // holding back tears
      { hex: '1f48c', alt: '💌' },  // love letter
      { hex: '1f60f', alt: '😏' },  // smirk
      { hex: '1f61c', alt: '😜' },  // winking tongue
      { hex: '1fae0', alt: '🫠' },  // melting face
      { hex: '1f493', alt: '💓' },  // beating heart
      { hex: '1f4ab', alt: '💫' },  // dizzy
      { hex: '2728',  alt: '✨' },  // sparkles
      { hex: '1f97a', alt: '🥺' },  // pleading face
      { hex: '1f633', alt: '😳' },  // flushed face
      { hex: '1f917', alt: '🤗' },  // hugging face
      { hex: '1f60a', alt: '😊' },  // smiling face with smiling eyes
      { hex: '1f643', alt: '🙃' },  // upside-down face
      { hex: '1f607', alt: '😇' },  // smiling face with halo
    ],
    spicy: [
      { hex: '1f525', alt: '🔥' },  // fire
      { hex: '1f975', alt: '🥵' },  // hot face
      { hex: '1f608', alt: '😈' },  // smiling imp
      { hex: '1f60b', alt: '😋' },  // yum
      { hex: '1f924', alt: '🤤' },  // drooling face
      { hex: '1fae6', alt: '🫦' },  // biting lip
      { hex: '1f4a5', alt: '💥' },  // collision
      { hex: '1f386', alt: '🎆' },  // fireworks
      { hex: '1f624', alt: '😤' },  // face with steam
      { hex: '1f929', alt: '🤩' },  // star-struck
      { hex: '1f911', alt: '🤑' },  // money-mouth face
      { hex: '1f920', alt: '🤠' },  // cowboy hat face
      { hex: '1f921', alt: '🤡' },  // clown face
      { hex: '1f635', alt: '😵' },  // dizzy face
      { hex: '1f47f', alt: '👿' },  // angry face with horns
      { hex: '1f47d', alt: '👽' },  // alien (replaces goblin 1f47a — 404)
      { hex: '1f479', alt: '👹' },  // ogre
      { hex: '1f480', alt: '💀' },  // skull
      { hex: '1f922', alt: '🤢' },  // nauseated face
      { hex: '1f92e', alt: '🤮' },  // face vomiting
      { hex: '1f913', alt: '🤓' },  // nerd face
      { hex: '1f47b', alt: '👻' },  // ghost
      { hex: '1f916', alt: '🤖' },  // robot face
      { hex: '1f47e', alt: '👾' },  // alien monster
    ],
    fun: [
      { hex: '1f602', alt: '😂' },  // face with tears of joy
      { hex: '1f973', alt: '🥳' },  // partying face
      { hex: '1f389', alt: '🎉' },  // party popper
      { hex: '1f648', alt: '🙈' },  // see-no-evil
      { hex: '1f649', alt: '🙉' },  // hear-no-evil
      { hex: '1f64a', alt: '🙊' },  // speak-no-evil
      { hex: '1f974', alt: '🥴' },  // woozy face
      { hex: '1f62d', alt: '😭' },  // loudly crying
      { hex: '1f923', alt: '🤣' },  // rolling on floor laughing
      { hex: '1f92f', alt: '🤯' },  // exploding head
      { hex: '1fae3', alt: '🫣' },  // face with peeking eye
      { hex: '1f92d', alt: '🤭' },  // face with hand over mouth
      { hex: '1f631', alt: '😱' },  // face screaming
      { hex: '1fae1', alt: '🫡' },  // saluting face
      { hex: '1f978', alt: '🥸' },  // disguised face
      { hex: '1faf6', alt: '🫶' },  // heart hands
      { hex: '1f918', alt: '🤘' },  // sign of horns
      { hex: '1fac2', alt: '🫂' },  // people hugging
      { hex: '1f914', alt: '🤔' },  // thinking face
      { hex: '1fae4', alt: '🫤' },  // face with diagonal mouth
      { hex: '1fae8', alt: '🫨' },  // shaking face
      { hex: '1fae5', alt: '🫥' },  // dotted line face
      { hex: '1fae7', alt: '🫧' },  // bubbles
      { hex: '1fae2', alt: '🫢' },  // face with open eyes and hand over mouth
    ],
    reactions: [
      { hex: '1f976', alt: '🥶' },  // cold face
      { hex: '1f925', alt: '🤥' },  // lying face
      { hex: '1f92b', alt: '🤫' },  // shushing face
      { hex: '1f9d0', alt: '🧐' },  // face with monocle
      { hex: '1f644', alt: '🙄' },  // face with rolling eyes
      { hex: '1f621', alt: '😡' },  // pouting face
      { hex: '1f92c', alt: '🤬' },  // face with symbols over mouth
      { hex: '1f612', alt: '😒' },  // unamused face
      { hex: '1f614', alt: '😔' },  // pensive face
      { hex: '1f62c', alt: '😬' },  // grimacing face
      { hex: '1f634', alt: '😴' },  // sleeping face
      { hex: '1f62b', alt: '😫' },  // tired face
      { hex: '1f629', alt: '😩' },  // weary face
      { hex: '1f627', alt: '😧' },  // anguished face
      { hex: '1f628', alt: '😨' },  // fearful face
      { hex: '1f630', alt: '😰' },  // anxious face with sweat
      { hex: '1f626', alt: '😦' },  // frowning face with open mouth
      { hex: '1f625', alt: '😥' },  // sad but relieved face
      { hex: '1f622', alt: '😢' },  // crying face
      { hex: '1f61e', alt: '😞' },  // disappointed face
      { hex: '1f620', alt: '😠' },  // angry face
      { hex: '1f615', alt: '😕' },  // confused face
      { hex: '1f641', alt: '🙁' },  // slightly frowning face
      { hex: '1f610', alt: '😐' },  // neutral face
    ],
    hands: [
      { hex: '1faf6', alt: '🫶' },  // heart hands
      { hex: '1faf0', alt: '🫰' },  // hand with index finger and thumb crossed
      { hex: '1faf1', alt: '🫱' },  // rightwards hand
      { hex: '1faf2', alt: '🫲' },  // leftwards hand
      { hex: '1faf3', alt: '🫳' },  // palm down hand
      { hex: '1faf4', alt: '🫴' },  // palm up hand
      { hex: '1faf5', alt: '🫵' },  // index pointing at viewer
      { hex: '1faf7', alt: '🫷' },  // leftwards pushing hand
      { hex: '1faf8', alt: '🫸' },  // rightwards pushing hand
      { hex: '1f91d', alt: '🤝' },  // handshake
      { hex: '1f44f', alt: '👏' },  // clapping hands
      { hex: '1f44b', alt: '👋' },  // waving hand
      { hex: '1f91f', alt: '🤟' },  // love-you gesture
      { hex: '1f918', alt: '🤘' },  // sign of horns
      { hex: '1f919', alt: '🤙' },  // call me hand
      { hex: '1f44d', alt: '👍' },  // thumbs up
      { hex: '1f44e', alt: '👎' },  // thumbs down
      { hex: '1f91e', alt: '🤞' },  // crossed fingers
      { hex: '1fac2', alt: '🫂' },  // people hugging
      { hex: '1f64f', alt: '🙏' },  // folded hands
      { hex: '1f64c', alt: '🙌' },  // raising hands
      { hex: '1f932', alt: '🤲' },  // palms up together
      { hex: '1f4aa', alt: '💪' },  // flexed biceps (replaces man dancing 1f57a — 404)
      { hex: '1f483', alt: '💃' },  // woman dancing (confirmed 200)
    ],
  };

  function gifUrl(hex) {
    return `${CDN}/${hex}/512.gif`;
  }

  function pick(hex) {
    const url = gifUrl(hex);
    dispatch('pick', `[gif:${url}]`);
  }

  // Track broken images per-tab so we can hide failed GIFs cleanly
  let brokenHexes = new Set();

  function handleImgError(hex) {
    brokenHexes = new Set([...brokenHexes, hex]);
  }

  function reposition() {
    if (!wrapEl || !anchor) return;
    const ab = anchor.getBoundingClientRect();
    const pw = wrapEl.offsetWidth  || 300;
    const ph = wrapEl.offsetHeight || 340;
    const vw = window.innerWidth;

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
    <div class="sp-tabs" role="tablist" aria-label="Sticker categories">
      {#each TABS as tab}
        <button
          class="sp-tab"
          class:sp-tab--active={activeTab === tab.id}
          on:click={() => activeTab = tab.id}
          type="button"
          role="tab"
          aria-selected={activeTab === tab.id}
          aria-controls="sp-panel-{tab.id}"
        >{tab.label}</button>
      {/each}
    </div>
    <div
      class="sp-grid"
      role="tabpanel"
      id="sp-panel-{activeTab}"
      aria-label="{TABS.find(t => t.id === activeTab)?.label} stickers"
    >
      {#each STICKERS[activeTab].filter(s => !brokenHexes.has(s.hex)) as s (s.hex)}
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
            width="48"
            height="48"
            on:error={() => handleImgError(s.hex)}
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
    width: 300px;
    /* Teal-tinted dark surface — matches SecretChatPanel token system */
    background: #0a0a18;
    border: 1px solid rgba(20, 184, 166, 0.12);
    border-radius: var(--radius-xl, 20px);
    box-shadow:
      0 16px 48px rgba(0, 0, 0, 0.75),
      0 0 0 1px rgba(20, 184, 166, 0.06),
      inset 0 1px 0 rgba(255, 255, 255, 0.04);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    animation: sp-pop var(--duration-normal, 200ms) var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
    max-height: min(340px, 60dvh);
  }

  .sp-tabs {
    display: flex;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    flex-shrink: 0;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .sp-tabs::-webkit-scrollbar { display: none; }

  .sp-tab {
    flex-shrink: 0;
    padding: 0 var(--space-2-5, 10px);
    background: none;
    border: none;
    color: rgba(255, 255, 255, 0.32);
    font-size: var(--text-xs, 0.75rem);
    font-weight: 500;
    cursor: pointer;
    /* Use design system font — not system-ui */
    font-family: var(--font-sans, 'Nunito', sans-serif);
    transition: color var(--duration-fast, 100ms), background var(--duration-fast, 100ms);
    border-bottom: 2px solid transparent;
    white-space: nowrap;
    touch-action: manipulation;
    /* Minimum 44px touch target */
    min-height: 44px;
    display: flex;
    align-items: center;
  }
  .sp-tab:hover {
    color: rgba(255, 255, 255, 0.65);
    background: rgba(255, 255, 255, 0.04);
  }
  /* Teal active state — replaces purple #818cf8 */
  .sp-tab--active {
    color: var(--primary-500, #14b8a6);
    border-bottom-color: var(--primary-500, #14b8a6);
  }
  .sp-tab:focus-visible {
    outline: 2px solid var(--primary-500, #14b8a6);
    outline-offset: -2px;
  }

  .sp-grid {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-2, 8px);
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 2px;
    align-content: start;
  }
  .sp-grid::-webkit-scrollbar { width: 3px; }
  .sp-grid::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.07);
    border-radius: var(--radius-full, 9999px);
  }

  .sp-sticker-btn {
    width: 100%;
    aspect-ratio: 1;
    background: none;
    border: none;
    border-radius: var(--radius-sm2, 8px);
    cursor: pointer;
    padding: 3px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background var(--duration-fast, 100ms), transform var(--duration-fast, 100ms);
    touch-action: manipulation;
    /* Minimum 44px tap target via grid column sizing */
    min-width: 44px;
    min-height: 44px;
  }
  .sp-sticker-btn:hover {
    background: rgba(255, 255, 255, 0.07);
    transform: scale(1.12);
  }
  /* Teal active press — replaces purple rgba(129,140,248,0.18) */
  .sp-sticker-btn:active {
    background: rgba(20, 184, 166, 0.16);
    transform: scale(0.95);
  }
  .sp-sticker-btn:focus-visible {
    outline: 2px solid var(--primary-500, #14b8a6);
    outline-offset: 2px;
  }

  .sp-img {
    width: 40px;
    height: 40px;
    object-fit: contain;
    display: block;
    border-radius: var(--radius-xs, 4px);
  }

  @keyframes sp-pop {
    from { opacity: 0; transform: scale(0.88) translateY(8px); }
    to   { opacity: 1; transform: scale(1)    translateY(0);   }
  }

  @media (prefers-reduced-motion: reduce) {
    .sp-wrap { animation: none; }
    .sp-sticker-btn:hover  { transform: none; }
    .sp-sticker-btn:active { transform: none; }
  }
</style>
