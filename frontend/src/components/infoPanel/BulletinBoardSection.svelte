<script>
  import { authUser } from '../../lib/stores/auth.js';
  import { myRooms, roomNotes } from '../../lib/stores/rooms.js';
  import { emitPostRoomNote, emitDeleteRoomNote, emitGetRoomNotes } from '../../lib/socket.js';
  import SectionHeader from '../primitives/SectionHeader.svelte';

  // ── F8: Room bulletin board ────────────────────────────────────────────────
  let openBoardRoom = $state(null);
  let noteDraft = $state('');

  let myUserId = $derived($authUser?.userId);

  function toggleBoard(roomCode) {
    if (openBoardRoom === roomCode) {
      openBoardRoom = null;
    } else {
      openBoardRoom = roomCode;
      emitGetRoomNotes(roomCode);
    }
  }

  function postNote(roomCode) {
    const body = noteDraft.trim();
    if (!body || body.length > 200) return;
    emitPostRoomNote(roomCode, body);
    noteDraft = '';
  }

  function deleteNote(noteId, roomCode) {
    emitDeleteRoomNote(noteId, roomCode);
  }

  function formatNoteTs(ts) {
    if (!ts) return '';
    return new Date(ts).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
  }
</script>

<!-- ── F8: ROOM BULLETIN BOARD ───────────────────────────────────── -->
{#if $myRooms.length > 0}
  <div class="feature-section">
    <SectionHeader title="Bulletin Board" level={4} />
    {#each $myRooms as room}
      <div class="board-room">
        <button class="collapsible-header" onclick={() => toggleBoard(room.code)} aria-expanded={openBoardRoom === room.code}>
          <span class="room-name">{room.name || room.code}</span>
          <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true" style="transform: rotate({openBoardRoom === room.code ? 180 : 0}deg); transition: transform 200ms"><polyline points="6 9 12 15 18 9"/></svg>
        </button>
        {#if openBoardRoom === room.code}
          <div class="board-content">
            <div class="board-compose">
              <input
                class="feature-input"
                type="text"
                maxlength="200"
                placeholder="Post a note..."
                bind:value={noteDraft}
                aria-label="Write a note for {room.name || room.code}"
              />
              <button class="btn btn-primary btn-sm" onclick={() => postNote(room.code)} disabled={!noteDraft.trim()}>Post</button>
            </div>
            {#if !$roomNotes.has(room.code)}
              <!-- Skeleton while notes load over the socket -->
              <div class="skeleton-block" role="status" aria-label="Loading notes" aria-busy="true">
                <div class="skel-row skel-wide"></div>
                <div class="skel-row skel-mid"></div>
              </div>
            {:else}
              {#each ($roomNotes.get(room.code) || []) as note}
                <div class="board-note">
                  <div class="note-meta">
                    <span class="note-author">{note.authorName || 'Unknown'}</span>
                    <span class="note-time">{formatNoteTs(note.createdAt)}</span>
                    {#if note.authorId === myUserId}
                      <button class="btn-note-delete" onclick={() => deleteNote(note.id, room.code)} aria-label="Delete note">
                        <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                      </button>
                    {/if}
                  </div>
                  <p class="note-body">{note.body}</p>
                </div>
              {/each}
            {/if}
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

  .collapsible-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    width: 100%;
    min-height: 44px;
    color: var(--text-primary);
  }
  .collapsible-header:hover { opacity: 0.8; }

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

  /* F8: bulletin board */
  .board-room {
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    overflow: hidden;
  }

  .board-content {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 8px;
  }

  .board-compose {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }
  .board-compose .feature-input {
    min-width: 120px;
  }

  .board-note {
    padding: 8px 10px;
    background: var(--surface-inset);
    border-radius: var(--radius-md);
    border: 1px solid var(--border-subtle);
  }

  .note-meta {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 4px;
  }

  .note-author {
    font-size: 11px;
    font-weight: 600;
    color: var(--primary-700);
  }

  .note-time {
    font-size: 11px;
    color: var(--text-tertiary);
    flex: 1;
  }

  .btn-note-delete {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: var(--radius-sm);
    background: none;
    border: none;
    color: var(--text-tertiary);
    cursor: pointer;
    padding: 0;
    flex-shrink: 0;
    transition: color 120ms, background 120ms;
    -webkit-tap-highlight-color: transparent;
  }
  @media (max-width: 767px) {
    .btn-note-delete {
      width: 44px;
      height: 44px;
    }
  }
  /* Deleting a note is routine — ink hover, not red. */
  .btn-note-delete:hover {
    color: var(--text-primary);
    background: var(--surface-hover);
  }
  .btn-note-delete:focus-visible {
    outline: 2px solid var(--primary-400);
    outline-offset: 2px;
  }

  .note-body {
    font-size: var(--text-sm);
    color: var(--text-primary);
    margin: 0;
    line-height: 1.4;
    word-break: break-word;
  }

  /* ── Loading skeleton ─────────────────────────────────────────────────── */
  .skeleton-block {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-1) 0;
  }

  .skel-row {
    height: var(--space-4);
    border-radius: var(--radius-sm);
    background: color-mix(in oklch, var(--text-primary) 7%, transparent);
    animation: skel-pulse var(--skeleton-duration, 1.6s) ease-in-out infinite;
  }

  .skel-wide { width: 100%; }
  .skel-mid  { width: 70%; }

  @keyframes skel-pulse {
    0%, 100% { opacity: 0.5; }
    50%       { opacity: 1; }
  }

  @media (prefers-reduced-motion: reduce) {
    .skel-row { animation: none; opacity: 0.7; }
  }
</style>
