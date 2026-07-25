<script>
  /**
   * PulseButton — one-tap "I'm safe" heartbeat for the Hub.
   *
   * The calm middle ground between silent tracking and a full SOS: a single tap
   * broadcasts reassurance to everyone who can see you (existing emitIAmSafe()),
   * and your own row remembers "Checked in 6:42pm" for the rest of the day.
   *
   * DB load: ZERO — emitIAmSafe() is a WS broadcast; the "checked in" memory is
   * localStorage on this device.
   */
  import { onMount } from 'svelte';
  import { emitIAmSafe } from '../../lib/socket.js';
  import { haptics } from '../../lib/haptics.js';
  import { allowMotion } from '../../lib/stores/effects.js';

  const KEY = 'kinnect_pulse_self';
  let checkedAt = $state(null); // ms of today's check-in, or null

  function isToday(ms) {
    if (!ms) return false;
    const a = new Date(ms), b = new Date();
    return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
  }

  onMount(() => {
    try {
      const raw = Number(localStorage.getItem(KEY));
      if (isToday(raw)) checkedAt = raw;
    } catch { /* private mode */ }
  });

  function pulse() {
    emitIAmSafe();
    haptics.success();
    const now = Date.now();
    checkedAt = now;
    try { localStorage.setItem(KEY, String(now)); } catch { /* ignore */ }
  }

  let timeStr = $derived(checkedAt ? new Date(checkedAt).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' }) : '');
</script>

<button class="pulse" class:done={checkedAt} class:calm={!$allowMotion} onclick={pulse}
  aria-label={checkedAt ? `Checked in at ${timeStr}. Tap to reassure your family again` : 'Tell your family you are safe'}>
  <span class="pulse-glyph" aria-hidden="true">
    {#if checkedAt}
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>
    {:else}
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
    {/if}
  </span>
  <span class="pulse-label">
    {#if checkedAt}Checked in · {timeStr}{:else}I'm safe{/if}
  </span>
</button>

<style>
  .pulse {
    width: 100%; min-height: 48px;
    display: flex; align-items: center; justify-content: center; gap: var(--space-2);
    padding: 0 var(--space-4);
    border-radius: var(--radius-full, 999px);
    border: 1px solid var(--primary-500-30);
    background: var(--primary-500);
    color: var(--text-on-primary); font-family: var(--font-display, system-ui);
    font-size: var(--text-sm, 14px); font-weight: 700; cursor: pointer;
    transition: transform var(--duration-fast, 120ms) var(--ease-out), background var(--duration-standard, 200ms) var(--ease-out), border-color var(--duration-standard, 200ms) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .pulse:hover { background: var(--primary-400); }
  .pulse:active { transform: scale(0.97); }
  .pulse:focus-visible { outline: 2px solid var(--primary-300); outline-offset: 2px; }

  /* Checked-in: settle to a calm success surface */
  .pulse.done {
    background: var(--success-500-12);
    border-color: var(--success-500-30);
    color: var(--success-300);
  }
  .pulse.done:hover { background: var(--success-500-20); }

  .pulse-glyph { display: flex; }
  /* Gentle heartbeat on the un-tapped call to action — GPU-only, calm-gated */
  .pulse:not(.done):not(.calm) .pulse-glyph { animation: beat 2.4s ease-in-out infinite; }
  @keyframes beat { 0%,100% { transform: scale(1); } 50% { transform: scale(1.14); } }

  @media (prefers-reduced-motion: reduce) {
    .pulse-glyph { animation: none !important; }
    .pulse:active { transform: none; }
  }
</style>
