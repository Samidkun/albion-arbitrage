package engine

import (
	"math"

	"albionflow/backend/pkg/model"
)

// baseFocusCosts maps Tier -> [enchantment 0..4] -> base focus cost.
var baseFocusCosts = map[int][5]int{
	4: {34, 67, 135, 269, 538},
	5: {67, 135, 269, 538, 1076},
	6: {135, 269, 538, 1076, 2152},
	7: {269, 538, 1076, 2152, 4304},
	8: {538, 1076, 2152, 4304, 8608},
}

// GetBaseFocusCost returns the official base focus cost for refining a given tier and enchantment.
func GetBaseFocusCost(tier int, enchant int) (int, error) {
	if tier < 4 || tier > 8 {
		return 0, model.ErrInvalidTier
	}
	if enchant < 0 || enchant > 4 {
		return 0, model.ErrInvalidEnchantment
	}
	return baseFocusCosts[tier][enchant], nil
}

// CalculateActualFocus computes the focus cost adjusted for crafting mastery level (0-100).
// ponytail: single mastery curve without cross-specialization masteries (ceiling: 0-100 linear node curve; add item-family mastery aggregation if full destiny board sync is modeled).
func CalculateActualFocus(baseFocus int, masteryLevel int) int {
	if masteryLevel < 0 {
		masteryLevel = 0
	} else if masteryLevel > 100 {
		masteryLevel = 100
	}

	cost := int(math.Round(float64(baseFocus) * math.Pow(0.5, float64(masteryLevel)/100.0)))
	if cost < 1 {
		return 1
	}
	return cost
}

// CalculateSPF computes the Silver per Focus efficiency metric: (ProfitWithFocus - ProfitWithoutFocus) / FocusSpent.
func CalculateSPF(profitWithFocus float64, profitWithoutFocus float64, focusCost int) float64 {
	if focusCost <= 0 {
		return 0.0
	}
	return (profitWithFocus - profitWithoutFocus) / float64(focusCost)
}

// FocusOptimizationResult holds the comparative analysis of refining with and without focus.
type FocusOptimizationResult struct {
	Recipe             *model.RefiningRecipe `json:"recipe"`
	BaseFocusCost      int                   `json:"base_focus_cost"`
	ActualFocusCost    int                   `json:"actual_focus_cost"`
	ProfitWithoutFocus float64               `json:"profit_without_focus"`
	ProfitWithFocus    float64               `json:"profit_with_focus"`
	FocusProfitDelta   float64               `json:"focus_profit_delta"`
	SilverPerFocus     float64               `json:"silver_per_focus"`
	DailyFocusSilver   float64               `json:"daily_focus_silver"`
}

// EvaluateFocusEfficiency evaluates refining economics with and without focus to measure SPF return.
func EvaluateFocusEfficiency(baseInput RefiningCostInput, masteryLevel int) (*FocusOptimizationResult, error) {
	if baseInput.Recipe == nil {
		return nil, ErrNilRecipe
	}

	baseFocus, err := GetBaseFocusCost(baseInput.Recipe.Tier, baseInput.Recipe.Enchantment)
	if err != nil {
		return nil, err
	}

	actualFocus := CalculateActualFocus(baseFocus, masteryLevel)

	noFocusInput := baseInput
	noFocusInput.UseFocus = false
	noFocusRes, err := CalculateRefining(noFocusInput)
	if err != nil {
		return nil, err
	}

	focusInput := baseInput
	focusInput.UseFocus = true
	focusRes, err := CalculateRefining(focusInput)
	if err != nil {
		return nil, err
	}

	profitWithoutFocus := noFocusRes.NetProfitPerUnit
	profitWithFocus := focusRes.NetProfitPerUnit
	focusProfitDelta := profitWithFocus - profitWithoutFocus
	spf := CalculateSPF(profitWithFocus, profitWithoutFocus, actualFocus)
	dailyFocusSilver := spf * 10000.0

	return &FocusOptimizationResult{
		Recipe:             baseInput.Recipe,
		BaseFocusCost:      baseFocus,
		ActualFocusCost:    actualFocus,
		ProfitWithoutFocus: profitWithoutFocus,
		ProfitWithFocus:    profitWithFocus,
		FocusProfitDelta:   focusProfitDelta,
		SilverPerFocus:     spf,
		DailyFocusSilver:   dailyFocusSilver,
	}, nil
}
