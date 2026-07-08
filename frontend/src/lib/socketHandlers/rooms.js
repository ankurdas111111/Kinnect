import { myRooms, myShareCode, myContactInfo, roomNotes } from '../stores/rooms.js';
import { bumpHubBadge } from '../stores/hubBadge.js';

/**
 * Room-domain socket handlers: share code, room membership, meeting points,
 * and the room bulletin board (notes).
 */
export function register(socket, ctx) {
  const { setBanner } = ctx;

  // Share code + personal info
  socket.on('myShareCode', (data) => {
    if (data && data.shareCode) myShareCode.set(data.shareCode);
    if (data) myContactInfo.set({ email: data.email || '', mobile: data.mobile || '' });
  });

  // Rooms
  socket.on('myRooms', (data) => myRooms.set(data || []));

  // Room action results
  socket.on('roomError', (data) => {
    setBanner({ type: 'info', text: data?.message || 'Something went wrong with this room', actions: [] }, 2500);
  });
  socket.on('roomCreated', (data) => {
    setBanner({ type: 'info', text: `"${data.name}" created — share code ${data.code} with family`, actions: [] }, 4000);
  });
  socket.on('roomJoined', (data) => {
    setBanner({ type: 'info', text: `You joined "${data.name}"`, actions: [] }, 2000);
    bumpHubBadge(false);
  });
  socket.on('roomLeft', (data) => {
    setBanner({ type: 'info', text: `You left "${data?.name || 'the room'}"`, actions: [] }, 2000);
  });

  // ── F3: Meeting point updated ──────────────────────────────────────
  socket.on('meetingPointUpdated', (data) => {
    if (!data?.roomCode) return;
    myRooms.update(rooms => rooms.map(r => {
      if (r.code !== data.roomCode) return r;
      if (data.lat != null && data.lng != null) {
        return { ...r, meetingPoint: { lat: data.lat, lng: data.lng, label: data.label || '', setBy: data.setBy || '', setAt: data.setAt || 0 } };
      }
      // cleared
      const { meetingPoint: _mp, ...rest } = r;
      return rest;
    }));
  });

  // ── F8: Room bulletin board ────────────────────────────────────────
  socket.on('roomNotes', (data) => {
    if (!data?.roomCode) return;
    roomNotes.update(m => {
      const nm = new Map(m);
      nm.set(data.roomCode, data.notes || []);
      return nm;
    });
  });

  // Go sends note fields flat on the payload (id, roomCode, authorId, authorName, body, createdAt).
  // Extract the note object without roomCode and prepend it to the room's list.
  socket.on('roomNoteAdded', (data) => {
    if (!data?.roomCode || !data?.id) return;
    const { roomCode, ...note } = data;
    roomNotes.update(m => {
      const nm = new Map(m);
      const existing = nm.get(roomCode) || [];
      nm.set(roomCode, [note, ...existing].slice(0, 20));
      return nm;
    });
  });

  socket.on('roomNoteDeleted', (data) => {
    if (!data?.noteId || !data?.roomCode) return;
    roomNotes.update(m => {
      const nm = new Map(m);
      const existing = nm.get(data.roomCode) || [];
      nm.set(data.roomCode, existing.filter(n => n.id !== data.noteId));
      return nm;
    });
  });
}
