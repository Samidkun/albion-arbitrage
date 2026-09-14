package engine

import (
	"errors"
	"math"
	"testing"

	"albionflow/backend/pkg/model"
)

func TestGetBaseFocusCost(t *testing.T) {
	expectedCosts := map[int][]int{
		4: {34, 67, 135, 269, 538},
		5: {67, 135, 269, 538, 1076},
		6: {135, 269, 538, 1076, 2152},
		7: {269, 538, 1076, 2152, 4304},
		8: {538, 1076, 2152, 4304, 8608},
	}

	for tier, enchants := range expectedCosts {
		for enchant, want := range enchants {
			got, err := GetBaseFocusCost(tier, enchant)
			if err != nil {
				t.Fatalf("GetBaseFocusCost(%d, %d) unexpected error: %v", tier, enchant, err)
			}
			if got != want {
				t.Errorf("GetBaseFocusCost(%d, %d) = %d, want %d", tier, enchant, got, want)
			}
		}
	}

	// Invalid tiers
	if _, err := GetBaseFocusCost(3, 0); !errors.Is(err, model.ErrInvalidTier) {
		t.Errorf("GetBaseFocusCost(3, 0) expected ErrInvalidTier, got %v", err)
	}
	if _, err := GetBaseFocusCost(9, 0); !errors.Is(err, model.ErrInvalidTier) {
		t.Errorf("GetBaseFocusCost(9, 0) expected ErrInvalidTier, got %v", err)
	}

	// Invalid enchantments
	if _, err := GetBaseFocusCost(4, -1); !errors.Is(err, model.ErrInvalidEnchantment) {
		t.Errorf("GetBaseFocusCost(4, -1) expected ErrInvalidEnchantment, got %v", err)
	}
	if _, err := GetBaseFocusCost(4, 5); !errors.Is(err, model.ErrInvalidEnchantment) {
		t.Errorf("GetBaseFocusCost(4, 5) expected ErrInvalidEnchantment, got %v", err)
	}
}

func TestCalculateActualFocus(t *testing.T) {
	tests := []struct {
		name         string
		baseFocus    int
		masteryLevel int
		want         int
	}{
		{
			name:         "0 mastery is 100% cost",
			baseFocus:    538,
			masteryLevel: 0,
			want:         538,
		},
		{
			name:         "100 mastery is 50% cost",
			baseFocus:    538,
			masteryLevel: 100,
			want:         269,
		},
		{
			name:         "100 mastery with 34 base focus",
			baseFocus:    34,
			masteryLevel: 100,
			want:         17,
		},
		{
			name:         "50 mastery with 538 base focus",
			baseFocus:    538,
			masteryLevel: 50,
			want:         int(math.Round(538.0 * math.Pow(0.5, 0.5))), // 380
		},
		{
			name:         "negative mastery clamped to 0",
			baseFocus:    200,
			masteryLevel: -10,
			want:         200,
		},
		{
			name:         "mastery above 100 clamped to 100",
			baseFocus:    200,
			masteryLevel: 150,
			want:         100,
		},
		{
			name:         "minimum focus return value is 1",
			baseFocus:    0,
			masteryLevel: 100,
			want:         1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateActualFocus(tc.baseFocus, tc.masteryLevel)
			if got != tc.want {
				t.Errorf("CalculateActualFocus(%d, %d) = %d, want %d", tc.baseFocus, tc.masteryLevel, got, tc.want)
			}
		})
	}
}

func TestCalculateSPF(t *testing.T) {
	tests := []struct {
		name               string
		profitWithFocus    float64
		profitWithoutFocus float64
		focusCost          int
		want               float64
	}{
		{
			name:               "positive focus cost and positive profit delta",
			profitWithFocus:    200.0,
			profitWithoutFocus: 100.0,
			focusCost:          50,
			want:               2.0,
		},
		{
			name:               "zero focus cost returns 0.0",
			profitWithFocus:    200.0,
			profitWithoutFocus: 100.0,
			focusCost:          0,
			want:               0.0,
		},
		{
			name:               "negative focus cost returns 0.0",
			profitWithFocus:    200.0,
			profitWithoutFocus: 100.0,
			focusCost:          -5,
			want:               0.0,
		},
		{
			name:               "negative profit delta",
			profitWithFocus:    50.0,
			profitWithoutFocus: 100.0,
			focusCost:          50,
			want:               -1.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateSPF(tc.profitWithFocus, tc.profitWithoutFocus, tc.focusCost)
			if !almostEqual(got, tc.want) {
				t.Errorf("CalculateSPF() = %f, want %f", got, tc.want)
			}
		})
	}
}

func TestEvaluateFocusEfficiency(t *testing.T) {
	recipe, err := model.GetRefiningRecipe(model.CommodityOre, 5, 0)
	if err != nil {
		t.Fatalf("unexpected error getting recipe: %v", err)
	}

	baseInput := RefiningCostInput{
		Recipe:                    recipe,
		RawPrice:                  300.0,
		PrevRefinedPrice:          200.0,
		SellPrice:                 1500.0,
		StationFeePer100Nutrition: 600.0,
		IsBonusCity:               true, // bonus city gives 36.7% RRR without focus, 53.9% with focus
		UseFocus:                  false,
		SalesTaxRate:              0.04,
		SetupFeeRate:              0.025,
	}

	masteryLevel := 50
	res, err := EvaluateFocusEfficiency(baseInput, masteryLevel)
	if err != nil {
		t.Fatalf("EvaluateFocusEfficiency() unexpected error: %v", err)
	}

	// T5.0 base focus is 67
	if res.BaseFocusCost != 67 {
		t.Errorf("BaseFocusCost = %d, want 67", res.BaseFocusCost)
	}

	expectedActualFocus := CalculateActualFocus(67, 50)
	if res.ActualFocusCost != expectedActualFocus {
		t.Errorf("ActualFocusCost = %d, want %d", res.ActualFocusCost, expectedActualFocus)
	}

	// Focus must increase profit because RRR jumps from 36.7% to 53.9%
	if res.ProfitWithFocus <= res.ProfitWithoutFocus {
		t.Errorf("ProfitWithFocus (%f) <= ProfitWithoutFocus (%f)", res.ProfitWithFocus, res.ProfitWithoutFocus)
	}

	wantDelta := res.ProfitWithFocus - res.ProfitWithoutFocus
	if !almostEqual(res.FocusProfitDelta, wantDelta) {
		t.Errorf("FocusProfitDelta = %f, want %f", res.FocusProfitDelta, wantDelta)
	}

	wantSPF := wantDelta / float64(expectedActualFocus)
	if !almostEqual(res.SilverPerFocus, wantSPF) {
		t.Errorf("SilverPerFocus = %f, want %f", res.SilverPerFocus, wantSPF)
	}

	if !almostEqual(res.DailyFocusSilver, wantSPF*10000.0) {
		t.Errorf("DailyFocusSilver = %f, want %f", res.DailyFocusSilver, wantSPF*10000.0)
	}

	// Verify error handling
	nilInput := baseInput
	nilInput.Recipe = nil
	if _, err := EvaluateFocusEfficiency(nilInput, 50); !errors.Is(err, ErrNilRecipe) {
		t.Errorf("expected ErrNilRecipe, got %v", err)
	}

	negInput := baseInput
	negInput.RawPrice = -10.0
	if _, err := EvaluateFocusEfficiency(negInput, 50); !errors.Is(err, ErrNegativeValue) {
		t.Errorf("expected ErrNegativeValue, got %v", err)
	}
}
