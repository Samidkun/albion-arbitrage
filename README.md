# AlbionFlow



<p align="center">
  <img src="docs/screenshots/preview.png" alt="Application Preview" width="100%" style="border-radius: 10px; border: 1px solid rgba(255, 255, 255, 0.1);" />
</p>

AlbionFlow is a high-precision, real-time market and refining arbitrage engine for Albion Online. It ingests prices and 24-hour trade volumes from The Albion Online Data Project (AODP) across Asia, Americas, and Europe into an embedded SQLite store to identify profitable refining loops and transport arbitrage opportunities while eliminating common player calculation traps (stale data, illiquid volumes, nutrition fee miscalculations, intermediate tax double-counting, and stone enchantment discrepancies).

---

## Documentation

Comprehensive documentation is provided for both game operators and engineering maintainers:

- 📖 **[End-User Guide (Panduan Pengguna)](docs/END_USER_GUIDE.md):** Complete guide for Albion Online crafters, traders, haulers, and guild managers. Explains the 4 fatal traps of generic calculators, Control Bar usage, Silver per Focus (SPF) masterclass, "Show the Math" audit drawers, and cross-city hauling risk management.
- 🛠️ **[Developer Runbook & Technical Manual](docs/DEVELOPER_RUNBOOK.md):** Deep-dive technical manual for engineers. Covers system topology, project directory structure, SQLite WAL configuration, mathematical proofs, AODP ingestion pipelines, E2E test verification, and troubleshooting.

---

## Key Features

- **Anti-Stale Data Guard (<60m):** Classifies price quotes into Fresh (<15m), Aging (15–60m), and Stale (>60m). Eliminates stale orders that lead to silver losses.
- **Silver Per Focus (SPF) Leaderboard:** Evaluates the exact marginal profit of burning 10,000 daily Focus points adjusted for node mastery (0–100).
- **Chained Refining Calculator:** Models multi-tier refining (e.g. T4 -> T5 -> T6) recursively from raw inputs without double-counting intermediate sales taxes.
- **Cross-City Hauling Matrix (`/arbitrage`):** Scans spreads between 6 Royal Cities and Caerleon, tagging routes by 24h market volume liquidity and travel risk tiers (Low/Med/High).
- **"Show the Math" Audit & Discord Export:** Inspect unit-level breakdowns of raw costs, RRR returns, station fees, and export clean markdown summaries directly to Discord.
- **Anti-UI-Slop Dark Console:** Clean dark interface (`#16181C`, hairline `#2A2E35`, electric lime `#C7F53B`) with tabular figures for currency alignment.

---

## Tech Stack

- **Backend:** Go 1.27 + Chi HTTP Router + `modernc.org/sqlite` (Pure Go, WAL mode)
- **Frontend:** Astro 5 + TypeScript + Tailwind CSS (High-density dark console `#16181C`)
- **Data Source:** The Albion Online Data Project (AODP) public community REST APIs
- **Database:** Pure-Go SQLite (`modernc.org/sqlite`, zero CGO dependency)

---

## Quickstart

### Prerequisites
- **Go:** 1.27 or newer (`go version`)
- **Node.js:** v22 or newer (`node -v`)
- **pnpm:** 9 or newer (`pnpm -v`)

### 1. Backend API Server

```bash
cd backend

# Run the HTTP API daemon (listens on port 8081)
go run ./cmd/server/main.go
```

Verify backend health:
```bash
curl http://localhost:8081/healthz
# Response: {"status":"ok","app":"albionflow_backend"}
```

Trigger market price refresh:
```bash
curl -X POST "http://localhost:8081/api/v1/prices/refresh?region=asia"
```

Run test suite:
```bash
# Unit tests
go test -v ./...

# Full end-to-end simulation tests
go test -v ./tests/e2e/...
```

### 2. Frontend Web Console

```bash
cd frontend

# Install dependencies
pnpm install

# Start local development server (port 4321)
pnpm dev
```

Open your browser at `http://localhost:4321` (Refining Matrix) or `http://localhost:4321/arbitrage` (Hauling Matrix).

To generate a static production build:
```bash
cd frontend
pnpm build
```

---

## API Summary

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/healthz` | Backend health check |
| `GET` | `/api/v1/refining/matrix` | Refining profitability matrix with SPF ranking |
| `GET` | `/api/v1/arbitrage/routes` | Cross-city hauling arbitrage opportunities |
| `POST` | `/api/v1/prices/refresh` | Ingest fresh prices and 24h trade volumes from AODP |

---

## License & Compliance

AlbionFlow is 100% legal and strictly compliant with Sandbox Interactive's Terms of Service:
- Zero game memory inspection / RAM injection.
- Zero network packet sniffing.
- All market figures are pulled from community-contributed public data via The Albion Online Data Project (AODP).