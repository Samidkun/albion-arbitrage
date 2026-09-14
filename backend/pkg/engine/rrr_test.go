package engine

import (
	"math"
	"testing"
)

const floatEpsilon = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < floatEpsilon
}

func TestCalculateRRR(t *testing.T) {
	tests := []struct {
		name        string
		isBonusCity bool
		useFocus    bool
		wantRRR     float64
	}{
		{
			name:        "base city without focus",
			isBonusCity: false,
			useFocus:    false,
			wantRRR:     0.152,
		},
		{
			name:        "city bonus without focus",
			isBonusCity: true,
			useFocus:    false,
			wantRRR:     0.367,
		},
		{
			name:        "base city with focus",
			isBonusCity: false,
			useFocus:    true,
			wantRRR:     0.435,
		},
		{
			name:        "city bonus with focus",
			isBonusCity: true,
			useFocus:    true,
			wantRRR:     0.539,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateRRR(tc.isBonusCity, tc.useFocus)
			if !almostEqual(got, tc.wantRRR) {
				t.Errorf("CalculateRRR(%v, %v) = %f, want %f", tc.isBonusCity, tc.useFocus, got, tc.wantRRR)
			}
		})
	}
}

func TestCalculateEffectiveMaterialUsage(t *testing.T) {
	tests := []struct {
		name     string
		qty      int
		rrr      float64
		wantUsed float64
	}{
		{
			name:     "5 raw items with 53.9% RRR",
			qty:      5,
			rrr:      0.539,
			wantUsed: 5.0 * (1.0 - 0.539), // 2.305
		},
		{
			name:     "2 raw items with 36.7% RRR",
			qty:      2,
			rrr:      0.367,
			wantUsed: 2.0 * (1.0 - 0.367), // 1.266
		},
		{
			name:     "1 refined item with 15.2% RRR",
			qty:      1,
			rrr:      0.152,
			wantUsed: 1.0 * (1.0 - 0.152), // 0.848
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateEffectiveMaterialUsage(tc.qty, tc.rrr)
			if !almostEqual(got, tc.wantUsed) {
				t.Errorf("CalculateEffectiveMaterialUsage(%d, %f) = %f, want %f", tc.qty, tc.rrr, got, tc.wantUsed)
			}
		})
	}
}

func TestCalculateStationFee(t *testing.T) {
	tests := []struct {
		name      string
		itemValue int
		feeRate   float64
		wantFee   float64
	}{
		{
			name:      "T4 Bar (IV=16) at 600 silver/100 nutrition",
			itemValue: 16,
			feeRate:   600.0,
			wantFee:   10.8, // (16 * 0.1125 * 600) / 100 = 1.8 * 6 = 10.8
		},
		{
			name:      "T6.2 Bar (IV=256) at 800 silver/100 nutrition",
			itemValue: 256,
			feeRate:   800.0,
			wantFee:   230.4, // (256 * 0.1125 * 800) / 100 = 28.8 * 8 = 230.4
		},
		{
			name:      "Zero Item Value",
			itemValue: 0,
			feeRate:   600.0,
			wantFee:   0.0,
		},
		{
			name:      "Zero Fee Rate",
			itemValue: 128,
			feeRate:   0.0,
			wantFee:   0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateStationFee(tc.itemValue, tc.feeRate)
			if !almostEqual(got, tc.wantFee) {
				t.Errorf("CalculateStationFee(%d, %f) = %f, want %f", tc.itemValue, tc.feeRate, got, tc.wantFee)
			}
		})
	}
}

func TestCalculateEffectiveMaterialCost(t *testing.T) {
	tests := []struct {
		name             string
		rawPrice         float64
		prevRefinedPrice float64
		rawAmount        int
		prevAmount       int
		rrr              float64
		wantEffective    float64
		wantRawRefund    float64
		wantPrevRefund   float64
	}{
		{
			name:             "T5 Ore Refining in Thetford with Focus",
			rawPrice:         100.0,
			prevRefinedPrice: 200.0,
			rawAmount:        3,
			prevAmount:       1,
			rrr:              0.539,
			// rawRefund = 3 * 0.539 = 1.617
			// prevRefund = 1 * 0.539 = 0.539
			// grossCost = 100*3 + 200*1 = 500
			// effectiveCost = 500 * (1 - 0.539) = 230.5
			wantEffective:  230.5,
			wantRawRefund:  1.617,
			wantPrevRefund: 0.539,
		},
		{
			name:             "T4 Wood Refining in Fort Sterling without Focus",
			rawPrice:         50.0,
			prevRefinedPrice: 30.0,
			rawAmount:        2,
			prevAmount:       1,
			rrr:              0.367,
			// rawRefund = 2 * 0.367 = 0.734
			// prevRefund = 1 * 0.367 = 0.367
			// grossCost = 50*2 + 30*1 = 130
			// effectiveCost = 130 * (1 - 0.367) = 82.29
			wantEffective:  82.29,
			wantRawRefund:  0.734,
			wantPrevRefund: 0.367,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotEffective, gotRawRefund, gotPrevRefund := CalculateEffectiveMaterialCost(
				tc.rawPrice,
				tc.prevRefinedPrice,
				tc.rawAmount,
				tc.prevAmount,
				tc.rrr,
			)

			if !almostEqual(gotEffective, tc.wantEffective) {
				t.Errorf("CalculateEffectiveMaterialCost effective = %f, want %f", gotEffective, tc.wantEffective)
			}
			if !almostEqual(gotRawRefund, tc.wantRawRefund) {
				t.Errorf("CalculateEffectiveMaterialCost rawRefund = %f, want %f", gotRawRefund, tc.wantRawRefund)
			}
			if !almostEqual(gotPrevRefund, tc.wantPrevRefund) {
				t.Errorf("CalculateEffectiveMaterialCost prevRefund = %f, want %f", gotPrevRefund, tc.wantPrevRefund)
			}
		})
	}
}
