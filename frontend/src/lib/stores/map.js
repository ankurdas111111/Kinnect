import { writable } from 'svelte/store';

export const otherUsers = writable(new Map());
export const myLocation = writable(null);
export const mySocketId = writable(null);
export const tracking = writable(false);
export const selectedUsers = writable([]);
export const mySafetyStatus = writable({
  geofence: { enabled: false, centerLat: null, centerLng: null, radiusM: 0 },
  autoSos: { enabled: false, noMoveMinutes: 5, hardStopMinutes: 2, geofence: false },
  checkIn: { enabled: false, intervalMinutes: 5, overdueMinutes: 7, lastCheckInAt: null }
});

/**
 * Set to a socketId, userId, or '__self__' to fly the map to that user and open their popup.
 * Automatically resets to null after the map consumes it.
 */
export const focusUser = writable(null);

/**
 * Set to { lng, lat, zoom? } to fly the map to that location.
 * Automatically resets to null after the map consumes it.
 */
export const mapFlyTo = writable(null);

/**
 * Set to a GeoJSON LineString geometry to draw a route polyline on the map.
 * Set to null to clear the route.
 */
export const routeGeometry = writable(null);

/**
 * Set to { lat, lng, name } to pre-fill Walk With Me destination and open the modal.
 * Components that render WalkWithMe should subscribe and open their modal when set.
 */
export const walkDestination = writable(null);

/**
 * Navigation state for turn-by-turn directions.
 * Set to { active: true, destLat, destLng, destName, routeCoords: [[lng,lat],...] }
 * to enter navigation mode. Set to { active: false } to exit.
 */
export const navigationState = writable({ active: false });

/**
 * Set to a user object when a map marker is tapped on mobile.
 * UsersList watches this to open the quick-action sheet directly.
 * Automatically reset to null after consumption.
 */
export const mapTappedUser = writable(null);

/**
 * Set to { id: string, name: string } to open a secret chat directly from the map popup.
 * MainApp consumes and resets to null.
 */
export const mapChatRequest = writable(null);

/**
 * Navigation camera-follow state. Set by PlaceSearch (via createFollower onUpdate) while
 * navigation is active. Map.svelte subscribes and calls map.easeTo() on each update.
 *
 * active=true  → map smoothly follows at zoom≥16.5, pitch 50°, bearing from GPS course
 * active=false → map resets to pitch 0 / bearing 0
 */
export const navFollow = writable({ active: false, lat: 0, lng: 0, bearing: 0 });
