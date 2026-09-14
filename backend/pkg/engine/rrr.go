package engine

const (
	RRRBase              = 0.152 // Base return rate (no city bonus, no focus)
	RRRCityBonus         = 0.367 // City specialization bonus without focus
	RRRFocusOnly         = 0.435 // Focus without city specialization bonus
	RRRCityBonusAndFocus = 0.539 // City specialization bonus + Focus

	// NutritionPerItemValue is the standard nutrition consumed per unit of Item Value.
	NutritionPerItemValue = 0.1125
)

// CalculateRRR returns the official Resource Return Rate based on city specialization and focus usage.
// ponytail: static rates without zone daily production bonuses (ceiling: 4 fixed states; add dynamic zone event modifiers if daily events are modeled).
func CalculateRRR(isBonusCity bool, useFocus bool) float64 {
	if isBonusCity && useFocus {
		return RRRCityBonusAndFocus
	}
	if useFocus {
		return RRRFocusOnly
	}
	if isBonusCity {
		return RRRCityBonus
	}
	return RRRBase
}

// CalculateEffectiveMaterialUsage returns the net quantity of materials consumed after RRR returns.
func CalculateEffectiveMaterialUsage(requiredQty int, rrr float64) float64 {
	return float64(requiredQty) * (1.0 - rrr)
}

// CalculateStationFee returns the crafting station silver fee based on item value and fee per 100 nutrition.
func CalculateStationFee(itemValue int, feePer100Nutrition float64) float64 {
	nutritionConsumed := float64(itemValue) * NutritionPerItemValue
	return (nutritionConsumed * feePer100Nutrition) / 100.0
}

// CalculateEffectiveMaterialCost calculates the net material silver cost and individual refund quantities.
func CalculateEffectiveMaterialCost(rawPrice, prevRefinedPrice float64, rawAmount, prevAmount int, rrr float64) (effectiveCost float64, rawRefund float64, prevRefund float64) {
	rawRefund = float64(rawAmount) * rrr
	prevRefund = float64(prevAmount) * rrr
	effectiveCost = (rawPrice*float64(rawAmount) + prevRefinedPrice*float64(prevAmount)) * (1.0 - rrr)
	return effectiveCost, rawRefund, prevRefund
}
