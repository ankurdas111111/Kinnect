# Kinnect

**Kin + Connect** — Real-time GPS location sharing for families.

Built with **Go**, **Svelte 5**, **MapLibre GL**, **PostgreSQL**, **WebSockets**, and **Capacitor**.

## Features

- **Real-time location sharing** via rooms, contacts, and guardian/ward relationships
- **Manual + auto SOS** — no-movement, hard-stop, and geofence breach triggers
- **Acknowledgment pipeline** with tokenized emergency watch links
- **Check-in monitoring** with overdue alerts to guardians/managers
- **Guardian/ward permissions** with time-limited roles and majority-vote room admin
- **Live sharing links** — 1h, 6h, 24h, 48h, or permanent; revocable
- **Emergency profile** — medical info, contacts, escalation plan for responders
- **Secret chat** — end-to-end encrypted messaging
- **Route history & playback** with saved places and place alerts
- **Offline position buffering** with batch replay on reconnect
- **Kalman-filtered GPS** with speed-adaptive smoothing
- **Smooth marker interpolation**, dark mode, responsive UI, mobile-ready (Capacitor)

## Architecture

| Layer | Tech |
|---|---|
| Backend | Go (`backend/`, module `kinnect-v3`) — HTTP API + WebSocket hub |
| Database | PostgreSQL via `pgx/v5` (schema auto-initialized on startup) |
| Realtime | Raw WebSockets (`github.com/coder/websocket`) |
| Frontend | Svelte 5 + Vite 8, hash-routed SPA (`svelte-spa-router`) |
| Maps | MapLibre GL (three.js powers the landing hero) |
| Monitoring | Prometheus metrics on a separate internal port |
| Mobile | Capacitor 8 (Android + iOS) |
| Cache | In-memory; optional Redis |

## Prerequisites

- **Go** 1.26+
- **Node.js** 20.19+ or 22.12+ (Vite 8 requirement)
- **PostgreSQL** (any recent version) — **required**; the backend exits if unreachable
- **Redis** (optional — distributed cache/scaling)

## Local Development

```bash
# 1. Install dependencies (root AND frontend — two separate package.json files)
npm install
cd frontend && npm install && cd ..

# 2. Configure environment
cp .env.example .env
# Edit .env — set DATABASE_URL and SESSION_SECRET

# 3. Start everything (recommended)
npm run dev
# Builds the frontend, starts the Go backend on :3001,
# monitoring on :9090, and the Vite dev server on :5173

# — or run the pieces separately —
npm run dev:be    # Go backend on :3001 (loads .env)
npm run dev:fe    # Vite dev server on :5173 with HMR (proxies /api + /ws to :3001)

# 4. Open the app
# App:     http://localhost:5173
# Landing: http://localhost:5173/#/landing
```

> **Windows note:** if Vite fails with `Cannot find native binding` (rolldown), npm's
> [optional-deps bug](https://github.com/npm/cli/issues/4828) skipped the native binary.
> Fix: `cd frontend && npm i --no-save @rolldown/binding-win32-x64-msvc`
> (match the version of `node_modules/rolldown`), or delete `node_modules` +
> `package-lock.json` and reinstall.

### Production-like local run (single port)

```bash
npm start
# Builds the frontend, compiles the Go server, and runs it —
# serves API + built frontend from one port (default :3000)
```

## All Commands

| Command | Description |
|---|---|
| `npm run dev` | Unified dev: frontend build + Go backend (:3001) + monitoring (:9090) + Vite (:5173) |
| `npm run dev:be` | Go backend only (port 3001, loads `.env`) |
| `npm run dev:fe` | Vite frontend dev server (port 5173, HMR) |
| `npm run build` | Production frontend build to `frontend/dist/` |
| `npm run build:backend` | Compile the Go server binary |
| `npm start` | Build frontend + compile & run Go server (single port) |
| `npm test` | Backend integration tests (`backend/test/integration_test.sh`) |
| `npm run bench` | WebSocket load benchmark |
| `npm run lint:colors` | Enforce design tokens (no raw colors) |
| `npm run check:contrast` | WCAG contrast checks on theme tokens |
| `npm run build:android:prod -- <url>` | Build Android APK with backend URL |
| `npm run build:ios:prod -- <url>` | Build iOS with backend URL |
| `npm run run:android` / `run:ios` | Run on device/emulator |
| `npm run apk:release` / `aab:release` | Signed Android release artifacts |

## Environment Variables

Copy `.env.example` to `.env` and configure:

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | **Yes** | PostgreSQL connection string |
| `SESSION_SECRET` | **Yes** | Random 64-char string for session encryption |
| `PORT` | No | Server port (default: 3000; dev scripts use 3001) |
| `NODE_ENV` | No | `production` for deployed environments |
| `LOG_LEVEL` | No | `debug`, `info` (default), `warn`, `error` |
| `CORS_ALLOWED_ORIGINS` | No | Comma-separated extra allowed origins (required in production) |
| `ADMIN_EMAIL` | No | First user with this email gets admin role |
| `REDIS_URL` | No | Enables Redis cache for multi-instance scaling |
| `MONITORING_PORT` | No | Prometheus `/metrics` + `/health` port (default: 9090, internal only) |
| `FRONTEND_DIR` | No | Path to built frontend served by the Go server |
| `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` / `VAPID_SUBJECT` | No | Web push notifications |
| `TWILIO_ACCOUNT_SID` / `TWILIO_AUTH_TOKEN` / `TWILIO_FROM_NUMBER` | No | SMS alerts |
| `OLA_MAPS_API_KEY` | No | Geocoding / place search provider |
| `VITE_API_URL` | No | Backend origin for Capacitor native builds (build-time) |

## Deployment

Pre-configured for **Render** via `render.yaml` — Docker deploy using
`Dockerfile.backend` (multi-stage: frontend build + Go build), health check at
`/health`. A `docker-compose.prod.yml` is included for self-hosting.

## Mobile Builds (Capacitor)

Android and iOS builds require the production backend URL as an argument:

```bash
npm run build:android:prod -- https://your-backend.onrender.com
npm run build:ios:prod -- https://your-backend.onrender.com
```

The URL must be a public HTTPS origin with no trailing slash or path.

## Project Structure

```
backend/
  main.go               # Entry — config, DB pool, schema init, HTTP + WS + monitoring
  internal/
    api/                # HTTP router & handlers — auth, admin, search, health, SPA serving
    auth/               # Sessions, password hashing, rate limiting
    cache/              # In-memory cache; optional Redis layer
    config/             # Env config loading & validation
    db/                 # pgx pool, schema DDL, queries
    intelligence/       # Safety/anomaly heuristics
    monitoring/         # Prometheus metrics server
    ws/                 # WebSocket hub — positions, SOS, safety features
  test/                 # Integration test + benchmark shell scripts
frontend/src/
  pages/                # Landing, Login, Register, MainApp, viewers, dashboards
  components/           # Map, panels; emergency/, sharing/, landing/, usersList/, primitives/
  lib/                  # Realtime client, API client, stores, Kalman filter, offline buffer, three/
  styles/               # OKLCH theme tokens, auth styles
scripts/                # Dev orchestrator (start-v3.js), mobile builds, color/contrast checks
```

## License

ISC
