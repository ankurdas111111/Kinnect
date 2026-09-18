<script>
  import { myRooms } from '../../lib/stores/rooms.js';
  import { emitSetMeetingPoint, emitClearMeetingPoint } from '../../lib/socket.js';
  import SectionHeader from '../primitives/SectionHeader.svelte';

  // ── F3: Meeting point ──────────────────────────────────────────────────────
  let meetingRoomCode = $state(null);
  let meetingLat = $state('');
  let meetingLng = $state('');
  let meetingLabel = $state('');

  function openMeetingPointEditor(roomCode) {
    meetingRoomCode = meetingRoomCode === roomCode ? null : roomCode;
    meetingLat = '';
    meetingLng = '';
    meetingLabel = '';
  }

  function saveMeetingPoint() {
    const lat = parseFloat(meetingLat);
    const lng = parseFloat(meetingLng);
    if (!meetingRoomCode || isNaN(lat) || isNaN(lng)) return;
    if (lat < -90 || lat > 90 || lng < -180 || lng > 180) return;
    emitSetMeetingPoint(meetingRoomCode, lat, lng, meetingLabel);
    meetingRoomCode = null;
  }

  function clearMeetingPoint(roomCode) {
    emitClearMeetingPoint(roomCode);
  }
</script>

<!-- ── F3: MEETING POINT PER ROOM ────────────────────────────────── -->
{#if $myRooms.length > 0}
  <div class="feature-section">
    <SectionHeader title="Meeting Points" level={4} />
    {#each $myRooms as room}
      <div class="room-meeting-row">
        <div class="room-meeting-header">
          <span class="room-name">{room.name || room.code}</span>
          {#if room.meetingPoint}
            <span class="meeting-set-badge">Set</span>
            <button class="btn btn-ghost btn-xs" onclick={() => clearMeetingPoint(room.code)} aria-label="Clear meeting point for {room.name || room.code}">Clear</button>
          {/if}
          <button class="btn btn-secondary btn-xs" onclick={() => openMeetingPointEditor(room.code)} aria-label="Set meeting point for {room.name || room.code}">
            {meetingRoomCode === room.code ? 'Cancel' : 'Set'}
          </button>
        </div>
        {#if room.meetingPoint}
          <p class="meeting-point-info">{room.meetingPoint.label || 'Meeting point'} — {room.meetingPoint.lat.toFixed(4)}, {room.meetingPoint.lng.toFixed(4)}</p>
        {/if}
        {#if meetingRoomCode === room.code}
          <div class="meeting-point-form">
            <input class="feature-input" type="number" placeholder="Latitude" bind:value={meetingLat} step="0.0001" aria-label="Meeting point latitude" />
            <input class="feature-input" type="number" placeholder="Longitude" bind:value={meetingLng} step="0.0001" aria-label="Meeting point longitude" />
            <input class="feature-input" type="text" maxlength="80" placeholder="Label (optional)" bind:value={meetingLabel} aria-label="Meeting point label" />
            <button class="btn btn-primary btn-sm" onclick={saveMeetingPoint}>Save</button>
          </div>
        {/if}
      </div>
    {/each}
  </div>
{/if}

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

  /* F3: meeting point */
  .room-meeting-row {
    padding: 6px 0;
    border-bottom: 1px solid var(--border-subtle);
  }
  .room-meeting-row:last-child { border-bottom: none; }

  .room-meeting-header {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
    min-height: 44px;
  }

  .room-name {
    flex: 1;
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }

  .meeting-set-badge {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 2px 6px;
    border-radius: var(--radius-full);
    background: var(--success-500-08, color-mix(in oklch, var(--success-500) 8%, transparent));
    color: var(--success-700);
    border: 1px solid var(--success-500-20, color-mix(in oklch, var(--success-500) 20%, transparent));
  }

  .meeting-point-info {
    font-size: 11px;
    color: var(--text-secondary);
    margin: 2px 0 4px;
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
  }

  .meeting-point-form {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 6px;
  }
</style>
