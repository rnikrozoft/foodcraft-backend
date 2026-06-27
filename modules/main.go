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

	if err := ensureLeaderboards(ctx, logger, nk); err != nil {
		return err
	}

	if err := initializer.RegisterRpc("discover_recipe", rpcDiscoverRecipe); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("sync_discoveries", rpcSyncDiscoveries); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("record_craft", rpcRecordCraft); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("get_profile", rpcGetProfile); err != nil {
		return err
	}
	if err := initializer.RegisterRpc("get_leaderboard", rpcGetLeaderboard); err != nil {
		return err
	}

	logger.Info("Foodcraft Nakama module loaded")
	return nil
}

func ensureLeaderboards(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule) error {
	boards := []struct {
		id    string
		title string
	}{
		{leaderboardFame, "Culinary Fame"},
		{leaderboardExplorer, "Explorer"},
	}
	for _, board := range boards {
		if err := nk.LeaderboardCreate(ctx, board.id, true, "desc", "best", "0 0 * * *", map[string]interface{}{
			"title": board.title,
		}, false); err != nil {
			if !isAlreadyExists(err) {
				logger.Error("leaderboard create failed for %s: %v", board.id, err)
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
