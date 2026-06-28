package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	collectionPlayer       = "player"
	keyState               = "state"
	collectionFirstDiscover = "first_discover"
)

func readPlayerState(ctx context.Context, nk runtime.NakamaModule, userID string) (*PlayerState, error) {
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{{
		Collection: collectionPlayer,
		Key:        keyState,
		UserID:     userID,
	}})
	if err != nil {
		return nil, err
	}
	if len(objects) == 0 {
		return &PlayerState{
			Discovered:          []string{},
			UnlockedIngredients: []string{},
			ShopItemExpiresAt:   map[string]int64{},
			ShopActiveIDs:       []string{},
		}, nil
	}
	var state PlayerState
	if err := json.Unmarshal([]byte(objects[0].Value), &state); err != nil {
		return nil, err
	}
	if state.Discovered == nil {
		state.Discovered = []string{}
	}
	if state.UnlockedIngredients == nil {
		state.UnlockedIngredients = []string{}
	}
	if state.ShopItemExpiresAt == nil {
		state.ShopItemExpiresAt = map[string]int64{}
	}
	if state.ShopActiveIDs == nil {
		state.ShopActiveIDs = []string{}
	}
	return &state, nil
}

func writePlayerState(ctx context.Context, nk runtime.NakamaModule, userID string, state *PlayerState) error {
	bytes, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{{
		Collection:      collectionPlayer,
		Key:             keyState,
		UserID:          userID,
		Value:           string(bytes),
		PermissionRead:  1,
		PermissionWrite: 0,
	}})
	return err
}

func recordFirstDiscover(ctx context.Context, nk runtime.NakamaModule, itemID, userID, username string) (bool, error) {
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{{
		Collection: collectionFirstDiscover,
		Key:        itemID,
		UserID:     "",
	}})
	if err != nil {
		return false, err
	}
	if len(objects) > 0 {
		return false, nil
	}

	record := FirstDiscoverRecord{
		UserID:       userID,
		Username:     username,
		DiscoveredAt: time.Now().UTC().Unix(),
	}
	bytes, err := json.Marshal(record)
	if err != nil {
		return false, err
	}
	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{{
		Collection:      collectionFirstDiscover,
		Key:             itemID,
		UserID:          "",
		Value:           string(bytes),
		PermissionRead:  2,
		PermissionWrite: 0,
	}})
	if err != nil {
		return false, err
	}
	return true, nil
}

func getOwnerRank(ctx context.Context, nk runtime.NakamaModule, boardID, userID string) (int, error) {
	records, _, _, _, err := nk.LeaderboardRecordsList(ctx, boardID, []string{userID}, 1, "", 0)
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 0, nil
	}
	return int(records[0].Rank), nil
}
