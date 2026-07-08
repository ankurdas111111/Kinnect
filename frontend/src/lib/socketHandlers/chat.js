import { get } from 'svelte/store';
import { authUser } from '../stores/auth.js';
import { addSecretMessage, confirmOptimisticMessage, setSecretMessages, removeSecretMessage, updateSecretMessageSeen, secretChatPresence } from '../stores/secretChat.js';

/**
 * Secret chat socket handlers — ciphertext-only delivery; decryption is client-side.
 */
export function register(socket) {
  socket.on('secretMsgReceived', (msg) => {
    if (!msg || !msg.senderId) return;
    addSecretMessage(msg.senderId, msg);
  });
  socket.on('secretMsgSent', (msg) => {
    if (!msg || !msg.receiverId) return;
    const myId = get(authUser)?.userId;
    // Guard: if auth store hasn't resolved yet, myId is undefined.
    // A message stored with senderId:undefined would be indistinguishable
    // from a received message (msg.senderId !== myId → always true).
    // Drop and rely on the secretMsgsHistory fetch to backfill instead.
    if (!myId) return;
    // Replace the optimistic pending message with the server-confirmed one.
    confirmOptimisticMessage(msg.receiverId, { ...msg, senderId: myId });
  });
  socket.on('secretMsgsHistory', (data) => {
    if (!data || !data.peerId) return;
    setSecretMessages(data.peerId, data.messages ?? []);
  });
  socket.on('secretMsgDeleted', (data) => {
    if (!data || !data.id) return;
    removeSecretMessage(data.id);
  });
  socket.on('secretMsgSeen', (data) => {
    if (!data || !data.id) return;
    updateSecretMessageSeen(data.id, data.seenAt);
  });
  socket.on('secretChatPresence', (data) => {
    if (!data || !data.userId) return;
    secretChatPresence.update(m => {
      const copy = new Map(m);
      copy.set(data.userId, { open: !!data.open, at: data.at ?? Date.now() });
      return copy;
    });
  });
}
