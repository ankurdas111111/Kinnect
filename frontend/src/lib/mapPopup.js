/**
 * mapPopup.js — safe HTML builders for MapLibre marker popups + custom marker
 * elements.
 *
 * WHY THIS EXISTS: MapLibre popups are set via `Popup.setHTML(string)`, which
 * assigns raw innerHTML. Building that string with template concatenation while
 * interpolating user-controlled fields (displayName, room names, SOS reason,
 * device type, connection quality, meeting-point labels) is XSS-adjacent — an
 * attacker-controlled name like `<img onerror=…>` would execute in every
 * viewer's map. Every function here routes ALL user-supplied strings through
 * `escapeAttr` (HTML-entity encoder) before they touch the string, and only
 * ever interpolates numbers/enums that this module itself derives.
 *
 * This module is presentation-only: it emits token-classed markup (.pu-* /
 * .tabular-nums) whose styling lives in Map.svelte's <style> block. It owns no
 * map state, no camera logic, no interpolation — pure string/DOM construction.
 */

import { escapeAttr, calculateDistance, formatDistance } from './tracking.js';

/** Shorthand: escape any value to a safe HTML text fragment. */
const s = (v) => escapeAttr(String(v ?? ''));

/** Accuracy → semantic value class (good/warn/danger). Numeric-only decision. */
function accuracyClass(accuracy) {
  return accuracy <= 15 ? 'pu-good' : accuracy <= 50 ? 'pu-warn' : 'pu-danger';
}

/** Battery pct → semantic value class. */
function batteryClass(pct) {
  return pct > 50 ? 'pu-good' : pct > 20 ? 'pu-warn' : 'pu-danger';
}

/** Connection quality enum → semantic value class. */
function connectionClass(quality) {
  return quality === 'Good' ? 'pu-good' : quality === 'OK' ? 'pu-warn' : 'pu-danger';
}

/** Human radius string (m / km). Input is a number. */
function formatRadius(radiusM) {
  if (!radiusM) return '';
  return radiusM >= 1000 ? (radiusM / 1000).toFixed(1) + 'km' : radiusM + 'm';
}

/**
 * Build the popup HTML for another family member's marker.
 * @param {object} user   — the tracked user (untrusted string fields)
 * @param {object|null} myLoc — the viewer's own location for distance calc
 * @returns {string} safe HTML
 */
export function buildMemberPopup(user, myLoc) {
  const name = s(user.displayName || 'User');
  const isOnline = user.online !== false;

  let html = `<div class="pu-wrap">`;
  html += `<div class="pu-hdr">`;
  html += `<strong class="pu-name">${name}</strong>`;
  html += `<span class="pu-status ${isOnline ? 'pu-online' : 'pu-offline'}"><span class="pu-dot"></span>${isOnline ? 'Online' : 'Offline'}</span>`;
  html += `</div>`;

  html += `<div class="pu-grid">`;
  const speed = parseFloat(user.speed) >= 1 ? Math.round(parseFloat(user.speed)) : 0;
  html += `<span class="pu-lbl">Speed</span><span class="pu-val tabular-nums">${speed} km/h</span>`;

  if (myLoc && user.latitude != null && user.longitude != null) {
    const dist = calculateDistance(myLoc.latitude, myLoc.longitude, user.latitude, user.longitude);
    const formatted = formatDistance(dist);
    if (formatted) html += `<span class="pu-lbl">Distance</span><span class="pu-val tabular-nums">${s(formatted)}</span>`;
  }
  if (user.accuracy != null) {
    html += `<span class="pu-lbl">Accuracy</span><span class="pu-val tabular-nums ${accuracyClass(user.accuracy)}">~${Math.round(user.accuracy)}m</span>`;
  }
  if (user.formattedTime) {
    html += `<span class="pu-lbl">Updated</span><span class="pu-val">${s(user.formattedTime)}</span>`;
  }
  if (user.batteryPct != null) {
    const glyph = user.batteryPct > 75 ? '🔋' : '🪫';
    html += `<span class="pu-lbl">Battery</span><span class="pu-val tabular-nums ${batteryClass(user.batteryPct)}">${glyph} ${Math.round(user.batteryPct)}%</span>`;
  }
  if (user.deviceType) {
    html += `<span class="pu-lbl">Device</span><span class="pu-val">${s(user.deviceType)}</span>`;
  }
  if (user.connectionQuality && user.connectionQuality !== 'Unknown') {
    html += `<span class="pu-lbl">Signal</span><span class="pu-val ${connectionClass(user.connectionQuality)}">${s(user.connectionQuality)}</span>`;
  }
  if (user.latitude != null && user.longitude != null) {
    html += `<span class="pu-lbl">Position</span><span class="pu-val pu-mono tabular-nums">${Number(user.latitude).toFixed(5)}, ${Number(user.longitude).toFixed(5)}</span>`;
  }
  html += `</div>`;

  const badges = [];
  if (user.sos?.active) {
    const sosReason = user.sos.reason ? ': ' + s(user.sos.reason) : '';
    const sosTime = user.sos.at ? ' at ' + s(new Date(user.sos.at).toLocaleTimeString()) : '';
    badges.push(`<div class="pu-badge pu-badge-sos">⚠ SOS Active${sosReason}${sosTime}</div>`);
  }
  if (user.geofence?.enabled) {
    const r = formatRadius(user.geofence.radiusM);
    badges.push(`<div class="pu-badge pu-badge-geo">⬡ Geofence${r ? ' · ' + s(r) : ''}</div>`);
  }
  if (user.autoSos?.enabled) {
    badges.push(`<div class="pu-badge pu-badge-autoSos">⏱ Auto-SOS · ${s(user.autoSos.noMoveMinutes || '?')}min</div>`);
  }
  if (user.checkIn?.enabled) {
    const lastCI = user.checkIn.lastCheckInAt ? s(new Date(user.checkIn.lastCheckInAt).toLocaleTimeString()) : 'never';
    badges.push(`<div class="pu-badge pu-badge-checkin">✓ Check-in · every ${s(user.checkIn.intervalMinutes || '?')}min · last: ${lastCI}</div>`);
  }
  if (badges.length) html += `<div class="pu-badges">${badges.join('')}</div>`;

  if (user.rooms && user.rooms.length > 0) {
    html += `<div class="pu-rooms"><span class="pu-lbl">Rooms:</span> ${user.rooms.map((r) => s(r)).join(', ')}</div>`;
  }

  if (user.userId) {
    const uid = escapeAttr(user.userId);
    html += `<div class="pu-actions">`;
    html += `<button class="pu-chat-btn" data-userid="${uid}" data-name="${name}"><svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg> Chat</button>`;
    html += `<button class="pu-trail-btn" data-userid="${uid}"><svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg> Trail</button>`;
    html += `</div>`;
  }
  html += `</div>`;
  return html;
}

/**
 * Build the popup HTML for the viewer's own marker ("You").
 * @param {object} loc    — my location { latitude, longitude, speed, formattedTime, accuracy }
 * @param {object|null} safety — mySafetyStatus { geofence, autoSos, checkIn }
 * @returns {string} safe HTML
 */
export function buildSelfPopup(loc, safety) {
  const { latitude, longitude, speed, formattedTime, accuracy } = loc;

  let html = '<div class="pu-wrap">';
  html += '<div class="pu-hdr"><strong class="pu-name">You</strong>';
  html += '<span class="pu-status pu-online"><span class="pu-dot"></span>Connected</span></div>';
  html += '<div class="pu-grid">';
  const spd = speed >= 1 ? Math.round(speed) : 0;
  html += `<span class="pu-lbl">Speed</span><span class="pu-val tabular-nums">${spd} km/h</span>`;
  if (accuracy != null) {
    html += `<span class="pu-lbl">Accuracy</span><span class="pu-val tabular-nums ${accuracyClass(accuracy)}">~${Math.round(accuracy)}m</span>`;
  }
  if (formattedTime) {
    html += `<span class="pu-lbl">Updated</span><span class="pu-val">${s(formattedTime)}</span>`;
  }
  html += `<span class="pu-lbl">Position</span><span class="pu-val pu-mono tabular-nums">${Number(latitude).toFixed(5)}, ${Number(longitude).toFixed(5)}</span>`;
  html += '</div>';

  const feats = [];
  if (safety?.geofence?.enabled) feats.push('<span class="pu-feat pu-feat-geo">⬡ Geofence</span>');
  if (safety?.autoSos?.enabled) feats.push('<span class="pu-feat pu-feat-autoSos">⏱ Auto-SOS</span>');
  if (safety?.checkIn?.enabled) feats.push('<span class="pu-feat pu-feat-checkin">✓ Check-in</span>');
  if (feats.length) html += '<div class="pu-feats">' + feats.join('') + '</div>';

  html += '</div>';
  return html;
}

/**
 * Build a cache hash for a member popup so callers can skip rebuilds when no
 * user-visible field changed. Pure function of the same inputs as
 * buildMemberPopup.
 */
export function memberPopupHash(user, myLoc) {
  return `${user.displayName}|${user.online}|${user.speed}|${user.accuracy}|${user.formattedTime}|${user.batteryPct}|${user.latitude?.toFixed(4)}|${user.longitude?.toFixed(4)}|${user.sos?.active}|${user.geofence?.enabled}|${user.checkIn?.lastCheckInAt}|${myLoc?.latitude?.toFixed(3)}|${myLoc?.longitude?.toFixed(3)}`;
}

/**
 * Create the Google-Maps-style navigation arrow marker element.
 * All-static markup (no user strings); tokenized colors.
 * @returns {HTMLElement}
 */
export function createNavArrowEl() {
  const el = document.createElement('div');
  el.className = 'nav-arrow-marker';
  el.innerHTML = `
    <div class="nav-arrow-inner">
      <div class="nav-arrow-pulse" aria-hidden="true"></div>
      <div class="nav-arrow-core">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2L4.5 20.29l.71.71L12 18l6.79 3 .71-.71z"/></svg>
      </div>
    </div>`;
  return el;
}

/**
 * Create a green flag-style meeting-point marker element.
 * @param {string} label — untrusted room / meeting-point label (escaped)
 * @returns {HTMLElement}
 */
export function createMeetingPointEl(label) {
  const el = document.createElement('div');
  el.className = 'meeting-point-marker';
  el.innerHTML = `
    <div class="mp-flag">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M4 15V5l5 3 3-4 3 4 5-3v10"/><line x1="4" y1="22" x2="4" y2="15" stroke="currentColor" stroke-width="2"/></svg>
    </div>
    <div class="mp-label">${s(label || 'Meet here')}</div>
    <div class="mp-stem" aria-hidden="true"></div>`;
  return el;
}

/**
 * Create the destination pin element used during navigation.
 * All-static markup; tokenized colors.
 * @returns {HTMLElement}
 */
export function createDestMarkerEl() {
  const el = document.createElement('div');
  el.className = 'dest-marker';
  el.innerHTML = `
    <div class="dest-pin">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5A2.5 2.5 0 119.5 9 2.5 2.5 0 0112 11.5z"/></svg>
    </div>
    <div class="dest-stem" aria-hidden="true"></div>`;
  return el;
}
