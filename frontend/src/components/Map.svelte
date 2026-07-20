<script>
  import { run } from 'svelte/legacy';

  import { onMount, onDestroy } from 'svelte';
  // maplibregl is loaded dynamically inside onMount so the main bundle
  // does not block on the ~283 kB maplibre chunk at parse time.
  let maplibregl = $state();
  import { otherUsers, myLocation, mySocketId, mySafetyStatus, focusUser, mapFlyTo, routeGeometry, navigationState, mapTappedUser, mapChatRequest, navFollow, aiDirectives } from '../lib/stores/map.js';
  import { recentTrailResult } from '../lib/stores/trail.js';
  import { emitGetRecentTrail } from '../lib/socket.js';
  import { haptics } from '../lib/haptics.js';
  import { createMapIcon, createPersonMarker, getPresenceState, circleGeoJSON } from '../lib/tracking.js';
  import { buildMemberPopup, buildSelfPopup, memberPopupHash, createNavArrowEl, createMeetingPointEl, createDestMarkerEl } from '../lib/mapPopup.js';
  import { animateMarkerTo, cancelAnimation, cancelAllAnimations } from '../lib/markerInterpolator.js';
  import { getUserColor } from '../lib/getUserColor.js';
  import { MAP_STYLE, RASTER_STYLE } from '../lib/mapStyle.js';
  import { debounce } from '../lib/debounce.js';
  // F3: meeting point markers
  import { myRooms } from '../lib/stores/rooms.js';

  /**
   * @typedef {Object} Props
   * @property {boolean} [followMode]
   */

  /** @type {Props} */
  let { followMode = $bindable(false) } = $props();

  let mapContainer = $state();
  let map = $state();
  let markers = new Map();       // sid → maplibregl.Marker
  let markerPopups = new Map();   // sid → maplibregl.Popup
  let markerUsers = new Map();    // sid → user object (for mobile tap → QA sheet)
  let markerState = new Map();
  let geofenceIds = new Set();
  let myMarker = $state(null);
  let myPopup = $state(null);
  let hasSetView = $state(false);
  // F3: roomCode → maplibregl.Marker
  let meetingMarkers = new Map();

  // ── Navigation camera state ────────────────────────────────────────
  let prevNavLat = $state(null);
  let prevNavLng = $state(null);
  let currentBearing = $state(0); // degrees, 0=north, 90=east

  // navFollow subscription (camera-follow driven by PlaceSearch follower)
  let _navFollowUnsub = null;
  let _navFollowPrevActive = false;
  let _navFollowLastEaseTo = 0;

  /** Compute bearing in degrees from point A to point B */
  function computeBearing(lat1, lng1, lat2, lng2) {
    const toRad = Math.PI / 180;
    const dLng = (lng2 - lng1) * toRad;
    const y = Math.sin(dLng) * Math.cos(lat2 * toRad);
    const x = Math.cos(lat1 * toRad) * Math.sin(lat2 * toRad) -
              Math.sin(lat1 * toRad) * Math.cos(lat2 * toRad) * Math.cos(dLng);
    return ((Math.atan2(y, x) * 180 / Math.PI) + 360) % 360;
  }
  let isMobile = $state(typeof window !== 'undefined' ? window.innerWidth < 768 : false);
  let renderUsersRaf = $state(null);
  let pendingUsers = $state(new Map());
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

    // Delegated handler for action buttons inside desktop marker popups.
    // Uses capture phase so it intercepts before MapLibre's own click handling.
    mapContainer.addEventListener('click', (e) => {
      const chatBtn = e.target.closest('.pu-chat-btn');
      if (chatBtn) {
        e.stopPropagation();
        const userId = chatBtn.dataset.userid;
        const name = chatBtn.dataset.name;
        if (userId) { haptics.tap?.(); mapChatRequest.set({ id: userId, name }); }
        return;
      }
      const trailBtn = e.target.closest('.pu-trail-btn');
      if (trailBtn) {
        e.stopPropagation();
        const userId = trailBtn.dataset.userid;
        if (userId) { haptics.tap?.(); requestTrailForUser(userId); }
        return;
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

    // ── navFollow: camera-follow driven by the PlaceSearch follower ───────
    // Subscribe after map is created so the initial callback (active=false) is a no-op.
    _navFollowUnsub = navFollow.subscribe(nf => {
      if (!map) return;
      const now = Date.now();
      if (nf.active) {
        // Throttle: skip if we issued an easeTo <900 ms ago
        if (now - _navFollowLastEaseTo < 900) return;
        _navFollowLastEaseTo = now;
        map.easeTo({
          center: [nf.lng, nf.lat],
          bearing: nf.bearing,
          zoom: Math.max(map.getZoom(), 16.5),
          pitch: 50,
          duration: 950,
          essential: true,
        });
      } else if (_navFollowPrevActive) {
        // active just flipped false — reset to flat north-up view
        map.easeTo({ pitch: 0, bearing: 0, duration: 600 });
      }
      _navFollowPrevActive = nf.active;
    });
  });

  onDestroy(() => {
    cancelAllAnimations();
    if (renderUsersRaf) cancelAnimationFrame(renderUsersRaf);
    _navFollowUnsub?.();
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



  // Popup HTML is built by the escaped, token-classed helpers in lib/mapPopup.js.
  // This wrapper only adds the per-socket memoization used by dirty tracking.
  function buildPopupCached(user) {
    const ml = $myLocation;
    const hash = memberPopupHash(user, ml);
    const cached = popupCache.get(user.socketId);
    if (cached && cached.hash === hash) return cached.html;
    const html = buildMemberPopup(user, ml);
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
    // TECHNIQUE 9/10: second ring layer — staggered 0.7s for multi-ring radar sweep
    const ring2 = document.createElement('div');
    ring2.className = 'marker-ring marker-ring-2 animate-pulse-ring';
    ring2.style.setProperty('--user-color', color);
    wrapper.appendChild(ring);
    wrapper.appendChild(ring2);
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


  // Snapshot of last-rendered cluster positions — skip rebuild if nothing moved
  let _clusterPosSnapshot = new Map(); // socketId → 'lat,lng,sos,online'




  // ── Trail playback layer — draw breadcrumb path for a user ──────────
  let activeTrailUserId = $state(null); // userId whose trail is shown (or null)


  function clearTrail() {
    if (!map) return;
    if (map.getLayer('trail-line')) map.removeLayer('trail-line');
    if (map.getLayer('trail-glow')) map.removeLayer('trail-glow');
    if (map.getSource('trail-route')) map.removeSource('trail-route');
    activeTrailUserId = null;
    recentTrailResult.set(null);
  }

  function requestTrailForUser(userId) {
    clearTrail();
    activeTrailUserId = userId;
    emitGetRecentTrail(userId, 60);
  }

  // ── Navigation mode — follow user + destination marker + fit bounds ──
  let destMarker = $state(null);
  let navActive = $state(false);

  let mapReady = false;




  run(() => {
    if (map) {
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
        const destEl = createDestMarkerEl();
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
  });
  run(() => {
    if (map && $myLocation) {
      const { latitude, longitude } = $myLocation;
      const lngLat = [longitude, latitude];
      const selfPopupHtml = buildSelfPopup($myLocation, $mySafetyStatus);

      if (!myMarker) {
        const el = navActive ? createNavArrowEl() : createMapIcon('var(--primary-500)', '', { markerType: 'self' });
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
          const newEl = createNavArrowEl();
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
  });
  run(() => {
    if (map && map.loaded()) {
      const gf = $mySafetyStatus?.geofence;
      if (gf?.enabled && gf.centerLat != null && gf.centerLng != null && gf.radiusM > 0) {
        updateCircleSource('my-geofence', [gf.centerLng, gf.centerLat], gf.radiusM);
      } else {
        updateCircleSource('my-geofence', [0, 0], 0);
      }
    }
  });
  run(() => {
    if (map) {
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
  });
  // Update cluster GeoJSON only when user positions actually change
  run(() => {
    if (map && map.getSource('users-cluster')) {
      let dirty = false;
      for (const user of $otherUsers.values()) {
        const key = `${user.latitude},${user.longitude},${!!user.sos?.active},${user.online !== false}`;
        if (_clusterPosSnapshot.get(user.socketId) !== key) { dirty = true; break; }
      }
      // Also rebuild if a user was removed
      if (!dirty && _clusterPosSnapshot.size !== $otherUsers.size) dirty = true;
      if (dirty) {
        const features = [];
        for (const user of $otherUsers.values()) {
          if (user.latitude == null || user.longitude == null || user.online === false) continue;
          features.push({
            type: 'Feature',
            geometry: { type: 'Point', coordinates: [user.longitude, user.latitude] },
            properties: { name: user.displayName || 'User', sos: !!user.sos?.active },
          });
          _clusterPosSnapshot.set(user.socketId, `${user.latitude},${user.longitude},${!!user.sos?.active},${user.online !== false}`);
        }
        map.getSource('users-cluster').setData({ type: 'FeatureCollection', features });
      }
    }
  });
  // ── AI copilot annotation helpers ────────────────────────────────────
  let aiMarkers = [];

  function clearAiAnnotations() {
    for (const m of aiMarkers) m.remove();
    aiMarkers = [];
    if (map.getLayer('ai-trail-line')) map.removeLayer('ai-trail-line');
    if (map.getLayer('ai-trail-glow')) map.removeLayer('ai-trail-glow');
    if (map.getSource('ai-trail')) map.removeSource('ai-trail');
  }

  function executeAiDirectives(directives) {
    const allCoords = [];
    const trailFeatures = [];
    for (const d of directives) {
      if (d.type === 'add_pin' && d.lat != null && d.lng != null) {
        const el = document.createElement('div');
        el.className = 'ai-pin';
        el.innerHTML = `<span class="ai-pin-dot"></span>${d.label ? `<span class="ai-pin-label">${escapeHtml(d.label)}</span>` : ''}`;
        const marker = new maplibregl.Marker({ element: el, anchor: 'bottom' })
          .setLngLat([d.lng, d.lat])
          .addTo(map);
        aiMarkers.push(marker);
        allCoords.push([d.lng, d.lat]);
      } else if (d.type === 'draw_trail' && Array.isArray(d.points) && d.points.length > 1) {
        // Accumulate every trail into ONE FeatureCollection — a single 'ai-trail'
        // source with setData would otherwise let each trail overwrite the last.
        trailFeatures.push({ type: 'Feature', geometry: { type: 'LineString', coordinates: d.points }, properties: { user: d.user || '' } });
        allCoords.push(...d.points);
      }
    }
    if (trailFeatures.length) {
      const fc = { type: 'FeatureCollection', features: trailFeatures };
      if (map.getSource('ai-trail')) {
        map.getSource('ai-trail').setData(fc);
      } else {
        map.addSource('ai-trail', { type: 'geojson', data: fc });
        // NOTE: violet hex literals mirror --accent-500 tokens (MapLibre needs literals).
        map.addLayer({ id: 'ai-trail-glow', type: 'line', source: 'ai-trail',
          paint: { 'line-color': '#a78bfa', 'line-width': 10, 'line-opacity': 0.22, 'line-blur': 4 } });
        map.addLayer({ id: 'ai-trail-line', type: 'line', source: 'ai-trail',
          paint: { 'line-color': '#c4b5fd', 'line-width': 4, 'line-opacity': 0.9 },
          layout: { 'line-cap': 'round', 'line-join': 'round' } });
      }
    }
    // Camera: explicit fly_to wins; otherwise fit to everything annotated.
    const fly = directives.find(d => d.type === 'fly_to' && d.lat != null && d.lng != null);
    if (fly) {
      map.flyTo({ center: [fly.lng, fly.lat], zoom: fly.zoom || 15, duration: 1100 });
    } else if (allCoords.length > 1) {
      const bounds = allCoords.reduce((b, c) => b.extend(c), new maplibregl.LngLatBounds(allCoords[0], allCoords[0]));
      map.fitBounds(bounds, { padding: 90, duration: 900, maxZoom: 16.5 });
    } else if (allCoords.length === 1) {
      map.flyTo({ center: allCoords[0], zoom: 15.5, duration: 1100 });
    }
  }

  function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, ch => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[ch]));
  }

  // ── Place search fly-to ──────────────────────────────────────────────
  run(() => {
    if (map && $mapFlyTo) {
      const target = $mapFlyTo;
      mapFlyTo.set(null);
      map.flyTo({ center: [target.lng, target.lat], zoom: target.zoom || 16, duration: 900 });
    }
  });
  // ── Route polyline from directions ─────────────────────────────────
  run(() => {
    if (map) {
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
          // NOTE: brand-indigo hex literals mirror --primary-500 / --indigo-400 tokens.
          // MapLibre paint needs literal colors (this file does not read CSS vars at runtime).
          map.addLayer({ id: glowId, type: 'line', source: srcId,
            paint: { 'line-color': '#6366f1', 'line-width': 9, 'line-opacity': 0.24, 'line-blur': 4 } });
          map.addLayer({ id: layerId, type: 'line', source: srcId,
            paint: { 'line-color': '#818cf8', 'line-width': 4.5, 'line-opacity': 0.92 },
            layout: { 'line-cap': 'round', 'line-join': 'round' } });
        }
      } else {
        // Clear route
        if (map.getLayer('directions-route-line')) map.removeLayer('directions-route-line');
        if (map.getLayer('directions-route-glow')) map.removeLayer('directions-route-glow');
        if (map.getSource('directions-route')) map.removeSource('directions-route');
      }
    }
  });
  // ── Ask-the-Map AI copilot directives ───────────────────────────────
  run(() => {
    if (map && $aiDirectives) {
      const payload = $aiDirectives;
      aiDirectives.set(null);
      clearAiAnnotations();
      if (!payload.clear && Array.isArray(payload.directives)) {
        executeAiDirectives(payload.directives);
      }
    }
  });
  run(() => {
    if (map && $recentTrailResult) {
      const res = $recentTrailResult;
      if (!res.ok || !res.points?.length) {
        // Clear on error or empty
        if (map.getLayer('trail-line')) map.removeLayer('trail-line');
        if (map.getLayer('trail-glow')) map.removeLayer('trail-glow');
        if (map.getSource('trail-route')) map.removeSource('trail-route');
        activeTrailUserId = null;
      } else {
        activeTrailUserId = res.targetUserId;
        const coords = res.points.map(p => [p.lng, p.lat]);
        const geojson = { type: 'Feature', geometry: { type: 'LineString', coordinates: coords }, properties: {} };
        if (map.getSource('trail-route')) {
          map.getSource('trail-route').setData(geojson);
        } else {
          map.addSource('trail-route', { type: 'geojson', data: geojson });
          // NOTE: amber hex literals mirror --warning-500 / --warning-400 tokens.
          // Trail breadcrumb — clearer but calm: a touch wider/more opaque than before.
          map.addLayer({ id: 'trail-glow', type: 'line', source: 'trail-route',
            paint: { 'line-color': '#f59e0b', 'line-width': 11, 'line-opacity': 0.20, 'line-blur': 4 } });
          map.addLayer({ id: 'trail-line', type: 'line', source: 'trail-route',
            paint: { 'line-color': '#fbbf24', 'line-width': 3.5, 'line-opacity': 0.9, 'line-dasharray': [2.5, 1.5] },
            layout: { 'line-cap': 'round', 'line-join': 'round' } });
        }
        // Fit to trail bounds
        if (coords.length > 1) {
          const bounds = coords.reduce((b, c) => b.extend(c), new maplibregl.LngLatBounds(coords[0], coords[0]));
          map.fitBounds(bounds, { padding: 80, duration: 700, maxZoom: 17 });
        }
      }
    }
  });
  run(() => {
    if (map && $focusUser) {
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
  });
  // ── F3: Meeting point markers — one green flag per room that has a meeting point ──
  run(() => {
    if (map && maplibregl) {
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
  });
</script>

<div class="map-container" bind:this={mapContainer}
     role="application"
     aria-label="Live family location map. Use arrow keys to pan. Family member markers can be focused with Tab."></div>

<!-- MERIDIAN: Map vignette — connects UI chrome to map visually -->
<div class="map-vignette-top" aria-hidden="true"></div>
<div class="map-vignette-bottom" aria-hidden="true"></div>

<!-- MERIDIAN SPATIAL: Contextual float chips — accuracy + speed -->
{#if $myLocation}
  {#if $myLocation.accuracy != null}
    {@const acc = $myLocation.accuracy}
    <div class="map-float-chip accuracy-float-chip tabular-nums"
         class:chip-precise={acc <= 20}
         class:chip-ok={acc > 20 && acc <= 80}
         class:chip-rough={acc > 80}
         aria-live="polite"
         aria-atomic="true"
         aria-label="GPS accuracy {Math.round(acc)}m"
         style="top: {isMobile ? 'calc(var(--safe-top, 0px) + 116px)' : 'var(--space-3)'}; right: var(--space-3);">
      <span class="chip-dot" aria-hidden="true"></span>
      {acc <= 20 ? `±${Math.round(acc)}m` : acc <= 80 ? `~${Math.round(acc)}m` : 'Rough GPS'}
    </div>
  {/if}
  {#if $myLocation.speed != null && $myLocation.speed >= 4}
    <div class="map-float-chip chip-speed tabular-nums"
         aria-live="polite"
         aria-atomic="true"
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

  /* ── AI copilot pins (Ask the Map) ───────────────────────────── */
  :global(.ai-pin) {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    pointer-events: none;
  }
  :global(.ai-pin-label) {
    order: -1;
    max-width: 220px;
    padding: 4px 10px;
    border-radius: 999px;
    background: color-mix(in oklab, var(--bg-elevated, #171a2e) 88%, transparent);
    border: 1px solid color-mix(in oklab, var(--accent-400, #a78bfa) 45%, transparent);
    color: var(--text-primary, #e7e9f4);
    font-size: 11.5px;
    font-weight: 600;
    line-height: 1.35;
    text-align: center;
    backdrop-filter: blur(6px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.35);
    animation: ai-pin-in 0.4s cubic-bezier(0.34, 1.56, 0.64, 1) both;
  }
  :global(.ai-pin-dot) {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: radial-gradient(circle at 35% 35%, #c4b5fd, #7c3aed);
    border: 2px solid #fff;
    box-shadow: 0 0 0 4px rgba(167, 139, 250, 0.25), 0 2px 8px rgba(0, 0, 0, 0.4);
    animation: ai-pin-in 0.4s cubic-bezier(0.34, 1.56, 0.64, 1) both;
  }
  @keyframes ai-pin-in {
    from { opacity: 0; transform: translateY(-8px) scale(0.6); }
    to   { opacity: 1; transform: translateY(0) scale(1); }
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
    background: linear-gradient(135deg, var(--indigo-400), var(--indigo-500));
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
  :global(.map-pin:focus-visible),
  :global([role="button"].map-pin:focus-visible),
  :global([role="button"][tabindex="0"]:focus-visible) {
    outline: 3px solid var(--primary-500);
    outline-offset: 3px;
    border-radius: var(--radius-full);
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
    background: var(--danger-500, #ef4444);
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
    background: var(--danger-500, #ef4444);
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
    color: var(--popup-text-val, #1e293b);
    border-radius: var(--radius-xl, 20px);
    box-shadow:
      0 12px 40px rgba(0, 0, 0, 0.22),
      0 0 0 1px rgba(0, 0, 0, 0.06),
      inset 0 1px 0 rgba(255, 255, 255, 0.70);
    padding: 16px 18px;
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
  :global(.pu-hdr)   { display: flex; align-items: center; gap: 6px; margin-bottom: 10px; padding-bottom: 8px; border-bottom: 1px solid rgba(0,0,0,0.07); }
  :global(.pu-name)  { font-family: var(--font-display); font-size: 15px; font-weight: 700; letter-spacing: -0.01em; color: var(--popup-text-heading, #0f172a); }
  :global(.pu-status) { display: inline-flex; align-items: center; gap: 4px; margin-left: auto; font-size: 10px; font-weight: 600; }
  :global(.pu-dot)   { width: 7px; height: 7px; border-radius: 50%; background: currentColor; display: inline-block; }
  :global(.pu-online)  { color: var(--success-500, #22c55e); }
  :global(.pu-offline) { color: var(--gray-400, #9ca3af); }
  :global(.pu-grid)  { display: grid; grid-template-columns: auto 1fr; gap: 6px 14px; font-size: 13px; align-items: baseline; }
  :global(.pu-lbl)   { font-size: 11px; font-weight: 500; letter-spacing: 0.02em; text-transform: uppercase; color: var(--popup-text-label, #64748b); }
  :global(.pu-val)   { font-size: 13px; font-weight: 600; text-align: right; color: var(--popup-text-val, #1e293b); }
  :global(.pu-good)  { color: var(--success-500, #22c55e); font-weight: 700; }
  :global(.pu-warn)  { color: var(--warning-500, #eab308); font-weight: 700; }
  :global(.pu-danger){ color: var(--danger-500, #ef4444); font-weight: 700; }
  :global(.pu-mono)  { font-family: monospace; font-size: 11px; letter-spacing: -0.02em; }
  :global(.pu-badges){ margin-top: 10px; display: flex; flex-direction: column; gap: 4px; font-size: 10px; }
  :global(.pu-badge) { border-radius: 8px; padding: 4px 9px; font-weight: 500; border: 1px solid transparent; }
  :global(.pu-badge-sos)    { background: rgba(220, 38, 38, 0.10); color: var(--danger-600, #dc2626); border-color: rgba(220, 38, 38, 0.25); font-weight: 700; }
  :global(.pu-badge-geo)    { background: rgba(124, 58, 237, 0.10); color: var(--violet-600); border-color: rgba(124, 58, 237, 0.25); }
  :global(.pu-badge-autoSos){ background: rgba(217, 119, 6, 0.10); color: var(--warning-600); border-color: rgba(217, 119, 6, 0.25); }
  :global(.pu-badge-checkin){ background: rgba(8, 145, 178, 0.10); color: #0891b2; border-color: rgba(8, 145, 178, 0.25); }
  :global(.pu-feats) { margin-top: 8px; display: flex; flex-wrap: wrap; gap: 5px; font-size: 10px; }
  :global(.pu-feat)  { font-weight: 600; }
  :global(.pu-feat-geo)    { color: #8b5cf6; }
  :global(.pu-feat-autoSos){ color: var(--warning-500); }
  :global(.pu-feat-checkin){ color: #06b6d4; }
  :global(.pu-rooms) { margin-top: 8px; font-size: 10px; color: var(--popup-text-label, #64748b); }
  :global(.pu-actions) { display: flex; gap: 8px; align-items: center; margin-top: 10px; padding-top: 10px; border-top: 1px solid rgba(0,0,0,0.08); }
  :global(.pu-chat-btn) { display: inline-flex; align-items: center; gap: 5px; padding: 6px 13px; border-radius: 9px; background: rgba(99,102,241,0.10); border: 1px solid rgba(99,102,241,0.22); color: var(--indigo-500); font-size: 12px; font-weight: 600; cursor: pointer; transition: background 120ms; }
  :global(.pu-chat-btn:hover) { background: rgba(99,102,241,0.18); }
  :global(.pu-trail-btn) { display: inline-flex; align-items: center; gap: 5px; padding: 6px 13px; border-radius: 9px; background: rgba(245,158,11,0.10); border: 1px solid rgba(245,158,11,0.22); color: var(--warning-600); font-size: 12px; font-weight: 600; cursor: pointer; transition: background 120ms; }
  :global(.pu-trail-btn:hover) { background: rgba(245,158,11,0.18); }
  :global([data-theme="dark"] .pu-trail-btn) { background: rgba(252,211,77,0.10); border-color: rgba(252,211,77,0.22); color: var(--warning-300); }
  :global([data-theme="dark"] .pu-trail-btn:hover) { background: rgba(252,211,77,0.18); }

  /* Fix #1 continued: mobile-specific font size bump for popup text */
  @media (max-width: 480px) {
    :global(.pu-wrap) { font-size: 14px; }
    :global(.pu-val)  { font-size: 14px; }
    :global(.pu-lbl)  { font-size: 12px; }
    :global(.pu-grid) { font-size: 14px; }
  }

  /* Dark mode: text colours for popup */
  :global([data-theme="dark"] .pu-hdr) { border-bottom-color: rgba(255, 255, 255, 0.09); }
  :global([data-theme="dark"] .pu-lbl) { color: rgba(255, 255, 255, 0.50); }
  :global([data-theme="dark"] .pu-val) { color: rgba(255, 255, 255, 0.90); }
  :global([data-theme="dark"] .pu-name) { color: rgba(255, 255, 255, 0.95); }
  :global([data-theme="dark"] .pu-rooms) { color: rgba(255, 255, 255, 0.55); }
  :global([data-theme="dark"] .pu-actions) { border-top-color: rgba(255, 255, 255, 0.10); }
  /* Dark mode: badge colours stay vivid, labels adapt */
  :global([data-theme="dark"] .pu-badge-sos)    { background: rgba(220, 38, 38, 0.20); border-color: rgba(220, 38, 38, 0.40); color: var(--danger-300); }
  :global([data-theme="dark"] .pu-badge-geo)    { background: rgba(167, 139, 250, 0.15); border-color: rgba(167, 139, 250, 0.30); color: var(--violet-300); }
  :global([data-theme="dark"] .pu-badge-autoSos){ background: rgba(252, 211, 77, 0.12); border-color: rgba(252, 211, 77, 0.25); color: var(--warning-300); }
  :global([data-theme="dark"] .pu-badge-checkin){ background: rgba(103, 232, 249, 0.12); border-color: rgba(103, 232, 249, 0.25); color: #67e8f9; }

  /* ── Contextual float chip — floating-glass chip tier + status halo ──────
     Base chip lives in global.css; these overrides re-source the material onto
     the shared --glass-chip-* tier (calm/minimal degradation for free) and add
     breathing room plus a soft, token-derived status ring so the accuracy/speed
     dot reads at a glance. */
  :global(.map-float-chip) {
    padding: 5px 12px;
    gap: 6px;
    letter-spacing: 0.02em;
    background: var(--glass-chip-bg);
    border: 1px solid var(--glass-chip-border);
    box-shadow: var(--glass-chip-shadow);
    backdrop-filter: var(--glass-chip-blur);
    -webkit-backdrop-filter: var(--glass-chip-blur);
  }
  :global(.map-float-chip .chip-dot) {
    width: 7px;
    height: 7px;
  }
  :global(.map-float-chip.chip-precise .chip-dot) { box-shadow: 0 0 0 3px color-mix(in oklch, var(--success-400) 22%, transparent); }
  :global(.map-float-chip.chip-ok .chip-dot)      { box-shadow: 0 0 0 3px color-mix(in oklch, var(--warning-400) 22%, transparent); }
  :global(.map-float-chip.chip-rough .chip-dot)   { box-shadow: 0 0 0 3px color-mix(in oklch, var(--danger-400) 22%, transparent); }
  :global(.map-float-chip.chip-speed .chip-dot)   { box-shadow: 0 0 0 3px color-mix(in oklch, var(--primary-400) 22%, transparent); }

  .safety-overlay {
    position: absolute;
    top: calc(var(--safe-top, 0px) + 92px);
    left: 50%;
    transform: translateX(-50%);
    z-index: var(--z-panel, 1000);
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
  .safety-chip.geofence { background: rgba(139, 92, 246, 0.18); border: 1px solid rgba(139, 92, 246, 0.35); color: var(--violet-600); }
  .safety-chip.autosos  { background: rgba(245, 158, 11, 0.18); border: 1px solid rgba(245, 158, 11, 0.35); color: var(--warning-600); }
  .safety-chip.checkin  { background: rgba(6, 182, 212, 0.18); border: 1px solid rgba(6, 182, 212, 0.35); color: #0891b2; }

  :global([data-theme="dark"]) .safety-chip.geofence { background: rgba(139, 92, 246, 0.22); color: var(--violet-400); }
  :global([data-theme="dark"]) .safety-chip.autosos  { background: rgba(245, 158, 11, 0.22); color: var(--warning-400); }
  :global([data-theme="dark"]) .safety-chip.checkin  { background: rgba(6, 182, 212, 0.22); color: var(--cyan-400); }
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
    border: 2px solid var(--user-color, var(--blue-500));
    pointer-events: none;
    animation: marker-pulse-ring 2s ease-out infinite;
  }

  /* TECHNIQUE 9/10: second ring — staggered delay, slightly larger inset for concentric effect */
  :global(.marker-ring-2) {
    inset: -12px;
    border-width: 1.5px;
    opacity: 0.65;
    animation-delay: 0.7s;
  }

  @keyframes marker-pulse-ring {
    0%   { transform: scale(1);   opacity: 0.5; }
    100% { transform: scale(1.8); opacity: 0; }
  }

  /* ── Navigation arrow marker (built by lib/mapPopup.js createNavArrowEl) ── */
  :global(.nav-arrow-marker) {
    z-index: 10 !important;
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  :global(.nav-arrow-inner) { position: relative; width: 48px; height: 48px; }
  :global(.nav-arrow-pulse) {
    position: absolute;
    inset: 0;
    border-radius: 50%;
    background: var(--primary-500-12);
    animation: nav-pulse 2s ease-out infinite;
    pointer-events: none;
  }
  :global(.nav-arrow-core) {
    position: absolute;
    inset: 8px;
    border-radius: 50%;
    background: var(--primary-500);
    border: 3px solid var(--surface-0);
    box-shadow: 0 2px 8px color-mix(in oklch, var(--primary-500) 50%, transparent);
    color: var(--surface-0);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  @keyframes nav-pulse {
    0% { transform: scale(1); opacity: 0.6; }
    100% { transform: scale(2.2); opacity: 0; }
  }
  @media (prefers-reduced-motion: reduce) {
    :global(.nav-arrow-pulse) { animation: none; opacity: 0.35; }
  }

  /* ── Meeting-point marker (createMeetingPointEl) — green flag ─────────── */
  :global(.meeting-point-marker) {
    display: flex;
    flex-direction: column;
    align-items: center;
    cursor: pointer;
  }
  :global(.meeting-point-marker .mp-flag) {
    background: linear-gradient(135deg, var(--success-500), var(--success-600));
    color: var(--surface-0);
    width: 34px;
    height: 34px;
    border-radius: 50%;
    border: 3px solid var(--surface-0);
    box-shadow: 0 2px 14px color-mix(in oklch, var(--success-500) 55%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  :global(.meeting-point-marker .mp-label) {
    background: color-mix(in oklch, var(--success-500) 92%, transparent);
    color: var(--surface-0);
    font-size: 10px;
    font-weight: 700;
    padding: 2px 7px;
    border-radius: 6px;
    margin-top: 3px;
    white-space: nowrap;
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    box-shadow: 0 1px 6px color-mix(in oklch, var(--gray-950) 18%, transparent);
  }
  :global(.meeting-point-marker .mp-stem) {
    width: 2px;
    height: 7px;
    background: var(--success-500);
    margin-top: 1px;
    border-radius: 0 0 2px 2px;
    opacity: 0.7;
  }

  /* ── Destination marker (createDestMarkerEl) — red pin during nav ──────── */
  :global(.dest-marker) {
    display: flex;
    flex-direction: column;
    align-items: center;
  }
  :global(.dest-marker .dest-pin) {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    background: linear-gradient(135deg, var(--danger-500), var(--danger-600));
    color: var(--surface-0);
    border: 3px solid var(--surface-0);
    box-shadow: 0 2px 16px color-mix(in oklch, var(--danger-500) 50%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  :global(.dest-marker .dest-stem) {
    width: 2px;
    height: 10px;
    background: var(--danger-500);
    margin: 0 auto;
    border-radius: 0 0 2px 2px;
    opacity: 0.6;
  }

  @media (prefers-reduced-motion: reduce) {
    :global(.marker-ring),
    :global(.marker-ring-2) {
      animation: none;
      opacity: 0.4;
    }
  }
</style>
