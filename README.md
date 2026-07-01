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

### Health check

```
GET /health
```

Response:

```json
{
  "status": "ok"
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

This project uses a Taskfile to simplify common development workflows.

### ▶️ Start API
```bash
task run
```

### 🧪 Run Tests

```bash
task test
```

### 🏗 Build Binary

```bash
task build
```

---

## 📦 Project Structure

```
cmd/
  api/                # main entrypoint

internal/
  api/                # HTTP layer (Gin)
  config/             # configuration
  contract/           # contracts
  datasource/         # external integrations (SolarLog)
  logger/             # logger
  model/              # domain models
  repository/         # database access
  service/            # business logic
  worker/             # background sync

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