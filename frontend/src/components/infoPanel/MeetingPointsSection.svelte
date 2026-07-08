<script>
  import { myRooms } from '../../lib/stores/rooms.js';
  import { emitSetMeetingPoint, emitClearMeetingPoint } from '../../lib/socket.js';

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
    <span class="card-eyebrow">Meeting Points</span>
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
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-xl);
    padding: var(--space-3) var(--space-4);
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
    font-size: 9px;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    padding: 2px 6px;
    border-radius: var(--radius-full);
    background: rgba(16, 185, 129, 0.14);
    color: var(--success-500);
    border: 1px solid rgba(16, 185, 129, 0.25);
  }

  .meeting-point-info {
    font-size: 11px;
    color: var(--text-secondary);
    margin: 2px 0 4px;
    font-family: var(--font-mono);
  }

  .meeting-point-form {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 6px;
  }
</style>
