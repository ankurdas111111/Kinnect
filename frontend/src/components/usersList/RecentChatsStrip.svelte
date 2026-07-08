<script>
  import { createEventDispatcher } from 'svelte';
  import { secretChats } from '../../lib/stores/secretChat.js';
  import { getAvatarStyle } from './avatarPalette.js';

  
  /**
   * @typedef {Object} Props
   * @property {any} [userList] - Sorted list of visible users (from UsersList).
   */

  /** @type {Props} */
  let { userList = [] } = $props();

  const dispatch = createEventDispatcher();

  // ── Recent chats strip ───────────────────────────────────────────────────
  let _userById = $derived(new Map(
    userList.filter(u => u.userId).map(u => [u.userId, u])
  ));
  let recentChatPeers = $derived((() => {
    const out = [];
    for (const [peerId, chat] of $secretChats) {
      if (!chat.messages || !chat.messages.length) continue;
      const u = _userById.get(peerId);
      if (!u) continue;
      out.push({
        id: peerId,
        name: u.displayName || '?',
        user: u,
        latestAt: chat.messages[0]?.createdAt || 0,
        hasUnread: chat.messages.some(m => !m.seenAt && m.senderId === peerId),
      });
    }
    return out.sort((a, b) => new Date(b.latestAt) - new Date(a.latestAt)).slice(0, 6);
  })());
</script>

<!-- Recent chats strip — quick-access avatars for ongoing conversations -->
{#if recentChatPeers.length > 0}
  <div class="recent-chats-strip" aria-label="Recent chats">
    <span class="recent-label">Chats</span>
    <div class="recent-scroll" role="list">
      {#each recentChatPeers as peer (peer.id)}
        <button
          class="recent-avatar-btn"
          onclick={() => dispatch('secretChat', { id: peer.id, name: peer.name })}
          aria-label="Chat with {peer.name}{peer.hasUnread ? ', unread messages' : ''}"
          role="listitem"
        >
          <div class="recent-avatar" style="{getAvatarStyle(peer.name)}">
            {(peer.name || '?')[0].toUpperCase()}
            {#if peer.hasUnread}
              <span class="recent-unread" aria-hidden="true"></span>
            {/if}
          </div>
          <span class="recent-name">{peer.name.split(' ')[0]}</span>
        </button>
      {/each}
    </div>
  </div>
{/if}

<style>
  /* ── Recent chats strip ─────────────────────────────────────────────────── */
  .recent-chats-strip {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2) var(--space-4) var(--space-2);
    border-bottom: 1px solid var(--border-subtle);
  }

  .recent-label {
    font-family: var(--font-display);
    font-size: var(--text-2xs);
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.07em;
    flex-shrink: 0;
  }

  .recent-scroll {
    display: flex;
    gap: var(--space-3);
    overflow-x: auto;
    overflow-y: visible;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
    flex: 1;
    padding: 4px 0;
  }

  .recent-scroll::-webkit-scrollbar { display: none; }

  .recent-avatar-btn {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;
    min-width: 44px;
    -webkit-tap-highlight-color: transparent;
    flex-shrink: 0;
  }

  .recent-avatar-btn:active { transform: scale(0.88); transition: transform 80ms; }

  .recent-avatar {
    position: relative;
    width: 38px;
    height: 38px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 15px;
    font-weight: 700;
    font-family: var(--font-display);
    border: 2px solid rgba(255,255,255,0.08);
    transition: border-color 120ms;
  }

  :global([data-theme="light"]) .recent-avatar {
    border-color: rgba(0,0,0,0.06);
  }

  .recent-avatar-btn:hover .recent-avatar {
    border-color: rgba(20,184,166,0.4);
  }

  .recent-unread {
    position: absolute;
    top: -1px;
    right: -1px;
    width: 9px;
    height: 9px;
    background: var(--danger-500);
    border-radius: 50%;
    border: 2px solid var(--surface-base);
    box-shadow: 0 0 4px rgba(239,68,68,0.5);
  }

  .recent-name {
    font-size: 10px;
    font-weight: 600;
    color: var(--text-secondary);
    max-width: 44px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: center;
    line-height: 1;
  }
</style>
