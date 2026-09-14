# AlbionFlow — Technical Developer Runbook & Engineering Operations Manual

> **Document Version:** 2026.09.14 · **Classification:** Engineering / Internal Architecture  
> **Backend Service:** Go 1.27 (port `8081`) · **Frontend Console:** Astro 5 + TypeScript + Tailwind CSS (port `4321`)  
> **Database:** Pure-Go SQLite (`modernc.org/sqlite`, WAL Mode) · **Repository Root:** `/mnt/data/01_Projects/Porto/albion-arbitrage`

---

## Table of Contents
1. [System Topology & High-Level Architecture](#1-system-topology--high-level-architecture)
2. [Project Directory Layout](#2-project-directory-layout)
3. [Local Development Setup & Prerequisites](#3-local-development-setup--prerequisites)
4. [Running the Backend Daemon & API Surface](#4-running-the-backend-daemon--api-surface)
5. [Running the Astro 5 Web Console](#5-running-the-astro-5-web-console)
6. [SQLite Embedded Store & Schema Specifications](#6-sqlite-embedded-store--schema-specifications)
7. [Ingestion Client & Polling Strategy](#7-ingestion-client--polling-strategy)
8. [Mathematical Engine Reference & Formal Proofs](#8-mathematical-engine-reference--formal-proofs)
9. [Testing & Verification Runbook](#9-testing--verification-runbook)
10. [Operational Troubleshooting Guide](#10-operational-troubleshooting-guide)

---

## 1. System Topology & High-Level Architecture

AlbionFlow is a high-precision, zero-latency financial decision engine for Albion Online. It continuously monitors market prices and 24-hour trade volumes across all three official game regions (Asia, Americas, Europe), computes resource return rates (RRR), calculates Silver per Focus (SPF), and identifies cross-city hauling arbitrage opportunities while strictly enforcing data freshness guards (<60 minutes) and liquidity thresholds.

```
┌────────────────────────────────────────────────────────────────────────┐
│ The Albion Online Data Project (AODP) Public REST APIs                │
│ • Asia (Singapore)     : https://east.albion-online-data.com           │
│ • Americas (Washington): https://west.albion-online-data.com           │
│ • Europe (Amsterdam)   : https://europe.albion-online-data.com         │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ HTTP GET (50 IDs per batch chunk)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ AlbionFlow Backend Daemon (Go 1.27 / Chi Router on :8081)             │
│                                                                        │
│ ┌──────────────────────┐  ┌──────────────────┐  ┌────────────────────┐ │
│ │  aodp.Client         │  │ aodp.Guard       │  │ engine.Refining    │ │
│ │  - Multi-region pull │  │ - Freshness gate │  │ - RRR refunds      │ │
│ │  - Zero-price filter │  │ - Stale flagger  │  │ - Chained stack    │ │
│ └──────────┬───────────┘  └────────┬─────────┘  └─────────┬──────────┘ │
│            │                       │                      │            │
│            ▼                       ▼                      ▼            │
│ ┌────────────────────────────────────────────────────────────────────┐ │
│ │ SQLite Embedded Store (data/market.db via modernc.org/sqlite)      │ │
│ │ • PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000;           │ │
│ │ • price_snapshots (composite index: region, item_id, city, date)   │ │
│ │ • market_volumes  (unique: region, item_id, city, quality)         │ │
│ └──────────────────────────────────┬─────────────────────────────────┘ │
│                                    │                                   │
│ ┌──────────────────────────────────▼─────────────────────────────────┐ │
│ │ HTTP JSON API Handlers                                             │ │
│ │ • GET  /healthz                                                    │ │
│ │ • GET  /api/v1/refining/matrix                                     │ │
│ │ • GET  /api/v1/arbitrage/routes                                    │ │
│ │ • POST /api/v1/prices/refresh                                      │ │
│ └──────────────────────────────────┬─────────────────────────────────┘ │
└────────────────────────────────────┼───────────────────────────────────┘
                                     │ JSON REST API / CORS
                                     ▼
┌────────────────────────────────────────────────────────────────────────┐
│ AlbionFlow Astro 5 Web Console (:4321)                                 │
│ • Layout: Clean Dark (#16181C), hairline (#2A2E35), lime (#C7F53B)     │
│ • ControlBar.astro: Region switcher, tax slider, channel selector      │
│ • RefiningTable.astro: SPF-ranked matrix, 24h volume, freshness badge  │
│ • ArbitrageTable.astro: Cross-city routes, risk badges, net margins    │
│ • ShowTheMathModal.astro: Step-by-step audit drawer & Discord export   │
└────────────────────────────────────────────────────────────────────────┘
```

### Key Architectural Tenets
1. **Zero Approximation:** All refining material inputs (raw and previous-tier refined) and their respective RRR returns are modeled down to the exact floating-point decimal before rounding.
2. **Stone Exception Isolation:** Stone blocks strictly lack enchantments (.1 to .4) in Albion Online. The item catalog (`pkg/model`) enforces this constraint, preventing corrupt schemas and phantom permutations.
3. **Pure-Go SQLite:** Built on `modernc.org/sqlite` without CGO. Runs on any platform with zero external C compiler dependencies.
4. **Anti-UI-Slop Contract:** The frontend adheres strictly to high-density dark UI specifications: no pitch black `#000000`, no floating emoji cards, monospace tabular figures for all currency amounts, and hairline `#2A2E35` dividers.

---

## 2. Project Directory Layout

```
/mnt/data/01_Projects/Porto/albion-arbitrage/
├── README.md                           # Monorepo overview and quickstart guide
├── docs/
│   ├── END_USER_GUIDE.md               # User manual for crafters, traders, and haulers
│   ├── DEVELOPER_RUNBOOK.md            # Technical runbook and operational manual (this file)
│   └── superpowers/                    # SDD design specifications and execution plans
├── backend/
│   ├── go.mod                          # Go 1.27 module definition
│   ├── go.sum                          # Go cryptographic dependency checksums
│   ├── cmd/
│   │   └── server/
│   │       ├── main.go                 # Backend entrypoint (port 8081)
│   │       └── main_test.go            # HTTP handler integration tests
│   ├── pkg/
│   │   ├── model/
│   │   │   ├── items.go                # Item definitions, recipe catalog, stone exceptions
│   │   │   └── items_test.go           # Recipe validation and stone constraint tests
│   │   ├── store/
│   │   │   ├── db.go                   # SQLite WAL connection, PRAGMA init, schema DDL
│   │   │   ├── prices.go               # Price snapshot upsert, querying, age calculations
│   │   │   ├── prices_test.go          # Price snapshot storage tests
│   │   │   ├── volumes.go              # 24h market trade volume aggregation
│   │   │   └── volumes_test.go         # Trade volume queries and liquidity guard tests
│   │   ├── engine/
│   │   │   ├── rrr.go                  # Resource Return Rate & Station Nutrition fee math
│   │   │   ├── rrr_test.go             # RRR refund precision unit tests
│   │   │   ├── refining.go             # Hierarchical & chained refining cost engine
│   │   │   ├── refining_test.go        # Unit and batch refining economics tests
│   │   │   ├── focus.go                # Silver per Focus (SPF) & mastery curve algorithms
│   │   │   ├── focus_test.go           # SPF evaluation and focus cost tests
│   │   │   ├── arbitrage.go            # Cross-city hauling arbitrage route finder & taxes
│   │   │   └── arbitrage_test.go       # Route scanning, tax channels, liquidity filter tests
│   │   ├── aodp/
│   │   │   ├── client.go               # Multi-region AODP HTTP client with batching
│   │   │   ├── client_test.go          # AODP mock ingestion and zero-price filtering tests
│   │   │   ├── guard.go                # Timestamp age calculation & freshness classifier
│   │   │   └── guard_test.go           # Freshness tier classification tests
│   │   └── api/
│   │       ├── router.go               # Chi router setup and CORS middleware
│   │       └── handlers.go             # HTTP handlers for refining, arbitrage, and refresh
│   └── tests/
│       └── e2e/
│           └── arbitrage_workflow_test.go # Full end-to-end multi-city market simulation tests
└── frontend/
    ├── package.json                    # Astro 5 dependencies (Astro, Tailwind, TypeScript)
    ├── astro.config.mjs                # Astro configuration (static output)
    ├── tsconfig.json                   # TypeScript configuration
    ├── src/
    │   ├── pages/
    │   │   ├── index.astro             # Home console: Refining Matrix & SPF Leaderboard
    │   │   └── arbitrage.astro         # Cross-City Hauling Arbitrage Matrix
    │   ├── components/
    │   │   ├── ControlBar.astro        # Global region, commodity, tier, tax controls
    │   │   ├── RefiningTable.astro     # High-density refining data table
    │   │   ├── ArbitrageTable.astro    # Hauling trade route matrix with risk badges
    │   │   └── ShowTheMathModal.astro  # Step-by-step audit drawer with Discord export
    │   └── styles/
    │       └── global.css              # Dark theme tokens (#16181C, hairline #2A2E35)
    └── dist/                           # Static production build artifacts
```

---

## 3. Local Development Setup & Prerequisites

### System Requirements
- **Go:** Version 1.27 or higher (`go version`)
- **Node.js:** Version 22+ (recommended: Node 26+)
- **Package Manager:** `pnpm` (version 9 or 10)
- **OS:** Linux, macOS, or Windows (WSL2)

### Verification Commands
```bash
# Verify Go installation
go version
# Output: go version go1.27.x linux/amd64

# Verify Node & pnpm
node -v
pnpm -v
```

### Initial Clone & Dependency Installation
```bash
git clone https://github.com/placeholder/albion-arbitrage.git
cd albion-arbitrage

# Install backend dependencies
cd backend
go mod download
go mod verify

# Install frontend dependencies
cd ../frontend
pnpm install
```

### Environment Variables
| Variable | Service | Default | Description |
|---|---|---|---|
| `PORT` | Backend | `8081` | Port for the Go HTTP API daemon |
| `DB_PATH` | Backend | `data/market.db` | Filepath for the embedded SQLite database (`:memory:` for ephemeral memory) |
| `PUBLIC_API_URL` | Frontend | `http://localhost:8081` | Target URL of the backend API for client-side fetches |

---

## 4. Running the Backend Daemon & API Surface

### Start the Backend Server
```bash
cd backend
go run ./cmd/server/main.go
```
*Console output:*
```
2026/09/14 18:13:31 AlbionFlow backend listening on :8081
```

### API Endpoint Reference

#### 1. Health Check
- **Endpoint:** `GET /healthz`
- **Response:** `200 OK`
```json
{
  "status": "ok",
  "app": "albionflow_backend"
}
```

#### 2. Refining Profitability Matrix
- **Endpoint:** `GET /api/v1/refining/matrix`
- **Query Parameters:**
  - `region` (string, default: `asia`): `asia`, `americas`, `europe`.
  - `commodity` (string, optional): `wood`, `ore`, `hide`, `fiber`, `stone`.
  - `tier` (integer, optional): `4` to `8`.
  - `tax` (float, default: `600`): Station fee in Silver per 100 nutrition.
  - `channel` (string, default: `sell_order`): `direct`, `instant_buy`, `sell_order`.
  - `premium` (boolean, default: `true`): `true` or `false`.
  - `chained` (boolean, default: `false`): Enable recursive chained refining.
  - `mastery` (integer, default: `0`): Refining node mastery from `0` to `100`.
  - `city` (string, optional): Filter to specific city (defaults to each commodity's official bonus city).
- **Sample Response Item (`200 OK`):**
```json
[
  {
    "item_id": "T6_PLANKS_LEVEL1@1",
    "commodity": "Wood",
    "tier": 6,
    "enchantment": 1,
    "city": "Fort Sterling",
    "is_bonus_city": true,
    "item_value": 256,
    "raw_item_id": "T6_WOOD_LEVEL1@1",
    "raw_amount": 4,
    "raw_price": 420,
    "prev_refined_item_id": "T5_PLANKS_LEVEL1@1",
    "prev_refined_amount": 1,
    "prev_refined_price": 510,
    "sell_price": 2450,
    "station_fee": 172.8,
    "effective_raw_cost": 1063.44,
    "effective_prev_cost": 322.83,
    "total_craft_cost": 1559.07,
    "gross_revenue": 2450,
    "sales_tax": 98.0,
    "setup_fee": 61.25,
    "net_revenue": 2290.75,
    "profit_without_focus": 731.68,
    "profit_with_focus": 1108.48,
    "focus_profit_delta": 376.80,
    "base_focus_cost": 269,
    "actual_focus_cost": 269,
    "silver_per_focus": 1.40,
    "daily_focus_silver": 14007.43,
    "profit_margin_pct": 46.93,
    "rrr": 0.367,
    "is_chained": false,
    "traded_volume_24h": 450,
    "is_illiquid": false,
    "price_age_seconds": 360,
    "freshness": "fresh",
    "is_stale": false
  }
]
```

#### 3. Cross-City Hauling Arbitrage Routes
- **Endpoint:** `GET /api/v1/arbitrage/routes`
- **Query Parameters:**
  - `region` (string, default: `asia`): `asia`, `americas`, `europe`.
  - `min_profit` (float, default: `0`): Minimum net profit Silver per unit.
  - `min_margin` (float, default: `0`): Minimum net margin percentage.
  - `exclude_stale` (boolean, default: `true`): Exclude quotes older than 60 minutes.
  - `exclude_illiquid` (boolean, default: `true`): Exclude items with 24h volume < 5 units.
  - `channel` (string, default: `sell_order`): `direct`, `instant_buy`, `sell_order`.
  - `premium` (boolean, default: `true`): `true` or `false`.
- **Sample Response Item (`200 OK`):**
```json
[
  {
    "item_id": "T6_ORE",
    "origin_city": "Thetford",
    "target_city": "Caerleon",
    "buy_price": 500,
    "buy_price_age": 720000000000,
    "is_buy_stale": false,
    "sell_price": 850,
    "sell_price_age": 540000000000,
    "is_sell_stale": false,
    "is_stale_route": false,
    "tax_channel": "sell_order",
    "sales_tax_rate": 0.04,
    "setup_fee_rate": 0.025,
    "gross_revenue": 850,
    "net_revenue": 794.75,
    "net_profit_per_unit": 294.75,
    "net_margin_percent": 58.95,
    "traded_volume_24h": 1240,
    "is_illiquid": false
  }
]
```

#### 4. Trigger Market Price Refresh
- **Endpoint:** `POST /api/v1/prices/refresh`
- **Query / Body Parameter:** `region=asia` (or `americas`, `europe`).
- **Response (`200 OK`):**
```json
{
  "status": "ok",
  "region": "asia",
  "prices_ingested": 245,
  "volumes_ingested": 245
}
```

---

## 5. Running the Astro 5 Web Console

### Start the Development Server
```bash
cd frontend
pnpm dev
```
Accessible at: `http://localhost:4321`

### Build Production Static Assets
```bash
cd frontend
pnpm build
```
Generates static HTML and optimized JavaScript bundles in `frontend/dist/`.

### Design System & Anti-UI-Slop Token Reference
| CSS Token | Hex Value | Purpose |
|---|---|---|
| `--color-bg-base` | `#16181C` | Primary dark application background |
| `--color-bg-panel` | `#1E2228` | Elevated card & table container background |
| `--color-border-hairline` | `#2A2E35` | 1px border dividers and card boundaries |
| `--color-accent-lime` | `#C7F53B` | Highlighting top SPF yields & positive net profits |
| `--color-status-fresh` | `#10B981` | Fresh data indicator (< 15 min) |
| `--color-status-aging` | `#F59E0B` | Aging data indicator (15 - 60 min) |
| `--color-status-stale` | `#EF4444` | Stale data indicator (> 60 min) |
| `font-variant-numeric` | `tabular-nums` | Monospace column alignment for currency & volumes |

---

## 6. SQLite Embedded Store & Schema Specifications

The storage layer uses pure-Go `modernc.org/sqlite` without CGO. Data is persisted to `data/market.db`.

### Database Initialization & Pragmas
```sql
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;
```
- **WAL Mode:** Write-Ahead Logging allows concurrent readers alongside a single active writer without lock contention.
- **Busy Timeout:** Configured to 5,000ms (5 seconds) to handle transient database locks automatically.

### DDL Schema

```sql
-- Table: price_snapshots
CREATE TABLE IF NOT EXISTS price_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    region TEXT NOT NULL,
    item_id TEXT NOT NULL,
    city TEXT NOT NULL,
    quality INTEGER NOT NULL DEFAULT 1,
    sell_price_min INTEGER NOT NULL DEFAULT 0,
    sell_price_min_date TEXT NOT NULL,
    buy_price_max INTEGER NOT NULL DEFAULT 0,
    buy_price_max_date TEXT NOT NULL,
    recorded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_region_item_city_quality UNIQUE(region, item_id, city, quality)
);

CREATE INDEX IF NOT EXISTS idx_prices_lookup ON price_snapshots(region, item_id, city);
CREATE INDEX IF NOT EXISTS idx_prices_lookup_recorded ON price_snapshots(region, item_id, city, recorded_at);

-- Table: market_volumes
CREATE TABLE IF NOT EXISTS market_volumes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    region TEXT NOT NULL,
    item_id TEXT NOT NULL,
    city TEXT NOT NULL,
    quality INTEGER NOT NULL DEFAULT 1,
    item_count BIGINT NOT NULL DEFAULT 0,
    silver_amount BIGINT NOT NULL DEFAULT 0,
    timestamp TEXT NOT NULL,
    recorded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_vol_region_item_city_quality UNIQUE(region, item_id, city, quality)
);

CREATE INDEX IF NOT EXISTS idx_volume_lookup ON market_volumes(region, item_id, city);
```

---

## 7. Ingestion Client & Polling Strategy

### Multi-Region Ingestion Endpoints
- **Americas (West):** `https://west.albion-online-data.com/api/v2/stats/prices/{items}.json`
- **Asia (East):** `https://east.albion-online-data.com/api/v2/stats/prices/{items}.json`
- **Europe (EU):** `https://europe.albion-online-data.com/api/v2/stats/prices/{items}.json`
- **Trade History Endpoint:** `/api/v2/stats/history/{items}.json?time-scale=24`

### Batching & Chunking Strategy
- The client partitions item requests into batches of **50 items** per HTTP GET call to prevent URL length overflow (HTTP 414) and server timeout errors.
- Rate limiting: 100ms pause between successive chunk requests.

### Zero-Price Guard
Any item with `sell_price_min <= 0` or `buy_price_max <= 0` is discarded during ingestion to prevent corrupted division-by-zero or infinite-profit artifacts in the arbitrage scanner.

### Freshness Classification Strategy
Market quote timestamps are parsed using RFC3339 layouts. Age is computed against `now.UTC()`:
- **`fresh`:** Age < 30 minutes.
- **`aging`:** Age between 30 and 60 minutes.
- **`stale`:** Age > 60 minutes.

---

## 8. Mathematical Engine Reference & Formal Proofs

### 8.1 Resource Return Rate (RRR) Refund Formula
When refining an item with required raw quantity $Q_{\text{raw}}$ and required previous-tier refined quantity $Q_{\text{prev}}$:
$$\text{Refund}_{\text{raw}} = Q_{\text{raw}} \times \text{RRR}$$
$$\text{Refund}_{\text{prev}} = Q_{\text{prev}} \times \text{RRR}$$
$$\text{Net Raw Cost} = Q_{\text{raw}} \times P_{\text{raw}} \times (1 - \text{RRR})$$
$$\text{Net Prev Cost} = Q_{\text{prev}} \times P_{\text{prev}} \times (1 - \text{RRR})$$

Where $\text{RRR}$ values are:
- Base City (No biome bonus, no focus): `0.152` (15.2%)
- Biome Bonus City (Without focus): `0.367` (36.7%)
- Focus Only (No biome bonus): `0.435` (43.5%)
- Biome Bonus City + Crafting Focus: `0.539` (53.9%)

### 8.2 Station Nutrition Fee Formula
Each refined item has a base `Item Value` (IV) defined by the game:
$$\text{Nutrition Consumed} = \text{Item Value} \times 0.1125$$
$$\text{Station Fee (Silver)} = \frac{\text{Nutrition Consumed} \times \text{Station Tax Rate}}{100}$$

*Example:* A T6.1 Plank has $\text{Item Value} = 256$. At a station tax rate of $600$ Silver / 100 Nutrition:
$$\text{Nutrition Consumed} = 256 \times 0.1125 = 28.8$$
$$\text{Station Fee} = \frac{28.8 \times 600}{100} = 172.8\text{ Silver}$$

### 8.3 Mastery Curve & Silver Per Focus (SPF)
Crafting mastery (0 to 100) reduces the base focus cost according to an exponential decay curve:
$$\text{Actual Focus} = \text{round}\left(\text{Base Focus} \times 0.5^{\frac{\text{Mastery}}{100}}\right)$$
*(Enforced ceiling: minimum 1 focus point per unit).*

Silver Per Focus evaluates the marginal silver generated per focus point burned:
$$\text{SPF} = \frac{\text{Net Profit}_{\text{with focus}} - \text{Net Profit}_{\text{without focus}}}{\text{Actual Focus}}$$
$$\text{Daily Focus Silver} = \text{SPF} \times 10,000$$

### 8.4 Recursive Chained Refining Cost
Under chained refining, intermediate tier products are not purchased from the market; instead, their production cost is computed bottom-up starting from T4:
$$\text{Cost}_{T} = \left( Q_{\text{raw}, T} \times P_{\text{raw}, T} + Q_{\text{prev}, T} \times \text{Cost}_{T-1} \right) \times (1 - \text{RRR}) + \text{StationFee}_{T}$$
This eliminates market sales tax and setup fee penalties on all intermediate stages.

### 8.5 Arbitrage Net Revenue & Margin
$$\text{Gross Revenue} = P_{\text{sell}}$$
$$\text{Sales Tax} = P_{\text{sell}} \times R_{\text{tax}}$$
$$\text{Setup Fee} = P_{\text{sell}} \times R_{\text{setup}}$$
$$\text{Net Revenue} = \text{Gross Revenue} - \text{Sales Tax} - \text{Setup Fee}$$
$$\text{Net Profit} = \text{Net Revenue} - P_{\text{buy}}$$
$$\text{Net Margin \%} = \frac{\text{Net Profit}}{P_{\text{buy}}} \times 100$$

| Tax Channel | Premium Status | Sales Tax ($R_{\text{tax}}$) | Setup Fee ($R_{\text{setup}}$) | Total Deductions |
|---|---|---|---|---|
| **Direct Trade** | Any | 0.00 (0%) | 0.00 (0%) | **0.0%** |
| **Instant Buy** | Premium | 0.04 (4%) | 0.00 (0%) | **4.0%** |
| **Instant Buy** | Non-Premium | 0.08 (8%) | 0.00 (0%) | **8.0%** |
| **Sell Order** | Premium | 0.04 (4%) | 0.025 (2.5%) | **6.5%** |
| **Sell Order** | Non-Premium | 0.08 (8%) | 0.025 (2.5%) | **10.5%** |

---

## 9. Testing & Verification Runbook

### Running All Backend Tests
```bash
cd backend
go test -v ./...
```
Expected output: All unit tests across `pkg/model`, `pkg/store`, `pkg/engine`, `pkg/aodp`, and `cmd/server` must pass with `0 failures`.

### Running Full Workflow E2E Tests
```bash
cd backend
go test -v ./tests/e2e/...
```
Verifies end-to-end price ingestion, RRR refund accounting, chained refining cost recursion, data age classification, and multi-city arbitrage route generation.

### Running Frontend Type Checks & Build
```bash
cd frontend
pnpm build
```
Ensures all TypeScript contracts, Astro page components, and client-side modules compile cleanly with zero errors.

---

## 10. Operational Troubleshooting Guide

### Issue 1: Port `8081` Already in Use
**Symptom:** `bind: address already in use` upon starting backend.  
**Resolution:** Check active listeners and terminate or override port:
```bash
# Check PID occupying port 8081
lsof -i :8081

# Override port via environment variable
PORT=8082 go run ./cmd/server/main.go
```

### Issue 2: SQLite `database is locked`
**Symptom:** SQLite operations return error `database is locked`.  
**Resolution:**
1. Verify WAL mode is active:
   ```bash
   sqlite3 data/market.db "PRAGMA journal_mode;"
   # Must return: wal
   ```
2. Check if a dangling process holds an exclusive transaction lock:
   ```bash
   fuser data/market.db
   ```
3. The store driver sets `PRAGMA busy_timeout = 5000;` automatically on startup to retry busy locks for up to 5 seconds.

### Issue 3: AODP 429 Too Many Requests / Network Downtime
**Symptom:** `failed to ingest market data: 429 Too Many Requests` or connection timeout during `/api/v1/prices/refresh`.  
**Resolution:**
1. AODP community servers apply per-IP rate limits. Ensure batch chunking size is at least 50 items and pause 100ms between calls.
2. In production, maintain local cached prices in `data/market.db`. The system continues serving calculations from existing snapshots when external ingestion is temporarily throttled.

### Issue 4: Empty Refining Matrix or Zero Routes Found
**Symptom:** Dashboard shows 0 items or no arbitrage routes.  
**Resolution:**
1. Check if the database has seeded price snapshots for the chosen region:
   ```bash
   sqlite3 data/market.db "SELECT count(*) FROM price_snapshots WHERE region='asia';"
   ```
2. If count is 0, trigger a price refresh:
   ```bash
   curl -X POST "http://localhost:8081/api/v1/prices/refresh?region=asia"
   ```
3. Ensure filters (e.g. Min Profit, Min Margin) are not set higher than available market spreads.
