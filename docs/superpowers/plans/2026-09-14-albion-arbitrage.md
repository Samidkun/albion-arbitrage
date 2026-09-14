# AlbionFlow — Real-Time Albion Market & Refining Arbitrage Engine: Master Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a high-precision, multi-region web platform (Go 1.27 + SQLite + Astro 5) that solves the 5 critical player pain points: stale data traps, intermediate tax double-counting, stone enchantment discrepancies, illiquid low-volume traps, and rigid tax channels.

**Architecture:** A standalone Go backend daemon ingests prices and 24h trade volumes from The Albion Online Data Project (AODP) across Asia, Americas, and Europe into an embedded SQLite store with a strict 60-minute data freshness guard. A deterministic mathematical engine models tier hierarchy, RRR material refunds, and station nutrition fees, serving a high-density, anti-UI-slop dark Astro web console with "Show the Math" step-by-step audit drawers.

**Tech Stack:** Go 1.27 (Chi router, `modernc.org/sqlite`, standard library HTTP/math), Astro 5 (Tailwind CSS, clean dark `#16181C`, tabular monospace figures), SQLite (WAL mode).

**Spec:** `docs/superpowers/specs/2026-09-14-albion-arbitrage-design.md`

## Global Constraints
- Ground Truth Precision: Zero approximation error. Formulas must model exact previous-tier refined consumption (e.g. 1 T6 Bar requires 1 T5 Bar + 4 T6 Ore) and exact RRR refunds on both materials.
- Stone Exception: Stone blocks do not have enchantments (.1-.4). Catalog must strictly respect this to prevent corrupted math.
- Chained Refining: Provide an option for multi-tier refining without double-deducting market tax on intermediate steps.
- Trade Volume Guard: Must track 24h trade volume (`item_count`) from AODP history so players don't sink capital into illiquid commodities.
- Data Freshness Guard: Any price quote older than 60 minutes must be flagged as `STALE` and excluded from default top recommendations.
- Zero Botting / 100% Legal: Zero game RAM injection, zero game packet sniffing. All data consumed strictly via AODP public community REST APIs.
- Anti-UI-Slop Contract: No pitch-black `#000000`, no emoji buttons, no floaty blurry cards. Clean dark `#16181C`, hairline borders `#2A2E35`, accent electric lime `#C7F53B`, and monospace tabular figures for silver amounts.
- Documentation Standard: Separate End-User Guide (how to safely execute trades in-game) and Developer Runbook (math proofs, architecture, API reference).

---

## Phase 0 — Scaffolding, Models, SQLite & Trade Volume Ingest

### Task 0.1: Monorepo Scaffolding & Toolchain Setup
**Files:**
- Create: `.gitignore`
- Create: `README.md`
- Scaffold: `backend/` via `go mod init albionflow/backend`
- Scaffold: `frontend/` (Astro 5 web application)

- [x] **Step 1: Setup root .gitignore and README.md**
- [x] **Step 2: Initialize Go module in `backend/` and install dependencies (`chi`, `modernc.org/sqlite`)**
- [x] **Step 3: Initialize Astro 5 frontend in `frontend/`**
- [x] **Step 4: Verify builds (`go build ./...` in backend & `npm run build` in frontend)**
- [x] **Step 5: Commit:** `chore: scaffold albionflow monorepo with go backend and astro frontend`

---

### Task 0.2: Item Metadata Catalog & Stone Exception Handling
Files:
- Create: `backend/pkg/model/items.go`
- Create: `backend/pkg/model/items_test.go`

- [x] **Step 1: Define commodity types (Wood, Ore, Hide, Fiber, Stone), T4-T8 tiers, and .0 to .4 enchantments**
- [x] **Step 2: Implement stone exception logic (Stone blocks do not support .1-.4 enchantments)**
- [x] **Step 3: Map Item Values (IV) and raw/refined input ratios according to Albion Wiki data**
- [x] **Step 4: Write tests verifying recipe lookups and stone exception constraints**
- [x] **Step 5: Commit:** `feat(backend): item metadata catalog with stone exception handling`

---

### Task 0.3: SQLite Storage & Market Price Snapshot Table
**Files:**
- Create: `backend/pkg/store/db.go`
- Create: `backend/pkg/store/prices.go`
- Test: `backend/pkg/store/prices_test.go`

- [x] **Step 1: Initialize pure-Go SQLite with WAL mode (`PRAGMA journal_mode = WAL`)**
- [x] **Step 2: Create `price_snapshots` table with composite index `(region, item_id, city, recorded_at)`**
- [x] **Step 3: Implement upsert and query functions with data age calculation**
- [x] **Step 4: Write tests for price insertion, retrieval, and age filtering**
- [x] **Step 5: Commit:** `feat(backend): sqlite storage engine with price snapshot and age index`

---

### Task 0.4: Market 24h Trade Volume Schema & Store (Anti-Illiquidity)
**Files:**
- Create: `backend/pkg/store/volumes.go`
- Test: `backend/pkg/store/volumes_test.go`

- [x] **Step 1: Create `market_volumes` table storing 24h traded `item_count` per city and item**
- [x] **Step 2: Implement query functions for 24h volume aggregation**
- [x] **Step 3: Write tests verifying volume retrieval and liquidity flagging**
- [x] **Step 4: Commit:** `feat(backend): trade volume storage to guard against illiquid trades`

---

## Phase 1 — High-Precision Mathematical Engines

### Task 1.1: Resource Return Rate (RRR) & Station Nutrition Fee Calculator
**Files:**
- Create: `backend/pkg/engine/rrr.go`
- Test: `backend/pkg/engine/rrr_test.go`

- [x] **Step 1: Implement Resource Return Rate (RRR) formula (15.2% base, 36.7% city bonus, 53.9% focus bonus)**
- [x] **Step 2: Implement Station Nutrition Fee calculation based on Item Value (IV)**
- [x] **Step 3: Write unit tests verifying nutrition cost and RRR refund precision**
- [x] **Step 4: Commit:** `feat(backend): rrr and station nutrition fee calculation engine`

---

### Task 1.2: Hierarchical Refining with Chained Stack Option
**Files:**
- Create: `backend/pkg/engine/refining.go`
- Test: `backend/pkg/engine/refining_test.go`

- [x] **Step 1: Implement hierarchical input cost calculation (T5 requires 3 raw + 1 T4 refined)**
- [x] **Step 2: Implement "Chained Refining" mode that eliminates intermediate market taxes**
- [x] **Step 3: Write unit tests comparing direct vs chained refining costs**
- [x] **Step 4: Commit:** `feat(backend): hierarchical refining engine with chained intermediate support`

---

### Task 1.3: Silver per Focus (SPF) Optimization Engine
**Files:**
- Create: `backend/pkg/engine/focus.go`
- Test: `backend/pkg/engine/focus_test.go`

- [x] **Step 1: Map base focus cost per tier and enchantment level**
- [x] **Step 2: Calculate Silver per Focus (SPF) metric: `(ProfitWithFocus - ProfitWithoutFocus) / FocusSpent`**
- [x] **Step 3: Write unit tests checking SPF ranking against ground truth**
- [x] **Step 4: Commit:** `feat(backend): silver per focus efficiency calculator`

---

### Task 1.4: Cross-City Hauling Arbitrage Scanner with Flexible Tax Channels
**Files:**
- Create: `backend/pkg/engine/arbitrage.go`
- Test: `backend/pkg/engine/arbitrage_test.go`

- [ ] **Step 1: Implement configurable sales tax channels (0% direct, 2.5% buy order, 4% premium sell, 8% non-premium)**
- [ ] **Step 2: Scan all pairs of 6 Royal Cities + Caerleon for positive spread**
- [ ] **Step 3: Attach Data Freshness Guard: flag routes using data > 60m old as STALE**
- [ ] **Step 4: Write unit tests verifying taxes, setup fees, and margin ranking**
- [ ] **Step 5: Commit:** `feat(backend): cross-city arbitrage route scanner with flexible tax channels`

---

## Phase 2 — Ingestion Pipeline & REST API

### Task 2.1: AODP Multi-Region Ingestion Client with Timestamp Age Tagging
**Files:**
- Create: `backend/pkg/aodp/client.go`
- Create: `backend/pkg/aodp/client_test.go`

- [ ] **Step 1: Implement HTTP client supporting Asia, Americas, and Europe endpoints**
- [ ] **Step 2: Parse price responses with zero-price sanitization (skip empty market slots)**
- [ ] **Step 3: Store fetched prices and 24h volume into SQLite store**
- [ ] **Step 4: Write tests with mock AODP HTTP responses**
- [ ] **Step 5: Commit:** `feat(backend): aodp rest client with multi-region ingestion and zero-price filter`

---

### Task 2.2: Data Freshness Guard & Stale Data Filter
**Files:**
- Create: `backend/pkg/aodp/guard.go`
- Test: `backend/pkg/aodp/guard_test.go`

- [ ] **Step 1: Implement age categorizer: `<30m` Fresh, `30-60m` Aging, `>60m` Stale**
- [ ] **Step 2: Automatically exclude stale quotes from top recommendations unless explicitly bypassed**
- [ ] **Step 3: Write unit tests for freshness classification**
- [ ] **Step 4: Commit:** `feat(backend): data freshness guard and stale price filter`

---

### Task 2.3: Backend HTTP API Server
**Files:**
- Create: `backend/cmd/server/main.go`
- Create: `backend/cmd/server/handlers.go`
- Test: `backend/cmd/server/handlers_test.go`

- [ ] **Step 1: Wire Chi router with CORS and JSON middleware**
- [ ] **Step 2: Implement `GET /api/v1/refining/matrix` (filterable by region, commodity, tier, station tax, tax channel)**
- [ ] **Step 3: Implement `GET /api/v1/arbitrage/routes` (filterable by region, min profit, min 24h volume, freshness)**
- [ ] **Step 4: Implement `POST /api/v1/prices/refresh` (triggers on-demand snapshot ingestion)**
- [ ] **Step 5: Write HTTP endpoint integration tests**
- [ ] **Step 6: Commit:** `feat(backend): http api server exposing refining matrix and arbitrage endpoints`

---

## Phase 3 — Astro High-Density Web Console

### Task 3.1: Anti-UI-Slop Dark Layout & Global Control Bar
**Files:**
- Create: `frontend/src/layouts/Layout.astro`
- Create: `frontend/src/components/Header.astro`
- Create: `frontend/src/components/ControlBar.astro`

- [ ] **Step 1: Setup Clean Dark theme tokens (`#16181C`, hairline `#2A2E35`, electric lime `#C7F53B`)**
- [ ] **Step 2: Implement Region Selector (Asia / Americas / Europe) with instant state toggle**
- [ ] **Step 3: Implement Station Tax Slider & Tax Channel Selector (0%, 2.5%, 4%, 8%)**
- [ ] **Step 4: Commit:** `feat(frontend): dark console layout with region switcher and parameter controls`

---

### Task 3.2: Refining Matrix Table with SPF Sorting & Trade Volume Indicators
**Files:**
- Create: `frontend/src/components/RefiningTable.astro`
- Create: `frontend/src/pages/index.astro`

- [ ] **Step 1: Build dense table ranking commodities by Silver per Focus (SPF) with tabular figures**
- [ ] **Step 2: Add Data Freshness Badge (`<15m` Green, `15-60m` Amber, `>60m` Red Stale)**
- [ ] **Step 3: Add 24h Trade Volume column with liquidity warning badge**
- [ ] **Step 4: Commit:** `feat(frontend): refining matrix table with spf ranking and volume indicators`

---

### Task 3.3: "Show the Math" Step-by-Step Audit Modal
**Files:**
- Create: `frontend/src/components/ShowTheMathModal.astro`

- [ ] **Step 1: Render exact breakdown: raw material cost + previous tier cost - RRR refund = net material cost**
- [ ] **Step 2: Render exact station fee + market tax breakdown**
- [ ] **Step 3: Provide copyable summary text for Discord/guild trade planning**
- [ ] **Step 4: Commit:** `feat(frontend): show the math interactive calculation audit modal`

---

### Task 3.4: Cross-City Hauling Arbitrage Matrix
**Files:**
- Create: `frontend/src/components/ArbitrageTable.astro`
- Create: `frontend/src/pages/arbitrage.astro`

- [ ] **Step 1: Build trade route matrix (Origin City -> Target City -> Net Spread)**
- [ ] **Step 2: Filter by minimum net margin & exclude stale data**
- [ ] **Step 3: Commit:** `feat(frontend): cross-city hauling arbitrage route matrix`

---

## Phase 4 — E2E Verification & Documentation

### Task 4.1: Full Simulation Verification with Multi-City Real World Feed
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
