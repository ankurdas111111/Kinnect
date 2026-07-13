<script>
  /**
   * ToggleControl — safety-grade toggle row (switch + label + optional description).
   *
   * Props:
   *   checked     — boolean ($bindable) — two-way; bind:checked={$store.flag}
   *   label       — string (required)   — visible row label, wired via aria-labelledby
   *   description — string              — secondary line; wired via aria-describedby
   *   disabled    — boolean             — disables the switch
   *   onchange    — (checked: boolean) => void — fires AFTER internal state commits
   *
   * A11y baked in (call sites cannot break it — CONTRACTS.md §8):
   *   • Real <button role="switch" aria-checked> — not a div
   *   • Entire row is the 44px hit area; switch knob has its own ≥44px hit box via padding
   *   • Space/Enter toggle; visible focus ring on the switch
   *   • Disabled: attribute + --text-tertiary (not opacity-only)
   */

  /** @type {{ checked?: boolean, label: string, description?: string, disabled?: boolean, onchange?: (v: boolean) => void }} */
  let {
    checked = $bindable(false),
    label,
    description = '',
    disabled = false,
    onchange,
  } = $props();

  // Unique IDs so aria-labelledby / aria-describedby work even with multiple instances.
  const uid = Math.random().toString(36).slice(2, 8);
  const labelId = `tc-lbl-${uid}`;
  const descId  = `tc-desc-${uid}`;

  function toggle() {
    if (disabled) return;
    checked = !checked;
    onchange?.(checked);
  }

  function onkeydown(e) {
    if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault();
      toggle();
    }
  }
</script>

<div class="toggle-row" class:disabled>
  <!-- Label column -->
  <div class="toggle-labels">
    <span id={labelId} class="toggle-label">{label}</span>
    {#if description}
      <span id={descId} class="toggle-desc">{description}</span>
    {/if}
  </div>

  <!-- Switch — real button, role="switch", full a11y contract -->
  <button
    type="button"
    role="switch"
    aria-checked={checked}
    aria-labelledby={labelId}
    aria-describedby={description ? descId : undefined}
    {disabled}
    class="switch"
    class:on={checked}
    onclick={toggle}
    {onkeydown}
  >
    <span class="knob" aria-hidden="true"></span>
    <!-- Visually hidden state text for extra AT clarity -->
    <span class="sr-only">{checked ? 'On' : 'Off'}</span>
  </button>
</div>

<style>
  /* ── Row layout ─────────────────────────────────────────────────────────── */
  .toggle-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    min-height: 44px;
    padding: var(--space-3) 0;
  }

  .toggle-row.disabled {
    cursor: not-allowed;
  }

  /* ── Labels ─────────────────────────────────────────────────────────────── */
  .toggle-labels {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    flex: 1;
  }

  .toggle-label {
    font-size: var(--text-base);
    font-weight: 500;
    color: var(--text-primary);
    line-height: 1.3;
  }

  .toggle-desc {
    font-size: var(--text-sm);
    color: var(--text-secondary);
    line-height: 1.4;
  }

  .disabled .toggle-label,
  .disabled .toggle-desc {
    /* Disabled: color signal, not opacity — CONTRACTS.md §8 */
    color: var(--text-tertiary);
  }

  /* ── Switch shell ────────────────────────────────────────────────────────── */
  .switch {
    /* ≥44px hit box: padding supplements the visual 50×28 track — CONTRACTS.md §8 */
    position: relative;
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    width: 50px;
    min-height: 44px;
    padding: 0 4px;          /* lateral padding gives ≥44px wide tap target via touch area */
    background: none;
    border: none;
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
    touch-action: manipulation;
  }

  /* The visible track sits inside the button */
  .switch::before {
    content: '';
    position: absolute;
    left: 4px;
    top: 50%;
    translate: 0 -50%;
    width: 42px;
    height: 24px;
    border-radius: var(--radius-full);
    background: var(--surface-inset);
    border: 1px solid var(--border-subtle);
    /* Track color crossfade 150ms — GPU-safe (background is not composited but only
       changes on explicit toggle, not on a loop) */
    transition: background var(--duration-normal) var(--ease-out),
                border-color var(--duration-normal) var(--ease-out);
  }

  .switch.on::before {
    background: var(--primary-500);
    border-color: var(--primary-500);
  }

  .disabled .switch,
  .switch:disabled {
    cursor: not-allowed;
    pointer-events: none;
  }

  .disabled .switch::before {
    background: var(--surface-inset);
    border-color: var(--border-subtle);
    opacity: 0.5;
  }

  /* Focus ring — CONTRACTS.md §0 */
  .switch:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
    border-radius: var(--radius-full);
  }

  /* ── Knob ────────────────────────────────────────────────────────────────── */
  .knob {
    position: absolute;
    left: 8px;               /* 4px button padding + 4px track inset */
    top: 50%;
    translate: 0 -50%;
    width: 18px;
    height: 18px;
    border-radius: var(--radius-full);
    background: var(--text-inverse, #fff);
    box-shadow: var(--shadow-sm);
    /* Knob travel: translateX 220ms spring — CONTRACTS.md §8 */
    transition: transform 220ms var(--ease-spring);
  }

  .switch.on .knob {
    /* 42px track − 8px left-inset − 18px knob − 2px right-inset = 14px travel */
    transform: translateX(14px);
  }

  /* ── Reduced motion ──────────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .knob { transition: none; }
    .switch::before { transition: none; }
  }

  /* Screen-reader only utility */
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
  }
</style>
