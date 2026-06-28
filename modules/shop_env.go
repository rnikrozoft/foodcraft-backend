package main

import (
	"os"
	"strconv"
	"strings"
)

// Shop tunables are loaded from environment variables (see .env.example).
// Nakama Console "Server Configuration" shows local.yml only (logger, session, etc.)
// — not game shop settings. Use .env + docker compose env_file for shop balance.
func applyShopEnvOverrides(cfg ShopConfig) ShopConfig {
	cfg = normalizeShopConfig(cfg)

	if v := strings.TrimSpace(os.Getenv("SHOP_ROTATION_COUNT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.RotationCount = n
		}
	}

	weights := map[string]string{
		"common":    "SHOP_WEIGHT_COMMON",
		"uncommon":  "SHOP_WEIGHT_UNCOMMON",
		"rare":      "SHOP_WEIGHT_RARE",
		"epic":      "SHOP_WEIGHT_EPIC",
		"legendary": "SHOP_WEIGHT_LEGENDARY",
	}
	for rarity, envKey := range weights {
		if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				cfg.RarityRotationWeights[rarity] = n
			}
		}
	}

	costs := map[string]string{
		"common":    "SHOP_COST_COMMON",
		"uncommon":  "SHOP_COST_UNCOMMON",
		"rare":      "SHOP_COST_RARE",
		"epic":      "SHOP_COST_EPIC",
		"legendary": "SHOP_COST_LEGENDARY",
	}
	for rarity, envKey := range costs {
		if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				cfg.RarityCosts[rarity] = n
			}
		}
	}

	return cfg
}
