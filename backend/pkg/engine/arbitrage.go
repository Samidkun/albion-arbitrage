package engine

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"albionflow/backend/pkg/store"
)

// TaxChannel defines the marketplace tax mechanism applied on item sale.
type TaxChannel string

const (
	// TaxChannelDirect: peer-to-peer / Discord trade (0% sales tax, 0% setup fee).
	TaxChannelDirect TaxChannel = "direct"
	// TaxChannelInstantBuy: instant fill against buyer's order (0% setup fee, 4% premium or 8% non-premium sales tax).
	TaxChannelInstantBuy TaxChannel = "instant_buy"
	// TaxChannelSellOrder: create market sell order listing (2.5% setup fee, 4% premium or 8% non-premium sales tax).
	TaxChannelSellOrder TaxChannel = "sell_order"
)

// ArbitrageRoute represents a cross-city hauling opportunity between origin and target cities.
type ArbitrageRoute struct {
	ItemID           string        `json:"item_id"`
	OriginCity       string        `json:"origin_city"`
	TargetCity       string        `json:"target_city"`
	BuyPrice         int           `json:"buy_price"`
	BuyPriceAge      time.Duration `json:"buy_price_age"`
	IsBuyStale       bool          `json:"is_buy_stale"`
	SellPrice        int           `json:"sell_price"`
	SellPriceAge     time.Duration `json:"sell_price_age"`
	IsSellStale      bool          `json:"is_sell_stale"`
	IsStaleRoute     bool          `json:"is_stale_route"`
	TaxChannel       TaxChannel    `json:"tax_channel"`
	SalesTaxRate     float64       `json:"sales_tax_rate"`
	SetupFeeRate     float64       `json:"setup_fee_rate"`
	GrossRevenue     float64       `json:"gross_revenue"`
	NetRevenue       float64       `json:"net_revenue"`
	NetProfitPerUnit float64       `json:"net_profit_per_unit"`
	NetMarginPercent float64       `json:"net_margin_percent"`
	TradedVolume24h  int64         `json:"traded_volume_24h"`
	IsIlliquid       bool          `json:"is_illiquid"`
}

// ArbitrageScanOptions controls filters and parameters for hauling arbitrage scanner.
type ArbitrageScanOptions struct {
	MinNetProfit        float64       `json:"min_net_profit"`
	MinNetMarginPercent float64       `json:"min_net_margin_percent"`
	MaxDataAge          time.Duration `json:"max_data_age"`
	ExcludeStale        bool          `json:"exclude_stale"`
	Min24hVolume        int64         `json:"min_24h_volume"`
	ExcludeIlliquid     bool          `json:"exclude_illiquid"`
	TaxChannel          TaxChannel    `json:"tax_channel"`
	IsPremium           bool          `json:"is_premium"`
}

// GetTaxRates returns the sales tax rate and market setup fee rate for a given tax channel and premium status.
func GetTaxRates(channel TaxChannel, isPremium bool) (salesTax float64, setupFee float64) {
	switch channel {
	case TaxChannelDirect:
		return 0.0, 0.0
	case TaxChannelInstantBuy:
		if isPremium {
			return 0.04, 0.0
		}
		return 0.08, 0.0
	case TaxChannelSellOrder:
		fallthrough
	default:
		if isPremium {
			return 0.04, 0.025
		}
		return 0.08, 0.025
	}
}

var marketDateFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z07:00",
}

func parseSnapshotDate(dateStr string) (time.Time, error) {
	trimmed := strings.TrimSpace(dateStr)
	if trimmed == "" {
		return time.Time{}, errors.New("empty time string")
	}
	for _, format := range marketDateFormats {
		if t, err := time.Parse(format, trimmed); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %s", dateStr)
}

func getBuyPriceAge(p *store.PriceSnapshot, now time.Time, threshold time.Duration) (time.Duration, bool) {
	if p.BuyPriceMaxDate == "" || strings.HasPrefix(p.BuyPriceMaxDate, "0001-01-01") {
		return p.Age(now), p.IsStale(now, threshold)
	}

	parsed, err := parseSnapshotDate(p.BuyPriceMaxDate)
	if err != nil || parsed.Year() <= 1 {
		return p.Age(now), p.IsStale(now, threshold)
	}

	age := now.Sub(parsed)
	if age < 0 {
		age = 0
	}
	return age, age > threshold
}

// ScanArbitrageRoutes pairs origin and target cities for each item, calculating net hauling margins.
// ponytail: routes evaluate pairwise city connections without multi-hop waypoints (ceiling: direct 1-leg hauling; add Dijkstra graph router if overland convoy waypoints with red zone avoidance are requested).
func ScanArbitrageRoutes(prices []*store.PriceSnapshot, volumes []*store.MarketVolume, now time.Time, opts ArbitrageScanOptions) []*ArbitrageRoute {
	if len(prices) == 0 {
		return []*ArbitrageRoute{}
	}

	maxDataAge := opts.MaxDataAge
	if maxDataAge <= 0 {
		maxDataAge = 60 * time.Minute
	}

	min24hVolume := opts.Min24hVolume
	if min24hVolume <= 0 {
		min24hVolume = 5
	}

	channel := opts.TaxChannel
	if channel == "" {
		channel = TaxChannelSellOrder
	}

	volumeMap := make(map[string]int64, len(volumes))
	for _, v := range volumes {
		if v == nil {
			continue
		}
		key := v.ItemID + ":" + v.City
		volumeMap[key] += v.ItemCount
	}

	itemOrder := make([]string, 0)
	byItem := make(map[string][]*store.PriceSnapshot)
	for _, p := range prices {
		if p == nil {
			continue
		}
		if _, exists := byItem[p.ItemID]; !exists {
			itemOrder = append(itemOrder, p.ItemID)
		}
		byItem[p.ItemID] = append(byItem[p.ItemID], p)
	}

	salesTaxRate, setupFeeRate := GetTaxRates(channel, opts.IsPremium)
	routes := make([]*ArbitrageRoute, 0)

	for _, itemID := range itemOrder {
		snapshots := byItem[itemID]
		for _, origin := range snapshots {
			buyPrice := origin.SellPriceMin
			if buyPrice <= 0 {
				continue
			}

			buyAge := origin.Age(now)
			isBuyStale := origin.IsStale(now, maxDataAge)

			for _, target := range snapshots {
				if origin.City == target.City {
					continue
				}

				var sellPrice int
				var sellAge time.Duration
				var isSellStale bool

				if channel == TaxChannelInstantBuy {
					sellPrice = target.BuyPriceMax
					if sellPrice <= 0 {
						continue
					}
					sellAge, isSellStale = getBuyPriceAge(target, now, maxDataAge)
				} else {
					sellPrice = target.SellPriceMin
					if sellPrice <= 0 {
						continue
					}
					sellAge = target.Age(now)
					isSellStale = target.IsStale(now, maxDataAge)
				}

				isStaleRoute := isBuyStale || isSellStale || buyAge > maxDataAge || sellAge > maxDataAge
				if opts.ExcludeStale && isStaleRoute {
					continue
				}

				volKey := itemID + ":" + target.City
				targetVol, hasVol := volumeMap[volKey]
				isIlliquid := !hasVol || targetVol < min24hVolume
				if opts.ExcludeIlliquid && isIlliquid {
					continue
				}

				grossRevenue := float64(sellPrice)
				salesTax := grossRevenue * salesTaxRate
				setupFee := grossRevenue * setupFeeRate
				netRevenue := grossRevenue - salesTax - setupFee
				netProfit := netRevenue - float64(buyPrice)

				if opts.MinNetProfit > 0 {
					if netProfit < opts.MinNetProfit {
						continue
					}
				} else if netProfit <= 0 {
					continue
				}

				var netMargin float64
				if buyPrice > 0 {
					netMargin = (netProfit / float64(buyPrice)) * 100.0
				}

				if opts.MinNetMarginPercent > 0 && netMargin < opts.MinNetMarginPercent {
					continue
				}

				route := &ArbitrageRoute{
					ItemID:           itemID,
					OriginCity:       origin.City,
					TargetCity:       target.City,
					BuyPrice:         buyPrice,
					BuyPriceAge:      buyAge,
					IsBuyStale:       isBuyStale,
					SellPrice:        sellPrice,
					SellPriceAge:     sellAge,
					IsSellStale:      isSellStale,
					IsStaleRoute:     isStaleRoute,
					TaxChannel:       channel,
					SalesTaxRate:     salesTaxRate,
					SetupFeeRate:     setupFeeRate,
					GrossRevenue:     grossRevenue,
					NetRevenue:       netRevenue,
					NetProfitPerUnit: netProfit,
					NetMarginPercent: netMargin,
					TradedVolume24h:  targetVol,
					IsIlliquid:       isIlliquid,
				}
				routes = append(routes, route)
			}
		}
	}

	sort.Slice(routes, func(i, j int) bool {
		if routes[i].NetProfitPerUnit != routes[j].NetProfitPerUnit {
			return routes[i].NetProfitPerUnit > routes[j].NetProfitPerUnit
		}
		if routes[i].NetMarginPercent != routes[j].NetMarginPercent {
			return routes[i].NetMarginPercent > routes[j].NetMarginPercent
		}
		if routes[i].ItemID != routes[j].ItemID {
			return routes[i].ItemID < routes[j].ItemID
		}
		if routes[i].OriginCity != routes[j].OriginCity {
			return routes[i].OriginCity < routes[j].OriginCity
		}
		return routes[i].TargetCity < routes[j].TargetCity
	})

	return routes
}
