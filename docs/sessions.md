# EnerFlux – Session Log

This file tracks development progress session by session.
Each session is designed for ~1 hour of focused work.

---

## 📊 Overall Progress

- [x] Session 1 – Project Setup
- [ ] Session 2 – Database & Schema
- [ ] Session 3 – Measurement API
- [ ] Session 4 – SolarLog Integration (Read)
- [ ] Session 5 – Automated Ingestion
- [ ] Session 6 – Basic Analytics
- [ ] Session 7 – Energy Flow Logic
- [ ] Session 8 – Refactoring & Architecture Cleanup
- [ ] Session 9 – Docker & Deployment
- [ ] Session 10 – SaaS Preparation

---

# 🧱 Session 1 – Project Setup

## 🎯 Goal
Create a running Go API with database connection.

## 📌 Tasks
- [x] Initialize Go module
- [x] Setup Gin HTTP server
- [x] Create `/health` endpoint
- [x] Setup PostgreSQL via Docker
- [x] Connect Go application to database
- [x] Test DB connection (SELECT 1)

## 🧪 Done when:
- [x] `GET /health` returns `{"status":"ok"}`
- [x] App starts without errors
- [x] DB connection works

## 📝 Notes
-

---

# 🗄 Session 2 – Database & Schema

## 🎯 Goal
Store and retrieve energy measurements.

## 📌 Tasks
- [ ] Create `measurements` table
- [ ] Setup migration system
- [ ] Create Go model struct
- [ ] Implement repository layer (insert/select)
- [ ] Test DB read/write manually

## 🧪 Done when:
- [ ] You can insert a measurement
- [ ] You can query stored data

## 📝 Notes
-

---

# 📥 Session 3 – Measurement API

## 🎯 Goal
Expose API to store measurements.

## 📌 Tasks
- [ ] Create `POST /measurements`
- [ ] Validate input payload
- [ ] Store data in database
- [ ] Return stored object or success response
- [ ] Test with curl/Postman

## 🧪 Done when:
- [ ] Data can be sent via HTTP
- [ ] Data appears in database

## 📝 Notes
-

---

# ☀️ Session 4 – SolarLog Integration (Read Only)

## 🎯 Goal
Fetch PV data from SolarLog system.

## 📌 Tasks
- [ ] Identify SolarLog API endpoint
- [ ] Implement HTTP client in Go
- [ ] Parse JSON response
- [ ] Log raw PV data
- [ ] Handle authentication if needed

## 🧪 Done when:
- [ ] PV data is successfully fetched
- [ ] Data is visible in logs

## 📝 Notes
-

---

# 🔄 Session 5 – Automated Ingestion

## 🎯 Goal
Automatically store PV data.

## 📌 Tasks
- [ ] Create polling mechanism (timer/cron)
- [ ] Map SolarLog data → Measurement struct
- [ ] Store data in DB automatically
- [ ] Add basic error handling

## 🧪 Done when:
- [ ] Data is stored without manual requests

## 📝 Notes
-

---

# 📊 Session 6 – Basic Analytics

## 🎯 Goal
First aggregated insights.

## 📌 Tasks
- [ ] Create daily aggregation query
- [ ] Build `/stats/day` endpoint
- [ ] Return PV + consumption totals

## 🧪 Done when:
- [ ] Daily energy values are visible via API

## 📝 Notes
-

---

# ⚡ Session 7 – Energy Flow Logic

## 🎯 Goal
Calculate self-consumption & grid flow.

## 📌 Tasks
- [ ] Define energy flow rules
- [ ] Calculate self-consumption
- [ ] Calculate grid import/export
- [ ] Expose results via API

## 🧪 Done when:
- [ ] Energy balance is understandable

## 📝 Notes
-

---

# 🧹 Session 8 – Refactoring

## 🎯 Goal
Clean architecture.

## 📌 Tasks
- [ ] Separate service layer
- [ ] Improve repository abstraction
- [ ] Add config system
- [ ] Improve logging

## 🧪 Done when:
- [ ] Code structure is clean and readable

## 📝 Notes
-

---

# 🐳 Session 9 – Dockerization

## 🎯 Goal
Run everything with one command.

## 📌 Tasks
- [ ] Create Dockerfile for Go app
- [ ] Setup docker-compose (API + DB)
- [ ] Test full stack startup

## 🧪 Done when:
- [ ] `docker-compose up` runs full system

## 📝 Notes
-

---

# 🌐 Session 10 – SaaS Preparation

## 🎯 Goal
Prepare multi-user structure.

## 📌 Tasks
- [ ] Add `user_id` or `house_id`
- [ ] Separate data per tenant
- [ ] Prepare API key structure
- [ ] Document SaaS vision

## 🧪 Done when:
- [ ] System supports multiple logical users

## 📝 Notes
-

---

# 🧭 How to use this file

After each session:

1. Mark tasks as completed: `- [x]`
2. Write 2–3 lines in Notes
3. Optionally update progress section

This keeps progress visible without extra tooling.