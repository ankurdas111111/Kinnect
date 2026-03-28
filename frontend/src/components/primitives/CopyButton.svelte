<script>
  export let text = '';
  export let label = 'Copy';
  let copied = false;

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
    } catch (_) {}
    copied = true;
    setTimeout(() => copied = false, 2000);
  }
</script>

<button class="copy-btn" class:copied on:click={copy} aria-label={copied ? 'Copied!' : label}>
  {#if copied}
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor"
         stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="M2 7l4 4 6-7" stroke-dasharray="20" stroke-dashoffset="20"
            style="animation: check-draw 250ms var(--ease-out) forwards"/>
    </svg>
    Copied!
  {:else}
    {label}
  {/if}
</button>

<style>
.copy-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: var(--space-1-5) var(--space-3);
  font-family: var(--font-sans);
  font-size: var(--text-xs);
  font-weight: 600;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-1);
  color: var(--text-secondary);
  cursor: pointer;
  white-space: nowrap;
  transition: background 150ms var(--ease-out), color 150ms var(--ease-out), border-color 150ms var(--ease-out);
  user-select: none;
}
.copy-btn:hover {
  background: var(--surface-2);
  color: var(--text-primary);
  border-color: var(--border-strong);
}
.copy-btn.copied {
  background: var(--success-500);
  color: white;
  border-color: transparent;
}
.copy-btn:active {
  transform: scale(0.95);
}
</style>
