<script>
  import { alertState, sosNarratives, activeSosUsers } from '../lib/stores/sos.js';
  import Modal from './primitives/Modal.svelte';
  import { haptics } from '../lib/haptics.js';

  // Derive narrative from first active SOS user
  $: activeNarrative = (() => {
    for (const [userId, sos] of $activeSosUsers) {
      const n = $sosNarratives.get(userId);
      if (n) return n.narrative;
      // Also check if narrative is embedded in sos.sos
      if (sos.sos && sos.sos.narrative) return sos.sos.narrative;
    }
    return null;
  })();

  let audioCtx = null;
  let oscillator = null;
  let shaking = false;

  $: if ($alertState.visible && $alertState.alarmMs > 0) {
    startAlarm();
    triggerHaptic();
    triggerShake();
  } else {
    stopAlarm();
  }

  function triggerHaptic() {
    haptics.sos();
  }

  function triggerShake() {
    shaking = true;
    setTimeout(() => { shaking = false; }, 500);
  }

  function startAlarm() {
    const prefersReduced = typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (prefersReduced) return;

    try {
      audioCtx = new (window.AudioContext || window.webkitAudioContext)();
      oscillator = audioCtx.createOscillator();
      oscillator.type = 'sawtooth';
      oscillator.frequency.setValueAtTime(880, audioCtx.currentTime);
      oscillator.connect(audioCtx.destination);
      oscillator.start();
      setTimeout(() => stopAlarm(), $alertState.alarmMs);
    } catch (_) {}
  }

  function stopAlarm() {
    try {
      if (oscillator) { oscillator.stop(); oscillator = null; }
      if (audioCtx) { audioCtx.close(); audioCtx = null; }
    } catch (_) {}
  }

  function dismiss() {
    stopAlarm();
    alertState.set({ visible: false, title: '', body: '', actions: [], alarmMs: 0 });
  }
</script>

<!-- Camera shake wrapper -->
<div class="alert-shake-wrapper" class:shaking>
  <Modal open={$alertState.visible} urgent={true} title={$alertState.title || 'Alert'} on:close={dismiss}>
    <div class="alert-body-wrap">
      <div class="alert-sos-icon" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
          <line x1="12" y1="9" x2="12" y2="13"/>
          <line x1="12" y1="17" x2="12.01" y2="17"/>
        </svg>
      </div>
      <p class="alert-body">{$alertState.body}</p>
      {#if activeNarrative}
        <div class="narrative-chips">
          {#if activeNarrative.motionSummary}
            <span class="narrative-chip motion">{activeNarrative.motionSummary}</span>
          {/if}
          {#if activeNarrative.batteryPct != null}
            <span class="narrative-chip battery">Battery {activeNarrative.batteryPct}%</span>
          {/if}
          {#if activeNarrative.triggerRule && activeNarrative.triggerRule !== 'manual'}
            <span class="narrative-chip trigger">Auto: {activeNarrative.triggerRule}</span>
          {/if}
        </div>
      {/if}
    </div>

    <svelte:fragment slot="footer">
      {#each $alertState.actions as action}
        <button class="btn {action.kind || 'btn-primary'} btn-lg" on:click={() => { if (action.onClick) action.onClick(); dismiss(); }}>{action.label}</button>
      {/each}
      <button class="btn btn-secondary btn-lg" on:click={dismiss}>Dismiss</button>
    </svelte:fragment>
  </Modal>
</div>

<style>
  .alert-shake-wrapper {
    position: contents;
  }

  .alert-shake-wrapper.shaking :global(.modal-backdrop) {
    animation: camera-shake 0.45s var(--ease-out);
  }

  .alert-body-wrap {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    text-align: center;
    padding-top: var(--space-2);
  }

  .alert-sos-icon {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: rgba(239, 68, 68, 0.12);
    border: 2px solid rgba(239, 68, 68, 0.30);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--danger-500);
    animation: sos-urgent-pulse 1.2s ease infinite;
    flex-shrink: 0;
  }

  .alert-body {
    color: var(--text-secondary);
    font-size: var(--text-base);
    text-align: center;
    line-height: var(--leading-relaxed);
    margin: 0;
  }

  .narrative-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    justify-content: center;
    margin-top: 4px;
  }

  .narrative-chip {
    display: inline-block;
    font-size: 11px;
    font-weight: 600;
    padding: 3px 9px;
    border-radius: var(--radius-full);
    letter-spacing: 0.02em;
  }

  .narrative-chip.motion {
    background: rgba(245, 158, 11, 0.15);
    color: #f59e0b;
    border: 1px solid rgba(245, 158, 11, 0.25);
  }

  .narrative-chip.battery {
    background: rgba(16, 185, 129, 0.12);
    color: var(--success-500);
    border: 1px solid rgba(16, 185, 129, 0.20);
  }

  .narrative-chip.trigger {
    background: rgba(239, 68, 68, 0.12);
    color: var(--danger-500);
    border: 1px solid rgba(239, 68, 68, 0.20);
  }
</style>
