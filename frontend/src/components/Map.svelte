<script>
  import { onMount, onDestroy } from 'svelte';
  import maplibregl from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';
  import { otherUsers, myLocation, mySocketId, mySafetyStatus, focusUser, mapFlyTo, routeGeometry, navigationState } from '../lib/stores/map.js';
  import { savedPlaces } from '../lib/stores/savedPlaces.js';
  import { arrivalProjections } from '../lib/stores/arrivals.js';
  import { trailData } from '../lib/stores/trail.js';
  import { createMapIcon, createPersonMarker, getPresenceState, escapeAttr, calculateDistance, formatDistance, circleGeoJSON, createCustomPinIcon } from '../lib/tracking.js';
  import { animateMarkerTo, cancelAnimation, cancelAllAnimations } from '../lib/markerInterpolator.js';
  import { getUserColor } from '../lib/getUserColor.js';
  import { MAP_STYLE, RASTER_STYLE } from '../lib/mapStyle.js';
  import { debounce } from '../lib/debounce.js';
  import { emitSyncPlace } from '../lib/socket.js';
  import { apiDelete } from '../lib/api.js';
  import CustomPinDialog from './CustomPinDialog.svelte';

  export let followMode = false;

  let mapContainer;
  let map;
  let markers = new Map();       // sid → maplibregl.Marker
  let markerPopups = new Map();   // sid → maplibregl.Popup
  let markerState = new Map();
  let geofenceIds = new Set();
  let myMarker = null;
  let myPopup = null;
  let hasSetView = false;

  /** Creates a Google Maps-style blue navigation arrow marker */
  function createNavArrow() {
    const el = document.createElement('div');
    el.className = 'nav-arrow-marker';
    el.style.cssText = 'width:48px;height:48px;display:flex;align-items:center;justify-content:center;';
    el.innerHTML = `
      <div style="position:relative;width:48px;height:48px;">
        <div style="position:absolute;inset:0;border-radius:50%;background:rgba(59,130,246,0.15);animation:nav-pulse 2s ease-out infinite;"></div>
        <div style="position:absolute;inset:8px;border-radius:50%;background:#3b82f6;border:3px solid #fff;box-shadow:0 2px 8px rgba(59,130,246,0.5);display:flex;align-items:center;justify-content:center;">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="#fff"><path d="M12 2L4.5 20.29l.71.71L12 18l6.79 3 .71-.71z"/></svg>
        </div>
      </div>`;
    return el;
  }

  // ── Navigation camera state ────────────────────────────────────────
  let prevNavLat = null;
  let prevNavLng = null;
  let currentBearing = 0; // degrees, 0=north, 90=east

  /** Compute bearing in degrees from point A to point B */
  function computeBearing(lat1, lng1, lat2, lng2) {
    const toRad = Math.PI / 180;
    const dLng = (lng2 - lng1) * toRad;
    const y = Math.sin(dLng) * Math.cos(lat2 * toRad);
    const x = Math.cos(lat1 * toRad) * Math.sin(lat2 * toRad) -
              Math.sin(lat1 * toRad) * Math.cos(lat2 * toRad) * Math.cos(dLng);
    return ((Math.atan2(y, x) * 180 / Math.PI) + 360) % 360;
  }
  let isMobile = typeof window !== 'undefined' ? window.innerWidth < 768 : false;
  let renderUsersRaf = null;
  let pendingUsers = new Map();
  const popupCache = new Map();
  const _lastRenderedUsers = new Map();

  // Dirty-only rendering: only update markers that changed
  let dirtyMarkers = new Set();
  let dirtyRafPending = false;

  function markDirty(sid) {
    dirtyMarkers.add(sid);
    if (!dirtyRafPending) {
      dirtyRafPending = true;
      requestAnimationFrame(flushDirty);
    }
  }

  function flushDirty() {
    dirtyRafPending = false;
    if (!map) return;
    for (const sid of dirtyMarkers) {
      const user = pendingUsers.get(sid);
      if (user) updateSingleMarker(sid, user);
    }
    dirtyMarkers.clear();
  }

  function checkMobile() {
    isMobile = typeof window !== 'undefined' && window.innerWidth < 768;
  }

  function ensureCircleSource(id) {
    if (!map.getSource(id)) {
      map.addSource(id, { type: 'geojson', data: { type: 'FeatureCollection', features: [] } });
    }
  }

  function ensureCircleLayer(id, sourceId, color, opacity, outline, outlineWidth, dasharray) {
    if (!map.getLayer(id)) {
      map.addLayer({
        id, type: 'fill', source: sourceId,
        paint: { 'fill-color': color, 'fill-opacity': opacity }
      });
    }
    const outlineId = id + '-outline';
    if (outline && !map.getLayer(outlineId)) {
      const paint = { 'line-color': outline, 'line-width': outlineWidth || 1, 'line-opacity': 0.7 };
      if (dasharray) paint['line-dasharray'] = dasharray;
      map.addLayer({ id: outlineId, type: 'line', source: sourceId, paint });
    }
  }

  function updateCircleSource(sourceId, center, radiusM) {
    const src = map.getSource(sourceId);
    if (!src) return;
    if (radiusM > 0) {
      src.setData(circleGeoJSON(center, radiusM));
    } else {
      src.setData({ type: 'FeatureCollection', features: [] });
    }
  }

  const debouncedCheckMobile = debounce(checkMobile, 80);

  onMount(() => {
    checkMobile();
    window.addEventListener('resize', debouncedCheckMobile);

    // Configure MapLibre GL worker (required for proper rendering)
    maplibregl.workerUrl = '/maplibre-gl-csp-worker.js';
    maplibregl.workerCount = isMobile ? 1 : 2;

    function addCircleSources() {
      ensureCircleSource('my-geofence');
      ensureCircleLayer('my-geofence-fill', 'my-geofence', '#8b5cf6', 0.10, '#8b5cf6', 2.5, [8, 5]);

      // Cluster source for dense groups (only used when zoom < 12)
      map.addSource('users-cluster', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] },
        cluster: true,
        clusterMaxZoom: 12,
        clusterRadius: 48,
      });

      // Cluster circle
      map.addLayer({
        id: 'cluster-circles',
        type: 'circle',
        source: 'users-cluster',
        filter: ['has', 'point_count'],
        paint: {
          'circle-color': [
            'step', ['get', 'point_count'],
            'rgba(59,130,246,0.85)',  3,
            'rgba(139,92,246,0.85)', 7,
            'rgba(239,68,68,0.85)'
          ],
          'circle-radius': [
            'step', ['get', 'point_count'],
            22, 5, 28, 10, 34
          ],
          'circle-stroke-width': 2,
          'circle-stroke-color': 'rgba(255,255,255,0.5)',
        }
      });

      // Cluster count label
      map.addLayer({
        id: 'cluster-count',
        type: 'symbol',
        source: 'users-cluster',
        filter: ['has', 'point_count'],
        layout: {
          'text-field': '{point_count_abbreviated}',
          'text-font': ['Noto Sans Bold'],
          'text-size': 13,
        },
        paint: {
          'text-color': '#ffffff',
        }
      });

      // Click cluster → zoom in
      map.on('click', 'cluster-circles', (e) => {
        const features = map.queryRenderedFeatures(e.point, { layers: ['cluster-circles'] });
        if (!features.length) return;
        const clusterId = features[0].properties.cluster_id;
        map.getSource('users-cluster').getClusterExpansionZoom(clusterId, (err, zoom) => {
          if (err) return;
          map.easeTo({ center: features[0].geometry.coordinates, zoom, duration: 500 });
        });
      });

      map.on('mouseenter', 'cluster-circles', () => { map.getCanvas().style.cursor = 'pointer'; });
      map.on('mouseleave', 'cluster-circles', () => { map.getCanvas().style.cursor = ''; });

      // Click on empty map area → open add-pin dialog
      map.on('click', (e) => {
        // Skip if click landed on a person/pin marker or a cluster
        if (e.originalEvent.target.closest('.maplibregl-marker')) return;
        const features = map.queryRenderedFeatures(e.point, { layers: ['cluster-circles'] });
        if (features.length > 0) return; // hit a cluster, skip
        pendingPin = { lat: e.lngLat.lat, lng: e.lngLat.lng };
        showPinDialog = true;
      });
    }

    map = new maplibregl.Map({
      container: mapContainer,
      style: MAP_STYLE,
      center: [78.9629, 20.5937], // Center of India
      zoom: 4,
      attributionControl: true
    });

    map.addControl(new maplibregl.NavigationControl({ showCompass: false }),
      isMobile ? 'bottom-right' : 'top-right');

    map.on('dragstart', () => {
      // During navigation, let user pan but re-center on next GPS update
      if (!navActive) followMode = false;
    });
    map.on('load', () => {
      mapReady = true;
      // Apply Hindi/regional label preference if configured
      const labelPref = localStorage.getItem('kinnect_map_lang') || 'auto';
      if (labelPref === 'hi' || (labelPref === 'auto' && navigator.language?.startsWith('hi'))) {
        try {
          for (const layer of map.getStyle().layers) {
            if (layer.layout?.['text-field']) {
              map.setLayoutProperty(layer.id, 'text-field', [
                'coalesce', ['get', 'name:hi'], ['get', 'name:en'], ['get', 'name']
              ]);
            }
          }
        } catch { /* style may not have named layers */ }
      }
      addCircleSources();
      // Subscribe to trail data — draw/update dashed polylines for requested users
      trailData.subscribe($trailData => {
        if (!map.isStyleLoaded()) return;
        for (const [userId, data] of $trailData) {
          if (!Array.isArray(data.points) || data.points.length < 2) continue;
          const sourceId = `trail-src-${userId}`;
          const layerId = `trail-line-${userId}`;
          const coords = data.points.map(p => [p.lng, p.lat]);
          const geojson = { type: 'Feature', geometry: { type: 'LineString', coordinates: coords }, properties: {} };
          if (map.getSource(sourceId)) {
            map.getSource(sourceId).setData(geojson);
          } else {
            map.addSource(sourceId, { type: 'geojson', data: geojson });
            map.addLayer({
              id: layerId, type: 'line', source: sourceId,
              paint: { 'line-color': '#818cf8', 'line-width': 2, 'line-opacity': 0.65, 'line-dasharray': [2, 2] }
            });
          }
        }
      });
    });
  });

  onDestroy(() => {
    cancelAllAnimations();
    if (renderUsersRaf) cancelAnimationFrame(renderUsersRaf);
    for (const m of markers.values()) m.remove();
    markers.clear();
    for (const p of markerPopups.values()) p.remove();
    markerPopups.clear();
    for (const m of pinMarkers.values()) m.remove();
    pinMarkers.clear();
    for (const p of pinPopups.values()) p.remove();
    pinPopups.clear();
    if (myMarker) myMarker.remove();
    if (myPopup) myPopup.remove();
    if (map) map.remove();
    if (typeof window !== 'undefined') window.removeEventListener('resize', debouncedCheckMobile);
  });

  $: if (map && $myLocation) {
    const { latitude, longitude, speed, formattedTime, accuracy } = $myLocation;
    const lngLat = [longitude, latitude];
    const selfBadges = [];
    if ($mySafetyStatus?.geofence?.enabled) selfBadges.push('<span class="pu-feat pu-feat-geo">⬡ Geofence</span>');
    if ($mySafetyStatus?.autoSos?.enabled) selfBadges.push('<span class="pu-feat pu-feat-autoSos">⏱ Auto-SOS</span>');
    if ($mySafetyStatus?.checkIn?.enabled) selfBadges.push('<span class="pu-feat pu-feat-checkin">✓ Check-in</span>');
    const accCls0 = accuracy == null ? '' : accuracy <= 15 ? 'pu-good' : accuracy <= 50 ? 'pu-warn' : 'pu-danger';
    let selfPopupHtml = '<div class="pu-wrap">';
    selfPopupHtml += '<div class="pu-hdr"><strong class="pu-name">You</strong>';
    selfPopupHtml += '<span class="pu-status pu-online"><span class="pu-dot"></span>Connected</span></div>';
    selfPopupHtml += '<div class="pu-grid">';
    selfPopupHtml += `<span class="pu-lbl">Speed</span><span class="pu-val">${speed >= 1 ? speed : 0} km/h</span>`;
    if (accuracy != null) selfPopupHtml += `<span class="pu-lbl">Accuracy</span><span class="pu-val ${accCls0}">~${Math.round(accuracy)}m</span>`;
    if (formattedTime) selfPopupHtml += `<span class="pu-lbl">Updated</span><span class="pu-val">${escapeAttr(String(formattedTime))}</span>`;
    selfPopupHtml += `<span class="pu-lbl">Position</span><span class="pu-val pu-mono">${Number(latitude).toFixed(5)}, ${Number(longitude).toFixed(5)}</span>`;
    selfPopupHtml += '</div>';
    if (selfBadges.length) selfPopupHtml += '<div class="pu-feats">' + selfBadges.join('') + '</div>';
    selfPopupHtml += '</div>';

    if (!myMarker) {
      const el = navActive ? createNavArrow() : createMapIcon('var(--primary-500)', '', { markerType: 'self' });
      myPopup = new maplibregl.Popup({ offset: [0, -44], maxWidth: '280px', closeButton: false })
        .setHTML(selfPopupHtml);
      myMarker = new maplibregl.Marker({ element: el, anchor: 'center', rotationAlignment: 'map' })
        .setLngLat(lngLat)
        .setPopup(myPopup)
        .addTo(map);
    } else {
      animateMarkerTo('__self__', myMarker, lngLat);
      myPopup.setHTML(selfPopupHtml);
      // Swap marker style when entering/exiting nav mode
      const el = myMarker.getElement();
      if (navActive && !el.classList.contains('nav-arrow-marker')) {
        const newEl = createNavArrow();
        myMarker.remove();
        myMarker = new maplibregl.Marker({ element: newEl, anchor: 'center', rotationAlignment: 'map' })
          .setLngLat(lngLat).setPopup(myPopup).addTo(map);
      } else if (!navActive && el.classList.contains('nav-arrow-marker')) {
        const newEl = createMapIcon('var(--primary-500)', '', { markerType: 'self' });
        myMarker.remove();
        myMarker = new maplibregl.Marker({ element: newEl, anchor: 'bottom' })
          .setLngLat(lngLat).setPopup(myPopup).addTo(map);
      }
    }

    if (followMode && navActive) {
      // ── Google Maps-style 3D navigation camera ──────────────────
      // Compute bearing from previous position (need >5m movement to avoid jitter)
      if (prevNavLat != null && prevNavLng != null) {
        const dLat = latitude - prevNavLat;
        const dLng = longitude - prevNavLng;
        const movedM = Math.sqrt(dLat * dLat + dLng * dLng) * 111320;
        if (movedM > 5) {
          currentBearing = computeBearing(prevNavLat, prevNavLng, latitude, longitude);
          prevNavLat = latitude;
          prevNavLng = longitude;
        }
      } else {
        prevNavLat = latitude;
        prevNavLng = longitude;
      }

      // Tilt + rotate + follow from behind (user at bottom 1/3)
      map.easeTo({
        center: lngLat,
        zoom: 17.5,
        bearing: currentBearing,
        pitch: 55,  // 3D tilt
        duration: 600,
        padding: { top: 200, bottom: 40, left: 0, right: 0 },
      });
    } else if (followMode) {
      map.easeTo({ center: lngLat, zoom: Math.max(map.getZoom(), 15), duration: 600 });
    } else if (!hasSetView) {
      map.jumpTo({ center: lngLat, zoom: 15 });
    }
    hasSetView = true;
  }

  $: if (map && map.loaded()) {
    const gf = $mySafetyStatus?.geofence;
    if (gf?.enabled && gf.centerLat != null && gf.centerLng != null && gf.radiusM > 0) {
      updateCircleSource('my-geofence', [gf.centerLng, gf.centerLat], gf.radiusM);
    } else {
      updateCircleSource('my-geofence', [0, 0], 0);
    }
  }

  function buildPopup(user) {
    const name = escapeAttr(user.displayName || 'User');
    const s = (v) => escapeAttr(String(v ?? ''));
    const isOnline = user.online !== false;

    let html = `<div class="pu-wrap">`;
    html += `<div class="pu-hdr">`;
    html += `<strong class="pu-name">${name}</strong>`;
    html += `<span class="pu-status ${isOnline ? 'pu-online' : 'pu-offline'}"><span class="pu-dot"></span>${isOnline ? 'Online' : 'Offline'}</span>`;
    html += `</div>`;

    html += `<div class="pu-grid">`;
    html += `<span class="pu-lbl">Speed</span><span class="pu-val">${parseFloat(user.speed) >= 1 ? user.speed : 0} km/h</span>`;

    const myLoc = $myLocation;
    if (myLoc && user.latitude != null && user.longitude != null) {
      const dist = calculateDistance(myLoc.latitude, myLoc.longitude, user.latitude, user.longitude);
      const formatted = formatDistance(dist);
      if (formatted) html += `<span class="pu-lbl">Distance</span><span class="pu-val">${formatted}</span>`;
    }
    if (user.accuracy != null) {
      const accCls = user.accuracy <= 15 ? 'pu-good' : user.accuracy <= 50 ? 'pu-warn' : 'pu-danger';
      html += `<span class="pu-lbl">Accuracy</span><span class="pu-val ${accCls}">~${Math.round(user.accuracy)}m</span>`;
    }
    if (user.formattedTime) html += `<span class="pu-lbl">Updated</span><span class="pu-val">${s(user.formattedTime)}</span>`;
    if (user.batteryPct != null) {
      const batCls = user.batteryPct > 50 ? 'pu-good' : user.batteryPct > 20 ? 'pu-warn' : 'pu-danger';
      html += `<span class="pu-lbl">Battery</span><span class="pu-val ${batCls}">${user.batteryPct > 75 ? '🔋' : '🪫'} ${user.batteryPct}%</span>`;
    }
    if (user.deviceType) html += `<span class="pu-lbl">Device</span><span class="pu-val">${s(user.deviceType)}</span>`;
    if (user.connectionQuality && user.connectionQuality !== 'Unknown') {
      const cqCls = user.connectionQuality === 'Good' ? 'pu-good' : user.connectionQuality === 'OK' ? 'pu-warn' : 'pu-danger';
      html += `<span class="pu-lbl">Signal</span><span class="pu-val ${cqCls}">${s(user.connectionQuality)}</span>`;
    }
    if (user.latitude != null && user.longitude != null) {
      html += `<span class="pu-lbl">Position</span><span class="pu-val pu-mono">${Number(user.latitude).toFixed(5)}, ${Number(user.longitude).toFixed(5)}</span>`;
    }
    html += `</div>`;

    const badges = [];
    if (user.sos?.active) {
      const sosReason = user.sos.reason ? ': ' + s(user.sos.reason) : '';
      const sosTime = user.sos.at ? ' at ' + new Date(user.sos.at).toLocaleTimeString() : '';
      badges.push(`<div class="pu-badge pu-badge-sos">⚠ SOS Active${sosReason}${sosTime}</div>`);
    }
    if (user.geofence?.enabled) {
      const r = user.geofence.radiusM ? (user.geofence.radiusM >= 1000 ? (user.geofence.radiusM / 1000).toFixed(1) + 'km' : user.geofence.radiusM + 'm') : '';
      badges.push(`<div class="pu-badge pu-badge-geo">⬡ Geofence${r ? ' · ' + r : ''}</div>`);
    }
    if (user.autoSos?.enabled) badges.push(`<div class="pu-badge pu-badge-autoSos">⏱ Auto-SOS · ${user.autoSos.noMoveMinutes || '?'}min</div>`);
    if (user.checkIn?.enabled) {
      const lastCI = user.checkIn.lastCheckInAt ? new Date(user.checkIn.lastCheckInAt).toLocaleTimeString() : 'never';
      badges.push(`<div class="pu-badge pu-badge-checkin">✓ Check-in · every ${user.checkIn.intervalMinutes || '?'}min · last: ${lastCI}</div>`);
    }
    if (badges.length) html += `<div class="pu-badges">${badges.join('')}</div>`;
    if (user.rooms && user.rooms.length > 0) html += `<div class="pu-rooms"><span class="pu-lbl">Rooms:</span> ${user.rooms.map(r => s(r)).join(', ')}</div>`;
    html += `</div>`;
    return html;
  }

  function buildPopupCached(user) {
    const ml = $myLocation;
    const hash = `${user.displayName}|${user.online}|${user.speed}|${user.accuracy}|${user.formattedTime}|${user.batteryPct}|${user.latitude?.toFixed(4)}|${user.longitude?.toFixed(4)}|${user.sos?.active}|${user.geofence?.enabled}|${user.checkIn?.lastCheckInAt}|${ml?.latitude?.toFixed(3)}|${ml?.longitude?.toFixed(3)}`;
    const cached = popupCache.get(user.socketId);
    if (cached && cached.hash === hash) return cached.html;
    const html = buildPopup(user);
    popupCache.set(user.socketId, { hash, html });
    return html;
  }

  // renderUserMarkers is replaced by updateSingleMarker + dirty tracking above.

  function wrapWithRing(iconEl, user, color) {
    const isLive = user.online !== false && !user.sos?.active;
    const isOffline = user.online === false;
    if (isOffline) {
      iconEl.style.filter = 'grayscale(0.7)';
      iconEl.style.opacity = '0.7';
    }
    if (!isLive) return iconEl;
    const wrapper = document.createElement('div');
    wrapper.className = 'marker-wrapper';
    const ring = document.createElement('div');
    ring.className = 'marker-ring animate-pulse-ring';
    ring.style.setProperty('--user-color', color);
    wrapper.appendChild(ring);
    wrapper.appendChild(iconEl);
    return wrapper;
  }

  function updateSingleMarker(sid, user) {
    if (!map || user.latitude == null || user.longitude == null) return;
    const lngLat = [user.longitude, user.latitude];
    const color = getUserColor(user.userId);
    const isSos = !!user.sos?.active;
    const presenceState = getPresenceState(user);
    // Include presenceState + motionClass in key so rings/badges re-render when state changes
    const mClass = user.motionClass || '';
    const qhActive = !!user.quietHoursActive;
    const iconKey = `person|${color}|${presenceState}|${isSos}|${mClass}|${qhActive}`;
    const popupContent = buildPopupCached(user);

    function makePersonEl() {
      return createPersonMarker({
        displayName: user.displayName,
        userId: user.userId,
        color: isSos ? '#ef4444' : (user.online === false ? '#6b7280' : color),
        isSelf: false,
        isSos,
        presenceState,
        motionClass: mClass,
        quietHoursActive: qhActive,
      });
    }

    if (markers.has(sid)) {
      const m = markers.get(sid);
      animateMarkerTo(sid, m, lngLat);
      if (markerState.get(sid) !== iconKey) {
        const el = makePersonEl();
        const newMarker = new maplibregl.Marker({ element: el, anchor: 'bottom' })
          .setLngLat(m.getLngLat())
          .addTo(map);
        const popup = markerPopups.get(sid);
        if (popup) { popup.setHTML(popupContent); newMarker.setPopup(popup); }
        m.remove();
        markers.set(sid, newMarker);
        markerState.set(sid, iconKey);
      } else {
        const popup = markerPopups.get(sid);
        if (popup) popup.setHTML(popupContent);
      }
    } else {
      const el = makePersonEl();
      const popup = new maplibregl.Popup({ offset: [0, -54], maxWidth: '280px', closeButton: true })
        .setHTML(popupContent);
      const m = new maplibregl.Marker({ element: el, anchor: 'bottom' })
        .setLngLat(lngLat)
        .setPopup(popup)
        .addTo(map);
      markers.set(sid, m);
      markerPopups.set(sid, popup);
      markerState.set(sid, iconKey);
    }
  }

  $: if (map) {
    const current = $otherUsers;
    pendingUsers = current;

    // Remove markers for users no longer present
    for (const [sid, m] of markers) {
      if (!current.has(sid)) {
        cancelAnimation(sid);
        m.remove();
        markers.delete(sid);
        markerState.delete(sid);
        popupCache.delete(sid);
        _lastRenderedUsers.delete(sid);
        if (markerPopups.has(sid)) { markerPopups.get(sid).remove(); markerPopups.delete(sid); }
        dirtyMarkers.delete(sid);
      }
    }

    // Clean stale geofence sources
    for (const sid of geofenceIds) {
      if (!current.has(sid)) {
        const srcId = 'gf-' + sid;
        if (map.getLayer(srcId + '-fill')) map.removeLayer(srcId + '-fill');
        if (map.getLayer(srcId + '-outline')) map.removeLayer(srcId + '-outline');
        if (map.getSource(srcId)) map.removeSource(srcId);
        geofenceIds.delete(sid);
      }
    }

    // Mark only changed users as dirty
    for (const [sid, user] of current) {
      if (user.latitude == null || user.longitude == null) continue;
      if (_lastRenderedUsers.get(sid) !== user) {
        _lastRenderedUsers.set(sid, user);
        markDirty(sid);
      }
    }

    // Handle geofences (still rendered via renderUserMarkers for simplicity)
    if (renderUsersRaf) cancelAnimationFrame(renderUsersRaf);
    renderUsersRaf = requestAnimationFrame(() => {
      renderUsersRaf = null;
      // Only handle geofences in the full render pass
      for (const [sid, user] of current) {
        if (!map.loaded()) continue;
        const gf = user.geofence;
        const srcId = 'gf-' + sid;
        if (gf?.enabled && gf.centerLat != null && gf.centerLng != null && gf.radiusM > 0) {
          ensureCircleSource(srcId);
          ensureCircleLayer(srcId + '-fill', srcId, '#8b5cf6', 0.08, '#8b5cf6', 2, [6, 4]);
          updateCircleSource(srcId, [gf.centerLng, gf.centerLat], gf.radiusM);
          geofenceIds.add(sid);
        } else if (geofenceIds.has(sid)) {
          if (map.getLayer(srcId + '-fill')) map.removeLayer(srcId + '-fill');
          if (map.getLayer(srcId + '-outline')) map.removeLayer(srcId + '-outline');
          if (map.getSource(srcId)) map.removeSource(srcId);
          geofenceIds.delete(sid);
        }
      }
    });
  }

  // Update cluster GeoJSON whenever user positions change
  $: if (map && map.getSource('users-cluster')) {
    const features = [];
    for (const user of $otherUsers.values()) {
      if (user.latitude == null || user.longitude == null || user.online === false) continue;
      features.push({
        type: 'Feature',
        geometry: { type: 'Point', coordinates: [user.longitude, user.latitude] },
        properties: { name: user.displayName || 'User', sos: !!user.sos?.active },
      });
    }
    map.getSource('users-cluster').setData({ type: 'FeatureCollection', features });
  }

  // ── Place search fly-to ──────────────────────────────────────────────
  $: if (map && $mapFlyTo) {
    const target = $mapFlyTo;
    mapFlyTo.set(null);
    map.flyTo({ center: [target.lng, target.lat], zoom: target.zoom || 16, duration: 900 });
  }

  // ── Route polyline from directions ─────────────────────────────────
  $: if (map) {
    const geo = $routeGeometry;
    if (geo && geo.coordinates?.length) {
      const srcId = 'directions-route';
      const layerId = 'directions-route-line';
      const glowId = 'directions-route-glow';
      const geojson = { type: 'Feature', geometry: geo, properties: {} };
      if (map.getSource(srcId)) {
        map.getSource(srcId).setData(geojson);
      } else {
        map.addSource(srcId, { type: 'geojson', data: geojson });
        map.addLayer({ id: glowId, type: 'line', source: srcId,
          paint: { 'line-color': '#6366f1', 'line-width': 8, 'line-opacity': 0.18, 'line-blur': 3 } });
        map.addLayer({ id: layerId, type: 'line', source: srcId,
          paint: { 'line-color': '#818cf8', 'line-width': 4, 'line-opacity': 0.85 },
          layout: { 'line-cap': 'round', 'line-join': 'round' } });
      }
    } else {
      // Clear route
      if (map.getLayer('directions-route-line')) map.removeLayer('directions-route-line');
      if (map.getLayer('directions-route-glow')) map.removeLayer('directions-route-glow');
      if (map.getSource('directions-route')) map.removeSource('directions-route');
    }
  }

  // ── Navigation mode — follow user + destination marker + fit bounds ──
  let destMarker = null;
  let navActive = false;

  // ── Custom location pins ────────────────────────────────────────────────────
  let pinMarkers = new Map();        // placeId → maplibregl.Marker
  let pinPopups = new Map();         // placeId → maplibregl.Popup
  let showPinDialog = false;
  let pendingPin = null;             // { lat, lng } awaiting dialog
  let pendingPinMarker = null;       // maplibregl.Marker — preview dot while dialog is open
  let mapReady = false;

  $: if (map) {
    const nav = $navigationState;
    if (nav?.active && nav.routeCoords?.length) {
      navActive = true;
      followMode = true;

      // Compute initial bearing: user → destination
      if ($myLocation?.latitude) {
        currentBearing = computeBearing(
          $myLocation.latitude, $myLocation.longitude,
          nav.destLat, nav.destLng
        );
        prevNavLat = $myLocation.latitude;
        prevNavLng = $myLocation.longitude;
      }

      // Add destination marker (red pin with flag)
      if (destMarker) destMarker.remove();
      const destEl = document.createElement('div');
      destEl.innerHTML = `<div style="width:36px;height:36px;border-radius:50%;background:linear-gradient(135deg,#ef4444,#dc2626);border:3px solid #fff;box-shadow:0 2px 16px rgba(239,68,68,0.5);display:flex;align-items:center;justify-content:center;">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="#fff"><path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5A2.5 2.5 0 119.5 9 2.5 2.5 0 0112 11.5z"/></svg>
      </div><div style="width:2px;height:10px;background:#ef4444;margin:0 auto;border-radius:0 0 2px 2px;opacity:0.6;"></div>`;
      destEl.style.cssText = 'display:flex;flex-direction:column;align-items:center;';
      destMarker = new maplibregl.Marker({ element: destEl, anchor: 'bottom' })
        .setLngLat([nav.destLng, nav.destLat])
        .addTo(map);

      // Step 1: Fit to route overview (2D, north-up)
      const coords = nav.routeCoords;
      const bounds = coords.reduce((b, c) => b.extend(c), new maplibregl.LngLatBounds(coords[0], coords[0]));
      map.fitBounds(bounds, { padding: { top: 100, bottom: 100, left: 50, right: 50 }, duration: 800, pitch: 0, bearing: 0 });

      // Step 2: After overview, swoop into 3D navigation view
      setTimeout(() => {
        if (!$myLocation?.latitude) return;
        map.easeTo({
          center: [$myLocation.longitude, $myLocation.latitude],
          zoom: 17.5,
          bearing: currentBearing,
          pitch: 55,
          duration: 1000,
          padding: { top: 200, bottom: 40, left: 0, right: 0 },
        });
      }, 1400);

    } else if (navActive) {
      // ── Navigation ended — reset camera to 2D north-up ──────────
      navActive = false;
      followMode = false;
      prevNavLat = null;
      prevNavLng = null;
      if (destMarker) { destMarker.remove(); destMarker = null; }
      // Smoothly return to flat north-up view
      map.easeTo({ pitch: 0, bearing: 0, duration: 600 });
    }
  }

  // ── Render saved place pins from store ──────────────────────────────────────
  $: if (map && mapReady && $savedPlaces) {
    const places = $savedPlaces;

    // Remove markers for deleted/missing pins
    for (const [id, marker] of pinMarkers) {
      if (!places.has(id)) {
        marker.remove();
        pinMarkers.delete(id);
        const popup = pinPopups.get(id);
        if (popup) { popup.remove(); pinPopups.delete(id); }
      }
    }

    // Add markers for new pins
    for (const [id, pin] of places) {
      if (pinMarkers.has(id)) continue;
      if (pin.latitude == null || pin.longitude == null) continue;

      const el = createCustomPinIcon(pin.icon || 'pin', pin.name);

      const visLabel = pin.visibility === 'universal' ? '👨‍👩‍👧 Family' : '🔒 Personal';
      const popupHtml = `<div class="pu-wrap">
        <div class="pu-hdr"><strong class="pu-name">${escapeAttr(pin.name)}</strong></div>
        <div class="pu-grid">
          <span class="pu-lbl">Visibility</span><span class="pu-val">${visLabel}</span>
          <span class="pu-lbl">Position</span>
          <span class="pu-val pu-mono">${Number(pin.latitude).toFixed(5)}, ${Number(pin.longitude).toFixed(5)}</span>
        </div>
        <div style="margin-top:8px;text-align:right;">
          <button class="btn btn-sm btn-danger" data-pin-delete="${id}" style="font-size:11px;padding:4px 10px;">Remove</button>
        </div>
      </div>`;

      const popup = new maplibregl.Popup({ offset: [0, -50], maxWidth: '220px', closeButton: true })
        .setHTML(popupHtml);

      const marker = new maplibregl.Marker({ element: el, anchor: 'bottom' })
        .setLngLat([pin.longitude, pin.latitude])
        .setPopup(popup)
        .addTo(map);

      // Delegate delete-button clicks from inside the popup
      popup.on('open', () => {
        const btn = popup.getElement()?.querySelector(`[data-pin-delete="${id}"]`);
        if (btn) {
          btn.onclick = async () => {
            popup.remove();
            await apiDelete(`/api/places/${id}`);
            const wasUniversal = pin.visibility === 'universal';
            savedPlaces.update(m => { m.delete(id); return m; });
            if (wasUniversal) emitSyncPlace('remove', { id, visibility: 'universal' });
          };
        }
      });

      pinMarkers.set(id, marker);
      pinPopups.set(id, popup);
    }
  }

  // ── Preview pin — visible marker while add-location dialog is open ───────
  $: if (map) {
    if (pendingPin) {
      // Remove any old preview marker
      if (pendingPinMarker) { pendingPinMarker.remove(); pendingPinMarker = null; }

      // Build the preview element
      const el = document.createElement('div');
      el.className = 'pending-pin-marker';
      el.innerHTML = `
        <div class="pending-pin-pulse"></div>
        <div class="pending-pin-dot">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="white" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5A2.5 2.5 0 119.5 9 2.5 2.5 0 0112 11.5z"/>
          </svg>
        </div>
        <div class="pending-pin-stem"></div>
      `;
      pendingPinMarker = new maplibregl.Marker({ element: el, anchor: 'bottom' })
        .setLngLat([pendingPin.lng, pendingPin.lat])
        .addTo(map);
    } else {
      if (pendingPinMarker) { pendingPinMarker.remove(); pendingPinMarker = null; }
    }
  }

  $: if (map && $focusUser) {
    const sid = $focusUser;
    focusUser.set(null);

    if (sid === '__self__' && myMarker && $myLocation) {
      map.flyTo({ center: [$myLocation.longitude, $myLocation.latitude], zoom: 17, duration: 800 });
      setTimeout(() => myMarker.togglePopup(), 900);
    } else if (markers.has(sid)) {
      const m = markers.get(sid);
      const ll = m.getLngLat();
      map.flyTo({ center: [ll.lng, ll.lat], zoom: 17, duration: 800 });
      setTimeout(() => m.togglePopup(), 900);
    } else {
      for (const [mSid, user] of $otherUsers) {
        if (user.userId === sid && markers.has(mSid)) {
          const m = markers.get(mSid);
          map.flyTo({ center: m.getLngLat(), zoom: 17, duration: 800 });
          setTimeout(() => m.togglePopup(), 900);
          break;
        }
      }
    }
  }

</script>

<div class="map-container" bind:this={mapContainer}></div>

{#if showPinDialog && pendingPin}
  <CustomPinDialog
    lat={pendingPin.lat}
    lng={pendingPin.lng}
    onClose={() => { showPinDialog = false; pendingPin = null; }}
  />
{/if}
<!-- MERIDIAN: Map vignette — connects UI chrome to map visually -->
<div class="map-vignette-top" aria-hidden="true"></div>
<div class="map-vignette-bottom" aria-hidden="true"></div>

<!-- MERIDIAN SPATIAL: Contextual float chips — accuracy + speed -->
{#if $myLocation}
  {#if $myLocation.accuracy != null}
    {@const acc = $myLocation.accuracy}
    <div class="map-float-chip accuracy-float-chip"
         class:chip-precise={acc <= 20}
         class:chip-ok={acc > 20 && acc <= 80}
         class:chip-rough={acc > 80}
         aria-label="GPS accuracy {Math.round(acc)}m"
         style="top: {isMobile ? 'calc(var(--safe-top, 0px) + 116px)' : 'var(--space-3)'}; right: var(--space-3);">
      <span class="chip-dot" aria-hidden="true"></span>
      {acc <= 20 ? `±${Math.round(acc)}m` : acc <= 80 ? `~${Math.round(acc)}m` : 'Rough GPS'}
    </div>
  {/if}
  {#if $myLocation.speed != null && $myLocation.speed >= 4}
    <div class="map-float-chip chip-speed"
         aria-label="Speed {Math.round($myLocation.speed)} km/h"
         style="top: {isMobile ? 'calc(var(--safe-top, 0px) + 152px)' : 'calc(var(--space-3) + 32px)'}; right: var(--space-3);">
      <span class="chip-dot" aria-hidden="true"></span>
      {Math.round($myLocation.speed)} km/h
    </div>
  {/if}
{/if}

{#if $mySafetyStatus?.geofence?.enabled || $mySafetyStatus?.autoSos?.enabled || $mySafetyStatus?.checkIn?.enabled}
  <div class="safety-overlay" role="status" aria-label="Active safety features">
    {#if $mySafetyStatus.geofence.enabled}
      <div class="safety-chip geofence">
        <span class="safety-icon">⬡</span>
        <span class="safety-label">Geofence</span>
        {#if $mySafetyStatus.geofence.radiusM}
          <span class="safety-detail">{$mySafetyStatus.geofence.radiusM >= 1000 ? ($mySafetyStatus.geofence.radiusM / 1000).toFixed(1) + 'km' : $mySafetyStatus.geofence.radiusM + 'm'}</span>
        {/if}
      </div>
    {/if}
    {#if $mySafetyStatus.autoSos.enabled}
      <div class="safety-chip autosos">
        <span class="safety-icon">⏱</span>
        <span class="safety-label">Auto-SOS</span>
        {#if $mySafetyStatus.autoSos.noMoveMinutes}
          <span class="safety-detail">{$mySafetyStatus.autoSos.noMoveMinutes}min</span>
        {/if}
      </div>
    {/if}
    {#if $mySafetyStatus.checkIn.enabled}
      <div class="safety-chip checkin">
        <span class="safety-icon">✓</span>
        <span class="safety-label">Check-in</span>
        {#if $mySafetyStatus.checkIn.intervalMinutes}
          <span class="safety-detail">every {$mySafetyStatus.checkIn.intervalMinutes}min</span>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<!-- Arrival Intelligence — ETA chips for contacts heading to saved places -->
{#if $arrivalProjections.size > 0}
  <div class="arrival-chips-container" role="status" aria-label="Arrival projections">
    {#each [...$arrivalProjections.values()] as proj (proj.userId)}
      <div class="arrival-chip">
        <span class="arrival-chip-icon">📍</span>
        <span class="arrival-chip-body">
          <span class="arrival-name">{proj.displayName}</span>
          <span class="arrival-eta">~{proj.etaSeconds < 60 ? 'arriving' : `${Math.round(proj.etaSeconds / 60)}min`} to {proj.placeName}</span>
        </span>
      </div>
    {/each}
  </div>
{/if}

<style>
  .map-container {
    position: absolute;
    inset: 0;
    z-index: 1;
  }

  /* ── Pending pin preview marker ──────────────────────────────── */
  :global(.pending-pin-marker) {
    display: flex;
    flex-direction: column;
    align-items: center;
    pointer-events: none;
  }
  :global(.pending-pin-dot) {
    width: 40px;
    height: 40px;
    border-radius: 50% 50% 50% 4px;
    transform: rotate(-45deg);
    background: linear-gradient(135deg, #818cf8, #6366f1);
    border: 2.5px solid #fff;
    box-shadow: 0 4px 18px rgba(99,102,241,0.55), 0 2px 6px rgba(0,0,0,0.25);
    display: flex;
    align-items: center;
    justify-content: center;
    animation: pending-pin-drop 0.35s cubic-bezier(0.34, 1.56, 0.64, 1) both;
  }
  :global(.pending-pin-dot svg) {
    transform: rotate(45deg);
  }
  :global(.pending-pin-stem) {
    width: 2px;
    height: 8px;
    background: rgba(99,102,241,0.5);
    border-radius: 0 0 2px 2px;
    margin-top: -1px;
  }
  :global(.pending-pin-pulse) {
    position: absolute;
    bottom: 8px;
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background: rgba(99,102,241,0.2);
    animation: pending-pin-pulse 1.4s ease-out infinite;
    pointer-events: none;
  }
  @keyframes -global-pending-pin-drop {
    from { transform: rotate(-45deg) translateY(-20px) scale(0.7); opacity: 0; }
    to   { transform: rotate(-45deg) translateY(0)     scale(1);   opacity: 1; }
  }
  @keyframes -global-pending-pin-pulse {
    0%   { transform: scale(0.6); opacity: 0.7; }
    100% { transform: scale(1.8); opacity: 0; }
  }

  /* MERIDIAN: Gradient vignettes blend the map into the UI chrome */
  .map-vignette-top {
    position: absolute;
    top: 0; left: 0; right: 0;
    height: 80px;
    z-index: 2;
    pointer-events: none;
    background: linear-gradient(to bottom, var(--surface-0) 0%, transparent 100%);
    opacity: 0.32;
  }
  .map-vignette-bottom {
    position: absolute;
    bottom: 0; left: 0; right: 0;
    height: 100px;
    z-index: 2;
    pointer-events: none;
    background: linear-gradient(to top, var(--surface-0) 0%, transparent 100%);
    opacity: 0.28;
  }

  :global(.map-pin) {
    cursor: pointer;
    filter: drop-shadow(0 3px 6px rgba(0, 0, 0, 0.32));
    transition: opacity 0.2s ease, filter 0.2s ease, transform 150ms cubic-bezier(0.34, 1.56, 0.64, 1);
    pointer-events: auto;
    overflow: visible;
  }
  :global(.map-pin:hover) {
    filter: drop-shadow(0 5px 12px rgba(0, 0, 0, 0.40));
    transform: scale(1.08);
  }
  :global(.map-pin svg) {
    display: block;
  }

  /* MERIDIAN: Self-marker — indigo ripple ring */
  :global(.map-pin.pin-self::after) {
    content: '';
    position: absolute;
    bottom: 0;
    left: 50%;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--primary-500);
    opacity: 0;
    transform: translate(-50%, 50%);
    animation: pin-ripple 2.2s cubic-bezier(0.4, 0, 0.2, 1) infinite;
    pointer-events: none;
    z-index: -1;
  }

  /* MERIDIAN: SOS markers — double ring radar sweep */
  :global(.map-pin.pin-sos::after) {
    content: '';
    position: absolute;
    bottom: 0;
    left: 50%;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: #ef4444;
    opacity: 0;
    transform: translate(-50%, 50%);
    animation: pin-ripple-sos 1.2s cubic-bezier(0.4, 0, 0.2, 1) infinite;
    pointer-events: none;
    z-index: -1;
  }

  /* Second SOS ring — staggered for radar effect */
  :global(.map-pin.pin-sos::before) {
    content: '';
    position: absolute;
    bottom: 0;
    left: 50%;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: #ef4444;
    opacity: 0;
    transform: translate(-50%, 50%);
    animation: pin-ripple-sos 1.2s cubic-bezier(0.4, 0, 0.2, 1) 0.4s infinite;
    pointer-events: none;
    z-index: -1;
  }

  :global(.map-pin.pin-offline) {
    opacity: 0.5;
  }
  :global(.map-pin.pin-stored) {
    opacity: 0.3;
  }

  @keyframes pin-ripple {
    0%   { opacity: 0.45; transform: translate(-50%, 50%) scale(1); }
    100% { opacity: 0;    transform: translate(-50%, 50%) scale(5); }
  }
  @keyframes pin-ripple-sos {
    0%   { opacity: 0.55; transform: translate(-50%, 50%) scale(1); }
    100% { opacity: 0;    transform: translate(-50%, 50%) scale(6); }
  }

  :global(.maplibregl-popup-content) {
    background: rgba(255, 255, 255, 0.96);
    color: #1e293b;
    border-radius: var(--radius-xl, 20px);
    box-shadow:
      0 12px 40px rgba(0, 0, 0, 0.22),
      0 0 0 1px rgba(0, 0, 0, 0.06),
      inset 0 1px 0 rgba(255, 255, 255, 0.70);
    padding: 14px 16px;
    line-height: 1.5;
    font-family: var(--font-sans, 'Inter', sans-serif);
    backdrop-filter: blur(28px) saturate(1.8);
    -webkit-backdrop-filter: blur(28px) saturate(1.8);
  }
  :global([data-theme="dark"] .maplibregl-popup-content) {
    background: rgba(12, 12, 24, 0.94);
    color: rgba(255, 255, 255, 0.90);
    box-shadow:
      0 12px 40px rgba(0, 0, 0, 0.55),
      0 0 0 1px rgba(255, 255, 255, 0.08),
      inset 0 1px 0 rgba(255, 255, 255, 0.06);
  }
  :global(.maplibregl-popup-tip) {
    border-top-color: white;
  }
  :global([data-theme="dark"] .maplibregl-popup-tip) {
    border-top-color: rgba(20, 25, 40, 0.92);
  }
  :global(.maplibregl-popup-close-button) {
    font-size: 18px;
    color: var(--gray-400);
    padding: 4px 8px;
    border-radius: 6px;
    transition: color 0.15s ease, background 0.15s ease;
  }
  :global(.maplibregl-popup-close-button:hover) {
    color: var(--gray-700);
    background: rgba(0,0,0,0.06);
  }

  /* ── Popup content classes (light + dark mode aware) ─────────────────── */
  :global(.pu-wrap)  { min-width: 190px; font-size: 12px; line-height: 1.5; }
  :global(.pu-hdr)   { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
  :global(.pu-name)  { font-family: var(--font-display); font-size: 15px; font-weight: 700; letter-spacing: -0.01em; color: #0f172a; }
  :global(.pu-status) { display: inline-flex; align-items: center; gap: 3px; font-size: 10px; font-weight: 600; }
  :global(.pu-dot)   { width: 7px; height: 7px; border-radius: 50%; background: currentColor; display: inline-block; }
  :global(.pu-online)  { color: #22c55e; }
  :global(.pu-offline) { color: #9ca3af; }
  :global(.pu-grid)  { display: grid; grid-template-columns: auto 1fr; gap: 2px 10px; font-size: 11px; }
  :global(.pu-lbl)   { font-weight: 600; color: #64748b; }
  :global(.pu-val)   { color: #1e293b; }
  :global(.pu-good)  { color: #22c55e; font-weight: 600; }
  :global(.pu-warn)  { color: #eab308; font-weight: 600; }
  :global(.pu-danger){ color: #ef4444; font-weight: 600; }
  :global(.pu-mono)  { font-family: monospace; font-size: 10px; }
  :global(.pu-badges){ margin-top: 6px; display: flex; flex-direction: column; gap: 3px; font-size: 10px; }
  :global(.pu-badge) { border-radius: 6px; padding: 3px 8px; font-weight: 500; border: 1px solid transparent; }
  :global(.pu-badge-sos)    { background: rgba(220, 38, 38, 0.10); color: #dc2626; border-color: rgba(220, 38, 38, 0.25); font-weight: 700; }
  :global(.pu-badge-geo)    { background: rgba(124, 58, 237, 0.10); color: #7c3aed; border-color: rgba(124, 58, 237, 0.25); }
  :global(.pu-badge-autoSos){ background: rgba(217, 119, 6, 0.10); color: #d97706; border-color: rgba(217, 119, 6, 0.25); }
  :global(.pu-badge-checkin){ background: rgba(8, 145, 178, 0.10); color: #0891b2; border-color: rgba(8, 145, 178, 0.25); }
  :global(.pu-feats) { margin-top: 5px; display: flex; flex-wrap: wrap; gap: 4px; font-size: 10px; }
  :global(.pu-feat)  { font-weight: 600; }
  :global(.pu-feat-geo)    { color: #8b5cf6; }
  :global(.pu-feat-autoSos){ color: #f59e0b; }
  :global(.pu-feat-checkin){ color: #06b6d4; }
  :global(.pu-rooms) { margin-top: 5px; font-size: 10px; color: #64748b; }

  /* Dark mode: text colours for popup */
  :global([data-theme="dark"] .pu-lbl) { color: rgba(255, 255, 255, 0.50); }
  :global([data-theme="dark"] .pu-val) { color: rgba(255, 255, 255, 0.90); }
  :global([data-theme="dark"] .pu-name) { color: rgba(255, 255, 255, 0.95); }
  :global([data-theme="dark"] .pu-rooms) { color: rgba(255, 255, 255, 0.55); }
  /* Dark mode: badge colours stay vivid, labels adapt */
  :global([data-theme="dark"] .pu-badge-sos)    { background: rgba(220, 38, 38, 0.20); border-color: rgba(220, 38, 38, 0.40); color: #fca5a5; }
  :global([data-theme="dark"] .pu-badge-geo)    { background: rgba(167, 139, 250, 0.15); border-color: rgba(167, 139, 250, 0.30); color: #c4b5fd; }
  :global([data-theme="dark"] .pu-badge-autoSos){ background: rgba(252, 211, 77, 0.12); border-color: rgba(252, 211, 77, 0.25); color: #fcd34d; }
  :global([data-theme="dark"] .pu-badge-checkin){ background: rgba(103, 232, 249, 0.12); border-color: rgba(103, 232, 249, 0.25); color: #67e8f9; }

  .safety-overlay {
    position: absolute;
    top: calc(var(--safe-top, 0px) + 92px);
    left: 50%;
    transform: translateX(-50%);
    z-index: 1000;
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
    justify-content: center;
    pointer-events: none;
  }
  .safety-chip {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 5px 12px;
    border-radius: var(--radius-full);
    font-size: 11px;
    font-weight: 600;
    line-height: 1;
    backdrop-filter: var(--glass-blur-sm, blur(12px) saturate(1.4));
    -webkit-backdrop-filter: var(--glass-blur-sm, blur(12px) saturate(1.4));
    box-shadow: var(--map-chip-shadow, 0 2px 12px rgba(0,0,0,0.10));
    pointer-events: auto;
    animation: chip-in 0.3s var(--ease-spring);
  }
  .safety-chip.geofence { background: rgba(139, 92, 246, 0.18); border: 1px solid rgba(139, 92, 246, 0.35); color: #7c3aed; }
  .safety-chip.autosos  { background: rgba(245, 158, 11, 0.18); border: 1px solid rgba(245, 158, 11, 0.35); color: #d97706; }
  .safety-chip.checkin  { background: rgba(6, 182, 212, 0.18); border: 1px solid rgba(6, 182, 212, 0.35); color: #0891b2; }

  :global([data-theme="dark"]) .safety-chip.geofence { background: rgba(139, 92, 246, 0.22); color: #a78bfa; }
  :global([data-theme="dark"]) .safety-chip.autosos  { background: rgba(245, 158, 11, 0.22); color: #fbbf24; }
  :global([data-theme="dark"]) .safety-chip.checkin  { background: rgba(6, 182, 212, 0.22); color: #22d3ee; }
  .safety-icon { font-size: 13px; }
  .safety-detail { opacity: 0.7; font-weight: 500; font-size: 10px; }
  @media (max-width: 767px) {
    .safety-overlay { top: calc(var(--safe-top, 0px) + 116px); }
  }
  @keyframes chip-in {
    from { opacity: 0; transform: translateY(-6px); }
    to { opacity: 1; transform: translateY(0); }
  }

  @media (max-width: 767px) {
    :global(.maplibregl-ctrl-group) {
      margin-bottom: calc(var(--bottom-tab-height, 56px) + var(--space-4)) !important;
      margin-right: var(--space-3) !important;
    }
  }

  /* Marker entrance animation — defined here so it's in the same scope as the global marker elements */
  :global(.map-pin-enter) {
    animation: markerPop 400ms cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
  }

  @keyframes markerPop {
    0%   { transform: scale(0) translateY(12px); opacity: 0; }
    65%  { transform: scale(1.18) translateY(-4px); opacity: 1; }
    100% { transform: scale(1) translateY(0); }
  }

  /* SOS marker: more dramatic entrance */
  :global(.map-pin.pin-sos.map-pin-enter) {
    animation: sosPinEnter 500ms cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
  }

  @keyframes sosPinEnter {
    0%   { transform: scale(0) translateY(16px); opacity: 0; }
    60%  { transform: scale(1.25) translateY(-6px); opacity: 1; }
    80%  { transform: scale(0.95) translateY(2px); }
    100% { transform: scale(1) translateY(0); }
  }

  /* Live user marker ring */
  :global(.marker-wrapper) {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  :global(.marker-ring) {
    position: absolute;
    inset: -8px;
    border-radius: 50%;
    border: 2px solid var(--user-color, #3b82f6);
    pointer-events: none;
    animation: marker-pulse-ring 2s ease-out infinite;
  }

  @keyframes marker-pulse-ring {
    0%   { transform: scale(1);   opacity: 0.5; }
    100% { transform: scale(1.8); opacity: 0; }
  }

  /* ── Arrival Intelligence chips ──────────────────────────────────────────── */
  .arrival-chips-container {
    position: absolute;
    bottom: calc(var(--bottom-tab-height, 56px) + 16px);
    left: 50%;
    transform: translateX(-50%);
    z-index: 10;
    display: flex;
    flex-direction: column;
    gap: 6px;
    pointer-events: none;
    max-width: 280px;
    width: max-content;
  }
  .arrival-chip {
    display: flex;
    align-items: center;
    gap: 8px;
    background: rgba(17, 24, 39, 0.88);
    backdrop-filter: blur(10px);
    border: 1px solid rgba(99, 102, 241, 0.4);
    border-radius: 999px;
    padding: 6px 14px 6px 10px;
    box-shadow: 0 2px 12px rgba(0,0,0,0.3);
    animation: chip-slide-up 0.25s ease;
  }
  @keyframes chip-slide-up {
    from { opacity: 0; transform: translateY(8px); }
    to   { opacity: 1; transform: translateY(0); }
  }
  .arrival-chip-icon { font-size: 15px; flex-shrink: 0; }
  .arrival-chip-body { display: flex; flex-direction: column; gap: 1px; }
  .arrival-name {
    font-size: 12px;
    font-weight: 700;
    color: #f1f5f9;
    line-height: 1.2;
  }
  .arrival-eta {
    font-size: 11px;
    color: #a5b4fc;
    line-height: 1.2;
  }

  /* Navigation arrow pulse */
  :global(.nav-arrow-marker) { z-index: 10 !important; }
  @keyframes nav-pulse {
    0% { transform: scale(1); opacity: 0.6; }
    100% { transform: scale(2.2); opacity: 0; }
  }
</style>
