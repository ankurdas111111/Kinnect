<script>
  /**
   * RetentionPillGroup — segmented pill selector for offline-visibility retention period.
   *
   * Props:
   *   value    — string ($bindable) — active retention key ('default'|'48h'|'5d'|'10d'|'30d')
   *   onchange — (value: string) => void — fires after internal state commits
   *
   * A11y: radiogroup + radio pattern; keyboard nav via arrow keys inside the group.
   */

  const PILLS = [
    { key: 'default', label: '1 Day'  },
    { key: '48h',     label: '2 Days' },
    { key: '5d',      label: '5 Days' },
    { key: '10d',     label: '10 Days'},
    { key: '30d',     label: '30 Days'},
  ];

  /** @type {{ value?: string, onchange?: (v: string) => void }} */
  let { value = $bindable('default'), onchange } = $props();

  function select(key) {
    value = key;
    onchange?.(key);
  }

  function onkeydown(e, key, idx) {
    let next = -1;
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
      next = (idx + 1) % PILLS.length;
    } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
      next = (idx - 1 + PILLS.length) % PILLS.length;
    }
    if (next >= 0) {
      e.preventDefault();
      select(PILLS[next].key);
      // Move DOM focus to the newly selected pill
      const group = e.currentTarget.closest('[role="radiogroup"]');
      group?.querySelectorAll('[role="radio"]')[next]?.focus();
    }
  }
</script>

<div class="rpg" role="radiogroup" aria-label="Offline visibility retention period">
  {#each PILLS as pill, idx}
    <button
      type="button"
      role="radio"
      aria-checked={value === pill.key}
      tabindex={value === pill.key ? 0 : -1}
      class="rpg-pill"
      class:active={value === pill.key}
      onclick={() => select(pill.key)}
      onkeydown={(e) => onkeydown(e, pill.key, idx)}
    >{pill.label}</button>
  {/each}
</div>

<style>
  .rpg {
    display: flex;
    gap: var(--space-1-5);
    flex-wrap: wrap;
  }

  .rpg-pill {
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-xs);
    font-weight: 600;
    border: 1px solid var(--border-default);
    border-radius: var(--radius-full);
    background: var(--surface-1);
    color: var(--text-secondary);
    cursor: pointer;
    white-space: nowrap;
    min-height: 44px;
    min-width: 44px;
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
    transition:
      background var(--duration-fast, 100ms) var(--ease-out, ease-out),
      color      var(--duration-fast, 100ms) var(--ease-out, ease-out),
      transform  200ms var(--ease-spring, cubic-bezier(0.34,1.56,0.64,1));
  }

  .rpg-pill:hover {
    background: var(--surface-2);
    color: var(--text-primary);
  }

  .rpg-pill.active {
    background: var(--primary-600);
    color: white; /* raw-color-ok — white is the universal contrast pair for primary-600 */
    border-color: var(--primary-500);
    box-shadow: var(--glow-primary, none), var(--shadow-xs, none);
    transform: scale(1.04);
  }

  .rpg-pill:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }

  @media (prefers-reduced-motion: reduce) {
    .rpg-pill {
      transition: none;
    }
    .rpg-pill.active {
      transform: none;
    }
  }
</style>
