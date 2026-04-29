import { writable } from 'svelte/store';

export const savedPlaces = writable([]);
export const placeAlerts = writable([]);
export const speedAlerts = writable([]);
export const privacyPause = writable(null);

// F6: geofence entry/exit event log
export const geofenceLog = writable([]);

// F7: proximity alerts configured by the current user
export const proximityAlerts = writable([]);
