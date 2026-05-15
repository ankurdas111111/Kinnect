<script>
  import { createEventDispatcher } from 'svelte';
  import { haptics } from '../../lib/haptics.js';

  export let activeTab = 'track';
  export let isAdmin = false;
  export let isTracking = false;
  export let hasNotification = false;

  const dispatch = createEventDispatcher();
  const tabOrder = ['track', 'people', 'share', 'safety', 'me'];

  function selectTab(tab) {
    if (tab === 'safety') haptics.warning?.();
    else if (tab === 'people' || tab === 'share') haptics.confirm?.();
    else haptics.tap?.();
    dispatch('tabChange', tab);
  }

  const tabMeta = {
    track:  { label: 'Map',     ariaBase: 'Map' },
    people: { label: 'People',  ariaBase: 'People' },
    share:  { label: 'Connect', ariaBase: 'Connect' },
    safety: { label: 'Safety',  ariaBase: 'Safety' },
    me:     { label: 'Profile', ariaBase: 'Profile' },
  };

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

  $: activeIndex = tabOrder.indexOf(activeTab);
  $: pillOffset = activeIndex >= 0 ? activeIndex * (100 / tabOrder.length) : 0;
</script>

<div class="bottom-tabs" role="tablist" aria-label="Navigation">
  <!-- Holographic sliding pill indicator — spring physics -->
  <div
    class="tab-pill"
    aria-hidden="true"
    style="left: 4px; transform: translateX({pillOffset}%)"
  ></div>
  <!-- Pill glow layer — separate element so it can have blur without affecting pill content -->
  <div
    class="tab-pill-glow"
    aria-hidden="true"
    style="left: 4px; transform: translateX({pillOffset}%)"
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
    aria-label={isTracking ? 'Map, tracking active' : 'Map'}
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
    <span class="tab-label">Map</span>
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
    <div class="tab-icon-wrap">
      {#if activeTab === 'people'}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/><path d="M16 3.13a4 4 0 0 1 0 7.75" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
      {/if}
    </div>
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
    aria-label="Connect"
  >
    <div class="tab-icon-wrap">
      {#if activeTab === 'share'}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/><line x1="12" y1="2" x2="12" y2="15" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"/></svg>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6"/><line x1="12" y1="2" x2="12" y2="15"/></svg>
      {/if}
    </div>
    <span class="tab-label">Connect</span>
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
    <div class="tab-icon-wrap">
      {#if activeTab === 'safety'}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="currentColor" stroke="none" aria-hidden="true"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3l-8.47-14.14a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13" stroke="white" stroke-width="2" stroke-linecap="round"/><circle cx="12" cy="17" r="1" fill="white"/></svg>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3l-8.47-14.14a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
      {/if}
    </div>
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
    <div class="tab-icon-wrap">
      {#if activeTab === 'me'}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M20 21a8 8 0 0 0-16 0"/><circle cx="12" cy="7" r="4" fill="currentColor" stroke="none"/></svg>
      {:else}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M20 21a8 8 0 0 0-16 0"/><circle cx="12" cy="7" r="4"/></svg>
      {/if}
    </div>
    <span class="tab-label">Profile</span>
  </button>
</div>

<style>
  .bottom-tabs {
    display: flex;
    align-items: stretch;
    justify-content: space-around;
    /* Deep glass surface — stronger than before */
    background: rgba(7, 8, 18, 0.97);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-bottom: none;
    /* Multi-layer shadow for genuine 3D depth */
    box-shadow:
      0 -1px 0 rgba(255, 255, 255, 0.06),
      0 -4px 24px rgba(0, 0, 0, 0.45),
      0 -12px 40px rgba(0, 0, 0, 0.25),
      inset 0 1px 0 rgba(255, 255, 255, 0.07);
    margin: 0;
    padding-left: max(10px, env(safe-area-inset-left, 0px));
    padding-right: max(10px, env(safe-area-inset-right, 0px));
    border-radius: 22px 22px 0 0;
    padding-bottom: var(--safe-bottom);
    z-index: var(--z-navbar);
    position: relative;
    flex-shrink: 0;
    /* Subtle noise texture backdrop */
    backdrop-filter: blur(2px);
    -webkit-backdrop-filter: blur(2px);
  }

  :global([data-theme="light"]) .bottom-tabs {
    background: rgba(252, 252, 255, 0.97);
    border-color: rgba(0, 0, 0, 0.06);
    box-shadow: 0 -2px 16px rgba(0, 0, 0, 0.08), inset 0 1px 0 rgba(255, 255, 255, 0.70);
  }

  /* ── Holographic sliding pill — spring motion + neon glow ─────────────── */
  .tab-pill {
    position: absolute;
    top: 7px;
    width: calc(20% - 10px);
    bottom: calc(7px + var(--safe-bottom, 0px));
    /* Iridescent teal glass */
    background: linear-gradient(
      135deg,
      rgba(20, 184, 166, 0.24) 0%,
      rgba(16, 185, 129, 0.18) 50%,
      rgba(6, 182, 212, 0.20) 100%
    );
    border: 1px solid rgba(20, 184, 166, 0.45);
    border-top-color: rgba(45, 212, 191, 0.60);
    border-radius: var(--radius-lg);
    pointer-events: none;
    /* Spring physics slide */
    transition: transform 400ms cubic-bezier(0.34, 1.56, 0.64, 1);
    z-index: 0;
    /* Holographic shimmer — traveling highlight */
    overflow: hidden;
  }

  /* Shimmer sweep on pill */
  .tab-pill::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(
      105deg,
      transparent 20%,
      rgba(255,255,255,0.12) 45%,
      rgba(45,212,191,0.10) 55%,
      transparent 80%
    );
    border-radius: inherit;
    transform: translateX(-120%);
    animation: holo-travel 3.5s ease-in-out infinite;
    pointer-events: none;
  }

  /* Glow layer — separate for blur without clipping */
  .tab-pill-glow {
    position: absolute;
    top: 7px;
    width: calc(20% - 10px);
    bottom: calc(7px + var(--safe-bottom, 0px));
    border-radius: var(--radius-lg);
    pointer-events: none;
    transition: transform 400ms cubic-bezier(0.34, 1.56, 0.64, 1);
    z-index: 0;
    box-shadow:
      0 4px 20px rgba(20, 184, 166, 0.38),
      0 0 12px rgba(20, 184, 166, 0.22),
      inset 0 1px 0 rgba(255, 255, 255, 0.14);
    /* Breathe animation for the glow */
    animation: neon-breathe-brand 3s ease-in-out infinite;
  }

  :global([data-theme="light"]) .tab-pill {
    background: rgba(20, 184, 166, 0.12);
    border-color: rgba(20, 184, 166, 0.30);
    box-shadow: 0 4px 12px rgba(20, 184, 166, 0.18), inset 0 1px 0 rgba(255, 255, 255, 0.30);
  }

  :global([data-theme="light"]) .tab-pill-glow {
    box-shadow: 0 4px 14px rgba(20, 184, 166, 0.20);
    animation: none;
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
    font-family: var(--font-display);
    /* Ripple feedback overlay */
    overflow: hidden;
  }

  /* Touch ripple on press */
  .tab-item::before {
    content: '';
    position: absolute;
    width: 36px; height: 36px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.12);
    top: 50%; left: 50%;
    transform: translate(-50%, -50%) scale(0);
    opacity: 0;
    pointer-events: none;
    transition: transform 0ms, opacity 0ms;
  }

  .tab-item:active::before {
    transform: translate(-50%, -50%) scale(1.6);
    opacity: 0;
    transition: transform 400ms cubic-bezier(0.4, 0, 0.2, 1), opacity 400ms ease-out;
  }

  .tab-item:hover {
    color: var(--text-secondary, rgba(255, 255, 255, 0.65));
  }

  :global([data-theme="light"]) .tab-item:hover {
    color: var(--text-secondary);
  }

  .tab-item:active {
    transform: scale(0.86) perspective(400px) translateZ(-4px);
    transition-duration: 70ms;
  }

  .tab-item:focus-visible {
    outline: 2px solid var(--primary-400, #2dd4bf);
    outline-offset: -3px;
    border-radius: var(--radius-md, 8px);
  }

  /* Active tab — neon teal + larger icon */
  .tab-item.active {
    color: var(--primary-400);
    /* Subtle text glow */
    text-shadow: 0 0 10px rgba(45, 212, 191, 0.45);
  }

  :global([data-theme="light"]) .tab-item.active {
    color: var(--primary-600);
    text-shadow: none;
  }

  /* Icon spring bounce on tab activation */
  .tab-item.active .tab-icon-wrap {
    animation: tab-bounce-3d 380ms cubic-bezier(0.34, 1.56, 0.64, 1) both;
  }

  /* Tracking tab: success green tint */
  .tracking-active { color: var(--success-400) !important; text-shadow: 0 0 10px rgba(52, 211, 153, 0.45) !important; }

  /* ── Label ─────────────────────────────────────────────────────────────── */
  .tab-label {
    font-size: var(--text-2xs);
    font-weight: 700;
    letter-spacing: 0.025em;
    line-height: 1;
    transition: opacity var(--duration-fast) var(--ease-out);
  }

  @media (min-width: 390px) {
    .tab-label { font-size: var(--text-xs, 12px); }
  }

  /* ── Icon wrapper ──────────────────────────────────────────────────────── */
  .tab-icon-wrap {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  /* ── Tracking dot — neon aurora pulse ─────────────────────────────────── */
  .tracking-dot {
    position: absolute;
    top: -3px;
    right: -3px;
    width: 8px;
    height: 8px;
    background: var(--success-400);
    border-radius: 50%;
    border: 2px solid rgba(5, 5, 18, 0.92);
    /* Neon glow pulse */
    animation: neon-breathe-live 2s ease-in-out infinite;
  }

  :global([data-theme="light"]) .tracking-dot {
    border-color: rgba(252, 252, 255, 0.94);
    animation: aurora-pulse 2s ease-in-out infinite;
  }

  /* ── Notification dot — neon red ───────────────────────────────────────── */
  .tab-dot {
    position: absolute;
    top: 4px;
    right: calc(50% - 15px);
    width: 8px;
    height: 8px;
    background: var(--danger-500);
    border-radius: 50%;
    border: 2px solid rgba(5, 5, 18, 0.92);
    /* Red neon glow */
    box-shadow: 0 0 6px rgba(239,68,68,0.70), 0 0 14px rgba(239,68,68,0.35);
    animation: neon-breathe-sos 1.8s ease-in-out infinite;
  }

  :global([data-theme="light"]) .tab-dot {
    border-color: rgba(252, 252, 255, 0.94);
    animation: pulse-ring 2s ease-in-out infinite;
  }

  /* ── 2026 Keyframes ─────────────────────────────────────────────────────── */
  @keyframes tab-bounce-3d {
    0%   { transform: scale(1) translateY(0) perspective(400px) translateZ(0); }
    30%  { transform: scale(1.28) translateY(-5px) perspective(400px) translateZ(8px); }
    60%  { transform: scale(0.92) translateY(0) perspective(400px) translateZ(-2px); }
    80%  { transform: scale(1.04) translateY(-1px) perspective(400px) translateZ(2px); }
    100% { transform: scale(1) translateY(0) perspective(400px) translateZ(0); }
  }

  @media (prefers-reduced-motion: reduce) {
    .tab-item.active .tab-icon-wrap { animation: none; }
    .tracking-dot { animation: none; }
    .tab-dot      { animation: none; box-shadow: 0 0 0 2px rgba(239,68,68,0.60); }
    .tab-pill     { transition: none; }
    .tab-pill-glow { animation: none; }
    .tab-pill::after { animation: none; }
    .tab-item.active { text-shadow: none; }
    .tracking-active { text-shadow: none !important; }
  }

  @media (min-width: 768px) {
    .bottom-tabs { display: none; }
  }
</style>
