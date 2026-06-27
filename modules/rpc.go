package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"

	"github.com/heroiclabs/nakama-common/runtime"
)

func rpcDiscoverRecipe(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	if err := ensureCatalog(logger); err != nil {
		return "", rpcError("catalog unavailable", 13)
	}

	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}
	username := mustUsername(ctx)

	var req DiscoverRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		return "", rpcError("invalid payload", 3)
	}
	a, b, err := normalizePair(req.From)
	if err != nil {
		return "", rpcError(err.Error(), 3)
	}
	if err := catalog.ValidateDiscovery(req.ItemID, a, b); err != nil {
		return "", rpcError(err.Error(), 3)
	}

	state, err := readPlayerState(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}

	known := discoveredSet(state.Discovered)
	if !catalog.HasIngredient(known, a) || !catalog.HasIngredient(known, b) {
		return "", rpcError("missing prerequisite discoveries", 9)
	}

	isNew := !containsID(state.Discovered, req.ItemID)
	firstDiscoverer := ""
	rewardType := ""
	rewardAmount := 0
	wasFirst := false
	if isNew {
		state.Discovered = uniqueAppend(state.Discovered, req.ItemID)
		state.DiscoveryPoints = totalPoints(catalog, state.Discovered)
		rewardType, rewardAmount = rollDiscoveryReward(catalog.TierFor(req.ItemID))
		applyReward(state, rewardType, rewardAmount)
		if ok, err := recordFirstDiscover(ctx, nk, req.ItemID, userID, username); err != nil {
			logger.Warn("first discover write failed: %v", err)
		} else if ok {
			firstDiscoverer = username
			wasFirst = true
		}
		onNewDiscovery(state, req.ItemID, wasFirst)
	}

	if err := writePlayerState(ctx, nk, userID, state); err != nil {
		return "", rpcError("failed to save state", 13)
	}
	if err := updateLeaderboards(ctx, nk, userID, username, state); err != nil {
		logger.Warn("leaderboard update failed: %v", err)
	}

	resp := DiscoverResponse{
		IsNew:           isNew,
		ItemID:          req.ItemID,
		DiscoveryPoints: state.DiscoveryPoints,
		DiscoveryCount:  discoveryMenuCount(state),
		CraftCount:      state.CraftCount,
		Coins:           state.Coins,
		Stars:           state.Stars,
		RewardType:      rewardType,
		RewardAmount:    rewardAmount,
		FirstDiscoverer: firstDiscoverer,
		Discovered:      state.Discovered,
	}
	bytes, _ := json.Marshal(resp)
	return string(bytes), nil
}

func rpcSyncDiscoveries(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	if err := ensureCatalog(logger); err != nil {
		return "", rpcError("catalog unavailable", 13)
	}

	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}
	username := mustUsername(ctx)

	var req SyncRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		return "", rpcError("invalid payload", 3)
	}

	state, err := readPlayerState(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}

	if req.CraftCount > state.CraftCount {
		state.CraftCount = req.CraftCount
		refreshSeason(state)
	}

	known := discoveredSet(state.Discovered)
	for _, discovery := range req.Discoveries {
		a, b, err := normalizePair(discovery.From)
		if err != nil {
			continue
		}
		if err := catalog.ValidateDiscovery(discovery.ItemID, a, b); err != nil {
			continue
		}
		if !catalog.HasIngredient(known, a) || !catalog.HasIngredient(known, b) {
			continue
		}
		if containsID(state.Discovered, discovery.ItemID) {
			continue
		}
		state.Discovered = uniqueAppend(state.Discovered, discovery.ItemID)
		known[discovery.ItemID] = true
		rewardType, rewardAmount := rollDiscoveryReward(catalog.TierFor(discovery.ItemID))
		applyReward(state, rewardType, rewardAmount)
		wasFirst := false
		if ok, _ := recordFirstDiscover(ctx, nk, discovery.ItemID, userID, username); ok {
			wasFirst = true
		}
		onNewDiscovery(state, discovery.ItemID, wasFirst)
	}

	state.DiscoveryPoints = totalPoints(catalog, state.Discovered)
	if err := writePlayerState(ctx, nk, userID, state); err != nil {
		return "", rpcError("failed to save state", 13)
	}
	if err := updateLeaderboards(ctx, nk, userID, username, state); err != nil {
		logger.Warn("leaderboard update failed: %v", err)
	}

	resp := DiscoverResponse{
		IsNew:           false,
		DiscoveryPoints: state.DiscoveryPoints,
		DiscoveryCount:  discoveryMenuCount(state),
		CraftCount:      state.CraftCount,
		Coins:           state.Coins,
		Stars:           state.Stars,
		Discovered:      state.Discovered,
	}
	bytes, _ := json.Marshal(resp)
	return string(bytes), nil
}

func rpcRecordCraft(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}
	username := mustUsername(ctx)

	var req struct {
		IsNewDiscovery bool `json:"is_new_discovery"`
	}
	if payload != "" {
		_ = json.Unmarshal([]byte(payload), &req)
	}

	state, err := readPlayerState(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}
	state.CraftCount++
	onCraftRecorded(state, req.IsNewDiscovery)
	if err := writePlayerState(ctx, nk, userID, state); err != nil {
		return "", rpcError("failed to save state", 13)
	}
	if err := updateLeaderboards(ctx, nk, userID, username, state); err != nil {
		logger.Warn("leaderboard update failed: %v", err)
	}
	bytes, _ := json.Marshal(map[string]int{"craft_count": state.CraftCount})
	return string(bytes), nil
}

func rpcGetProfile(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}
	username := mustUsername(ctx)

	state, err := readPlayerState(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}

	menuCount := discoveryMenuCount(state)
	efficiency := 0.0
	if state.CraftCount > 0 {
		efficiency = float64(menuCount) / float64(state.CraftCount) * 100.0
	}

	bytes, _ := json.Marshal(map[string]interface{}{
		"user_id":              userID,
		"username":             username,
		"discovery_count":      menuCount,
		"discovery_points":     state.DiscoveryPoints,
		"craft_count":          state.CraftCount,
		"coins":                state.Coins,
		"stars":                state.Stars,
		"efficiency":           efficiency,
		"first_discover_count": state.FirstDiscoverCount,
		"max_combo_streak":     state.MaxComboStreak,
		"rare_discover_count":  state.RareDiscoverCount,
		"discovered":           state.Discovered,
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
