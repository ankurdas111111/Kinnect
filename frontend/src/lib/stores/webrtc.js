import { writable } from 'svelte/store';

// 'idle' | 'calling' | 'connected'
export const callState = writable('idle');

// { userID: string, displayName: string } | null
export const callPeer = writable(null);

// { fromUserID: string, fromName: string, sdp: string } | null
export const incomingOffer = writable(null);
