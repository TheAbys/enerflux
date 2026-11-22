# ⚡ EnerFlux

EnerFlux is a lightweight energy tracking backend written in Go.

It collects, stores, and exposes energy-related measurements such as:

- Solar production (SolarLog integration)
- Energy consumption (planned)
- Future: pellet, water, and other utility tracking

---

## 🧠 Architecture

EnerFlux is split into clear layers:

```
SolarLog / external systems
        ↓
Datasource (clients + parsers)
        ↓
Service layer
        ↓
Repository layer
        ↓
PostgreSQL
        ↓
API (Gin)
```

A background worker periodically syncs data from external sources into the database.

---

## ⚙️ Features (current state)

### ✅ Implemented
- SolarLog integration (local device)
- Data parsing and mapping
- PostgreSQL persistence
- Background worker (polling sync)
- REST API (Gin)
- Measurement listing endpoint
- Health check endpoint
- Config-based setup

### 🚧 In progress
- Statistics endpoints (daily / monthly aggregation)
- Better filtering & pagination
- Multi-source energy inputs (ETA heating etc.)

---

## 🚀 API

Base URL:
```
http://localhost:8080/api/v1
```

### Health checks

```
GET /api/v1/health/live
GET /api/v1/health/ready
```

`/api/v1/health/live` checks that the API process is responsive. `/api/v1/health/ready` also checks its PostgreSQL connection; `/api/v1/health` remains an alias for readiness.

Ready response:

```json
{
  "status": "ok",
  "version": "0.1.0",
  "database": "ok"
}
```

---

### Measurements

```
GET /measurements
```

Query parameters:

- `limit` (optional)

Example:

```
GET /measurements?limit=100
```

Response:

```json
[
  {
    "timestamp": "2026-06-30T12:00:00Z",
    "type": "pv.power",
    "value": 1234,
    "unit": "W",
    "source": "solarlog"
  }
]
```

---

## 🔧 Configuration

The application is configured via environment variables.

### Required variables:

```bash
DATABASE_URL=postgres://enerflux:enerflux@localhost:5432/enerflux
SOLARLOG_URL=http://solar-log/getjp
POLL_INTERVAL=5s
```
---

## 🧰 Taskfile Commands

The Go backend and Vue frontend live in the same repository. Go templates continue to render the page shell; Vite builds the Vue, TypeScript, and Tailwind assets served by Gin.

Install the frontend dependencies once after checkout:

```bash
task frontend:install
```

### ▶️ Development

Starts PostgreSQL and MQTT with Docker Compose, waits for PostgreSQL to become healthy, then runs the API, collection agent, and Vite development server. Open `http://localhost:5173` to use Vite's proxy with HMR, or `http://localhost:8080` to reach Gin directly. For remote access to Gin directly, set `VITE_DEV_SERVER` in `.env` to the browser-reachable host on port `5173`.

Stop only the local dependency containers with `task services:down`; the database volume is retained.

```bash
task dev
```

### ▶️ Run the built application

```bash
task run
```

### 🏗 Build frontend and Go binaries

```bash
task build
```

### ✅ Check code

```bash
task test
```

The frontend build is written to `internal/web/static/build` for Gin to serve. `task frontend:typecheck` checks Vue and TypeScript source without running the Go test suite.

### 🌐 Languages and translations

German is the default language. The language selector stores the choice in browser storage so it is kept between visits. Shared Vue I18n messages live in `frontend/src/i18n/messages/de.ts` and `frontend/src/i18n/messages/en.ts`; keep their keys in sync when adding or changing UI text. Use Vue I18n keys in components rather than hard-coding user-facing copy. Number and date formatting follows the selected language.

## ☸️ Deploy with k3s

The [k3s deployment folder](./k3s/README.md) includes Helm charts and scripts for running the API, agent, PostgreSQL, and Mosquitto together.

---

## 📦 Project Structure

```
cmd/
  api/                # main entrypoint
  agent/              # collection worker

frontend/
  src/components/     # shared Vue components
  src/i18n/           # Vue I18n setup and German/English message catalogs
  src/pages/          # Vue and TypeScript page components
  src/lib/             # typed API, data formats, and shared helpers
  src/styles.css       # Tailwind theme and source scanning
  vite.config.ts      # Vite and Tailwind build

internal/
  api/                # HTTP API and page routes (Gin)
  config/             # configuration
  contract/           # contracts
  datasource/         # external integrations (SolarLog)
  logger/             # logger
  analytics/          # resource calculations and summaries
  repository/         # database access
  service/            # business logic
  web/templates/      # Go-rendered document shell and shared navigation
  web/static/build/   # generated Vite assets served by Gin

migrations/           # DB schema
testdata/            # sample SolarLog responses
```

---

## 🧭 Roadmap

### Next steps
- Stats API (daily / monthly / live power)
- ETA heating integration
- Filtering & pagination
- Docker setup

### Future vision
EnerFlux will evolve into a multi-source energy monitoring platform:

- Solar
- Heating (pellets / ETA)
- Water
- Electricity consumption
- SaaS-ready architecture with agents

---

## 📌 Notes

This project evolves in small, incremental sessions.
Each session introduces one focused improvement.
