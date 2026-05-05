<script>
  import { onMount, onDestroy } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { authUser, authLoading } from '../lib/stores/auth.js';
  import { socket, setupSocketHandlers, cancelReconnectBanner, setBanner as socketSetBanner } from '../lib/socket.js';
  import { banner, mySosActive } from '../lib/stores/sos.js';
  import { pendingIncomingRequests } from '../lib/stores/guardians.js';
  import { otherUsers, mySocketId, myLocation, tracking, focusUser, mapFlyTo, routeGeometry, walkDestination, mapChatRequest } from '../lib/stores/map.js';

  import AppLayout from '../components/layout/AppLayout.svelte';
  import Sidebar from '../components/layout/Sidebar.svelte';
  import Navbar from '../components/Navbar.svelte';
  import Banner from '../components/Banner.svelte';
  import MapView from '../components/Map.svelte';
  import UsersList from '../components/UsersList.svelte';
  import InfoPanel from '../components/InfoPanel.svelte';
  import AdminPanel from '../components/AdminPanel.svelte';
  import SharingPanel from '../components/SharingPanel.svelte';
  import SuperAdminPanel from '../components/SuperAdminPanel.svelte';
  import SettingsPanel from '../components/SettingsPanel.svelte';
  import SavedPlacesPanel from '../components/SavedPlacesPanel.svelte';
  import AlertOverlay from '../components/AlertOverlay.svelte';
  import BottomSheet from '../components/primitives/BottomSheet.svelte';
  import BottomTabBar from '../components/primitives/BottomTabBar.svelte';
  import MapFab from '../components/primitives/MapFab.svelte';
  import PlaceSearch from '../components/PlaceSearch.svelte';
  import MobileTopBar from '../components/primitives/MobileTopBar.svelte';
  import TrackingNowCard from '../components/primitives/TrackingNowCard.svelte';
  import OnboardingOverlay from '../components/OnboardingOverlay.svelte';
  import PulseButton from '../components/primitives/PulseButton.svelte';
  import SosFloat from '../components/SosFloat.svelte';
  import SecretChatPanel from '../components/SecretChatPanel.svelte';
  import HubSpotlight from '../components/HubSpotlight.svelte';
  import FeatureGuide from '../components/FeatureGuide.svelte';
  import { calculateDistance } from '../lib/tracking.js';
  import { GPSKalmanFilter, VelocityKalmanFilter } from '../lib/kalman.js';
  import { startMotionSensor, stopMotionSensor, getMotionState } from '../lib/motionSensor.js';
  import { recordFix, resetMetrics, trackingMetrics } from '../lib/stores/metrics.js';
  import { bufferPosition, clearBuffer, bufferSize } from '../lib/offlineBuffer.js';
  import { startGeo, stopGeo, warmUp, checkPermission, isNativePlatform } from '../lib/geoProvider.js';
  import { setupNotificationChannels, setAppActive } from '../lib/nativeNotifications.js';
  import { connectivityStore, setOnlineStatus, setSocketConnected, setBufferedCount } from '../lib/stores/connectivity.js';
  import { uiShellStore, setMobileTab, setSheetOpen } from '../lib/stores/uiShell.js';
  import { latencyMetrics } from '../lib/stores/latency.js';
  import { haptics } from '../lib/haptics.js';
  import { debounce } from '../lib/debounce.js';
  import { isIgnoringBatteryOptimizations, requestIgnoreBatteryOptimizations } from '../lib/batteryOptimization.js';
  import { rideShare } from '../lib/stores/rideShare.js';
  import * as trackingNotif from '../lib/trackingNotification.js';

  let activePanel = null;
  let sidebarTab = 'info';
  let sidebarCollapsed = false;
  let sosConfirmOpen = false;
  let batteryPromptOpen = false;

  // Feature 7: Panic Mode — read from localStorage (set in SettingsPanel)
  // Double-tap the SOS FAB to fire SOS instantly without the confirm modal.
  let sosFabLastTap = 0;
  function onSosFabClick() {
    if ($mySosActive) { socket.emit('cancelSOS'); return; }
    const panicMode = localStorage.getItem('kinnect_panic_mode') === 'true';
    const now = Date.now();
    if (panicMode && now - sosFabLastTap < 400) {
      // Double-tap in panic mode → fire immediately
      sosFabLastTap = 0;
      haptics.sos?.();
      socket.emit('triggerSOS', { reason: 'SOS', medicalCard: getMedicalSnapshot() });
    } else {
      sosFabLastTap = now;
      sosConfirmOpen = true;
    }
  }
  let secretChatPeer = null; // { id: string, name: string }

  // Map popup "Chat" button → open chat panel directly without going via action sheet
  $: if ($mapChatRequest) { secretChatPeer = $mapChatRequest; mapChatRequest.set(null); }

  /**
   * Detect Android device manufacturer from the WebView UA string.
   * Returns one of: 'miui' | 'coloros' | 'funtouch' | 'samsung' | 'generic'
   */
  function detectAndroidManufacturer() {
    const ua = (typeof navigator !== 'undefined' ? navigator.userAgent : '').toLowerCase();
    if (ua.includes('xiaomi') || ua.includes('redmi') || ua.includes('miui')) return 'miui';
    if (ua.includes('oppo') || ua.includes('realme')) return 'coloros';
    if (ua.includes('vivo')) return 'funtouch';
    if (ua.includes('samsung')) return 'samsung';
    return 'generic';
  }

  const batteryManufacturer = detectAndroidManufacturer();

  const BATTERY_INSTRUCTIONS = {
    miui: {
      brand: 'Xiaomi / Redmi',
      steps: [
        'Open Settings → Apps',
        'Find and tap Kinnect',
        'Tap Battery Saver',
        'Select "No restrictions"',
      ],
    },
    coloros: {
      brand: 'Oppo / Realme',
      steps: [
        'Open Settings → App Management',
        'Find and tap Kinnect',
        'Tap Battery',
        'Enable "Allow background activity"',
      ],
    },
    funtouch: {
      brand: 'Vivo',
      steps: [
        'Open Settings → Battery',
        'Tap "Background power consumption"',
        'Find Kinnect → set to "Unrestricted"',
      ],
    },
    samsung: {
      brand: 'Samsung',
      steps: [
        'Open Settings → Apps → Kinnect',
        'Tap Battery → select "Unrestricted"',
        'Then tap "Allow background activity"',
      ],
    },
    generic: {
      brand: null,
      steps: null,
    },
  };
  let isMobile = typeof window !== 'undefined' ? window.innerWidth < 768 : false;
  let mobileTab = 'track';
  let sheetOpen = false;
  let followMode = false;
  let meSubTab = 'info';
  let showOnboarding = false;
  let showFeatureGuide = false;
  let lastAcceptedFix = null;
  let lastEmittedFix = null;
  let lastEmitAt = 0;
  let lastCoarseNoticeAt = 0;
  let lastRawLat = null;
  let lastRawLng = null;
  let geoPermission = 'unknown';
  let _wakeLock = null;

  // F1: Battery level cache — read every 30s max; use synchronously in position payloads
  let _cachedBatteryPct = null;
  let _batteryReadAt = 0;

  async function _refreshBattery() {
    const now = Date.now();
    if (_cachedBatteryPct !== null && now - _batteryReadAt < 30000) return;
    try {
      const { Device } = await import(/* @vite-ignore */ '@capacitor/device');
      const info = await Device.getBatteryInfo();
      if (info && typeof info.batteryLevel === 'number') {
        _cachedBatteryPct = Math.round(info.batteryLevel * 100);
        _batteryReadAt = now;
        return;
      }
    } catch (_) {}
    try {
      const bat = await navigator.getBattery?.();
      if (bat) {
        _cachedBatteryPct = Math.round(bat.level * 100);
        _batteryReadAt = now;
      }
    } catch (_) {}
  }

  const gpsFilter = new GPSKalmanFilter({ Q: 3, R: 10 });
  const speedFilter = new VelocityKalmanFilter({ Q: 2, R: 25 });

  // Reads the emergency profile from localStorage and returns a sanitised snapshot.
  // Called at SOS trigger time so the data is always fresh.
  function getMedicalSnapshot() {
    try {
      const raw = localStorage.getItem('kinnect_emergency_profile');
      if (!raw) return null;
      const p = JSON.parse(raw);
      const hasData = p.bloodType || p.allergies || p.medications || p.emergencyName ||
                      p.emergencyContacts?.length || p.conditions;
      return hasData ? p : null;
    } catch { return null; }
  }

  $: if (!$authLoading && !$authUser) push('/login');
  $: isAdmin = $authUser && $authUser.role === 'admin';

  // Force-update persistent notification when ride status changes
  $: if ($tracking && $rideShare && trackingNotif.isActive()) {
    const rs = $rideShare;
    trackingNotif.showOrUpdate({
      visibleCount: $otherUsers.size,
      accuracy: $myLocation?.accuracy,
      rideActive: rs.active,
      rideDest: rs.dest,
      rideVehicle: rs.vehicle,
      rideEtaMins: rs.eta ? Math.max(0, Math.round((rs.eta - Date.now()) / 60000)) : null,
      force: true,
    });
  }
  $: rightPanelOpen = activePanel === 'users' || activePanel === 'superAdmin';
  $: sidebarOpen = !sidebarCollapsed;
  $: hasNotification = $pendingIncomingRequests.length > 0;
  $: mobileTab = $uiShellStore.mobileTab;
  $: sheetOpen = $uiShellStore.sheetOpen;

  // Fix #2: Reactive FAB bottom offset — lift SOS FAB above the BottomSheet
  // when the sheet is open on mobile so the button is never obscured.
  // The peek state reveals ~35vh of the sheet from the bottom; add that as clearance.
  $: fabBottomOffset = (isMobile && sheetOpen)
    ? `calc(var(--bottom-tab-height, 56px) + var(--safe-bottom, 0px) + var(--space-4) + min(35vh, 280px))`
    : `calc(var(--bottom-tab-height, 56px) + var(--safe-bottom, 0px) + var(--space-4))`;

  // Wire SOS state to global CSS app-state for full-app red tint
  $: {
    if (typeof document !== 'undefined') {
      if ($mySosActive) {
        document.documentElement.dataset.appState = 'sos';
      } else {
        delete document.documentElement.dataset.appState;
      }
    }
  }

  // When flying to a user on the map, auto-close panels/sheets so the map is visible
  $: if ($focusUser) {
    if (isMobile) {
      setSheetOpen(false);
      setMobileTab('track');
    }
    if (activePanel === 'users') {
      activePanel = null;
    }
  }

  function setPanel(panel) {
    activePanel = activePanel === panel ? null : panel;
  }

  function onNavbarToggle(e) {
    const panel = e.detail;
    if (['info', 'sharing', 'admin', 'places', 'settings'].includes(panel)) {
      if (sidebarTab === panel && !sidebarCollapsed) {
        sidebarCollapsed = true;
      } else {
        sidebarTab = panel;
        sidebarCollapsed = false;
      }
      activePanel = null;
    } else {
      if (activePanel === panel) activePanel = null;
      else activePanel = panel;
    }
  }

  function onSidebarTabChange(e) {
    sidebarTab = e.detail;
  }

  function onSidebarToggle(e) {
    sidebarCollapsed = e.detail;
  }

  function onMobileTabChange(e) {
    const tab = e.detail;
    setMobileTab(tab);
    activePanel = null;
    if (tab === 'track') {
      sidebarTab = 'info';
      setSheetOpen(true);
      return;
    }
    if (tab === 'people') sidebarTab = 'users';
    else if (tab === 'share') sidebarTab = 'sharing';
    else if (tab === 'safety') sidebarTab = 'admin';
    else if (tab === 'me') sidebarTab = 'info';
    setSheetOpen(true);
  }

  async function pushProfile() {
    const dt = isNativePlatform() ? 'Mobile' : (/Mobi|Android/i.test(navigator.userAgent || '') ? 'Mobile' : 'Desktop');
    let connectionQuality = 'Unknown';
    let batteryPct = null;

    // Battery via @capacitor/device (reliable) — navigator.getBattery is deprecated/removed
    if (isNativePlatform()) {
      try {
        const { Device } = await import(/* @vite-ignore */ '@capacitor/device');
        const info = await Device.getBatteryInfo();
        if (info.batteryLevel != null) batteryPct = Math.round(info.batteryLevel * 100);
      } catch (_) {}
    }

    socket.emit('profileUpdate', { batteryPct, deviceType: dt, connectionQuality });
  }

  function toggleTrackingAction() {
    if ($tracking) {
      haptics.confirm();
      stopTracking();
    } else {
      haptics.tap();
      startTracking();
    }
  }

  function applyFix(pos, forceEmit) {
    const { latitude: rawLat, longitude: rawLng, accuracy, speed: rawSpeed } = pos;
    const now = Date.now();
    if (!Number.isFinite(rawLat) || !Number.isFinite(rawLng)) return;
    // Fix D: reject zero/negative accuracy (some Android GPS drivers report 0 on cold start)
    if (!Number.isFinite(accuracy) || accuracy <= 0) return;
    if (lastAcceptedFix && accuracy > 20000) return;

    if (rawLat === lastRawLat && rawLng === lastRawLng && !forceEmit) return;
    lastRawLat = rawLat;
    lastRawLng = rawLng;

    // Fix B: reject coarse positions aggressively to prevent cell-tower fixes corrupting Kalman.
    // But emit a stale-position heartbeat every 30s so contacts know the last seen location.
    if (lastAcceptedFix && accuracy > 500) {
      if (lastEmittedFix && now - lastEmitAt >= 30000) {
        lastEmitAt = now;
        // F1: include cached battery level in stale heartbeat
        _refreshBattery().catch(() => {});
        const stalePayload = { latitude: lastEmittedFix.latitude, longitude: lastEmittedFix.longitude, speed: 0, formattedTime: new Date().toLocaleTimeString(), accuracy, timestamp: now, batteryPct: _cachedBatteryPct };
        if (socket.connected) {
          socket.emit('position', stalePayload);
          setBufferedCount(bufferSize());
        } else {
          bufferPosition(stalePayload);
          setBufferedCount(bufferSize());
        }
      }
      return;
    }
    if (lastAcceptedFix && accuracy > 200 && now - lastAcceptedFix.ts < 5000) return;

    if (lastAcceptedFix) {
      const jumpDistance = calculateDistance(lastAcceptedFix.latitude, lastAcceptedFix.longitude, rawLat, rawLng);
      const dtSec = Math.max((now - lastAcceptedFix.ts) / 1000, 1);
      const impliedKmh = (jumpDistance / dtSec) * 3.6;
      // Fix A: scale jump rejection with accuracy — poor-accuracy fixes are rejected at lower implied speeds
      if (impliedKmh > 150 && accuracy > 30) return;
      if (impliedKmh > 350) return; // absolute cap (faster than any ground vehicle)
    }

    // Sensor-fused speed estimation (Google Maps approach):
    //
    // 1. ZUPT (Zero Velocity Update) from accelerometer — if IMU says stationary,
    //    trust it over GPS completely. This is the most reliable signal.
    // 2. Velocity Kalman filter — smooths out GPS speed spikes over time.
    // 3. Position-implied speed as upper bound — GPS can't report faster than you moved.
    const rawKmh = rawSpeed != null && Number.isFinite(rawSpeed) ? rawSpeed * 3.6 : 0;
    const dtSec = lastAcceptedFix ? Math.max((now - lastAcceptedFix.ts) / 1000, 0.1) : 1;
    let impliedKmh = 0;
    if (lastAcceptedFix) {
      const movedM = calculateDistance(lastAcceptedFix.latitude, lastAcceptedFix.longitude, rawLat, rawLng);
      impliedKmh = (movedM / dtSec) * 3.6;
    }

    // ZUPT: accelerometer overrides everything when available
    const motion = getMotionState();
    let speed;
    if (motion.available && motion.stationary) {
      // IMU confirms stationary — zero velocity update, reset filter
      speedFilter.reset();
      speed = 0;
    } else {
      // Fuse GPS + position-implied; can't exceed what position actually shows
      const measuredKmh = rawKmh > 0 && impliedKmh > 0
        ? Math.min(rawKmh, impliedKmh * 1.5)
        : (impliedKmh > 0 ? impliedKmh : rawKmh);
      // If IMU confirms moving, increase filter responsiveness
      if (motion.available && motion.moving) speedFilter._Q = 8;
      else speedFilter._Q = 2;
      speed = speedFilter.filter(measuredKmh, dtSec);
    }

    gpsFilter.setSpeed(speed);
    let latitude, longitude, kalmanCorrectionM;
    if (!gpsFilter.isWarm && accuracy > 100) {
      // First fix is too coarse to seed the Kalman state — use raw coords for local display
      // but defer filter initialization until a sub-100m fix arrives.
      latitude = rawLat;
      longitude = rawLng;
      kalmanCorrectionM = 0;
    } else {
      const filtered = gpsFilter.filter(rawLat, rawLng, accuracy);
      latitude = filtered.lat;
      longitude = filtered.lng;
      kalmanCorrectionM = filtered.correctionM;
    }

    const formattedTime = new Date().toLocaleTimeString();
    lastAcceptedFix = { latitude, longitude, ts: now };
    myLocation.set({ latitude, longitude, speed, formattedTime, accuracy });
    recordFix({ accuracy, kalmanCorrectionM, filterWarm: gpsFilter.isWarm });
    if (accuracy > 150 && now - lastCoarseNoticeAt > 7000) {
      lastCoarseNoticeAt = now;
      // KR-003: use socketSetBanner so this goes through the timer-managed path and
      // doesn't overwrite or persist-clear SOS/reconnect banners unconditionally.
      socketSetBanner({ type: 'info', text: `Location is approximate (~${Math.round(accuracy)}m). Step outside for better accuracy.`, actions: [] }, 7000);
    }

    const timeSinceLastEmit = now - lastEmitAt;
    const distanceSinceLastEmit = lastEmittedFix
      ? calculateDistance(lastEmittedFix.latitude, lastEmittedFix.longitude, latitude, longitude)
      : Infinity;
    const moving = speed > 1;
    const shouldEmit = forceEmit ||
      !lastEmittedFix ||
      distanceSinceLastEmit >= 2 ||
      (moving && timeSinceLastEmit >= 250) ||
      timeSinceLastEmit >= 5000;

    if (shouldEmit) {
      lastEmittedFix = { latitude, longitude };
      lastEmitAt = now;
      // F1: refresh battery cache async (fire-and-forget); use cached value synchronously
      _refreshBattery().catch(() => {});
      const payload = { latitude, longitude, speed, formattedTime, accuracy, timestamp: now, batteryPct: _cachedBatteryPct };
      if (socket.connected) {
        socket.emit('position', payload);
        setBufferedCount(bufferSize());
      } else {
        bufferPosition(payload);
        setBufferedCount(bufferSize());
      }

      // Update persistent notification (throttled internally to every 30s)
      const rs = $rideShare;
      trackingNotif.showOrUpdate({
        visibleCount: $otherUsers.size,
        accuracy,
        rideActive: rs.active,
        rideDest: rs.dest,
        rideVehicle: rs.vehicle,
        rideEtaMins: rs.eta ? Math.max(0, Math.round((rs.eta - now) / 60000)) : null,
      });
    }
  }

  function startTracking() {
    if (geoPermission === 'denied') {
      banner.set({ type: 'info', text: "Kinnect needs location access to work. Turn it on in your device settings.", actions: [] });
      return;
    }
    if ($tracking) return;
    tracking.set(true);
    setBufferedCount(bufferSize());
    lastAcceptedFix = null;
    lastEmittedFix = null;
    lastEmitAt = 0;
    lastCoarseNoticeAt = 0;
    gpsFilter.reset();
    speedFilter.reset();

    startGeo(
      (pos, forceEmit) => applyFix(pos, forceEmit || !lastAcceptedFix),
      (err) => {
        if (err.code === 1) {
          const isDesktop = !/Android|iPhone|iPad|iPod/i.test(navigator.userAgent);
          const msg = isDesktop
            ? "Location access is off. Allow it in your browser settings, then try again."
            : "Location access is off. Enable it in your device settings for Kinnect.";
          banner.set({ type: 'info', text: msg, actions: [] });
          stopTracking();
          return;
        }
      }
    );

    // If no fix arrives within 12s (primary watchPosition timeout is 10s), replace the
    // "Starting..." banner so the user knows we're still searching, not frozen.
    setTimeout(() => {
      if ($tracking && !lastAcceptedFix) {
        banner.set({ type: 'info', text: "Finding your location...", actions: [] });
      }
    }, 12000);

    // Screen wake lock — keep display on while actively tracking so GPS stays alive
    if ('wakeLock' in navigator) {
      navigator.wakeLock.request('screen').then(lock => { _wakeLock = lock; }).catch(() => {});
    }

    // Start IMU sensor fusion for better speed accuracy on mobile
    startMotionSensor();

    // Show persistent Android notification
    trackingNotif.showOrUpdate({ visibleCount: $otherUsers.size, force: true });
  }

  function stopTracking() {
    stopGeo();
    stopMotionSensor();
    // Release wake lock when tracking stops
    if (_wakeLock) { _wakeLock.release().catch(() => {}); _wakeLock = null; }
    tracking.set(false);
    lastAcceptedFix = null;
    lastEmittedFix = null;
    lastEmitAt = 0;
    lastRawLat = null;
    lastRawLng = null;
    lastCoarseNoticeAt = 0;
    lastRawLat = null;
    lastRawLng = null;
    gpsFilter.reset();
    speedFilter.reset();
    resetMetrics();
    clearBuffer();
    setBufferedCount(0);

    // Dismiss persistent notification
    trackingNotif.dismiss();
  }

  function checkMobile() {
    isMobile = window.innerWidth < 768;
    if (isMobile) {
      if (!$uiShellStore.mobileTab) setMobileTab('track');
      if (!$uiShellStore.sheetOpen && $uiShellStore.mobileTab === 'track') setSheetOpen(true);
    }
  }

  // Pre-warm AudioContext on first user gesture (required for Safari)
  function prewarmAudio() {
    try {
      const ctx = new (window.AudioContext || window.webkitAudioContext)();
      ctx.resume().then(() => ctx.close()).catch(() => {});
    } catch (_) {}
    document.removeEventListener('pointerdown', prewarmAudio);
  }

  const debouncedCheckMobile = debounce(checkMobile, 80);

  onMount(async () => {
    if (!$authUser) { push('/login'); return; }
    setupSocketHandlers();
    pushProfile();
    const profileInterval = setInterval(pushProfile, 30000);
    checkMobile();
    window.addEventListener('resize', debouncedCheckMobile);
    setOnlineStatus(typeof navigator !== 'undefined' ? navigator.onLine : true);
    setSocketConnected(socket.connected);
    setBufferedCount(bufferSize());

    const onOnline = () => setOnlineStatus(true);
    const onOffline = () => setOnlineStatus(false);
    const onSocketConnect = () => {
      setSocketConnected(true);
      setTimeout(() => setBufferedCount(bufferSize()), 700);
    };
    const onSocketDisconnect = () => {
      setSocketConnected(false);
      setBufferedCount(bufferSize());
    };
    window.addEventListener('online', onOnline);
    window.addEventListener('offline', onOffline);
    socket.on('connect', onSocketConnect);
    socket.on('disconnect', onSocketDisconnect);

    // Pre-warm AudioContext on first touch/click (Safari requires user gesture)
    document.addEventListener('pointerdown', prewarmAudio, { once: true, passive: true });

    // Show onboarding for first-time users (no share code seen before)
    const onboardingKey = 'kinnect_onboarded_' + ($authUser?.userId || '');
    if (!localStorage.getItem(onboardingKey)) {
      setTimeout(() => { showOnboarding = true; }, 800);
    }

    // Fix C: await permission before tracking can start, so the denied-permission guard is reliable
    geoPermission = await checkPermission();

    // Warm up GPS hardware
    warmUp();

    // Fix H: guard against component unmounting before the dynamic import resolves
    let mounted = true;
    let appListenerCleanup = null;
    if (isNativePlatform()) {
      // Set up local notification channels once at startup
      setupNotificationChannels().catch(() => {});
      trackingNotif.setupTrackingChannel().then(() => trackingNotif.registerActions()).catch(() => {});

      // Wire up notification action buttons
      trackingNotif.onAction('share_ride', () => {
        // Bring app to foreground — action tap does this automatically
        setMobileTab('share');
      });
      trackingNotif.onAction('on_my_way', () => {
        socket.emit('onMyWay', {});
      });
      trackingNotif.onAction('pause', () => {
        stopTracking();
      });
      trackingNotif.onAction('reached_safely', () => {
        socket.emit('endRide', {});
        rideShare.set({ active: false, token: '', vehicle: '', vehicleType: '', dest: '', startedAt: 0, eta: 0 });
      });
      trackingNotif.onAction('sos', () => {
        socket.emit('triggerSOS', { reason: 'Triggered from notification', type: 'manual' });
      });

      // Prompt the user to disable battery optimization if not already done
      setTimeout(async () => {
        if (!mounted) return;
        const ignoring = await isIgnoringBatteryOptimizations();
        if (!ignoring) batteryPromptOpen = true;
      }, 2000);

      // @capacitor/network — accurate WiFi/cellular detection replacing browser online/offline
      import('@capacitor/network').then(({ Network }) => {
        if (!mounted) return;
        Network.getStatus().then(s => setOnlineStatus(s.connected)).catch(() => {});
        Network.addListener('networkStatusChange', s => {
          if (!mounted) return;
          setOnlineStatus(s.connected);
        });
      }).catch(() => {});

      import('@capacitor/app').then(({ App }) => {
        if (!mounted) return; // component already destroyed — skip adding listeners
        const listeners = [];
        listeners.push(App.addListener('appStateChange', ({ isActive }) => {
          setAppActive(isActive); // tells nativeNotifications whether to fire or skip
          if (isActive) {
            // Resumed from background — cancel any pending "Reconnecting…" banner before
            // reconnecting so normal background/foreground cycles are always silent.
            cancelReconnectBanner();
            if (!socket.connected) socket.connect();
            if ($tracking) {
              warmUp();
              if ('wakeLock' in navigator && !_wakeLock) {
                navigator.wakeLock.request('screen').then(lock => { _wakeLock = lock; }).catch(() => {});
              }
            }
          } else {
            // Going to background — release wake lock (OS reclaims it anyway)
            if (_wakeLock) { _wakeLock.release().catch(() => {}); _wakeLock = null; }
          }
        }));
        listeners.push(App.addListener('backButton', ({ canGoBack }) => {
          if (sheetOpen) { setSheetOpen(false); return; }
          if (activePanel) { activePanel = null; return; }
          if (canGoBack) { window.history.back(); }
        }));
        appListenerCleanup = () => {
          Promise.all(listeners).then(handles => handles.forEach(h => { if (h && h.remove) h.remove(); }));
        };
      }).catch(() => {});
    }

    return () => {
      mounted = false;
      clearInterval(profileInterval);
      stopTracking();
      window.removeEventListener('resize', debouncedCheckMobile);
      window.removeEventListener('online', onOnline);
      window.removeEventListener('offline', onOffline);
      socket.off('connect', onSocketConnect);
      socket.off('disconnect', onSocketDisconnect);
      document.removeEventListener('pointerdown', prewarmAudio);
      if (appListenerCleanup) appListenerCleanup();
    };
  });
</script>

<AppLayout {sidebarOpen} {rightPanelOpen}>
  <svelte:fragment slot="topBar">
    <MobileTopBar
      activeTab={mobileTab}
      trackingActive={$tracking}
      {hasNotification}
      lastAccuracy={$trackingMetrics.lastAccuracy}
      latencyMs={$latencyMetrics.avgE2eMs}
      isOnline={$connectivityStore.isOnline}
      socketConnected={$connectivityStore.socketConnected}
      bufferedCount={$connectivityStore.bufferedCount}
      on:openMe={() => onMobileTabChange({ detail: 'me' })}
    />
  </svelte:fragment>

  <svelte:fragment slot="navbar">
    <Navbar
      {isAdmin}
      activePanel={activePanel || sidebarTab}
      on:togglePanel={onNavbarToggle}
      on:toggleTracking={toggleTrackingAction}
      isTracking={$tracking}
    />
  </svelte:fragment>

  <svelte:fragment slot="sidebar">
    <Sidebar
      activeTab={sidebarTab}
      {isAdmin}
      collapsed={sidebarCollapsed}
      on:tabChange={onSidebarTabChange}
      on:toggle={onSidebarToggle}
    >
      {#if sidebarTab === 'info'}
        <InfoPanel embedded={true} />
      {:else if sidebarTab === 'sharing'}
        <SharingPanel embedded={true} />
      {:else if sidebarTab === 'admin'}
        <AdminPanel embedded={true} />
      {:else if sidebarTab === 'places'}
        <SavedPlacesPanel embedded={true} />
      {:else if sidebarTab === 'settings'}
        <SettingsPanel embedded={true} on:openGuide={() => showFeatureGuide = true} />
      {/if}
    </Sidebar>
  </svelte:fragment>

  <svelte:fragment slot="map">
    <MapView {followMode} />
    <div class="place-search-overlay">
      <PlaceSearch
        on:select={e => mapFlyTo.set({ lat: e.detail.lat, lng: e.detail.lng, zoom: 15 })}
        on:route={e => routeGeometry.set(e.detail.geometry)}
        on:clearRoute={() => routeGeometry.set(null)}
        on:setDestination={e => walkDestination.set(e.detail)}
      />
    </div>
  </svelte:fragment>

  <svelte:fragment slot="banner">
    <Banner />
  </svelte:fragment>

  <svelte:fragment slot="rightPanel">
    {#if activePanel === 'users'}
      <UsersList on:close={() => activePanel = null} on:secretChat={(e) => { activePanel = null; secretChatPeer = e.detail; }} />
    {:else if activePanel === 'superAdmin' && isAdmin}
      <SuperAdminPanel on:close={() => activePanel = null} />
    {/if}
  </svelte:fragment>

  <svelte:fragment slot="bottomSheet">
    <BottomSheet
      open={sheetOpen}
      title={mobileTab === 'track' ? 'Map' : mobileTab === 'people' ? 'People' : mobileTab === 'share' ? 'Connect' : mobileTab === 'safety' ? 'Safety' : 'Me'}
      on:close={() => {
        setSheetOpen(false);
      }}
    >
      {#if mobileTab === 'track'}
        <TrackingNowCard
          location={$myLocation}
          trackingActive={$tracking}
          bufferedCount={$connectivityStore.bufferedCount}
          socketConnected={$connectivityStore.socketConnected}
          on:toggleTracking={toggleTrackingAction}
          on:centerOnMe={() => focusUser.set('__self__')}
          on:toggleFollow={() => (followMode = !followMode)}
        />
      {:else if mobileTab === 'me'}
        <div class="page-nav-row">
          <button class="page-nav-btn" on:click={() => push('/dashboard')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
            Dashboard
          </button>
          <button class="page-nav-btn" on:click={() => push('/activity')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
            Activity
          </button>
          <button class="page-nav-btn" on:click={() => push('/emergency')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
            Emergency
          </button>
          <button class="page-nav-btn" on:click={() => push('/checkins')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
            Check-ins
          </button>
          <button class="page-nav-btn" on:click={() => push('/replay')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 .49-4.5"/></svg>
            Route History
          </button>
        </div>
        <div class="spatial-subtabs">
          <button class="spatial-subtab" class:active={meSubTab === 'info'} on:click={() => meSubTab = 'info'}>Status</button>
          <button class="spatial-subtab" class:active={meSubTab === 'places'} on:click={() => meSubTab = 'places'}>Places</button>
          <button class="spatial-subtab" class:active={meSubTab === 'settings'} on:click={() => meSubTab = 'settings'}>Settings</button>
        </div>
        {#if meSubTab === 'info'}
          <InfoPanel embedded={true} />
        {:else if meSubTab === 'places'}
          <SavedPlacesPanel embedded={true} />
        {:else if meSubTab === 'settings'}
          <SettingsPanel embedded={true} on:openGuide={() => showFeatureGuide = true} />
        {/if}
      {:else if mobileTab === 'share'}
        <SharingPanel embedded={true} />
      {:else if mobileTab === 'safety'}
        <div class="safety-quick-actions">
          <button
            class="btn"
            class:btn-danger={!$mySosActive}
            class:btn-secondary={$mySosActive}
            on:click={() => { if ($mySosActive) { haptics.sosCancelled(); socket.emit('cancelSOS'); } else { haptics.warning(); sosConfirmOpen = true; } }}
          >
            {$mySosActive ? 'Cancel SOS' : 'Trigger SOS'}
          </button>
          <button class="btn btn-secondary" on:click={() => socket.emit('checkInAck')}>I'm OK</button>
        </div>
        <div class="page-nav-row">
          <button class="page-nav-btn" on:click={() => push('/emergency')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
            Emergency Profile
          </button>
          <button class="page-nav-btn" on:click={() => push('/checkins')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
            Check-in Schedule
          </button>
        </div>
        <AdminPanel embedded={true} />
      {:else if mobileTab === 'people'}
        <UsersList embedded={true} on:secretChat={(e) => { setSheetOpen(false); secretChatPeer = e.detail; }} />
      {/if}
    </BottomSheet>
  </svelte:fragment>

  <svelte:fragment slot="bottomTabs">
    <BottomTabBar
      activeTab={mobileTab}
      {isAdmin}
      isTracking={$tracking}
      {hasNotification}
      on:tabChange={onMobileTabChange}
    />
  </svelte:fragment>

  <svelte:fragment slot="overlay">
    <AlertOverlay />

    <!-- Secret encrypted chat overlay -->
    {#if secretChatPeer}
      <SecretChatPanel
        peerId={secretChatPeer.id}
        peerName={secretChatPeer.name}
        onClose={() => secretChatPeer = null}
      />
    {/if}

    <!-- Persistent emergency float card — appears above SOS FAB when a network member has active SOS + medical data -->
    <SosFloat />

    <!-- First-run Hub discovery coach mark (desktop only, shows once) -->
    <HubSpotlight />

    <!-- SOS FAB — always visible bottom-left.
         Fix #2: inline style overrides the CSS @media bottom value on mobile so the FAB
         animates out of the way when the BottomSheet is at peek state. -->
    <button
      class="sos-fab"
      class:active={$mySosActive}
      style={isMobile ? `bottom: ${fabBottomOffset}` : undefined}
      on:click={onSosFabClick}
      aria-label={$mySosActive ? 'Cancel SOS' : 'Send SOS'}
    >
      {#if $mySosActive}
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      {:else}
        <span class="sos-text">SOS</span>
      {/if}
    </button>

    <!-- Pulse Check-In FAB -->
    <PulseButton />

    <!-- FAB cluster — bottom-right: track + center + follow -->
    <div class="fab-wrapper" class:fab-wrapper--mobile={isMobile}>
      <MapFab
        isTracking={$tracking}
        {followMode}
        on:toggleTracking={toggleTrackingAction}
        on:centerOnMe={() => focusUser.set('__self__')}
        on:toggleFollow={() => followMode = !followMode}
      />
    </div>

    <!-- SOS Confirmation Modal -->
    {#if sosConfirmOpen}
      <div class="sos-confirm-backdrop" on:click|self={() => sosConfirmOpen = false} on:keydown={(e) => { if (e.key === 'Escape') sosConfirmOpen = false; }} role="dialog" aria-modal="true" aria-label="Confirm SOS" tabindex="-1">
        <div class="sos-confirm-card-spatial">
          <div class="sos-icon-ring">
            <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#ef4444" stroke-width="2.5"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
          </div>
          <h3 class="sos-confirm-title">Send an SOS to your family?</h3>
          <p class="sos-confirm-desc">Everyone connected to you will be notified right away. They'll see your live location until you're safe.</p>
          <div class="sos-confirm-actions">
            <button class="btn btn-ghost sos-cancel-btn" on:click={() => sosConfirmOpen = false}>Cancel</button>
            <button class="btn btn-danger sos-send-btn" on:click={() => { haptics.sos(); socket.emit('triggerSOS', { reason: 'SOS', medicalCard: getMedicalSnapshot() }); sosConfirmOpen = false; }}>Yes, send SOS</button>
          </div>
        </div>
      </div>
    {/if}

    <!-- Battery Optimization Prompt -->
    {#if batteryPromptOpen}
      <div class="battery-prompt-backdrop" on:click|self={() => batteryPromptOpen = false} on:keydown={(e) => { if (e.key === 'Escape') batteryPromptOpen = false; }} role="dialog" aria-modal="true" aria-label="Allow background access" tabindex="-1">
        <div class="battery-prompt-card">
          <div class="battery-prompt-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#f59e0b" stroke-width="2"><rect x="2" y="7" width="16" height="10" rx="2"/><path d="M22 11v2"/><path d="M6 11v2"/><path d="M10 11v2"/></svg>
          </div>
          <h3 class="battery-prompt-title">Allow Background Access</h3>
          {#if BATTERY_INSTRUCTIONS[batteryManufacturer].steps}
            {@const info = BATTERY_INSTRUCTIONS[batteryManufacturer]}
            <p class="battery-prompt-brand">{info.brand} detected</p>
            <ol class="battery-steps-list">
              {#each info.steps as step}
                <li>{step}</li>
              {/each}
            </ol>
          {:else}
            <p class="battery-prompt-desc">For Kinnect to share your location when the screen is off or the app is minimized, allow it to run unrestricted in the background.</p>
          {/if}
          <div class="battery-prompt-actions">
            <button class="btn battery-skip-btn" on:click={() => batteryPromptOpen = false}>Not now</button>
            <button class="btn battery-allow-btn" on:click={async (e) => {
              e.target.disabled = true;
              try { await requestIgnoreBatteryOptimizations(); } finally { batteryPromptOpen = false; }
            }}>Open Battery Settings</button>
          </div>
        </div>
      </div>
    {/if}

    <!-- First-run onboarding -->
    <OnboardingOverlay
      visible={showOnboarding}
      on:requestPermission={startTracking}
      on:dismiss={() => {
        showOnboarding = false;
        const key = 'kinnect_onboarded_' + ($authUser?.userId || '');
        localStorage.setItem(key, '1');
        // Show feature guide for first-time users after onboarding
        if (!localStorage.getItem('kinnect_guide_seen')) {
          setTimeout(() => { showFeatureGuide = true; }, 600);
        }
      }}
    />

    <!-- Feature Guide overlay (first run + accessible from Settings) -->
    <FeatureGuide bind:open={showFeatureGuide} />
  </svelte:fragment>
</AppLayout>

<style>
  /* spatial-subtabs styles are in global.css */

  /* ── Page navigation row ─────────────────────────────────────────────────── */
  .page-nav-row {
    display: flex;
    flex-shrink: 0;
    gap: 8px;
    margin-bottom: 12px;
    overflow-x: auto;
    overflow-y: hidden;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
    padding-bottom: 2px;
  }
  .page-nav-row::-webkit-scrollbar { display: none; }

  /* Fix #1: raised touch target to 44px for iOS HIG compliance.
     Before: display:inline-flex; padding:8px 12px; (no min-height → ~28-32px tall)
     After:  display:flex; justify-content:center; min-height:44px; padding:8px 12px */
  .page-nav-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 8px 12px;
    min-height: 44px;
    border-radius: var(--radius-lg);
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid rgba(255, 255, 255, 0.10);
    color: var(--text-secondary);
    font-family: var(--font-display);
    font-size: 12px;
    font-weight: 600;
    white-space: nowrap;
    cursor: pointer;
    flex-shrink: 0;
    transition: background var(--duration-fast) var(--ease-out), color var(--duration-fast) var(--ease-out), transform 120ms var(--ease-spring);
    -webkit-tap-highlight-color: transparent;
  }
  .page-nav-btn:hover {
    background: rgba(99, 102, 241, 0.14);
    border-color: rgba(99, 102, 241, 0.30);
    color: var(--primary-300);
  }
  .page-nav-btn:active {
    transform: scale(0.93);
    transition-duration: 70ms;
  }

  .safety-quick-actions {
    display: flex;
    gap: 8px;
    margin-bottom: 10px;
    flex-wrap: wrap;
  }

  .safety-quick-actions .btn {
    min-height: 44px;
  }

  /* ── SOS FAB — 3D physical button ─────────────────────────────────────── */
  .sos-fab {
    position: fixed;
    bottom: var(--space-4);
    left: var(--space-4);
    width: 52px;
    height: 52px;
    border-radius: 50%;
    background: linear-gradient(135deg, #ef4444 0%, #b91c1c 100%);
    color: white;
    border: 3px solid rgba(255,255,255,0.85);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    /* 3D raised emergency button */
    box-shadow:
      0 6px 20px rgba(220, 38, 38, 0.50),
      0 2px 6px rgba(220, 38, 38, 0.35),
      inset 0 2px 4px rgba(255, 255, 255, 0.20),
      inset 0 -3px 6px rgba(0, 0, 0, 0.20);
    z-index: calc(var(--z-panel, 100) + 2);
    transform-style: preserve-3d;
    /* Fix #2: bottom added to transition list so FAB slides smoothly when
       sheet opens/closes. Before: only transform, box-shadow, background. */
    transition:
      bottom 300ms ease,
      transform var(--duration-3d) var(--ease-3d-spring),
      box-shadow var(--duration-3d) var(--ease-3d-out),
      background 0.2s ease;
    isolation: isolate;
  }
  .sos-fab:hover {
    transform: perspective(600px) translateY(-3px) translateZ(6px) scale(1.08);
    box-shadow:
      0 10px 32px rgba(220, 38, 38, 0.60),
      0 4px 10px rgba(220, 38, 38, 0.40),
      0 0 0 4px rgba(220, 38, 38, 0.15),
      inset 0 2px 4px rgba(255, 255, 255, 0.22),
      inset 0 -3px 6px rgba(0, 0, 0, 0.18);
  }
  .sos-fab:active {
    transform: perspective(600px) translateZ(-6px) scale(0.92);
    box-shadow:
      0 1px 6px rgba(220, 38, 38, 0.40),
      inset 0 3px 8px rgba(0, 0, 0, 0.25);
  }
  .sos-fab .sos-text {
    font-size: 14px;
    font-weight: 900;
    letter-spacing: 1px;
    line-height: 1;
  }
  .sos-fab.active {
    background: #991b1b;
  }
  /* Ripple ring via pseudo-element — uses transform+opacity (composited, no paint) */
  .sos-fab::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: 50%;
    background: rgba(239, 68, 68, 0.5);
    transform: scale(1);
    opacity: 0;
    pointer-events: none;
    will-change: transform, opacity;
  }
  .sos-fab.active::after {
    animation: sos-ring 1.2s ease-out infinite;
  }
  @keyframes sos-ring {
    0%   { transform: scale(1);   opacity: 0.6; }
    100% { transform: scale(1.9); opacity: 0; }
  }

  /* CSS baseline for mobile bottom — the inline style binding (fabBottomOffset)
     overrides this reactively when isMobile && sheetOpen. */
  @media (max-width: 767px) {
    .sos-fab {
      bottom: calc(var(--bottom-tab-height, 56px) + var(--safe-bottom, 0px) + var(--space-4));
    }
  }

  /* Desktop/tablet: keep SOS FAB outside sidebar panel area */
  :global(.app-layout.sidebar-open:not(.mobile)) .sos-fab {
    left: calc(var(--sidebar-width, 400px) + var(--space-4));
  }
  :global(.app-layout.tablet.sidebar-open) .sos-fab {
    left: calc(var(--sidebar-tablet, 320px) + var(--space-4));
  }
  :global(.app-layout.sidebar-closed:not(.mobile)) .sos-fab {
    left: calc(var(--sidebar-collapsed, 56px) + var(--space-4));
  }

  /* ── MapFab wrapper ───────────────────────────────────────────────────── */
  .fab-wrapper {
    position: fixed;
    bottom: var(--space-6);
    right: var(--space-4);
    z-index: calc(var(--z-panel, 100) + 1);
  }

  .fab-wrapper--mobile {
    bottom: calc(var(--bottom-tab-height, 56px) + var(--safe-bottom, 0px) + var(--space-4));
  }

  /* ── SOS Confirmation Modal ───────────────────────────────────────────── */
  .sos-confirm-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.65);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    z-index: calc(var(--z-panel, 100) + 10);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4);
    animation: fade-in 0.15s ease;
  }
  /* sos-confirm-card-spatial, .sos-icon-ring styles are in global.css */
  .sos-confirm-title {
    font-size: 20px;
    font-weight: 800;
    color: white;
    margin: 0 0 8px;
    letter-spacing: -0.02em;
  }
  .sos-confirm-desc {
    font-size: 13px;
    color: rgba(255, 255, 255, 0.65);
    line-height: 1.55;
    margin: 0 0 24px;
  }
  .sos-confirm-actions {
    display: flex;
    gap: 10px;
    justify-content: center;
  }
  .sos-cancel-btn {
    flex: 1;
    padding: 11px 16px;
    border-radius: var(--radius-lg);
    font-weight: 600;
    font-size: 14px;
    background: rgba(255, 255, 255, 0.07);
    color: rgba(255, 255, 255, 0.70);
    border: 1px solid rgba(255, 255, 255, 0.10);
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out);
  }
  .sos-cancel-btn:hover { background: rgba(255, 255, 255, 0.12); }
  .sos-send-btn {
    flex: 1;
    padding: 11px 16px;
    border-radius: var(--radius-lg);
    font-weight: 800;
    font-size: 14px;
    letter-spacing: 0.01em;
    background: #dc2626;
    color: white;
    border: none;
    cursor: pointer;
    box-shadow: 0 4px 16px rgba(220, 38, 38, 0.50);
    transition: background var(--duration-fast) var(--ease-out), box-shadow var(--duration-fast) var(--ease-out);
  }
  .sos-send-btn:hover { background: #ef4444; box-shadow: 0 6px 24px rgba(239, 68, 68, 0.55); }
  @keyframes fade-in { from { opacity: 0; } to { opacity: 1; } }

  /* ── Battery Optimization Prompt ─────────────────────────────────────── */
  .battery-prompt-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.65);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    z-index: calc(var(--z-panel, 100) + 10);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4);
    animation: fade-in 0.2s ease;
  }
  .battery-prompt-card {
    background: #1a1a2e;
    border: 1px solid rgba(245, 158, 11, 0.25);
    border-radius: var(--radius-xl, 20px);
    padding: 28px 24px 24px;
    max-width: 340px;
    width: 100%;
    text-align: center;
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.6);
  }
  .battery-prompt-icon {
    width: 60px;
    height: 60px;
    border-radius: 50%;
    background: rgba(245, 158, 11, 0.12);
    border: 1px solid rgba(245, 158, 11, 0.25);
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0 auto 16px;
  }
  .battery-prompt-title {
    font-size: 18px;
    font-weight: 800;
    color: white;
    margin: 0 0 8px;
    letter-spacing: -0.02em;
  }
  .battery-prompt-desc {
    font-size: 13px;
    color: rgba(255, 255, 255, 0.60);
    line-height: 1.55;
    margin: 0 0 24px;
  }
  .battery-prompt-brand {
    font-size: 11px;
    font-weight: 700;
    color: #f59e0b;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin: 0 0 10px;
  }
  .battery-steps-list {
    text-align: left;
    font-size: 13px;
    color: rgba(255, 255, 255, 0.75);
    line-height: 1.6;
    margin: 0 0 24px;
    padding-left: 20px;
  }
  .battery-steps-list li {
    margin-bottom: 4px;
  }
  .battery-prompt-actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .battery-allow-btn {
    padding: 12px 16px;
    border-radius: var(--radius-lg);
    font-weight: 700;
    font-size: 14px;
    background: #f59e0b;
    color: #0a0a15;
    border: none;
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out);
  }
  .battery-allow-btn:hover { background: #fbbf24; }
  .battery-skip-btn {
    padding: 10px 16px;
    border-radius: var(--radius-lg);
    font-weight: 500;
    font-size: 13px;
    background: transparent;
    color: rgba(255, 255, 255, 0.45);
    border: none;
    cursor: pointer;
  }
  .battery-skip-btn:hover { color: rgba(255, 255, 255, 0.70); }

  /* ── Place search overlay on map ─────────────────────────────────── */
  /* Fix #3: wrapper changed from pointer-events:auto to pointer-events:none
     so the map canvas under the empty areas around the search widget stays
     interactive. Child content (.ps-wrap, .nav-hud) restores pointer-events.
     Before: pointer-events: auto on wrapper (entire overlay ate touch events)
     After:  pointer-events: none on wrapper; auto restored on direct children */
  .place-search-overlay {
    position: absolute;
    top: calc(var(--safe-top, 0px) + 52px);
    left: 50%;
    transform: translateX(-50%);
    z-index: 15;
    pointer-events: none;
  }
  .place-search-overlay :global(.ps-wrap),
  .place-search-overlay :global(.nav-hud) {
    pointer-events: auto;
  }
  @media (max-width: 767px) {
    .place-search-overlay {
      /* Fixed so it escapes layout-map overflow:hidden.
         Top is positioned BELOW the MobileTopBar which is safe-top + ~92px tall.
         Using 108px (same as layout-map padding-top) + 8px breathing room. */
      position: fixed;
      top: calc(var(--safe-top, 0px) + 116px);
      left: 10px;
      right: 10px;
      transform: none;
      z-index: 2500; /* above mobile navbar (2000) */
    }
    .place-search-overlay :global(.ps-wrap) {
      width: 100%;
    }
  }
</style>
