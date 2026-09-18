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
          <span class="cpd-vis-icon" aria-hidden="true">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          </span>
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
          <span class="cpd-vis-icon" aria-hidden="true">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>
          </span>
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
          <span class="cpd-vis-icon" aria-hidden="true">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          </span>
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
    background: color-mix(in oklch, var(--ink) 45%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4, 16px);
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
  }

  .cpd-card {
    background: var(--surface-1);
    border-radius: var(--radius-sheet, 24px);
    box-shadow: var(--shadow-xl);
    width: 340px;
    max-width: calc(100vw - 32px);
    max-height: calc(100dvh - var(--safe-bottom, 0px) - 32px);
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  .cpd-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 16px 0;
  }

  .cpd-title {
    font-size: var(--text-xl);
    font-weight: 600;
    color: var(--text-primary);
    font-family: var(--font-display);
    letter-spacing: -0.01em;
  }

  .cpd-close {
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-secondary);
    padding: 4px;
    border-radius: var(--radius-input);
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 44px;
    min-height: 44px;
    touch-action: manipulation;
    -webkit-tap-highlight-color: transparent;
  }
  .cpd-close:hover { background: var(--surface-hover); color: var(--text-primary); }
  .cpd-close:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }

  .cpd-body {
    padding: 12px 16px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .cpd-coords {
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
    font-size: 11px;
    color: var(--text-secondary);
    background: var(--surface-3);
    padding: 4px 8px;
    border-radius: var(--radius-sm);
    align-self: flex-start;
  }

  .cpd-label {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .cpd-input {
    width: 100%;
    padding: var(--space-2-5) var(--space-3);
    min-height: 44px;
    border-radius: var(--radius-input);
    border: 1px solid var(--border-subtle);
    background: var(--surface-3);
    color: var(--text-primary);
    /* 16px floor — prevents iOS zoom-on-focus */
    font-size: max(16px, var(--text-base));
    font-family: var(--font-sans);
    outline: none;
    box-sizing: border-box;
    transition: border-color 150ms var(--ease-out), box-shadow 150ms var(--ease-out);
  }
  .cpd-input::placeholder { color: var(--text-tertiary); }
  .cpd-input:focus {
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px var(--primary-500-12);
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
    min-height: 44px;
    border-radius: var(--radius-input);
    border: 1px solid transparent;
    background: var(--surface-2);
    cursor: pointer;
    transition: border-color 120ms var(--ease-out), background 120ms var(--ease-out);
  }
  .cpd-icon-btn:hover {
    background: var(--surface-3);
  }
  .cpd-icon-btn.selected {
    background: var(--primary-100, var(--primary-500-12));
    border-color: var(--primary-500);
  }
  .cpd-icon-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }

  .cpd-emoji { font-size: 20px; line-height: 1; }
  .cpd-icon-label {
    font-size: 11px;
    font-weight: 500;
    color: var(--text-secondary);
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
    min-height: 44px;
    border-radius: var(--radius-input);
    border: 1px solid var(--border-subtle);
    background: var(--surface-2);
    cursor: pointer;
    text-align: center;
    transition: border-color 120ms var(--ease-out), background 120ms var(--ease-out);
  }
  .cpd-vis-btn:hover:not(:disabled) {
    background: var(--surface-3);
  }
  .cpd-vis-btn.selected {
    background: var(--primary-100, var(--primary-500-12));
    border-color: var(--primary-500);
  }
  .cpd-vis-btn:focus-visible { outline: 2px solid var(--primary-400); outline-offset: 2px; }
  .cpd-vis-btn:disabled {
    cursor: not-allowed;
  }
  .cpd-vis-btn:disabled .cpd-vis-icon,
  .cpd-vis-btn:disabled strong,
  .cpd-vis-btn:disabled span {
    color: var(--text-tertiary);
  }

  .cpd-vis-icon {
    display: inline-flex;
    color: var(--text-secondary);
  }
  .cpd-vis-btn.selected .cpd-vis-icon { color: var(--primary-700); }

  .cpd-vis-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .cpd-vis-text strong {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-primary);
    line-height: 1.2;
  }
  .cpd-vis-text span {
    font-size: 11px;
    color: var(--text-secondary);
    line-height: 1.25;
  }

  /* Room selector */
  .cpd-room-select {
    width: 100%;
    padding: var(--space-2-5) var(--space-3);
    min-height: 44px;
    border-radius: var(--radius-input);
    border: 1px solid var(--border-subtle);
    background: var(--surface-3);
    color: var(--text-primary);
    font-size: max(16px, var(--text-base)); /* 16px minimum prevents iOS Safari auto-zoom on focus */
    font-family: var(--font-sans);
    outline: none;
    box-sizing: border-box;
    cursor: pointer;
    touch-action: manipulation;
  }
  .cpd-room-select:focus-visible {
    border-color: var(--primary-500);
    box-shadow: 0 0 0 3px var(--primary-500-12);
  }

  /* Validation speaks in ochre — a gentle correction, never an alarm. */
  .cpd-error {
    font-size: var(--text-sm);
    color: var(--warning-700);
    background: var(--warning-500-08, color-mix(in oklch, var(--warning-500) 8%, transparent));
    padding: 6px 10px;
    border-radius: var(--radius-sm2, 8px);
    border: 1px solid var(--warning-500-20, color-mix(in oklch, var(--warning-500) 20%, transparent));
  }

  .cpd-footer {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
    padding: 0 16px 14px;
  }

  @media (prefers-reduced-motion: reduce) {
    .cpd-icon-btn, .cpd-vis-btn, .cpd-input { transition: none; }
  }
</style>
