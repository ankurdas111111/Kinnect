<script>
  import { onMount, onDestroy } from 'svelte';
  // maplibregl is loaded dynamically inside onMount so the main bundle
  // does not block on the ~283 kB maplibre chunk at parse time.
  let maplibregl;
  import { otherUsers, myLocation, mySocketId, mySafetyStatus, focusUser, mapFlyTo, routeGeometry, navigationState, mapTappedUser, mapChatRequest } from '../lib/stores/map.js';
  import { haptics } from '../lib/haptics.js';
  import { createMapIcon, createPersonMarker, getPresenceState, escapeAttr, calculateDistance, formatDistance, circleGeoJSON } from '../lib/tracking.js';
  import { animateMarkerTo, cancelAnimation, cancelAllAnimations } from '../lib/markerInterpolator.js';
  import { getUserColor } from '../lib/getUserColor.js';
  import { MAP_STYLE, RASTER_STYLE } from '../lib/mapStyle.js';
  import { debounce } from '../lib/debounce.js';
  // F3: meeting point markers
  import { myRooms } from '../lib/stores/rooms.js';

  export let followMode = false;

  let mapContainer;
  let map;
  let markers = new Map();       // sid → maplibregl.Marker
  let markerPopups = new Map();   // sid → maplibregl.Popup
  let markerUsers = new Map();    // sid → user object (for mobile tap → QA sheet)
  let markerState = new Map();
  let geofenceIds = new Set();
  let myMarker = null;
  let myPopup = null;
  let hasSetView = false;
  // F3: roomCode → maplibregl.Marker
  let meetingMarkers = new Map();

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

  // F3: create a green flag-style meeting point marker element
  function createMeetingPointEl(label) {
    const el = document.createElement('div');
    el.style.cssText = 'display:flex;flex-direction:column;align-items:center;cursor:pointer;';
    const safeLabel = (label || 'Meet here').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    el.innerHTML = `
      <div style="background:linear-gradient(135deg,#10b981,#059669);width:34px;height:34px;border-radius:50%;border:3px solid #fff;box-shadow:0 2px 14px rgba(16,185,129,0.55);display:flex;align-items:center;justify-content:center;">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="#fff"><path d="M4 15V5l5 3 3-4 3 4 5-3v10"/><line x1="4" y1="22" x2="4" y2="15" stroke="#fff" stroke-width="2"/></svg>
      </div>
      <div style="background:rgba(16,185,129,0.92);color:#fff;font-size:10px;font-weight:700;padding:2px 7px;border-radius:6px;margin-top:3px;white-space:nowrap;max-width:120px;overflow:hidden;text-overflow:ellipsis;box-shadow:0 1px 6px rgba(0,0,0,0.18);">${safeLabel}</div>
      <div style="width:2px;height:7px;background:#10b981;margin-top:1px;border-radius:0 0 2px 2px;opacity:0.7;"></div>`;
    return el;
  }

  const debouncedCheckMobile = debounce(checkMobile, 80);

  onMount(async () => {
    // Dynamic import breaks the eager-load dependency on the maplibre chunk.
    // The Vite manualChunks config already puts maplibre in its own chunk; this
    // import() call makes that chunk truly lazy so main-bundle parse time is lower.
    const ml = await import('maplibre-gl');
    maplibregl = ml.default;
    await import('maplibre-gl/dist/maplibre-gl.css');

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

    }

    map = new maplibregl.Map({
      container: mapContainer,
      style: MAP_STYLE,
      center: [78.9629, 20.5937], // Center of India
      zoom: 4,
      attributionControl: true
    });

    // NavigationControl only on desktop — mobile users pinch-to-zoom naturally,
    // and bottom-right on mobile conflicts with the MapFab cluster.
    if (!isMobile) {
      map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-right');
    }

    // Delegated handler for chat buttons inside desktop marker popups.
    // Uses capture phase so it intercepts before MapLibre's own click handling.
    mapContainer.addEventListener('click', (e) => {
      const btn = e.target.closest('.pu-chat-btn');
      if (!btn) return;
      e.stopPropagation();
      const userId = btn.dataset.userid;
      const name = btn.dataset.name;
      if (userId) {
        haptics.tap?.();
        mapChatRequest.set({ id: userId, name });
      }
    }, true);

    map.on('dragstart', () => {
      // During navigation, let user pan but re-center on next GPS update
      if (!navActive) followMode = false;
    });
    map.on('load', () => {
      // Defensive resize in case the container had 0×0 dimensions at init time
      // (e.g. the DOM was not yet fully laid out when MapLibre mounted).
      requestAnimationFrame(() => map.resize());
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
    });

    // Feature 5: Double-tap map canvas to center on self
    // Use non-passive capture so we can preventDefault to stop MapLibre's native double-tap zoom.
    let dtLastTap = 0;
    let dtLastX = 0;
    let dtLastY = 0;
    function onCanvasDoubleTap(e) {
      const now = Date.now();
      const touch = e.changedTouches[0];
      const dx = Math.abs(touch.clientX - dtLastX);
      const dy = Math.abs(touch.clientY - dtLastY);
      if (now - dtLastTap < 300 && dx < 30 && dy < 30) {
        e.preventDefault();
        focusUser.set('__self__');
        haptics.tap?.();
        dtLastTap = 0; // reset so triple-tap doesn't re-fire
      } else {
        dtLastTap = now;
        dtLastX = touch.clientX;
        dtLastY = touch.clientY;
      }
    }
    if (isMobile) {
      map.getCanvas().addEventListener('touchend', onCanvasDoubleTap, { passive: false, capture: true });
    }
  });

  onDestroy(() => {
    cancelAllAnimations();
    if (renderUsersRaf) cancelAnimationFrame(renderUsersRaf);
    for (const m of markers.values()) m.remove();
    markers.clear();
    for (const p of markerPopups.values()) p.remove();
    markerPopups.clear();
    markerUsers.clear();
    if (myMarker) myMarker.remove();
    if (myPopup) myPopup.remove();
    // F3: clean up meeting point markers
    for (const m of meetingMarkers.values()) m.remove();
    meetingMarkers.clear();
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
    if (user.userId) {
      html += `<div class="pu-actions"><button class="pu-chat-btn" data-userid="${escapeAttr(user.userId)}" data-name="${name}"><svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg> Chat</button></div>`;
    }
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

    // On mobile: tapping a marker directly opens the quick-action sheet (skipping the
    // MapLibre popup). On desktop the popup behaviour is unchanged.
    function attachMobileTap(el, userObj) {
      el.addEventListener('click', () => {
        haptics.tap?.();
        mapTappedUser.set(userObj);
      });
    }

    markerUsers.set(sid, user);

    if (markers.has(sid)) {
      const m = markers.get(sid);
      animateMarkerTo(sid, m, lngLat);
      if (markerState.get(sid) !== iconKey) {
        const el = makePersonEl();
        if (isMobile) {
          attachMobileTap(el, user);
          const newMarker = new maplibregl.Marker({ element: el, anchor: 'bottom' })
            .setLngLat(m.getLngLat())
            .addTo(map);
          m.remove();
          markers.set(sid, newMarker);
          markerState.set(sid, iconKey);
        } else {
          const newMarker = new maplibregl.Marker({ element: el, anchor: 'bottom' })
            .setLngLat(m.getLngLat())
            .addTo(map);
          const popup = markerPopups.get(sid);
          if (popup) { popup.setHTML(popupContent); newMarker.setPopup(popup); }
          m.remove();
          markers.set(sid, newMarker);
          markerState.set(sid, iconKey);
        }
      } else {
        if (!isMobile) {
          const popup = markerPopups.get(sid);
          if (popup) popup.setHTML(popupContent);
        }
      }
    } else {
      const el = makePersonEl();
      if (isMobile) {
        attachMobileTap(el, user);
        const m = new maplibregl.Marker({ element: el, anchor: 'bottom' })
          .setLngLat(lngLat)
          .addTo(map);
        markers.set(sid, m);
        markerState.set(sid, iconKey);
      } else {
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
        markerUsers.delete(sid);
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

  // ── F3: Meeting point markers — one green flag per room that has a meeting point ──
  $: if (map && maplibregl) {
    const rooms = $myRooms;
    const currentCodes = new Set(rooms.filter(r => r.meetingPoint).map(r => r.code));

    // Remove markers for rooms that no longer have a meeting point
    for (const [code, m] of meetingMarkers) {
      if (!currentCodes.has(code)) {
        m.remove();
        meetingMarkers.delete(code);
      }
    }

    // Add or update markers for rooms with a meeting point
    for (const room of rooms) {
      if (!room.meetingPoint) continue;
      const mp = room.meetingPoint;
      const lngLat = [mp.lng, mp.lat];
      if (meetingMarkers.has(room.code)) {
        meetingMarkers.get(room.code).setLngLat(lngLat);
      } else {
        const el = createMeetingPointEl(mp.label || room.name || room.code);
        const m = new maplibregl.Marker({ element: el, anchor: 'bottom' })
          .setLngLat(lngLat)
          .addTo(map);
        meetingMarkers.set(room.code, m);
      }
    }
  }

</script>

<div class="map-container" bind:this={mapContainer}></div>

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
  /* Fix #3: reduce vignette opacity in dark mode — gradient is already dark so it becomes too heavy */
  :global([data-theme="dark"]) .map-vignette-top    { opacity: 0.18; }
  :global([data-theme="dark"]) .map-vignette-bottom { opacity: 0.15; }

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
  /* Fix #1: increased base font sizes for readability on mobile */
  :global(.pu-wrap)  { min-width: 190px; font-size: 13px; line-height: 1.5; }
  :global(.pu-hdr)   { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
  :global(.pu-name)  { font-family: var(--font-display); font-size: 15px; font-weight: 700; letter-spacing: -0.01em; color: #0f172a; }
  :global(.pu-status) { display: inline-flex; align-items: center; gap: 3px; font-size: 10px; font-weight: 600; }
  :global(.pu-dot)   { width: 7px; height: 7px; border-radius: 50%; background: currentColor; display: inline-block; }
  :global(.pu-online)  { color: #22c55e; }
  :global(.pu-offline) { color: #9ca3af; }
  :global(.pu-grid)  { display: grid; grid-template-columns: auto 1fr; gap: 2px 10px; font-size: 13px; }
  :global(.pu-lbl)   { font-size: 12px; font-weight: 600; color: #64748b; }
  :global(.pu-val)   { font-size: 13px; color: #1e293b; }
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
  :global(.pu-actions) { margin-top: 8px; padding-top: 8px; border-top: 1px solid rgba(0,0,0,0.08); }
  :global(.pu-chat-btn) { display: inline-flex; align-items: center; gap: 5px; padding: 5px 12px; border-radius: 8px; background: rgba(99,102,241,0.10); border: 1px solid rgba(99,102,241,0.22); color: #6366f1; font-size: 12px; font-weight: 600; cursor: pointer; transition: background 120ms; }
  :global(.pu-chat-btn:hover) { background: rgba(99,102,241,0.18); }

  /* Fix #1 continued: mobile-specific font size bump for popup text */
  @media (max-width: 480px) {
    :global(.pu-wrap) { font-size: 14px; }
    :global(.pu-val)  { font-size: 14px; }
    :global(.pu-lbl)  { font-size: 13px; }
    :global(.pu-grid) { font-size: 14px; }
  }

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

  /* Navigation arrow pulse */
  :global(.nav-arrow-marker) { z-index: 10 !important; }
  @keyframes nav-pulse {
    0% { transform: scale(1); opacity: 0.6; }
    100% { transform: scale(2.2); opacity: 0; }
  }
</style>
