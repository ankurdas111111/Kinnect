<script>
  /**
   * SecretChatInlineDecrypt — PIN entry panel shown inline under a locked message.
   * Extracted from SecretChatMessage to keep that component under 300 lines.
   *
   * Props:
   *   msgId      — message ID, used for accessible label/describedby IDs
   *   pin        — current PIN string (controlled by parent)
   *   error      — error string from a failed decrypt attempt
   *   unlocking  — true while decrypt is in-flight
   *
   * Events:
   *   pinInput(value)  — fires on every keystroke with the cleaned numeric string
   *   submit           — fires on Unlock button click or Enter key
   */
  import { createEventDispatcher } from 'svelte';
  import { fade } from 'svelte/transition';

  /**
   * @typedef {Object} Props
   * @property {string} [msgId]
   * @property {string} [pin]
   * @property {string} [error]
   * @property {boolean} [unlocking]
   */

  /** @type {Props} */
  let {
    msgId = '',
    pin = '',
    error = '',
    unlocking = false
  } = $props();

  const dispatch = createEventDispatcher();

  function handleInput(e) {
    dispatch('pinInput', e.target.value.replace(/\D/g, ''));
  }

  function handleKeydown(e) {
    if (e.key === 'Enter') dispatch('submit');
  }
</script>

<div class="sid-wrap" transition:fade={{ duration: 100 }}>
  <label class="sid-sr" for="inline-{msgId}">Sender's PIN to decrypt</label>
  <input
    id="inline-{msgId}"
    class="sid-pin"
    type="password"
    inputmode="numeric"
    pattern="[0-9]*"
    maxlength="8"
    placeholder="Sender's PIN"
    value={pin}
    oninput={handleInput}
    onkeydown={handleKeydown}
    disabled={unlocking}
    autocomplete="off"
    aria-describedby={error ? `inline-err-${msgId}` : undefined}
  />
  <button
    class="sid-btn"
    onclick={() => dispatch('submit')}
    disabled={unlocking || pin.length < 4}
    type="button"
    aria-label="Decrypt this message"
  >
    {unlocking ? '…' : 'Unlock'}
  </button>
  {#if error}
    <span class="sid-err" id="inline-err-{msgId}" role="alert">{error}</span>
  {/if}
</div>

<style>
  .sid-wrap {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-wrap: nowrap;
    margin-top: var(--space-1);
    padding: var(--space-2) var(--space-2-5);
    background: var(--primary-100);
    border: 1px solid color-mix(in oklch, var(--primary-500) 22%, transparent);
    border-radius: var(--radius-button);
    width: 100%;
    box-sizing: border-box;
  }

  .sid-sr {
    position: absolute; width: 1px; height: 1px;
    padding: 0; margin: -1px; overflow: hidden;
    clip: rect(0,0,0,0); white-space: nowrap; border: 0;
  }

  /* Warm inset PIN field — paper well, ember focus ring */
  .sid-pin {
    flex: 1;
    min-width: 0;
    padding: var(--space-2-5) var(--space-3);
    border-radius: var(--radius-input);
    border: 1px solid var(--border-subtle);
    background: var(--surface-1);
    color: var(--text-primary);
    /* 16px base — above iOS auto-zoom threshold; @media below pushes to 18px */
    font-size: var(--text-base);
    letter-spacing: 0.25em;
    text-align: center;
    outline: none;
    font-family: var(--font-sans);
    font-variant-numeric: tabular-nums;
    -webkit-appearance: none;
    min-height: 44px;
    transition: border-color var(--duration-fast) var(--ease-out), box-shadow var(--duration-fast) var(--ease-out);
    touch-action: manipulation;
  }

  @media (max-width: 767px) {
    .sid-pin { font-size: 18px; }
  }

  .sid-pin:focus {
    border-color: var(--primary-500);
    box-shadow: 0 0 0 2px color-mix(in oklch, var(--primary-500) 16%, transparent);
  }

  .sid-btn {
    padding: var(--space-2-5) var(--space-4);
    border-radius: var(--radius-input);
    border: none;
    background: var(--primary-500);
    color: var(--text-on-primary);
    font-size: var(--text-sm);
    font-weight: 600;
    font-family: var(--font-sans);
    cursor: pointer;
    min-height: 44px;
    flex-shrink: 0;
    white-space: nowrap;
    touch-action: manipulation;
    transition: background var(--duration-fast) var(--ease-out), transform var(--duration-fast) var(--ease-out);
  }

  .sid-btn:hover:not(:disabled) {
    background: var(--primary-600);
  }

  .sid-btn:active:not(:disabled) { transform: scale(0.98); }

  .sid-btn:disabled { opacity: 0.4; cursor: not-allowed; }

  .sid-btn:focus-visible {
    outline: 2px solid var(--primary-500);
    outline-offset: 2px;
  }

  /* Wrong PIN is a stumble, not an emergency — ochre, never vermilion. */
  .sid-err {
    font-size: var(--text-sm);
    font-family: var(--font-sans);
    color: var(--warning-700);
    width: 100%;
    font-weight: 500;
  }

  @media (prefers-reduced-motion: reduce) {
    .sid-btn:active { transform: none; }
  }
</style>
