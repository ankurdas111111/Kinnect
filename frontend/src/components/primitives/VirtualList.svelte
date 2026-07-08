<script>
  import { onMount, } from 'svelte';

  /**
   * @typedef {Object} Props
   * @property {any} [items]
   * @property {number} [itemHeight] - estimated px per item
   * @property {number} [overscan] - extra items above/below viewport
   * @property {import('svelte').Snippet<[any]>} [children]
   */

  /** @type {Props} */
  let {
    items = [],
    itemHeight = 72,
    overscan = 3,
    children
  } = $props();

  let containerEl = $state();
  let scrollTop = $state(0);
  let viewportHeight = $state(0);
  let rafId = null;

  let totalHeight = $derived(items.length * itemHeight);

  let startIndex = $derived(Math.max(0, Math.floor(scrollTop / itemHeight) - overscan));
  let endIndex = $derived(Math.min(
    items.length,
    Math.ceil((scrollTop + viewportHeight) / itemHeight) + overscan
  ));

  let visibleItems = $derived(items.slice(startIndex, endIndex).map((item, i) => ({
    item,
    index: startIndex + i,
    top: (startIndex + i) * itemHeight,
  })));

  function onScroll(e) {
    if (rafId) return;
    rafId = requestAnimationFrame(() => {
      scrollTop = e.target.scrollTop;
      rafId = null;
    });
  }

  onMount(() => {
    if (containerEl) {
      viewportHeight = containerEl.clientHeight;
      const ro = new ResizeObserver(entries => {
        viewportHeight = entries[0].contentRect.height;
      });
      ro.observe(containerEl);
      return () => ro.disconnect();
    }
  });
</script>

<div
  class="virtual-list"
  bind:this={containerEl}
  onscroll={onScroll}
>
  <div class="virtual-spacer" style="height:{totalHeight}px; position:relative;">
    {#each visibleItems as { item, index, top } (item.socketId ?? item.id ?? index)}
      <div
        class="virtual-item"
        style="position:absolute;top:{top}px;left:0;right:0;height:{itemHeight}px;"
      >
        {@render children?.({ item, index, })}
      </div>
    {/each}
  </div>
</div>

<style>
  .virtual-list {
    overflow-y: auto;
    overflow-x: hidden;
    -webkit-overflow-scrolling: touch;
    height: 100%;
    min-height: 0;
    flex: 1;
  }
  .virtual-spacer {
    position: relative;
    /* overflow:hidden removed — was clipping absolutely positioned children
       (presence rings, badges, SOS indicators) that extend outside row bounds */
    overflow: visible;
  }
  .virtual-item {
    /* overflow:hidden removed — was clipping presence rings, badges, and other
       absolutely positioned children that extend outside the fixed-height row.
       Clip is still effective via the virtual-list container overflow:hidden. */
    overflow: visible;
  }
</style>
