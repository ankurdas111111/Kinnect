import { writable } from 'svelte/store';

// Map<peerId, { messages: Array, locked: boolean, decryptedMessages: Map<msgId, string> }>
export const secretChats = writable(new Map());

// Map<userId, { open: boolean, at: number (unix ms) }>
// Tracks whether a peer currently has our secret chat open
export const secretChatPresence = writable(new Map());

// Which peer's chat window is currently open (userId string or null)
export const activeSecretChatPeer = writable(null);

/** Mark a conversation as locked — clears decrypted text from memory. */
export function lockSecretChat(peerId) {
  secretChats.update((m) => {
    const copy = new Map(m);
    const chat = copy.get(peerId) ?? { messages: [], locked: true, decryptedMessages: new Map() };
    copy.set(peerId, { ...chat, locked: true, decryptedMessages: new Map() });
    return copy;
  });
}

/** Prepend a message to a conversation (newest first). Keeps last 50. */
export function addSecretMessage(peerId, msg) {
  secretChats.update((m) => {
    const copy = new Map(m);
    const chat = copy.get(peerId) ?? { messages: [], locked: true, decryptedMessages: new Map() };
    const msgs = [msg, ...chat.messages].slice(0, 50);
    copy.set(peerId, { ...chat, messages: msgs });
    return copy;
  });
}

/**
 * Add an optimistic (pre-ACK) message. Uses a negative tempId so it never
 * collides with BIGSERIAL server IDs (always positive).
 * msg must include: { id (negative), senderId, receiverId, pending: true, createdAt }
 */
export function addOptimisticMessage(peerId, msg) {
  secretChats.update((m) => {
    const copy = new Map(m);
    const chat = copy.get(peerId) ?? { messages: [], locked: true, decryptedMessages: new Map() };
    const msgs = [msg, ...chat.messages].slice(0, 51);
    copy.set(peerId, { ...chat, messages: msgs });
    return copy;
  });
}

/**
 * Replace the pending message with the server-confirmed message.
 * Migrates the decrypted plaintext from tempId → confirmedMsg.id so
 * retry still works if the user re-opens the chat.
 */
export function confirmOptimisticMessage(peerId, confirmedMsg) {
  secretChats.update((m) => {
    const copy = new Map(m);
    const chat = copy.get(peerId);
    if (!chat) {
      const base = { messages: [], locked: true, decryptedMessages: new Map() };
      copy.set(peerId, { ...base, messages: [confirmedMsg] });
      return copy;
    }
    const pendingIdx = chat.messages.findIndex((msg) => msg.pending === true);
    if (pendingIdx === -1) {
      // No pending slot — just prepend (shouldn't normally happen)
      const msgs = [confirmedMsg, ...chat.messages].slice(0, 50);
      copy.set(peerId, { ...chat, messages: msgs });
      return copy;
    }
    const tempId = chat.messages[pendingIdx].id;
    const tempDecrypted = chat.decryptedMessages.get(tempId);
    const msgs = [...chat.messages];
    msgs[pendingIdx] = { ...confirmedMsg, pending: false };
    const dm = new Map(chat.decryptedMessages);
    dm.delete(tempId);
    if (tempDecrypted !== undefined && confirmedMsg.id) {
      dm.set(confirmedMsg.id, tempDecrypted);
    }
    copy.set(peerId, { ...chat, messages: msgs, decryptedMessages: dm });
    return copy;
  });
}

/** Mark the pending message as failed (network/encryption error). */
export function failOptimisticMessage(peerId, tempId) {
  secretChats.update((m) => {
    const copy = new Map(m);
    const chat = copy.get(peerId);
    if (!chat) return copy;
    const msgs = chat.messages.map((msg) =>
      msg.id === tempId ? { ...msg, pending: false, failed: true } : msg
    );
    copy.set(peerId, { ...chat, messages: msgs });
    return copy;
  });
}

/** Remove a specific optimistic message by tempId (negative id). */
export function removeSecretMessageByTempId(peerId, tempId) {
  secretChats.update((m) => {
    const copy = new Map(m);
    const chat = copy.get(peerId);
    if (!chat) return copy;
    const msgs = chat.messages.filter((msg) => msg.id !== tempId);
    const dm = new Map(chat.decryptedMessages);
    dm.delete(tempId);
    copy.set(peerId, { ...chat, messages: msgs, decryptedMessages: dm });
    return copy;
  });
}

/**
 * Replace the full message list for a peer (used by history fetch).
 * Preserves any pending/failed optimistic messages not yet confirmed by the server,
 * so a history refresh doesn't wipe in-flight messages.
 */
export function setSecretMessages(peerId, messages) {
  secretChats.update((m) => {
    const copy = new Map(m);
    const existing = copy.get(peerId) ?? { messages: [], locked: true, decryptedMessages: new Map() };
    const serverIds = new Set(messages.map((msg) => msg.id));
    // Keep optimistic messages that aren't yet reflected in the server list
    const optimistic = existing.messages.filter(
      (msg) => (msg.pending || msg.failed) && !serverIds.has(msg.id)
    );
    copy.set(peerId, { ...existing, messages: [...optimistic, ...messages] });
    return copy;
  });
}

/** Store a successfully decrypted plaintext for a message. Unlocks the conversation. */
export function storeDecrypted(peerId, msgId, plaintext) {
  secretChats.update((m) => {
    const copy = new Map(m);
    const chat = copy.get(peerId);
    if (!chat) return copy;
    const dm = new Map(chat.decryptedMessages);
    dm.set(msgId, plaintext);
    copy.set(peerId, { ...chat, locked: false, decryptedMessages: dm });
    return copy;
  });
}

/** Update the seenAt timestamp on a message (sender gets notified when receiver decrypts). */
export function updateSecretMessageSeen(msgId, seenAt) {
  secretChats.update((m) => {
    const copy = new Map(m);
    for (const [peerId, chat] of copy) {
      const idx = chat.messages.findIndex((msg) => msg.id === msgId);
      if (idx !== -1) {
        const updated = [...chat.messages];
        updated[idx] = { ...updated[idx], seenAt };
        copy.set(peerId, { ...chat, messages: updated });
        break;
      }
    }
    return copy;
  });
}

/** Remove a deleted message from all conversations. */
export function removeSecretMessage(msgId) {
  secretChats.update((m) => {
    const copy = new Map(m);
    for (const [peerId, chat] of copy) {
      const msgs = chat.messages.filter((msg) => msg.id !== msgId);
      if (msgs.length !== chat.messages.length) {
        copy.set(peerId, { ...chat, messages: msgs });
      }
    }
    return copy;
  });
}
