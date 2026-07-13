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
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-xl);
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
    font-size: var(--text-sm);
    padding: 7px 10px;
    border-radius: var(--radius-md);
    border: 1px solid var(--border-subtle);
    background: var(--surface-3);
    color: var(--text-primary);
  }
  @media (max-width: 767px) {
    .feature-input {
      min-height: 44px;
      font-size: var(--text-base);
    }
  }
</style>
