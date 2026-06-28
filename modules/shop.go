package main

import (
	"fmt"
	"sort"
	"time"
)

func isShopActive(state *PlayerState, id string) bool {
	if len(state.ShopActiveIDs) == 0 {
		return true
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
	if count <= 0 || count >= len(locked) {
		state.ShopActiveIDs = make([]string, 0, len(locked))
		for _, offer := range locked {
			state.ShopActiveIDs = append(state.ShopActiveIDs, offer.ID)
		}
		sort.Strings(state.ShopActiveIDs)
		return
	}
	for i := len(locked) - 1; i > 0; i-- {
		j := secureRandInt(i + 1)
		locked[i], locked[j] = locked[j], locked[i]
	}
	state.ShopActiveIDs = make([]string, 0, count)
	for i := 0; i < count; i++ {
		state.ShopActiveIDs = append(state.ShopActiveIDs, locked[i].ID)
	}
	sort.Strings(state.ShopActiveIDs)
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
	if state.ShopItemExpiresAt == nil {
		state.ShopItemExpiresAt = map[string]int64{}
	}
	if state.ShopActiveIDs == nil {
		state.ShopActiveIDs = []string{}
	}
	normalizeShopCycle(state)
	if len(state.ShopActiveIDs) == 0 {
		rollShopRotation(state)
	}
	ensureShopItemWindows(state)
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
			cost = 150
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

func shopCooldownDays(rarity string) int {
	days := catalog.Shop.RarityCooldownDays[rarity]
	if days <= 0 {
		days = 3
	}
	return days
}

func normalizeShopCycle(state *PlayerState) {
	now := time.Now().UTC().Unix()
	cycleSec := int64(catalog.Shop.ResetCycleSec)
	if cycleSec <= 0 {
		cycleSec = 86400
	}
	if state.ShopCycleStart <= 0 {
		state.ShopCycleStart = now
		state.ShopManualResetCount = 0
		return
	}
	if now-state.ShopCycleStart >= cycleSec {
		state.ShopCycleStart = now
		state.ShopManualResetCount = 0
	}
}

func ensureShopItemWindows(state *PlayerState) {
	now := time.Now().UTC().Unix()
	for _, id := range state.ShopActiveIDs {
		if isIngredientUnlocked(state, id) {
			continue
		}
		if _, ok := state.ShopItemExpiresAt[id]; ok {
			continue
		}
		days := shopCooldownDays(shopRarityForID(id))
		state.ShopItemExpiresAt[id] = now + int64(days*86400)
	}
}

func isShopIngredientBuyable(state *PlayerState, id string) bool {
	if isIngredientUnlocked(state, id) {
		return false
	}
	if !isShopActive(state, id) {
		return false
	}
	ensureShopState(state)
	expires := state.ShopItemExpiresAt[id]
	now := time.Now().UTC().Unix()
	if now < expires {
		return true
	}
	days := shopCooldownDays(shopRarityForID(id))
	restockAt := expires + int64(days*86400)
	if now >= restockAt {
		state.ShopItemExpiresAt[id] = now + int64(days*86400)
		return true
	}
	return false
}

func shopIngredientCooldownSec(state *PlayerState, id, rarity string) int64 {
	if isIngredientUnlocked(state, id) {
		return 0
	}
	ensureShopState(state)
	expires := state.ShopItemExpiresAt[id]
	now := time.Now().UTC().Unix()
	if now < expires {
		return 0
	}
	if rarity == "" {
		rarity = shopRarityForID(id)
	}
	days := shopCooldownDays(rarity)
	restockAt := expires + int64(days*86400)
	remaining := restockAt - now
	if remaining < 0 {
		return 0
	}
	return remaining
}

func shopIngredientWindowRemainingSec(state *PlayerState, id string) int64 {
	if isIngredientUnlocked(state, id) {
		return 0
	}
	if !isShopIngredientBuyable(state, id) {
		return 0
	}
	expires := state.ShopItemExpiresAt[id]
	remaining := expires - time.Now().UTC().Unix()
	if remaining < 0 {
		return 0
	}
	return remaining
}

func shopResetCoinCost(state *PlayerState) int {
	info := getShopResetInfo(state)
	return info.CoinCost
}

func getShopResetInfo(state *PlayerState) ShopResetInfo {
	normalizeShopCycle(state)
	cfg := catalog.Shop
	priceType := "free"
	coinCost := 0
	if state.ShopManualResetCount >= 1 {
		if state.ShopManualResetCount < 1+cfg.ResetMidCount {
			priceType = "mid"
		} else {
			priceType = "high"
		}
	}
	priceLabel := "ฟรี"
	switch priceType {
	case "mid":
		coinCost = cfg.ResetPrices.Mid
		priceLabel = fmt.Sprintf("%d", coinCost)
	case "high":
		coinCost = cfg.ResetPrices.High
		priceLabel = fmt.Sprintf("%d", coinCost)
	}
	now := time.Now().UTC().Unix()
	cycleSec := int64(cfg.ResetCycleSec)
	if cycleSec <= 0 {
		cycleSec = 86400
	}
	nextFree := cycleSec - (now - state.ShopCycleStart)
	if nextFree < 0 {
		nextFree = 0
	}
	return ShopResetInfo{
		PriceType:        priceType,
		PriceLabel:       priceLabel,
		CoinCost:         coinCost,
		ResetsUsed:       state.ShopManualResetCount,
		NextFreeResetSec: nextFree,
	}
}

func buildShopStateResponse(state *PlayerState) ShopStateResponse {
	ensureShopState(state)
	offers := make([]ShopIngredientOffer, 0)
	for _, raw := range buildShopIngredientCatalog() {
		if !isShopActive(state, raw.ID) {
			continue
		}
		unlocked := isIngredientUnlocked(state, raw.ID)
		buyable := !unlocked && isShopIngredientBuyable(state, raw.ID)
		cooldown := int64(0)
		if !unlocked {
			cooldown = shopIngredientCooldownSec(state, raw.ID, raw.Rarity)
		}
		windowRemaining := int64(0)
		if buyable {
			windowRemaining = shopIngredientWindowRemainingSec(state, raw.ID)
		}
		offers = append(offers, ShopIngredientOffer{
			ID:                 raw.ID,
			Title:              raw.Title,
			Emoji:              raw.Emoji,
			Rarity:             raw.Rarity,
			RarityLabel:        raw.RarityLabel,
			Cost:               raw.Cost,
			Unlocked:           unlocked,
			Buyable:            buyable,
			CooldownSec:        cooldown,
			WindowRemainingSec: windowRemaining,
		})
	}
	expiresCopy := make(map[string]int64, len(state.ShopItemExpiresAt))
	for id, ts := range state.ShopItemExpiresAt {
		expiresCopy[id] = ts
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
		Coins:                state.Coins,
		ShopCycleStart:       state.ShopCycleStart,
		ShopManualResetCount: state.ShopManualResetCount,
		ShopItemExpiresAt:    expiresCopy,
		ShopActiveIDs:        activeCopy,
		UnlockedIngredients:  unlockedCopy,
		ResetInfo:            getShopResetInfo(state),
		Offers:               offers,
	}
}

func performShopReset(state *PlayerState) {
	normalizeShopCycle(state)
	state.ShopManualResetCount++
	rollShopRotation(state)
	refreshShopItemWindows(state)
}

func refreshShopItemWindows(state *PlayerState) {
	now := time.Now().UTC().Unix()
	for _, id := range state.ShopActiveIDs {
		if isIngredientUnlocked(state, id) {
			continue
		}
		days := shopCooldownDays(shopRarityForID(id))
		state.ShopItemExpiresAt[id] = now + int64(days*86400)
	}
}

func markShopItemPurchased(state *PlayerState, id string) {
	state.ShopItemExpiresAt[id] = time.Now().UTC().Unix()
	ensureShopActiveID(state, id)
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
	return true
}

func purchaseShopIngredient(state *PlayerState, id string) error {
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
	if state.Coins < cost {
		return fmt.Errorf("insufficient coins")
	}
	state.Coins -= cost
	if !unlockIngredient(state, id) {
		return fmt.Errorf("unlock failed")
	}
	markShopItemPurchased(state, id)
	return nil
}

func resetShopFree(state *PlayerState) error {
	normalizeShopCycle(state)
	if state.ShopManualResetCount >= 1 {
		return fmt.Errorf("free reset already used")
	}
	performShopReset(state)
	return nil
}

func resetShopPaid(state *PlayerState) error {
	normalizeShopCycle(state)
	cost := shopResetCoinCost(state)
	if cost <= 0 {
		return fmt.Errorf("invalid reset price")
	}
	if state.Coins < cost {
		return fmt.Errorf("insufficient coins")
	}
	state.Coins -= cost
	performShopReset(state)
	return nil
}
