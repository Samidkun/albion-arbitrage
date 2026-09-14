package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"albionflow/backend/pkg/api"
	"albionflow/backend/pkg/aodp"
	"albionflow/backend/pkg/engine"
	"albionflow/backend/pkg/model"
	"albionflow/backend/pkg/store"

	"github.com/go-chi/chi/v5"
)

// setupRouter initializes the API router with a provided store and optional AODP client.
func setupRouter(s *store.Store, client ...*aodp.Client) *chi.Mux {
	var c *aodp.Client
	if len(client) > 0 && client[0] != nil {
		c = client[0]
	} else {
		c = aodp.NewClient()
	}
	return api.SetupRouter(&api.ServerState{
		Store:      s,
		AODPClient: c,
	})
}

// newTestStore creates an isolated in-memory SQLite store.
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to initialize in-memory test store: %v", err)
	}
	return s
}

// seedMultiCityQuotes seeds realistic price snapshots and 24h volumes across all 6 cities.
func seedMultiCityQuotes(t *testing.T, s *store.Store, now time.Time) {
	t.Helper()
	ctx := context.Background()

	tFresh := now.Add(-10 * time.Minute).Format(time.RFC3339)
	tAging := now.Add(-45 * time.Minute).Format(time.RFC3339)
	tStale := now.Add(-85 * time.Minute).Format(time.RFC3339)

	type priceSeed struct {
		city        string
		itemID      string
		sellMin     int
		buyMax      int
		dateStr     string
		volume24h   int64
		isStaleDate bool
	}

	seeds := []priceSeed{
		// Fort Sterling (Wood bonus city)
		{"Fort Sterling", "T3_PLANKS", 50, 45, tFresh, 5000, false},
		{"Fort Sterling", "T4_WOOD", 100, 90, tFresh, 8000, false},
		{"Fort Sterling", "T4_PLANKS", 350, 310, tFresh, 4000, false},
		{"Fort Sterling", "T5_WOOD", 200, 180, tAging, 3500, false},
		{"Fort Sterling", "T5_PLANKS", 800, 720, tAging, 2000, false},

		// Thetford (Ore bonus city)
		{"Thetford", "T3_METALBAR", 40, 35, tFresh, 6000, false},
		{"Thetford", "T4_ORE", 110, 100, tFresh, 9000, false},
		{"Thetford", "T4_METALBAR", 360, 320, tFresh, 4500, false},
		{"Thetford", "T5_ORE", 220, 200, tFresh, 3200, false},
		{"Thetford", "T5_METALBAR", 850, 780, tFresh, 2100, false},

		// Lymhurst (Fiber bonus city)
		{"Lymhurst", "T3_CLOTH", 45, 40, tFresh, 5500, false},
		{"Lymhurst", "T4_FIBER", 105, 95, tFresh, 8500, false},
		{"Lymhurst", "T4_CLOTH", 340, 300, tFresh, 4200, false},
		{"Lymhurst", "T5_FIBER", 210, 190, tFresh, 3100, false},
		{"Lymhurst", "T5_CLOTH", 820, 750, tFresh, 1900, false},

		// Martlock (Hide bonus city)
		{"Martlock", "T3_LEATHER", 48, 42, tFresh, 5200, false},
		{"Martlock", "T4_HIDE", 108, 98, tStale, 7500, true}, // STALE price
		{"Martlock", "T4_LEATHER", 355, 315, tStale, 3800, true},
		{"Martlock", "T5_HIDE", 215, 195, tFresh, 2800, false},
		{"Martlock", "T5_LEATHER", 830, 760, tFresh, 1800, false},
		// Wood in Martlock (non-bonus city for Wood)
		{"Martlock", "T3_PLANKS", 65, 55, tFresh, 2000, false},
		{"Martlock", "T4_WOOD", 140, 125, tFresh, 3000, false},
		{"Martlock", "T4_PLANKS", 420, 380, tFresh, 2500, false},

		// Bridgewatch (Stone bonus city)
		{"Bridgewatch", "T3_STONEBLOCK", 30, 25, tFresh, 4000, false},
		{"Bridgewatch", "T4_ROCK", 70, 60, tFresh, 6000, false},
		{"Bridgewatch", "T4_STONEBLOCK", 220, 190, tFresh, 3500, false},
		{"Bridgewatch", "T5_ROCK", 150, 130, tFresh, 2500, false},
		{"Bridgewatch", "T5_STONEBLOCK", 550, 480, tFresh, 1500, false},
		// Caerleon (High demand / Black Market destination hub)
		{"Caerleon", "T4_WOOD", 180, 160, tFresh, 5000, false},
		{"Caerleon", "T4_PLANKS", 550, 500, tFresh, 4500, false},
		{"Caerleon", "T4_ORE", 190, 170, tFresh, 5200, false},
		{"Caerleon", "T4_METALBAR", 580, 520, tFresh, 4800, false},
		{"Caerleon", "T4_CLOTH", 530, 480, tFresh, 4100, false},
		{"Caerleon", "T4_LEATHER", 560, 510, tFresh, 3900, false},
		{"Caerleon", "T4_STONEBLOCK", 380, 330, tFresh, 3200, false},
		// Stale item in Caerleon
		{"Caerleon", "T4_HIDE", 220, 200, tStale, 2500, true},
		// Low-liquidity trade: T5_STONEBLOCK in Caerleon has volume 2 (< 5)
		{"Caerleon", "T5_STONEBLOCK", 900, 820, tFresh, 2, false},
	}

	for _, seed := range seeds {
		err := s.UpsertPriceSnapshot(ctx, &store.PriceSnapshot{
			Region:           "asia",
			ItemID:           seed.itemID,
			City:             seed.city,
			Quality:          1,
			SellPriceMin:     seed.sellMin,
			SellPriceMinDate: seed.dateStr,
			BuyPriceMax:      seed.buyMax,
			BuyPriceMaxDate:  seed.dateStr,
			RecordedAt:       now,
		})
		if err != nil {
			t.Fatalf("failed to seed price %s %s: %v", seed.itemID, seed.city, err)
		}

		err = s.UpsertMarketVolume(ctx, &store.MarketVolume{
			Region:       "asia",
			ItemID:       seed.itemID,
			City:         seed.city,
			Quality:      1,
			ItemCount:    seed.volume24h,
			SilverAmount: seed.volume24h * int64(seed.sellMin),
			Timestamp:    seed.dateStr,
		})
		if err != nil {
			t.Fatalf("failed to seed volume %s %s: %v", seed.itemID, seed.city, err)
		}
	}
}

// Test Case 1: Refining Matrix Workflow (GET /api/v1/refining/matrix)
func TestE2E_RefiningMatrixWorkflow(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	now := time.Now().UTC()
	seedMultiCityQuotes(t, store, now)
	router := setupRouter(store)

	t.Run("bonus_cities_and_rrr", func(t *testing.T) {
		type bonusTest struct {
			commodity   string
			tier        int
			expectedCity string
			expectedRRR float64
		}

		bonusTests := []bonusTest{
			{"Wood", 4, "Fort Sterling", 0.367},
			{"Ore", 4, "Thetford", 0.367},
			{"Fiber", 4, "Lymhurst", 0.367},
			{"Hide", 4, "Martlock", 0.367},
			{"Stone", 4, "Bridgewatch", 0.367},
		}

		for _, bt := range bonusTests {
			url := fmt.Sprintf("/api/v1/refining/matrix?region=asia&commodity=%s&tier=%d", bt.commodity, bt.tier)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status %d for %s: %s", rr.Code, bt.commodity, rr.Body.String())
			}

			var items []api.RefiningMatrixItem
			if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
				t.Fatalf("unmarshal %s: %v", bt.commodity, err)
			}
			if len(items) == 0 {
				t.Fatalf("expected items for %s tier %d, got 0", bt.commodity, bt.tier)
			}

			item := items[0]
			if item.City != bt.expectedCity {
				t.Errorf("%s bonus city: expected %s, got %s", bt.commodity, bt.expectedCity, item.City)
			}
			if !item.IsBonusCity {
				t.Errorf("%s expected is_bonus_city to be true in %s", bt.commodity, item.City)
			}
			if math.Abs(item.RRR-bt.expectedRRR) > 1e-4 {
				t.Errorf("%s RRR: expected %.3f, got %.3f", bt.commodity, bt.expectedRRR, item.RRR)
			}
		}

		// Non-bonus city check: Wood in Martlock should have RRR = 0.152 and IsBonusCity = false
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/refining/matrix?region=asia&commodity=Wood&tier=4&city=Martlock", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status %d for non-bonus city: %s", rr.Code, rr.Body.String())
		}
		var nonBonusItems []api.RefiningMatrixItem
		if err := json.Unmarshal(rr.Body.Bytes(), &nonBonusItems); err != nil {
			t.Fatalf("unmarshal non-bonus items: %v", err)
		}
		if len(nonBonusItems) == 0 {
			t.Fatalf("expected items for Wood in Martlock")
		}
		item := nonBonusItems[0]
		if item.IsBonusCity {
			t.Errorf("Wood in Martlock should NOT be bonus city")
		}
		if math.Abs(item.RRR-0.152) > 1e-4 {
			t.Errorf("Wood in Martlock expected RRR 0.152, got %.3f", item.RRR)
		}
	})

	t.Run("stone_exception_only_enchantment_zero", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/refining/matrix?region=asia&commodity=Stone", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}

		var items []api.RefiningMatrixItem
		if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
			t.Fatalf("unmarshal stone items: %v", err)
		}
		if len(items) == 0 {
			t.Fatalf("expected stone items, got 0")
		}

		for _, item := range items {
			if item.Commodity != model.CommodityStone {
				t.Errorf("expected Stone commodity, got %s", item.Commodity)
			}
			if item.Enchantment != 0 {
				t.Fatalf("STONE EXCEPTION VIOLATION: Stone item %s has enchantment %d, only 0 is allowed!", item.ItemID, item.Enchantment)
			}
		}
	})

	t.Run("chained_refining_reduces_cost_without_double_taxation", func(t *testing.T) {
		// Non-chained T5 Wood
		reqDirect, _ := http.NewRequest(http.MethodGet, "/api/v1/refining/matrix?region=asia&commodity=Wood&tier=5&chained=false", nil)
		rrDirect := httptest.NewRecorder()
		router.ServeHTTP(rrDirect, reqDirect)

		if rrDirect.Code != http.StatusOK {
			t.Fatalf("status %d direct: %s", rrDirect.Code, rrDirect.Body.String())
		}
		var directItems []api.RefiningMatrixItem
		if err := json.Unmarshal(rrDirect.Body.Bytes(), &directItems); err != nil {
			t.Fatalf("unmarshal direct: %v", err)
		}
		if len(directItems) == 0 {
			t.Fatalf("expected direct items")
		}
		directT5 := directItems[0]

		// Chained T5 Wood
		reqChained, _ := http.NewRequest(http.MethodGet, "/api/v1/refining/matrix?region=asia&commodity=Wood&tier=5&chained=true", nil)
		rrChained := httptest.NewRecorder()
		router.ServeHTTP(rrChained, reqChained)

		if rrChained.Code != http.StatusOK {
			t.Fatalf("status %d chained: %s", rrChained.Code, rrChained.Body.String())
		}
		var chainedItems []api.RefiningMatrixItem
		if err := json.Unmarshal(rrChained.Body.Bytes(), &chainedItems); err != nil {
			t.Fatalf("unmarshal chained: %v", err)
		}
		if len(chainedItems) == 0 {
			t.Fatalf("expected chained items")
		}
		chainedT5 := chainedItems[0]

		if !chainedT5.IsChained {
			t.Errorf("expected is_chained to be true on chained refining result")
		}
		if directT5.IsChained {
			t.Errorf("expected is_chained to be false on non-chained refining result")
		}

		// Because crafting T4 Planks costs ~169 silver, while buying from market costs 350 silver,
		// chained refining must strictly reduce the total craft cost of T5 Planks!
		if chainedT5.TotalCraftCost >= directT5.TotalCraftCost {
			t.Errorf("chained total craft cost (%.2f) must be strictly less than unchained craft cost (%.2f)",
				chainedT5.TotalCraftCost, directT5.TotalCraftCost)
		}
		if chainedT5.ProfitWithoutFocus <= directT5.ProfitWithoutFocus {
			t.Errorf("chained profit without focus (%.2f) must be higher than unchained (%.2f)",
				chainedT5.ProfitWithoutFocus, directT5.ProfitWithoutFocus)
		}
	})

	t.Run("sorted_by_silver_per_focus_descending", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/refining/matrix?region=asia", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}

		var items []api.RefiningMatrixItem
		if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(items) < 2 {
			t.Fatalf("expected at least 2 items, got %d", len(items))
		}

		for i := 0; i < len(items)-1; i++ {
			if items[i].SilverPerFocus < items[i+1].SilverPerFocus {
				t.Errorf("sorting order violation at index %d: item %s (SPF %.2f) < item %s (SPF %.2f)",
					i, items[i].ItemID, items[i].SilverPerFocus, items[i+1].ItemID, items[i+1].SilverPerFocus)
			}
		}
	})

	t.Run("freshness_classification", func(t *testing.T) {
		// Fort Sterling T4 Wood was seeded 10m ago (<30m) -> fresh
		reqFresh, _ := http.NewRequest(http.MethodGet, "/api/v1/refining/matrix?region=asia&commodity=Wood&tier=4", nil)
		rrFresh := httptest.NewRecorder()
		router.ServeHTTP(rrFresh, reqFresh)

		var freshItems []api.RefiningMatrixItem
		_ = json.Unmarshal(rrFresh.Body.Bytes(), &freshItems)
		if len(freshItems) == 0 {
			t.Fatalf("expected fresh items")
		}
		if freshItems[0].Freshness != aodp.FreshnessFresh || freshItems[0].IsStale {
			t.Errorf("expected fresh (<30m), got freshness=%s is_stale=%v", freshItems[0].Freshness, freshItems[0].IsStale)
		}

		// Martlock T4 Hide was seeded 85m ago (>60m) -> stale
		reqStale, _ := http.NewRequest(http.MethodGet, "/api/v1/refining/matrix?region=asia&commodity=Hide&tier=4", nil)
		rrStale := httptest.NewRecorder()
		router.ServeHTTP(rrStale, reqStale)

		var staleItems []api.RefiningMatrixItem
		_ = json.Unmarshal(rrStale.Body.Bytes(), &staleItems)
		if len(staleItems) == 0 {
			t.Fatalf("expected stale items")
		}
		if staleItems[0].Freshness != aodp.FreshnessStale || !staleItems[0].IsStale {
			t.Errorf("expected stale (>60m), got freshness=%s is_stale=%v", staleItems[0].Freshness, staleItems[0].IsStale)
		}
	})
}

// Test Case 2: Cross-City Arbitrage Scanner Workflow (GET /api/v1/arbitrage/routes)
func TestE2E_CrossCityArbitrageScannerWorkflow(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	now := time.Now().UTC()
	seedMultiCityQuotes(t, store, now)
	router := setupRouter(store)

	t.Run("origin_must_not_equal_target_city", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/arbitrage/routes?region=asia&exclude_stale=false&exclude_illiquid=false", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}

		var routes []*engine.ArbitrageRoute
		if err := json.Unmarshal(rr.Body.Bytes(), &routes); err != nil {
			t.Fatalf("unmarshal routes: %v", err)
		}
		if len(routes) == 0 {
			t.Fatalf("expected routes to be returned")
		}

		for _, route := range routes {
			if route.OriginCity == route.TargetCity {
				t.Fatalf("route origin equals target city: %s -> %s for item %s", route.OriginCity, route.TargetCity, route.ItemID)
			}
		}
	})

	t.Run("deducts_flexible_channel_taxes", func(t *testing.T) {
		// 1. Direct channel: 0% tax, 0% setup fee
		reqDirect, _ := http.NewRequest(http.MethodGet, "/api/v1/arbitrage/routes?region=asia&channel=direct&premium=true&exclude_stale=true&exclude_illiquid=true", nil)
		rrDirect := httptest.NewRecorder()
		router.ServeHTTP(rrDirect, reqDirect)
		var directRoutes []*engine.ArbitrageRoute
		_ = json.Unmarshal(rrDirect.Body.Bytes(), &directRoutes)
		if len(directRoutes) == 0 {
			t.Fatalf("expected direct routes")
		}
		for _, r := range directRoutes {
			if r.SalesTaxRate != 0.0 || r.SetupFeeRate != 0.0 {
				t.Errorf("direct channel: expected 0 tax/setup, got salesTax=%.3f setup=%.3f", r.SalesTaxRate, r.SetupFeeRate)
			}
			expectedNetRev := float64(r.SellPrice)
			if math.Abs(r.NetRevenue-expectedNetRev) > 1e-3 {
				t.Errorf("direct net revenue mismatch: got %.2f, expected %.2f", r.NetRevenue, expectedNetRev)
			}
		}

		// 2. Instant Buy channel: 4% tax (premium), 0% setup fee
		reqIB, _ := http.NewRequest(http.MethodGet, "/api/v1/arbitrage/routes?region=asia&channel=instant_buy&premium=true&exclude_stale=true&exclude_illiquid=true", nil)
		rrIB := httptest.NewRecorder()
		router.ServeHTTP(rrIB, reqIB)
		var ibRoutes []*engine.ArbitrageRoute
		_ = json.Unmarshal(rrIB.Body.Bytes(), &ibRoutes)
		if len(ibRoutes) == 0 {
			t.Fatalf("expected instant buy routes")
		}
		for _, r := range ibRoutes {
			if math.Abs(r.SalesTaxRate-0.04) > 1e-4 || r.SetupFeeRate != 0.0 {
				t.Errorf("instant buy premium: expected 0.04 tax, 0 setup; got tax=%.3f setup=%.3f", r.SalesTaxRate, r.SetupFeeRate)
			}
			expectedNetRev := float64(r.SellPrice) * 0.96
			if math.Abs(r.NetRevenue-expectedNetRev) > 1e-3 {
				t.Errorf("instant buy net revenue mismatch: got %.2f, expected %.2f", r.NetRevenue, expectedNetRev)
			}
		}

		// 3. Sell Order channel (premium): 4% tax + 2.5% setup fee
		reqSO, _ := http.NewRequest(http.MethodGet, "/api/v1/arbitrage/routes?region=asia&channel=sell_order&premium=true&exclude_stale=true&exclude_illiquid=true", nil)
		rrSO := httptest.NewRecorder()
		router.ServeHTTP(rrSO, reqSO)
		var soRoutes []*engine.ArbitrageRoute
		_ = json.Unmarshal(rrSO.Body.Bytes(), &soRoutes)
		if len(soRoutes) == 0 {
			t.Fatalf("expected sell order routes")
		}
		for _, r := range soRoutes {
			if math.Abs(r.SalesTaxRate-0.04) > 1e-4 || math.Abs(r.SetupFeeRate-0.025) > 1e-4 {
				t.Errorf("sell order premium: expected 0.04/0.025, got tax=%.3f setup=%.3f", r.SalesTaxRate, r.SetupFeeRate)
			}
			expectedNetRev := float64(r.SellPrice) * (1.0 - 0.04 - 0.025)
			if math.Abs(r.NetRevenue-expectedNetRev) > 1e-3 {
				t.Errorf("sell order premium net revenue mismatch: got %.2f, expected %.2f", r.NetRevenue, expectedNetRev)
			}
		}

		// 4. Sell Order channel (non-premium): 8% tax + 2.5% setup fee
		reqNonPrem, _ := http.NewRequest(http.MethodGet, "/api/v1/arbitrage/routes?region=asia&channel=sell_order&premium=false&exclude_stale=true&exclude_illiquid=true", nil)
		rrNonPrem := httptest.NewRecorder()
		router.ServeHTTP(rrNonPrem, reqNonPrem)
		var nonPremRoutes []*engine.ArbitrageRoute
		_ = json.Unmarshal(rrNonPrem.Body.Bytes(), &nonPremRoutes)
		if len(nonPremRoutes) == 0 {
			t.Fatalf("expected non-premium sell order routes")
		}
		for _, r := range nonPremRoutes {
			if math.Abs(r.SalesTaxRate-0.08) > 1e-4 || math.Abs(r.SetupFeeRate-0.025) > 1e-4 {
				t.Errorf("sell order non-premium: expected 0.08/0.025, got tax=%.3f setup=%.3f", r.SalesTaxRate, r.SetupFeeRate)
			}
			expectedNetRev := float64(r.SellPrice) * (1.0 - 0.08 - 0.025)
			if math.Abs(r.NetRevenue-expectedNetRev) > 1e-3 {
				t.Errorf("sell order non-premium net revenue mismatch: got %.2f, expected %.2f", r.NetRevenue, expectedNetRev)
			}
		}
	})

	t.Run("flags_and_filters_stale_prices", func(t *testing.T) {
		// When exclude_stale=true (default), stale routes (such as T4_HIDE Martlock->Caerleon) must NOT be present
		reqExclude, _ := http.NewRequest(http.MethodGet, "/api/v1/arbitrage/routes?region=asia&exclude_stale=true&exclude_illiquid=false", nil)
		rrExclude := httptest.NewRecorder()
		router.ServeHTTP(rrExclude, reqExclude)

		var cleanRoutes []*engine.ArbitrageRoute
		_ = json.Unmarshal(rrExclude.Body.Bytes(), &cleanRoutes)
		for _, r := range cleanRoutes {
			if r.IsStaleRoute {
				t.Fatalf("found stale route %s (%s -> %s) when exclude_stale=true", r.ItemID, r.OriginCity, r.TargetCity)
			}
		}

		// When exclude_stale=false, stale route must be included and flagged as IsStaleRoute=true
		reqInclude, _ := http.NewRequest(http.MethodGet, "/api/v1/arbitrage/routes?region=asia&exclude_stale=false&exclude_illiquid=false", nil)
		rrInclude := httptest.NewRecorder()
		router.ServeHTTP(rrInclude, reqInclude)

		var allRoutes []*engine.ArbitrageRoute
		_ = json.Unmarshal(rrInclude.Body.Bytes(), &allRoutes)
		foundStale := false
		for _, r := range allRoutes {
			if r.IsStaleRoute {
				foundStale = true
				if !r.IsBuyStale && !r.IsSellStale {
					t.Errorf("stale route %s should have IsBuyStale or IsSellStale set to true", r.ItemID)
				}
			}
		}
		if !foundStale {
			t.Errorf("expected to find at least one stale route when exclude_stale=false")
		}
	})

	t.Run("flags_and_filters_illiquid_trades", func(t *testing.T) {
		// When exclude_illiquid=true (default), illiquid routes (<5 volume) must NOT be present
		reqExclude, _ := http.NewRequest(http.MethodGet, "/api/v1/arbitrage/routes?region=asia&exclude_illiquid=true&exclude_stale=false", nil)
		rrExclude := httptest.NewRecorder()
		router.ServeHTTP(rrExclude, reqExclude)

		var cleanRoutes []*engine.ArbitrageRoute
		_ = json.Unmarshal(rrExclude.Body.Bytes(), &cleanRoutes)
		for _, r := range cleanRoutes {
			if r.IsIlliquid || r.TradedVolume24h < 5 {
				t.Fatalf("found illiquid route %s (%s -> %s) when exclude_illiquid=true", r.ItemID, r.OriginCity, r.TargetCity)
			}
		}

		// When exclude_illiquid=false, illiquid routes must be returned and flagged as IsIlliquid=true
		// We seeded Bridgewatch -> Caerleon for T4_ROCK_ILLIQUID with volume 2
		reqInclude, _ := http.NewRequest(http.MethodGet, "/api/v1/arbitrage/routes?region=asia&exclude_illiquid=false&exclude_stale=false", nil)
		rrInclude := httptest.NewRecorder()
		router.ServeHTTP(rrInclude, reqInclude)

		var allRoutes []*engine.ArbitrageRoute
		_ = json.Unmarshal(rrInclude.Body.Bytes(), &allRoutes)
		foundIlliquid := false
		for _, r := range allRoutes {
			if r.IsIlliquid {
				foundIlliquid = true
				if r.TradedVolume24h >= 5 {
					t.Errorf("illiquid route %s has 24h volume %d >= 5", r.ItemID, r.TradedVolume24h)
				}
			}
		}
		if !foundIlliquid {
			t.Errorf("expected to find at least one illiquid route when exclude_illiquid=false")
		}
	})
}

// Test Case 3: Price Ingestion & Refresh Workflow (POST /api/v1/prices/refresh)
func TestE2E_PriceIngestionAndRefreshWorkflow(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	// Mock external AODP REST API
	mockPrices := []aodp.PriceResponseItem{
		{
			ItemID:           "T4_WOOD",
			City:             "Fort Sterling",
			Quality:          1,
			SellPriceMin:     175,
			SellPriceMinDate: nowStr,
			BuyPriceMax:      160,
			BuyPriceMaxDate:  nowStr,
		},
		{
			ItemID:           "T4_PLANKS",
			City:             "Fort Sterling",
			Quality:          1,
			SellPriceMin:     420,
			SellPriceMinDate: nowStr,
			BuyPriceMax:      390,
			BuyPriceMaxDate:  nowStr,
		},
	}

	mockVolumes := []aodp.HistoryResponseItem{
		{
			ItemID:   "T4_WOOD",
			Location: "Fort Sterling",
			Quality:  1,
			Data: []aodp.HistoryDataPoint{
				{ItemCount: 15400, SilverAmount: 2695000, Timestamp: nowStr},
			},
		},
		{
			ItemID:   "T4_PLANKS",
			Location: "Fort Sterling",
			Quality:  1,
			Data: []aodp.HistoryDataPoint{
				{ItemCount: 8200, SilverAmount: 3444000, Timestamp: nowStr},
			},
		},
	}

	mockAODPServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer mockAODPServer.Close()

	// Initialize AODP client configured with mock server URL for region "asia"
	aodpClient := aodp.NewClient(aodp.WithBaseURL("asia", mockAODPServer.URL))
	router := setupRouter(store, aodpClient)

	// Trigger on-demand price refresh via API
	req, err := http.NewRequest(http.MethodPost, "/api/v1/prices/refresh?region=asia", nil)
	if err != nil {
		t.Fatalf("create refresh request: %v", err)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Status          string `json:"status"`
		Region          string `json:"region"`
		PricesIngested  int    `json:"prices_ingested"`
		VolumesIngested int    `json:"volumes_ingested"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal refresh response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
	if resp.Region != "asia" {
		t.Errorf("expected region 'asia', got %q", resp.Region)
	}
	if resp.PricesIngested != 2 {
		t.Errorf("expected 2 prices ingested, got %d", resp.PricesIngested)
	}
	if resp.VolumesIngested != 2 {
		t.Errorf("expected 2 volumes ingested, got %d", resp.VolumesIngested)
	}

	// Verify store upsert for price snapshot
	savedPrice, err := store.GetLatestPrice(ctx, "asia", "T4_WOOD", "Fort Sterling", 1)
	if err != nil {
		t.Fatalf("failed to query saved price: %v", err)
	}
	if savedPrice == nil {
		t.Fatalf("expected saved price for T4_WOOD in Fort Sterling to exist")
	}
	if savedPrice.SellPriceMin != 175 {
		t.Errorf("expected saved sell price 175, got %d", savedPrice.SellPriceMin)
	}
	if savedPrice.BuyPriceMax != 160 {
		t.Errorf("expected saved buy price 160, got %d", savedPrice.BuyPriceMax)
	}

	// Verify store upsert for 24h market volume
	savedVol, err := store.GetMarketVolume(ctx, "asia", "T4_WOOD", "Fort Sterling", 1)
	if err != nil {
		t.Fatalf("failed to query saved volume: %v", err)
	}
	if savedVol == nil {
		t.Fatalf("expected saved volume for T4_WOOD in Fort Sterling to exist")
	}
	if savedVol.ItemCount != 15400 {
		t.Errorf("expected saved volume count 15400, got %d", savedVol.ItemCount)
	}
	if savedVol.SilverAmount != 2695000 {
		t.Errorf("expected saved volume silver amount 2695000, got %d", savedVol.SilverAmount)
	}
}
