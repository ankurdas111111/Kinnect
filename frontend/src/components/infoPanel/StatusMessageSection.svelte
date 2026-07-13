<script>
  import { socket } from '../../lib/socket.js';
  import { banner } from '../../lib/stores/sos.js';
  import SectionHeader from '../primitives/SectionHeader.svelte';

  // Ambient status message
  let statusDraft = $state('');
  let statusExpiry = $state('60'); // minutes; '0' = no expiry

  function saveStatusMessage() {
    socket.emit('setStatusMessage', {
      message: statusDraft.trim(),
      expiryMinutes: statusExpiry === '0' ? 0 : parseInt(statusExpiry, 10),
    });
    banner.set({ type: 'info', text: statusDraft.trim() ? 'Status updated.' : 'Status cleared.', actions: [] });
    setTimeout(() => banner.set({ type: null, text: null, actions: [] }), 1500);
  }

  function clearStatusMessage() {
    statusDraft = '';
    socket.emit('setStatusMessage', { message: '', expiryMinutes: 0 });
  }
</script>

<!-- ── AMBIENT STATUS MESSAGE ─────────────────────────────────── -->
<div class="status-msg-zone">
  <SectionHeader title="Status Message" level={4} />
  <div class="status-msg-row">
    <input
      class="status-msg-input"
      type="text"
      maxlength="60"
      placeholder='e.g. "At school until 3pm"'
      bind:value={statusDraft}
      aria-label="Set a status message visible to your family"
    />
    <select class="status-expiry-select" bind:value={statusExpiry} aria-label="Status expires after">
      <option value="60">1h</option>
      <option value="240">4h</option>
      <option value="480">8h</option>
      <option value="1440">Today</option>
      <option value="0">Always</option>
    </select>
  </div>
  <div class="status-msg-actions">
    <button class="btn btn-primary btn-sm" onclick={saveStatusMessage} disabled={statusDraft.trim() === ''}>Set</button>
    <button class="btn btn-ghost btn-sm" onclick={clearStatusMessage}>Clear</button>
  </div>
</div>

<style>
  /* ── Ambient Status Message ──────────────────────────────────────── */
  .status-msg-zone {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    background: var(--surface-1);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-xl);
    padding: var(--space-3) var(--space-4);
  }
  .status-msg-row {
    display: flex;
    gap: 6px;
    align-items: center;
  }
  .status-msg-input {
    flex: 1;
    font-size: var(--text-sm);
    padding: 7px 10px;
    border-radius: var(--radius-md);
    border: 1px solid var(--border-subtle);
    background: var(--surface-3);
    color: var(--text-primary);
    min-width: 0;
  }
  .status-expiry-select {
    font-size: var(--text-xs);
    padding: 7px 6px;
    border-radius: var(--radius-md);
    border: 1px solid var(--border-subtle);
    background: var(--surface-3);
    color: var(--text-secondary);
    cursor: pointer;
  }
  .status-msg-actions {
    display: flex;
    gap: 6px;
  }
</style>
