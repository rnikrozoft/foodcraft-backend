package main

import (
	"fmt"
	"sort"
	"time"
)

func isShopActive(state *PlayerState, id string) bool {
	if len(state.ShopActiveIDs) == 0 {
		return false
	}
	return containsID(state.ShopActiveIDs, id)
}

func rollShopRotation(state *PlayerState) {
	locked := make([]shopCatalogEntry, 0)
	for _, offer := range buildShopIngredientCatalog() {
		if isIngredientUnlocked(state, offer.ID) {
			continue
		}
		locked = append(locked, offer)
	}
	count := catalog.Shop.RotationCount
	if count <= 0 {
		count = 8
	}
	state.ShopActiveIDs = weightedPickShopOffers(locked, count)
}

type shopCatalogEntry struct {
	ID          string
	Title       string
	Emoji       string
	Rarity      string
	RarityLabel string
	Cost        int
}

func ensureShopState(state *PlayerState) {
	if state.UnlockedIngredients == nil {
		state.UnlockedIngredients = []string{}
	}
	if state.ShopActiveIDs == nil {
		state.ShopActiveIDs = []string{}
	}
	migrateLegacyShopState(state)
	maybeAutoShopReset(state)
	if len(state.ShopActiveIDs) == 0 {
		rollShopRotation(state)
		state.ShopLastAutoResetUnix = todayMidnightUTC().Unix()
	}
}

func migrateLegacyShopState(state *PlayerState) {
	if state.ShopLastAutoResetUnix > 0 {
		return
	}
	// Players with an existing rotation keep it until the next UTC midnight.
	if len(state.ShopActiveIDs) > 0 {
		state.ShopLastAutoResetUnix = todayMidnightUTC().Unix()
	}
}

func maybeAutoShopReset(state *PlayerState) {
	todayMid := todayMidnightUTC().Unix()
	if state.ShopLastAutoResetUnix >= todayMid {
		return
	}
	rollShopRotation(state)
	state.ShopLastAutoResetUnix = todayMid
}

func nextShopResetSec() int64 {
	nextMidnight := todayMidnightUTC().Add(24 * time.Hour)
	remaining := nextMidnight.Unix() - time.Now().UTC().Unix()
	if remaining < 0 {
		return 0
	}
	return remaining
}

func isShopIngredientBuyable(state *PlayerState, id string) bool {
	if isIngredientUnlocked(state, id) {
		return false
	}
	if !isShopActive(state, id) {
		return false
	}
	ensureShopState(state)
	return true
}

func isIngredientUnlocked(state *PlayerState, id string) bool {
	if _, ok := catalog.Items[id]; !ok {
		return false
	}
	if catalog.IsStarter(id) {
		return true
	}
	return containsID(state.UnlockedIngredients, id)
}

func buildShopIngredientCatalog() []shopCatalogEntry {
	rarityOrder := []string{"common", "uncommon", "rare", "epic", "legendary"}
	cfg := catalog.Shop

	shopable := make([]ItemDef, 0)
	for _, item := range catalog.Items {
		if item.Category != "ingredient" || item.Tier != 0 || item.Starter {
			continue
		}
		shopable = append(shopable, item)
	}
	sort.Slice(shopable, func(i, j int) bool {
		return shopable[i].NameTH < shopable[j].NameTH
	})

	offers := make([]shopCatalogEntry, 0, len(shopable))
	for i, item := range shopable {
		bucket := int(float64(i) / float64(len(shopable)) * 5.0)
		if bucket > 4 {
			bucket = 4
		}
		rarity := rarityOrder[bucket]
		cost := cfg.RarityCosts[rarity]
		if cost <= 0 {
			continue
		}
		label := cfg.RarityLabels[rarity]
		if label == "" {
			label = rarity
		}
		offers = append(offers, shopCatalogEntry{
			ID:          item.ID,
			Title:       item.NameTH,
			Emoji:       item.Emoji,
			Rarity:      rarity,
			RarityLabel: label,
			Cost:        cost,
		})
	}
	return offers
}

func shopRarityForID(id string) string {
	for _, offer := range buildShopIngredientCatalog() {
		if offer.ID == id {
			return offer.Rarity
		}
	}
	return "common"
}

func buildShopStateResponse(state *PlayerState, wallet WalletResponse) ShopStateResponse {
	ensureShopState(state)
	offers := make([]ShopIngredientOffer, 0)
	for _, raw := range buildShopIngredientCatalog() {
		if !isShopActive(state, raw.ID) {
			continue
		}
		unlocked := isIngredientUnlocked(state, raw.ID)
		buyable := !unlocked && isShopIngredientBuyable(state, raw.ID)
		offers = append(offers, ShopIngredientOffer{
			ID:          raw.ID,
			Title:       raw.Title,
			Emoji:       raw.Emoji,
			Rarity:      raw.Rarity,
			RarityLabel: raw.RarityLabel,
			Cost:        raw.Cost,
			Unlocked:    unlocked,
			Buyable:     buyable,
		})
	}
	unlockedCopy := append([]string(nil), state.UnlockedIngredients...)
	if unlockedCopy == nil {
		unlockedCopy = []string{}
	}
	sort.Strings(unlockedCopy)
	activeCopy := append([]string(nil), state.ShopActiveIDs...)
	if activeCopy == nil {
		activeCopy = []string{}
	}
	sort.Strings(activeCopy)
	return ShopStateResponse{
		Coins:                 wallet.Coins,
		Stars:                 wallet.Stars,
		ShopLastAutoResetUnix: state.ShopLastAutoResetUnix,
		NextShopResetSec:      nextShopResetSec(),
		ShopActiveIDs:         activeCopy,
		UnlockedIngredients:   unlockedCopy,
		DiscoveredAt:          state.DiscoveredAt,
		CraftableCounts:       craftableRemainingCounts(state),
		Offers:                offers,
	}
}

func ensureShopActiveID(state *PlayerState, id string) {
	if containsID(state.ShopActiveIDs, id) {
		return
	}
	state.ShopActiveIDs = append(state.ShopActiveIDs, id)
	sort.Strings(state.ShopActiveIDs)
}

func unlockIngredient(state *PlayerState, id string) bool {
	item, ok := catalog.Items[id]
	if !ok || item.Category != "ingredient" {
		return false
	}
	if isIngredientUnlocked(state, id) {
		return true
	}
	state.UnlockedIngredients = uniqueAppend(state.UnlockedIngredients, id)
	recordDiscoveredAt(state, id)
	return true
}

func purchaseShopIngredient(state *PlayerState, id string, coins int) error {
	ensureShopState(state)
	if isIngredientUnlocked(state, id) {
		return fmt.Errorf("already unlocked")
	}
	if !isShopActive(state, id) {
		return fmt.Errorf("ingredient not in shop")
	}
	if !isShopIngredientBuyable(state, id) {
		return fmt.Errorf("ingredient not available")
	}
	cost := 0
	for _, offer := range buildShopIngredientCatalog() {
		if offer.ID == id {
			cost = offer.Cost
			break
		}
	}
	if cost <= 0 {
		return fmt.Errorf("unknown ingredient")
	}
	if coins < cost {
		return fmt.Errorf("insufficient coins")
	}
	if !unlockIngredient(state, id) {
		return fmt.Errorf("unlock failed")
	}
	ensureShopActiveID(state, id)
	return nil
}

func shopIngredientCost(id string) int {
	for _, offer := range buildShopIngredientCatalog() {
		if offer.ID == id {
			return offer.Cost
		}
	}
	return 0
}
