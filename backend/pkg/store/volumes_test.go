package store_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"albionflow/backend/pkg/store"
)

func TestMarketVolumesTableInitialization(t *testing.T) {
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory store: %v", err)
	}
	defer s.Close()

	// Verify market_volumes table exists by querying sqlite_master
	var name string
	err = s.DB().QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='market_volumes';").Scan(&name)
	if err != nil {
		t.Fatalf("table market_volumes does not exist: %v", err)
	}
	if name != "market_volumes" {
		t.Fatalf("expected table name market_volumes, got %s", name)
	}
}

func TestUpsertAndGetMarketVolume(t *testing.T) {
	ctx := context.Background()
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory store: %v", err)
	}
	defer s.Close()

	v := &store.MarketVolume{
		Region:       "Europe",
		ItemID:       "T4_BAG",
		City:         "Martlock",
		Quality:      1,
		ItemCount:    120,
		SilverAmount: 360000,
		Timestamp:    "2026-09-14T12:00:00Z",
	}

	if err := s.UpsertMarketVolume(ctx, v); err != nil {
		t.Fatalf("upsert market volume failed: %v", err)
	}

	if v.ID == 0 {
		t.Fatalf("expected non-zero ID after upsert, got %d", v.ID)
	}
	if v.RecordedAt.IsZero() {
		t.Fatalf("expected non-zero RecordedAt after upsert")
	}

	origID := v.ID

	got, err := s.GetMarketVolume(ctx, "Europe", "T4_BAG", "Martlock", 1)
	if err != nil {
		t.Fatalf("get market volume failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil market volume")
	}

	if got.Region != "Europe" || got.ItemID != "T4_BAG" || got.City != "Martlock" || got.Quality != 1 {
		t.Errorf("unexpected key fields: %+v", got)
	}
	if got.ItemCount != 120 || got.SilverAmount != 360000 || got.Timestamp != "2026-09-14T12:00:00Z" {
		t.Errorf("unexpected volume data: %+v", got)
	}

	// Update on conflict
	vUpdate := &store.MarketVolume{
		Region:       "Europe",
		ItemID:       "T4_BAG",
		City:         "Martlock",
		Quality:      1,
		ItemCount:    250,
		SilverAmount: 750000,
		Timestamp:    "2026-09-14T13:00:00Z",
	}

	if err := s.UpsertMarketVolume(ctx, vUpdate); err != nil {
		t.Fatalf("upsert update failed: %v", err)
	}

	if vUpdate.ID != origID {
		t.Errorf("expected ID to remain %d, got %d", origID, vUpdate.ID)
	}

	gotUpdated, err := s.GetMarketVolume(ctx, "Europe", "T4_BAG", "Martlock", 1)
	if err != nil {
		t.Fatalf("get updated market volume failed: %v", err)
	}
	if gotUpdated.ItemCount != 250 || gotUpdated.SilverAmount != 750000 || gotUpdated.Timestamp != "2026-09-14T13:00:00Z" {
		t.Errorf("updated volume mismatch: %+v", gotUpdated)
	}
}

func TestGetMarketVolumeNotFound(t *testing.T) {
	ctx := context.Background()
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory store: %v", err)
	}
	defer s.Close()

	_, err = s.GetMarketVolume(ctx, "Europe", "NON_EXISTENT", "Caerleon", 1)
	if err == nil {
		t.Fatal("expected error for non-existent volume record")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got: %v", err)
	}
}

func TestGetVolumesForItems(t *testing.T) {
	ctx := context.Background()
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory store: %v", err)
	}
	defer s.Close()

	// Empty query
	empty, err := s.GetVolumesForItems(ctx, "Europe", []string{})
	if err != nil {
		t.Fatalf("expected nil error on empty query, got: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(empty))
	}

	records := []*store.MarketVolume{
		{
			Region:       "Europe",
			ItemID:       "T4_BAG",
			City:         "Bridgewatch",
			Quality:      1,
			ItemCount:    50,
			SilverAmount: 150000,
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		},
		{
			Region:       "Europe",
			ItemID:       "T4_BAG",
			City:         "Fort Sterling",
			Quality:      1,
			ItemCount:    75,
			SilverAmount: 225000,
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		},
		{
			Region:       "Europe",
			ItemID:       "T5_BAG",
			City:         "Bridgewatch",
			Quality:      1,
			ItemCount:    30,
			SilverAmount: 180000,
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		},
		{
			Region:       "America",
			ItemID:       "T4_BAG",
			City:         "Bridgewatch",
			Quality:      1,
			ItemCount:    999,
			SilverAmount: 999999,
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		},
	}

	for _, rec := range records {
		if err := s.UpsertMarketVolume(ctx, rec); err != nil {
			t.Fatalf("failed to insert volume: %v", err)
		}
	}

	// Fetch Europe items for T4_BAG and T5_BAG
	vols, err := s.GetVolumesForItems(ctx, "Europe", []string{"T4_BAG", "T5_BAG"})
	if err != nil {
		t.Fatalf("query volumes failed: %v", err)
	}

	if len(vols) != 3 {
		t.Fatalf("expected 3 volume records for Europe, got %d", len(vols))
	}

	for _, v := range vols {
		if v.Region != "Europe" {
			t.Errorf("expected region Europe, got %s", v.Region)
		}
	}
}

func TestMarketVolumeIsIlliquid(t *testing.T) {
	// Nil check
	var nilVol *store.MarketVolume
	if !nilVol.IsIlliquid(5) {
		t.Errorf("nil volume should be considered illiquid")
	}

	// Default minCount threshold (5)
	zeroVol := &store.MarketVolume{ItemCount: 0}
	if !zeroVol.IsIlliquid(0) {
		t.Errorf("count 0 should be illiquid with default threshold")
	}
	if !zeroVol.IsIlliquid(-1) {
		t.Errorf("count 0 should be illiquid with negative threshold")
	}

	fourVol := &store.MarketVolume{ItemCount: 4}
	if !fourVol.IsIlliquid(0) {
		t.Errorf("count 4 should be illiquid with default threshold of 5")
	}

	fiveVol := &store.MarketVolume{ItemCount: 5}
	if fiveVol.IsIlliquid(0) {
		t.Errorf("count 5 should NOT be illiquid with default threshold of 5")
	}

	fiftyVol := &store.MarketVolume{ItemCount: 50}
	if fiftyVol.IsIlliquid(0) {
		t.Errorf("count 50 should NOT be illiquid with default threshold of 5")
	}

	// Custom threshold
	if !fiftyVol.IsIlliquid(100) {
		t.Errorf("count 50 should be illiquid when minCount is 100")
	}
	if fiftyVol.IsIlliquid(50) {
		t.Errorf("count 50 should NOT be illiquid when minCount is 50")
	}
}
