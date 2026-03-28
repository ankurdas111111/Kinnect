import { writable } from 'svelte/store';

// pulseMap: Map<userId, {userId, displayName, type:'ok'|'callme', lat, lng, expiresAt}>
export const pulseMap = writable(new Map());
