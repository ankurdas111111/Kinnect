import { writable } from 'svelte/store';

// Map<userId, {placeName, placeId, etaSeconds, distanceM, displayName, confidence}>
export const arrivalProjections = writable(new Map());
