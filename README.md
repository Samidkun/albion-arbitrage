# AlbionFlow

AlbionFlow is a high-precision, real-time market and refining arbitrage engine for Albion Online. It ingests prices and 24-hour trade volumes from The Albion Online Data Project (AODP) across Asia, Americas, and Europe into an embedded SQLite store to identify profitable refining loops and transport arbitrage opportunities while eliminating common player calculation traps (stale data, illiquid volumes, nutrition fee miscalculations, and stone enchantment discrepancies).

## Tech Stack

- **Backend:** Go 1.27 + Chi HTTP Router + `modernc.org/sqlite` (Pure Go, WAL mode)
- **Frontend:** Astro 5 + TypeScript + Tailwind CSS (High-density dark console `#16181C`)
- **Data Source:** The Albion Online Data Project (AODP) public community REST APIs

## Quickstart

### Backend

```bash
cd backend
go run cmd/server/main.go
```

The HTTP API server will start on port `8081` (health check: `http://localhost:8081/healthz`).

To run tests:

```bash
cd backend
go test -v ./...
```

### Frontend

```bash
cd frontend
pnpm install
pnpm dev
```

The web console will be accessible at `http://localhost:4321`.

To create a production build:

```bash
cd frontend
pnpm build
```
