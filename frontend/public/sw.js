// ── Cache configuration ────────────────────────────────────────────────────
// v3: drop poisoned tile caches that hold an expired TileJSON index
const CACHE_VERSION = 'v3';
const STATIC_CACHE = `kinnect-static-${CACHE_VERSION}`;
const TILE_CACHE = `kinnect-tiles-${CACHE_VERSION}`;
const TILE_CACHE_MAX = 500;

// Install: precache the app shell so a fully offline launch still renders,
// then take control immediately. Hashed assets are cached on first fetch.
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(STATIC_CACHE)
      .then((cache) => cache.addAll(['/', '/favicon.svg', '/map-style.json']).catch(() => {}))
      .then(() => self.skipWaiting())
  );
});

// Activate: delete stale versioned caches so old assets are not served.
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then(keys =>
      Promise.all(
        keys
          .filter(k => k.startsWith('kinnect-') && k !== STATIC_CACHE && k !== TILE_CACHE)
          .map(k => caches.delete(k))
      )
    ).then(() => self.clients.claim())
  );
});

// Fetch: three strategies depending on the request origin/path.
self.addEventListener('fetch', (event) => {
  const { request } = event;
  // Only handle GET requests
  if (request.method !== 'GET') return;

  var url;
  try { url = new URL(request.url); } catch (_) { return; }

  // ── 1. OpenFreeMap: cache-first for tile FILES, network-first for indexes ─
  // Actual tiles/fonts/sprites are immutable at their URL → cache indefinitely.
  // BUT the TileJSON index (/planet) points at DATED tile paths that expire
  // server-side; caching it forever eventually 403s every tile → blank map.
  if (url.hostname === 'tiles.openfreemap.org') {
    var isTileFile = /\.(pbf|png|jpg|jpeg|webp)(\?|$)/.test(url.pathname);
    if (!isTileFile) {
      // TileJSON / index: network-first, cached copy only as offline fallback.
      event.respondWith(
        caches.open(TILE_CACHE).then(async function(cache) {
          try {
            var fresh = await fetch(request);
            if (fresh.ok) cache.put(request, fresh.clone());
            return fresh;
          } catch (_) {
            var fallback = await cache.match(request);
            if (fallback) return fallback;
            throw _;
          }
        })
      );
      return;
    }
    event.respondWith(
      caches.open(TILE_CACHE).then(async function(cache) {
        var cached = await cache.match(request);
        if (cached) return cached;
        var response = await fetch(request);
        if (response.ok) {
          var keys = await cache.keys();
          if (keys.length >= TILE_CACHE_MAX) {
            await cache.delete(keys[0]);
          }
          cache.put(request, response.clone());
        }
        return response;
      })
    );
    return;
  }

  // ── 2. Cache-first for local static assets (JS/CSS chunks) ──────────────
  // Vite outputs hashed filenames under /assets/ — safe to cache indefinitely.
  if (url.origin === self.location.origin && url.pathname.startsWith('/assets/')) {
    event.respondWith(
      caches.open(STATIC_CACHE).then(async function(cache) {
        var cached = await cache.match(request);
        if (cached) return cached;
        var response = await fetch(request);
        if (response.ok) cache.put(request, response.clone());
        return response;
      })
    );
    return;
  }

  // ── 3. Network-first for API calls — fresh data, offline fallback ────────
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(
      fetch(request).catch(function() { return caches.match(request); })
    );
    return;
  }

  // ── 4. Network-first app shell for navigations ───────────────────────────
  // Keeps the served HTML fresh (it references hashed asset URLs) while
  // guaranteeing the app still opens with the last-known shell when offline.
  if (request.mode === 'navigate' && url.origin === self.location.origin) {
    event.respondWith(
      caches.open(STATIC_CACHE).then(async function(cache) {
        try {
          var response = await fetch(request);
          if (response.ok) cache.put('/', response.clone());
          return response;
        } catch (_) {
          var cached = await cache.match('/');
          if (cached) return cached;
          throw _;
        }
      })
    );
    return;
  }
});

// ── Push notifications ─────────────────────────────────────────────────────
self.addEventListener('push', function(event) {
  var data = {};
  try { data = event.data.json(); } catch (_) { data = { title: 'Kinnect', body: event.data ? event.data.text() : 'New notification' }; }

  var title = data.title || 'Kinnect';
  var options = {
    body: data.body || '',
    icon: '/favicon.ico',
    badge: '/favicon.ico',
    tag: data.tag || 'kinnect-' + Date.now(),
    renotify: true,
    data: data
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', function(event) {
  event.notification.close();
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function(clientList) {
      for (var i = 0; i < clientList.length; i++) {
        if (clientList[i].url && clientList[i].focus) {
          return clientList[i].focus();
        }
      }
      if (self.clients.openWindow) {
        return self.clients.openWindow('/');
      }
    })
  );
});
