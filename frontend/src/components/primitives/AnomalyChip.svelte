<script>
  import { createEventDispatcher } from 'svelte';

  // Props
  export let state = null; // 'normal' | 'unusual' | 'unknown' | null
  export let detail = '';

  const dispatch = createEventDispatcher();

  function handleClick() {
    if (state === 'unusual' || state === 'unknown') {
      dispatch('detail', { state, detail });
    }
  }
</script>

{#if state === 'unusual'}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <span class="anomaly-chip chip-unusual" on:click={handleClick} role="button" tabindex="0" aria-label="Unusual activity — tap for detail">
    ⚠ Unusual
  </span>
{:else if state === 'unknown'}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <span class="anomaly-chip chip-unknown" on:click={handleClick} role="button" tabindex="0" aria-label="New — tap for detail">
    New
  </span>
{/if}

<style>
  .anomaly-chip {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 2px 8px;
    border-radius: var(--radius-full);
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.04em;
    cursor: pointer;
    user-select: none;
    animation: sheet-arrive 250ms var(--ease-spring);
    transition: opacity 150ms;
  }
  .anomaly-chip:hover { opacity: 0.82; }

  .chip-unusual {
    background: rgba(245, 158, 11, 0.14);
    border: 1px solid rgba(245, 158, 11, 0.30);
    color: var(--warning-500);
  }
  .chip-unknown {
    background: var(--surface-2);
    border: 1px solid var(--border-default);
    color: var(--text-tertiary);
  }
</style>
