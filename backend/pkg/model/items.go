package model

import (
	"errors"
	"fmt"
	"strings"
)

// CommodityType represents the refineable raw/refined resource types in Albion Online.
type CommodityType string

const (
	CommodityWood  CommodityType = "Wood"
	CommodityOre   CommodityType = "Ore"
	CommodityHide  CommodityType = "Hide"
	CommodityFiber CommodityType = "Fiber"
	CommodityStone CommodityType = "Stone"

	// Short-hand aliases
	Wood  = CommodityWood
	Ore   = CommodityOre
	Hide  = CommodityHide
	Fiber = CommodityFiber
	Stone = CommodityStone
)

// Tier represents refining tiers (T4 through T8).
type Tier int

const (
	Tier4 Tier = 4
	Tier5 Tier = 5
	Tier6 Tier = 6
	Tier7 Tier = 7
	Tier8 Tier = 8

	// Short-hand aliases
	T4 = Tier4
	T5 = Tier5
	T6 = Tier6
	T7 = Tier7
	T8 = Tier8
)

// City represents the major royal trade hubs and Caerleon.
type City string

const (
	CityThetford     City = "Thetford"
	CityFortSterling City = "Fort Sterling"
	CityLymhurst     City = "Lymhurst"
	CityBridgewatch  City = "Bridgewatch"
	CityMartlock     City = "Martlock"
	CityCaerleon     City = "Caerleon"

	// Short-hand aliases
	Thetford     = CityThetford
	FortSterling = CityFortSterling
	Lymhurst     = CityLymhurst
	Bridgewatch  = CityBridgewatch
	Martlock     = CityMartlock
	Caerleon     = CityCaerleon
)

var (
	// ErrStoneCannotBeEnchanted is returned when an enchanted recipe is queried for Stone.
	ErrStoneCannotBeEnchanted = errors.New("stone blocks do not support enchantments")
	// ErrInvalidCommodity is returned when an unsupported commodity type is specified.
	ErrInvalidCommodity = errors.New("invalid commodity type")
	// ErrInvalidTier is returned when a tier outside 4-8 is requested.
	ErrInvalidTier = errors.New("invalid tier: must be between 4 and 8")
	// ErrInvalidEnchantment is returned when an enchantment outside 0-4 is requested.
	ErrInvalidEnchantment = errors.New("invalid enchantment: must be between 0 and 4")
	// ErrRecipeNotFound is returned when no recipe matches the query.
	ErrRecipeNotFound = errors.New("refining recipe not found")
)

// RefiningRecipe specifies the input, output, Item Value, and bonus city for refining.
type RefiningRecipe struct {
	ResultItemID      string        `json:"result_item_id"`
	Commodity         CommodityType `json:"commodity"`
	Tier              int           `json:"tier"`
	Enchantment       int           `json:"enchantment"`
	RawItemID         string        `json:"raw_item_id"`
	RawAmount         int           `json:"raw_amount"`
	PrevRefinedItemID string        `json:"prev_refined_item_id"`
	PrevRefinedAmount int           `json:"prev_refined_amount"`
	ItemValue         int           `json:"item_value"`
	BonusCity         City          `json:"bonus_city"`
}

type commodityMeta struct {
	rawPrefix     string
	refinedPrefix string
	bonusCity     City
}

var commoditySpecs = map[CommodityType]commodityMeta{
	CommodityWood:  {rawPrefix: "WOOD", refinedPrefix: "PLANKS", bonusCity: CityFortSterling},
	CommodityOre:   {rawPrefix: "ORE", refinedPrefix: "METALBAR", bonusCity: CityThetford},
	CommodityHide:  {rawPrefix: "HIDE", refinedPrefix: "LEATHER", bonusCity: CityMartlock},
	CommodityFiber: {rawPrefix: "FIBER", refinedPrefix: "CLOTH", bonusCity: CityLymhurst},
	CommodityStone: {rawPrefix: "ROCK", refinedPrefix: "STONEBLOCK", bonusCity: CityBridgewatch},
}

type recipeKey struct {
	commodity CommodityType
	tier      int
	enchant   int
}

var (
	recipesByKey      = make(map[recipeKey]*RefiningRecipe)
	recipesByResultID = make(map[string]*RefiningRecipe)
	allRecipesList    []*RefiningRecipe
)

func init() {
	tiers := []int{4, 5, 6, 7, 8}
	commodities := []CommodityType{
		CommodityWood,
		CommodityOre,
		CommodityHide,
		CommodityFiber,
		CommodityStone,
	}

	rawAmounts := map[int]int{
		4: 2,
		5: 3,
		6: 4,
		7: 5,
		8: 5,
	}

	for _, commodity := range commodities {
		meta := commoditySpecs[commodity]
		for _, tier := range tiers {
			maxEnchant := 4
			if commodity == CommodityStone {
				maxEnchant = 0 // Stone only supports enchantment 0
			}

			for enchant := 0; enchant <= maxEnchant; enchant++ {
				// Raw Item ID
				var rawID string
				if enchant == 0 {
					rawID = fmt.Sprintf("T%d_%s", tier, meta.rawPrefix)
				} else {
					rawID = fmt.Sprintf("T%d_%s_LEVEL%d@%d", tier, meta.rawPrefix, enchant, enchant)
				}

				// Result Item ID
				var resultID string
				if enchant == 0 {
					resultID = fmt.Sprintf("T%d_%s", tier, meta.refinedPrefix)
				} else {
					resultID = fmt.Sprintf("T%d_%s_LEVEL%d@%d", tier, meta.refinedPrefix, enchant, enchant)
				}

				// Previous Refined Item ID
				var prevRefinedID string
				if tier == 4 {
					// T3 refined item has no enchantment even for enchanted T4 refining
					prevRefinedID = fmt.Sprintf("T3_%s", meta.refinedPrefix)
				} else {
					prevTier := tier - 1
					if enchant == 0 {
						prevRefinedID = fmt.Sprintf("T%d_%s", prevTier, meta.refinedPrefix)
					} else {
						prevRefinedID = fmt.Sprintf("T%d_%s_LEVEL%d@%d", prevTier, meta.refinedPrefix, enchant, enchant)
					}
				}

				// Item Value calculation:
				// Base IV: T3=8, T4=16, T5=32, T6=64, T7=128, T8=256 -> 1 << tier
				// Enchantment multiplier: 0=1x, 1=2x, 2=4x, 3=8x, 4=16x -> 1 << enchant
				itemValue := 1 << (tier + enchant)

				recipe := &RefiningRecipe{
					ResultItemID:      resultID,
					Commodity:         commodity,
					Tier:              tier,
					Enchantment:       enchant,
					RawItemID:         rawID,
					RawAmount:         rawAmounts[tier],
					PrevRefinedItemID: prevRefinedID,
					PrevRefinedAmount: 1,
					ItemValue:         itemValue,
					BonusCity:         meta.bonusCity,
				}

				key := recipeKey{commodity: commodity, tier: tier, enchant: enchant}
				recipesByKey[key] = recipe
				recipesByResultID[resultID] = recipe
				allRecipesList = append(allRecipesList, recipe)
			}
		}
	}
}

// GetBonusCity returns the biome bonus city for the given commodity.
func GetBonusCity(commodity CommodityType) City {
	meta, ok := commoditySpecs[commodity]
	if !ok {
		return ""
	}
	return meta.bonusCity
}

// GetRefiningRecipe returns the recipe for the specified commodity, tier (4-8), and enchantment (0-4).
// Returns ErrStoneCannotBeEnchanted if Stone is requested with enchantment > 0.
func GetRefiningRecipe(commodity CommodityType, tier int, enchant int) (*RefiningRecipe, error) {
	if _, ok := commoditySpecs[commodity]; !ok {
		return nil, ErrInvalidCommodity
	}
	if tier < 4 || tier > 8 {
		return nil, ErrInvalidTier
	}
	if enchant < 0 || enchant > 4 {
		return nil, ErrInvalidEnchantment
	}
	if commodity == CommodityStone && enchant > 0 {
		return nil, ErrStoneCannotBeEnchanted
	}

	recipe, ok := recipesByKey[recipeKey{commodity: commodity, tier: tier, enchant: enchant}]
	if !ok {
		return nil, ErrRecipeNotFound
	}
	return recipe, nil
}

// GetRecipeByResultID finds a refining recipe by its output item ID (e.g. "T5_METALBAR", "T6_PLANKS_LEVEL1@1").
// If the item ID is an enchanted stoneblock (which cannot exist), ErrStoneCannotBeEnchanted is returned.
func GetRecipeByResultID(itemID string) (*RefiningRecipe, error) {
	if strings.Contains(itemID, "STONEBLOCK") && (strings.Contains(itemID, "@") || strings.Contains(itemID, "LEVEL")) {
		return nil, ErrStoneCannotBeEnchanted
	}

	recipe, ok := recipesByResultID[itemID]
	if !ok {
		return nil, ErrRecipeNotFound
	}
	return recipe, nil
}

// GetAllRecipes returns a slice of all 105 official refining recipes.
func GetAllRecipes() []*RefiningRecipe {
	dst := make([]*RefiningRecipe, len(allRecipesList))
	copy(dst, allRecipesList)
	return dst
}
