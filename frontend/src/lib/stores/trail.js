import { writable } from 'svelte/store';
export const trailData = writable(new Map()); // userId -> { userId, points: [{lat, lng, ts}] }

// Last response from getRecentTrail: { ok: bool, points?, windowMinutes?, targetUserId?, error? }
// Set by socket.js; consumed by Map.svelte to draw the trail layer.
export const recentTrailResult = writable(null);
