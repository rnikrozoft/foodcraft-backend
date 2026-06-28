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
	ItemPoints      map[string]int
	DiscoverableIDs map[string]bool
	StarterIDs      map[string]bool
	Shop            ShopConfig
}

var catalog *Catalog

func loadCatalog() (*Catalog, error) {
	var raw struct {
		Items        []ItemDef   `json:"items"`
		Recipes      []RecipeDef `json:"recipes"`
		Shop         ShopConfig  `json:"shop"`
		StarterItems []string    `json:"starter_items"`
	}
	if err := json.Unmarshal(gameDataJSON, &raw); err != nil {
		return nil, err
	}

	c := &Catalog{
		Items:           make(map[string]ItemDef, len(raw.Items)),
		RecipeByKey:     make(map[string]string, len(raw.Recipes)),
		ItemPoints:      make(map[string]int, len(raw.Items)),
		DiscoverableIDs: make(map[string]bool),
		StarterIDs:      make(map[string]bool),
		Shop:            normalizeShopConfig(raw.Shop),
	}

	outDegree := make(map[string]int)
	for _, recipe := range raw.Recipes {
		c.RecipeByKey[recipeKey(recipe.A, recipe.B)] = recipe.Result
		outDegree[recipe.A]++
		outDegree[recipe.B]++
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
	base := tier * 10
	bonus := branchFactor * 2
	if bonus > 30 {
		bonus = 30
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
