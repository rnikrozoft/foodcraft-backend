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
}

var catalog *Catalog

func loadCatalog() (*Catalog, error) {
	var raw struct {
		Items   []ItemDef   `json:"items"`
		Recipes []RecipeDef `json:"recipes"`
	}
	if err := json.Unmarshal(gameDataJSON, &raw); err != nil {
		return nil, err
	}

	c := &Catalog{
		Items:           make(map[string]ItemDef, len(raw.Items)),
		RecipeByKey:     make(map[string]string, len(raw.Recipes)),
		ItemPoints:      make(map[string]int, len(raw.Items)),
		DiscoverableIDs: make(map[string]bool),
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

func (c *Catalog) HasIngredient(discovered map[string]bool, id string) bool {
	if c.IsTierZero(id) {
		return true
	}
	return discovered[id]
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

func discoveredSet(list []string) map[string]bool {
	out := make(map[string]bool, len(list))
	for _, id := range list {
		out[id] = true
	}
	return out
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
