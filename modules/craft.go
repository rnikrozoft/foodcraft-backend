package main

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
)

type craftOutcome struct {
	IsNew             bool
	RewardType        string
	RewardAmount      int
	FirstDiscoverer   string
	WasFirst          bool
}

func applyValidatedCraft(
	ctx context.Context,
	nk runtime.NakamaModule,
	userID, username string,
	state *PlayerState,
	itemID, a, b string,
	walletDelta map[string]int64,
) (craftOutcome, error) {
	outcome := craftOutcome{}
	resultKind := catalog.CraftResultKind(itemID)

	switch resultKind {
	case "ingredient":
		outcome.IsNew = !isIngredientUnlocked(state, itemID)
		if outcome.IsNew {
			unlockIngredient(state, itemID)
			outcome.RewardType, outcome.RewardAmount = rollDiscoveryReward(1)
			mergeChangeset(walletDelta, rewardChangeset(outcome.RewardType, outcome.RewardAmount))
		}
	case "menu":
		outcome.IsNew = !containsID(state.Discovered, itemID)
		if outcome.IsNew {
			state.Discovered = uniqueAppend(state.Discovered, itemID)
			state.DiscoveryPoints = totalPoints(catalog, state.Discovered)
			outcome.RewardType, outcome.RewardAmount = rollDiscoveryReward(catalog.TierFor(itemID))
			mergeChangeset(walletDelta, rewardChangeset(outcome.RewardType, outcome.RewardAmount))
		if ok, err := recordFirstDiscover(ctx, nk, itemID, userID, username); err == nil && ok {
			outcome.FirstDiscoverer = username
			outcome.WasFirst = true
		}
			onNewDiscovery(state, itemID, outcome.WasFirst)
		}
	}

	return outcome, nil
}

func buildDiscoverResponse(state *PlayerState, wallet WalletResponse, outcome craftOutcome, itemID string) DiscoverResponse {
	return DiscoverResponse{
		IsNew:               outcome.IsNew,
		ItemID:              itemID,
		DiscoveryPoints:     state.DiscoveryPoints,
		DiscoveryCount:      discoveryMenuCount(state),
		CraftCount:          state.CraftCount,
		Coins:               wallet.Coins,
		Stars:               wallet.Stars,
		RewardType:          outcome.RewardType,
		RewardAmount:        outcome.RewardAmount,
		FirstDiscoverer:     outcome.FirstDiscoverer,
		Discovered:          state.Discovered,
		UnlockedIngredients: copyStringSlice(state.UnlockedIngredients),
	}
}

func updateLeaderboardsSafe(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, username string, state *PlayerState) {
	if err := updateLeaderboards(ctx, nk, userID, username, state); err != nil {
		logger.Warn("leaderboard update failed: %v", err)
	}
}
