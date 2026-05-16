<script>
  /**
   * Input — Animated floating-label input
   *
   * Props:
   *   label       — floating label text
   *   type        — input type
   *   value       — bind:value
   *   placeholder — shown when focused (not as label)
   *   hint        — helper text below input
   *   error       — error message (overrides hint, activates error state)
   *   success     — bool: show success state
   *   disabled    — bool
   *   prefix      — slot: leading icon / adornment
   *   suffix      — slot: trailing icon / adornment
   *   size        — 'sm' | 'md' | 'lg'
   *   id          — input id (auto-generated if omitted)
   */
  import { tweened } from 'svelte/motion';
  import { cubicOut } from 'svelte/easing';
  import { createEventDispatcher } from 'svelte';
  import { onMount } from 'svelte';

  export let label       = '';
  export let type        = 'text';
  export let value       = '';
  export let placeholder = '';
  export let hint        = '';
  export let error       = '';
  export let success     = false;
  export let disabled    = false;
  export let size        = 'md';
  export let id          = '';

  const dispatch = createEventDispatcher();

  // Auto-generate stable id
  let uid;
  onMount(() => {
    uid = id || ('input-' + Math.random().toString(36).slice(2, 8));
  });

  let focused = false;
  let inputEl;

  // Label float: 0 = resting, 1 = floated
  const labelY = tweened(0, { duration: 180, easing: cubicOut });
  const labelScale = tweened(1, { duration: 180, easing: cubicOut });

  $: floated = focused || (value !== '' && value !== null && value !== undefined);

  $: {
    labelY.set(floated ? -1 : 0);
    labelScale.set(floated ? 0.78 : 1);
  }

  $: stateClass = error ? 'state-error' : success ? 'state-success' : focused ? 'state-focused' : '';

  function onFocus(e) {
    focused = true;
    dispatch('focus', e);
  }
  function onBlur(e) {
    focused = false;
    dispatch('blur', e);
  }
  function onInput(e) {
    value = e.target.value;
    dispatch('input', e);
  }
  function onChange(e) {
    dispatch('change', e);
  }
</script>

<div class="field field-{size}" class:disabled class:has-prefix={$$slots.prefix} class:has-suffix={$$slots.suffix}>
  <div class="field-inner {stateClass}">
    <!-- Prefix slot -->
    {#if $$slots.prefix}
      <span class="field-adornment field-prefix" aria-hidden="true">
        <slot name="prefix" />
      </span>
    {/if}

    <!-- Input -->
    <input
      bind:this={inputEl}
      {type}
      id={uid}
      {value}
      {disabled}
      placeholder={focused ? placeholder : ''}
      class="field-input"
      aria-label={label || undefined}
      aria-describedby={hint || error ? (uid + '-desc') : undefined}
      aria-invalid={error ? 'true' : undefined}
      on:focus={onFocus}
      on:blur={onBlur}
      on:input={onInput}
      on:change={onChange}
    />

    <!-- Floating label -->
    {#if label}
      <label
        for={uid}
        class="field-label"
        class:floated
        style="
          transform: translateY(calc({$labelY} * 28px)) scale({$labelScale});
          transform-origin: left center;
        "
      >
        {label}
      </label>
    {/if}

    <!-- Suffix slot -->
    {#if $$slots.suffix}
      <span class="field-adornment field-suffix" aria-hidden="true">
        <slot name="suffix" />
      </span>
    {/if}

    <!-- Focus ring (animated underline / border) -->
    <div class="field-focus-ring" aria-hidden="true"></div>
  </div>

  <!-- Helper / Error text -->
  {#if error || hint}
    <p
      id={uid ? (uid + '-desc') : undefined}
      class="field-hint"
      class:field-error={!!error}
    >
      {#if error}
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
          <circle cx="6" cy="6" r="5.5" stroke="currentColor"/>
          <path d="M6 3.5V6.5M6 8H6.01" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      {/if}
      {error || hint}
    </p>
  {/if}
</div>

<style>
  /* ── Field wrapper ────────────────────────────────────────────────────── */
  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1-5);
    width: 100%;
  }

  .field.disabled { opacity: 0.5; pointer-events: none; }

  /* ── Inner container ──────────────────────────────────────────────────── */
  .field-inner {
    position: relative;
    display: flex;
    align-items: center;
    background: var(--surface-3, rgba(15, 23, 42, 0.06));
    border: 1.5px solid var(--border-default);
    border-radius: var(--radius-md, 8px);
    transition:
      border-color  200ms var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1)),
      box-shadow    200ms var(--ease-out),
      background    200ms var(--ease-out);
    box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.08);
  }

  /* ── States ───────────────────────────────────────────────────────────── */
  .field-inner.state-focused {
    border-color: var(--primary-400);
    background: var(--surface-3, rgba(20, 184, 166, 0.04));
    box-shadow:
      0 0 0 3px rgba(20, 184, 166, 0.18),
      0 0 0 6px rgba(20, 184, 166, 0.06),
      inset 0 2px 4px rgba(0, 0, 0, 0.06);
  }

  .field-inner.state-error {
    border-color: var(--danger-500);
    box-shadow:
      0 0 0 3px rgba(239, 68, 68, 0.14),
      inset 0 2px 4px rgba(239, 68, 68, 0.06);
    animation: field-shake 0.38s var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1));
  }

  .field-inner.state-success {
    border-color: var(--success-500);
    box-shadow:
      0 0 0 3px rgba(16, 185, 129, 0.14),
      inset 0 2px 4px rgba(16, 185, 129, 0.04);
  }

  @keyframes field-shake {
    0%, 100% { transform: translateX(0); }
    20%       { transform: translateX(-5px); }
    40%       { transform: translateX(5px); }
    60%       { transform: translateX(-3px); }
    80%       { transform: translateX(3px); }
  }

  /* ── Sizes ────────────────────────────────────────────────────────────── */
  .field-sm .field-inner { min-height: 36px; }
  .field-md .field-inner { min-height: 44px; }
  .field-lg .field-inner { min-height: 52px; }

  @media (max-width: 767px) {
    .field-md .field-inner { min-height: 48px; }
    .field-sm .field-inner { min-height: 40px; }
  }

  /* ── Input ────────────────────────────────────────────────────────────── */
  .field-input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    color: var(--text-primary);
    font-family: var(--font-sans);
    font-size: var(--text-base);
    line-height: var(--leading-normal);
    caret-color: var(--primary-400);
    padding: var(--space-3) var(--space-4);
    /* Extra top padding when label is present so text sits below floated label */
    padding-top: var(--space-5);
  }

  .field:not([class*="has-"]) .field-input {
    padding-top: var(--space-3);
  }

  .field-input::placeholder {
    color: var(--text-tertiary);
    opacity: 0;
    transition: opacity 150ms;
  }

  .field-input:focus::placeholder {
    opacity: 1;
  }

  /* Mobile font-size ≥ 16px prevents iOS auto-zoom */
  @media (max-width: 767px) {
    .field-input { font-size: 16px; }
  }

  .field-sm .field-input { font-size: var(--text-sm); padding: var(--space-2) var(--space-3); }
  .field-lg .field-input { font-size: var(--text-lg); padding: var(--space-4) var(--space-5); }

  /* ── Floating label ───────────────────────────────────────────────────── */
  .field-label {
    position: absolute;
    left: var(--space-4);
    top: 50%;
    margin-top: -0.6em; /* vertical center of 1-line label */
    font-family: var(--font-sans);
    font-size: var(--text-base);
    font-weight: 500;
    color: var(--text-tertiary);
    pointer-events: none;
    will-change: transform;
    transition:
      color 180ms var(--ease-out);
    white-space: nowrap;
  }

  .field-label.floated {
    color: var(--primary-400);
    font-weight: 600;
  }

  .state-error .field-label.floated   { color: var(--danger-400); }
  .state-success .field-label.floated { color: var(--success-500); }

  /* ── Adornments ───────────────────────────────────────────────────────── */
  .field-adornment {
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-tertiary);
    flex-shrink: 0;
    transition: color 180ms;
  }

  .state-focused .field-adornment   { color: var(--primary-400); }
  .state-error .field-adornment     { color: var(--danger-400); }
  .state-success .field-adornment   { color: var(--success-500); }

  .field-prefix {
    padding-left: var(--space-3);
    padding-right: var(--space-1);
  }
  .field-suffix {
    padding-right: var(--space-3);
    padding-left:  var(--space-1);
  }

  /* Shift input padding when adornments exist */
  .has-prefix .field-input  { padding-left: var(--space-1); }
  .has-suffix .field-input  { padding-right: var(--space-1); }

  /* ── Focus ring (animated bottom line for glass style) ────────────────── */
  .field-focus-ring {
    position: absolute;
    bottom: -1.5px;
    left: 20%;
    right: 20%;
    height: 2px;
    background: linear-gradient(90deg,
      transparent,
      var(--primary-400) 40%,
      var(--primary-400) 60%,
      transparent
    );
    border-radius: 0 0 4px 4px;
    transform: scaleX(0);
    transform-origin: center;
    transition: transform 220ms var(--ease-out, cubic-bezier(0.16, 1, 0.3, 1));
    pointer-events: none;
  }

  .state-focused .field-focus-ring { transform: scaleX(1); }
  .state-error   .field-focus-ring {
    background: linear-gradient(90deg, transparent, var(--danger-500), transparent);
    transform: scaleX(1);
  }
  .state-success .field-focus-ring {
    background: linear-gradient(90deg, transparent, var(--success-500), transparent);
    transform: scaleX(1);
  }

  /* ── Helper / Error text ──────────────────────────────────────────────── */
  .field-hint {
    font-size: var(--text-xs);
    color: var(--text-tertiary);
    display: flex;
    align-items: center;
    gap: var(--space-1);
    padding: 0 var(--space-1);
    line-height: var(--leading-normal);
    animation: field-hint-appear 180ms var(--ease-out) both;
  }

  .field-hint.field-error {
    color: var(--danger-400);
  }

  .field-hint svg {
    flex-shrink: 0;
  }

  @keyframes field-hint-appear {
    from { opacity: 0; transform: translateY(-4px); }
    to   { opacity: 1; transform: translateY(0); }
  }

  /* ── Reduced motion ───────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .field-inner,
    .field-label,
    .field-focus-ring,
    .field-adornment,
    .field-hint { transition: none; animation: none; }
    .field-label { will-change: auto; }
    .field-inner.state-error { animation: none; }
  }
</style>
