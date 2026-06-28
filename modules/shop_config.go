package main

func defaultShopConfig() ShopConfig {
	return ShopConfig{
		ResetCycleSec: 86400,
		ResetMidCount: 5,
		ResetPrices: ShopResetPrices{
			Mid:      90,
			High:     150,
			Currency: "coins",
		},
		RarityCooldownDays: map[string]int{
			"common":    3,
			"uncommon":  5,
			"rare":      7,
			"epic":      10,
			"legendary": 14,
		},
		RarityCosts: map[string]int{
			"common":    80,
			"uncommon":  150,
			"rare":      280,
			"epic":      450,
			"legendary": 700,
		},
		RarityLabels: map[string]string{
			"common":    "ธรรมดา",
			"uncommon":  "หายาก",
			"rare":      "แรร์",
			"epic":      "เอปิค",
			"legendary": "ตำนาน",
		},
		RotationCount: 8,
	}
}

func normalizeShopConfig(cfg ShopConfig) ShopConfig {
	def := defaultShopConfig()
	if cfg.ResetCycleSec <= 0 {
		cfg.ResetCycleSec = def.ResetCycleSec
	}
	if cfg.ResetMidCount <= 0 {
		cfg.ResetMidCount = def.ResetMidCount
	}
	if cfg.ResetPrices.Mid <= 0 {
		cfg.ResetPrices.Mid = def.ResetPrices.Mid
	}
	if cfg.ResetPrices.High <= 0 {
		cfg.ResetPrices.High = def.ResetPrices.High
	}
	if cfg.ResetPrices.Currency == "" {
		cfg.ResetPrices.Currency = def.ResetPrices.Currency
	}
	if len(cfg.RarityCooldownDays) == 0 {
		cfg.RarityCooldownDays = def.RarityCooldownDays
	}
	if len(cfg.RarityCosts) == 0 {
		cfg.RarityCosts = def.RarityCosts
	}
	if len(cfg.RarityLabels) == 0 {
		cfg.RarityLabels = def.RarityLabels
	}
	if cfg.RotationCount <= 0 {
		cfg.RotationCount = def.RotationCount
	}
	return cfg
}
