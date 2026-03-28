<script>
  import { createEventDispatcher } from 'svelte';

  export let activeTab = 'track';
  export let isAdmin = false;
  export let isTracking = false;
  export let hasNotification = false;

  const dispatch = createEventDispatcher();
  const tabOrder = ['track', 'people', 'share', 'safety', 'network', 'me'];

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
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M12 20s7-5.2 7-11a7 7 0 1 0-14 0c0 5.8 7 11 7 11z"/><circle cx="12" cy="9" r="2.5" fill="white"/></svg>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M12 20s7-5.2 7-11a7 7 0 1 0-14 0c0 5.8 7 11 7 11z"/><circle cx="12" cy="9" r="2.5"/></svg>
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
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/><path d="M16 3.13a4 4 0 0 1 0 7.75" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
    {:else}
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
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
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/><line x1="12" y1="2" x2="12" y2="15" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"/></svg>
    {:else}
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6"/><line x1="12" y1="2" x2="12" y2="15"/></svg>
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
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3l-8.47-14.14a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13" stroke="white" stroke-width="2" stroke-linecap="round"/><circle cx="12" cy="17" r="1" fill="white"/></svg>
    {:else}
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3l-8.47-14.14a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
    {/if}
    <span class="tab-label">Safety</span>
    {#if hasNotification}
      <span class="tab-dot" aria-label="Notification"></span>
    {/if}
  </button>

  <button
    class="tab-item"
    class:active={activeTab === 'network'}
    on:click={() => selectTab('network')}
    on:keydown={(e) => onTabKeydown(e, 'network')}
    role="tab"
    aria-selected={activeTab === 'network'}
    tabindex={activeTab === 'network' ? 0 : -1}
    aria-label="Network graph"
  >
    {#if activeTab === 'network'}
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><circle cx="12" cy="5" r="2"/><circle cx="5" cy="19" r="2"/><circle cx="19" cy="19" r="2"/><path d="M12 7v4M8.5 16.5 12 11M15.5 16.5 12 11" stroke="white" stroke-width="1.5" stroke-linecap="round" fill="none"/></svg>
    {:else}
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="5" r="2"/><circle cx="5" cy="19" r="2"/><circle cx="19" cy="19" r="2"/><line x1="12" y1="7" x2="12" y2="11"/><line x1="8.5" y1="16.5" x2="12" y2="11"/><line x1="15.5" y1="16.5" x2="12" y2="11"/></svg>
    {/if}
    <span class="tab-label">Network</span>
  </button>

  <button
    class="tab-item"
    class:active={activeTab === 'me'}
    on:click={() => selectTab('me')}
    on:keydown={(e) => onTabKeydown(e, 'me')}
    role="tab"
    aria-selected={activeTab === 'me'}
    tabindex={activeTab === 'me' ? 0 : -1}
    aria-label={isAdmin ? 'Me and admin options' : 'Me and settings'}
  >
    {#if activeTab === 'me'}
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M20 21a8 8 0 0 0-16 0"/><circle cx="12" cy="7" r="4" fill="currentColor" stroke="none"/></svg>
    {:else}
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M20 21a8 8 0 0 0-16 0"/><circle cx="12" cy="7" r="4"/></svg>
    {/if}
    <span class="tab-label">Me</span>
  </button>
</div>

<style>
  .bottom-tabs {
    display: flex;
    align-items: stretch;
    justify-content: space-around;
    background: rgba(5, 5, 15, 0.88);
    backdrop-filter: blur(28px) saturate(1.6);
    -webkit-backdrop-filter: blur(28px) saturate(1.6);
    border-top: 1px solid rgba(255, 255, 255, 0.07);
    box-shadow: 0 -4px 32px rgba(0, 0, 0, 0.45), inset 0 1px 0 rgba(255,255,255,0.05);
    padding-bottom: var(--safe-bottom);
    z-index: var(--z-navbar);
    position: relative;
    flex-shrink: 0;
  }

  :global([data-theme="light"]) .bottom-tabs {
    background: rgba(255, 255, 255, 0.90);
    border-top-color: var(--border-default);
    box-shadow: var(--shadow-sheet);
  }

  /* MERIDIAN: Sliding pill indicator — solid indigo with glow */
  .tab-pill {
    position: absolute;
    top: 6px;
    width: calc(16.667% - 8px); /* 100%/6 tabs minus padding */
    bottom: calc(6px + var(--safe-bottom, 0px));
    background: rgba(99, 102, 241, 0.16);
    border: 1px solid rgba(99, 102, 241, 0.30);
    border-radius: var(--radius-lg);
    pointer-events: none;
    transition: left 350ms cubic-bezier(0.34, 1.56, 0.64, 1); /* spring easing */
    box-shadow: 0 0 16px rgba(99, 102, 241, 0.20);
    z-index: 0;
  }

  .tab-item {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
    padding: var(--space-1-5) 0;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-tertiary);
    transition:
      color var(--duration-fast) var(--ease-out),
      transform var(--duration-fast) var(--ease-spring);
    position: relative;
    z-index: 1;
    min-height: var(--bottom-tab-height);
    min-width: 48px;
    -webkit-tap-highlight-color: transparent;
  }

  .tab-item:active {
    transform: scale(0.90);
    transition-duration: 80ms;
  }

  .tab-item.active {
    color: var(--primary-400);
  }

  .tab-label {
    font-size: var(--text-2xs);
    font-weight: 600;
    letter-spacing: 0.02em;
  }

  .tab-icon-wrap {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  .tracking-dot {
    position: absolute;
    top: -2px;
    right: -2px;
    width: 7px;
    height: 7px;
    background: var(--success-500);
    border-radius: 50%;
    border: 1.5px solid transparent;
    animation: tracking-pulse 2s ease infinite;
  }

  .tracking-active { color: var(--success-400); }

  .tab-dot {
    position: absolute;
    top: 6px;
    right: calc(50% - 14px);
    width: 6px;
    height: 6px;
    background: var(--danger-500);
    border-radius: 50%;
    border: 1.5px solid transparent;
  }

  @keyframes tracking-pulse {
    0%, 100% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.5); }
    50%       { box-shadow: 0 0 0 5px rgba(16, 185, 129, 0); }
  }

  @media (min-width: 768px) {
    .bottom-tabs { display: none; }
  }
</style>
