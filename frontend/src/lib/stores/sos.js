import { writable } from 'svelte/store';

export const banner = writable({ type: null, text: null, actions: [] });
export const alertState = writable({ visible: false, title: '', body: '', actions: [], alarmMs: 0 });
export const myLiveLinks = writable([]);
export const mySosActive = writable(false);

// sosNarratives: Map<userId, {sosToken, narrative: {trackGeojson, motionSummary, batteryPct, triggerRule, lastSignalTs}}>
export const sosNarratives = writable(new Map());

// activeSosUsers: Map<userId, sosPayload> for AlertOverlay to show narrative
export const activeSosUsers = writable(new Map());

// Increment to trigger a camera shake in AlertOverlay without starting the audio alarm.
// Used by geofence breach events (placeAlert leave).
export const geofenceShake = writable(0);
