<script>
  /**
   * PlacesListSection — saved places list, zone story, and add-place form.
   * Receives places and callbacks from SavedPlacesPanel (the orchestrator).
   *
   * Props:
   *   places        — Place[]  — array of saved places
   *   iconOptions   — {value, label}[]
   *   iconEmoji     — Record<string, string>  — emoji lookup from ICON_MAP
   *   onAdd         — fn(name, radius, icon) → Promise<boolean>
   *   onRemove      — fn(placeId, placeName)
   *   onViewStory   — fn(placeId)
   *   storyPlaceId  — string | null
   *   storyLoading  — boolean
   *   storyVisits   — StoryVisit[]
   */
  import SectionHeader from './primitives/SectionHeader.svelte';
  import { ICON_MAP } from '../lib/alertConfig.js';

  /** @type {{ places?: any[], iconOptions?: any[], iconEmoji?: Record<string, string>, onAdd: Function, onRemove: Function, onViewStory: Function, storyPlaceId?: string|null, storyLoading?: boolean, storyVisits?: any[] }} */
  let {
    places = [],
    iconOptions = [],
    iconEmoji = {},
    onAdd,
    onRemove,
    onViewStory,
    storyPlaceId = null,
    storyLoading = false,
    storyVisits = []
  } = $props();

  let newPlaceName = $state('');
  let newPlaceRadius = $state(100);
  let newPlaceIcon = $state('pin');
  let showAddPlace = $state(false);

  function formatDuration(seconds) {
    if (!seconds) return '';
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    if (h > 0) return `${h}h ${m}m`;
    return `${m}m`;
  }

  function formatTime(iso) {
    if (!iso) return '';
    return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function formatDate(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    const today = new Date();
    if (d.toDateString() === today.toDateString()) return 'Today';
    const yesterday = new Date(today);
    yesterday.setDate(today.getDate() - 1);
    if (d.toDateString() === yesterday.toDateString()) return 'Yesterday';
    return d.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' });
  }

  /** aria-label for an emoji icon. Falls back to ICON_MAP label, then the emoji itself. */
  function iconLabel(iconKey) {
    return ICON_MAP[iconKey]?.label ?? iconEmoji[iconKey] ?? 'Place';
  }

  async function handleAdd() {
    if (!newPlaceName.trim()) return;
    const ok = await onAdd(newPlaceName.trim(), newPlaceRadius, newPlaceIcon);
    if (ok) {
      newPlaceName = '';
      newPlaceRadius = 100;
      newPlaceIcon = 'pin';
      showAddPlace = false;
    }
  }
</script>

<!-- ── Section header uses the shared SectionHeader primitive ── -->
<div class="section-wrap">
  <SectionHeader
    title="Saved Places"
    subtitle="Save home, work, school. Get notified when family arrives or leaves."
    level={3}
  >
    {#snippet action()}
      {#if places.length > 0}
        <span class="section-badge" aria-label="{places.length} saved places">{places.length}</span>
      {/if}
    {/snippet}
  </SectionHeader>
</div>

<div class="section-content">
  {#if places.length === 0 && !showAddPlace}
    <!-- Empty state: on-product CTA, brand rule compliance -->
    <div class="empty-state">
      <div class="empty-icon" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
      </div>
      <p class="empty-title">No saved places yet</p>
      <p class="empty-sub">Save Home to get arrival alerts and keep your family safe.</p>
      <button class="btn btn-primary btn-sm" onclick={() => showAddPlace = true}>
        Add Home or Work
      </button>
    </div>
  {:else}
    {#each places as place, i (place.id)}
      <!-- Place card: @starting-style entrance, stagger capped at 5 -->
      <div
        class="list-item"
        style="--_i: {Math.min(i, 4)}"
      >
        <!-- Emoji icon with accessible label + text fallback -->
        <span
          class="item-icon"
          role="img"
          aria-label={iconLabel(place.icon)}
          title={iconLabel(place.icon)}
        >
          <span aria-hidden="true">{iconEmoji[place.icon] ?? '📍'}</span>
          <!-- Screen-reader text fallback, hidden visually -->
          <span class="sr-only">{iconLabel(place.icon)}</span>
        </span>
        <div class="item-info">
          <span class="item-name">{place.name}</span>
          <span class="item-detail">{place.radiusM}m radius</span>
        </div>
        <button
          class="icon-action"
          onclick={() => onViewStory(place.id)}
          aria-label="View visit history for {place.name}"
          aria-expanded={storyPlaceId === place.id}
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
        </button>
        <button
          class="icon-action icon-action--danger"
          onclick={() => onRemove(place.id, place.name)}
          aria-label="Remove {place.name}"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>
        </button>
      </div>

      {#if storyPlaceId === place.id}
        <div class="zone-story" role="region" aria-label="Visit history for {place.name}">
          {#if storyLoading}
            <div class="story-skeleton" role="status" aria-label="Loading visit history" aria-busy="true">
              <div class="skel-row skel-wide"></div>
              <div class="skel-row skel-mid"></div>
            </div>
          {:else if storyVisits.length === 0}
            <p class="story-empty">No visits in the last 7 days.</p>
          {:else}
            {#each storyVisits as v}
              <div class="story-row">
                <div class="story-avatar" aria-hidden="true">{v.displayName?.charAt(0) ?? '?'}</div>
                <div class="story-info">
                  <span class="story-name">{v.displayName}</span>
                  <span class="story-time">
                    {formatDate(v.arrivedAt)} · {formatTime(v.arrivedAt)}
                    {#if v.departedAt} – {formatTime(v.departedAt)}{/if}
                    {#if v.durationSeconds} · {formatDuration(v.durationSeconds)}{/if}
                  </span>
                </div>
                {#if !v.departedAt}
                  <span class="story-badge-here" aria-label="Currently here">Here now</span>
                {/if}
              </div>
            {/each}
          {/if}
        </div>
      {/if}
    {/each}
  {/if}

  {#if showAddPlace}
    <div class="add-form" role="form" aria-label="Add a saved place">
      <input
        type="text"
        bind:value={newPlaceName}
        class="field-input field-full"
        placeholder="Name this place (Home, Work, School…)"
        maxlength="100"
        autocomplete="off"
        aria-label="Place name"
        style="font-size: 16px;"
      />
      <div class="form-row">
        <label class="sr-only" for="place-icon">Icon</label>
        <select id="place-icon" bind:value={newPlaceIcon} class="field-input field-sm" aria-label="Place icon">
          {#each iconOptions as opt}
            <option value={opt.value}>{iconEmoji[opt.value]} {opt.label}</option>
          {/each}
        </select>
        <label class="field-label-inline">
          <span>Radius</span>
          <input
            type="number"
            bind:value={newPlaceRadius}
            class="field-input field-num"
            min="50"
            max="5000"
            step="50"
            aria-label="Geofence radius in meters (50–5000)"
            style="font-size: 16px;"
          />
          <span class="field-unit" aria-hidden="true">m</span>
          <span class="sr-only">50 to 5000 meters</span>
        </label>
      </div>
      <div class="form-actions">
        <button
          class="btn btn-primary btn-sm"
          onclick={handleAdd}
          disabled={!newPlaceName.trim()}
        >
          Save This Place
        </button>
        <button
          class="btn btn-secondary btn-sm"
          onclick={() => { showAddPlace = false; newPlaceName = ''; }}
        >
          Cancel
        </button>
      </div>
    </div>
  {:else if places.length > 0}
    <button class="btn btn-secondary btn-sm add-btn" onclick={() => showAddPlace = true}>
      + Add a Place
    </button>
  {/if}
</div>

<style>
  /* ── Section wrap — hosts the SectionHeader primitive ───────────────────── */
  .section-wrap {
    padding: var(--space-4) var(--space-4) var(--space-1);
  }

  .section-badge {
    font-family: var(--font-display);
    font-size: var(--text-2xs);
    font-weight: 700;
    color: var(--text-tertiary);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-full);
    padding: 1px var(--space-2);
    min-width: 20px;
    text-align: center;
  }

  .section-content { padding: 0 var(--space-4) var(--space-3); }

  /* ── Place list items — @starting-style entrance ────────────────────────── */
  .list-item {
    display: flex;
    align-items: center;
    gap: var(--space-2-5);
    padding: var(--space-2) 0;
    border-bottom: 1px solid var(--border-subtle);
    /* Stagger: capped at 5 via --_i (set inline) */
    animation: place-enter var(--duration-normal, 200ms) var(--ease-out, ease-out)
               calc(var(--stagger-base, 50ms) + var(--_i, 0) * var(--stagger-step, 40ms))
               both;
  }
  .list-item:last-of-type { border-bottom: none; }

  @keyframes place-enter {
    from {
      opacity: 0;
      transform: translateY(8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .list-item {
      animation: none;
    }
  }

  /* ── Emoji icon — role=img with text fallback ───────────────────────────── */
  .item-icon {
    font-size: var(--text-lg);
    flex-shrink: 0;
    width: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    /* Don't leak color onto the emoji — presentation only */
  }

  .item-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .item-name {
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .item-detail {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }

  .icon-action {
    width: 44px;
    height: 44px;
    border-radius: var(--radius-sm2);
    border: none;
    background: transparent;
    color: var(--text-tertiary);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: background var(--duration-fast) var(--ease-out),
                color var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }

  .icon-action:hover { background: var(--surface-hover); color: var(--text-primary); }
  .icon-action:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  /* Danger hover: use CSS custom-property tint (no raw rgba) */
  .icon-action--danger:hover {
    background: color-mix(in srgb, var(--danger-500) 10%, transparent);
    color: var(--danger-500);
  }

  /* ── Empty state ─────────────────────────────────────────────────────────── */
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-6) var(--space-4);
    text-align: center;
    background: var(--surface-inset);
    border: 1px dashed var(--border-subtle);
    border-radius: var(--radius-lg);
    margin-bottom: var(--space-2);
  }

  .empty-icon {
    width: 48px;
    height: 48px;
    border-radius: var(--radius-full);
    /* Tint: primary-500-12 pattern — no raw rgba */
    background: color-mix(in srgb, var(--primary-500) 12%, transparent);
    border: 1px solid color-mix(in srgb, var(--primary-500) 18%, transparent);
    color: var(--primary-500);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .empty-title {
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--text-primary);
    margin: 0;
  }

  .empty-sub {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    line-height: var(--leading-normal);
    margin: 0;
    max-width: 220px;
  }

  /* ── Zone story ──────────────────────────────────────────────────────────── */
  .zone-story {
    margin: 0 0 var(--space-2);
    padding: var(--space-2) var(--space-3);
    background: var(--surface-inset);
    border-left: 3px solid var(--primary-500);
    border-radius: 0 var(--radius-md) var(--radius-md) 0;
  }

  .story-skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-1) 0;
  }

  .skel-row {
    height: var(--space-3);
    border-radius: var(--radius-sm);
    background: var(--skeleton-base, color-mix(in srgb, var(--text-primary) 5%, transparent));
    animation: skel-pulse 1.6s ease-in-out infinite;
  }
  .skel-wide { width: 100%; }
  .skel-mid  { width: 65%; }

  @keyframes skel-pulse {
    0%, 100% { opacity: 0.5; }
    50%       { opacity: 1; }
  }

  @media (prefers-reduced-motion: reduce) {
    .skel-row { animation: none; }
  }

  .story-empty {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    margin: 0;
    padding: var(--space-1) 0;
  }

  .story-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-1-5) 0;
    border-bottom: 1px solid var(--border-subtle);
  }
  .story-row:last-of-type { border-bottom: none; }

  .story-avatar {
    width: 24px;
    height: 24px;
    border-radius: var(--radius-full);
    background: var(--primary-500);
    color: var(--text-inverse);
    font-size: var(--text-2xs);
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    text-transform: uppercase;
  }

  .story-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .story-name {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-primary);
  }

  .story-time {
    font-size: var(--text-2xs);
    color: var(--text-tertiary);
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
  }

  .story-badge-here {
    font-size: var(--text-2xs);
    font-weight: 600;
    color: var(--success-600);
    background: color-mix(in srgb, var(--success-500) 12%, transparent);
    border: 1px solid color-mix(in srgb, var(--success-500) 24%, transparent);
    padding: 2px var(--space-1-5);
    border-radius: var(--radius-full);
    flex-shrink: 0;
    white-space: nowrap;
  }

  /* ── Add-place form ──────────────────────────────────────────────────────── */
  .add-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    margin-top: var(--space-2);
    padding: var(--space-3);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
  }

  .form-row {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    flex-wrap: wrap;
  }

  .field-input {
    padding: var(--space-2) var(--space-2-5);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    background: var(--surface-3);
    color: var(--text-primary);
    font-family: var(--font-sans);
    transition: border-color var(--duration-fast) var(--ease-out),
                box-shadow var(--duration-fast) var(--ease-out);
  }

  .field-input::placeholder { color: var(--text-tertiary); }
  .field-input option { background: var(--surface-2); color: var(--text-primary); }
  .field-input:focus {
    outline: none;
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--primary-500) 18%, transparent);
  }

  .field-full { width: 100%; box-sizing: border-box; }
  .field-sm   { flex: 1; min-width: 80px; }
  .field-num  { width: 64px; flex: none; }

  .field-label-inline {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    font-size: var(--text-xs);
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .field-unit {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
  }

  .form-actions {
    display: flex;
    gap: var(--space-2);
    flex-wrap: wrap;
  }

  .add-btn {
    align-self: flex-start;
    margin-top: var(--space-2);
  }

  /* ── Utilities ───────────────────────────────────────────────────────────── */
  .sr-only {
    position: absolute;
    width: 1px; height: 1px;
    padding: 0; margin: -1px;
    overflow: hidden;
    clip: rect(0,0,0,0);
    white-space: nowrap;
    border: 0;
  }
</style>
