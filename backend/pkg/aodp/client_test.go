package aodp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"albionflow/backend/pkg/aodp"
	"albionflow/backend/pkg/store"
)

func TestFetchPrices_Success(t *testing.T) {
	mockPrices := []aodp.PriceResponseItem{
		{
			ItemID:           "T4_WOOD",
			City:             "Fort Sterling",
			Quality:          1,
			SellPriceMin:     120,
			SellPriceMinDate: "2026-09-14T12:00:00",
			BuyPriceMax:      110,
			BuyPriceMaxDate:  "2026-09-14T12:00:00",
		},
		{
			ItemID:           "T4_WOOD",
			City:             "Thetford",
			Quality:          1,
			SellPriceMin:     140,
			SellPriceMinDate: "2026-09-14T12:05:00",
			BuyPriceMax:      130,
			BuyPriceMaxDate:  "2026-09-14T12:05:00",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v2/stats/prices/T4_WOOD,T5_WOOD.json") {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		locations := r.URL.Query().Get("locations")
		if locations != "Fort Sterling,Thetford" {
			t.Errorf("unexpected locations query: %q", locations)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockPrices)
	}))
	defer server.Close()

	client := aodp.NewClient(aodp.WithBaseURL("asia", server.URL))

	prices, err := client.FetchPrices(context.Background(), "asia", []string{"T4_WOOD", "T5_WOOD"}, []string{"Fort Sterling", "Thetford"})
	if err != nil {
		t.Fatalf("FetchPrices failed: %v", err)
	}

	if len(prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(prices))
	}
	if prices[0].ItemID != "T4_WOOD" || prices[0].City != "Fort Sterling" || prices[0].SellPriceMin != 120 {
		t.Errorf("unexpected price item 0: %+v", prices[0])
	}
	if prices[1].City != "Thetford" || prices[1].SellPriceMin != 140 {
		t.Errorf("unexpected price item 1: %+v", prices[1])
	}
}

func TestFetchPrices_EmptyItems(t *testing.T) {
	client := aodp.NewClient()
	prices, err := client.FetchPrices(context.Background(), "asia", nil, []string{"Thetford"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prices) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(prices))
	}
}

func TestFetchPrices_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("502 Bad Gateway"))
	}))
	defer server.Close()

	client := aodp.NewClient(aodp.WithBaseURL("asia", server.URL))
	_, err := client.FetchPrices(context.Background(), "asia", []string{"T4_WOOD"}, nil)
	if err == nil {
		t.Fatal("expected error on 502 status, got nil")
	}
}

func TestFetch24hVolumes_Success(t *testing.T) {
	mockHistory := []aodp.HistoryResponseItem{
		{
			ItemID:   "T4_WOOD",
			Location: "Thetford",
			Quality:  1,
			Data: []aodp.HistoryDataPoint{
				{
					ItemCount:    100,
					SilverAmount: 12000,
					Timestamp:    "2026-09-13T00:00:00",
				},
				{
					ItemCount:    450,
					SilverAmount: 54000,
					Timestamp:    "2026-09-14T00:00:00",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v2/stats/history/T4_WOOD.json") {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		q := r.URL.Query()
		if q.Get("time-scale") != "24" {
			t.Errorf("expected time-scale=24, got %q", q.Get("time-scale"))
		}
		if q.Get("locations") != "Thetford" {
			t.Errorf("expected locations=Thetford, got %q", q.Get("locations"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockHistory)
	}))
	defer server.Close()

	client := aodp.NewClient(aodp.WithBaseURL("asia", server.URL))

	history, err := client.Fetch24hVolumes(context.Background(), "asia", []string{"T4_WOOD"}, []string{"Thetford"})
	if err != nil {
		t.Fatalf("Fetch24hVolumes failed: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(history))
	}
	if history[0].ItemID != "T4_WOOD" || history[0].Location != "Thetford" {
		t.Errorf("unexpected history item: %+v", history[0])
	}
	if len(history[0].Data) != 2 {
		t.Errorf("expected 2 data points, got %d", len(history[0].Data))
	}
}

func TestFetch24hVolumes_EmptyItems(t *testing.T) {
	client := aodp.NewClient()
	history, err := client.Fetch24hVolumes(context.Background(), "asia", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(history))
	}
}

func TestInvalidRegion(t *testing.T) {
	client := aodp.NewClient()

	_, err := client.FetchPrices(context.Background(), "unknown_region", []string{"T4_WOOD"}, nil)
	if err == nil {
		t.Error("expected error for invalid region on FetchPrices, got nil")
	}

	_, err = client.Fetch24hVolumes(context.Background(), "unknown_region", []string{"T4_WOOD"}, nil)
	if err == nil {
		t.Error("expected error for invalid region on Fetch24hVolumes, got nil")
	}

	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	_, _, err = client.IngestToStore(context.Background(), s, "unknown_region", []string{"T4_WOOD"}, nil)
	if err == nil {
		t.Error("expected error for invalid region on IngestToStore, got nil")
	}
}

func TestIngestToStore_SuccessAndZeroFilter(t *testing.T) {
	mockPrices := []aodp.PriceResponseItem{
		{
			ItemID:           "T4_WOOD",
			City:             "Fort Sterling",
			Quality:          1,
			SellPriceMin:     120,
			SellPriceMinDate: "2026-09-14T10:00:00",
			BuyPriceMax:      110,
			BuyPriceMaxDate:  "2026-09-14T10:00:00",
		},
		{
			ItemID:           "T4_WOOD",
			City:             "Caerleon",
			Quality:          1,
			SellPriceMin:     0,
			SellPriceMinDate: "0001-01-01T00:00:00",
			BuyPriceMax:      0,
			BuyPriceMaxDate:  "0001-01-01T00:00:00",
		},
		{
			ItemID:           "T5_WOOD",
			City:             "Thetford",
			Quality:          1,
			SellPriceMin:     0,
			SellPriceMinDate: "0001-01-01T00:00:00",
			BuyPriceMax:      300,
			BuyPriceMaxDate:  "2026-09-14T10:00:00",
		},
		{
			ItemID:           "T5_WOOD",
			City:             "Lymhurst",
			Quality:          1,
			SellPriceMin:     450,
			SellPriceMinDate: "2026-09-14T10:00:00",
			BuyPriceMax:      0,
			BuyPriceMaxDate:  "0001-01-01T00:00:00",
		},
	}

	mockVolumes := []aodp.HistoryResponseItem{
		{
			ItemID:   "T4_WOOD",
			Location: "Fort Sterling",
			Quality:  1,
			Data: []aodp.HistoryDataPoint{
				{
					ItemCount:    100,
					SilverAmount: 11000,
					Timestamp:    "2026-09-13T00:00:00",
				},
				{
					ItemCount:    280,
					SilverAmount: 32000,
					Timestamp:    "2026-09-14T00:00:00",
				},
			},
		},
		{
			ItemID:   "T5_WOOD",
			Location: "Thetford",
			Quality:  1,
			Data: []aodp.HistoryDataPoint{
				{
					ItemCount:    50,
					SilverAmount: 15000,
					Timestamp:    "2026-09-14T00:00:00",
				},
			},
		},
		{
			ItemID:   "T5_WOOD",
			Location: "Lymhurst",
			Quality:  1,
			Data:     []aodp.HistoryDataPoint{}, // Empty data should be safely skipped
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/api/v2/stats/prices/") {
			_ = json.NewEncoder(w).Encode(mockPrices)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v2/stats/history/") {
			_ = json.NewEncoder(w).Encode(mockVolumes)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to init in-memory store: %v", err)
	}
	defer s.Close()

	client := aodp.NewClient(aodp.WithBaseURL("europe", server.URL))

	pricesCount, volumesCount, err := client.IngestToStore(
		context.Background(),
		s,
		"europe",
		[]string{"T4_WOOD", "T5_WOOD"},
		[]string{"Fort Sterling", "Caerleon", "Thetford", "Lymhurst"},
	)
	if err != nil {
		t.Fatalf("IngestToStore failed: %v", err)
	}

	// 4 items returned, but Caerleon has SellPriceMin=0 and BuyPriceMax=0 -> 3 ingested
	if pricesCount != 3 {
		t.Errorf("expected 3 ingested prices, got %d", pricesCount)
	}

	// 3 history items returned, 1 has empty data -> 2 ingested
	if volumesCount != 2 {
		t.Errorf("expected 2 ingested volumes, got %d", volumesCount)
	}

	// Verify price persisted
	p, err := s.GetLatestPrice(context.Background(), "europe", "T4_WOOD", "Fort Sterling", 1)
	if err != nil {
		t.Fatalf("GetLatestPrice failed: %v", err)
	}
	if p.SellPriceMin != 120 || p.BuyPriceMax != 110 {
		t.Errorf("unexpected price snapshot: %+v", p)
	}

	// Verify zero-price Caerleon was NOT persisted
	pZero, err := s.GetLatestPrice(context.Background(), "europe", "T4_WOOD", "Caerleon", 1)
	if err == nil && pZero != nil {
		t.Errorf("expected zero-price item to be absent from store, got %+v", pZero)
	}

	// Verify volume persisted with latest data point (item count 280)
	v, err := s.GetMarketVolume(context.Background(), "europe", "T4_WOOD", "Fort Sterling", 1)
	if err != nil {
		t.Fatalf("GetMarketVolume failed: %v", err)
	}
	if v.ItemCount != 280 {
		t.Errorf("expected latest volume itemCount 280, got %d", v.ItemCount)
	}
}
