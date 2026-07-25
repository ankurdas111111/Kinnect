<script>
  import { createEventDispatcher } from 'svelte';
  import { haptics } from '../../lib/haptics.js';

  /**
   * BottomTabBar — floating liquid-glass pill (2026 nav).
   *
   * Material: --glass-nav-* tier (tokens-fx.css) — blur + specular top edge +
   * one lift shadow; near-opaque flat under data-fx="minimal". Emotional layer:
   * the active indicator + active ink read --nav-tone-* (verdict tone), and the
   * bar surface carries --amb-warmth (daypart temperature).
   *
   * Motion: ONE spring slide on the indicator (transform-only, GPU) + a
   * one-shot settle on the tapped icon. No infinite loops — decoration that
   * doesn't explain state was deliberately removed (shimmer/neon/breathe).
   */

  /**
   * @typedef {Object} Props
   * @property {string} [activeTab]
   * @property {boolean} [isAdmin]
   * @property {boolean} [isTracking]
   * @property {boolean} [shareNotification]  pending connect/guardian requests
   * @property {{tone:string, count:string|null, urgent:boolean}} [familyBadge]
   * @property {{active:boolean, mine:boolean}} [helpBadge]
   */

  /** @type {Props} */
  let {
    activeTab = 'map',
    isAdmin = false,
    isTracking = false,
    shareNotification = false,
    familyBadge = { tone: 'safe', count: null, urgent: false },
    helpBadge = { active: false, mine: false }
  } = $props();

  const dispatch = createEventDispatcher();
  const tabOrder = ['family', 'map', 'share', 'help', 'me'];

  function selectTab(tab) {
    if (tab === 'help') haptics.warning?.();
    else if (tab === 'family' || tab === 'share') haptics.confirm?.();
    else haptics.tap?.();
    dispatch('tabChange', tab);
  }

  function onTabKeydown(e, tab) {
    const idx = tabOrder.indexOf(tab);
    if (idx < 0) return;
    var nextIdx = idx;
    if (e.key === 'ArrowRight') nextIdx = (idx + 1) % tabOrder.length;
    else if (e.key === 'ArrowLeft') nextIdx = (idx - 1 + tabOrder.length) % tabOrder.length;
    else if (e.key === 'Home') nextIdx = 0;
    else if (e.key === 'End') nextIdx = tabOrder.length - 1;
    else return;
    e.preventDefault();
    selectTab(tabOrder[nextIdx]);
  }

  let activeIndex = $derived(Math.max(0, tabOrder.indexOf(activeTab)));
</script>

<div class="bottom-tabs" role="tablist" aria-label="Navigation" style="--tab-index:{activeIndex}">
  <!-- Single tone-aware indicator — transform-only spring slide -->
  <div class="tab-indicator" aria-hidden="true"></div>

  <button
    class="tab-item"
    class:active={activeTab === 'family'}
    onclick={() => selectTab('family')}
    onkeydown={(e) => onTabKeydown(e, 'family')}
    role="tab"
    aria-selected={activeTab === 'family'}
    tabindex={activeTab === 'family' ? 0 : -1}
    aria-label={familyBadge.tone === 'alert' ? 'Family — needs attention' : familyBadge.tone === 'caution' ? 'Family — check in' : 'Family'}
  >
    <div class="tab-icon-wrap">
      {#if activeTab === 'family'}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/><path d="M16 3.13a4 4 0 0 1 0 7.75" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
      {/if}
      {#if familyBadge.count}
        <span class="tab-count" class:urgent={familyBadge.urgent} aria-hidden="true">{familyBadge.count}</span>
      {:else if familyBadge.tone !== 'safe'}
        <span class="tab-dot" class:tone-caution={familyBadge.tone === 'caution'} aria-hidden="true"></span>
      {/if}
    </div>
    <span class="tab-label">Family</span>
  </button>

  <button
    class="tab-item"
    class:active={activeTab === 'map'}
    class:tracking-active={isTracking}
    onclick={() => selectTab('map')}
    onkeydown={(e) => onTabKeydown(e, 'map')}
    role="tab"
    aria-selected={activeTab === 'map'}
    tabindex={activeTab === 'map' ? 0 : -1}
    aria-label={isTracking ? 'Map, tracking active' : 'Map'}
  >
    <div class="tab-icon-wrap">
      {#if activeTab === 'map'}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M12 20s7-5.2 7-11a7 7 0 1 0-14 0c0 5.8 7 11 7 11z"/><circle cx="12" cy="9" r="2.5" fill="var(--surface-0)"/></svg>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M12 20s7-5.2 7-11a7 7 0 1 0-14 0c0 5.8 7 11 7 11z"/><circle cx="12" cy="9" r="2.5"/></svg>
      {/if}
      {#if isTracking}
        <span class="tracking-dot" aria-hidden="true"></span>
      {/if}
    </div>
    <span class="tab-label">Map</span>
  </button>

  <button
    class="tab-item"
    class:active={activeTab === 'share'}
    onclick={() => selectTab('share')}
    onkeydown={(e) => onTabKeydown(e, 'share')}
    role="tab"
    aria-selected={activeTab === 'share'}
    tabindex={activeTab === 'share' ? 0 : -1}
    aria-label={shareNotification ? 'Share — new request' : 'Share'}
  >
    <div class="tab-icon-wrap">
      {#if activeTab === 'share'}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/><line x1="12" y1="2" x2="12" y2="15" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"/></svg>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6"/><line x1="12" y1="2" x2="12" y2="15"/></svg>
      {/if}
      {#if shareNotification}
        <span class="tab-dot tone-caution" aria-hidden="true"></span>
      {/if}
    </div>
    <span class="tab-label">Share</span>
  </button>

  <button
    class="tab-item"
    class:active={activeTab === 'help'}
    onclick={() => selectTab('help')}
    onkeydown={(e) => onTabKeydown(e, 'help')}
    role="tab"
    aria-selected={activeTab === 'help'}
    tabindex={activeTab === 'help' ? 0 : -1}
    aria-label={helpBadge.active ? (helpBadge.mine ? 'Help — your SOS is active' : 'Help — SOS active') : 'Help'}
  >
    <div class="tab-icon-wrap">
      {#if activeTab === 'help'}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3l-8.47-14.14a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13" stroke="var(--surface-0)" stroke-width="2" stroke-linecap="round"/><circle cx="12" cy="17" r="1" fill="var(--surface-0)"/></svg>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3l-8.47-14.14a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
      {/if}
      {#if helpBadge.active}
        <span class="tab-dot" aria-hidden="true"></span>
      {/if}
    </div>
    <span class="tab-label">Help</span>
  </button>

  <button
    class="tab-item"
    class:active={activeTab === 'me'}
    onclick={() => selectTab('me')}
    onkeydown={(e) => onTabKeydown(e, 'me')}
    role="tab"
    aria-selected={activeTab === 'me'}
    tabindex={activeTab === 'me' ? 0 : -1}
    aria-label={isAdmin ? 'Me — profile and admin options' : 'Me — profile and settings'}
  >
    <div class="tab-icon-wrap">
      {#if activeTab === 'me'}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M20 21a8 8 0 0 0-16 0"/><circle cx="12" cy="7" r="4" fill="currentColor" stroke="none"/></svg>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M20 21a8 8 0 0 0-16 0"/><circle cx="12" cy="7" r="4"/></svg>
      {/if}
    </div>
    <span class="tab-label">Me</span>
  </button>
</div>

<style>
  /* ── Floating liquid-glass pill ─────────────────────────────────────────── */
  .bottom-tabs {
    position: relative;
    display: flex;
    align-items: stretch;
    justify-content: space-around;
    width: min(420px, calc(100vw - 2 * var(--space-4) - var(--safe-left) - var(--safe-right)));
    min-height: var(--bottom-tab-height);
    margin-inline: auto;
    padding: var(--space-1);
    border-radius: var(--radius-full, 999px);
    /* nav tier: daypart-warmed glass + specular top edge + one lift shadow.
       --amb-warmth layers the time-of-day temperature over the material. */
    background:
      linear-gradient(var(--amb-warmth), var(--amb-warmth)),
      var(--glass-nav-bg);
    border: 1px solid var(--glass-nav-border);
    box-shadow: var(--glass-nav-shadow), var(--shadow-3d-float);
    backdrop-filter: var(--glass-nav-blur);
    -webkit-backdrop-filter: var(--glass-nav-blur);
    z-index: var(--z-navbar);
    pointer-events: auto;
  }

  /* Verdict tone wash — opacity-only crossfade in/out of the calm state.
     caution↔alert swaps color instantly (var change) — urgency must not lag. */
  .bottom-tabs::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: inherit;
    background: var(--nav-tone-tint);
    opacity: 0;
    transition: opacity var(--duration-slower) var(--ease-out);
    pointer-events: none;
    z-index: 0;
  }
  :global([data-tone="caution"]) .bottom-tabs::after,
  :global([data-tone="alert"]) .bottom-tabs::after {
    opacity: 1;
  }

  /* ── Active indicator — ONE element, transform-only spring slide ────────── */
  .tab-indicator {
    position: absolute;
    top: var(--space-1);
    bottom: var(--space-1);
    left: var(--space-1);
    width: calc((100% - 2 * var(--space-1)) / 5);
    border-radius: var(--radius-full, 999px);
    background: color-mix(in oklch, var(--nav-tone-accent) 14%, transparent);
    border: 1px solid color-mix(in oklch, var(--nav-tone-accent) 35%, transparent);
    box-shadow: inset 0 1px 0 var(--nav-specular);
    transform: translateX(calc(var(--tab-index, 0) * 100%));
    transition: transform var(--duration-3d, 250ms) var(--ease-spring);
    pointer-events: none;
    z-index: 0;
  }

  /* ── Tab items ──────────────────────────────────────────────────────────── */
  .tab-item {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 3px;
    padding: var(--space-2) 0;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-tertiary);
    transition:
      color var(--duration-fast) var(--ease-out),
      transform 120ms var(--ease-spring);
    position: relative;
    z-index: 1;
    min-height: 44px;
    min-width: 44px;
    -webkit-tap-highlight-color: transparent;
    font-family: var(--font-display);
    border-radius: var(--radius-full, 999px);
    overflow: hidden;
  }

  /* One-shot touch ripple — purposeful press feedback, not decoration */
  .tab-item::before {
    content: '';
    position: absolute;
    width: 36px; height: 36px;
    border-radius: 50%;
    background: color-mix(in oklch, var(--nav-tone-accent) 16%, transparent);
    top: 50%; left: 50%;
    transform: translate(-50%, -50%) scale(0);
    opacity: 0;
    pointer-events: none;
    transition: transform 0ms, opacity 0ms;
  }
  .tab-item:active::before {
    transform: translate(-50%, -50%) scale(1.6);
    opacity: 0;
    transition: transform 400ms var(--ease-out), opacity 400ms var(--ease-out);
  }

  .tab-item:hover { color: var(--text-secondary); }
  .tab-item:active {
    transform: scale(0.92);
    transition-duration: 70ms;
  }
  .tab-item:focus-visible {
    outline: 2px solid var(--nav-tone-accent);
    outline-offset: -3px;
  }

  /* Active tab — tone ink, no glow theatrics */
  .tab-item.active { color: var(--nav-tone-ink); }

  /* One-shot settle on activation — replaces the perspective bounce loop */
  .tab-item.active .tab-icon-wrap {
    animation: tab-settle 300ms var(--ease-spring) both;
  }
  @keyframes tab-settle {
    0%   { transform: scale(1); }
    45%  { transform: scale(1.08) translateY(-2px); }
    100% { transform: scale(1) translateY(0); }
  }

  /* Tracking tab: live tint (kept — it explains state) */
  .tracking-active { color: var(--success-400) !important; }

  .tab-label {
    font-size: var(--text-2xs, 10px);
    font-weight: 700;
    letter-spacing: 0.02em;
    line-height: 1;
  }

  .tab-icon-wrap {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  /* ── Status dots — static glow + one-shot appear (no breathing loops) ───── */
  .tracking-dot {
    position: absolute;
    top: -3px;
    right: -3px;
    width: 10px;
    height: 10px;
    background: var(--success-400);
    border-radius: 50%;
    border: 2px solid var(--surface-0);
    box-shadow: var(--glow-live-sm);
    animation: dot-appear 200ms var(--ease-spring) both;
  }

  .tab-dot {
    position: absolute;
    top: -3px;
    right: -3px;
    width: 8px;
    height: 8px;
    background: var(--danger-500);
    border-radius: 50%;
    border: 2px solid var(--surface-0);
    box-shadow: var(--glow-sos-sm, var(--glow-sos));
    animation: dot-appear 200ms var(--ease-spring) both;
  }
  .tab-dot.tone-caution {
    background: var(--warning-500);
    box-shadow: none;
  }

  /* Unread-count bubble (Family tab) — count = unread events, urgent = SOS */
  .tab-count {
    position: absolute;
    top: -6px;
    right: -10px;
    min-width: 16px;
    height: 16px;
    padding: 0 4px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: var(--primary-500);
    color: var(--text-on-primary);
    border: 2px solid var(--surface-0);
    border-radius: var(--radius-full, 999px);
    font-size: 9px;
    font-weight: 800;
    line-height: 1;
    font-variant-numeric: tabular-nums;
    animation: dot-appear 200ms var(--ease-spring) both;
  }
  .tab-count.urgent { background: var(--danger-500); }

  @keyframes dot-appear {
    from { transform: scale(0); }
    to   { transform: scale(1); }
  }

  @media (prefers-reduced-motion: reduce) {
    .tab-indicator { transition: none; }
    .tab-item.active .tab-icon-wrap { animation: none; }
    .tracking-dot, .tab-dot, .tab-count { animation: none; }
    .tab-item:active { transform: none; }
    .bottom-tabs::after { transition: none; }
  }

  @media (min-width: 768px) {
    .bottom-tabs { display: none; }
  }
</style>
