<script>
  import { createEventDispatcher } from 'svelte';
  import EmptyState from '../primitives/EmptyState.svelte';
  import GhostConstellation from '../primitives/GhostConstellation.svelte';

  /**
   * @typedef {Object} Props
   * @property {boolean} [solo] - Solo mode — self is visible but nobody else has joined yet.
   */

  /** @type {Props} */
  let { solo = false } = $props();

  const dispatch = createEventDispatcher();
</script>

{#if solo}
  <!-- Just self is visible — prompt to invite, with a concrete next step -->
  <EmptyState
    title="You're the only one here"
    body="Invite family or friends and they'll appear here the moment they connect."
  >
    {#snippet icon()}
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
    {/snippet}
    {#snippet action()}
      <button class="empty-cta" onclick={() => dispatch('addPeople')}>
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/></svg>
        Invite people
      </button>
    {/snippet}
  </EmptyState>
{:else}
  <!-- Full empty state — the ghost constellation previews the filled state -->
  <GhostConstellation
    title="Your people will appear here"
    body="Share your code with friends or family so you can see each other on the map"
    ctaLabel="Add people"
    oninvite={() => dispatch('addPeople')}
  />
{/if}

<style>
  /* CTA button in the solo empty state */
  .empty-cta {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1-5);
    margin-top: var(--space-3);
    padding: var(--space-2-5) var(--space-4);
    min-height: 44px;
    background: linear-gradient(135deg, var(--primary-500, #14b8a6), var(--primary-700, #0f766e));
    color: white;
    border: none;
    border-radius: var(--radius-lg);
    font-family: var(--font-display);
    font-size: var(--text-sm);
    font-weight: 700;
    cursor: pointer;
    box-shadow: 0 4px 14px rgba(20, 184, 166, 0.30);
    transition: transform var(--duration-fast) var(--ease-spring), box-shadow var(--duration-fast);
    -webkit-tap-highlight-color: transparent;
  }
  .empty-cta:hover { transform: translateY(-1px); box-shadow: 0 6px 20px rgba(20, 184, 166, 0.40); }
  .empty-cta:active { transform: scale(0.97); }

  @media (prefers-reduced-motion: reduce) {
    .empty-cta:hover { transform: none; }
  }
</style>
