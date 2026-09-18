<script>
  import { stopPropagation } from 'svelte/legacy';

  import { createEventDispatcher, onDestroy } from 'svelte';
  import { formatTimestamp } from '../../lib/tracking.js';
  import { haptics } from '../../lib/haptics.js';
  import UserAvatar from './UserAvatar.svelte';
  import UserBadges from './UserBadges.svelte';
  import UserSubline from './UserSubline.svelte';

  /**
   * @typedef {Object} Props
   * @property {any} user
   * @property {number} [index]
   * @property {boolean} [isAdmin]
   * @property {any} [deletingUser]
   * @property {boolean} [initialLoad] — true only during first list population;
   *   controls row entrance stagger. Later in-place updates never re-stagger.
   */

  /** @type {Props} */
  let {
    user,
    index = 0,
    isAdmin = false,
    deletingUser = null,
    initialLoad = false,
  } = $props();

  const dispatch = createEventDispatcher();

  // Single presence signal — drives the trailing dot and the hover/focus accent.
  // SOS > offline > online. Hearth ladder: live = sage, offline = quiet stone.
  let presence = $derived(
    user.sos?.active ? 'sos' : user.online === false ? 'gone' : 'recent'
  );
  let accentVar = $derived(
    presence === 'sos' ? 'var(--status-sos)'
      : presence === 'gone' ? 'var(--status-offline)'
      : 'var(--status-live)'
  );

  // Stagger index: cap at 5 so the 6th+ row shares the same delay (spec: cap at 6 items).
  let staggerI = $derived(Math.min(index, 5));

  // ── Swipe-right to locate on map ────────────────────────────────────────
  let swipeStartX = 0;
  let swipeStartY = 0;

  function onTouchStart(e) {
    swipeStartX = e.touches[0].clientX;
    swipeStartY = e.touches[0].clientY;
  }

  function onTouchEnd(e) {
    const dx = e.changedTouches[0].clientX - swipeStartX;
    const dy = Math.abs(e.changedTouches[0].clientY - swipeStartY);
    if (dx > 60 && dy < 30) {
      dispatch('locate', user.socketId);
    } else if (dx < -60 && dy < 30 && user.userId) {
      // Swipe-left — covert fast path to chat (no visual indicator by design)
      haptics.tap?.();
      dispatch('secretChat', { id: user.userId, name: user.displayName });
    }
  }

  // ── Long-press quick actions ─────────────────────────────────────────────
  let lpTimer = null;          // long-press timer
  let lpSuppressClick = false; // prevent click from firing after long-press
  let lpStartX = 0;
  let lpStartY = 0;

  function rowPD(e) {
    if (e.button != null && e.button !== 0) return; // ignore right-click
    lpStartX = e.clientX;
    lpStartY = e.clientY;
    lpTimer = setTimeout(() => {
      lpTimer = null;
      lpSuppressClick = true;
      haptics.confirm?.();
      dispatch('quickActions', user);
    }, 250);
  }

  function rowPM(e) {
    if (!lpTimer) return;
    if (Math.abs(e.clientX - lpStartX) > 10 || Math.abs(e.clientY - lpStartY) > 10) {
      clearTimeout(lpTimer);
      lpTimer = null;
    }
  }

  function rowPU() {
    if (lpTimer) { clearTimeout(lpTimer); lpTimer = null; }
  }

  // Cancel long-press timer on component destroy to prevent firing on unmounted component
  onDestroy(() => {
    if (lpTimer) { clearTimeout(lpTimer); lpTimer = null; }
  });

  function rowClick() {
    if (lpSuppressClick) { lpSuppressClick = false; return; }
    if (user.latitude == null || user.longitude == null) {
      dispatch('quickActions', user);
      return;
    }
    dispatch('locate', user.socketId);
  }

  function onKeydown(event) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      dispatch('locate', user.socketId);
    }
  }
</script>

<div
  class="user-item user-item-btn"
  class:user-depth-card={initialLoad}
  class:user-sos={user.sos?.active}
  class:user-offline={user.online === false}
  style="--stagger-i: {staggerI}; --row-accent: {accentVar};"
  role="button"
  tabindex="0"
  onclick={rowClick}
  onkeydown={onKeydown}
  ontouchstart={onTouchStart}
  ontouchend={onTouchEnd}
  onpointerdown={rowPD}
  onpointermove={rowPM}
  onpointerup={rowPU}
  onpointercancel={rowPU}
>
  <UserAvatar {user} />
  <div class="user-meta">
    <div class="user-name-row">
      <strong class="user-name">{user.displayName || 'User'}</strong>
      <span class="presence-dot presence-{presence}" aria-hidden="true"></span>
      <UserBadges {user} />
    </div>
    <!-- One quiet sentence: how they're doing · when we last heard -->
    <div class="user-sentence">
      <UserSubline {user} />
      {#if (user.formattedTime || user.lastUpdate) && user.online !== false}
        <span class="sentence-sep" aria-hidden="true">·</span>
        <span class="user-updated font-tabular">
          {user.formattedTime || formatTimestamp(user.lastUpdate)}
        </span>
      {/if}
    </div>
  </div>
  <div class="user-actions">
    {#if user.batteryPct != null}
      <span
        class="bat-chip font-tabular"
        class:bat-low={user.batteryPct <= 20}
        class:bat-ok={user.batteryPct > 20 && user.batteryPct <= 50}
        class:bat-good={user.batteryPct > 50}
        aria-label="Battery {user.batteryPct}%"
      >
        <!-- Battery icon -->
        <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="2" y="7" width="16" height="10" rx="2"/><path d="M22 11v2"/></svg>
        {user.batteryPct}%
      </span>
    {/if}
    {#if isAdmin}
      <button class="user-remove-btn" aria-label="Remove {user.displayName || 'user'}" onclick={stopPropagation(() => dispatch('delete', user))} disabled={deletingUser === user.socketId}>×</button>
    {/if}
    <!-- Right affordance — signals the row opens details / locates on map -->
    <span class="row-chevron" aria-hidden="true">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
    </span>
  </div>
</div>

<style>
  /* ── Continuous quiet paper row — whitespace + hairline, no card chrome ── */
  .user-item {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-4);
    min-height: 76px;
    /* Fill the VirtualList slot so the hairline sits on the slot boundary. */
    height: 100%;
    border-bottom: 1px solid var(--border-subtle);
  }

  /*
   * Entrance stagger — applied ONLY when initialLoad=true so VirtualList scroll
   * recycling never re-triggers the animation on rows that are already visible.
   * Uses stagger tokens: --stagger-base (0ms lead-in) + index * --stagger-step (60ms).
   * Cap at 5 (6th+ row shares the 5th delay) — more than 6 concurrent animations
   * degrade mid-range devices and add no glanceability gain.
   */
  .user-depth-card {
    animation: item-pop-in 300ms var(--ease-spring, cubic-bezier(0.34, 1.56, 0.64, 1)) both;
    animation-delay: calc(var(--stagger-base, 0ms) + var(--stagger-i, 0) * var(--stagger-step, 60ms));
  }

  .user-item-btn {
    width: 100%;
    background: none;
    border-top: none;
    border-left: none;
    border-right: none;
    cursor: pointer;
    text-align: left;
    color: inherit;
    font: inherit;
    border-radius: 0;
    transition:
      background var(--duration-fast) var(--ease-out),
      box-shadow var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
    position: relative;
  }

  .user-item-btn:hover {
    background: var(--surface-hover);
  }

  .user-item-btn:active {
    background: var(--surface-active, var(--surface-hover));
  }

  /* Presence-colored left accent on hover/focus (SOS keeps its permanent accent) */
  .user-item-btn:not(.user-sos):hover,
  .user-item-btn:not(.user-sos):focus-visible {
    box-shadow: inset 3px 0 0 var(--row-accent, var(--status-live));
  }

  /* Keyboard focus ring */
  .user-item-btn:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: -2px;
  }

  /*
   * SOS row — vermilion is reserved for crisis, and this IS one: a received
   * distress state. Flat warm tint + steady accent bar; no gradient noise.
   */
  .user-sos {
    background: color-mix(in oklch, var(--danger-500) 8%, transparent);
    box-shadow: inset 3px 0 0 var(--danger-500);
  }
  .user-sos:hover {
    background: color-mix(in oklch, var(--danger-500) 12%, transparent);
  }

  /* ── Meta ──────────────────────────────────────────────────────────────── */
  .user-meta {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-0-5);
  }

  .user-name-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
    overflow: hidden;
  }

  /* Trailing presence dot — one glanceable cue beside the name (Hearth ladder:
     sage = settled/live, ochre-free here; offline = quiet stone; SOS = vermilion). */
  .presence-dot {
    width: 6px;
    height: 6px;
    border-radius: var(--radius-full);
    flex-shrink: 0;
    background: var(--status-offline);
  }
  .presence-recent { background: var(--status-live); }
  .presence-gone   { background: var(--status-offline); }
  .presence-sos    { background: var(--status-sos); }

  .user-name {
    font-family: var(--font-sans);
    font-size: var(--text-base);
    font-weight: 600;
    letter-spacing: -0.01em;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--text-primary);
    line-height: var(--leading-tight);
  }

  /* Sentence line — subline + soft timestamp share one 16px reading line */
  .user-sentence {
    display: flex;
    align-items: baseline;
    min-width: 0;
    overflow: hidden;
    white-space: nowrap;
  }

  .sentence-sep {
    color: var(--text-tertiary);
    opacity: 0.7;
    margin: 0 var(--space-1);
    flex-shrink: 0;
  }

  /* Timestamp — soft, inside the sentence, never a separate metric line */
  .user-updated {
    font-size: var(--text-sm);
    color: var(--text-tertiary);
    letter-spacing: 0.01em;
    font-variant-numeric: tabular-nums;
    font-feature-settings: 'tnum' 1;
    flex-shrink: 0;
  }

  /* ── Actions column ─────────────────────────────────────────────────────── */
  .user-actions {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    flex-shrink: 0;
  }

  /* Chevron affordance — outline ink, warms to ember on intent */
  .row-chevron {
    color: var(--text-tertiary);
    display: flex;
    align-items: center;
    opacity: 0.6;
    transition: color var(--duration-fast) var(--ease-out),
                transform var(--duration-fast) var(--ease-out),
                opacity var(--duration-fast) var(--ease-out);
  }
  .user-item-btn:hover .row-chevron,
  .user-item-btn:focus-visible .row-chevron {
    color: var(--primary-700);
    opacity: 1;
    transform: translateX(2px);
  }

  /*
   * Battery chip — kept, but quiet: no battery shame. Low = ochre ("needs a
   * look"), never vermilion; healthy states stay near-silent.
   */
  .bat-chip {
    display: inline-flex;
    align-items: center;
    gap: var(--space-0-5);
    font-family: var(--font-sans);
    font-size: var(--text-2xs);
    font-weight: 600;
    padding: var(--space-0-5) var(--space-1-5);
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    color: var(--text-tertiary);
    letter-spacing: 0.01em;
    font-variant-numeric: tabular-nums;
    font-feature-settings: 'tnum' 1;
  }
  .bat-low {
    color: var(--warning-700);
    background: color-mix(in oklch, var(--warning-500) 14%, transparent);
  }
  .bat-ok {
    color: var(--text-secondary);
    background: var(--surface-inset);
  }
  .bat-good {
    color: var(--success-700);
    background: color-mix(in oklch, var(--success-500) 10%, transparent);
  }

  /* Admin remove — destructive but never vermilion (banned outside SOS).
     Quiet stone button that darkens on intent; 44px target. */
  .user-remove-btn {
    min-width: 44px;
    min-height: 44px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 1px solid var(--border-default);
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    color: var(--text-secondary);
    font-family: var(--font-sans);
    font-size: var(--text-lg);
    line-height: 1;
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out), color var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .user-remove-btn:hover {
    background: color-mix(in oklch, var(--warning-500) 14%, transparent);
    color: var(--warning-700);
  }
  .user-remove-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  @media (prefers-reduced-motion: reduce) {
    /* Stagger animation dies; rows appear instantly. */
    .user-depth-card {
      animation: none;
      animation-delay: 0ms;
    }
    .row-chevron { transition: none; }
  }
</style>
