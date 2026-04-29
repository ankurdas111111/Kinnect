import { writable } from 'svelte/store';

// userId → DailyActivityDay[]
export const dailyActivity = writable(new Map());
