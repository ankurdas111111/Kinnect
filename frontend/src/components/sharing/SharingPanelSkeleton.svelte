<script>
  /**
   * SharingPanelSkeleton — loading placeholder for the rooms/contacts fetch.
   * Skeleton-first (design rules: no center-of-card spinners). Renders a couple of
   * card-shaped shimmers per section so the layout doesn't jump when data lands.
   */
  import Skeleton from '../primitives/Skeleton.svelte';
</script>

<div class="sharing-skeleton" aria-hidden="true">
  <div class="sk-block">
    <Skeleton variant="line" width="30%" />
    <Skeleton variant="card" count={2} />
  </div>
  <div class="sk-block">
    <Skeleton variant="line" width="30%" />
    <Skeleton variant="card" count={2} />
  </div>
</div>

<style>
  .sharing-skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }
  .sk-block {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
</style>
