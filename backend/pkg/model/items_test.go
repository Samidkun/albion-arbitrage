package model_test

import (
	"errors"
	"testing"

	"albionflow/backend/pkg/model"
)

func TestRecipeLookupsAllCommodities(t *testing.T) {
	commodities := []model.CommodityType{
		model.CommodityWood,
		model.CommodityOre,
		model.CommodityHide,
		model.CommodityFiber,
		model.CommodityStone,
	}

	tiers := []int{4, 5, 6, 7, 8}

	for _, commodity := range commodities {
		for _, tier := range tiers {
			for enchant := 0; enchant <= 4; enchant++ {
				recipe, err := model.GetRefiningRecipe(commodity, tier, enchant)

				if commodity == model.CommodityStone && enchant > 0 {
					if !errors.Is(err, model.ErrStoneCannotBeEnchanted) {
						t.Errorf("expected ErrStoneCannotBeEnchanted for stone T%d.%d, got %v", tier, enchant, err)
					}
					if recipe != nil {
						t.Errorf("expected nil recipe for enchanted stone, got %+v", recipe)
					}
					continue
				}

				if err != nil {
					t.Fatalf("unexpected error for %s T%d.%d: %v", commodity, tier, enchant, err)
				}
				if recipe == nil {
					t.Fatalf("expected recipe for %s T%d.%d, got nil", commodity, tier, enchant)
				}

				if recipe.Commodity != commodity {
					t.Errorf("expected commodity %s, got %s", commodity, recipe.Commodity)
				}
				if recipe.Tier != tier {
					t.Errorf("expected tier %d, got %d", tier, recipe.Tier)
				}
				if recipe.Enchantment != enchant {
					t.Errorf("expected enchant %d, got %d", enchant, recipe.Enchantment)
				}
			}
		}
	}
}

func TestStoneException(t *testing.T) {
	for tier := 4; tier <= 8; tier++ {
		// Enchant 0 should succeed
		recipe, err := model.GetRefiningRecipe(model.CommodityStone, tier, 0)
		if err != nil {
			t.Fatalf("unexpected error for Stone T%d.0: %v", tier, err)
		}
		if recipe == nil {
			t.Fatalf("expected recipe for Stone T%d.0, got nil", tier)
		}
		if recipe.Enchantment != 0 {
			t.Errorf("expected enchant 0, got %d", recipe.Enchantment)
		}

		// Enchant 1 to 4 should return ErrStoneCannotBeEnchanted
		for enchant := 1; enchant <= 4; enchant++ {
			r, err := model.GetRefiningRecipe(model.CommodityStone, tier, enchant)
			if !errors.Is(err, model.ErrStoneCannotBeEnchanted) {
				t.Errorf("expected ErrStoneCannotBeEnchanted for Stone T%d.%d, got %v", tier, enchant, err)
			}
			if r != nil {
				t.Errorf("expected nil recipe for Stone T%d.%d, got %+v", tier, enchant, r)
			}
		}
	}

	// Also test GetRecipeByResultID for enchanted stone
	enchantedStoneIDs := []string{
		"T4_STONEBLOCK_LEVEL1@1",
		"T5_STONEBLOCK_LEVEL2@2",
		"T6_STONEBLOCK_LEVEL3@3",
		"T7_STONEBLOCK_LEVEL4@4",
		"T8_STONEBLOCK_LEVEL1@1",
	}
	for _, id := range enchantedStoneIDs {
		r, err := model.GetRecipeByResultID(id)
		if !errors.Is(err, model.ErrStoneCannotBeEnchanted) {
			t.Errorf("expected ErrStoneCannotBeEnchanted for %s, got %v", id, err)
		}
		if r != nil {
			t.Errorf("expected nil recipe for %s, got %+v", id, r)
		}
	}
}

func TestItemValueCalculations(t *testing.T) {
	tests := []struct {
		commodity model.CommodityType
		tier      int
		enchant   int
		expected  int
	}{
		{model.CommodityWood, 4, 0, 16},
		{model.CommodityWood, 4, 1, 32},
		{model.CommodityOre, 5, 0, 32},
		{model.CommodityOre, 6, 2, 256},
		{model.CommodityHide, 7, 0, 128},
		{model.CommodityHide, 7, 3, 1024},
		{model.CommodityFiber, 8, 0, 256},
		{model.CommodityFiber, 8, 4, 4096},
		{model.CommodityStone, 4, 0, 16},
		{model.CommodityStone, 8, 0, 256},
	}

	for _, tt := range tests {
		recipe, err := model.GetRefiningRecipe(tt.commodity, tt.tier, tt.enchant)
		if err != nil {
			t.Fatalf("unexpected error for %s T%d.%d: %v", tt.commodity, tt.tier, tt.enchant, err)
		}
		if recipe.ItemValue != tt.expected {
			t.Errorf("ItemValue for %s T%d.%d = %d; want %d",
				tt.commodity, tt.tier, tt.enchant, recipe.ItemValue, tt.expected)
		}
	}
}

func TestRequiredAmounts(t *testing.T) {
	expectedRawAmount := map[int]int{
		4: 2,
		5: 3,
		6: 4,
		7: 5,
		8: 5,
	}

	recipes := model.GetAllRecipes()
	if len(recipes) != 105 {
		t.Fatalf("expected 105 total recipes, got %d", len(recipes))
	}

	for _, r := range recipes {
		expRaw, ok := expectedRawAmount[r.Tier]
		if !ok {
			t.Errorf("unexpected tier %d in recipe %s", r.Tier, r.ResultItemID)
		}
		if r.RawAmount != expRaw {
			t.Errorf("recipe %s RawAmount = %d; want %d", r.ResultItemID, r.RawAmount, expRaw)
		}
		if r.PrevRefinedAmount != 1 {
			t.Errorf("recipe %s PrevRefinedAmount = %d; want 1", r.ResultItemID, r.PrevRefinedAmount)
		}
	}
}

func TestItemIDsAndRecipeLookups(t *testing.T) {
	tests := []struct {
		commodity         model.CommodityType
		tier              int
		enchant           int
		expectedResultID  string
		expectedRawID     string
		expectedPrevRefID string
	}{
		// Wood
		{model.CommodityWood, 4, 0, "T4_PLANKS", "T4_WOOD", "T3_PLANKS"},
		{model.CommodityWood, 4, 1, "T4_PLANKS_LEVEL1@1", "T4_WOOD_LEVEL1@1", "T3_PLANKS"},
		{model.CommodityWood, 5, 0, "T5_PLANKS", "T5_WOOD", "T4_PLANKS"},
		{model.CommodityWood, 5, 1, "T5_PLANKS_LEVEL1@1", "T5_WOOD_LEVEL1@1", "T4_PLANKS_LEVEL1@1"},
		{model.CommodityWood, 6, 2, "T6_PLANKS_LEVEL2@2", "T6_WOOD_LEVEL2@2", "T5_PLANKS_LEVEL2@2"},
		// Ore
		{model.CommodityOre, 4, 0, "T4_METALBAR", "T4_ORE", "T3_METALBAR"},
		{model.CommodityOre, 5, 0, "T5_METALBAR", "T5_ORE", "T4_METALBAR"},
		{model.CommodityOre, 6, 1, "T6_METALBAR_LEVEL1@1", "T6_ORE_LEVEL1@1", "T5_METALBAR_LEVEL1@1"},
		// Hide
		{model.CommodityHide, 4, 0, "T4_LEATHER", "T4_HIDE", "T3_LEATHER"},
		{model.CommodityHide, 7, 3, "T7_LEATHER_LEVEL3@3", "T7_HIDE_LEVEL3@3", "T6_LEATHER_LEVEL3@3"},
		// Fiber
		{model.CommodityFiber, 4, 0, "T4_CLOTH", "T4_FIBER", "T3_CLOTH"},
		{model.CommodityFiber, 8, 4, "T8_CLOTH_LEVEL4@4", "T8_FIBER_LEVEL4@4", "T7_CLOTH_LEVEL4@4"},
		// Stone
		{model.CommodityStone, 4, 0, "T4_STONEBLOCK", "T4_ROCK", "T3_STONEBLOCK"},
		{model.CommodityStone, 5, 0, "T5_STONEBLOCK", "T5_ROCK", "T4_STONEBLOCK"},
		{model.CommodityStone, 8, 0, "T8_STONEBLOCK", "T8_ROCK", "T7_STONEBLOCK"},
	}

	for _, tt := range tests {
		recipe, err := model.GetRefiningRecipe(tt.commodity, tt.tier, tt.enchant)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if recipe.ResultItemID != tt.expectedResultID {
			t.Errorf("ResultItemID = %s; want %s", recipe.ResultItemID, tt.expectedResultID)
		}
		if recipe.RawItemID != tt.expectedRawID {
			t.Errorf("RawItemID = %s; want %s", recipe.RawItemID, tt.expectedRawID)
		}
		if recipe.PrevRefinedItemID != tt.expectedPrevRefID {
			t.Errorf("PrevRefinedItemID = %s; want %s", recipe.PrevRefinedItemID, tt.expectedPrevRefID)
		}

		// Also verify lookup by ResultItemID
		lookupByResult, err := model.GetRecipeByResultID(tt.expectedResultID)
		if err != nil {
			t.Fatalf("unexpected error looking up %s: %v", tt.expectedResultID, err)
		}
		if lookupByResult.ResultItemID != tt.expectedResultID {
			t.Errorf("GetRecipeByResultID got %s; want %s", lookupByResult.ResultItemID, tt.expectedResultID)
		}
	}
}

func TestBonusCityMapping(t *testing.T) {
	tests := []struct {
		commodity model.CommodityType
		city      model.City
	}{
		{model.CommodityWood, model.CityFortSterling},
		{model.CommodityOre, model.CityThetford},
		{model.CommodityHide, model.CityMartlock},
		{model.CommodityFiber, model.CityLymhurst},
		{model.CommodityStone, model.CityBridgewatch},
	}

	for _, tt := range tests {
		city := model.GetBonusCity(tt.commodity)
		if city != tt.city {
			t.Errorf("GetBonusCity(%s) = %s; want %s", tt.commodity, city, tt.city)
		}
	}
}

func TestInvalidInputs(t *testing.T) {
	// Invalid commodity
	_, err := model.GetRefiningRecipe("Invalid", 4, 0)
	if !errors.Is(err, model.ErrInvalidCommodity) {
		t.Errorf("expected ErrInvalidCommodity, got %v", err)
	}

	// Invalid tier
	_, err = model.GetRefiningRecipe(model.CommodityWood, 3, 0)
	if !errors.Is(err, model.ErrInvalidTier) {
		t.Errorf("expected ErrInvalidTier for tier 3, got %v", err)
	}
	_, err = model.GetRefiningRecipe(model.CommodityWood, 9, 0)
	if !errors.Is(err, model.ErrInvalidTier) {
		t.Errorf("expected ErrInvalidTier for tier 9, got %v", err)
	}

	// Invalid enchantment
	_, err = model.GetRefiningRecipe(model.CommodityWood, 4, -1)
	if !errors.Is(err, model.ErrInvalidEnchantment) {
		t.Errorf("expected ErrInvalidEnchantment for enchant -1, got %v", err)
	}
	_, err = model.GetRefiningRecipe(model.CommodityWood, 4, 5)
	if !errors.Is(err, model.ErrInvalidEnchantment) {
		t.Errorf("expected ErrInvalidEnchantment for enchant 5, got %v", err)
	}

	// Non-existent ResultItemID
	_, err = model.GetRecipeByResultID("NON_EXISTENT_ITEM")
	if !errors.Is(err, model.ErrRecipeNotFound) {
		t.Errorf("expected ErrRecipeNotFound for unknown item, got %v", err)
	}
}
