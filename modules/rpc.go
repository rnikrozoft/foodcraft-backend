package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"

	"github.com/heroiclabs/nakama-common/runtime"
)

func rpcProcessCraft(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	if err := ensureCatalog(logger); err != nil {
		return "", rpcError("catalog unavailable", 13)
	}

	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}
	if err := checkRateLimit(userID, "craft", craftRateLimit); err != nil {
		return "", err
	}
	username := mustUsername(ctx)

	var req CraftRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		return "", rpcError("invalid payload", 3)
	}
	a, b, err := normalizePair(req.From)
	if err != nil {
		return "", rpcError(err.Error(), 3)
	}
	itemID, ok := catalog.LookupRecipe(a, b)
	if !ok {
		return "", rpcError("invalid recipe combination", 3)
	}
	if err := catalog.ValidateCraftResult(itemID, a, b); err != nil {
		return "", rpcError(err.Error(), 3)
	}

	var outcome craftOutcome
	err = modifyPlayerStateWithWallet(ctx, nk, userID, func(state *PlayerState) (map[string]int64, map[string]interface{}, error) {
		known := discoveredSet(state.Discovered)
		unlocked := unlockedIngredientSet(state.UnlockedIngredients)
		if !catalog.HasIngredient(known, unlocked, a) || !catalog.HasIngredient(known, unlocked, b) {
			return nil, nil, rpcError("missing prerequisite discoveries", 9)
		}

		resultKind := catalog.CraftResultKind(itemID)
		isNew := false
		switch resultKind {
		case "ingredient":
			isNew = !isIngredientUnlocked(state, itemID)
		case "menu":
			isNew = !containsID(state.Discovered, itemID)
		}

		state.CraftCount++
		onCraftRecorded(state, isNew)

		walletDelta := map[string]int64{}
		var applyErr error
		outcome, applyErr = applyValidatedCraft(ctx, nk, userID, username, state, itemID, a, b, walletDelta)
		if applyErr != nil {
			return nil, nil, applyErr
		}

		return walletDelta, map[string]interface{}{
			"source":  "process_craft",
			"item_id": itemID,
			"is_new":  outcome.IsNew,
		}, nil
	})
	if err != nil {
		return "", err
	}

	state, err := readPlayerState(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}
	updateLeaderboardsSafe(ctx, logger, nk, userID, username, state)

	wallet, err := walletResponse(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read wallet", 13)
	}

	bytes, _ := json.Marshal(buildDiscoverResponse(state, wallet, outcome, itemID))
	return string(bytes), nil
}

func rpcGetProfile(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}
	username := mustUsername(ctx)

	record, err := readPlayerStateRecord(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}
	if err := ensureWalletMigrated(ctx, nk, userID, record); err != nil {
		return "", rpcError("failed to migrate wallet", 13)
	}
	state := record.State

	menuCount := discoveryMenuCount(state)
	efficiency := 0.0
	if state.CraftCount > 0 {
		efficiency = float64(menuCount) / float64(state.CraftCount) * 100.0
	}

	wallet, err := walletResponse(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read wallet", 13)
	}

	bytes, _ := json.Marshal(map[string]interface{}{
		"user_id":              userID,
		"username":             username,
		"discovery_count":      menuCount,
		"discovery_points":     state.DiscoveryPoints,
		"craft_count":          state.CraftCount,
		"coins":                wallet.Coins,
		"stars":                wallet.Stars,
		"efficiency":           efficiency,
		"first_discover_count": state.FirstDiscoverCount,
		"max_combo_streak":     state.MaxComboStreak,
		"rare_discover_count":  state.RareDiscoverCount,
		"discovered":           state.Discovered,
		"unlocked_ingredients": state.UnlockedIngredients,
		"ranks":                allBoardRanks(ctx, nk, userID),
	})
	return string(bytes), nil
}

func rpcListLeaderboards(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	resp := LeaderboardListResponse{Boards: leaderboardDefs}
	bytes, _ := json.Marshal(resp)
	return string(bytes), nil
}

func rpcGetLeaderboard(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}

	var req struct {
		BoardID string `json:"board_id"`
		Limit   int    `json:"limit"`
	}
	if payload != "" {
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			return "", rpcError("invalid payload", 3)
		}
	}
	if req.BoardID == "" {
		req.BoardID = leaderboardFame
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 20
	}

	def, ok := boardDefFor(req.BoardID)
	if !ok {
		return "", rpcError("unknown board", 3)
	}

	records, ownerRecords, _, _, err := nk.LeaderboardRecordsList(ctx, req.BoardID, []string{userID}, req.Limit, "", 0)
	if err != nil {
		return "", rpcError("leaderboard unavailable", 13)
	}

	resp := LeaderboardResponse{
		BoardID:     req.BoardID,
		Title:       def.Title,
		Description: def.Description,
		ScoreUnit:   def.ScoreUnit,
		Tier:        def.Tier,
		Records:     make([]LeaderboardEntry, 0, len(records)),
	}
	for _, record := range records {
		resp.Records = append(resp.Records, LeaderboardEntry{
			Rank:     record.Rank,
			UserID:   record.OwnerId,
			Username: record.GetUsername().GetValue(),
			Score:    record.Score,
		})
	}
	if len(ownerRecords) > 0 {
		owner := ownerRecords[0]
		resp.Owner = &LeaderboardEntry{
			Rank:     owner.Rank,
			UserID:   owner.OwnerId,
			Username: owner.GetUsername().GetValue(),
			Score:    owner.Score,
		}
	}

	bytes, _ := json.Marshal(resp)
	return string(bytes), nil
}

func rpcGetHallOfFame(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	if err := ensureCatalog(logger); err != nil {
		return "", rpcError("catalog unavailable", 13)
	}
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}

	var req struct {
		Limit  int `json:"limit"`
		Cursor int `json:"cursor"`
	}
	if payload != "" {
		_ = json.Unmarshal([]byte(payload), &req)
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 30
	}

	objects, _, err := nk.StorageList(ctx, userID, "", collectionFirstDiscover, req.Limit, "")
	if err != nil {
		return "", rpcError("hall of fame unavailable", 13)
	}

	entries := make([]HallOfFameEntry, 0, len(objects))
	for _, obj := range objects {
		var record FirstDiscoverRecord
		if err := json.Unmarshal([]byte(obj.Value), &record); err != nil {
			continue
		}
		item, ok := catalog.Items[obj.Key]
		name := obj.Key
		emoji := ""
		if ok {
			name = item.NameTH
			emoji = item.Emoji
		}
		entries = append(entries, HallOfFameEntry{
			ItemID:       obj.Key,
			ItemName:     name,
			Emoji:        emoji,
			UserID:       record.UserID,
			Username:     record.Username,
			DiscoveredAt: record.DiscoveredAt,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].DiscoveredAt > entries[j].DiscoveredAt
	})

	resp := HallOfFameResponse{Entries: entries}
	bytes, _ := json.Marshal(resp)
	return string(bytes), nil
}

func rpcGetGameConfig(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	if err := ensureCatalog(logger); err != nil {
		return "", rpcError("catalog unavailable", 13)
	}

	resp := catalog.gameConfigResponse()
	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", rpcError("failed to encode config", 13)
	}
	return string(bytes), nil
}

func rpcGetShopState(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	if err := ensureCatalog(logger); err != nil {
		return "", rpcError("catalog unavailable", 13)
	}
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}

	var resp ShopStateResponse
	err = modifyPlayerState(ctx, nk, userID, func(state *PlayerState) error {
		ensureShopState(state)
		wallet, walletErr := walletResponse(ctx, nk, userID)
		if walletErr != nil {
			return walletErr
		}
		resp = buildShopStateResponse(state, wallet)
		return nil
	})
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", rpcError("failed to encode shop state", 13)
	}
	return string(bytes), nil
}

func rpcSyncWallet(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}

	record, err := readPlayerStateRecord(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}
	if err := ensureWalletMigrated(ctx, nk, userID, record); err != nil {
		return "", rpcError("failed to migrate wallet", 13)
	}

	resp, err := walletResponse(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read wallet", 13)
	}
	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", rpcError("failed to encode wallet", 13)
	}
	return string(bytes), nil
}

func rpcPurchaseShopIngredient(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	if err := ensureCatalog(logger); err != nil {
		return "", rpcError("catalog unavailable", 13)
	}
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}
	if err := checkRateLimit(userID, "shop_purchase", purchaseRateLimit); err != nil {
		return "", err
	}
	var req PurchaseIngredientRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		return "", rpcError("invalid payload", 3)
	}
	if req.IngredientID == "" {
		return "", rpcError("missing ingredient_id", 3)
	}

	var resp ShopStateResponse
	err = modifyPlayerStateWithWallet(ctx, nk, userID, func(state *PlayerState) (map[string]int64, map[string]interface{}, error) {
		wallet, err := walletResponse(ctx, nk, userID)
		if err != nil {
			return nil, nil, err
		}
		cost := shopIngredientCost(req.IngredientID)
		if err := purchaseShopIngredient(state, req.IngredientID, wallet.Coins); err != nil {
			return nil, nil, rpcError(err.Error(), 9)
		}
		walletDelta := map[string]int64{}
		if cost > 0 {
			walletDelta[walletKeyCoins] = -int64(cost)
		}
		ensureShopState(state)
		resp = buildShopStateResponse(state, wallet)
		if cost > 0 {
			resp.Coins = wallet.Coins - cost
		}
		return walletDelta, map[string]interface{}{
			"source":        "purchase_shop_ingredient",
			"ingredient_id": req.IngredientID,
			"coin_cost":     cost,
		}, nil
	})
	if err != nil {
		return "", err
	}

	updatedWallet, err := walletResponse(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read wallet", 13)
	}
	state, err := readPlayerState(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}
	resp = buildShopStateResponse(state, updatedWallet)

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", rpcError("failed to encode shop state", 13)
	}
	return string(bytes), nil
}
