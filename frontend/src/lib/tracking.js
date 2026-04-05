export function formatCoordinate(coord) {
  return coord != null ? coord.toFixed(6) : 'N/A';
}

export function calculateDistance(lat1, lon1, lat2, lon2) {
  const R = 6371e3;
  const p1 = lat1 * Math.PI / 180;
  const p2 = lat2 * Math.PI / 180;
  const dp = (lat2 - lat1) * Math.PI / 180;
  const dl = (lon2 - lon1) * Math.PI / 180;
  const a = Math.sin(dp / 2) * Math.sin(dp / 2) +
    Math.cos(p1) * Math.cos(p2) * Math.sin(dl / 2) * Math.sin(dl / 2);
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  return R * c;
}

export function escapeAttr(str) {
  if (typeof str !== 'string') str = String(str == null ? '' : str);
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/'/g, '&#39;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

const _iconCache = new Map();

const _pinSizes = {
  self:    [32, 42],
  sos:     [32, 42],
  contact: [28, 37],
  default: [28, 37],
  offline: [24, 32],
  stored:  [20, 27],
};

/**
 * Create an SVG teardrop-pin marker element for MapLibre GL.
 * The pin tip sits at the bottom-center — use `anchor: 'bottom'`
 * on the MapLibre Marker so the tip lands on the exact coordinate.
 *
 * @param {string} color     CSS color for the pin fill
 * @param {string} [_text]   Unused, kept for call-site compat
 * @param {object} [options] { markerType, pulse }
 */
export function createMapIcon(color, _text, options = {}) {
  const type = options.markerType || 'default';
  const cacheKey = `${color}|${type}`;
  const cached = _iconCache.get(cacheKey);
  if (cached) return cached.cloneNode(true);

  const [w, h] = _pinSizes[type] || [28, 37];
  const el = document.createElement('div');
  el.className = `map-pin pin-${type}`;
  el.style.cssText = `width:${w}px;height:${h}px;`;

  if (type === 'self') {
    // MERIDIAN: Diamond marker — instantly distinguishable from contact pins
    el.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 34" width="${w}" height="${h}">`
      + `<path d="M12 0 L24 14 L12 34 L0 14 Z" fill="${escapeAttr(color)}" stroke="white" stroke-width="1.8"/>`
      + `<circle cx="12" cy="13" r="4.5" fill="white" fill-opacity="0.92"/>`
      + `</svg>`;
  } else {
    el.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 34" width="${w}" height="${h}">`
      + `<path d="M12 0C5.37 0 0 5.37 0 12c0 8.13 10.81 20.45 11.39 21.12a.77.77 0 0 0 1.22 0C13.19 32.45 24 20.13 24 12 24 5.37 18.63 0 12 0z" fill="${escapeAttr(color)}" stroke="white" stroke-width="1.8"/>`
      + `<circle cx="12" cy="11" r="5.8" fill="white" fill-opacity="0.93"/>`
      + `</svg>`;
  }

  _iconCache.set(cacheKey, el.cloneNode(true));
  return el;
}

/**
 * Generate a GeoJSON Polygon approximating a circle.
 * @param {[number, number]} center - [lng, lat]
 * @param {number} radiusMeters
 * @param {number} [points=64]
 */
export function circleGeoJSON(center, radiusMeters, points = 64) {
  const [lng, lat] = center;
  const km = radiusMeters / 1000;
  const distX = km / (111.320 * Math.cos(lat * Math.PI / 180));
  const distY = km / 110.574;
  const coords = [];
  for (let i = 0; i <= points; i++) {
    const theta = (i / points) * 2 * Math.PI;
    coords.push([lng + distX * Math.cos(theta), lat + distY * Math.sin(theta)]);
  }
  return { type: 'Feature', geometry: { type: 'Polygon', coordinates: [coords] }, properties: {} };
}

/**
 * Determine a user's presence state from their socket data.
 * Returns: 'now' | 'recent' | 'away' | 'gone' | 'sos'
 */
export function getPresenceState(user) {
  if (!user) return 'gone';
  if (user.sos?.active) return 'sos';
  if (user.online === false) {
    const ts = user.timestamp;
    if (!ts) return 'gone';
    const ago = Date.now() - ts;
    if (ago < 1800000) return 'away';
    return 'gone';
  }
  // User is connected — check how recently they sent position
  const ts = user.timestamp;
  if (!ts || (Date.now() - ts) < 45000) return 'now';
  return 'recent';
}

/**
 * Human-readable presence text for a user object.
 */
export function getPresenceText(user) {
  if (!user) return 'Not seen yet';
  if (user.sos?.active) return 'SOS Active';
  if (user.online !== false) {
    const ts = user.timestamp;
    if (!ts || (Date.now() - ts) < 45000) return 'Here now';
    return 'Connected · not moving';
  }
  const ts = user.timestamp;
  if (!ts) return 'Never connected';
  const ago = Date.now() - ts;
  if (ago < 60000)   return 'Just left';
  if (ago < 300000)  return `Active ${Math.round(ago / 60000)}m ago`;
  if (ago < 3600000) return `Away · ${Math.round(ago / 60000)}m ago`;
  if (ago < 86400000) return `Last seen ${Math.round(ago / 3600000)}h ago`;
  return 'A while ago';
}

/**
 * Create an avatar-style person marker for MapLibre GL.
 * Shows initials in a circle with a presence-state ring.
 * Use anchor: 'bottom' on the MapLibre Marker.
 */
export function createPersonMarker(options = {}) {
  const {
    displayName = '?',
    color = '#6366f1',
    isSelf = false,
    isSos = false,
    presenceState = 'gone',
    motionClass = '',
    quietHoursActive = false,
  } = options;

  const initials = (displayName || '?')
    .split(' ')
    .filter(Boolean)
    .map(n => n[0] || '')
    .join('')
    .toUpperCase()
    .slice(0, 2) || '?';

  const el = document.createElement('div');
  el.style.cssText = 'display:flex;flex-direction:column;align-items:center;cursor:pointer;';

  if (isSelf) {
    // Diamond — instantly distinguishable as "me"
    el.innerHTML =
      `<div style="width:40px;height:40px;border-radius:50%;background:rgba(8,8,20,0.92);border:2.5px solid #818cf8;box-shadow:0 0 0 3px rgba(99,102,241,0.14),0 0 20px rgba(99,102,241,0.28);display:flex;align-items:center;justify-content:center;animation:glow-breathe 2.5s ease-in-out infinite;">`
      + `<svg width="14" height="18" viewBox="0 0 14 18" fill="none"><path d="M7 0L14 7L7 18L0 7Z" fill="#818cf8"/></svg>`
      + `</div>`
      + `<div style="width:2px;height:7px;background:#818cf8;border-radius:0 0 2px 2px;margin:0 auto;opacity:0.6;"></div>`;
    return el;
  }

  // Presence ring style
  let ringBorder, ringGlow, textColor, tailColor;
  if (isSos) {
    ringBorder = '2.5px solid #ef4444';
    ringGlow   = '0 0 0 3px rgba(239,68,68,0.18),0 0 20px rgba(239,68,68,0.40)';
    textColor  = '#ef4444';
    tailColor  = '#ef4444';
  } else if (presenceState === 'now') {
    ringBorder = `2.5px solid ${color}`;
    ringGlow   = `0 0 0 3px ${color}28,0 0 14px ${color}40`;
    textColor  = 'white';
    tailColor  = color;
  } else if (presenceState === 'recent') {
    ringBorder = `2px solid ${color}`;
    ringGlow   = `0 0 8px ${color}30`;
    textColor  = 'white';
    tailColor  = color;
  } else if (presenceState === 'away') {
    ringBorder = `2px solid ${color}`;
    ringGlow   = 'none';
    textColor  = 'rgba(255,255,255,0.7)';
    tailColor  = color;
  } else {
    // gone / offline
    ringBorder = '2px solid #6b7280';
    ringGlow   = 'none';
    textColor  = '#9ca3af';
    tailColor  = '#6b7280';
  }

  const opacity = presenceState === 'away' ? '0.78' : presenceState === 'gone' ? '0.50' : '1';
  const animation = presenceState === 'now' && !isSos
    ? 'animation:person-marker-arrive 460ms cubic-bezier(0.34,1.56,0.64,1) both;'
    : 'animation:person-marker-arrive 460ms cubic-bezier(0.34,1.56,0.64,1) both;';

  const sosSvg = isSos
    ? `<div style="position:absolute;inset:-7px;border-radius:50%;border:2px solid rgba(239,68,68,0.50);animation:signal-ring-out 1.4s ease-out infinite;pointer-events:none;"></div>`
      + `<div style="position:absolute;inset:-7px;border-radius:50%;border:2px solid rgba(239,68,68,0.40);animation:signal-ring-out 1.4s ease-out 0.5s infinite;pointer-events:none;"></div>`
    : '';

  const motionBadgeColors = { still: '#6b7280', walk: '#22c55e', run: '#f59e0b', vehicle: '#3b82f6' };
  const badgeColor = motionBadgeColors[motionClass] || '';
  const motionBadge = badgeColor && !isSelf && !isSos && presenceState !== 'gone'
    ? `<div style="width:9px;height:9px;border-radius:50%;background:${badgeColor};border:1.5px solid rgba(8,8,20,0.9);position:absolute;bottom:-2px;right:-2px;pointer-events:none;"></div>`
    : '';

  // Moon badge for quiet hours (top-left corner)
  const moonBadge = quietHoursActive && !isSelf && !isSos
    ? `<div style="position:absolute;top:-4px;left:-4px;width:14px;height:14px;border-radius:50%;background:rgba(99,102,241,0.85);border:1.5px solid rgba(8,8,20,0.9);display:flex;align-items:center;justify-content:center;pointer-events:none;font-size:8px;" title="Quiet Hours active">🌙</div>`
    : '';

  el.innerHTML =
    `<div style="width:42px;height:42px;border-radius:50%;background:rgba(8,8,20,0.92);border:${ringBorder};box-shadow:${ringGlow};display:flex;align-items:center;justify-content:center;position:relative;opacity:${opacity};${animation}">`
    + `<span style="font-family:Inter,sans-serif;font-size:13px;font-weight:800;color:${textColor};text-transform:uppercase;letter-spacing:-0.02em;user-select:none;">${escapeAttr(initials)}</span>`
    + sosSvg
    + motionBadge
    + moonBadge
    + `</div>`
    + `<div style="width:2px;height:7px;background:${tailColor};border-radius:0 0 2px 2px;margin:0 auto;opacity:0.6;"></div>`;

  return el;
}

export function formatDistance(meters) {
  if (meters == null || !isFinite(meters)) return null;
  if (meters >= 1000) return (meters / 1000).toFixed(1) + ' km';
  return Math.round(meters) + ' m';
}

export function formatTimestamp(timestamp) {
  return new Date(timestamp).toLocaleTimeString();
}

export function escHtml(str) {
  const d = document.createElement('div');
  d.textContent = str == null ? '' : String(str);
  return d.innerHTML;
}
