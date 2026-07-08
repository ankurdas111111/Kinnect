<script>
  import { run } from 'svelte/legacy';

  import { fade, scale } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { PIN_ICONS } from '../lib/pinIcons.js';
  import { apiPost } from '../lib/api.js';
  import { savedPlaces } from '../lib/stores/savedPlaces.js';
  import { myRooms } from '../lib/stores/rooms.js';

  let { lat, lng, onClose } = $props();

  let name = $state('');
  let selectedIcon = $state('pin');
  let visibility = $state('personal'); // 'personal' | 'room' | 'universal'
  let selectedRoomCode = $state('');
  let saving = $state(false);
  let error = $state('');

  // Pre-select first room when user switches to 'room' visibility
  run(() => {
    if (visibility === 'room' && !selectedRoomCode && $myRooms.length > 0) {
      selectedRoomCode = $myRooms[0].code;
    }
  });

  function onBackdropClick(e) {
    if (e.target === e.currentTarget) onClose();
  }

  async function handleCreate() {
    error = '';
    const trimmed = name.trim();
    if (!trimmed) { error = 'Please enter a name.'; return; }
    if (visibility === 'room' && !selectedRoomCode) { error = 'Please select a room.'; return; }
    saving = true;
    try {
      const place = await apiPost('/api/places', {
        name: trimmed,
        icon: selectedIcon,
        latitude: lat,
        longitude: lng,
        radiusM: 0,
        visibility,
        roomCode: visibility === 'room' ? selectedRoomCode : '',
      });
      if (place?.id) {
        savedPlaces.update(m => { m.set(place.id, place); return m; });
        // Note: emitSyncPlace('add', place) used to be called here for shared
        // pins, but that export (and any syncPlace event) never existed — the
        // TypeError was caught below and shown as a false "Network error".
        // Shared pins reach other members via the places API on their next load.
        onClose();
      } else {
        error = place?.error || 'Failed to save pin.';
      }
    } catch {
      error = 'Network error.';
    } finally {
      saving = false;
    }
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="cpd-backdrop" onclick={onBackdropClick} transition:fade={{ duration: 150 }} role="presentation">
  <div
    class="cpd-card"
    role="dialog"
    aria-modal="true"
    aria-label="Add location pin"
    transition:scale={{ start: 0.95, duration: 200, easing: cubicOut }}
  >
    <div class="cpd-header">
      <span class="cpd-title">Add Pin</span>
      <button class="cpd-close" onclick={onClose} aria-label="Close">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>

    <div class="cpd-body">
      <div class="cpd-coords">
        {Number(lat).toFixed(5)}, {Number(lng).toFixed(5)}
      </div>

      <label class="cpd-label" for="pin-name">Name</label>
      <input
        id="pin-name"
        class="cpd-input"
        type="text"
        maxlength="100"
        placeholder="e.g. Home, School, Meeting point"
        bind:value={name}
        onkeydown={(e) => e.key === 'Enter' && !saving && handleCreate()}
        autofocus
      />

      <div class="cpd-label">Icon</div>
      <div class="cpd-icon-grid" role="radiogroup" aria-label="Pin icon">
        {#each PIN_ICONS as icon (icon.id)}
          <button
            class="cpd-icon-btn"
            class:selected={selectedIcon === icon.id}
            role="radio"
            aria-checked={selectedIcon === icon.id}
            aria-label={icon.label}
            title={icon.label}
            onclick={() => selectedIcon = icon.id}
          >
            <span class="cpd-emoji">{icon.emoji}</span>
            <span class="cpd-icon-label">{icon.label}</span>
          </button>
        {/each}
      </div>

      <div class="cpd-label">Who can see this?</div>
      <div class="cpd-vis-row" role="radiogroup" aria-label="Pin visibility">
        <button
          class="cpd-vis-btn"
          class:selected={visibility === 'personal'}
          role="radio"
          aria-checked={visibility === 'personal'}
          onclick={() => visibility = 'personal'}
        >
          <span class="cpd-vis-icon">🔒</span>
          <span class="cpd-vis-text">
            <strong>Personal</strong>
            <span>Only you</span>
          </span>
        </button>
        <button
          class="cpd-vis-btn"
          class:selected={visibility === 'room'}
          role="radio"
          aria-checked={visibility === 'room'}
          onclick={() => visibility = 'room'}
          disabled={$myRooms.length === 0}
          title={$myRooms.length === 0 ? 'You are not in any rooms' : undefined}
        >
          <span class="cpd-vis-icon">🏠</span>
          <span class="cpd-vis-text">
            <strong>Room</strong>
            <span>Specific room</span>
          </span>
        </button>
        <button
          class="cpd-vis-btn"
          class:selected={visibility === 'universal'}
          role="radio"
          aria-checked={visibility === 'universal'}
          onclick={() => visibility = 'universal'}
        >
          <span class="cpd-vis-icon">👨‍👩‍👧</span>
          <span class="cpd-vis-text">
            <strong>Family</strong>
            <span>Everyone</span>
          </span>
        </button>
      </div>

      {#if visibility === 'room' && $myRooms.length > 0}
        <select class="cpd-room-select" bind:value={selectedRoomCode} aria-label="Select room">
          {#each $myRooms as room (room.code)}
            <option value={room.code}>{room.name} ({room.code})</option>
          {/each}
        </select>
      {/if}

      {#if error}
        <div class="cpd-error">{error}</div>
      {/if}
    </div>

    <div class="cpd-footer">
      <button class="btn btn-secondary" onclick={onClose}>Cancel</button>
      <button class="btn btn-primary" onclick={handleCreate} disabled={saving}>
        {saving ? 'Saving…' : 'Add Pin'}
      </button>
    </div>
  </div>
</div>

<style>
  .cpd-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-modal, 5000);
    background: rgba(0, 0, 0, 0.45);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4, 16px);
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
  }

  .cpd-card {
    background: var(--surface-2, #fff);
    border: 1px solid var(--border-default, #e2e8f0);
    border-radius: var(--radius-xl, 20px);
    box-shadow: var(--shadow-xl, 0 20px 60px rgba(0,0,0,0.2));
    width: 340px;
    max-width: calc(100vw - 32px);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .cpd-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 16px 0;
  }

  .cpd-title {
    font-size: 16px;
    font-weight: 700;
    color: var(--text-primary, #0f172a);
    font-family: var(--font-display, 'Inter', sans-serif);
  }

  .cpd-close {
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-secondary, #64748b);
    padding: 4px;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 44px;
    min-height: 44px;
    touch-action: manipulation;
    -webkit-tap-highlight-color: transparent;
  }
  .cpd-close:hover { background: var(--surface-3, #f1f5f9); }

  .cpd-body {
    padding: 12px 16px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .cpd-coords {
    font-family: monospace;
    font-size: 10px;
    color: var(--text-secondary, #64748b);
    background: var(--surface-1, #f8fafc);
    padding: 4px 8px;
    border-radius: 6px;
    border: 1px solid var(--border-default, #e2e8f0);
    align-self: flex-start;
  }

  .cpd-label {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary, #64748b);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .cpd-input {
    width: 100%;
    padding: 8px 12px;
    border-radius: var(--radius-md, 10px);
    border: 1.5px solid var(--border-default, #e2e8f0);
    background: var(--surface-1, #f8fafc);
    color: var(--text-primary, #0f172a);
    font-size: 16px;
    font-family: var(--font-sans, 'Inter', sans-serif);
    outline: none;
    box-sizing: border-box;
    transition: border-color 0.15s;
  }
  .cpd-input:focus {
    border-color: var(--primary-400, #818cf8);
    background: var(--surface-2, #fff);
  }

  .cpd-icon-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 6px;
  }

  .cpd-icon-btn {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    padding: 8px 4px 6px;
    border-radius: 10px;
    border: 1.5px solid transparent;
    background: var(--surface-1, #f8fafc);
    cursor: pointer;
    transition: border-color 0.12s, background 0.12s;
  }
  .cpd-icon-btn:hover {
    background: var(--primary-50, #eef2ff);
    border-color: var(--primary-200, #c7d2fe);
  }
  .cpd-icon-btn.selected {
    background: var(--primary-50, #eef2ff);
    border-color: var(--primary-400, #818cf8);
  }

  .cpd-emoji { font-size: 20px; line-height: 1; }
  .cpd-icon-label {
    font-size: 9px;
    font-weight: 600;
    color: var(--text-secondary, #64748b);
    text-align: center;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100%;
  }

  /* Visibility toggle — 3 equal columns */
  .cpd-vis-row {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: 6px;
  }

  .cpd-vis-btn {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 10px 6px;
    border-radius: 12px;
    border: 1.5px solid var(--border-default, #e2e8f0);
    background: var(--surface-1, #f8fafc);
    cursor: pointer;
    text-align: center;
    transition: border-color 0.12s, background 0.12s;
  }
  .cpd-vis-btn:hover:not(:disabled) {
    background: var(--primary-50, #eef2ff);
    border-color: var(--primary-200, #c7d2fe);
  }
  .cpd-vis-btn.selected {
    background: var(--primary-50, #eef2ff);
    border-color: var(--primary-400, #818cf8);
  }
  .cpd-vis-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .cpd-vis-icon { font-size: 20px; line-height: 1; }

  .cpd-vis-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .cpd-vis-text strong {
    font-size: 11px;
    font-weight: 700;
    color: var(--text-primary, #0f172a);
    line-height: 1.2;
  }
  .cpd-vis-text span {
    font-size: 9px;
    color: var(--text-secondary, #64748b);
    line-height: 1.2;
  }

  /* Room selector */
  .cpd-room-select {
    width: 100%;
    padding: 8px 12px;
    border-radius: var(--radius-md, 10px);
    border: 1.5px solid var(--primary-400, #818cf8);
    background: var(--surface-1, #f8fafc);
    color: var(--text-primary, #0f172a);
    font-size: 16px; /* 16px minimum prevents iOS Safari auto-zoom on focus */
    font-family: var(--font-sans, 'Inter', sans-serif);
    outline: none;
    box-sizing: border-box;
    cursor: pointer;
    touch-action: manipulation;
  }

  .cpd-error {
    font-size: 12px;
    color: var(--danger-500, #ef4444);
    background: rgba(239, 68, 68, 0.08);
    padding: 6px 10px;
    border-radius: 8px;
    border: 1px solid rgba(239, 68, 68, 0.2);
  }

  .cpd-footer {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
    padding: 0 16px 14px;
  }
</style>
