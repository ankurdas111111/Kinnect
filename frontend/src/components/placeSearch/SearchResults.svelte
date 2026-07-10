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
  .sr-list {
    position: absolute; top: calc(100% + 4px); left: 0; right: 0;
    background: rgba(8,12,24,0.96);
    backdrop-filter: blur(24px); -webkit-backdrop-filter: blur(24px);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 14px; padding: 4px;
    list-style: none; margin: 0;
    max-height: 300px; overflow-y: auto;
    box-shadow: 0 12px 40px rgba(0,0,0,0.5);
    scrollbar-width: thin; scrollbar-color: rgba(255,255,255,0.06) transparent;
    z-index: 30;
  }
  .sr-section {
    padding: 6px 12px 2px;
    font-size: 9px; font-weight: 700; letter-spacing: 0.08em;
    color: rgba(255,255,255,0.20); text-transform: uppercase;
  }
  .sr-row {
    display: flex; align-items: center; gap: 10px;
    padding: 9px 12px; min-height: 44px; width: 100%;
    border-radius: 10px; border: none; background: transparent;
    text-align: left; cursor: pointer;
    transition: background 0.1s;
    touch-action: manipulation; -webkit-tap-highlight-color: transparent;
  }
  .sr-row:hover, .sr-hl { background: rgba(99,102,241,0.10); }
  .sr-row:active { background: rgba(99,102,241,0.18); }
  .sr-icon {
    color: rgba(99,102,241,0.55); flex-shrink: 0;
    display: flex; align-items: center; justify-content: center;
    width: 20px; height: 20px;
  }
  .sr-icon--emoji { font-size: 15px; line-height: 1; }
  .sr-text { display: flex; flex-direction: column; gap: 1px; min-width: 0; flex: 1; }
  .sr-name { font-size: 13px; font-weight: 600; color: rgba(255,255,255,0.88); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .sr-sub { font-size: 11px; color: rgba(255,255,255,0.30); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
