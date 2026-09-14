package engine_test

import (
	"math"
	"testing"
	"time"

	"albionflow/backend/pkg/engine"
	"albionflow/backend/pkg/store"
)

const floatTolerance = 0.0001

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < floatTolerance
}

func TestGetTaxRates(t *testing.T) {
	tests := []struct {
		name         string
		channel      engine.TaxChannel
		isPremium    bool
		wantSalesTax float64
		wantSetupFee float64
	}{
		{
			name:         "direct non-premium",
			channel:      engine.TaxChannelDirect,
			isPremium:    false,
			wantSalesTax: 0.0,
			wantSetupFee: 0.0,
		},
		{
			name:         "direct premium",
			channel:      engine.TaxChannelDirect,
			isPremium:    true,
			wantSalesTax: 0.0,
			wantSetupFee: 0.0,
		},
		{
			name:         "instant_buy non-premium",
			channel:      engine.TaxChannelInstantBuy,
			isPremium:    false,
			wantSalesTax: 0.08,
			wantSetupFee: 0.0,
		},
		{
			name:         "instant_buy premium",
			channel:      engine.TaxChannelInstantBuy,
			isPremium:    true,
			wantSalesTax: 0.04,
			wantSetupFee: 0.0,
		},
		{
			name:         "sell_order non-premium",
			channel:      engine.TaxChannelSellOrder,
			isPremium:    false,
			wantSalesTax: 0.08,
			wantSetupFee: 0.025,
		},
		{
			name:         "sell_order premium",
			channel:      engine.TaxChannelSellOrder,
			isPremium:    true,
			wantSalesTax: 0.04,
			wantSetupFee: 0.025,
		},
		{
			name:         "unknown channel defaults to sell_order non-premium",
			channel:      engine.TaxChannel("custom"),
			isPremium:    false,
			wantSalesTax: 0.08,
			wantSetupFee: 0.025,
		},
		{
			name:         "empty channel defaults to sell_order premium",
			channel:      "",
			isPremium:    true,
			wantSalesTax: 0.04,
			wantSetupFee: 0.025,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			salesTax, setupFee := engine.GetTaxRates(tt.channel, tt.isPremium)
			if !almostEqual(salesTax, tt.wantSalesTax) {
				t.Errorf("GetTaxRates(%s, %v) salesTax = %v, want %v", tt.channel, tt.isPremium, salesTax, tt.wantSalesTax)
			}
			if !almostEqual(setupFee, tt.wantSetupFee) {
				t.Errorf("GetTaxRates(%s, %v) setupFee = %v, want %v", tt.channel, tt.isPremium, setupFee, tt.wantSetupFee)
			}
		})
	}
}

func TestScanArbitrageRoutes_BasicHaulingRoute(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	twentyMinAgo := now.Add(-20 * time.Minute).Format(time.RFC3339)

	prices := []*store.PriceSnapshot{
		{
			ItemID:           "T4_BAG",
			City:             "Fort Sterling",
			SellPriceMin:     300,
			SellPriceMinDate: twentyMinAgo,
			BuyPriceMax:      250,
			BuyPriceMaxDate:  twentyMinAgo,
			RecordedAt:       now.Add(-20 * time.Minute),
		},
		{
			ItemID:           "T4_BAG",
			City:             "Caerleon",
			SellPriceMin:     500,
			SellPriceMinDate: twentyMinAgo,
			BuyPriceMax:      450,
			BuyPriceMaxDate:  twentyMinAgo,
			RecordedAt:       now.Add(-20 * time.Minute),
		},
	}

	volumes := []*store.MarketVolume{
		{
			ItemID:    "T4_BAG",
			City:      "Caerleon",
			ItemCount: 150,
		},
		{
			ItemID:    "T4_BAG",
			City:      "Fort Sterling",
			ItemCount: 100,
		},
	}

	opts := engine.ArbitrageScanOptions{
		TaxChannel: engine.TaxChannelSellOrder,
		IsPremium:  true,
	}

	routes := engine.ScanArbitrageRoutes(prices, volumes, now, opts)
	if len(routes) != 1 {
		t.Fatalf("expected 1 profitable route (Fort Sterling -> Caerleon), got %d", len(routes))
	}

	r := routes[0]
	if r.ItemID != "T4_BAG" {
		t.Errorf("expected ItemID T4_BAG, got %s", r.ItemID)
	}
	if r.OriginCity != "Fort Sterling" {
		t.Errorf("expected OriginCity Fort Sterling, got %s", r.OriginCity)
	}
	if r.TargetCity != "Caerleon" {
		t.Errorf("expected TargetCity Caerleon, got %s", r.TargetCity)
	}
	if r.BuyPrice != 300 {
		t.Errorf("expected BuyPrice 300, got %d", r.BuyPrice)
	}
	if r.SellPrice != 500 {
		t.Errorf("expected SellPrice 500, got %d", r.SellPrice)
	}
	if !almostEqual(r.SalesTaxRate, 0.04) {
		t.Errorf("expected SalesTaxRate 0.04, got %f", r.SalesTaxRate)
	}
	if !almostEqual(r.SetupFeeRate, 0.025) {
		t.Errorf("expected SetupFeeRate 0.025, got %f", r.SetupFeeRate)
	}
	if !almostEqual(r.GrossRevenue, 500.0) {
		t.Errorf("expected GrossRevenue 500.0, got %f", r.GrossRevenue)
	}

	// NetRevenue = 500 - (500 * 0.04) - (500 * 0.025) = 500 - 20 - 12.5 = 467.5
	wantNetRevenue := 467.5
	if !almostEqual(r.NetRevenue, wantNetRevenue) {
		t.Errorf("expected NetRevenue %f, got %f", wantNetRevenue, r.NetRevenue)
	}

	// NetProfit = 467.5 - 300 = 167.5
	wantProfit := 167.5
	if !almostEqual(r.NetProfitPerUnit, wantProfit) {
		t.Errorf("expected NetProfitPerUnit %f, got %f", wantProfit, r.NetProfitPerUnit)
	}

	// Margin = (167.5 / 300) * 100 = 55.8333%
	wantMargin := (wantProfit / 300.0) * 100.0
	if !almostEqual(r.NetMarginPercent, wantMargin) {
		t.Errorf("expected NetMarginPercent %f, got %f", wantMargin, r.NetMarginPercent)
	}

	if r.TradedVolume24h != 150 {
		t.Errorf("expected TradedVolume24h 150, got %d", r.TradedVolume24h)
	}
	if r.IsIlliquid {
		t.Errorf("expected IsIlliquid false for volume 150, got true")
	}
	if r.IsStaleRoute {
		t.Errorf("expected IsStaleRoute false for 20m old quote, got true")
	}
}

func TestScanArbitrageRoutes_TaxChannels(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	twentyMinAgo := now.Add(-20 * time.Minute).Format(time.RFC3339)

	prices := []*store.PriceSnapshot{
		{
			ItemID:           "T5_ORE",
			City:             "Thetford",
			SellPriceMin:     200,
			SellPriceMinDate: twentyMinAgo,
			BuyPriceMax:      180,
			BuyPriceMaxDate:  twentyMinAgo,
			RecordedAt:       now.Add(-20 * time.Minute),
		},
		{
			ItemID:           "T5_ORE",
			City:             "Bridgewatch",
			SellPriceMin:     400,
			SellPriceMinDate: twentyMinAgo,
			BuyPriceMax:      350,
			BuyPriceMaxDate:  twentyMinAgo,
			RecordedAt:       now.Add(-20 * time.Minute),
		},
	}

	volumes := []*store.MarketVolume{
		{ItemID: "T5_ORE", City: "Bridgewatch", ItemCount: 1000},
	}

	// Direct channel (0% sales tax, 0% setup fee)
	optsDirect := engine.ArbitrageScanOptions{
		TaxChannel: engine.TaxChannelDirect,
		IsPremium:  false,
	}
	routesDirect := engine.ScanArbitrageRoutes(prices, volumes, now, optsDirect)
	if len(routesDirect) != 1 {
		t.Fatalf("expected 1 route for direct channel, got %d", len(routesDirect))
	}
	// SellPrice is 400 (SellPriceMin), Gross = 400, Tax = 0, NetRevenue = 400, Profit = 400 - 200 = 200
	if !almostEqual(routesDirect[0].NetRevenue, 400.0) {
		t.Errorf("expected NetRevenue 400 for direct, got %f", routesDirect[0].NetRevenue)
	}
	if !almostEqual(routesDirect[0].NetProfitPerUnit, 200.0) {
		t.Errorf("expected Profit 200 for direct, got %f", routesDirect[0].NetProfitPerUnit)
	}

	// InstantBuy channel (sells to buy order: BuyPriceMax = 350, 0% setup fee, 4% premium tax)
	optsInstant := engine.ArbitrageScanOptions{
		TaxChannel: engine.TaxChannelInstantBuy,
		IsPremium:  true,
	}
	routesInstant := engine.ScanArbitrageRoutes(prices, volumes, now, optsInstant)
	if len(routesInstant) != 1 {
		t.Fatalf("expected 1 route for instant_buy channel, got %d", len(routesInstant))
	}
	// SellPrice is 350 (BuyPriceMax), Tax = 350 * 0.04 = 14, NetRevenue = 350 - 14 = 336, Profit = 336 - 200 = 136
	if routesInstant[0].SellPrice != 350 {
		t.Errorf("expected SellPrice 350 for instant_buy, got %d", routesInstant[0].SellPrice)
	}
	if !almostEqual(routesInstant[0].NetRevenue, 336.0) {
		t.Errorf("expected NetRevenue 336 for instant_buy, got %f", routesInstant[0].NetRevenue)
	}
	if !almostEqual(routesInstant[0].NetProfitPerUnit, 136.0) {
		t.Errorf("expected Profit 136 for instant_buy, got %f", routesInstant[0].NetProfitPerUnit)
	}
}

func TestScanArbitrageRoutes_StaleDataGuard(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	twentyMinAgo := now.Add(-20 * time.Minute).Format(time.RFC3339)
	seventyMinAgo := now.Add(-70 * time.Minute).Format(time.RFC3339)

	prices := []*store.PriceSnapshot{
		{
			ItemID:           "T4_BAG",
			City:             "Fort Sterling",
			SellPriceMin:     300,
			SellPriceMinDate: twentyMinAgo,
			RecordedAt:       now.Add(-20 * time.Minute),
		},
		{
			ItemID:           "T4_BAG",
			City:             "Caerleon",
			SellPriceMin:     500,
			SellPriceMinDate: seventyMinAgo, // > 60m old
			RecordedAt:       now.Add(-70 * time.Minute),
		},
	}

	volumes := []*store.MarketVolume{
		{ItemID: "T4_BAG", City: "Caerleon", ItemCount: 100},
	}

	// 1. Without ExcludeStale: route should be present, but flagged IsStaleRoute: true
	optsInclude := engine.ArbitrageScanOptions{
		TaxChannel:   engine.TaxChannelSellOrder,
		ExcludeStale: false,
		MaxDataAge:   60 * time.Minute,
	}
	routes := engine.ScanArbitrageRoutes(prices, volumes, now, optsInclude)
	if len(routes) != 1 {
		t.Fatalf("expected 1 route when ExcludeStale is false, got %d", len(routes))
	}
	if !routes[0].IsStaleRoute {
		t.Errorf("expected route with 70m old target quote to have IsStaleRoute: true")
	}
	if !routes[0].IsSellStale {
		t.Errorf("expected IsSellStale true")
	}
	if routes[0].IsBuyStale {
		t.Errorf("expected IsBuyStale false for 20m old quote")
	}

	// 2. With ExcludeStale: route should be omitted
	optsExclude := engine.ArbitrageScanOptions{
		TaxChannel:   engine.TaxChannelSellOrder,
		ExcludeStale: true,
		MaxDataAge:   60 * time.Minute,
	}
	routesExcluded := engine.ScanArbitrageRoutes(prices, volumes, now, optsExclude)
	if len(routesExcluded) != 0 {
		t.Errorf("expected 0 routes when ExcludeStale is true, got %d", len(routesExcluded))
	}
}

func TestScanArbitrageRoutes_IlliquidGuard(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	twentyMinAgo := now.Add(-20 * time.Minute).Format(time.RFC3339)

	prices := []*store.PriceSnapshot{
		{
			ItemID:           "T4_BAG",
			City:             "Fort Sterling",
			SellPriceMin:     300,
			SellPriceMinDate: twentyMinAgo,
			RecordedAt:       now.Add(-20 * time.Minute),
		},
		{
			ItemID:           "T4_BAG",
			City:             "Caerleon",
			SellPriceMin:     500,
			SellPriceMinDate: twentyMinAgo,
			RecordedAt:       now.Add(-20 * time.Minute),
		},
	}

	// Target city volume is only 3 (< default 5)
	volumes := []*store.MarketVolume{
		{ItemID: "T4_BAG", City: "Caerleon", ItemCount: 3},
	}

	// 1. Without ExcludeIlliquid: route should be present, but IsIlliquid: true
	optsInclude := engine.ArbitrageScanOptions{
		ExcludeIlliquid: false,
		Min24hVolume:    5,
	}
	routes := engine.ScanArbitrageRoutes(prices, volumes, now, optsInclude)
	if len(routes) != 1 {
		t.Fatalf("expected 1 route when ExcludeIlliquid is false, got %d", len(routes))
	}
	if !routes[0].IsIlliquid {
		t.Errorf("expected IsIlliquid true for 3 volume (<5 threshold)")
	}
	if routes[0].TradedVolume24h != 3 {
		t.Errorf("expected TradedVolume24h 3, got %d", routes[0].TradedVolume24h)
	}

	// 2. With ExcludeIlliquid: route should be omitted
	optsExclude := engine.ArbitrageScanOptions{
		ExcludeIlliquid: true,
		Min24hVolume:    5,
	}
	routesExcluded := engine.ScanArbitrageRoutes(prices, volumes, now, optsExclude)
	if len(routesExcluded) != 0 {
		t.Errorf("expected 0 routes when ExcludeIlliquid is true, got %d", len(routesExcluded))
	}
}

func TestScanArbitrageRoutes_SortingOrder(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	twentyMinAgo := now.Add(-20 * time.Minute).Format(time.RFC3339)

	prices := []*store.PriceSnapshot{
		// Item A: Buy 100, Sell 200 -> Profit approx 87
		{ItemID: "ITEM_A", City: "Fort Sterling", SellPriceMin: 100, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		{ItemID: "ITEM_A", City: "Caerleon", SellPriceMin: 200, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		// Item B: Buy 100, Sell 500 -> Profit approx 367 (Highest profit)
		{ItemID: "ITEM_B", City: "Fort Sterling", SellPriceMin: 100, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		{ItemID: "ITEM_B", City: "Caerleon", SellPriceMin: 500, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		// Item C: Buy 100, Sell 300 -> Profit approx 180 (Middle profit)
		{ItemID: "ITEM_C", City: "Fort Sterling", SellPriceMin: 100, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		{ItemID: "ITEM_C", City: "Caerleon", SellPriceMin: 300, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
	}

	volumes := []*store.MarketVolume{
		{ItemID: "ITEM_A", City: "Caerleon", ItemCount: 100},
		{ItemID: "ITEM_B", City: "Caerleon", ItemCount: 100},
		{ItemID: "ITEM_C", City: "Caerleon", ItemCount: 100},
	}

	opts := engine.ArbitrageScanOptions{
		IsPremium: true,
	}

	routes := engine.ScanArbitrageRoutes(prices, volumes, now, opts)
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}

	if routes[0].ItemID != "ITEM_B" {
		t.Errorf("expected first route to be ITEM_B, got %s", routes[0].ItemID)
	}
	if routes[1].ItemID != "ITEM_C" {
		t.Errorf("expected second route to be ITEM_C, got %s", routes[1].ItemID)
	}
	if routes[2].ItemID != "ITEM_A" {
		t.Errorf("expected third route to be ITEM_A, got %s", routes[2].ItemID)
	}

	// Verify descending order
	if routes[0].NetProfitPerUnit < routes[1].NetProfitPerUnit || routes[1].NetProfitPerUnit < routes[2].NetProfitPerUnit {
		t.Errorf("routes not sorted in descending order: %f, %f, %f",
			routes[0].NetProfitPerUnit, routes[1].NetProfitPerUnit, routes[2].NetProfitPerUnit)
	}
}

func TestScanArbitrageRoutes_ProfitAndMarginFilters(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	twentyMinAgo := now.Add(-20 * time.Minute).Format(time.RFC3339)

	prices := []*store.PriceSnapshot{
		// Item 1: Buy 1000, Sell 1200 -> Gross 1200, NetRev 1122 (pre tax 4%+2.5%), Profit 122, Margin 12.2%
		{ItemID: "ITEM_LOW_MARGIN", City: "Martlock", SellPriceMin: 1000, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		{ItemID: "ITEM_LOW_MARGIN", City: "Caerleon", SellPriceMin: 1200, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		// Item 2: Buy 100, Sell 300 -> Gross 300, NetRev 280.5, Profit 180.5, Margin 180.5%
		{ItemID: "ITEM_HIGH_MARGIN", City: "Martlock", SellPriceMin: 100, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		{ItemID: "ITEM_HIGH_MARGIN", City: "Caerleon", SellPriceMin: 300, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
	}

	volumes := []*store.MarketVolume{
		{ItemID: "ITEM_LOW_MARGIN", City: "Caerleon", ItemCount: 50},
		{ItemID: "ITEM_HIGH_MARGIN", City: "Caerleon", ItemCount: 50},
	}

	// Test MinNetProfit: require >= 150 -> should omit ITEM_LOW_MARGIN (profit 122)
	optsProfit := engine.ArbitrageScanOptions{
		MinNetProfit: 150.0,
		IsPremium:    true,
	}
	routesProfit := engine.ScanArbitrageRoutes(prices, volumes, now, optsProfit)
	if len(routesProfit) != 1 || routesProfit[0].ItemID != "ITEM_HIGH_MARGIN" {
		t.Errorf("expected only ITEM_HIGH_MARGIN for MinNetProfit=150, got %d routes", len(routesProfit))
	}

	// Test MinNetMarginPercent: require >= 50% -> should omit ITEM_LOW_MARGIN (12.2%)
	optsMargin := engine.ArbitrageScanOptions{
		MinNetMarginPercent: 50.0,
		IsPremium:           true,
	}
	routesMargin := engine.ScanArbitrageRoutes(prices, volumes, now, optsMargin)
	if len(routesMargin) != 1 || routesMargin[0].ItemID != "ITEM_HIGH_MARGIN" {
		t.Errorf("expected only ITEM_HIGH_MARGIN for MinNetMarginPercent=50, got %d routes", len(routesMargin))
	}
}

func TestScanArbitrageRoutes_ZeroAndNegativePricesIgnored(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	twentyMinAgo := now.Add(-20 * time.Minute).Format(time.RFC3339)

	prices := []*store.PriceSnapshot{
		// Zero buy price at origin
		{ItemID: "ITEM_ZERO_BUY", City: "Thetford", SellPriceMin: 0, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		{ItemID: "ITEM_ZERO_BUY", City: "Caerleon", SellPriceMin: 500, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		// Zero sell price at target
		{ItemID: "ITEM_ZERO_SELL", City: "Thetford", SellPriceMin: 300, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		{ItemID: "ITEM_ZERO_SELL", City: "Caerleon", SellPriceMin: 0, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		// Negative prices
		{ItemID: "ITEM_NEG", City: "Thetford", SellPriceMin: -10, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
		{ItemID: "ITEM_NEG", City: "Caerleon", SellPriceMin: -5, SellPriceMinDate: twentyMinAgo, RecordedAt: now},
	}

	routes := engine.ScanArbitrageRoutes(prices, nil, now, engine.ArbitrageScanOptions{})
	if len(routes) != 0 {
		t.Errorf("expected 0 routes for zero/negative prices, got %d", len(routes))
	}
}
