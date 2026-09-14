package engine

import (
	"errors"
	"testing"

	"albionflow/backend/pkg/model"
)

func TestCalculateRefining_T4Comparison(t *testing.T) {
	recipe, err := model.GetRefiningRecipe(model.CommodityOre, 4, 0)
	if err != nil {
		t.Fatalf("unexpected error getting recipe: %v", err)
	}

	baseInput := RefiningCostInput{
		Recipe:                    recipe,
		RawPrice:                  100.0,
		PrevRefinedPrice:          50.0,
		SellPrice:                 300.0,
		StationFeePer100Nutrition: 600.0,
		IsBonusCity:               false,
		UseFocus:                  false,
		SalesTaxRate:              0.04,  // Premium 4%
		SetupFeeRate:              0.025, // Sell order 2.5%
		IsChainedRefine:           false,
	}

	bonusFocusInput := baseInput
	bonusFocusInput.IsBonusCity = true
	bonusFocusInput.UseFocus = true

	baseRes, err := CalculateRefining(baseInput)
	if err != nil {
		t.Fatalf("CalculateRefining(baseInput) error: %v", err)
	}

	bonusFocusRes, err := CalculateRefining(bonusFocusInput)
	if err != nil {
		t.Fatalf("CalculateRefining(bonusFocusInput) error: %v", err)
	}

	// Verify Base RRR (15.2%)
	if !almostEqual(baseRes.RRR, RRRBase) {
		t.Errorf("baseRes.RRR = %f, want %f", baseRes.RRR, RRRBase)
	}
	// T4 Ore recipe: RawAmount=2, PrevRefinedAmount=1, ItemValue=16
	// EffectiveRawCost = 2 * 100 * (1 - 0.152) = 200 * 0.848 = 169.6
	wantBaseRawCost := 200.0 * (1.0 - 0.152)
	if !almostEqual(baseRes.EffectiveRawCost, wantBaseRawCost) {
		t.Errorf("baseRes.EffectiveRawCost = %f, want %f", baseRes.EffectiveRawCost, wantBaseRawCost)
	}
	// EffectivePrevCost = 1 * 50 * (1 - 0.152) = 50 * 0.848 = 42.4
	wantBasePrevCost := 50.0 * (1.0 - 0.152)
	if !almostEqual(baseRes.EffectivePrevCost, wantBasePrevCost) {
		t.Errorf("baseRes.EffectivePrevCost = %f, want %f", baseRes.EffectivePrevCost, wantBasePrevCost)
	}
	// Station fee = (16 * 0.1125 * 600) / 100 = 10.8
	wantStationFee := 10.8
	if !almostEqual(baseRes.StationFee, wantStationFee) {
		t.Errorf("baseRes.StationFee = %f, want %f", baseRes.StationFee, wantStationFee)
	}
	wantBaseCraftCost := wantBaseRawCost + wantBasePrevCost + wantStationFee
	if !almostEqual(baseRes.TotalCraftCostPerUnit, wantBaseCraftCost) {
		t.Errorf("baseRes.TotalCraftCostPerUnit = %f, want %f", baseRes.TotalCraftCostPerUnit, wantBaseCraftCost)
	}
	// GrossRevenue = 300
	// SalesTax = 300 * 0.04 = 12.0
	// SetupFee = 300 * 0.025 = 7.5
	// NetRevenue = 300 - 12 - 7.5 = 280.5
	wantNetRevenue := 300.0 - 12.0 - 7.5
	if !almostEqual(baseRes.NetRevenue, wantNetRevenue) {
		t.Errorf("baseRes.NetRevenue = %f, want %f", baseRes.NetRevenue, wantNetRevenue)
	}
	wantBaseProfit := wantNetRevenue - wantBaseCraftCost
	if !almostEqual(baseRes.NetProfitPerUnit, wantBaseProfit) {
		t.Errorf("baseRes.NetProfitPerUnit = %f, want %f", baseRes.NetProfitPerUnit, wantBaseProfit)
	}
	wantBaseMargin := (wantBaseProfit / wantBaseCraftCost) * 100.0
	if !almostEqual(baseRes.ProfitMarginPercent, wantBaseMargin) {
		t.Errorf("baseRes.ProfitMarginPercent = %f, want %f", baseRes.ProfitMarginPercent, wantBaseMargin)
	}

	// Verify Bonus City + Focus RRR (53.9%)
	if !almostEqual(bonusFocusRes.RRR, RRRCityBonusAndFocus) {
		t.Errorf("bonusFocusRes.RRR = %f, want %f", bonusFocusRes.RRR, RRRCityBonusAndFocus)
	}
	wantBonusRawCost := 200.0 * (1.0 - 0.539)
	if !almostEqual(bonusFocusRes.EffectiveRawCost, wantBonusRawCost) {
		t.Errorf("bonusFocusRes.EffectiveRawCost = %f, want %f", bonusFocusRes.EffectiveRawCost, wantBonusRawCost)
	}
	wantBonusPrevCost := 50.0 * (1.0 - 0.539)
	if !almostEqual(bonusFocusRes.EffectivePrevCost, wantBonusPrevCost) {
		t.Errorf("bonusFocusRes.EffectivePrevCost = %f, want %f", bonusFocusRes.EffectivePrevCost, wantBonusPrevCost)
	}
	// Station fee remains unchanged: station fee does not scale with RRR
	if !almostEqual(bonusFocusRes.StationFee, wantStationFee) {
		t.Errorf("bonusFocusRes.StationFee = %f, want %f", bonusFocusRes.StationFee, wantStationFee)
	}
	// Bonus + Focus should have strictly lower craft cost and higher net profit
	if bonusFocusRes.TotalCraftCostPerUnit >= baseRes.TotalCraftCostPerUnit {
		t.Errorf("bonusFocus craft cost (%f) should be strictly less than base (%f)",
			bonusFocusRes.TotalCraftCostPerUnit, baseRes.TotalCraftCostPerUnit)
	}
	if bonusFocusRes.NetProfitPerUnit <= baseRes.NetProfitPerUnit {
		t.Errorf("bonusFocus net profit (%f) should be strictly greater than base (%f)",
			bonusFocusRes.NetProfitPerUnit, baseRes.NetProfitPerUnit)
	}
}

func TestCalculateRefining_HierarchicalT5T6(t *testing.T) {
	// Test T5: 3 raw + 1 T4 refined
	t5Recipe, err := model.GetRefiningRecipe(model.CommodityWood, 5, 0)
	if err != nil {
		t.Fatalf("GetRefiningRecipe(Wood, 5, 0) error: %v", err)
	}
	if t5Recipe.RawAmount != 3 || t5Recipe.PrevRefinedAmount != 1 {
		t.Fatalf("unexpected T5 recipe inputs: raw=%d, prev=%d", t5Recipe.RawAmount, t5Recipe.PrevRefinedAmount)
	}

	t5Input := RefiningCostInput{
		Recipe:                    t5Recipe,
		RawPrice:                  200.0,
		PrevRefinedPrice:          350.0,
		SellPrice:                 1200.0,
		StationFeePer100Nutrition: 500.0,
		IsBonusCity:               true, // Fort Sterling bonus for Wood
		UseFocus:                  false,
		SalesTaxRate:              0.04,
		SetupFeeRate:              0.025,
	}

	t5Res, err := CalculateRefining(t5Input)
	if err != nil {
		t.Fatalf("CalculateRefining(t5Input) error: %v", err)
	}

	rrr := RRRCityBonus // 0.367
	expectedRawCost := 3.0 * 200.0 * (1.0 - rrr)
	expectedPrevCost := 1.0 * 350.0 * (1.0 - rrr)
	if !almostEqual(t5Res.EffectiveRawCost, expectedRawCost) {
		t.Errorf("t5 EffectiveRawCost = %f, want %f", t5Res.EffectiveRawCost, expectedRawCost)
	}
	if !almostEqual(t5Res.EffectivePrevCost, expectedPrevCost) {
		t.Errorf("t5 EffectivePrevCost = %f, want %f", t5Res.EffectivePrevCost, expectedPrevCost)
	}

	// Test T6: 4 raw + 1 T5 refined
	t6Recipe, err := model.GetRefiningRecipe(model.CommodityWood, 6, 0)
	if err != nil {
		t.Fatalf("GetRefiningRecipe(Wood, 6, 0) error: %v", err)
	}
	if t6Recipe.RawAmount != 4 || t6Recipe.PrevRefinedAmount != 1 {
		t.Fatalf("unexpected T6 recipe inputs: raw=%d, prev=%d", t6Recipe.RawAmount, t6Recipe.PrevRefinedAmount)
	}

	t6Input := RefiningCostInput{
		Recipe:                    t6Recipe,
		RawPrice:                  500.0,
		PrevRefinedPrice:          1100.0,
		SellPrice:                 3800.0,
		StationFeePer100Nutrition: 500.0,
		IsBonusCity:               true,
		UseFocus:                  false,
		SalesTaxRate:              0.04,
		SetupFeeRate:              0.025,
	}

	t6Res, err := CalculateRefining(t6Input)
	if err != nil {
		t.Fatalf("CalculateRefining(t6Input) error: %v", err)
	}

	expectedT6RawCost := 4.0 * 500.0 * (1.0 - rrr)
	expectedT6PrevCost := 1.0 * 1100.0 * (1.0 - rrr)
	if !almostEqual(t6Res.EffectiveRawCost, expectedT6RawCost) {
		t.Errorf("t6 EffectiveRawCost = %f, want %f", t6Res.EffectiveRawCost, expectedT6RawCost)
	}
	if !almostEqual(t6Res.EffectivePrevCost, expectedT6PrevCost) {
		t.Errorf("t6 EffectivePrevCost = %f, want %f", t6Res.EffectivePrevCost, expectedT6PrevCost)
	}
}

func TestCalculateRefining_ChainedRefining(t *testing.T) {
	recipe, err := model.GetRefiningRecipe(model.CommodityFiber, 5, 0)
	if err != nil {
		t.Fatalf("GetRefiningRecipe error: %v", err)
	}

	// Unchained: buying T4 cloth from market at 400 silver
	unchainedInput := RefiningCostInput{
		Recipe:                    recipe,
		RawPrice:                  150.0,
		PrevRefinedPrice:          400.0,
		SellPrice:                 1000.0,
		StationFeePer100Nutrition: 600.0,
		IsBonusCity:               true, // Lymhurst bonus for Fiber
		UseFocus:                  false,
		SalesTaxRate:              0.04,
		SetupFeeRate:              0.025,
		IsChainedRefine:           false,
		PrevTierCraftCost:         0.0,
	}

	// Chained: player refines T4 cloth themselves at a craft cost of 260 silver
	// (eliminates market tax, setup fee, and seller's markup on intermediate T4)
	chainedInput := unchainedInput
	chainedInput.IsChainedRefine = true
	chainedInput.PrevTierCraftCost = 260.0

	unchainedRes, err := CalculateRefining(unchainedInput)
	if err != nil {
		t.Fatalf("CalculateRefining(unchained) error: %v", err)
	}

	chainedRes, err := CalculateRefining(chainedInput)
	if err != nil {
		t.Fatalf("CalculateRefining(chained) error: %v", err)
	}

	// Verify chained flag is reflected
	if unchainedRes.IsChainedRefine != false {
		t.Errorf("unchainedRes.IsChainedRefine = true, want false")
	}
	if chainedRes.IsChainedRefine != true {
		t.Errorf("chainedRes.IsChainedRefine = false, want true")
	}

	// EffectivePrevCost should use PrevTierCraftCost (260) instead of PrevRefinedPrice (400)
	rrr := RRRCityBonus
	expectedChainedPrevCost := 260.0 * (1.0 - rrr)
	if !almostEqual(chainedRes.EffectivePrevCost, expectedChainedPrevCost) {
		t.Errorf("chainedRes.EffectivePrevCost = %f, want %f", chainedRes.EffectivePrevCost, expectedChainedPrevCost)
	}

	// Total craft cost saving should equal (400 - 260) * (1 - rrr)
	expectedSaving := (400.0 - 260.0) * (1.0 - rrr)
	costDiff := unchainedRes.TotalCraftCostPerUnit - chainedRes.TotalCraftCostPerUnit
	if !almostEqual(costDiff, expectedSaving) {
		t.Errorf("costDiff = %f, want %f", costDiff, expectedSaving)
	}

	// Profit should increase by the exact craft cost reduction
	profitDiff := chainedRes.NetProfitPerUnit - unchainedRes.NetProfitPerUnit
	if !almostEqual(profitDiff, expectedSaving) {
		t.Errorf("profitDiff = %f, want %f", profitDiff, expectedSaving)
	}
}

func TestCalculateBatchRefining(t *testing.T) {
	recipe, err := model.GetRefiningRecipe(model.CommodityHide, 5, 1) // T5.1 Leather
	if err != nil {
		t.Fatalf("GetRefiningRecipe error: %v", err)
	}

	input := RefiningCostInput{
		Recipe:                    recipe,
		RawPrice:                  300.0,
		PrevRefinedPrice:          500.0,
		SellPrice:                 1800.0,
		StationFeePer100Nutrition: 600.0,
		IsBonusCity:               true, // Martlock bonus for Hide
		UseFocus:                  true,
		SalesTaxRate:              0.04,
		SetupFeeRate:              0.025,
		IsChainedRefine:           false,
	}

	unitRes, err := CalculateRefining(input)
	if err != nil {
		t.Fatalf("CalculateRefining error: %v", err)
	}

	unitBatch, err := CalculateBatchRefining(input, 1)
	if err != nil {
		t.Fatalf("CalculateBatchRefining(1) error: %v", err)
	}

	batchSizes := []int{10, 50, 100, 500}
	for _, size := range batchSizes {
		bRes, err := CalculateBatchRefining(input, size)
		if err != nil {
			t.Fatalf("CalculateBatchRefining(%d) error: %v", size, err)
		}

		if bRes.BatchSize != size {
			t.Errorf("BatchSize = %d, want %d", bRes.BatchSize, size)
		}

		// Linear scaling assertions:
		wantRaw := unitBatch.TotalRawRequired * size
		if bRes.TotalRawRequired != wantRaw {
			t.Errorf("size %d TotalRawRequired = %d, want %d", size, bRes.TotalRawRequired, wantRaw)
		}

		wantPrev := unitBatch.TotalPrevRefinedRequired * size
		if bRes.TotalPrevRefinedRequired != wantPrev {
			t.Errorf("size %d TotalPrevRefinedRequired = %d, want %d", size, bRes.TotalPrevRefinedRequired, wantPrev)
		}

		wantRawRefund := unitBatch.EstimatedRawRefund * float64(size)
		if !almostEqual(bRes.EstimatedRawRefund, wantRawRefund) {
			t.Errorf("size %d EstimatedRawRefund = %f, want %f", size, bRes.EstimatedRawRefund, wantRawRefund)
		}

		wantPrevRefund := unitBatch.EstimatedPrevRefund * float64(size)
		if !almostEqual(bRes.EstimatedPrevRefund, wantPrevRefund) {
			t.Errorf("size %d EstimatedPrevRefund = %f, want %f", size, bRes.EstimatedPrevRefund, wantPrevRefund)
		}

		wantCapital := unitBatch.TotalCapitalRequired * float64(size)
		if !almostEqual(bRes.TotalCapitalRequired, wantCapital) {
			t.Errorf("size %d TotalCapitalRequired = %f, want %f", size, bRes.TotalCapitalRequired, wantCapital)
		}

		wantProfit := unitRes.NetProfitPerUnit * float64(size)
		if !almostEqual(bRes.TotalNetProfit, wantProfit) {
			t.Errorf("size %d TotalNetProfit = %f, want %f", size, bRes.TotalNetProfit, wantProfit)
		}
	}
}

func TestCalculateRefining_ValidationErrors(t *testing.T) {
	recipe, err := model.GetRefiningRecipe(model.CommodityOre, 4, 0)
	if err != nil {
		t.Fatalf("GetRefiningRecipe error: %v", err)
	}

	validInput := RefiningCostInput{
		Recipe:                    recipe,
		RawPrice:                  100.0,
		PrevRefinedPrice:          50.0,
		SellPrice:                 200.0,
		StationFeePer100Nutrition: 500.0,
		SalesTaxRate:              0.04,
		SetupFeeRate:              0.025,
	}

	// 1. Nil Recipe
	nilRecipeInput := validInput
	nilRecipeInput.Recipe = nil
	if _, err := CalculateRefining(nilRecipeInput); !errors.Is(err, ErrNilRecipe) {
		t.Errorf("expected ErrNilRecipe, got: %v", err)
	}

	// 2. Negative RawPrice
	negRaw := validInput
	negRaw.RawPrice = -1.0
	if _, err := CalculateRefining(negRaw); !errors.Is(err, ErrNegativeValue) {
		t.Errorf("expected ErrNegativeValue for negative raw price, got: %v", err)
	}

	// 3. Negative PrevRefinedPrice
	negPrev := validInput
	negPrev.PrevRefinedPrice = -5.0
	if _, err := CalculateRefining(negPrev); !errors.Is(err, ErrNegativeValue) {
		t.Errorf("expected ErrNegativeValue for negative prev price, got: %v", err)
	}

	// 4. Negative SellPrice
	negSell := validInput
	negSell.SellPrice = -10.0
	if _, err := CalculateRefining(negSell); !errors.Is(err, ErrNegativeValue) {
		t.Errorf("expected ErrNegativeValue for negative sell price, got: %v", err)
	}

	// 5. Negative StationFee
	negFee := validInput
	negFee.StationFeePer100Nutrition = -100.0
	if _, err := CalculateRefining(negFee); !errors.Is(err, ErrNegativeValue) {
		t.Errorf("expected ErrNegativeValue for negative station fee, got: %v", err)
	}

	// 6. Negative Tax Rate
	negTax := validInput
	negTax.SalesTaxRate = -0.01
	if _, err := CalculateRefining(negTax); !errors.Is(err, ErrNegativeValue) {
		t.Errorf("expected ErrNegativeValue for negative tax rate, got: %v", err)
	}

	// 7. Negative Setup Fee Rate
	negSetup := validInput
	negSetup.SetupFeeRate = -0.01
	if _, err := CalculateRefining(negSetup); !errors.Is(err, ErrNegativeValue) {
		t.Errorf("expected ErrNegativeValue for negative setup fee, got: %v", err)
	}

	// 8. Chained refine with negative craft cost
	negChained := validInput
	negChained.IsChainedRefine = true
	negChained.PrevTierCraftCost = -20.0
	if _, err := CalculateRefining(negChained); !errors.Is(err, ErrNegativeValue) {
		t.Errorf("expected ErrNegativeValue for negative chained prev craft cost, got: %v", err)
	}

	// 9. Batch size <= 0
	if _, err := CalculateBatchRefining(validInput, 0); !errors.Is(err, ErrInvalidBatchSize) {
		t.Errorf("expected ErrInvalidBatchSize for batch size 0, got: %v", err)
	}
	if _, err := CalculateBatchRefining(validInput, -5); !errors.Is(err, ErrInvalidBatchSize) {
		t.Errorf("expected ErrInvalidBatchSize for batch size -5, got: %v", err)
	}
}
