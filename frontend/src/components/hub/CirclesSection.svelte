<script>
  /**
   * Circles — every room you've joined, as quiet paper rows.
   * The Everyone tab answers "how is everyone doing"; this tab answers
   * "which circles am I in, who's in each, and is anything waiting on me".
   * Data is the myRooms payload verbatim (code/name/members/createdBy/
   * myRoomRole/pendingAdminRequests) plus myContacts for one-to-one people.
   */
  import { getUserColor } from '../../lib/getUserColor.js';

  let { rooms = [], contacts = [], members = [], meId = '', meName = '', onManage } = $props();

  // Presence by userId — familyMembers only carries people who share with you,
  // so a member missing here simply shows no dot (unknown ≠ offline).
  let presence = $derived(new Map(members.map((m) => [m.userId, m.online !== false])));

  const MAX_PEBBLES = 5;

  function initial(name) {
    return (name || '?').trim().charAt(0).toUpperCase() || '?';
  }

  function othersFirst(room) {
    // Self last so the row reads as "them, and you're in it too".
    const list = [...(room.members || [])];
    list.sort((a, b) => (a.userId === meId ? 1 : 0) - (b.userId === meId ? 1 : 0));
    return list;
  }

  function roomSentence(room) {
    const all = room.members || [];
    const n = all.length;
    const live = all.filter((m) => m.userId !== meId && presence.get(m.userId)).length;
    const people = n === 1 ? 'Just you here' : `${n} people`;
    const sharing = live > 0 ? ` · ${live} sharing now` : '';
    let role = '';
    if (room.createdBy === meId) role = ' · you started this circle';
    else if (room.myRoomRole === 'admin') role = ' · you keep this circle';
    return `${people}${sharing}${role}`;
  }
</script>

<div class="cs">
  {#if rooms.length === 0 && contacts.length === 0}
    <div class="cs-empty">
      <p class="cs-empty-verdict verdict-voice">You haven't joined a circle yet.</p>
      <p class="cs-empty-body">
        A circle is a small group — your family, a trip, a household — where everyone
        can see each other. Start one, or join with a code someone sent you.
      </p>
      <button class="cs-manage-filled" onclick={onManage}>Create or join a circle</button>
    </div>
  {:else}
    <ul class="cs-list" aria-label="Circles you belong to">
      {#each rooms as room (room.code)}
        <li class="cs-row">
          <div class="cs-row-top">
            <h3 class="cs-name">{room.name || 'Unnamed circle'}</h3>
            <div class="cs-pebbles" aria-hidden="true">
              {#each othersFirst(room).slice(0, MAX_PEBBLES) as m (m.userId)}
                <span
                  class="cs-pebble"
                  class:cs-pebble-me={m.userId === meId}
                  style={m.userId === meId ? '' : `background: color-mix(in oklch, ${getUserColor(m.userId)} 22%, var(--surface-1)); color: ${getUserColor(m.userId)};`}
                  title={m.displayName}
                >
                  {initial(m.userId === meId ? meName : m.displayName)}
                  {#if m.userId !== meId && presence.get(m.userId)}
                    <span class="cs-dot"></span>
                  {/if}
                </span>
              {/each}
              {#if (room.members || []).length > MAX_PEBBLES}
                <span class="cs-pebble cs-pebble-more">+{room.members.length - MAX_PEBBLES}</span>
              {/if}
            </div>
          </div>
          <p class="cs-sentence">{roomSentence(room)}</p>
          {#if (room.pendingAdminRequests || []).length > 0}
            <p class="cs-pending">
              {room.pendingAdminRequests.length === 1
                ? 'An admin request is waiting for your vote'
                : `${room.pendingAdminRequests.length} admin requests are waiting for your vote`}
              — it's in Connect.
            </p>
          {/if}
        </li>
      {/each}

      {#if contacts.length > 0}
        <li class="cs-row">
          <div class="cs-row-top">
            <h3 class="cs-name">One-to-one</h3>
            <div class="cs-pebbles" aria-hidden="true">
              {#each contacts.slice(0, MAX_PEBBLES) as c (c.userId)}
                <span
                  class="cs-pebble"
                  style={`background: color-mix(in oklch, ${getUserColor(c.userId)} 22%, var(--surface-1)); color: ${getUserColor(c.userId)};`}
                  title={c.displayName}
                >
                  {initial(c.displayName)}
                  {#if c.online}<span class="cs-dot"></span>{/if}
                </span>
              {/each}
              {#if contacts.length > MAX_PEBBLES}
                <span class="cs-pebble cs-pebble-more">+{contacts.length - MAX_PEBBLES}</span>
              {/if}
            </div>
          </div>
          <p class="cs-sentence">
            {contacts.length === 1
              ? 'One person you share with directly, outside any circle.'
              : `${contacts.length} people you share with directly, outside any circle.`}
          </p>
        </li>
      {/if}
    </ul>

    <button class="cs-manage" onclick={onManage}>Manage circles in Connect →</button>
  {/if}
</div>

<style>
  .cs { display: flex; flex-direction: column; gap: var(--space-2); }

  /* Quiet paper rows — whitespace separation, tonal hover, no card borders */
  .cs-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; }
  .cs-row {
    padding: var(--space-4) var(--space-3);
    border-radius: var(--radius-lg);
    transition: background var(--duration-fast) var(--ease-out);
  }
  .cs-row:hover { background: var(--surface-hover); }
  .cs-row + .cs-row { margin-top: var(--space-1); }

  .cs-row-top {
    display: flex; align-items: center; justify-content: space-between;
    gap: var(--space-3);
  }
  .cs-name {
    margin: 0;
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--text-primary);
    min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }

  /* Overlapping member pebbles, paper-ringed so the stack reads as stones */
  .cs-pebbles { display: flex; flex-shrink: 0; }
  .cs-pebble {
    position: relative;
    width: 32px; height: 32px;
    border-radius: var(--radius-full);
    border: 2px solid var(--paper);
    display: inline-flex; align-items: center; justify-content: center;
    font-size: var(--text-xs); font-weight: 600;
    background: var(--surface-inset); color: var(--text-secondary);
  }
  .cs-pebble + .cs-pebble { margin-left: calc(-1 * var(--space-2)); }
  .cs-pebble-me { background: var(--primary-100); color: var(--primary-700); }
  .cs-pebble-more { background: var(--surface-3); color: var(--text-tertiary); }
  .cs-dot {
    position: absolute; right: -1px; bottom: -1px;
    width: 9px; height: 9px; border-radius: var(--radius-full);
    background: var(--status-live);
    border: 2px solid var(--paper);
  }

  .cs-sentence {
    margin: var(--space-1) 0 0;
    font-size: var(--text-base);
    line-height: var(--leading-normal, 1.5);
    color: var(--text-secondary);
  }
  .cs-pending {
    margin: var(--space-1) 0 0;
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--warning-700);
  }

  /* Footer path into Connect — quiet text link, 44px floor */
  .cs-manage {
    align-self: flex-start;
    min-height: 44px;
    padding: 0 var(--space-3);
    margin-top: var(--space-1);
    background: none; border: none; cursor: pointer;
    font-family: var(--font-sans);
    font-size: var(--text-sm); font-weight: 600;
    color: var(--primary-700);
    border-radius: var(--radius-md);
  }
  .cs-manage:hover { background: var(--surface-hover); }
  .cs-manage:focus-visible { outline: 2px solid var(--primary-500); outline-offset: 2px; }

  /* Empty state — a verdict, one plain sentence, one real action */
  .cs-empty {
    padding: var(--space-6) var(--space-3);
    display: flex; flex-direction: column; align-items: flex-start; gap: var(--space-3);
  }
  .cs-empty-verdict {
    margin: 0;
    font-size: var(--text-2xl);
    color: var(--text-primary);
  }
  .cs-empty-body {
    margin: 0;
    font-size: var(--text-base); line-height: var(--leading-relaxed, 1.625);
    color: var(--text-secondary);
    max-width: 42ch;
  }
  .cs-manage-filled {
    min-height: 48px;
    padding: 0 var(--space-5);
    background: var(--primary-500); color: var(--text-on-primary);
    border: none; border-radius: var(--radius-md); cursor: pointer;
    font-family: var(--font-sans); font-size: var(--text-base); font-weight: 600;
    box-shadow: var(--shadow-sm);
    transition: background var(--duration-normal) var(--ease-out);
  }
  .cs-manage-filled:hover { background: var(--primary-600); }
  .cs-manage-filled:focus-visible { outline: 2px solid var(--primary-700); outline-offset: 2px; }
</style>
