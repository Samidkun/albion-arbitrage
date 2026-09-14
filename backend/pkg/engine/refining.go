package engine

import (
	"errors"

	"albionflow/backend/pkg/model"
)

var (
	// ErrNilRecipe is returned when refining calculation is called without a recipe.
	ErrNilRecipe = errors.New("recipe cannot be nil")
	// ErrNegativeValue is returned when input prices, costs, or rates are negative.
	ErrNegativeValue = errors.New("prices, costs, and fee rates must not be negative")
	// ErrInvalidBatchSize is returned when batch size is less than or equal to zero.
	ErrInvalidBatchSize = errors.New("batch size must be greater than zero")
)

// RefiningCostInput holds all parameter inputs required to calculate refining profitability.
type RefiningCostInput struct {
	Recipe                    *model.RefiningRecipe
	RawPrice                  float64
	PrevRefinedPrice          float64
	SellPrice                 float64
	StationFeePer100Nutrition float64
	IsBonusCity               bool
	UseFocus                  bool
	SalesTaxRate              float64
	SetupFeeRate              float64
	IsChainedRefine           bool
	PrevTierCraftCost         float64
}

// RefiningCostResult details the per-unit costs, returns, fees, and net profit of refining.
type RefiningCostResult struct {
	Recipe                *model.RefiningRecipe
	RRR                   float64
	EffectiveRawCost      float64
	EffectivePrevCost     float64
	StationFee            float64
	TotalCraftCostPerUnit float64
	GrossRevenue          float64
	SalesTax              float64
	SetupFee              float64
	NetRevenue            float64
	NetProfitPerUnit      float64
	ProfitMarginPercent   float64
	IsChainedRefine       bool
}

// BatchRefiningResult summarizes multi-craft requirements, refunds, capital, and net profit.
type BatchRefiningResult struct {
	PerUnit                  *RefiningCostResult
	BatchSize                int
	TotalRawRequired         int
	TotalPrevRefinedRequired int
	EstimatedRawRefund       float64
	EstimatedPrevRefund      float64
	TotalCapitalRequired     float64
	TotalNetProfit           float64
}

// CalculateRefining computes unit refining economics, RRR material returns, and net profit.
func CalculateRefining(input RefiningCostInput) (*RefiningCostResult, error) {
	if input.Recipe == nil {
		return nil, ErrNilRecipe
	}
	if input.RawPrice < 0 || input.PrevRefinedPrice < 0 || input.SellPrice < 0 ||
		input.StationFeePer100Nutrition < 0 || input.SalesTaxRate < 0 || input.SetupFeeRate < 0 {
		return nil, ErrNegativeValue
	}
	if input.IsChainedRefine && input.PrevTierCraftCost < 0 {
		return nil, ErrNegativeValue
	}

	rrr := CalculateRRR(input.IsBonusCity, input.UseFocus)

	prevCost := input.PrevRefinedPrice
	if input.IsChainedRefine {
		prevCost = input.PrevTierCraftCost
	}

	effectiveRawCost := float64(input.Recipe.RawAmount) * input.RawPrice * (1.0 - rrr)
	effectivePrevCost := float64(input.Recipe.PrevRefinedAmount) * prevCost * (1.0 - rrr)
	stationFee := CalculateStationFee(input.Recipe.ItemValue, input.StationFeePer100Nutrition)
	totalCraftCostPerUnit := effectiveRawCost + effectivePrevCost + stationFee

	grossRevenue := input.SellPrice
	salesTax := grossRevenue * input.SalesTaxRate
	setupFee := grossRevenue * input.SetupFeeRate
	netRevenue := grossRevenue - salesTax - setupFee
	netProfitPerUnit := netRevenue - totalCraftCostPerUnit

	// ponytail: profit margin is calculated as ROI relative to total craft cost (ceiling: single ROI formula; add revenue-based margin if alternate view is required).
	var profitMarginPercent float64
	if totalCraftCostPerUnit > 0 {
		profitMarginPercent = (netProfitPerUnit / totalCraftCostPerUnit) * 100.0
	}

	return &RefiningCostResult{
		Recipe:                input.Recipe,
		RRR:                   rrr,
		EffectiveRawCost:      effectiveRawCost,
		EffectivePrevCost:     effectivePrevCost,
		StationFee:            stationFee,
		TotalCraftCostPerUnit: totalCraftCostPerUnit,
		GrossRevenue:          grossRevenue,
		SalesTax:              salesTax,
		SetupFee:              setupFee,
		NetRevenue:            netRevenue,
		NetProfitPerUnit:      netProfitPerUnit,
		ProfitMarginPercent:   profitMarginPercent,
		IsChainedRefine:       input.IsChainedRefine,
	}, nil
}

// CalculateBatchRefining computes batch-level material requirements, estimated refunds, upfront capital, and total profit.
func CalculateBatchRefining(input RefiningCostInput, batchSize int) (*BatchRefiningResult, error) {
	if batchSize <= 0 {
		return nil, ErrInvalidBatchSize
	}

	perUnit, err := CalculateRefining(input)
	if err != nil {
		return nil, err
	}

	prevCost := input.PrevRefinedPrice
	if input.IsChainedRefine {
		prevCost = input.PrevTierCraftCost
	}

	totalRaw := input.Recipe.RawAmount * batchSize
	totalPrev := input.Recipe.PrevRefinedAmount * batchSize

	estRawRefund := float64(totalRaw) * perUnit.RRR
	estPrevRefund := float64(totalPrev) * perUnit.RRR

	totalCapital := (float64(totalRaw)*input.RawPrice + float64(totalPrev)*prevCost) + (perUnit.StationFee * float64(batchSize))
	totalProfit := perUnit.NetProfitPerUnit * float64(batchSize)

	return &BatchRefiningResult{
		PerUnit:                  perUnit,
		BatchSize:                batchSize,
		TotalRawRequired:         totalRaw,
		TotalPrevRefinedRequired: totalPrev,
		EstimatedRawRefund:       estRawRefund,
		EstimatedPrevRefund:      estPrevRefund,
		TotalCapitalRequired:     totalCapital,
		TotalNetProfit:           totalProfit,
	}, nil
}
