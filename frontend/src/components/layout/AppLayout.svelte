<script>
  import { onMount, onDestroy } from 'svelte';
  import { debounce } from '../../lib/debounce.js';

  /**
   * @typedef {Object} Props
   * @property {boolean} [sidebarOpen]
   * @property {boolean} [rightPanelOpen]
   * @property {import('svelte').Snippet} [topBar]
   * @property {import('svelte').Snippet} [navbar]
   * @property {import('svelte').Snippet} [sidebar]
   * @property {import('svelte').Snippet} [map]
   * @property {import('svelte').Snippet} [banner]
   * @property {import('svelte').Snippet} [rightPanel]
   * @property {import('svelte').Snippet} [bottomSheet]
   * @property {import('svelte').Snippet} [bottomTabs]
   * @property {import('svelte').Snippet} [overlay]
   */

  /** @type {Props} */
  let {
    sidebarOpen = true,
    rightPanelOpen = false,
    topBar,
    navbar,
    sidebar,
    map,
    banner,
    rightPanel,
    bottomSheet,
    bottomTabs,
    overlay
  } = $props();

  let isMobile = $state(typeof window !== 'undefined' ? window.innerWidth < 768 : false);
  let isTablet = $state(typeof window !== 'undefined' ? (window.innerWidth >= 768 && window.innerWidth < 1024) : false);

  function checkBreakpoint() {
    const w = window.innerWidth;
    isMobile = w < 768;
    isTablet = w >= 768 && w < 1024;
  }

  const debouncedCheck = debounce(checkBreakpoint, 80);

  onMount(() => {
    checkBreakpoint();
    window.addEventListener('resize', debouncedCheck);
  });

  onDestroy(() => {
    if (typeof window !== 'undefined') window.removeEventListener('resize', debouncedCheck);
  });
</script>

<div
  class="app-layout"
  class:sidebar-open={sidebarOpen && !isMobile}
  class:sidebar-closed={!sidebarOpen || isMobile}
  class:right-open={rightPanelOpen && !isMobile}
  class:mobile={isMobile}
  class:tablet={isTablet}
>
  {#if isMobile}
    {@render topBar?.()}
  {/if}

  {#if !isMobile}
    <div class="layout-navbar">
      {@render navbar?.()}
    </div>
  {/if}

  <div class="layout-body">
    {#if !isMobile}
      <div class="layout-sidebar">
        {@render sidebar?.()}
      </div>
    {/if}

    <div class="layout-map" id="main-content">
      {@render map?.()}
      {@render banner?.()}
    </div>

    {#if rightPanelOpen && !isMobile}
      <div class="layout-right">
        {@render rightPanel?.()}
      </div>
    {/if}
  </div>

  {#if isMobile}
    {@render bottomSheet?.()}
    <div class="layout-tabs">
      {@render bottomTabs?.()}
    </div>
  {/if}

  {@render overlay?.()}
</div>

<style>
  .app-layout {
    height: 100vh;
    height: 100dvh;
    width: 100%;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
  }

  /*
   * Desktop: sidebar width scales with the viewport instead of the fixed
   * 400px global token. Overriding the custom property here (rather than
   * hardcoding a width in Sidebar.svelte) means every consumer of
   * var(--sidebar-width) — Sidebar, SosFloat, PulseButton, MainApp map
   * controls — stays perfectly aligned with the sidebar edge.
   */
  @media (min-width: 1024px) {
    .app-layout {
      --sidebar-width: clamp(300px, 23vw, 420px);
    }
  }

  .layout-navbar {
    flex-shrink: 0;
    z-index: var(--z-navbar);
  }

  .layout-body {
    flex: 1;
    min-height: 0;
    display: grid;
    /* minmax(0, 1fr) lets the map column absorb ALL remaining width without
       being pushed by intrinsic content size at any viewport */
    grid-template-columns: auto minmax(0, 1fr) auto;
    grid-template-rows: minmax(0, 1fr);
    overflow: hidden;
    position: relative;
  }

  .layout-sidebar {
    grid-column: 1;
    grid-row: 1;
    display: flex;
    min-height: 0;
    z-index: var(--z-panel);
    overflow: hidden;
  }

  .layout-map {
    grid-column: 2;
    grid-row: 1;
    position: relative;
    overflow: hidden;
    min-width: 0;
    min-height: 0;
  }

  .layout-right {
    grid-column: 3;
    grid-row: 1;
    z-index: var(--z-panel);
    overflow: hidden;
  }

  /* Mobile: single column, map fills everything */
  .app-layout.mobile .layout-body {
    grid-template-columns: 1fr;
  }

  .app-layout.mobile .layout-map {
    grid-column: 1;
    /* Base 108px accounts for topbar content height; safe-top adds notch/Dynamic Island offset */
    padding-top: calc(var(--safe-top, 0px) + 108px);
    /* Clearance so the floating tab pill + safe area never obscure map content
       (single source token — also consumed by BottomSheet's .sheet-body) */
    padding-bottom: var(--tab-bar-clearance);
    transition: padding-bottom 300ms ease;
  }

  /* Floating tab-bar slot — overlays the full-bleed map; the pill inside
     re-enables pointer events, the gutter around it stays tappable map. */
  .layout-tabs {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    display: flex;
    justify-content: center;
    padding-bottom: calc(var(--safe-bottom, 0px) + var(--space-2));
    pointer-events: none;
    z-index: var(--z-navbar);
  }
</style>
