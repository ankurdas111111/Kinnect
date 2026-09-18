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

  /** @type {{ quiet?: boolean }} quiet — secondary emphasis (zero-member dashboard state) */
  let { quiet = false } = $props();

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

<button class="pulse" class:done={checkedAt} class:quiet class:calm={!$allowMotion} onclick={pulse}
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
  /* Hearth / Family Circle — the reference's primary ember button: warm fill,
     paper-white label, approachable medium radius (never a pill, never a card). */
  .pulse {
    width: 100%; min-height: 48px;
    display: flex; align-items: center; justify-content: center; gap: var(--space-2);
    padding: 0 var(--space-5);
    border-radius: var(--radius-md);
    border: 1px solid transparent;
    background: var(--primary-500);
    color: var(--text-on-primary); font-family: var(--font-sans);
    font-size: var(--text-base); font-weight: 600; cursor: pointer;
    box-shadow: var(--shadow-xs);
    transition: transform var(--duration-fast) var(--ease-out), background var(--duration-normal) var(--ease-out), border-color var(--duration-normal) var(--ease-out), color var(--duration-normal) var(--ease-out);
    -webkit-tap-highlight-color: transparent;
  }
  .pulse:hover { background: var(--primary-600); }
  .pulse:active { transform: scale(0.97); }
  .pulse:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }

  /* Quiet secondary — the reference's paper-outline button. Used only when
     Invite takes the filled-ember emphasis instead (zero-member state). */
  .pulse.quiet:not(.done) {
    background: transparent;
    border-color: var(--outline-variant);
    color: var(--primary-700);
    box-shadow: none;
  }
  .pulse.quiet:not(.done):hover { background: var(--surface-hover); }

  /* Checked-in: settle onto a calm sage tint — arrival/settled language */
  .pulse.done {
    background: color-mix(in oklch, var(--success-500) 12%, transparent);
    border-color: color-mix(in oklch, var(--success-500) 30%, transparent);
    color: var(--success-600);
    box-shadow: none;
  }
  .pulse.done:hover { background: color-mix(in oklch, var(--success-500) 18%, transparent); }

  .pulse-glyph { display: flex; }
  /* Gentle heartbeat on the un-tapped call to action — GPU-only, calm-gated */
  .pulse:not(.done):not(.calm) .pulse-glyph { animation: beat 2.4s ease-in-out infinite; }
  @keyframes beat { 0%,100% { transform: scale(1); } 50% { transform: scale(1.14); } }

  @media (prefers-reduced-motion: reduce) {
    .pulse-glyph { animation: none !important; }
    .pulse:active { transform: none; }
  }
</style>
