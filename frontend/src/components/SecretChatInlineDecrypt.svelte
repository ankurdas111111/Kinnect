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

  export let msgId = '';
  export let pin = '';
  export let error = '';
  export let unlocking = false;

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
    on:input={handleInput}
    on:keydown={handleKeydown}
    disabled={unlocking}
    autocomplete="off"
    aria-describedby={error ? `inline-err-${msgId}` : undefined}
  />
  <button
    class="sid-btn"
    on:click={() => dispatch('submit')}
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
    gap: var(--space-2, 8px);
    flex-wrap: nowrap;
    margin-top: var(--space-1, 4px);
    padding: var(--space-2, 8px) var(--space-2-5, 10px);
    background: var(--chat-accent-subtle, rgba(20, 184, 166, 0.08));
    border: 1px solid var(--chat-border-accent, rgba(20, 184, 166, 0.22));
    border-radius: var(--radius-lg, 14px);
    width: 100%;
    box-sizing: border-box;
  }

  .sid-sr {
    position: absolute; width: 1px; height: 1px;
    padding: 0; margin: -1px; overflow: hidden;
    clip: rect(0,0,0,0); white-space: nowrap; border: 0;
  }

  .sid-pin {
    flex: 1;
    min-width: 0;
    padding: var(--space-2-5, 10px);
    border-radius: var(--radius-sm2, 8px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    background: rgba(0, 0, 0, 0.35);
    color: rgba(255, 255, 255, 0.92);
    /* 16px base — above iOS auto-zoom threshold; @media below pushes to 18px */
    font-size: var(--text-base, 1rem);
    letter-spacing: 0.25em;
    text-align: center;
    outline: none;
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    -webkit-appearance: none;
    min-height: 44px;
    transition: border-color 0.1s;
    touch-action: manipulation;
  }

  @media (max-width: 767px) {
    .sid-pin { font-size: 18px; }
  }

  .sid-pin:focus {
    border-color: var(--chat-accent, #14b8a6);
    box-shadow: 0 0 0 2px var(--chat-accent-subtle, rgba(20, 184, 166, 0.08));
  }

  .sid-btn {
    padding: var(--space-2-5, 10px) var(--space-4, 16px);
    border-radius: var(--radius-sm2, 8px);
    border: none;
    background: var(--chat-accent, #14b8a6);
    color: var(--text-on-primary);
    font-size: var(--text-xs, 0.75rem);
    font-weight: 700;
    font-family: var(--font-sans, 'Nunito', sans-serif);
    cursor: pointer;
    min-height: 44px;
    flex-shrink: 0;
    white-space: nowrap;
    touch-action: manipulation;
    transition: background 0.1s, transform 0.1s;
  }

  .sid-btn:hover:not(:disabled) {
    background: var(--primary-400, #2dd4bf);
    transform: scale(1.02);
  }

  .sid-btn:disabled { opacity: 0.3; cursor: not-allowed; }

  .sid-btn:focus-visible {
    outline: 2px solid var(--chat-accent, #14b8a6);
    outline-offset: 2px;
  }

  .sid-err {
    font-size: var(--text-2xs, 0.6875rem);
    font-family: var(--font-sans, 'Nunito', sans-serif);
    color: var(--danger-400, #f87171);
    width: 100%;
    font-weight: 500;
  }

  @media (prefers-reduced-motion: reduce) {
    .sid-btn:hover { transform: none; }
  }
</style>
