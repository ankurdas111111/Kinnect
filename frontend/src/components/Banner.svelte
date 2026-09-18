<script>
  import { banner } from '../lib/stores/sos.js';
</script>

{#if $banner.text}
  <div class="banner" class:banner-info={$banner.type === 'info'} class:banner-sos={$banner.type === 'sos'} role="status" aria-live="polite">
    <span class="banner-text">{$banner.text}</span>
    {#if $banner.actions}
      {#each $banner.actions as action}
        <button class="btn btn-sm banner-action {action.kind || 'btn-secondary'}" onclick={action.onClick}>{action.label}</button>
      {/each}
    {/if}
    <button class="banner-close" aria-label="Dismiss banner" onclick={() => banner.set({ type: null, text: null, actions: [] })} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); banner.set({ type: null, text: null, actions: [] }); } }}>
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
    </button>
  </div>
{/if}

<style>
  .banner {
    position: fixed;
    top: calc(var(--safe-top, 0px) + var(--navbar-height, 56px));
    /* Centered fit-content pill — a full-width strip dead-zoned the sidebar
       tabs and search underneath it while visible. */
    left: 50%;
    transform: translateX(-50%);
    width: max-content;
    max-width: min(calc(100vw - 2 * var(--space-4)), 560px);
    z-index: calc(var(--z-navbar, 2000) + 500); /* above navbar, below overlay */
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-3);
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-sm);
    font-weight: 500;
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-lg);
    animation: banner-in-top 0.3s var(--ease-out);
  }

  /* Mobile: position above bottom tab bar */
  @media (max-width: 767px) {
    .banner {
      top: auto;
      bottom: calc(var(--bottom-tab-height, 56px) + var(--safe-bottom, 0px) + var(--space-3));
      animation: banner-in-bottom 0.3s var(--ease-out);
    }
  }

  /* Hearth: informational banners are quiet paper with an ember edge, not a
     loud blue bar — the palette allows exactly one accent, and blue isn't it. */
  .banner-info {
    background: var(--surface-1);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    color: var(--text-primary);
    border: 1px solid var(--border-default);
    border-left: 3px solid var(--primary-500);
  }
  /* SOS keeps the full vermilion fill — the one place the app goes red. */
  .banner-sos {
    background: var(--danger-600);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    color: var(--text-inverse);
    border: 1px solid color-mix(in oklch, var(--danger-500) 60%, transparent);
    animation: banner-in-top 0.3s var(--ease-out), sos-urgent-pulse 1.5s ease infinite;
  }
  .banner-text { flex: 1; text-align: center; }
  .banner-close {
    background: none;
    border: none;
    cursor: pointer;
    color: inherit;
    width: 40px;
    height: 40px;
    border-radius: 12px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    opacity: 0.7;
  }
  .banner-close:hover { opacity: 1; }
  .banner-action {
    min-height: 40px;
    padding: 0 12px;
  }
  /* Keyframes carry the centering translateX so the entrance can't undo it */
  @keyframes banner-in-top {
    from { transform: translate(-50%, -20px); opacity: 0; }
    to { transform: translate(-50%, 0); opacity: 1; }
  }
  @keyframes banner-in-bottom {
    from { transform: translate(-50%, 20px); opacity: 0; }
    to { transform: translate(-50%, 0); opacity: 1; }
  }

  @media (prefers-reduced-motion: reduce) {
    .banner, .banner-sos { animation: none; }
  }
</style>
