package main

import "sort"

func shopRotationWeight(rarity string) int {
	weight := catalog.Shop.RarityRotationWeights[rarity]
	if weight <= 0 {
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
