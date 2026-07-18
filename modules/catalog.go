package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed data/game_data.json
var gameDataJSON []byte

type Catalog struct {
	Items           map[string]ItemDef
	RecipeByKey     map[string]string
	// ItemRecipeResults maps an ingredient id -> the distinct result ids of
	// every recipe it participates in (as either side of the pair). Used to
	// compute "how many more things can I still craft with this?" without
	// ever exposing the actual A+B pairs to the client.
	ItemRecipeResults map[string][]string
	ItemPoints      map[string]int
	DiscoverableIDs map[string]bool
	StarterIDs      map[string]bool
	Shop            ShopConfig
}

var catalog *Catalog

func loadCatalog() (*Catalog, error) {
	if err := loadGameBalance(); err != nil {
		return nil, fmt.Errorf("game balance: %w", err)
	}

	var raw struct {
		Items        []ItemDef   `json:"items"`
		Recipes      []RecipeDef `json:"recipes"`
		Shop         ShopConfig  `json:"shop"`
		StarterItems []string    `json:"starter_items"`
	}
	if err := json.Unmarshal(gameDataJSON, &raw); err != nil {
		return nil, err
	}

	shop, err := loadShopConfigFromEnv(raw.Shop.RarityLabels)
	if err != nil {
		return nil, fmt.Errorf("shop config: %w", err)
	}

	c := &Catalog{
		Items:           make(map[string]ItemDef, len(raw.Items)),
		RecipeByKey:     make(map[string]string, len(raw.Recipes)),
		ItemPoints:      make(map[string]int, len(raw.Items)),
		DiscoverableIDs: make(map[string]bool),
		StarterIDs:      make(map[string]bool),
		Shop:            shop,
	}

	outDegree := make(map[string]int)
	itemResultSets := make(map[string]map[string]bool)
	for _, recipe := range raw.Recipes {
		c.RecipeByKey[recipeKey(recipe.A, recipe.B)] = recipe.Result
		outDegree[recipe.A]++
		outDegree[recipe.B]++
		for _, ingredientID := range []string{recipe.A, recipe.B} {
			if itemResultSets[ingredientID] == nil {
				itemResultSets[ingredientID] = make(map[string]bool)
			}
			itemResultSets[ingredientID][recipe.Result] = true
		}
	}
	c.ItemRecipeResults = make(map[string][]string, len(itemResultSets))
	for ingredientID, resultSet := range itemResultSets {
		results := make([]string, 0, len(resultSet))
		for resultID := range resultSet {
			results = append(results, resultID)
		}
		sort.Strings(results)
		c.ItemRecipeResults[ingredientID] = results
	}

	for _, item := range raw.Items {
		c.Items[item.ID] = item
		if item.Tier >= 1 {
			c.DiscoverableIDs[item.ID] = true
			c.ItemPoints[item.ID] = discoveryPoint(item.Tier, outDegree[item.ID])
		}
	}

	for _, id := range raw.StarterItems {
		if id != "" {
			c.StarterIDs[id] = true
		}
	}
	if len(c.StarterIDs) == 0 {
		for _, item := range raw.Items {
			if item.Starter {
				c.StarterIDs[item.ID] = true
			}
		}
	}

	catalog = c
	return c, nil
}

func recipeKey(a, b string) string {
	pair := []string{a, b}
	sort.Strings(pair)
	return pair[0] + "|" + pair[1]
}

func discoveryPoint(tier int, branchFactor int) int {
	if tier < 1 {
		return 0
	}
	base := tier * gameBalance.DiscoveryPointsPerTier
	bonus := branchFactor * gameBalance.DiscoveryBranchBonus
	if bonus > gameBalance.DiscoveryBranchBonusMax {
		bonus = gameBalance.DiscoveryBranchBonusMax
	}
	return base + bonus
}

func (c *Catalog) LookupRecipe(a, b string) (string, bool) {
	result, ok := c.RecipeByKey[recipeKey(a, b)]
	return result, ok
}

func (c *Catalog) IsTierZero(id string) bool {
	item, ok := c.Items[id]
	return ok && item.Tier == 0
}

func (c *Catalog) IsDiscoverable(id string) bool {
	return c.DiscoverableIDs[id]
}

func (c *Catalog) PointsFor(id string) int {
	return c.ItemPoints[id]
}

func (c *Catalog) CraftResultKind(id string) string {
	item, ok := c.Items[id]
	if !ok {
		return ""
	}
	if item.Tier == 0 && item.Category == "ingredient" {
		return "ingredient"
	}
	if c.IsDiscoverable(id) {
		return "menu"
	}
	return ""
}

func (c *Catalog) ValidateCraftResult(itemID, a, b string) error {
	result, ok := c.LookupRecipe(a, b)
	if !ok || result != itemID {
		return fmt.Errorf("invalid recipe combination")
	}
	if c.CraftResultKind(itemID) == "" {
		return fmt.Errorf("item is not a valid craft result")
	}
	return nil
}

func (c *Catalog) ValidateDiscovery(itemID, a, b string) error {
	if !c.IsDiscoverable(itemID) {
		return fmt.Errorf("item is not discoverable: %s", itemID)
	}
	result, ok := c.LookupRecipe(a, b)
	if !ok || result != itemID {
		return fmt.Errorf("invalid recipe combination")
	}
	return nil
}

func (c *Catalog) IsStarter(id string) bool {
	return c.StarterIDs[id]
}

func (c *Catalog) gameConfigResponse() GameConfigResponse {
	items := make([]ItemDef, 0, len(c.Items))
	for _, item := range c.Items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	starterItems := make([]string, 0, len(c.StarterIDs))
	for id := range c.StarterIDs {
		starterItems = append(starterItems, id)
	}
	sort.Strings(starterItems)

	return GameConfigResponse{
		Version:           1,
		Items:             items,
		StarterItems:      starterItems,
		DiscoverableTotal: len(c.DiscoverableIDs),
		DailyRewardCoins:  gameBalance.DailyRewardCoins,
		Monetization: MonetizationConfig{
			AdRewardCoins:    gameBalance.AdRewardCoins,
			StarterPackCoins: gameBalance.StarterPackCoins,
		},
		Shop:              c.Shop,
	}
}

func (c *Catalog) HasIngredient(discovered map[string]bool, unlocked map[string]bool, id string) bool {
	item, ok := c.Items[id]
	if !ok {
		return false
	}
	if item.Tier == 0 {
		return c.IsStarter(id) || unlocked[id]
	}
	return discovered[id]
}

// RemainingCraftableCount returns how many distinct recipe results this item
// participates in (as either ingredient) that aren't in `discovered` yet.
// It's a completion/exploration hint ("N more things possible with this"),
// not a strict "you have the other ingredient too" check, and it never
// reveals the actual A+B pairs or result ids to the caller.
func (c *Catalog) RemainingCraftableCount(itemID string, discovered map[string]bool) int {
	count := 0
	for _, resultID := range c.ItemRecipeResults[itemID] {
		if !discovered[resultID] {
			count++
		}
	}
	return count
}

func (c *Catalog) TierFor(itemID string) int {
	item, ok := c.Items[itemID]
	if !ok || item.Tier < 1 {
		return 1
	}
	return item.Tier
}

func containsID(list []string, id string) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

func uniqueAppend(list []string, id string) []string {
	if containsID(list, id) {
		return list
	}
	return append(list, id)
}

func copyStringSlice(list []string) []string {
	if list == nil {
		return []string{}
	}
	out := append([]string(nil), list...)
	sort.Strings(out)
	return out
}

func discoveredSet(list []string) map[string]bool {
	out := make(map[string]bool, len(list))
	for _, id := range list {
		out[id] = true
	}
	return out
}

func unlockedIngredientSet(list []string) map[string]bool {
	return discoveredSet(list)
}

func totalPoints(c *Catalog, discovered []string) int {
	total := 0
	for _, id := range discovered {
		total += c.PointsFor(id)
	}
	return total
}

func normalizePair(from []string) (string, string, error) {
	if len(from) != 2 {
		return "", "", fmt.Errorf("from must contain exactly 2 ingredients")
	}
	a := strings.TrimSpace(from[0])
	b := strings.TrimSpace(from[1])
	if a == "" || b == "" {
		return "", "", fmt.Errorf("invalid ingredient ids")
	}
	return a, b, nil
}
