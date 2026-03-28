import { writable } from 'svelte/store';
export const trailData = writable(new Map()); // userId -> { userId, points: [{lat, lng, ts}] }
