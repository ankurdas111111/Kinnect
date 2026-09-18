<script>
  import { emitSetSpeedAlert } from '../../lib/socket.js';
  import SectionHeader from '../primitives/SectionHeader.svelte';

  // ── F5: Speed alert ────────────────────────────────────────────────────────
  let speedAlertKmh = $state('');

  function saveSpeedAlert() {
    const val = parseFloat(speedAlertKmh);
    if (isNaN(val) || val < 0) return;
    emitSetSpeedAlert(val);
  }

  function clearSpeedAlert() {
    speedAlertKmh = '';
    emitSetSpeedAlert(0);
  }
</script>

<!-- ── F5: SPEED ALERT CONFIG ─────────────────────────────────────── -->
<div class="feature-section">
  <SectionHeader title="Speed Alert" level={4} />
  <div class="feature-row">
    <input
      class="feature-input"
      type="number"
      min="0"
      max="300"
      placeholder="km/h (0 = off)"
      bind:value={speedAlertKmh}
      aria-label="Speed alert threshold in km/h"
    />
    <button class="btn btn-primary btn-sm" onclick={saveSpeedAlert} disabled={speedAlertKmh === ''}>Set</button>
    <button class="btn btn-ghost btn-sm" onclick={clearSpeedAlert}>Off</button>
  </div>
</div>

<style>
  .feature-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    background: var(--surface-1);
    border-radius: var(--radius-card, 20px);
    box-shadow: var(--shadow-xs);
    padding: var(--space-3) var(--space-4);
  }

  .feature-row {
    display: flex;
    gap: 6px;
    align-items: center;
    flex-wrap: wrap;
  }

  .feature-input {
    flex: 1;
    min-width: 80px;
    /* 16px floor — prevents iOS zoom-on-focus */
    font-size: max(16px, var(--text-base));
    padding: var(--space-2) var(--space-3);
    min-height: 44px;
    border-radius: var(--radius-input);
    border: 1px solid var(--border-subtle);
    background: var(--surface-3);
    color: var(--text-primary);
    font-family: var(--font-sans);
    transition: border-color 150ms var(--ease-out), box-shadow 150ms var(--ease-out);
  }
  .feature-input::placeholder { color: var(--text-tertiary); }
  .feature-input:focus {
    outline: none;
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px var(--primary-500-12);
  }
</style>
