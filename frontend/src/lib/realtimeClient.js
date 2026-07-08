import API_BASE from './env.js';

function buildWsUrl(base) {
  const origin = base || (typeof window !== 'undefined' ? window.location.origin : 'http://localhost:3000');
  // Ensure we use secure WebSocket (wss://) on HTTPS, regular ws:// on HTTP
  let wsOrigin = origin.replace(/^https/i, 'wss').replace(/^http(?!s)/i, 'ws');
  return wsOrigin.endsWith('/ws') ? wsOrigin : wsOrigin + '/ws';
}

function randomId() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) return crypto.randomUUID();
  return Date.now().toString(36) + '-' + Math.random().toString(36).slice(2);
}

class WsCompatSocket {
  constructor(base, options = {}) {
    this.base = base;
    this.url = buildWsUrl(base);
    this.ws = null;
    // Use the persistent clientId sent in auth as our socket ID —
    // the backend uses this same value as the socket/client ID.
    this.id = null;
    this.connected = false;
    this.listeners = {};
    this.ioListeners = {};
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = options.reconnectionAttempts ?? 50;
    this.reconnectDelay = options.reconnectionDelay ?? 200;
    this._heartbeatTimer = null;
    this.reconnectDelayMax = options.reconnectionDelayMax ?? 3000;
    this.randomizationFactor = options.randomizationFactor ?? 0.3;
    this.reconnection = options.reconnection !== false;
    this.manualDisconnect = false;
    this.io = {
      on: (event, cb) => {
        if (!this.ioListeners[event]) this.ioListeners[event] = [];
        this.ioListeners[event].push(cb);
      }
    };
    this._bfcacheClosed = false;
    // bfcache friendliness: an open WebSocket disqualifies a page from the
    // back/forward cache. pagehide with persisted=true means the page is
    // entering bfcache — close cleanly and reconnect on pageshow restore.
    // (persisted is false on real unloads and this never fires on mobile
    // app backgrounding, so the offline-grace flow is untouched.)
    if (typeof window !== 'undefined') {
      window.addEventListener('pagehide', (e) => {
        if (e.persisted && this.connected) {
          this._bfcacheClosed = true;
          this.disconnect();
        }
      });
      window.addEventListener('pageshow', (e) => {
        if (e.persisted && this._bfcacheClosed) {
          this._bfcacheClosed = false;
          this.connect();
        }
      });
    }
    if (options.autoConnect !== false) this.connect();
  }

  on(event, cb) {
    if (!this.listeners[event]) this.listeners[event] = [];
    this.listeners[event].push(cb);
  }

  off(event, cb) {
    if (!this.listeners[event]) return;
    if (!cb) { delete this.listeners[event]; return; }
    this.listeners[event] = this.listeners[event].filter((fn) => fn !== cb);
  }

  once(event, cb) {
    const wrapper = (...args) => {
      this.off(event, wrapper);
      cb(...args);
    };
    this.on(event, wrapper);
  }

  emit(event, data) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    this.ws.send(JSON.stringify({ e: event, d: data }));
  }

  connect() {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) return;
    this.manualDisconnect = false;
    this.open();
  }

  disconnect() {
    this.manualDisconnect = true;
    this._stopHeartbeat();
    if (this.ws) this.ws.close();
    this.connected = false;
  }

  open() {
    // Detach handlers from any existing WebSocket before replacing it.
    // This prevents an old socket's onclose from triggering another reconnect
    // after it gets evicted by the server when the new socket connects.
    if (this.ws) {
      this.ws.onopen = null;
      this.ws.onmessage = null;
      this.ws.onerror = null;
      this.ws.onclose = null;
    }
    this.ws = new WebSocket(this.url);
    this.ws.onopen = () => {
      this.connected = true;
      // id is set from the server's 'welcome' event, not here.
      if (this.reconnectAttempts > 0) this.fireIo('reconnect', this.reconnectAttempts);
      this.reconnectAttempts = 0;
      this.fire('connect');
      this._startHeartbeat();
    };
    this.ws.onmessage = (evt) => {
      try {
        const msg = JSON.parse(evt.data);
        if (msg?.e) this.fire(msg.e, msg.d);
      } catch (_) {}
    };
    this.ws.onerror = () => {
      this.fire('connect_error', new Error('websocket error'));
    };
    this.ws.onclose = () => {
      this._stopHeartbeat();
      this.connected = false;
      this.fire('disconnect', 'transport close');
      if (!this.manualDisconnect && this.reconnection && this.reconnectAttempts < this.maxReconnectAttempts) {
        this.scheduleReconnect();
      } else if (this.reconnectAttempts >= this.maxReconnectAttempts) {
        this.fireIo('reconnect_failed');
      }
    };
  }

  // Send a lightweight application-level heartbeat every 30s.
  // The Go server's ReadPump has a 90s read deadline — without this,
  // a silent connection (no tracking, app in background) gets closed
  // server-side after 90s of no messages, causing constant reconnects.
  _startHeartbeat() {
    this._stopHeartbeat();
    this._heartbeatTimer = setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        try { this.ws.send(JSON.stringify({ e: 'ping' })); } catch (_) {}
      }
    }, 30000);
  }

  _stopHeartbeat() {
    if (this._heartbeatTimer) {
      clearInterval(this._heartbeatTimer);
      this._heartbeatTimer = null;
    }
  }

  scheduleReconnect() {
    this.reconnectAttempts += 1;
    this.fireIo('reconnect_attempt', this.reconnectAttempts);
    const baseDelay = Math.min(this.reconnectDelay * Math.pow(1.5, this.reconnectAttempts - 1), this.reconnectDelayMax);
    const jitter = baseDelay * this.randomizationFactor * (Math.random() * 2 - 1);
    const wait = Math.max(0, baseDelay + jitter);
    setTimeout(() => {
      if (this.manualDisconnect) return;
      // Don't open a new socket if one is already open or connecting.
      if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) return;
      this.open();
    }, wait);
  }

  fire(event, ...args) {
    const cbs = this.listeners[event];
    if (!cbs) return;
    cbs.forEach((cb) => cb(...args));
  }

  fireIo(event, ...args) {
    const cbs = this.ioListeners[event];
    if (cbs) cbs.forEach((cb) => cb(...args));
    // Note: do NOT fire 'connect' here for 'reconnect' — ws.onopen already calls
    // this.fire('connect') immediately before fireIo('reconnect'), so firing it
    // again here would cause a double-connect and double handler execution.
  }
}

/**
 * Create a realtime socket — raw WebSocket against the Go backend's /ws
 * endpoint, wrapped in a Socket.IO-compatible surface (on/off/emit/io.on).
 */
export function createRealtimeSocket(options = {}) {
  return new WsCompatSocket(API_BASE || undefined, options);
}
