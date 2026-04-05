/**
 * Geocoding, Place Search (Photon), and Routing (OSRM).
 *
 * - Reverse geocoding: proxied through our backend (/api/geocode → Nominatim)
 * - Place search: Photon API by Komoot — instant autocomplete, no rate limit
 * - Routing: OSRM public server — free driving/walking/cycling directions
 */

import API_BASE from './env.js';

// ── Reverse geocoding ────────────────────────────────────────────────────

const _geoCache = new Map();
const GEO_CACHE_MAX = 500;

export async function reverseGeocode(lat, lng) {
  if (lat == null || lng == null) return null;
  const key = `${lat.toFixed(4)},${lng.toFixed(4)}`;
  if (_geoCache.has(key)) return _geoCache.get(key);

  try {
    const res = await fetch(`${API_BASE}/api/geocode?lat=${lat.toFixed(6)}&lng=${lng.toFixed(6)}`, {
      credentials: 'include',
    });
    if (!res.ok) return null;
    const data = await res.json();
    if (_geoCache.size >= GEO_CACHE_MAX) _geoCache.delete(_geoCache.keys().next().value);
    _geoCache.set(key, data);
    return data;
  } catch {
    return null;
  }
}

export function shortLabel(geo) {
  if (!geo) return '';
  const parts = [];
  if (geo.road) parts.push(geo.road);
  if (geo.suburb) parts.push(geo.suburb);
  else if (geo.city) parts.push(geo.city);
  return parts.join(', ') || geo.city || geo.displayName?.split(',')[0] || '';
}

// ── Place Search — Photon (Komoot) ───────────────────────────────────────
// Fast autocomplete, no API key, no strict rate limit.
// https://photon.komoot.io/

export async function searchPlaces(query, options = {}) {
  if (!query || query.length < 2) return [];

  const { limit = 6, lat, lng } = options;
  const params = new URLSearchParams({
    q: query,
    limit: String(limit),
    lang: 'en',
  });
  // Bias results toward user's location if available
  if (lat != null && lng != null) {
    params.set('lat', String(lat));
    params.set('lon', String(lng));
  }

  try {
    const res = await fetch(`https://photon.komoot.io/api/?${params}`);
    if (!res.ok) return [];
    const data = await res.json();
    if (!data.features) return [];

    return data.features.map(f => {
      const p = f.properties || {};
      const [fLng, fLat] = f.geometry?.coordinates || [0, 0];
      const name = p.name || p.street || p.city || query;
      const sub = [p.street, p.city || p.town || p.village, p.state]
        .filter(Boolean)
        .filter(s => s !== name)
        .join(', ');
      return {
        lat: fLat,
        lng: fLng,
        name,
        sub,
        type: p.osm_value || p.type || '',
        city: p.city || p.town || p.village || '',
        state: p.state || '',
        country: p.country || '',
        displayName: [name, sub].filter(Boolean).join(', '),
      };
    });
  } catch {
    return [];
  }
}

// ── Routing — OSRM (free public server) ──────────────────────────────────
// Returns { distance (m), duration (s), geometry (GeoJSON coords), steps }

const OSRM_BASE = 'https://routing.openstreetmap.de';

/**
 * Get driving/walking/cycling directions between two points.
 * @param {number} fromLat
 * @param {number} fromLng
 * @param {number} toLat
 * @param {number} toLng
 * @param {'car'|'foot'|'bike'} mode
 * @returns {{ distance, duration, geometry, steps, summary }}
 */
export async function getDirections(fromLat, fromLng, toLat, toLng, mode = 'car') {
  const profile = mode === 'foot' ? 'routed-foot' : mode === 'bike' ? 'routed-bike' : 'routed-car';
  const coords = `${fromLng},${fromLat};${toLng},${toLat}`;
  const url = `${OSRM_BASE}/${profile}/route/v1/driving/${coords}?overview=full&geometries=geojson&steps=true`;

  try {
    const res = await fetch(url);
    if (!res.ok) return null;
    const data = await res.json();
    if (data.code !== 'Ok' || !data.routes?.length) return null;

    const route = data.routes[0];
    const leg = route.legs?.[0];
    return {
      distance: route.distance,       // meters
      duration: route.duration,        // seconds
      geometry: route.geometry,        // GeoJSON LineString
      steps: (leg?.steps || []).map(s => ({
        instruction: s.maneuver?.type === 'turn'
          ? `Turn ${s.maneuver.modifier || ''} onto ${s.name || 'road'}`
          : s.maneuver?.type === 'depart'
          ? `Head ${s.maneuver.modifier || 'north'} on ${s.name || 'road'}`
          : s.maneuver?.type === 'arrive'
          ? 'You have arrived'
          : `${s.maneuver?.type || 'Continue'} ${s.maneuver?.modifier || ''} on ${s.name || 'road'}`.trim(),
        distance: s.distance,
        duration: s.duration,
        name: s.name || '',
      })),
      summary: leg?.summary || '',
    };
  } catch {
    return null;
  }
}

/** Format duration in seconds to human-readable string */
export function formatDuration(sec) {
  if (sec == null) return '';
  const h = Math.floor(sec / 3600);
  const m = Math.round((sec % 3600) / 60);
  if (h > 0) return `${h}h ${m}m`;
  return `${m} min`;
}

/** Format distance in meters to human-readable string */
export function formatDist(m) {
  if (m == null) return '';
  if (m >= 1000) return `${(m / 1000).toFixed(1)} km`;
  return `${Math.round(m)} m`;
}
