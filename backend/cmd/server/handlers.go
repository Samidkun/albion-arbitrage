package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"albionflow/backend/pkg/aodp"
	"albionflow/backend/pkg/engine"
	"albionflow/backend/pkg/model"
	"albionflow/backend/pkg/store"
)

// ServerState encapsulates dependencies for HTTP API handlers.
type ServerState struct {
	Store      *store.Store
	AODPClient *aodp.Client
}

// RefiningMatrixItem represents a computed entry in the refining matrix response.
type RefiningMatrixItem struct {
	ItemID             string              `json:"item_id"`
	Commodity          model.CommodityType `json:"commodity"`
	Tier               int                 `json:"tier"`
	Enchantment        int                 `json:"enchantment"`
	City               string              `json:"city"`
	IsBonusCity        bool                `json:"is_bonus_city"`
	ItemValue          int                 `json:"item_value"`
	RawItemID          string              `json:"raw_item_id"`
	RawAmount          int                 `json:"raw_amount"`
	RawPrice           float64             `json:"raw_price"`
	PrevRefinedItemID  string              `json:"prev_refined_item_id"`
	PrevRefinedAmount  int                 `json:"prev_refined_amount"`
	PrevRefinedPrice   float64             `json:"prev_refined_price"`
	SellPrice          float64             `json:"sell_price"`
	StationFee         float64             `json:"station_fee"`
	EffectiveRawCost   float64             `json:"effective_raw_cost"`
	EffectivePrevCost  float64             `json:"effective_prev_cost"`
	TotalCraftCost     float64             `json:"total_craft_cost"`
	GrossRevenue       float64             `json:"gross_revenue"`
	SalesTax           float64             `json:"sales_tax"`
	SetupFee           float64             `json:"setup_fee"`
	NetRevenue         float64             `json:"net_revenue"`
	ProfitWithoutFocus float64             `json:"profit_without_focus"`
	ProfitWithFocus    float64             `json:"profit_with_focus"`
	FocusProfitDelta   float64             `json:"focus_profit_delta"`
	BaseFocusCost      int                 `json:"base_focus_cost"`
	ActualFocusCost    int                 `json:"actual_focus_cost"`
	SilverPerFocus     float64             `json:"silver_per_focus"`
	DailyFocusSilver   float64             `json:"daily_focus_silver"`
	ProfitMarginPct    float64             `json:"profit_margin_pct"`
	RRR                float64             `json:"rrr"`
	IsChained          bool                `json:"is_chained"`
	TradedVolume24h    int64               `json:"traded_volume_24h"`
	IsIlliquid         bool                `json:"is_illiquid"`
	PriceAgeSeconds    int64               `json:"price_age_seconds"`
	Freshness          aodp.FreshnessTier  `json:"freshness"`
	IsStale            bool                `json:"is_stale"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func isValidRegion(r string) bool {
	switch strings.ToLower(strings.TrimSpace(r)) {
	case "asia", "americas", "europe":
		return true
	default:
		return false
	}
}

func isValidChannel(c string) bool {
	switch strings.ToLower(strings.TrimSpace(c)) {
	case string(engine.TaxChannelDirect),
		string(engine.TaxChannelInstantBuy),
		string(engine.TaxChannelSellOrder):
		return true
	default:
		return false
	}
}

func parseBoolParam(val string, defaultVal bool) bool {
	v := strings.ToLower(strings.TrimSpace(val))
	switch v {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return defaultVal
	}
}

var (
	standardCommodityItemIDs     []string
	standardCommodityItemIDsOnce sync.Once
)

func getStandardCommodityItemIDs() []string {
	standardCommodityItemIDsOnce.Do(func() {
		recipes := model.GetAllRecipes()
		seen := make(map[string]bool)
		for _, r := range recipes {
			for _, id := range []string{r.ResultItemID, r.RawItemID, r.PrevRefinedItemID} {
				if id != "" && !seen[id] {
					seen[id] = true
					standardCommodityItemIDs = append(standardCommodityItemIDs, id)
				}
			}
		}
		sort.Strings(standardCommodityItemIDs)
	})
	return standardCommodityItemIDs
}

// HandleRefiningMatrix returns calculated refining profitability and Silver per Focus for all or filtered recipes.
func (s *ServerState) HandleRefiningMatrix(w http.ResponseWriter, r *http.Request) {
	region := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("region")))
	if region == "" {
		region = "asia"
	}
	if !isValidRegion(region) {
		writeError(w, http.StatusBadRequest, "unsupported region: must be asia, americas, or europe")
		return
	}

	commodityParam := strings.TrimSpace(r.URL.Query().Get("commodity"))
	var commodityFilter model.CommodityType
	if commodityParam != "" {
		switch strings.ToLower(commodityParam) {
		case "wood":
			commodityFilter = model.CommodityWood
		case "ore":
			commodityFilter = model.CommodityOre
		case "hide":
			commodityFilter = model.CommodityHide
		case "fiber":
			commodityFilter = model.CommodityFiber
		case "stone":
			commodityFilter = model.CommodityStone
		default:
			writeError(w, http.StatusBadRequest, "invalid commodity: must be Wood, Ore, Hide, Fiber, or Stone")
			return
		}
	}

	tierParam := strings.TrimSpace(r.URL.Query().Get("tier"))
	var tierFilter int
	if tierParam != "" {
		t, err := strconv.Atoi(tierParam)
		if err != nil || t < 4 || t > 8 {
			writeError(w, http.StatusBadRequest, "invalid tier: must be between 4 and 8")
			return
		}
		tierFilter = t
	}

	tax := 600.0
	if taxParam := strings.TrimSpace(r.URL.Query().Get("tax")); taxParam != "" {
		t, err := strconv.ParseFloat(taxParam, 64)
		if err != nil || t < 0 {
			writeError(w, http.StatusBadRequest, "invalid tax: must be a non-negative number")
			return
		}
		tax = t
	}

	channel := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("channel")))
	if channel == "" {
		channel = string(engine.TaxChannelSellOrder)
	}
	if !isValidChannel(channel) {
		writeError(w, http.StatusBadRequest, "invalid channel: must be direct, instant_buy, or sell_order")
		return
	}

	premium := parseBoolParam(r.URL.Query().Get("premium"), true)
	chained := parseBoolParam(r.URL.Query().Get("chained"), false)

	mastery := 0
	if masteryParam := strings.TrimSpace(r.URL.Query().Get("mastery")); masteryParam != "" {
		m, err := strconv.Atoi(masteryParam)
		if err != nil || m < 0 || m > 100 {
			writeError(w, http.StatusBadRequest, "invalid mastery: must be between 0 and 100")
			return
		}
		mastery = m
	}

	cityParam := strings.TrimSpace(r.URL.Query().Get("city"))

	itemIDs := getStandardCommodityItemIDs()
	prices, err := s.Store.GetPricesForItems(r.Context(), region, itemIDs, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query prices: "+err.Error())
		return
	}

	volumes, err := s.Store.GetVolumesForItems(r.Context(), region, itemIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query volumes: "+err.Error())
		return
	}

	priceMap := make(map[string]*store.PriceSnapshot, len(prices))
	for _, p := range prices {
		if p == nil {
			continue
		}
		k := p.ItemID + ":" + p.City
		if existing, ok := priceMap[k]; !ok || (p.Quality == 1 && existing.Quality != 1) {
			priceMap[k] = p
		}
	}

	volumeMap := make(map[string]*store.MarketVolume, len(volumes))
	for _, v := range volumes {
		if v == nil {
			continue
		}
		k := v.ItemID + ":" + v.City
		if existing, ok := volumeMap[k]; !ok || (v.Quality == 1 && existing.Quality != 1) {
			volumeMap[k] = v
		}
	}

	commodities := []model.CommodityType{
		model.CommodityWood,
		model.CommodityOre,
		model.CommodityHide,
		model.CommodityFiber,
		model.CommodityStone,
	}

	type craftKey struct {
		commodity   model.CommodityType
		tier        int
		enchantment int
	}
	craftCostMap := make(map[craftKey]float64)

	salesTaxRate, setupFeeRate := engine.GetTaxRates(engine.TaxChannel(channel), premium)
	now := time.Now().UTC()
	results := make([]RefiningMatrixItem, 0)

	for _, comm := range commodities {
		for enchant := 0; enchant <= 4; enchant++ {
			if comm == model.CommodityStone && enchant > 0 {
				continue
			}
			for tier := 4; tier <= 8; tier++ {
				recipe, err := model.GetRefiningRecipe(comm, tier, enchant)
				if err != nil {
					continue
				}

				city := string(recipe.BonusCity)
				if cityParam != "" {
					city = cityParam
				}
				isBonusCity := strings.EqualFold(city, string(recipe.BonusCity))

				rawSnap := priceMap[recipe.RawItemID+":"+city]
				resultSnap := priceMap[recipe.ResultItemID+":"+city]
				prevSnap := priceMap[recipe.PrevRefinedItemID+":"+city]

				if rawSnap == nil || resultSnap == nil {
					continue
				}

				var sellPrice float64
				if channel == string(engine.TaxChannelInstantBuy) {
					sellPrice = float64(resultSnap.BuyPriceMax)
				} else {
					sellPrice = float64(resultSnap.SellPriceMin)
				}

				rawPrice := float64(rawSnap.SellPriceMin)

				var prevRefinedPrice float64
				if prevSnap != nil {
					prevRefinedPrice = float64(prevSnap.SellPriceMin)
				}

				if sellPrice <= 0 || rawPrice <= 0 {
					continue
				}

				prevTierCraftCost := craftCostMap[craftKey{commodity: comm, tier: tier - 1, enchantment: enchant}]
				isChainedRefine := chained && prevTierCraftCost > 0

				if !isChainedRefine && prevRefinedPrice <= 0 {
					continue
				}

				baseInput := engine.RefiningCostInput{
					Recipe:                    recipe,
					RawPrice:                  rawPrice,
					PrevRefinedPrice:          prevRefinedPrice,
					SellPrice:                 sellPrice,
					StationFeePer100Nutrition: tax,
					IsBonusCity:               isBonusCity,
					UseFocus:                  false,
					SalesTaxRate:              salesTaxRate,
					SetupFeeRate:              setupFeeRate,
					IsChainedRefine:           isChainedRefine,
					PrevTierCraftCost:         prevTierCraftCost,
				}

				calcRes, err := engine.CalculateRefining(baseInput)
				if err != nil {
					continue
				}

				craftCostMap[craftKey{commodity: comm, tier: tier, enchantment: enchant}] = calcRes.TotalCraftCostPerUnit

				// Filter checks
				if commodityFilter != "" && comm != commodityFilter {
					continue
				}
				if tierFilter != 0 && tier != tierFilter {
					continue
				}

				optRes, err := engine.EvaluateFocusEfficiency(baseInput, mastery)
				if err != nil {
					continue
				}

				var tradedVolume24h int64
				isIlliquid := true
				if vol := volumeMap[recipe.ResultItemID+":"+city]; vol != nil {
					tradedVolume24h = vol.ItemCount
					isIlliquid = vol.IsIlliquid(5)
				}

				dateStr := resultSnap.SellPriceMinDate
				if channel == string(engine.TaxChannelInstantBuy) && resultSnap.BuyPriceMaxDate != "" {
					dateStr = resultSnap.BuyPriceMaxDate
				}
				age, freshness := aodp.CalculatePriceAge(dateStr, resultSnap.RecordedAt, now)
				isStale := age > 60*time.Minute

				item := RefiningMatrixItem{
					ItemID:             recipe.ResultItemID,
					Commodity:          recipe.Commodity,
					Tier:               recipe.Tier,
					Enchantment:        recipe.Enchantment,
					City:               city,
					IsBonusCity:        isBonusCity,
					ItemValue:          recipe.ItemValue,
					RawItemID:          recipe.RawItemID,
					RawAmount:          recipe.RawAmount,
					RawPrice:           rawPrice,
					PrevRefinedItemID:  recipe.PrevRefinedItemID,
					PrevRefinedAmount:  recipe.PrevRefinedAmount,
					PrevRefinedPrice:   prevRefinedPrice,
					SellPrice:          sellPrice,
					StationFee:         calcRes.StationFee,
					EffectiveRawCost:   calcRes.EffectiveRawCost,
					EffectivePrevCost:  calcRes.EffectivePrevCost,
					TotalCraftCost:     calcRes.TotalCraftCostPerUnit,
					GrossRevenue:       calcRes.GrossRevenue,
					SalesTax:           calcRes.SalesTax,
					SetupFee:           calcRes.SetupFee,
					NetRevenue:         calcRes.NetRevenue,
					ProfitWithoutFocus: calcRes.NetProfitPerUnit,
					ProfitWithFocus:    optRes.ProfitWithFocus,
					FocusProfitDelta:   optRes.FocusProfitDelta,
					BaseFocusCost:      optRes.BaseFocusCost,
					ActualFocusCost:    optRes.ActualFocusCost,
					SilverPerFocus:     optRes.SilverPerFocus,
					DailyFocusSilver:   optRes.DailyFocusSilver,
					ProfitMarginPct:    calcRes.ProfitMarginPercent,
					RRR:                calcRes.RRR,
					IsChained:          isChainedRefine,
					TradedVolume24h:    tradedVolume24h,
					IsIlliquid:         isIlliquid,
					PriceAgeSeconds:    int64(age.Seconds()),
					Freshness:          freshness,
					IsStale:            isStale,
				}
				results = append(results, item)
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].SilverPerFocus != results[j].SilverPerFocus {
			return results[i].SilverPerFocus > results[j].SilverPerFocus
		}
		if results[i].ProfitWithoutFocus != results[j].ProfitWithoutFocus {
			return results[i].ProfitWithoutFocus > results[j].ProfitWithoutFocus
		}
		return results[i].ItemID < results[j].ItemID
	})

	writeJSON(w, http.StatusOK, results)
}

// HandleArbitrageRoutes finds profitable cross-city hauling routes based on stored price and volume snapshots.
func (s *ServerState) HandleArbitrageRoutes(w http.ResponseWriter, r *http.Request) {
	region := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("region")))
	if region == "" {
		region = "asia"
	}
	if !isValidRegion(region) {
		writeError(w, http.StatusBadRequest, "unsupported region: must be asia, americas, or europe")
		return
	}

	var minProfit float64
	if minProfitParam := strings.TrimSpace(r.URL.Query().Get("min_profit")); minProfitParam != "" {
		mp, err := strconv.ParseFloat(minProfitParam, 64)
		if err != nil || mp < 0 {
			writeError(w, http.StatusBadRequest, "invalid min_profit: must be a non-negative number")
			return
		}
		minProfit = mp
	}

	var minMargin float64
	if minMarginParam := strings.TrimSpace(r.URL.Query().Get("min_margin")); minMarginParam != "" {
		mm, err := strconv.ParseFloat(minMarginParam, 64)
		if err != nil || mm < 0 {
			writeError(w, http.StatusBadRequest, "invalid min_margin: must be a non-negative number")
			return
		}
		minMargin = mm
	}

	excludeStale := parseBoolParam(r.URL.Query().Get("exclude_stale"), true)
	excludeIlliquid := parseBoolParam(r.URL.Query().Get("exclude_illiquid"), true)

	channel := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("channel")))
	if channel == "" {
		channel = string(engine.TaxChannelSellOrder)
	}
	if !isValidChannel(channel) {
		writeError(w, http.StatusBadRequest, "invalid channel: must be direct, instant_buy, or sell_order")
		return
	}

	premium := parseBoolParam(r.URL.Query().Get("premium"), true)

	itemIDs := getStandardCommodityItemIDs()
	prices, err := s.Store.GetPricesForItems(r.Context(), region, itemIDs, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query prices: "+err.Error())
		return
	}

	volumes, err := s.Store.GetVolumesForItems(r.Context(), region, itemIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query volumes: "+err.Error())
		return
	}

	opts := engine.ArbitrageScanOptions{
		MinNetProfit:        minProfit,
		MinNetMarginPercent: minMargin,
		MaxDataAge:          60 * time.Minute,
		ExcludeStale:        excludeStale,
		Min24hVolume:        5,
		ExcludeIlliquid:     excludeIlliquid,
		TaxChannel:          engine.TaxChannel(channel),
		IsPremium:           premium,
	}

	now := time.Now().UTC()
	routes := engine.ScanArbitrageRoutes(prices, volumes, now, opts)
	if routes == nil {
		routes = []*engine.ArbitrageRoute{}
	}

	writeJSON(w, http.StatusOK, routes)
}

// HandlePriceRefresh pulls fresh market and trade volume data from AODP for standard commodities.
func (s *ServerState) HandlePriceRefresh(w http.ResponseWriter, r *http.Request) {
	region := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("region")))
	if region == "" && r.Body != nil {
		var bodyReq struct {
			Region string `json:"region"`
		}
		_ = json.NewDecoder(r.Body).Decode(&bodyReq)
		if bodyReq.Region != "" {
			region = strings.ToLower(strings.TrimSpace(bodyReq.Region))
		}
	}
	if region == "" {
		region = "asia"
	}
	if !isValidRegion(region) {
		writeError(w, http.StatusBadRequest, "unsupported region: must be asia, americas, or europe")
		return
	}

	if s.AODPClient == nil {
		writeError(w, http.StatusInternalServerError, "aodp client not initialized")
		return
	}

	itemIDs := getStandardCommodityItemIDs()
	pricesCount, volumesCount, err := s.AODPClient.IngestToStore(r.Context(), s.Store, region, itemIDs, nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to ingest market data: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "ok",
		"region":           region,
		"prices_ingested":  pricesCount,
		"volumes_ingested": volumesCount,
	})
}
