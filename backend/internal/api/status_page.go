package api

import (
	"net/http"
)

const statusPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Kinnect — Status</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

  :root {
    --bg:       #0f1117;
    --surface:  #1a1d27;
    --border:   #2a2d3a;
    --text:     #e2e8f0;
    --muted:    #64748b;
    --green:    #22c55e;
    --red:      #ef4444;
    --yellow:   #f59e0b;
    --blue:     #3b82f6;
    --purple:   #a78bfa;
    --radius:   12px;
  }

  body {
    background: var(--bg);
    color: var(--text);
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    min-height: 100vh;
    padding: 32px 20px;
  }

  .header {
    max-width: 720px;
    margin: 0 auto 32px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .logo-dot {
    width: 10px; height: 10px;
    border-radius: 50%;
    background: var(--green);
    box-shadow: 0 0 8px var(--green);
    animation: pulse 2s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0.4; }
  }

  h1 {
    font-size: 18px;
    font-weight: 600;
    letter-spacing: -0.3px;
  }

  .refresh-info {
    font-size: 12px;
    color: var(--muted);
  }

  .grid {
    max-width: 720px;
    margin: 0 auto;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 12px;
  }

  .card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 18px 20px;
    transition: border-color 0.2s;
  }

  .card:hover { border-color: #3a3d4a; }

  .card-label {
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--muted);
    margin-bottom: 10px;
  }

  .card-value {
    font-size: 26px;
    font-weight: 700;
    letter-spacing: -0.5px;
    line-height: 1;
  }

  .card-sub {
    font-size: 12px;
    color: var(--muted);
    margin-top: 6px;
  }

  /* Status pill */
  .pill {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    border-radius: 100px;
    font-size: 13px;
    font-weight: 600;
  }

  .pill-dot {
    width: 7px; height: 7px;
    border-radius: 50%;
  }

  .pill.ok     { background: rgba(34,197,94,.12); color: var(--green); }
  .pill.ok .pill-dot     { background: var(--green); }
  .pill.error  { background: rgba(239,68,68,.12);  color: var(--red); }
  .pill.error .pill-dot  { background: var(--red); }
  .pill.warn   { background: rgba(245,158,11,.12); color: var(--yellow); }
  .pill.warn .pill-dot   { background: var(--yellow); }
  .pill.none   { background: rgba(100,116,139,.12); color: var(--muted); }
  .pill.none .pill-dot   { background: var(--muted); }

  /* Wide cards */
  .card.wide {
    grid-column: 1 / -1;
  }

  /* DB pool bar */
  .pool-row {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-top: 4px;
  }

  .pool-bar-wrap {
    flex: 1;
    height: 6px;
    background: var(--border);
    border-radius: 3px;
    overflow: hidden;
  }

  .pool-bar {
    height: 100%;
    border-radius: 3px;
    background: var(--blue);
    transition: width 0.4s ease;
  }

  .pool-label {
    font-size: 11px;
    color: var(--muted);
    min-width: 60px;
    text-align: right;
  }

  /* Memory bars */
  .mem-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
  }

  .mem-key { font-size: 12px; color: var(--muted); width: 40px; }
  .mem-bar-wrap {
    flex: 1; height: 5px;
    background: var(--border);
    border-radius: 3px;
    overflow: hidden;
  }
  .mem-bar {
    height: 100%;
    border-radius: 3px;
    background: var(--purple);
    transition: width 0.4s ease;
  }
  .mem-val { font-size: 12px; color: var(--text); width: 40px; text-align: right; }

  /* Error state */
  .error-card {
    max-width: 720px;
    margin: 0 auto;
    background: rgba(239,68,68,.08);
    border: 1px solid rgba(239,68,68,.2);
    border-radius: var(--radius);
    padding: 20px;
    color: var(--red);
    font-size: 14px;
  }

  /* Updated timestamp */
  .timestamp {
    max-width: 720px;
    margin: 16px auto 0;
    font-size: 11px;
    color: var(--muted);
    text-align: right;
  }
</style>
</head>
<body>

<div class="header">
  <div class="logo">
    <div class="logo-dot" id="liveDot"></div>
    <h1>Kinnect Status</h1>
  </div>
  <span class="refresh-info">auto-refresh every 15s</span>
</div>

<div id="content"><div class="error-card">Loading…</div></div>
<div class="timestamp" id="ts"></div>

<script>
function pill(status) {
  const map = { ok: 'ok', error: 'error', not_configured: 'none', warn: 'warn' };
  const cls = map[status] || 'none';
  const label = status === 'not_configured' ? 'not set' : status;
  return '<span class="pill ' + cls + '"><span class="pill-dot"></span>' + label + '</span>';
}

function bar(pct, color) {
  color = color || 'var(--blue)';
  return '<div class="pool-bar-wrap"><div class="pool-bar" style="width:' + Math.min(pct,100) + '%;background:' + color + '"></div></div>';
}

function render(d) {
  const pool = d.db_pool || {};
  const mem  = d.memory_mb || {};
  const open = pool.open || 0;
  const inUse = pool.in_use || 0;
  const idle  = pool.idle  || 0;
  const maxPool = 10; // rough display max

  const heapMb  = mem.heap || 0;
  const sysMb   = mem.sys  || 0;
  const heapPct = Math.min((heapMb / Math.max(sysMb, 1)) * 100, 100);

  document.getElementById('content').innerHTML = '<div class="grid">' +

    // Status
    '<div class="card">' +
      '<div class="card-label">Status</div>' +
      '<div class="card-value">' + pill(d.status || 'error') + '</div>' +
      '<div class="card-sub">' + (d.env || '') + '</div>' +
    '</div>' +

    // DB
    '<div class="card">' +
      '<div class="card-label">Database</div>' +
      '<div class="card-value">' + pill(d.db || 'error') + '</div>' +
      '<div class="card-sub">' + (d.session_backend === 'redis' ? 'sessions → Redis' : 'sessions → Postgres') + '</div>' +
    '</div>' +

    // Redis
    '<div class="card">' +
      '<div class="card-label">Redis / Valkey</div>' +
      '<div class="card-value">' + pill(d.redis || 'not_configured') + '</div>' +
      '<div class="card-sub">' + (d.session_backend || '') + '</div>' +
    '</div>' +

    // Uptime
    '<div class="card">' +
      '<div class="card-label">Uptime</div>' +
      '<div class="card-value" style="font-size:20px;color:var(--text)">' + (d.uptime || '—') + '</div>' +
    '</div>' +

    // WS clients
    '<div class="card">' +
      '<div class="card-label">WS Clients</div>' +
      '<div class="card-value" style="color:var(--blue)">' + (d.ws_clients !== undefined ? d.ws_clients : '—') + '</div>' +
      '<div class="card-sub">connected now</div>' +
    '</div>' +

    // Rooms
    '<div class="card">' +
      '<div class="card-label">Rooms</div>' +
      '<div class="card-value" style="color:var(--purple)">' + (d.rooms !== undefined ? d.rooms : '—') + '</div>' +
      '<div class="card-sub">active rooms</div>' +
    '</div>' +

    // DB pool (wide)
    '<div class="card wide">' +
      '<div class="card-label">DB Connection Pool</div>' +
      '<div class="pool-row">' +
        '<span style="font-size:12px;color:var(--muted);width:48px">in use</span>' +
        bar((inUse / maxPool) * 100, 'var(--blue)') +
        '<span class="pool-label">' + inUse + ' / ' + open + '</span>' +
      '</div>' +
      '<div class="pool-row">' +
        '<span style="font-size:12px;color:var(--muted);width:48px">idle</span>' +
        bar((idle / maxPool) * 100, 'var(--border)') +
        '<span class="pool-label">' + idle + '</span>' +
      '</div>' +
    '</div>' +

    // Memory (wide)
    '<div class="card wide">' +
      '<div class="card-label">Memory</div>' +
      '<div class="mem-row">' +
        '<span class="mem-key">heap</span>' +
        bar(heapPct, 'var(--purple)') +
        '<span class="mem-val">' + heapMb + ' MB</span>' +
      '</div>' +
      '<div class="mem-row">' +
        '<span class="mem-key">sys</span>' +
        bar(100, 'var(--border)') +
        '<span class="mem-val">' + sysMb + ' MB</span>' +
      '</div>' +
      '<div class="card-sub" style="margin-top:10px">GC runs: ' + (mem.gc_runs || 0) + '</div>' +
    '</div>' +

  '</div>';
}

function load() {
  fetch('/health')
    .then(r => r.json())
    .then(d => {
      render(d);
      document.getElementById('ts').textContent = 'Last updated: ' + new Date().toLocaleTimeString();
      document.getElementById('liveDot').style.background = 'var(--green)';
    })
    .catch(() => {
      document.getElementById('content').innerHTML = '<div class="error-card">Could not reach /health — server may be down.</div>';
      document.getElementById('liveDot').style.background = 'var(--red)';
    });
}

load();
setInterval(load, 15000);
</script>
</body>
</html>`

// StatusPage serves the visual health dashboard at GET /status.
func StatusPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(statusPageHTML))
}
