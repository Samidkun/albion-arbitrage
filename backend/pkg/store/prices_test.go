package store_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"albionflow/backend/pkg/store"
)

func TestNewStoreMemory(t *testing.T) {
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory store: %v", err)
	}
	defer s.Close()

	if s.DB() == nil {
		t.Fatal("expected non-nil *sql.DB")
	}
}

func TestNewStoreFileCreation(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "subdir", "test_market.db")

	s, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("expected db file to exist at %s", dbPath)
	}
}

func TestUpsertAndGetLatestPrice(t *testing.T) {
	ctx := context.Background()
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	p := &store.PriceSnapshot{
		Region:           "Europe",
		ItemID:           "T4_BAG",
		City:             "Thetford",
		Quality:          1,
		SellPriceMin:     1500,
		SellPriceMinDate: "2026-09-14T10:00:00",
		BuyPriceMax:      1200,
		BuyPriceMaxDate:  "2026-09-14T09:50:00",
	}

	if err := s.UpsertPriceSnapshot(ctx, p); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	got, err := s.GetLatestPrice(ctx, "Europe", "T4_BAG", "Thetford", 1)
	if err != nil {
		t.Fatalf("get latest price failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected price snapshot, got nil")
	}

	if got.Region != "Europe" || got.ItemID != "T4_BAG" || got.City != "Thetford" || got.Quality != 1 {
		t.Errorf("metadata mismatch: %+v", got)
	}
	if got.SellPriceMin != 1500 || got.SellPriceMinDate != "2026-09-14T10:00:00" {
		t.Errorf("sell price mismatch: got %d at %s", got.SellPriceMin, got.SellPriceMinDate)
	}
	if got.BuyPriceMax != 1200 || got.BuyPriceMaxDate != "2026-09-14T09:50:00" {
		t.Errorf("buy price mismatch: got %d at %s", got.BuyPriceMax, got.BuyPriceMaxDate)
	}
	if got.RecordedAt.IsZero() {
		t.Error("expected non-zero recorded_at timestamp")
	}

	// Update on conflict
	pUpdate := &store.PriceSnapshot{
		Region:           "Europe",
		ItemID:           "T4_BAG",
		City:             "Thetford",
		Quality:          1,
		SellPriceMin:     1450,
		SellPriceMinDate: "2026-09-14T10:15:00",
		BuyPriceMax:      1250,
		BuyPriceMaxDate:  "2026-09-14T10:10:00",
	}

	if err := s.UpsertPriceSnapshot(ctx, pUpdate); err != nil {
		t.Fatalf("upsert update failed: %v", err)
	}

	gotUpdated, err := s.GetLatestPrice(ctx, "Europe", "T4_BAG", "Thetford", 1)
	if err != nil {
		t.Fatalf("get latest price after update failed: %v", err)
	}
	if gotUpdated.SellPriceMin != 1450 || gotUpdated.SellPriceMinDate != "2026-09-14T10:15:00" {
		t.Errorf("expected updated sell price 1450, got %d", gotUpdated.SellPriceMin)
	}
	if gotUpdated.BuyPriceMax != 1250 || gotUpdated.BuyPriceMaxDate != "2026-09-14T10:10:00" {
		t.Errorf("expected updated buy price 1250, got %d", gotUpdated.BuyPriceMax)
	}
}

func TestGetPricesForItems(t *testing.T) {
	ctx := context.Background()
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	snapshots := []*store.PriceSnapshot{
		{Region: "Europe", ItemID: "T4_ORE", City: "Thetford", Quality: 1, SellPriceMin: 100, SellPriceMinDate: "2026-09-14T10:00:00"},
		{Region: "Europe", ItemID: "T4_ORE", City: "Fort Sterling", Quality: 1, SellPriceMin: 120, SellPriceMinDate: "2026-09-14T10:00:00"},
		{Region: "Europe", ItemID: "T5_ORE", City: "Thetford", Quality: 1, SellPriceMin: 300, SellPriceMinDate: "2026-09-14T10:00:00"},
		{Region: "Europe", ItemID: "T5_ORE", City: "Lymhurst", Quality: 1, SellPriceMin: 350, SellPriceMinDate: "2026-09-14T10:00:00"},
		{Region: "Asia", ItemID: "T4_ORE", City: "Thetford", Quality: 1, SellPriceMin: 90, SellPriceMinDate: "2026-09-14T10:00:00"},
	}

	for _, snap := range snapshots {
		if err := s.UpsertPriceSnapshot(ctx, snap); err != nil {
			t.Fatalf("upsert snapshot failed: %v", err)
		}
	}

	// Filter by items and cities
	results, err := s.GetPricesForItems(ctx, "Europe", []string{"T4_ORE", "T5_ORE"}, []string{"Thetford", "Fort Sterling"})
	if err != nil {
		t.Fatalf("query items failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results (T4 Thetford, T4 Fort Sterling, T5 Thetford), got %d", len(results))
	}

	// Filter by items with all cities (nil / empty cities slice)
	allCities, err := s.GetPricesForItems(ctx, "Europe", []string{"T4_ORE"}, nil)
	if err != nil {
		t.Fatalf("query items across all cities failed: %v", err)
	}
	if len(allCities) != 2 {
		t.Fatalf("expected 2 results for T4_ORE in Europe across all cities, got %d", len(allCities))
	}

	// Empty items slice returns empty list
	emptyItems, err := s.GetPricesForItems(ctx, "Europe", []string{}, nil)
	if err != nil {
		t.Fatalf("empty items query failed: %v", err)
	}
	if len(emptyItems) != 0 {
		t.Fatalf("expected 0 results, got %d", len(emptyItems))
	}

	// Different region returns empty list
	americas, err := s.GetPricesForItems(ctx, "Americas", []string{"T4_ORE"}, nil)
	if err != nil {
		t.Fatalf("americas query failed: %v", err)
	}
	if len(americas) != 0 {
		t.Fatalf("expected 0 results for Americas, got %d", len(americas))
	}
}

func TestPriceAgeAndStaleness(t *testing.T) {
	baseTime := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		sellPriceMinDate string
		recordedAt       time.Time
		checkTime        time.Time
		threshold        time.Duration
		expectedAge      time.Duration
		expectedStale    bool
	}{
		{
			name:             "fresh within 30m threshold",
			sellPriceMinDate: "2026-09-14T11:45:00",
			recordedAt:       baseTime,
			checkTime:        baseTime,
			threshold:        60 * time.Minute,
			expectedAge:      15 * time.Minute,
			expectedStale:    false,
		},
		{
			name:             "stale older than 60m threshold",
			sellPriceMinDate: "2026-09-14T10:00:00",
			recordedAt:       baseTime,
			checkTime:        baseTime,
			threshold:        60 * time.Minute,
			expectedAge:      120 * time.Minute,
			expectedStale:    true,
		},
		{
			name:             "RFC3339 parsed correctly",
			sellPriceMinDate: "2026-09-14T11:30:00Z",
			recordedAt:       baseTime,
			checkTime:        baseTime,
			threshold:        60 * time.Minute,
			expectedAge:      30 * time.Minute,
			expectedStale:    false,
		},
		{
			name:             "zero date 0001-01-01 is marked stale",
			sellPriceMinDate: "0001-01-01T00:00:00",
			recordedAt:       baseTime.Add(-10 * time.Minute),
			checkTime:        baseTime,
			threshold:        60 * time.Minute,
			expectedAge:      10 * time.Minute, // falls back to RecordedAt
			expectedStale:    true,             // zero date is always stale
		},
		{
			name:             "empty date is marked stale",
			sellPriceMinDate: "",
			recordedAt:       baseTime.Add(-5 * time.Minute),
			checkTime:        baseTime,
			threshold:        60 * time.Minute,
			expectedAge:      5 * time.Minute, // falls back to RecordedAt
			expectedStale:    true,            // empty date is always stale
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &store.PriceSnapshot{
				SellPriceMinDate: tc.sellPriceMinDate,
				RecordedAt:       tc.recordedAt,
			}

			age := p.Age(tc.checkTime)
			if age != tc.expectedAge {
				t.Errorf("Age: expected %v, got %v", tc.expectedAge, age)
			}

			stale := p.IsStale(tc.checkTime, tc.threshold)
			if stale != tc.expectedStale {
				t.Errorf("IsStale: expected %v, got %v", tc.expectedStale, stale)
			}
		})
	}
}

func TestGetLatestPriceNotFound(t *testing.T) {
	ctx := context.Background()
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	_, err = s.GetLatestPrice(ctx, "Europe", "NON_EXISTENT", "Thetford", 1)
	if err == nil {
		t.Fatal("expected error for non-existent item, got nil")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
