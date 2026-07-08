import { adminOverview } from '../stores/admin.js';
import { networkGraph } from '../stores/network.js';
import { dailyActivity } from '../stores/activity.js';
import { recentTrailResult } from '../stores/trail.js';
import { incomingOffer } from '../stores/webrtc.js';

/**
 * Small cross-domain handlers grouped together: admin overview, network
 * graph, daily activity summary, trail playback, and WebRTC offer relay.
 */
export function register(socket) {
  // Admin overview
  socket.on('adminOverview', (data) => { if (data) adminOverview.set(data); });

  // Network graph
  socket.on('networkGraph', (data) => { if (data) networkGraph.set(data); });

  // ── F9: Daily activity summary ─────────────────────────────────────
  socket.on('dailyActivity', (data) => {
    if (!data?.userId) return;
    dailyActivity.update(m => {
      const nm = new Map(m);
      nm.set(data.userId, data.days || []);
      return nm;
    });
  });

  // ── F10: Trail playback ────────────────────────────────────────────
  socket.on('recentTrailData', (data) => {
    if (!data) return;
    recentTrailResult.set({ ok: true, ...data });
  });
  socket.on('trailError', (data) => {
    recentTrailResult.set({ ok: false, error: data?.error || 'Could not load trail' });
  });

  // ── WebRTC signaling relay ─────────────────────────────────────────
  // Only the offer sets the store (triggers IncomingCallOverlay).
  // answer/ice/hangup are handled by webrtc.js initWebRTCSocketHandlers().
  socket.on('webrtc:offer', (data) => { incomingOffer.set(data); });
}
