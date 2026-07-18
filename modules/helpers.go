package main

import (
	"context"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

func rpcError(message string, code int) error {
	return runtime.NewError(message, code)
}

func mustUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if !ok || userID == "" {
		return "", rpcError("unauthorized", 16)
	}
	return userID, nil
}

func mustUsername(ctx context.Context) string {
	username, _ := ctx.Value(runtime.RUNTIME_CTX_USERNAME).(string)
	return username
}

// recordDiscoveredAt stamps the moment a player first discovers/unlocks an
// item. Idempotent — a second call for an already-recorded id is a no-op, so
// it's safe to call from every unlock path without double-writing.
func recordDiscoveredAt(state *PlayerState, id string) {
	if id == "" {
		return
	}
	if state.DiscoveredAt == nil {
		state.DiscoveredAt = map[string]int64{}
	}
	if _, exists := state.DiscoveredAt[id]; exists {
		return
	}
	state.DiscoveredAt[id] = time.Now().UTC().Unix()
}

// craftableRemainingCounts computes, for every item currently in the
// player's collection, how many more distinct recipe results it can still
// lead to that the player hasn't discovered yet. Keyed by item id.
//
// "Currently in the collection" = discovered menu items + unlocked
// ingredients + STARTER ingredients. Starter ingredients (rice, egg, ...)
// are implicitly available from the very start and are never added to
// state.UnlockedIngredients (see Catalog.HasIngredient / IsStarter), so they
// have to be included explicitly here — otherwise every starter ingredient
// silently gets no entry in the map at all (not even a 0), and the client
// just falls back to a hidden badge for all of them regardless of their
// real remaining count.
func craftableRemainingCounts(state *PlayerState) map[string]int {
	discovered := discoveredSet(state.Discovered)
	counts := make(map[string]int, len(state.Discovered)+len(state.UnlockedIngredients)+len(catalog.StarterIDs))
	for _, id := range state.Discovered {
		counts[id] = catalog.RemainingCraftableCount(id, discovered)
	}
	for _, id := range state.UnlockedIngredients {
		if _, exists := counts[id]; exists {
			continue
		}
		counts[id] = catalog.RemainingCraftableCount(id, discovered)
	}
	for id := range catalog.StarterIDs {
		if _, exists := counts[id]; exists {
			continue
		}
		counts[id] = catalog.RemainingCraftableCount(id, discovered)
	}
	return counts
}

func ensureCatalog(logger runtime.Logger) error {
	if catalog != nil {
		return nil
	}
	_, err := loadCatalog()
	if err != nil {
		logger.Error("failed to load catalog: %v", err)
	}
	return err
}
