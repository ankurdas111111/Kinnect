<script>
  /**
   * SearchResults — blended dropdown with Saved / Recent / Places sections.
   * Extracted from PlaceSearch.svelte to keep that file under ~550 lines.
   */
  import { getPinIcon } from '../../lib/pinIcons.js';

  /**
   * @typedef {{ _source: 'saved', name: string, sub: string, lat: number, lng: number, icon: string }} SavedItem
   * @typedef {{ _source: 'recent', name: string, sub: string, lat: number, lng: number, mode: string, ts: number }} RecentItem
   * @typedef {{ _source: 'place', name: string, sub: string, lat: number, lng: number, type: string }} PlaceItem
   */

  /** @type {{ savedMatches?: SavedItem[], recentMatches?: RecentItem[], placeResults?: PlaceItem[], highlightIdx?: number, onselect?: (item: any) => void }} */
  let {
    savedMatches = [],
    recentMatches = [],
    placeResults = [],
    highlightIdx = -1,
    onselect = () => {},
  } = $props();

  function relTime(ts) {
    const ago = Date.now() - ts;
    if (ago < 60000) return 'just now';
    if (ago < 3600000) return `${Math.round(ago / 60000)}m ago`;
    if (ago < 86400000) return `${Math.round(ago / 3600000)}h ago`;
    return `${Math.round(ago / 86400000)}d ago`;
  }
</script>

<ul class="sr-list" role="listbox" aria-label="Search suggestions">

  {#if savedMatches.length}
    <li class="sr-section" role="presentation">SAVED</li>
    {#each savedMatches as item, i}
      {@const hl = highlightIdx === i}
      <li role="option" aria-selected={hl}>
        <button type="button" class="sr-row" class:sr-hl={hl} onclick={() => onselect(item)}>
          <span class="sr-icon sr-icon--emoji" aria-hidden="true">{getPinIcon(item.icon).emoji}</span>
          <span class="sr-text">
            <span class="sr-name">{item.name}</span>
            {#if item.sub}<span class="sr-sub">{item.sub}</span>{/if}
          </span>
        </button>
      </li>
    {/each}
  {/if}

  {#if recentMatches.length}
    <li class="sr-section" role="presentation">RECENT</li>
    {#each recentMatches as item, i}
      {@const hl = highlightIdx === savedMatches.length + i}
      <li role="option" aria-selected={hl}>
        <button type="button" class="sr-row" class:sr-hl={hl} onclick={() => onselect(item)}>
          <span class="sr-icon" aria-hidden="true">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
            </svg>
          </span>
          <span class="sr-text">
            <span class="sr-name">{item.name}</span>
            <span class="sr-sub">{relTime(item.ts)}{item.sub ? ' · ' + item.sub : ''}</span>
          </span>
        </button>
      </li>
    {/each}
  {/if}

  {#if placeResults.length}
    {#if savedMatches.length || recentMatches.length}
      <li class="sr-section" role="presentation">PLACES</li>
    {/if}
    {#each placeResults as item, i}
      {@const hl = highlightIdx === savedMatches.length + recentMatches.length + i}
      <li role="option" aria-selected={hl}>
        <button type="button" class="sr-row" class:sr-hl={hl} onclick={() => onselect(item)}>
          <span class="sr-icon" aria-hidden="true">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
              <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0118 0z"/>
              <circle cx="12" cy="10" r="3"/>
            </svg>
          </span>
          <span class="sr-text">
            <span class="sr-name">{item.name}</span>
            {#if item.sub || item.distanceM}
              <span class="sr-sub">{[
                item.distanceM ? (item.distanceM < 1000 ? Math.round(item.distanceM) + ' m' : (item.distanceM / 1000).toFixed(1) + ' km') : '',
                item.sub || ''
              ].filter(Boolean).join(' · ')}</span>
            {/if}
          </span>
        </button>
      </li>
    {/each}
  {/if}

</ul>

<style>
  /* Warm paper dropdown — quiet rows, ember highlight for the active one. */
  .sr-list {
    position: absolute; top: calc(100% + 4px); left: 0; right: 0;
    background: var(--surface-1);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-button, 14px); padding: 4px;
    list-style: none; margin: 0;
    max-height: 300px; overflow-y: auto;
    box-shadow: var(--shadow-lg);
    scrollbar-width: thin; scrollbar-color: var(--border-default) transparent;
    z-index: 30;
  }
  .sr-section {
    padding: 6px 12px 2px;
    font-size: 11px; font-weight: 600; letter-spacing: 0.06em;
    color: var(--text-tertiary); text-transform: uppercase;
  }
  .sr-row {
    display: flex; align-items: center; gap: 10px;
    padding: 9px 12px; min-height: 44px; width: 100%;
    border-radius: var(--radius-input); border: none; background: transparent;
    text-align: left; cursor: pointer;
    transition: background 100ms var(--ease-out, ease-out);
    touch-action: manipulation; -webkit-tap-highlight-color: transparent;
  }
  .sr-row:hover, .sr-hl { background: var(--primary-500-12, color-mix(in oklch, var(--primary-500) 10%, transparent)); }
  .sr-row:active { background: color-mix(in oklch, var(--primary-500) 18%, transparent); }
  .sr-row:focus-visible { outline: 2px solid var(--primary-400); outline-offset: -2px; }
  .sr-icon {
    color: var(--text-tertiary); flex-shrink: 0;
    display: flex; align-items: center; justify-content: center;
    width: 20px; height: 20px;
  }
  .sr-row:hover .sr-icon, .sr-hl .sr-icon { color: var(--primary-700); }
  .sr-icon--emoji { font-size: 15px; line-height: 1; }
  .sr-text { display: flex; flex-direction: column; gap: 1px; min-width: 0; flex: 1; }
  .sr-name { font-size: var(--text-base); font-weight: 500; color: var(--text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .sr-sub { font-size: var(--text-xs); color: var(--text-tertiary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  @media (prefers-reduced-motion: reduce) {
    .sr-row { transition: none; }
  }
</style>
