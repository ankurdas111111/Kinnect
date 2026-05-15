/**
 * webrtc.js — WebRTC walkie-talkie signaling + PTT logic.
 *
 * Architecture:
 *  - Server is a pure signaling relay; audio travels peer-to-peer.
 *  - One RTCPeerConnection per active call. PTT toggles audio track enabled flag
 *    (no ICE re-negotiation — instant transmit).
 *  - STUN-only (Google public servers, zero cost).
 */

import { get } from 'svelte/store';
import { callState, callPeer, incomingOffer } from './stores/webrtc.js';
import {
  emitWebRTCOffer,
  emitWebRTCAnswer,
  emitWebRTCIce,
  emitWebRTCHangup,
  socket,
} from './socket.js';

const ICE_SERVERS = [
  { urls: 'stun:stun.l.google.com:19302' },
  { urls: 'stun:stun1.l.google.com:19302' },
];

/** Module-level call state (mirrors stores for imperative code). */
let pc = null;
let localStream = null;
let currentPeer = null; // { userID, displayName }
let _handlersInited = false;

// ── Socket handler registration ───────────────────────────────────────────────

/**
 * Register socket listeners for WebRTC signaling messages.
 * Idempotent — safe to call from multiple component mounts.
 */
export function initWebRTCSocketHandlers() {
  if (_handlersInited) return;
  _handlersInited = true;

  socket.on('webrtc:answer', async ({ sdp }) => {
    if (!pc) return;
    try {
      await pc.setRemoteDescription({ type: 'answer', sdp });
    } catch (e) {
      console.error('[webrtc] setRemoteDescription answer failed', e);
    }
  });

  socket.on('webrtc:ice', async ({ candidate }) => {
    if (!pc || !candidate) return;
    try {
      await pc.addIceCandidate(candidate);
    } catch (e) {
      // Benign — can fire after connection is closed
    }
  });

  socket.on('webrtc:hangup', () => {
    _cleanup();
    callState.set('idle');
    callPeer.set(null);
  });
}

// ── Call initiation (caller side) ────────────────────────────────────────────

/**
 * Start a walkie-talkie call to targetUserID.
 * Mic is acquired muted; user must hold PTT to transmit.
 */
export async function startCall(targetUserID, displayName) {
  if (get(callState) !== 'idle') return;

  currentPeer = { userID: targetUserID, displayName };
  callState.set('calling');
  callPeer.set(currentPeer);

  try {
    localStream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
    // Start muted — user holds PTT to speak
    localStream.getAudioTracks().forEach((t) => { t.enabled = false; });

    pc = _createPeerConnection(targetUserID);
    localStream.getTracks().forEach((t) => pc.addTrack(t, localStream));

    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    emitWebRTCOffer(targetUserID, offer.sdp);
  } catch (e) {
    console.error('[webrtc] startCall failed', e);
    _cleanup();
    callState.set('idle');
    callPeer.set(null);
  }
}

// ── Call acceptance (callee side) ────────────────────────────────────────────

/**
 * Accept an incoming call. Called when user taps "Accept" in IncomingCallOverlay.
 */
export async function acceptCall(fromUserID, fromName, remoteSdp) {
  if (get(callState) !== 'idle') return;

  currentPeer = { userID: fromUserID, displayName: fromName };
  callState.set('connected');
  callPeer.set(currentPeer);
  incomingOffer.set(null);

  try {
    localStream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
    localStream.getAudioTracks().forEach((t) => { t.enabled = false; });

    pc = _createPeerConnection(fromUserID);
    localStream.getTracks().forEach((t) => pc.addTrack(t, localStream));

    await pc.setRemoteDescription({ type: 'offer', sdp: remoteSdp });
    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);
    emitWebRTCAnswer(fromUserID, answer.sdp);
  } catch (e) {
    console.error('[webrtc] acceptCall failed', e);
    _cleanup();
    callState.set('idle');
    callPeer.set(null);
  }
}

// ── PTT controls ─────────────────────────────────────────────────────────────

/** Hold to talk — enable the audio track. */
export function startTransmitting() {
  if (!localStream) return;
  localStream.getAudioTracks().forEach((t) => { t.enabled = true; });
}

/** Release to listen — disable the audio track. */
export function stopTransmitting() {
  if (!localStream) return;
  localStream.getAudioTracks().forEach((t) => { t.enabled = false; });
}

// ── Hangup ───────────────────────────────────────────────────────────────────

/** End the active call from either side. */
export function hangup() {
  const peer = currentPeer;
  _cleanup();
  callState.set('idle');
  callPeer.set(null);
  if (peer?.userID) emitWebRTCHangup(peer.userID);
}

// ── Internal helpers ─────────────────────────────────────────────────────────

function _createPeerConnection(targetUserID) {
  const conn = new RTCPeerConnection({ iceServers: ICE_SERVERS });

  conn.onicecandidate = ({ candidate }) => {
    if (candidate) emitWebRTCIce(targetUserID, candidate);
  };

  conn.onconnectionstatechange = () => {
    if (conn.connectionState === 'connected') {
      callState.set('connected');
    } else if (
      conn.connectionState === 'disconnected' ||
      conn.connectionState === 'failed' ||
      conn.connectionState === 'closed'
    ) {
      _cleanup();
      callState.set('idle');
      callPeer.set(null);
    }
  };

  conn.ontrack = ({ streams }) => {
    const remoteStream = streams[0];
    if (!remoteStream) return;
    // Play remote audio via Web Audio (no <audio> element needed in DOM)
    const audioCtx = new AudioContext();
    const src = audioCtx.createMediaStreamSource(remoteStream);
    src.connect(audioCtx.destination);
  };

  return conn;
}

function _cleanup() {
  if (localStream) {
    localStream.getTracks().forEach((t) => t.stop());
    localStream = null;
  }
  if (pc) {
    pc.close();
    pc = null;
  }
  currentPeer = null;
}
