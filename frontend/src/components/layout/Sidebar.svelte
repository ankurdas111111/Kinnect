<script>
  import { createEventDispatcher } from 'svelte';
  import { fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';

  /**
   * @typedef {Object} Props
   * @property {string} [activeTab]
   * @property {boolean} [isAdmin]
   * @property {boolean} [collapsed]
   * @property {{family?: {tone:string, count:string|null, urgent:boolean}, help?: {active:boolean}, share?: boolean}} [badges]
   * @property {import('svelte').Snippet} [header]
   * @property {import('svelte').Snippet} [children]
   */

  /** @type {Props} */
  let {
    activeTab = $bindable('family'),
    isAdmin = false,
    collapsed = $bindable(false),
    badges = {},
    header,
    children
  } = $props();

  const dispatch = createEventDispatcher();

  // Intent-first IA: what the user is trying to DO. Panel ids are kept where
  // possible (sharing/admin/places/settings) so MainApp wiring survives;
  // 'info' (Status) retires as a top-level tab and lives under Me.
  const tabs = [
    { id: 'family',   label: 'Family', icon: 'users'   },
    { id: 'places',   label: 'Map',    icon: 'map-pin' },
    { id: 'sharing',  label: 'Share',  icon: 'share'   },
    { id: 'admin',    label: 'Help',   icon: 'shield'  },
    { id: 'settings', label: 'Me',     icon: 'settings'},
  ];

  function selectTab(id) {
    if (collapsed) {
      collapsed = false;
      dispatch('toggle', false);
    }
    activeTab = id;
    dispatch('tabChange', id);
  }

  function toggleCollapse() {
    collapsed = !collapsed;
    dispatch('toggle', collapsed);
  }

  function onTabKeydown(e, id) {
    const tabIds = tabs.map(t => t.id);
    const idx = tabIds.indexOf(id);
    if (idx < 0) return;
    var nextIdx = idx;
    if (e.key === 'ArrowRight') nextIdx = (idx + 1) % tabIds.length;
    else if (e.key === 'ArrowLeft') nextIdx = (idx - 1 + tabIds.length) % tabIds.length;
    else if (e.key === 'Home') nextIdx = 0;
    else if (e.key === 'End') nextIdx = tabIds.length - 1;
    else return;
    e.preventDefault();
    selectTab(tabIds[nextIdx]);
  }
</script>

<aside
  class="sidebar"
  class:collapsed
  aria-label="Side panel"
  transition:fly={{ x: -420, duration: 250, easing: cubicOut }}
>
  {#if header}
    <div class="sidebar-header">
      {@render header()}
    </div>
  {/if}

  <div class="sidebar-tabs" role="tablist" aria-label="Panel tabs">
    {#each tabs as tab}
      <button
        class="sidebar-tab"
        class:active={activeTab === tab.id && !collapsed}
        onclick={() => selectTab(tab.id)}
        onkeydown={(e) => onTabKeydown(e, tab.id)}
        role="tab"
        aria-selected={activeTab === tab.id && !collapsed}
        tabindex={activeTab === tab.id && !collapsed ? 0 : -1}
        title={tab.label}
        aria-label={tab.id === 'admin' ? 'Help and safety rules' : tab.id === 'family' ? 'Family status' : tab.label}
      >
        {#if tab.icon === 'users'}
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
        {:else if tab.icon === 'share'}
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/><polyline points="16 6 12 2 8 6"/><line x1="12" y1="2" x2="12" y2="15"/></svg>
        {:else if tab.icon === 'shield'}
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
        {:else if tab.icon === 'map-pin'}
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
        {:else if tab.icon === 'settings'}
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
        {/if}
        {#if !collapsed}
          <span class="sidebar-tab-label">{tab.label}</span>
        {/if}
        {#if tab.id === 'family' && badges.family && badges.family.tone !== 'safe'}
          <span class="sidebar-tab-badge" class:urgent={badges.family.urgent} aria-hidden="true"></span>
        {:else if tab.id === 'family' && badges.family?.count}
          <span class="sidebar-tab-count" aria-hidden="true">{badges.family.count}</span>
        {:else if tab.id === 'admin' && badges.help?.active}
          <span class="sidebar-tab-badge urgent" aria-hidden="true"></span>
        {:else if tab.id === 'sharing' && badges.share}
          <span class="sidebar-tab-badge caution" aria-hidden="true"></span>
        {/if}
      </button>
    {/each}
  </div>

  {#if !collapsed}
    <div class="sidebar-content" role="tabpanel" aria-label={activeTab}>
      {@render children?.()}
    </div>
  {/if}

  <button class="sidebar-collapse-btn btn btn-icon btn-ghost" onclick={toggleCollapse} title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'} aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}>
    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="transform:{collapsed ? 'rotate(180deg)' : 'none'};transition:transform 0.2s">
      <polyline points="15 18 9 12 15 6"/>
    </svg>
  </button>
</aside>

<style>
  .sidebar {
    display: none;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    background:
      linear-gradient(var(--amb-warmth), var(--amb-warmth)),
      var(--glass-nav-bg);
    backdrop-filter: var(--glass-nav-blur);
    -webkit-backdrop-filter: var(--glass-nav-blur);
    border-right: 1px solid var(--border-default);
    box-shadow: var(--shadow-panel);
    overflow: hidden;
    position: relative;
    width: var(--sidebar-width);
    transition: width var(--duration-normal) var(--ease-out);
    z-index: var(--z-panel);
  }

  .sidebar.collapsed {
    width: var(--sidebar-collapsed);
  }

  .sidebar-tabs {
    display: flex;
    gap: var(--space-1);
    padding: var(--space-2);
    border-bottom: 1px solid var(--border-default);
    flex-shrink: 0;
    overflow-x: auto;
    overflow-y: hidden;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: thin;
    scrollbar-color: var(--border-default) transparent;
  }

  .sidebar-tabs::-webkit-scrollbar {
    height: 4px;
  }

  .sidebar-tabs::-webkit-scrollbar-track {
    background: transparent;
  }

  .sidebar-tabs::-webkit-scrollbar-thumb {
    background: var(--border-default);
    border-radius: 2px;
  }

  .sidebar-tabs::-webkit-scrollbar-thumb:hover {
    background: var(--text-secondary);
  }

  .sidebar-tab {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
    border: none;
    background: none;
    cursor: pointer;
    color: var(--text-secondary);
    font-size: var(--text-sm);
    font-weight: 600;
    border-radius: var(--radius-md);
    transition:
      color var(--duration-fast) var(--ease-out),
      background-color var(--duration-fast) var(--ease-out);
    white-space: nowrap;
    min-height: 44px;
  }

  .sidebar-tab:hover {
    color: var(--text-primary);
    background: var(--surface-hover);
  }

  .sidebar-tab.active {
    color: var(--primary-600);
    background: var(--surface-selected);
  }

  :global([data-theme="dark"]) .sidebar-tab.active {
    color: var(--primary-400);
  }

  .sidebar-tab-label {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sidebar-header {
    padding: var(--space-2) var(--space-2) 0;
    flex-shrink: 0;
  }

  /* Tone badges — dot pairs with the tab label (never color alone) */
  .sidebar-tab-badge {
    width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0;
    background: var(--warning-500);
  }
  .sidebar-tab-badge.urgent { background: var(--danger-500); }
  .sidebar-tab-badge.caution { background: var(--warning-500); }
  .sidebar-tab-count {
    min-width: 16px; height: 16px; padding: 0 4px;
    display: inline-flex; align-items: center; justify-content: center;
    background: var(--primary-500); color: var(--text-on-primary);
    border-radius: var(--radius-full, 999px);
    font-size: 9px; font-weight: 800; line-height: 1;
    font-variant-numeric: tabular-nums;
  }

  .collapsed .sidebar-tabs {
    flex-direction: column;
    align-items: center;
  }

  .collapsed .sidebar-tab {
    padding: var(--space-2);
    justify-content: center;
  }

  .sidebar-content {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
    -webkit-overflow-scrolling: touch;
  }

  .sidebar-collapse-btn {
    position: absolute;
    bottom: var(--space-3);
    right: var(--space-2);
    width: 36px;
    height: 36px;
    min-width: 36px;
    min-height: 36px;
    border-radius: 50%;
    opacity: 0.6;
  }

  .sidebar-collapse-btn:hover {
    opacity: 1;
  }

  @media (min-width: 768px) {
    .sidebar {
      display: flex;
    }
  }

  @media (min-width: 768px) and (max-width: 1023px) {
    .sidebar {
      width: var(--sidebar-tablet);
    }
    .sidebar.collapsed {
      width: var(--sidebar-collapsed);
    }
  }

  /*
   * Desktop: width is driven by var(--sidebar-width), which AppLayout
   * overrides to clamp(300px, 23vw, 420px) at >=1024px. The inline clamp
   * fallback keeps the sidebar proportional even if this component is
   * ever rendered outside AppLayout.
   */
  @media (min-width: 1024px) {
    .sidebar {
      width: var(--sidebar-width, clamp(300px, 23vw, 420px));
    }
    .sidebar.collapsed {
      width: var(--sidebar-collapsed, 56px);
    }
  }
</style>
