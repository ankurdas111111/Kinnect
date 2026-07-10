/**
 * Geocoding, Place Search, and Routing — via backend proxies.
 *
 * - Reverse geocoding: /api/geocode       → Nominatim
 * - Place search:      /api/search        → proxied autocomplete
 * - Routing:           /api/route         → proxied routing (FOSSGIS/Ola)
 */

import { apiGet } from './api.js';
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

// ── Place Search — via backend proxy ─────────────────────────────────────
// GET /api/search?q=<query>&lat=<f>&lng=<f>
// → { results: [{ name, sub, lat, lng, type }] }

export async function searchPlaces(query, options = {}) {
  if (!query || query.length < 2) return [];

  const { limit = 6, lat, lng } = options;
  const params = new URLSearchParams({ q: query });
  if (lat != null && lng != null) {
    params.set('lat', String(lat));
    params.set('lng', String(lng));
  }
  if (limit !== 6) params.set('limit', String(limit));

  try {
    const data = await apiGet(`/api/search?${params}`);
    if (!data?.results) return [];
    return data.results.map(r => ({
      lat: r.lat,
      lng: r.lng,
      name: r.name || '',
      sub: r.sub || '',
      type: r.type || '',
      distanceM: r.distanceM || 0,
      city: '',
      state: '',
      country: '',
      displayName: [r.name, r.sub].filter(Boolean).join(', '),
    }));
  } catch {
    return [];
  }
}

// ── Routing — via backend proxy ──────────────────────────────────────────
// GET /api/route?mode=car|foot|bike|scooter&from=<lat,lng>&to=<lat,lng>
// → { distanceM, durationS, geometry: GeoJSON LineString, steps: [{ instruction, distanceM, lat, lng }] }

/**
 * Get directions between two points.
 * @param {number} fromLat
 * @param {number} fromLng
 * @param {number} toLat
 * @param {number} toLng
 * @param {'car'|'foot'|'bike'|'scooter'} mode
 * @returns {{ distance, duration, geometry, steps, summary } | null}
 */
export async function getDirections(fromLat, fromLng, toLat, toLng, mode = 'car') {
  const params = new URLSearchParams({
    mode,
    from: `${fromLat},${fromLng}`,
    to: `${toLat},${toLng}`,
  });

  try {
    const data = await apiGet(`/api/route?${params}`);
    if (!data || data.ok === false || !data.geometry) return null;

    return {
      distance: data.distanceM,
      duration: data.durationS,
      geometry: data.geometry,
      steps: (data.steps || []).map(s => ({
        instruction: s.instruction || '',
        distance: s.distanceM ?? 0,
        duration: null,
        name: '',
        lat: s.lat,
        lng: s.lng,
      })),
      summary: '',
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
