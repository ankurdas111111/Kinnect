import { writable } from 'svelte/store';

// Map<peerId, { messages: Array, locked: boolean, decryptedMessages: Map<msgId, string> }>
export const secretChats = writable(new Map());

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

/** Replace the full message list for a peer (used by history fetch). */
export function setSecretMessages(peerId, messages) {
  secretChats.update((m) => {
    const copy = new Map(m);
    const existing = copy.get(peerId) ?? { messages: [], locked: true, decryptedMessages: new Map() };
    copy.set(peerId, { ...existing, messages });
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
