package main

import (
	"context"
	"database/sql"

	"github.com/heroiclabs/nakama-common/runtime"
)

func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	if _, err := loadCatalog(); err != nil {
		logger.Error("unable to load game catalog: %v", err)
		return err
	}
	logger.Info("Foodcraft catalog loaded: %d items, %d recipes", len(catalog.Items), len(catalog.RecipeByKey))
	logger.Info("Shop config: rotation=%d weights=%v costs=%v",
		catalog.Shop.RotationCount,
		catalog.Shop.RarityRotationWeights,
		catalog.Shop.RarityCosts,
	)

	if err := ensureLeaderboards(ctx, logger, nk); err != nil {
		return err
	}

	if err := initializer.RegisterAfterAuthenticateDevice(afterAuthenticateDevice); err != nil {
		return err
	}

	if err := initializer.RegisterRpc("process_craft", rpcProcessCraft); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("get_profile", rpcGetProfile); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("get_leaderboard", rpcGetLeaderboard); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("list_leaderboards", rpcListLeaderboards); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("get_hall_of_fame", rpcGetHallOfFame); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("get_game_config", rpcGetGameConfig); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("get_shop_state", rpcGetShopState); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("purchase_shop_ingredient", rpcPurchaseShopIngredient); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("sync_wallet", rpcSyncWallet); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("get_daily_reward", rpcGetDailyReward); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("claim_daily_reward", rpcClaimDailyReward); err != nil {
		return err
	}

	logger.Info("Foodcraft Nakama module loaded")
	return nil
}

func ensureLeaderboards(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule) error {
	seasonBoards := map[string]bool{
		leaderboardSeasonExplorer:   true,
		leaderboardSeasonEfficiency: true,
	}
	for _, def := range leaderboardDefs {
		reset := "0 0 * * *"
		if seasonBoards[def.ID] {
			reset = "0 0 1 * *"
		}
		if err := nk.LeaderboardCreate(ctx, def.ID, true, "desc", "best", reset, map[string]interface{}{
			"title": def.Title,
			"tier":  def.Tier,
		}, false); err != nil {
			if !isAlreadyExists(err) {
				logger.Error("leaderboard create failed for %s: %v", def.ID, err)
				return err
			}
		}
	}
	return nil
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "leaderboard already exists" || err.Error() == "Leaderboard already exists"
}
