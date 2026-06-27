package main

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	leaderboardFame     = "culinary_fame"
	leaderboardExplorer = "explorer"
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
	if isNew {
		state.Discovered = uniqueAppend(state.Discovered, req.ItemID)
		state.DiscoveryPoints = totalPoints(catalog, state.Discovered)
		if ok, err := recordFirstDiscover(ctx, nk, req.ItemID, userID, username); err != nil {
			logger.Warn("first discover write failed: %v", err)
		} else if ok {
			firstDiscoverer = username
		}
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
		DiscoveryCount:  len(state.Discovered),
		CraftCount:      state.CraftCount,
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
		_, _ = recordFirstDiscover(ctx, nk, discovery.ItemID, userID, username)
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
		DiscoveryCount:  len(state.Discovered),
		CraftCount:      state.CraftCount,
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

	state, err := readPlayerState(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}
	state.CraftCount++
	if err := writePlayerState(ctx, nk, userID, state); err != nil {
		return "", rpcError("failed to save state", 13)
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

	efficiency := 0.0
	if state.CraftCount > 0 {
		efficiency = float64(len(state.Discovered)) / float64(state.CraftCount) * 100.0
	}

	fameRank, _ := getOwnerRank(ctx, nk, leaderboardFame, userID)
	explorerRank, _ := getOwnerRank(ctx, nk, leaderboardExplorer, userID)

	resp := ProfileResponse{
		Username:        username,
		DiscoveryCount:  len(state.Discovered),
		DiscoveryPoints: state.DiscoveryPoints,
		CraftCount:      state.CraftCount,
		Efficiency:      efficiency,
		Discovered:      state.Discovered,
		Ranks: map[string]int{
			leaderboardFame:     fameRank,
			leaderboardExplorer: explorerRank,
		},
	}
	bytes, _ := json.Marshal(map[string]interface{}{
		"user_id":          userID,
		"username":         resp.Username,
		"discovery_count":  resp.DiscoveryCount,
		"discovery_points": resp.DiscoveryPoints,
		"craft_count":      resp.CraftCount,
		"efficiency":       resp.Efficiency,
		"discovered":       resp.Discovered,
		"ranks":            resp.Ranks,
	})
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

	title := "Culinary Fame"
	if req.BoardID == leaderboardExplorer {
		title = "Explorer"
	}

	records, ownerRecords, _, _, err := nk.LeaderboardRecordsList(ctx, req.BoardID, []string{userID}, req.Limit, "", 0)
	if err != nil {
		return "", rpcError("leaderboard unavailable", 13)
	}

	resp := LeaderboardResponse{
		BoardID: req.BoardID,
		Title:   title,
		Records: make([]LeaderboardEntry, 0, len(records)),
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
