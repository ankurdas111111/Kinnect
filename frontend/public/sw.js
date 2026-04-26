// ── Cache configuration ────────────────────────────────────────────────────
const CACHE_VERSION = 'v1';
const STATIC_CACHE = `kinnect-static-${CACHE_VERSION}`;
const TILE_CACHE = `kinnect-tiles-${CACHE_VERSION}`;
const TILE_CACHE_MAX = 500;

// Install: open caches and take control immediately.
// Static assets (maplibre chunk, CSS, etc.) are cached on first fetch below.
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(STATIC_CACHE)
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

  // ── 1. Cache-first for OpenFreeMap tiles ────────────────────────────────
  // Tiles are immutable at a given z/x/y so cache indefinitely (up to limit).
  if (url.hostname === 'tiles.openfreemap.org') {
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
