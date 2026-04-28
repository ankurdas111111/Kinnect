<script>
  import { createEventDispatcher } from 'svelte';

  export let activeTab = 'track';
  export let isAdmin = false;
  export let isTracking = false;
  export let hasNotification = false;

  const dispatch = createEventDispatcher();
  const tabOrder = ['track', 'people', 'share', 'safety', 'me'];

  function selectTab(tab) {
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

  // Sliding pill — calculates horizontal offset by tab index
  $: activeIndex = tabOrder.indexOf(activeTab);
  $: pillOffset = activeIndex >= 0 ? activeIndex * (100 / tabOrder.length) : 0;
</script>

<div class="bottom-tabs" role="tablist" aria-label="Navigation">
  <!-- MERIDIAN: Sliding pill indicator — springs between tabs -->
  <div
    class="tab-pill"
    aria-hidden="true"
    style="left: calc({pillOffset}% + 4px)"
  ></div>
  <button
    class="tab-item"
    class:active={activeTab === 'track'}
    class:tracking-active={isTracking}
    on:click={() => selectTab('track')}
    on:keydown={(e) => onTabKeydown(e, 'track')}
    role="tab"
    aria-selected={activeTab === 'track'}
    tabindex={activeTab === 'track' ? 0 : -1}
    aria-label={isTracking ? 'Track, tracking active' : 'Track'}
  >
    <div class="tab-icon-wrap">
      {#if activeTab === 'track'}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M12 20s7-5.2 7-11a7 7 0 1 0-14 0c0 5.8 7 11 7 11z"/><circle cx="12" cy="9" r="2.5" fill="white"/></svg>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M12 20s7-5.2 7-11a7 7 0 1 0-14 0c0 5.8 7 11 7 11z"/><circle cx="12" cy="9" r="2.5"/></svg>
      {/if}
      {#if isTracking}
        <span class="tracking-dot" aria-hidden="true"></span>
      {/if}
    </div>
    <span class="tab-label">Track</span>
  </button>

  <button
    class="tab-item"
    class:active={activeTab === 'people'}
    on:click={() => selectTab('people')}
    on:keydown={(e) => onTabKeydown(e, 'people')}
    role="tab"
    aria-selected={activeTab === 'people'}
    tabindex={activeTab === 'people' ? 0 : -1}
    aria-label="People"
  >
    {#if activeTab === 'people'}
      <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/><path d="M16 3.13a4 4 0 0 1 0 7.75" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
    {:else}
      <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
    {/if}
    <span class="tab-label">People</span>
  </button>

  <button
    class="tab-item"
    class:active={activeTab === 'share'}
    on:click={() => selectTab('share')}
    on:keydown={(e) => onTabKeydown(e, 'share')}
    role="tab"
    aria-selected={activeTab === 'share'}
    tabindex={activeTab === 'share' ? 0 : -1}
    aria-label="Share"
  >
    {#if activeTab === 'share'}
      <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/><line x1="12" y1="2" x2="12" y2="15" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"/></svg>
    {:else}
      <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6"/><line x1="12" y1="2" x2="12" y2="15"/></svg>
    {/if}
    <span class="tab-label">Share</span>
  </button>

  <button
    class="tab-item"
    class:active={activeTab === 'safety'}
    on:click={() => selectTab('safety')}
    on:keydown={(e) => onTabKeydown(e, 'safety')}
    role="tab"
    aria-selected={activeTab === 'safety'}
    tabindex={activeTab === 'safety' ? 0 : -1}
    aria-label="Safety"
  >
    {#if activeTab === 'safety'}
      <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3l-8.47-14.14a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13" stroke="white" stroke-width="2" stroke-linecap="round"/><circle cx="12" cy="17" r="1" fill="white"/></svg>
    {:else}
      <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3l-8.47-14.14a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
    {/if}
    <span class="tab-label">Safety</span>
    {#if hasNotification}
      <span class="tab-dot" aria-label="Notification"></span>
    {/if}
  </button>

  <button
    class="tab-item"
    class:active={activeTab === 'me'}
    on:click={() => selectTab('me')}
    on:keydown={(e) => onTabKeydown(e, 'me')}
    role="tab"
    aria-selected={activeTab === 'me'}
    tabindex={activeTab === 'me' ? 0 : -1}
    aria-label={isAdmin ? 'Profile and admin options' : 'Profile and settings'}
  >
    {#if activeTab === 'me'}
      <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M20 21a8 8 0 0 0-16 0"/><circle cx="12" cy="7" r="4" fill="currentColor" stroke="none"/></svg>
    {:else}
      <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M20 21a8 8 0 0 0-16 0"/><circle cx="12" cy="7" r="4"/></svg>
    {/if}
    <span class="tab-label">Profile</span>
  </button>
</div>

<style>
  /* Fix #3A: Replace fixed margin: 0 10px with padding-based safe area insets
     so iPad landscape horizontal safe areas are handled correctly. */
  .bottom-tabs {
    display: flex;
    align-items: stretch;
    justify-content: space-around;
    /* Solid surface — no backdrop-filter to prevent GPU stacking with top bar + panels */
    background: rgba(9, 9, 20, 0.97);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-bottom: none;
    box-shadow: 0 -2px 16px rgba(0, 0, 0, 0.30), inset 0 1px 0 rgba(255, 255, 255, 0.07);
    /* Island effect — margin replaced by padding to accommodate horizontal safe areas */
    margin: 0;
    padding-left: max(10px, env(safe-area-inset-left, 0px));
    padding-right: max(10px, env(safe-area-inset-right, 0px));
    border-radius: 22px 22px 0 0;
    padding-bottom: var(--safe-bottom);
    z-index: var(--z-navbar);
    position: relative;
    flex-shrink: 0;
  }

  :global([data-theme="light"]) .bottom-tabs {
    background: rgba(252, 252, 255, 0.97);
    border-color: rgba(0, 0, 0, 0.06);
    box-shadow: 0 -2px 16px rgba(0, 0, 0, 0.08), inset 0 1px 0 rgba(255, 255, 255, 0.70);
  }

  /* ── Sliding pill — spring motion ──────────────────────────────────── */
  .tab-pill {
    position: absolute;
    top: 7px;
    width: calc(20% - 10px);
    bottom: calc(7px + var(--safe-bottom, 0px));
    background: rgba(20, 184, 166, 0.22);
    border: 1px solid rgba(20, 184, 166, 0.40);
    border-top-color: rgba(20, 184, 166, 0.55);
    border-radius: var(--radius-lg);
    pointer-events: none;
    transition: left 400ms cubic-bezier(0.34, 1.56, 0.64, 1);
    box-shadow:
      0 4px 16px rgba(20, 184, 166, 0.30),
      inset 0 1px 0 rgba(255, 255, 255, 0.12);
    z-index: 0;
  }

  :global([data-theme="light"]) .tab-pill {
    background: rgba(20, 184, 166, 0.12);
    border-color: rgba(20, 184, 166, 0.30);
    box-shadow:
      0 4px 12px rgba(20, 184, 166, 0.18),
      inset 0 1px 0 rgba(255, 255, 255, 0.30);
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
    min-height: var(--bottom-tab-height);
    min-width: 44px;
    -webkit-tap-highlight-color: transparent;
    /* Display font for tab labels */
    font-family: var(--font-display);
  }

  .tab-item:hover {
    color: rgba(255, 255, 255, 0.65);
  }

  :global([data-theme="light"]) .tab-item:hover {
    color: var(--text-secondary);
  }

  .tab-item:active {
    transform: scale(0.86);
    transition-duration: 70ms;
  }

  /* Active tab — primary color + subtle icon scale */
  .tab-item.active {
    color: var(--primary-400);
  }

  :global([data-theme="light"]) .tab-item.active {
    color: var(--primary-600);
  }

  /* Icon bounce on tab activation */
  .tab-item.active .tab-icon-wrap {
    animation: tab-bounce 340ms var(--ease-spring) both;
  }

  /* ── Label ─────────────────────────────────────────────────────────────── */
  .tab-label {
    font-size: var(--text-2xs); /* 11px — legible minimum */
    font-weight: 700;
    letter-spacing: 0.025em;
    line-height: 1;
  }

  /* Fix #3C: Slightly larger label on wider phones (390px+, e.g. iPhone 14 Pro)
     where 5 labels have more horizontal breathing room. */
  @media (min-width: 390px) {
    .tab-label {
      font-size: 12px;
    }
  }

  /* ── Icon wrapper ──────────────────────────────────────────────────────── */
  .tab-icon-wrap {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  /* ── Tracking active state ─────────────────────────────────────────────── */
  /* Aurora green pulse — more cinematic than simple dot pulse */
  .tracking-dot {
    position: absolute;
    top: -3px;
    right: -3px;
    width: 8px;
    height: 8px;
    background: var(--success-400);
    border-radius: 50%;
    border: 2px solid rgba(5, 5, 18, 0.92);
    animation: aurora-pulse 2s ease-in-out infinite;
  }

  :global([data-theme="light"]) .tracking-dot {
    border-color: rgba(252, 252, 255, 0.94);
  }

  /* Tracking tab: entire tab shifts to success green */
  .tracking-active {
    color: var(--success-400) !important;
  }

  /* ── Notification dot ──────────────────────────────────────────────────── */
  .tab-dot {
    position: absolute;
    top: 4px;
    right: calc(50% - 15px);
    width: 7px;
    height: 7px;
    background: var(--danger-500);
    border-radius: 50%;
    border: 2px solid rgba(5, 5, 18, 0.92);
    box-shadow: 0 0 6px rgba(239, 68, 68, 0.50);
  }

  :global([data-theme="light"]) .tab-dot {
    border-color: rgba(252, 252, 255, 0.94);
  }

  /* ── Keyframe ──────────────────────────────────────────────────────────── */
  @keyframes tab-bounce {
    0%   { transform: scale(1) translateY(0); }
    35%  { transform: scale(1.22) translateY(-3px); }
    65%  { transform: scale(0.94) translateY(0); }
    100% { transform: scale(1) translateY(0); }
  }

  @media (min-width: 768px) {
    .bottom-tabs { display: none; }
  }
</style>
