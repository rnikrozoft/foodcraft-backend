package main

import "sort"

func defaultShopConfig() ShopConfig {
	return ShopConfig{
		RarityRotationWeights: map[string]int{
			"common":    50,
			"uncommon":  25,
			"rare":      15,
			"epic":      7,
			"legendary": 3,
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
	if len(cfg.RarityRotationWeights) == 0 {
		cfg.RarityRotationWeights = def.RarityRotationWeights
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

func shopRotationWeight(rarity string) int {
	weight := catalog.Shop.RarityRotationWeights[rarity]
	if weight <= 0 {
		def := defaultShopConfig().RarityRotationWeights[rarity]
		if def > 0 {
			return def
		}
		return 1
	}
	return weight
}

func weightedPickShopOffers(offers []shopCatalogEntry, count int) []string {
	if count <= 0 || len(offers) == 0 {
		return []string{}
	}
	if count >= len(offers) {
		ids := make([]string, 0, len(offers))
		for _, offer := range offers {
			ids = append(ids, offer.ID)
		}
		sort.Strings(ids)
		return ids
	}

	pool := append([]shopCatalogEntry(nil), offers...)
	ids := make([]string, 0, count)
	for len(ids) < count && len(pool) > 0 {
		totalWeight := 0
		weights := make([]int, len(pool))
		for i, offer := range pool {
			weights[i] = shopRotationWeight(offer.Rarity)
			totalWeight += weights[i]
		}
		if totalWeight <= 0 {
			totalWeight = len(pool)
			for i := range weights {
				weights[i] = 1
			}
		}

		roll := secureRandInt(totalWeight)
		sum := 0
		pick := 0
		for i, weight := range weights {
			sum += weight
			if roll < sum {
				pick = i
				break
			}
		}
		ids = append(ids, pool[pick].ID)
		pool = append(pool[:pick], pool[pick+1:]...)
	}
	sort.Strings(ids)
	return ids
}
