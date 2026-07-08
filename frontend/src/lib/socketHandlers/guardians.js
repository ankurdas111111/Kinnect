import { myGuardianData, canManage, pendingIncomingRequests } from '../stores/guardians.js';
import { notifyGuardianRequest } from '../nativeNotifications.js';

/**
 * Guardian/ward-domain socket handlers: guardian relationships, pending
 * incoming requests, room-admin voting, and guardian request lifecycle.
 */
export function register(socket, ctx) {
  const { setBanner } = ctx;

  // Guardians
  socket.on('myGuardians', (data) => {
    if (!data) return;
    myGuardianData.set({
      asGuardian: data.asGuardian ?? [],
      asWard: data.asWard ?? [],
      manageable: data.manageable ?? []
    });
    const cm = new Map();
    (data.manageable || []).forEach(m => cm.set(m.userId, m.displayName));
    canManage.set(cm);
  });

  // Persistent pending requests (sent on connect, survives reconnects)
  socket.on('pendingRequests', (data) => {
    if (!Array.isArray(data)) return;
    pendingIncomingRequests.set(data);
  });

  // Guardian/admin request events (real-time notifications)
  socket.on('roomAdminRequest', (data) => {
    if (!data) return;
    pendingIncomingRequests.update(arr => {
      if (arr.some(r => r.type === 'roomAdmin' && r.from === data.fromUserId && r.roomCode === data.roomCode)) return arr;
      return [...arr, { type: 'roomAdmin', from: data.fromUserId, fromName: data.fromName, roomCode: data.roomCode, expiresIn: data.expiresIn, approvals: data.approvals || 0, denials: data.denials || 0, totalEligible: data.totalEligible || 0, myVote: null }];
    });
    setBanner({ type: 'info', text: data.fromName + ' requested Room Admin in ' + data.roomCode + ' — Vote now!', actions: [] }, 3000);
  });

  socket.on('roomAdminVoteUpdate', (data) => {
    if (!data) return;
    if (data.denied) {
      pendingIncomingRequests.update(arr => arr.filter(r => !(r.type === 'roomAdmin' && r.from === data.userId && r.roomCode === data.roomCode)));
      return;
    }
    pendingIncomingRequests.update(arr => arr.map(r => {
      if (r.type === 'roomAdmin' && r.from === data.userId && r.roomCode === data.roomCode) {
        return { ...r, approvals: data.approvals, denials: data.denials, totalEligible: data.totalEligible, myVote: data.myVote };
      }
      return r;
    }));
  });

  // KR-002: guardianInfo is the success notification for requestGuardian / inviteGuardian.
  // Previously these used contactError which showed as a red error banner.
  socket.on('guardianInfo', (data) => {
    if (!data) return;
    setBanner({ type: 'info', text: data.message || 'Guardian request sent', actions: [] }, 3000);
  });

  socket.on('guardianRequest', (data) => {
    if (!data) return;
    pendingIncomingRequests.update(arr => {
      if (arr.some(r => r.type === 'guardian' && r.from === data.fromUserId)) return arr;
      return [...arr, { type: 'guardian', from: data.fromUserId, fromName: data.fromName, expiresIn: data.expiresIn }];
    });
    notifyGuardianRequest(data.fromName).catch(() => {});
    setBanner({ type: 'info', text: data.fromName + ' wants to be your guardian', actions: [] }, 3000);
  });

  socket.on('guardianInvite', (data) => {
    if (!data) return;
    pendingIncomingRequests.update(arr => {
      if (arr.some(r => r.type === 'guardianInvite' && r.from === data.fromUserId)) return arr;
      return [...arr, { type: 'guardianInvite', from: data.fromUserId, fromName: data.fromName, expiresIn: data.expiresIn }];
    });
    setBanner({ type: 'info', text: data.fromName + ' wants you to be their guardian', actions: [] }, 3000);
  });

  socket.on('roomAdminUpdated', (data) => {
    if (!data) return;
    if (data.role === 'admin' || data.role === 'denied') {
      pendingIncomingRequests.update(arr => arr.filter(r => !(r.type === 'roomAdmin' && r.from === data.userId && r.roomCode === data.roomCode)));
    }
    setBanner({ type: 'info', text: 'Room admin role updated in ' + data.roomCode, actions: [] }, 2000);
  });

  socket.on('guardianUpdated', (data) => {
    if (!data) return;
    if (data.status === 'active' || data.status === 'denied' || data.status === 'revoked' || data.status === 'expired') {
      pendingIncomingRequests.update(arr => arr.filter(r => {
        if (r.type === 'guardian' && r.from === data.guardianId) return false;
        if (r.type === 'guardianInvite' && r.from === data.wardId) return false;
        return true;
      }));
    }
    const statusMsg = data.status === 'active' ? 'accepted' : data.status === 'denied' ? 'declined' : data.status === 'revoked' ? 'ended' : data.status === 'expired' ? 'expired' : data.status;
    setBanner({ type: 'info', text: 'Guardian request ' + statusMsg, actions: [] }, 2000);
  });
}
