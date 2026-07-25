<script>
  /**
   * SidebarLinkRow — 44px quick-link row shared by the desktop sidebar and the
   * mobile sheet panels. Surfaces secondary pages ≤1 tap from their intent
   * home (the intent-first IA rule). Pure presentation + push().
   */
  import { push } from 'svelte-spa-router';

  /** @type {{ links: Array<{ label: string, route: string, icon?: 'hub'|'activity'|'emergency'|'checkins'|'routes'|'monitoring' }> }} */
  let { links = [] } = $props();
</script>

<div class="link-row" role="group" aria-label="Quick links">
  {#each links as l (l.route)}
    <button class="link-btn tactile" onclick={() => push(l.route)} aria-label="Open {l.label}">
      {#if l.icon === 'hub'}
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
      {:else if l.icon === 'activity'}
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
      {:else if l.icon === 'emergency'}
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
      {:else if l.icon === 'checkins'}
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
      {:else if l.icon === 'routes'}
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 .49-4.5"/></svg>
      {:else if l.icon === 'monitoring'}
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
      {/if}
      {l.label}
    </button>
  {/each}
</div>

<style>
  .link-row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
  }
  .link-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    min-height: 44px;
    padding: 0 var(--space-3);
    background: var(--surface-1);
    border: 1px solid var(--border-default, var(--glass-border));
    border-radius: var(--radius-md, 12px);
    color: var(--text-secondary);
    font-family: var(--font-display);
    font-size: var(--text-xs, 12px);
    font-weight: 600;
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out),
                color var(--duration-fast) var(--ease-out),
                border-color var(--duration-fast) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .link-btn:hover {
    background: var(--surface-hover);
    color: var(--text-primary);
    border-color: var(--glass-border-strong);
  }
  .link-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
</style>
