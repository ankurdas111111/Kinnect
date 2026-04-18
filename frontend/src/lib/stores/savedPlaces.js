import { writable } from 'svelte/store';

// Map of placeId → { id, userId, name, icon, latitude, longitude, radiusM, createdAt }
export const savedPlaces = writable(new Map());
