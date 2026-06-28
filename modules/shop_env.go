package main

import (
	"fmt"
	"strings"
)

var shopRarities = []string{"common", "uncommon", "rare", "epic", "legendary"}

// Shop tunables are loaded from required environment variables (see .env.example).
// Rarity display labels come from game_data.json (shop.rarity_labels) only.
func loadShopConfigFromEnv(jsonLabels map[string]string) (ShopConfig, error) {
	if len(jsonLabels) == 0 {
		return ShopConfig{}, fmt.Errorf("missing shop.rarity_labels in game_data.json")
	}
	for _, rarity := range shopRarities {
		if strings.TrimSpace(jsonLabels[rarity]) == "" {
			return ShopConfig{}, fmt.Errorf("missing shop.rarity_labels.%s in game_data.json", rarity)
		}
	}

	rotation, err := envRequiredIntPositive("SHOP_ROTATION_COUNT")
	if err != nil {
		return ShopConfig{}, err
	}

	weights := make(map[string]int, len(shopRarities))
	costs := make(map[string]int, len(shopRarities))
	labels := make(map[string]string, len(shopRarities))
	for _, rarity := range shopRarities {
		envWeight := "SHOP_WEIGHT_" + shopEnvSuffix(rarity)
		if weights[rarity], err = envRequiredIntPositive(envWeight); err != nil {
			return ShopConfig{}, err
		}
		envCost := "SHOP_COST_" + shopEnvSuffix(rarity)
		if costs[rarity], err = envRequiredIntPositive(envCost); err != nil {
			return ShopConfig{}, err
		}
		labels[rarity] = jsonLabels[rarity]
	}

	return ShopConfig{
		RarityRotationWeights: weights,
		RarityCosts:           costs,
		RarityLabels:          labels,
		RotationCount:         rotation,
	}, nil
}

func shopEnvSuffix(rarity string) string {
	switch rarity {
	case "common":
		return "COMMON"
	case "uncommon":
		return "UNCOMMON"
	case "rare":
		return "RARE"
	case "epic":
		return "EPIC"
	case "legendary":
		return "LEGENDARY"
	default:
		return rarity
	}
}
