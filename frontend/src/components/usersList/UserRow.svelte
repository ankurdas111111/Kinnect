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
   */

  /** @type {Props} */
  let {
    user,
    index = 0,
    isAdmin = false,
    deletingUser = null
  } = $props();

  const dispatch = createEventDispatcher();

  // Single presence signal — drives the leading dot and the hover/focus accent.
  // SOS > offline > online. (Kept binary here; finer "away" tiers live in the subline.)
  let presence = $derived(
    user.sos?.active ? 'sos' : user.online === false ? 'gone' : 'recent'
  );
  let accentVar = $derived(
    presence === 'sos' ? 'var(--presence-sos)'
      : presence === 'gone' ? 'var(--presence-gone)'
      : 'var(--presence-recent)'
  );

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
      // No location — open action sheet for available options (chat accessible via avatar tap)
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
  class="user-item user-item-btn user-depth-card"
  style="--stagger-i: {Math.min(index, 8)}; --row-accent: {accentVar};"
  class:user-sos={user.sos?.active}
  class:user-offline={user.online === false}
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
      <span class="presence-dot presence-{presence}" aria-hidden="true"></span>
      <strong class="user-name">{user.displayName || 'User'}</strong>
      <UserBadges {user} />
    </div>
    <UserSubline {user} />
    {#if (user.formattedTime || user.lastUpdate) && user.online !== false}
      <div class="user-updated font-tabular">
        {user.formattedTime || formatTimestamp(user.lastUpdate)}
      </div>
    {/if}
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
      <button class="btn btn-danger btn-sm" onclick={stopPropagation(() => dispatch('delete', user))} disabled={deletingUser === user.socketId}>×</button>
    {/if}
    <!-- Right affordance — signals the row opens details / locates on map -->
    <span class="row-chevron" aria-hidden="true">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
    </span>
  </div>
</div>

<style>
  /* ── User item — no borders, use spacing + hover bg ───────────────────── */
  .user-item {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2-5) var(--space-4);
    min-height: 76px;
  }

  /* 3D stagger entrance — index-driven delay via CSS custom property */
  .user-depth-card {
    animation: item-pop-in 300ms cubic-bezier(0.34, 1.56, 0.64, 1) both;
    animation-delay: calc(var(--stagger-i, 0) * 35ms);
  }

  .user-item-btn {
    width: 100%;
    background: none;
    border: none;
    border-bottom: none;
    cursor: pointer;
    text-align: left;
    color: inherit;
    font: inherit;
    border-radius: 0;
    transition:
      background var(--duration-fast) var(--ease-out),
      box-shadow var(--duration-fast) var(--ease-out),
      transform var(--duration-fast) var(--ease-spring);
    -webkit-tap-highlight-color: transparent;
    position: relative;
  }

  .user-item-btn:hover {
    background: var(--surface-hover);
  }

  .user-item-btn:active {
    background: var(--surface-active);
    transform: scale(0.97) translateZ(-2px);
    transition-duration: 60ms;
  }

  /* Presence-colored left accent on hover/focus (SOS keeps its permanent accent) */
  .user-item-btn:not(.user-sos):hover,
  .user-item-btn:not(.user-sos):focus-visible {
    box-shadow: inset 3px 0 0 var(--row-accent, var(--presence-recent));
  }

  /* Keyboard focus ring — design rules requires a visible focus affordance */
  .user-item-btn:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: -2px;
  }

  /* Avatar lives in the UserAvatar child — hover lift still driven by the row */
  .user-item-btn:hover :global(.user-avatar) {
    transform: scale(1.08) translateZ(4px);
    filter: brightness(1.12);
  }

  /* SOS — urgent red left accent + gradient sweep for readability */
  .user-sos {
    background:
      linear-gradient(90deg, rgba(239, 68, 68, 0.14) 0%, rgba(239, 68, 68, 0.06) 40%, transparent 80%),
      rgba(239, 68, 68, 0.10);
    box-shadow: inset 3px 0 0 var(--danger-500);
  }
  .user-sos:hover {
    background:
      linear-gradient(90deg, rgba(239, 68, 68, 0.20) 0%, rgba(239, 68, 68, 0.10) 40%, transparent 80%),
      rgba(239, 68, 68, 0.12);
  }

  /* ── Meta ──────────────────────────────────────────────────────────────── */
  .user-meta {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .user-name-row {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    min-width: 0;
    overflow: hidden;
  }

  /* Leading presence dot — one glanceable status cue on the primary line */
  .presence-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
    background: var(--presence-gone);
  }
  .presence-recent { background: var(--presence-recent); }
  .presence-now    { background: var(--presence-now); }
  .presence-away   { background: var(--presence-away); }
  .presence-gone   { background: var(--presence-gone); }
  .presence-sos    { background: var(--presence-sos); }

  .user-name {
    font-family: var(--font-display);
    font-size: var(--text-base);    /* 16px — legible primary label */
    font-weight: 600;
    letter-spacing: -0.01em;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--text-primary);
    line-height: 1.25;
  }

  /* Timestamp — tertiary, compact */
  .user-updated {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    letter-spacing: 0.01em;
    /* TECHNIQUE 8: tabular-nums for timestamps */
    font-variant-numeric: tabular-nums;
    font-feature-settings: 'tnum' 1;
  }

  /* ── Actions column ─────────────────────────────────────────────────────── */
  .user-actions {
    display: flex;
    align-items: center;
    gap: var(--space-1-5);
    flex-shrink: 0;
  }

  /* Chevron affordance — subtle, nudges right on hover */
  .row-chevron {
    color: var(--text-quaternary, var(--text-tertiary));
    display: flex;
    align-items: center;
    opacity: 0.5;
    transition: color var(--duration-fast) var(--ease-out),
                transform var(--duration-fast) var(--ease-spring),
                opacity var(--duration-fast) var(--ease-out);
  }
  .user-item-btn:hover .row-chevron,
  .user-item-btn:focus-visible .row-chevron {
    color: var(--primary-400);
    opacity: 1;
    transform: translateX(2px);
  }

  /* Battery chip — icon + percentage */
  .bat-chip {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-family: var(--font-display);
    font-size: 0.6875rem; /* 11px */
    font-weight: 700;
    padding: 3px 6px;
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    color: var(--text-tertiary);
    letter-spacing: 0.01em;
    /* TECHNIQUE 8: tabular-nums for battery % — never shifts layout */
    font-variant-numeric: tabular-nums;
    font-feature-settings: 'tnum' 1;
  }
  .bat-low  {
    color: var(--danger-400);
    background: rgba(239, 68, 68, 0.12);
    box-shadow: 0 0 0 1px rgba(239, 68, 68, 0.20);
  }
  .bat-ok   { color: var(--warning-500); background: rgba(245, 158, 11, 0.12); }
  .bat-good { color: var(--success-500); background: rgba(16, 185, 129, 0.10); }

  @media (prefers-reduced-motion: reduce) {
    .user-depth-card {
      animation: none;
      animation-delay: 0ms;
    }
    .row-chevron { transition: none; }
  }
</style>
