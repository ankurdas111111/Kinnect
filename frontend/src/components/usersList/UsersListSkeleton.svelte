<!-- Skeleton rows shown while the first people sync is still in flight -->
<div class="skeleton-list" role="status" aria-label="Loading people" aria-busy="true">
  {#each [0, 1, 2] as i}
    <div class="skel-item" style="--skel-i: {i}">
      <span class="skel-avatar"></span>
      <span class="skel-lines">
        <span class="skel-row skel-name"></span>
        <span class="skel-row skel-sub"></span>
      </span>
    </div>
  {/each}
</div>

<style>
  .skeleton-list {
    display: flex;
    flex-direction: column;
  }

  .skel-item {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2-5) var(--space-4);
    min-height: 76px;
  }

  .skel-avatar {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    flex-shrink: 0;
    background: var(--skeleton-base, rgba(255,255,255,0.05));
    animation: skel-pulse var(--skeleton-duration, 1.6s) ease-in-out infinite;
    animation-delay: calc(var(--skel-i, 0) * 120ms);
  }

  .skel-lines {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    min-width: 0;
  }

  .skel-row {
    height: var(--space-3);
    border-radius: var(--radius-sm);
    background: var(--skeleton-base, rgba(255,255,255,0.05));
    animation: skel-pulse var(--skeleton-duration, 1.6s) ease-in-out infinite;
    animation-delay: calc(var(--skel-i, 0) * 120ms);
  }

  .skel-name { width: 45%; }
  .skel-sub  { width: 70%; }

  @keyframes skel-pulse {
    0%, 100% { opacity: 0.5; }
    50%       { opacity: 1; }
  }

  @media (prefers-reduced-motion: reduce) {
    .skel-avatar,
    .skel-row { animation: none; opacity: 0.7; }
  }
</style>
