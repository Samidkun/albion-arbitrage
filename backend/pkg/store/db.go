package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Store encapsulates the SQLite database connection and operations.
type Store struct {
	db *sql.DB
}

const defaultDBPath = "data/market.db"

const schema = `
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
`

// NewStore opens and initializes the SQLite database with WAL mode and table schemas.
func NewStore(dbPath string) (*Store, error) {
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	isMemory := dbPath == ":memory:" || strings.Contains(dbPath, "mode=memory")
	if !isMemory {
		dir := filepath.Dir(dbPath)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("create db directory %s: %w", dir, err)
			}
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// Configure SQLite PRAGMAs
	if !isMemory {
		if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
			db.Close()
			return nil, fmt.Errorf("set journal_mode WAL: %w", err)
		}
	}

	pragmas := []string{
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA foreign_keys = ON;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("execute pragma %s: %w", p, err)
		}
	}

	// Apply Schema DDL
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("execute schema DDL: %w", err)
	}

	return &Store{db: db}, nil
}

// DB returns the underlying sql.DB instance.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
