package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"albionflow/backend/pkg/aodp"
	"albionflow/backend/pkg/engine"
	"albionflow/backend/pkg/model"
	"albionflow/backend/pkg/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create test in-memory store: %v", err)
	}
	return s
}

func TestHandleRefiningMatrix_Success(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	// Seed Wood prices in Fort Sterling (Bonus City for Wood)
	// T4 Wood
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T4_WOOD",
		City:             "Fort Sterling",
		Quality:          1,
		SellPriceMin:     100,
		SellPriceMinDate: nowStr,
	})
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T3_PLANKS",
		City:             "Fort Sterling",
		Quality:          1,
		SellPriceMin:     50,
		SellPriceMinDate: nowStr,
	})
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T4_PLANKS",
		City:             "Fort Sterling",
		Quality:          1,
		SellPriceMin:     350,
		SellPriceMinDate: nowStr,
	})
	// T4 Planks 24h Volume
	_ = s.UpsertMarketVolume(ctx, &store.MarketVolume{
		Region:       "asia",
		ItemID:       "T4_PLANKS",
		City:         "Fort Sterling",
		Quality:      1,
		ItemCount:    1200,
		SilverAmount: 420000,
		Timestamp:    nowStr,
	})

	state := &ServerState{
		Store:      s,
		AODPClient: aodp.NewClient(),
	}
	router := setupRouter(state)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/refining/matrix?region=asia&commodity=Wood&tier=4&tax=600&channel=sell_order&premium=true&mastery=50", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected application/json, got %s", contentType)
	}

	var items []RefiningMatrixItem
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item for T4 wood, got %d", len(items))
	}

	item := items[0]
	if item.ItemID != "T4_PLANKS" {
		t.Errorf("expected T4_PLANKS, got %s", item.ItemID)
	}
	if item.Commodity != model.CommodityWood {
		t.Errorf("expected Wood, got %s", item.Commodity)
	}
	if item.Tier != 4 {
		t.Errorf("expected Tier 4, got %d", item.Tier)
	}
	if item.City != "Fort Sterling" {
		t.Errorf("expected Fort Sterling, got %s", item.City)
	}
	if !item.IsBonusCity {
		t.Errorf("expected IsBonusCity=true")
	}
	if item.TradedVolume24h != 1200 {
		t.Errorf("expected TradedVolume24h=1200, got %d", item.TradedVolume24h)
	}
	if item.IsIlliquid {
		t.Errorf("expected IsIlliquid=false for 1200 volume")
	}
	if item.Freshness != aodp.FreshnessFresh {
		t.Errorf("expected fresh, got %s", item.Freshness)
	}
	if item.IsStale {
		t.Errorf("expected IsStale=false")
	}
	if item.ProfitWithoutFocus <= 0 {
		t.Errorf("expected positive profit without focus, got %f", item.ProfitWithoutFocus)
	}
	if item.ProfitWithFocus <= item.ProfitWithoutFocus {
		t.Errorf("expected profit with focus > without focus")
	}
	if item.SilverPerFocus <= 0 {
		t.Errorf("expected positive SPF, got %f", item.SilverPerFocus)
	}
	if item.BaseFocusCost != 34 {
		t.Errorf("expected T4.0 base focus 34, got %d", item.BaseFocusCost)
	}
	// At mastery 50, actual focus should be less than 34
	if item.ActualFocusCost >= 34 {
		t.Errorf("expected actual focus < 34, got %d", item.ActualFocusCost)
	}
}

func TestHandleRefiningMatrix_ChainedRefining(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	ctx := context.Background()
	nowStr := time.Now().UTC().Format(time.RFC3339)

	// Seed Wood T4 and T5 in Fort Sterling
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T4_WOOD",
		City:             "Fort Sterling",
		SellPriceMin:     100,
		SellPriceMinDate: nowStr,
	})
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T3_PLANKS",
		City:             "Fort Sterling",
		SellPriceMin:     50,
		SellPriceMinDate: nowStr,
	})
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T4_PLANKS",
		City:             "Fort Sterling",
		SellPriceMin:     400, // Market price of T4 planks is high
		SellPriceMinDate: nowStr,
	})
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T5_WOOD",
		City:             "Fort Sterling",
		SellPriceMin:     250,
		SellPriceMinDate: nowStr,
	})
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T5_PLANKS",
		City:             "Fort Sterling",
		SellPriceMin:     1200,
		SellPriceMinDate: nowStr,
	})

	state := &ServerState{
		Store:      s,
		AODPClient: aodp.NewClient(),
	}
	router := setupRouter(state)

	// Request unchained T5
	reqUnchained, _ := http.NewRequest(http.MethodGet, "/api/v1/refining/matrix?region=asia&commodity=Wood&tier=5&chained=false", nil)
	rrUnchained := httptest.NewRecorder()
	router.ServeHTTP(rrUnchained, reqUnchained)

	var unchainedItems []RefiningMatrixItem
	_ = json.Unmarshal(rrUnchained.Body.Bytes(), &unchainedItems)
	if len(unchainedItems) != 1 {
		t.Fatalf("expected 1 unchained item, got %d", len(unchainedItems))
	}

	// Request chained T5
	reqChained, _ := http.NewRequest(http.MethodGet, "/api/v1/refining/matrix?region=asia&commodity=Wood&tier=5&chained=true", nil)
	rrChained := httptest.NewRecorder()
	router.ServeHTTP(rrChained, reqChained)

	var chainedItems []RefiningMatrixItem
	_ = json.Unmarshal(rrChained.Body.Bytes(), &chainedItems)
	if len(chainedItems) != 1 {
		t.Fatalf("expected 1 chained item, got %d", len(chainedItems))
	}

	// Chained refining should have lower craft cost and higher net profit
	if chainedItems[0].TotalCraftCost >= unchainedItems[0].TotalCraftCost {
		t.Errorf("chained total craft cost (%f) should be strictly less than unchained (%f)",
			chainedItems[0].TotalCraftCost, unchainedItems[0].TotalCraftCost)
	}
	if chainedItems[0].ProfitWithoutFocus <= unchainedItems[0].ProfitWithoutFocus {
		t.Errorf("chained profit (%f) should be strictly greater than unchained (%f)",
			chainedItems[0].ProfitWithoutFocus, unchainedItems[0].ProfitWithoutFocus)
	}
	if !chainedItems[0].IsChained {
		t.Errorf("expected IsChained=true")
	}
}

func TestHandleRefiningMatrix_TaxChannelsAndSort(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	ctx := context.Background()
	nowStr := time.Now().UTC().Format(time.RFC3339)

	// Wood T4
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T4_WOOD",
		City:             "Fort Sterling",
		SellPriceMin:     100,
		SellPriceMinDate: nowStr,
	})
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T3_PLANKS",
		City:             "Fort Sterling",
		SellPriceMin:     50,
		SellPriceMinDate: nowStr,
	})
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T4_PLANKS",
		City:             "Fort Sterling",
		SellPriceMin:     300,
		BuyPriceMax:      280,
		SellPriceMinDate: nowStr,
		BuyPriceMaxDate:  nowStr,
	})

	// Ore T4 in Thetford
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T4_ORE",
		City:             "Thetford",
		SellPriceMin:     100,
		SellPriceMinDate: nowStr,
	})
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T3_METALBAR",
		City:             "Thetford",
		SellPriceMin:     50,
		SellPriceMinDate: nowStr,
	})
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T4_METALBAR",
		City:             "Thetford",
		SellPriceMin:     450,
		BuyPriceMax:      420,
		SellPriceMinDate: nowStr,
		BuyPriceMaxDate:  nowStr,
	})

	state := &ServerState{
		Store:      s,
		AODPClient: aodp.NewClient(),
	}
	router := setupRouter(state)

	// Instant buy channel test
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/refining/matrix?region=asia&tier=4&channel=instant_buy", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var items []RefiningMatrixItem
	_ = json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Verify items are sorted by SilverPerFocus descending
	if items[0].SilverPerFocus < items[1].SilverPerFocus {
		t.Errorf("items not sorted by SPF descending: %f < %f", items[0].SilverPerFocus, items[1].SilverPerFocus)
	}

	// Verify sell price matches BuyPriceMax in instant_buy channel
	for _, it := range items {
		if it.ItemID == "T4_PLANKS" && it.SellPrice != 280 {
			t.Errorf("expected sell price 280 (BuyPriceMax) for instant_buy, got %f", it.SellPrice)
		}
		if it.ItemID == "T4_METALBAR" && it.SellPrice != 420 {
			t.Errorf("expected sell price 420 (BuyPriceMax) for instant_buy, got %f", it.SellPrice)
		}
	}
}

func TestHandleRefiningMatrix_ValidationErrors(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	state := &ServerState{
		Store:      s,
		AODPClient: aodp.NewClient(),
	}
	router := setupRouter(state)

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantErrSub string
	}{
		{
			name:       "invalid region",
			query:      "?region=mars",
			wantStatus: http.StatusBadRequest,
			wantErrSub: "unsupported region",
		},
		{
			name:       "invalid commodity",
			query:      "?commodity=Diamonds",
			wantStatus: http.StatusBadRequest,
			wantErrSub: "invalid commodity",
		},
		{
			name:       "invalid tier low",
			query:      "?tier=3",
			wantStatus: http.StatusBadRequest,
			wantErrSub: "invalid tier",
		},
		{
			name:       "invalid tier high",
			query:      "?tier=9",
			wantStatus: http.StatusBadRequest,
			wantErrSub: "invalid tier",
		},
		{
			name:       "invalid negative tax",
			query:      "?tax=-100",
			wantStatus: http.StatusBadRequest,
			wantErrSub: "invalid tax",
		},
		{
			name:       "invalid channel",
			query:      "?channel=black_market",
			wantStatus: http.StatusBadRequest,
			wantErrSub: "invalid channel",
		},
		{
			name:       "invalid mastery high",
			query:      "?mastery=150",
			wantStatus: http.StatusBadRequest,
			wantErrSub: "invalid mastery",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/refining/matrix"+tc.query, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, rr.Code)
			}
			if !strings.Contains(rr.Body.String(), tc.wantErrSub) {
				t.Errorf("expected body to contain %q, got %s", tc.wantErrSub, rr.Body.String())
			}
		})
	}
}

func TestHandleArbitrageRoutes_Success(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	// Wood T4: Cheap in Thetford, Expensive in Fort Sterling
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T4_WOOD",
		City:             "Thetford",
		Quality:          1,
		SellPriceMin:     100,
		SellPriceMinDate: nowStr,
	})
	_ = s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
		Region:           "asia",
		ItemID:           "T4_WOOD",
		City:             "Fort Sterling",
		Quality:          1,
		SellPriceMin:     300,
		SellPriceMinDate: nowStr,
	})
	// Volume in Fort Sterling
	_ = s.UpsertMarketVolume(ctx, &store.MarketVolume{
		Region:       "asia",
		ItemID:       "T4_WOOD",
		City:         "Fort Sterling",
		Quality:      1,
		ItemCount:    500,
		SilverAmount: 150000,
		Timestamp:    nowStr,
	})

	state := &ServerState{
		Store:      s,
		AODPClient: aodp.NewClient(),
	}
	router := setupRouter(state)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/arbitrage/routes?region=asia&min_profit=10&exclude_stale=true&exclude_illiquid=true", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var routes []*engine.ArbitrageRoute
	if err := json.Unmarshal(rr.Body.Bytes(), &routes); err != nil {
		t.Fatalf("unmarshal routes: %v", err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}

	r := routes[0]
	if r.ItemID != "T4_WOOD" {
		t.Errorf("expected item T4_WOOD, got %s", r.ItemID)
	}
	if r.OriginCity != "Thetford" {
		t.Errorf("expected origin Thetford, got %s", r.OriginCity)
	}
	if r.TargetCity != "Fort Sterling" {
		t.Errorf("expected target Fort Sterling, got %s", r.TargetCity)
	}
	if r.BuyPrice != 100 {
		t.Errorf("expected buy price 100, got %d", r.BuyPrice)
	}
	if r.SellPrice != 300 {
		t.Errorf("expected sell price 300, got %d", r.SellPrice)
	}
	if r.NetProfitPerUnit <= 0 {
		t.Errorf("expected positive net profit, got %f", r.NetProfitPerUnit)
	}
	if r.TradedVolume24h != 500 {
		t.Errorf("expected 500 volume, got %d", r.TradedVolume24h)
	}
	if r.IsIlliquid {
		t.Errorf("expected IsIlliquid=false")
	}
}

func TestHandleArbitrageRoutes_ValidationAndFilters(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	state := &ServerState{
		Store:      s,
		AODPClient: aodp.NewClient(),
	}
	router := setupRouter(state)

	// Defaults test: empty query should return 200 OK with empty array
	reqDefaults, _ := http.NewRequest(http.MethodGet, "/api/v1/arbitrage/routes", nil)
	rrDefaults := httptest.NewRecorder()
	router.ServeHTTP(rrDefaults, reqDefaults)
	if rrDefaults.Code != http.StatusOK {
		t.Errorf("expected 200 for empty defaults, got %d", rrDefaults.Code)
	}
	var emptyList []*engine.ArbitrageRoute
	if err := json.Unmarshal(rrDefaults.Body.Bytes(), &emptyList); err != nil {
		t.Fatalf("unmarshal empty list: %v", err)
	}
	if len(emptyList) != 0 {
		t.Errorf("expected empty array, got %d items", len(emptyList))
	}

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantSub    string
	}{
		{
			name:       "invalid region",
			query:      "?region=atlantis",
			wantStatus: http.StatusBadRequest,
			wantSub:    "unsupported region",
		},
		{
			name:       "negative min_profit",
			query:      "?min_profit=-5",
			wantStatus: http.StatusBadRequest,
			wantSub:    "invalid min_profit",
		},
		{
			name:       "negative min_margin",
			query:      "?min_margin=-10",
			wantStatus: http.StatusBadRequest,
			wantSub:    "invalid min_margin",
		},
		{
			name:       "invalid channel",
			query:      "?channel=discord_illegal",
			wantStatus: http.StatusBadRequest,
			wantSub:    "invalid channel",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/arbitrage/routes"+tc.query, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, rr.Code)
			}
			if !strings.Contains(rr.Body.String(), tc.wantSub) {
				t.Errorf("expected body to contain %q, got %s", tc.wantSub, rr.Body.String())
			}
		})
	}
}

func TestHandlePriceRefresh_Success(t *testing.T) {
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
	}
	mockVolumes := []aodp.HistoryResponseItem{
		{
			ItemID:   "T4_WOOD",
			Location: "Fort Sterling",
			Quality:  1,
			Data: []aodp.HistoryDataPoint{
				{ItemCount: 300, SilverAmount: 36000, Timestamp: "2026-09-14T12:00:00"},
			},
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

	s := newTestStore(t)
	defer s.Close()

	client := aodp.NewClient(aodp.WithBaseURL("asia", server.URL), aodp.WithBaseURL("americas", server.URL))
	state := &ServerState{
		Store:      s,
		AODPClient: client,
	}
	router := setupRouter(state)

	// 1. Query parameter refresh (?region=asia)
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/prices/refresh?region=asia", nil)
	rr1 := httptest.NewRecorder()
	router.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr1.Code, rr1.Body.String())
	}

	var resp1 map[string]any
	if err := json.Unmarshal(rr1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp1["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp1["status"])
	}
	if resp1["region"] != "asia" {
		t.Errorf("expected region asia, got %v", resp1["region"])
	}
	if resp1["prices_ingested"] != float64(1) {
		t.Errorf("expected prices_ingested 1, got %v", resp1["prices_ingested"])
	}
	if resp1["volumes_ingested"] != float64(1) {
		t.Errorf("expected volumes_ingested 1, got %v", resp1["volumes_ingested"])
	}

	// Verify persistence in SQLite
	p, err := s.GetLatestPrice(context.Background(), "asia", "T4_WOOD", "Fort Sterling", 1)
	if err != nil {
		t.Fatalf("GetLatestPrice failed: %v", err)
	}
	if p.SellPriceMin != 120 {
		t.Errorf("expected SellPriceMin=120, got %d", p.SellPriceMin)
	}

	// 2. Body JSON refresh ({"region": "americas"})
	req2Body := bytes.NewBufferString(`{"region":"americas"}`)
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/prices/refresh", req2Body)
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}

	var resp2 map[string]any
	_ = json.Unmarshal(rr2.Body.Bytes(), &resp2)
	if resp2["region"] != "americas" {
		t.Errorf("expected region americas, got %v", resp2["region"])
	}
}

func TestHandlePriceRefresh_Errors(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	// Upstream error server
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Upstream failure"))
	}))
	defer errorServer.Close()

	client := aodp.NewClient(aodp.WithBaseURL("asia", errorServer.URL))
	state := &ServerState{
		Store:      s,
		AODPClient: client,
	}
	router := setupRouter(state)

	// 1. Unsupported region
	reqBadRegion, _ := http.NewRequest(http.MethodPost, "/api/v1/prices/refresh?region=jupiter", nil)
	rrBadRegion := httptest.NewRecorder()
	router.ServeHTTP(rrBadRegion, reqBadRegion)
	if rrBadRegion.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unsupported region, got %d", rrBadRegion.Code)
	}

	// 2. Upstream 500 triggers 502 Bad Gateway
	reqUpstreamErr, _ := http.NewRequest(http.MethodPost, "/api/v1/prices/refresh?region=asia", nil)
	rrUpstreamErr := httptest.NewRecorder()
	router.ServeHTTP(rrUpstreamErr, reqUpstreamErr)
	if rrUpstreamErr.Code != http.StatusBadGateway {
		t.Errorf("expected 502 for upstream error, got %d", rrUpstreamErr.Code)
	}
}
