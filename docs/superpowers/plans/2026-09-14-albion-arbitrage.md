# AlbionFlow — Real-Time Albion Market & Refining Arbitrage Engine: Master Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a high-precision, multi-region web platform (Go 1.27 + SQLite + Astro 5) that calculates optimal refining profits (Silver per Focus) and cross-city hauling arbitrage for Albion Online with zero stale-data errors.

**Architecture:** A standalone Go backend daemon ingests prices from The Albion Online Data Project (AODP) across Asia, Americas, and Europe into an embedded SQLite store with a strict 60-minute data freshness guard. A deterministic mathematical engine models tier hierarchy, RRR material refunds, and station nutrition fees, serving a high-density, anti-UI-slop dark Astro web console with "Show the Math" step-by-step audit drawers.

**Tech Stack:** Go 1.27 (Chi router, `modernc.org/sqlite`, standard library HTTP/math), Astro 5 (Tailwind CSS, clean dark `#16181C`, tabular monospace figures), SQLite (WAL mode).

**Spec:** `docs/superpowers/specs/2026-09-14-albion-arbitrage-design.md`

## Global Constraints
- Ground Truth Precision: Zero approximation error. Formulas must model exact previous-tier refined consumption (e.g. 1 T6 Bar requires 1 T5 Bar + 4 T6 Ore) and exact RRR refunds on both materials.
- Data Freshness Guard: Any price quote older than 60 minutes must be flagged as `STALE` and excluded from default top recommendations.
- Zero Botting / 100% Legal: Zero game RAM injection, zero game packet sniffing. All data consumed strictly via AODP public community REST APIs.
- Anti-UI-Slop Contract: No pitch-black `#000000`, no emoji buttons, no floaty blurry cards. Clean dark `#16181C`, hairline borders `#2A2E35`, accent electric lime `#C7F53B`, and monospace tabular figures for silver amounts.
- Documentation Standard: Separate End-User Guide (how to safely execute trades in-game) and Developer Runbook (math proofs, architecture, API reference).

---

## Phase 0 — Toolchain, Scaffolding & Data Foundation

### Task 0.1: Monorepo Scaffolding & Toolchain Setup
**Files:**
- Create: `.gitignore`
- Create: `README.md`
- Scaffold: `backend/` via `go mod init albionflow/backend`
- Scaffold: `frontend/` (Astro 5 web application)

- [ ] **Step 1: Setup root .gitignore and README.md**
- [ ] **Step 2: Initialize Go module in `backend/` and install dependencies (`chi`, `modernc.org/sqlite`)**
- [ ] **Step 3: Initialize Astro 5 frontend in `frontend/`**
- [ ] **Step 4: Verify builds (`go build ./...` in backend & `npm run build` in frontend)**
- [ ] **Step 5: Commit:** `chore: scaffold albionflow monorepo with go backend and astro frontend`

---

### Task 0.2: Item Metadata & Commodity Recipe Catalog
**Files:**
- Create: `backend/pkg/model/items.go`
- Create: `backend/pkg/model/items_test.go`

- [ ] **Step 1: Define commodity types (Wood, Ore, Hide, Fiber, Stone), T4-T8 tiers, and .0 to .4 enchantments**
- [ ] **Step 2: Map Item Values (IV) and raw/refined input ratios according to Albion Wiki data**
- [ ] **Step 3: Write tests verifying recipe lookups (e.g. T6 metal bar requires T5 metal bar + 4 T6 ore)**
- [ ] **Step 4: Commit:** `feat(backend): item metadata catalog and refining recipe definitions`

---

### Task 0.3: SQLite Storage & Market Schema
**Files:**
- Create: `backend/pkg/store/db.go`
- Create: `backend/pkg/store/prices.go`
- Test: `backend/pkg/store/prices_test.go`

- [ ] **Step 1: Initialize pure-Go SQLite with WAL mode (`PRAGMA journal_mode = WAL`)**
- [ ] **Step 2: Create `price_snapshots` table with composite index `(region, item_id, city, recorded_at)`**
- [ ] **Step 3: Implement upsert and query functions with data age calculation**
- [ ] **Step 4: Write tests for price insertion, retrieval, and age filtering**
- [ ] **Step 5: Commit:** `feat(backend): sqlite storage engine with price snapshot and age index`

---

## Phase 1 — High-Precision Mathematical Engines

### Task 1.1: Refining & Silver per Focus (SPF) Calculator
**Files:**
- Create: `backend/pkg/engine/refining.go`
- Test: `backend/pkg/engine/refining_test.go`

- [ ] **Step 1: Implement Resource Return Rate (RRR) formula (15.2% base, 36.7% city bonus, 53.9% focus bonus)**
- [ ] **Step 2: Implement Station Nutrition Fee calculation based on Item Value (IV)**
- [ ] **Step 3: Implement previous-tier refined refund and net profit calculation**
- [ ] **Step 4: Implement Silver per Focus (SPF) metric**
- [ ] **Step 5: Write comprehensive unit tests checking calculations against wiki ground truth**
- [ ] **Step 6: Commit:** `feat(backend): high-precision refining and silver per focus calculator`

---

### Task 1.2: Cross-City Hauling Arbitrage Scanner
**Files:**
- Create: `backend/pkg/engine/arbitrage.go`
- Test: `backend/pkg/engine/arbitrage_test.go`

- [ ] **Step 1: Implement net revenue deduction (4% premium / 8% non-premium tax + 2.5% setup fee)**
- [ ] **Step 2: Scan all pairs of 6 Royal Cities + Caerleon for positive spread**
- [ ] **Step 3: Attach Data Freshness Guard: flag routes using data > 60m old as STALE**
- [ ] **Step 4: Write unit tests verifying taxes, setup fees, and margin ranking**
- [ ] **Step 5: Commit:** `feat(backend): cross-city arbitrage route scanner with tax deductions`

---

## Phase 2 — Ingestion Pipeline & HTTP JSON API

### Task 2.1: AODP Client & Data Freshness Guard
**Files:**
- Create: `backend/pkg/aodp/client.go`
- Create: `backend/pkg/aodp/client_test.go`

- [ ] **Step 1: Implement HTTP client supporting Asia, Americas, and Europe endpoints**
- [ ] **Step 2: Parse price responses with zero-price sanitization (skip empty market slots)**
- [ ] **Step 3: Store fetched prices into SQLite store**
- [ ] **Step 4: Write tests with mock AODP HTTP responses**
- [ ] **Step 5: Commit:** `feat(backend): aodp rest client with multi-region ingestion and zero-price filter`

---

### Task 2.2: Backend HTTP API Server
**Files:**
- Create: `backend/cmd/server/main.go`
- Create: `backend/cmd/server/handlers.go`
- Test: `backend/cmd/server/handlers_test.go`

- [ ] **Step 1: Wire Chi router with CORS and JSON middleware**
- [ ] **Step 2: Implement `GET /api/v1/refining/matrix` (filterable by region, commodity, tier, station tax, premium)**
- [ ] **Step 3: Implement `GET /api/v1/arbitrage/routes` (filterable by region, min profit, freshness)**
- [ ] **Step 4: Implement `POST /api/v1/prices/refresh` (triggers on-demand snapshot ingestion)**
- [ ] **Step 5: Write HTTP endpoint integration tests**
- [ ] **Step 6: Commit:** `feat(backend): http api server exposing refining matrix and arbitrage endpoints`

---

## Phase 3 — Astro High-Density Web Console

### Task 3.1: Anti-UI-Slop Layout & Global Control Bar
**Files:**
- Create: `frontend/src/layouts/Layout.astro`
- Create: `frontend/src/components/Header.astro`
- Create: `frontend/src/components/ControlBar.astro`

- [ ] **Step 1: Setup Clean Dark theme tokens (`#16181C`, hairline `#2A2E35`, electric lime `#C7F53B`)**
- [ ] **Step 2: Implement Region Selector (Asia / Americas / Europe) with instant state toggle**
- [ ] **Step 3: Implement Station Tax Slider & Premium Toggle**
- [ ] **Step 4: Commit:** `feat(frontend): dark console layout with region switcher and parameter controls`

---

### Task 3.2: Refining Matrix & "Show the Math" Audit Modal
**Files:**
- Create: `frontend/src/components/RefiningTable.astro`
- Create: `frontend/src/components/ShowTheMathModal.astro`
- Create: `frontend/src/pages/index.astro`

- [ ] **Step 1: Build dense table ranking commodities by Silver per Focus (SPF) with tabular figures**
- [ ] **Step 2: Add Data Freshness Badge (`<15m` Green, `15-60m` Amber, `>60m` Red Stale)**
- [ ] **Step 3: Implement "Show the Math" interactive modal showing step-by-step cost & refund breakdown**
- [ ] **Step 4: Commit:** `feat(frontend): refining matrix table and step-by-step math audit modal`

---

### Task 3.3: Cross-City Hauling Arbitrage View
**Files:**
- Create: `frontend/src/components/ArbitrageTable.astro`
- Create: `frontend/src/pages/arbitrage.astro`

- [ ] **Step 1: Build trade route matrix (Origin City -> Target City -> Net Spread)**
- [ ] **Step 2: Filter by minimum net margin & exclude stale data**
- [ ] **Step 3: Commit:** `feat(frontend): cross-city hauling arbitrage route matrix`

---

## Phase 4 — E2E Verification & Documentation

### Task 4.1: End-to-End Simulation & Formula Audit Test
**Files:**
- Create: `backend/tests/e2e/simulation_test.go`

- [ ] **Step 1: Test Scenario: Seed realistic multi-city market prices across T4-T8**
- [ ] **Step 2: Verify zero-loss assertions: RRR refunds, tax deductions, and SPF rankings match exact expected values**
- [ ] **Step 3: Commit:** `test(e2e): full simulation verification for refining math and arbitrage routes`

---

### Task 4.2: Dual-Audience Documentation (RILIS-2026.09.14)
**Files:**
- Create: `docs/END_USER_GUIDE.md` (Safe trade execution guide, checklist before hauling)
- Create: `docs/DEVELOPER_RUNBOOK.md` (Math proofs, API reference, data pipeline maintenance)
- Update: `README.md`

- [ ] **Step 1: Write END_USER_GUIDE.md**
- [ ] **Step 2: Write DEVELOPER_RUNBOOK.md**
- [ ] **Step 3: Update README.md with quickstart commands**
- [ ] **Step 4: Commit:** `docs: add end-user guide, developer runbook, and quickstart docs`
