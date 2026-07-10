/**
 * navigation.js — Pure JS navigation helper. Zero API calls per GPS fix, zero DOM.
 *
 * Exported API:
 *   snapToRoute(lat, lng, coordinates) → { snappedLat, snappedLng, segmentIndex, distanceFromRouteM, remainingDistanceM }
 *   createFollower(opts)               → { feed(fix), stop() }
 */

// ── Geometry helpers (inline; keeps module self-contained) ───────────────

const R_EARTH = 6371000; // metres

function toRad(deg) { return deg * Math.PI / 180; }

/**
 * Compass bearing (0–360°) from point A → point B using the haversine / atan2 formula.
 */
function computeBearing(lat1, lng1, lat2, lng2) {
  const dLng = toRad(lng2 - lng1);
  const y = Math.sin(dLng) * Math.cos(toRad(lat2));
  const x = Math.cos(toRad(lat1)) * Math.sin(toRad(lat2)) -
            Math.sin(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.cos(dLng);
  return (Math.atan2(y, x) * 180 / Math.PI + 360) % 360;
}

/**
 * Bearing of the route segment at segmentIndex (fallback when the user hasn't moved).
 */
function segmentBearing(coords, segmentIndex) {
  if (!coords || coords.length < 2) return 0;
  const i = Math.min(segmentIndex, coords.length - 2);
  const [aLng, aLat] = coords[i];
  const [bLng, bLat] = coords[i + 1];
  return computeBearing(aLat, aLng, bLat, bLng);
}

function haversine(lat1, lng1, lat2, lng2) {
  const dLat = toRad(lat2 - lat1);
  const dLng = toRad(lng2 - lng1);
  const a = Math.sin(dLat / 2) ** 2 +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLng / 2) ** 2;
  return 2 * R_EARTH * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

/**
 * Project point (pLat, pLng) onto segment (aLat,aLng)→(bLat,bLng)
 * using equirectangular approximation (fine for short road segments).
 * Returns { t, lat, lng } where t ∈ [0,1].
 */
function nearestOnSegment(pLat, pLng, aLat, aLng, bLat, bLng) {
  const midCos = Math.cos(toRad((aLat + bLat) / 2));
  const dx = (bLng - aLng) * midCos;
  const dy = bLat - aLat;
  const len2 = dx * dx + dy * dy;
  if (len2 === 0) return { t: 0, lat: aLat, lng: aLng };
  const px = (pLng - aLng) * midCos;
  const py = pLat - aLat;
  const t = Math.max(0, Math.min(1, (px * dx + py * dy) / len2));
  return { t, lat: aLat + t * (bLat - aLat), lng: aLng + t * (bLng - aLng) };
}

// ── snapToRoute ───────────────────────────────────────────────────────────

/**
 * Snap a GPS fix to the nearest point on the route polyline.
 *
 * @param {number} lat
 * @param {number} lng
 * @param {Array<[number, number]>} coordinates  GeoJSON coords [[lng, lat], ...]
 * @returns {{ snappedLat, snappedLng, segmentIndex, distanceFromRouteM, remainingDistanceM }}
 */
export function snapToRoute(lat, lng, coordinates) {
  if (!coordinates || coordinates.length < 2) {
    return { snappedLat: lat, snappedLng: lng, segmentIndex: 0, distanceFromRouteM: 0, remainingDistanceM: 0 };
  }

  let bestLat = coordinates[0][1];
  let bestLng = coordinates[0][0];
  let bestDist = Infinity;
  let bestSeg = 0;

  for (let i = 0; i < coordinates.length - 1; i++) {
    const [aLng, aLat] = coordinates[i];
    const [bLng, bLat] = coordinates[i + 1];
    const n = nearestOnSegment(lat, lng, aLat, aLng, bLat, bLng);
    const d = haversine(lat, lng, n.lat, n.lng);
    if (d < bestDist) {
      bestDist = d;
      bestLat = n.lat;
      bestLng = n.lng;
      bestSeg = i;
    }
  }

  // Remaining distance: from snap point to end of route along polyline
  let remaining = haversine(bestLat, bestLng, coordinates[bestSeg + 1][1], coordinates[bestSeg + 1][0]);
  for (let i = bestSeg + 1; i < coordinates.length - 1; i++) {
    remaining += haversine(coordinates[i][1], coordinates[i][0], coordinates[i + 1][1], coordinates[i + 1][0]);
  }

  return {
    snappedLat: bestLat,
    snappedLng: bestLng,
    segmentIndex: bestSeg,
    distanceFromRouteM: bestDist,
    remainingDistanceM: remaining,
  };
}

// ── createFollower ────────────────────────────────────────────────────────

/** Arrival radius per mode (metres) */
const ARRIVAL_RADIUS = { foot: 50, bike: 100, car: 150, scooter: 150 };
const OFF_ROUTE_M = 50;           // metres before considering off-route
const OFF_ROUTE_STREAK = 3;       // consecutive off-route fixes required
const OFF_ROUTE_COOLDOWN = 30000; // ms minimum between onOffRoute calls
const SPEED_WINDOW = 5;           // rolling average window (fixes)
const MIN_SPEED_MS = 0.5;         // m/s; below this use route-proportional ETA

/**
 * Create a live navigation follower.
 *
 * onUpdate payload shape:
 *   { snappedLat, snappedLng, segmentIndex, distanceFromRouteM, remainingDistanceM,
 *     etaS, bearing, currentStepIndex, distanceToStepM }
 *
 * @param {{
 *   geometry: { type: 'LineString', coordinates: [[lng, lat], ...] },
 *   durationS: number,
 *   distanceM: number,
 *   mode: 'car'|'foot'|'bike'|'scooter',
 *   steps?: Array<{ instruction: string, distanceM: number, lat: number, lng: number }>,
 *   onUpdate: (info: object) => void,
 *   onOffRoute: () => void,
 *   onArrive: () => void,
 * }} opts
 * @returns {{ feed(fix: { lat, lng, timestamp?: number }): void, stop(): void }}
 */
export function createFollower({ geometry, durationS, distanceM, mode, steps, onUpdate, onOffRoute, onArrive }) {
  const coords = geometry?.coordinates ?? [];
  const radius = ARRIVAL_RADIUS[mode] ?? 150;

  let stopped = false;
  let arrived = false;

  let offRouteStreak = 0;
  let lastOffRouteAt = 0;

  /** @type {{ dist: number, dt: number }[]} */
  const speedBuf = [];
  let prevFix = null;

  // ── Bearing tracking ─────────────────────────────────────────────────────
  let prevSnappedLat = null;
  let prevSnappedLng = null;

  // ── Step tracking ─────────────────────────────────────────────────────────
  // Precompute each step's cumulative distance from route start by snapping
  // each step lat/lng onto the polyline.
  // cumulative[i] = distanceM - remainingDistanceM at step i's snapped position
  const stepCumulatives = [];
  if (steps && steps.length > 0) {
    for (const step of steps) {
      const s = snapToRoute(step.lat, step.lng, coords);
      stepCumulatives.push(distanceM - s.remainingDistanceM);
    }
  }

  function feed(fix) {
    if (stopped || arrived) return;

    const snap = snapToRoute(fix.lat, fix.lng, coords);

    // ── Bearing ───────────────────────────────────────────────────────────
    let bearing;
    if (prevSnappedLat === null) {
      // No previous fix yet — use the current segment's geometric bearing
      bearing = segmentBearing(coords, snap.segmentIndex);
      prevSnappedLat = snap.snappedLat;
      prevSnappedLng = snap.snappedLng;
    } else {
      const dFromPrev = haversine(prevSnappedLat, prevSnappedLng, snap.snappedLat, snap.snappedLng);
      if (dFromPrev > 3) {
        // Moved >3 m: compute course from previous → current snapped position
        bearing = computeBearing(prevSnappedLat, prevSnappedLng, snap.snappedLat, snap.snappedLng);
        prevSnappedLat = snap.snappedLat;
        prevSnappedLng = snap.snappedLng;
      } else {
        // Stationary / jitter: fall back to segment bearing to avoid spinning
        bearing = segmentBearing(coords, snap.segmentIndex);
      }
    }

    // ── Step tracking ─────────────────────────────────────────────────────
    // distanceTraveled = total route length minus remaining
    let currentStepIndex = stepCumulatives.length > 0 ? stepCumulatives.length - 1 : 0;
    let distanceToStepM = 0;
    if (stepCumulatives.length > 0) {
      const distanceTraveled = distanceM - snap.remainingDistanceM;
      let found = false;
      for (let i = 0; i < stepCumulatives.length; i++) {
        if (stepCumulatives[i] > distanceTraveled + 5) {
          currentStepIndex = i;
          distanceToStepM = stepCumulatives[i] - distanceTraveled;
          found = true;
          break;
        }
      }
      if (!found) {
        // Past all waypoints — show remaining route distance for last step
        distanceToStepM = Math.max(0, snap.remainingDistanceM);
      }
    }

    // ── Arrival ───────────────────────────────────────────────────────────
    // Must also be ON the route — avoids false-firing on parallel roads.
    if (snap.remainingDistanceM <= radius && snap.distanceFromRouteM <= Math.max(radius, OFF_ROUTE_M)) {
      arrived = true;
      onUpdate({ ...snap, etaS: 0, bearing, currentStepIndex, distanceToStepM });
      onArrive();
      return;
    }

    // ── Off-route detection ───────────────────────────────────────────────
    if (snap.distanceFromRouteM > OFF_ROUTE_M) {
      offRouteStreak++;
      if (offRouteStreak >= OFF_ROUTE_STREAK) {
        const now = Date.now();
        if (now - lastOffRouteAt >= OFF_ROUTE_COOLDOWN) {
          lastOffRouteAt = now;
          onOffRoute();
        }
      }
    } else {
      offRouteStreak = 0;
    }

    // ── Speed rolling average ─────────────────────────────────────────────
    let actualSpeedMs = 0;
    if (prevFix) {
      const dFix = haversine(prevFix.lat, prevFix.lng, fix.lat, fix.lng);
      const dtFix = (fix.timestamp != null && prevFix.timestamp != null)
        ? (fix.timestamp - prevFix.timestamp) / 1000
        : 1;
      if (dtFix > 0) {
        speedBuf.push({ dist: dFix, dt: dtFix });
        if (speedBuf.length > SPEED_WINDOW) speedBuf.shift();
      }
      if (speedBuf.length > 0) {
        const totDist = speedBuf.reduce((s, h) => s + h.dist, 0);
        const totTime = speedBuf.reduce((s, h) => s + h.dt, 0);
        actualSpeedMs = totTime > 0 ? totDist / totTime : 0;
      }
    }
    prevFix = { lat: fix.lat, lng: fix.lng, timestamp: fix.timestamp ?? Date.now() };

    // ── ETA: prefer real speed; fall back to route-proportional ──────────
    let etaS;
    if (actualSpeedMs > MIN_SPEED_MS) {
      etaS = snap.remainingDistanceM / actualSpeedMs;
    } else if (distanceM > 0) {
      etaS = durationS * (snap.remainingDistanceM / distanceM);
    } else {
      etaS = 0;
    }

    onUpdate({ ...snap, etaS, bearing, currentStepIndex, distanceToStepM });
  }

  function stop() { stopped = true; }

  return { feed, stop };
}
