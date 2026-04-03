import { writable } from 'svelte/store';

// crowdMode holds the local crowd / festival mode state.
export const crowdMode = writable({ active: false, radiusM: 200 });
