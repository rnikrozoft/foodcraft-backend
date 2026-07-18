package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	collectionPlayer        = "player"
	keyState                = "state"
	collectionFirstDiscover = "first_discover"
	systemUserID            = "00000000-0000-0000-0000-000000000000"
)

type playerStateRecord struct {
	State   *PlayerState
	Version string
}

func defaultPlayerState() *PlayerState {
	return &PlayerState{
		Discovered:          []string{},
		UnlockedIngredients: []string{},
		ShopItemExpiresAt:   map[string]int64{},
		ShopRarityExpiresAt: map[string]int64{},
		ShopActiveIDs:       []string{},
	}
}

func normalizePlayerState(state *PlayerState) {
	if state.Discovered == nil {
		state.Discovered = []string{}
	}
	if state.UnlockedIngredients == nil {
		state.UnlockedIngredients = []string{}
	}
	if state.ShopItemExpiresAt == nil {
		state.ShopItemExpiresAt = map[string]int64{}
	}
	if state.ShopRarityExpiresAt == nil {
		state.ShopRarityExpiresAt = map[string]int64{}
	}
	if state.ShopActiveIDs == nil {
		state.ShopActiveIDs = []string{}
	}
	backfillDiscoveredAt(state)
}

// backfillDiscoveredAt ensures every already-discovered/unlocked item has a
// DiscoveredAt entry. Items discovered before this field existed have no real
// history to recover, so they're all stamped with the player's account
// StartedAt — stable across reads (no jitter in sort order) even though it
// can't reflect their true original order. New discoveries from now on get
// an accurate timestamp the moment they happen (see recordDiscoveredAt).
func backfillDiscoveredAt(state *PlayerState) {
	if state.DiscoveredAt == nil {
		state.DiscoveredAt = map[string]int64{}
	}
	fallback := state.StartedAt
	for _, id := range state.Discovered {
		if _, ok := state.DiscoveredAt[id]; !ok {
			state.DiscoveredAt[id] = fallback
		}
	}
	for _, id := range state.UnlockedIngredients {
		if _, ok := state.DiscoveredAt[id]; !ok {
			state.DiscoveredAt[id] = fallback
		}
	}
}

func readPlayerStateRecord(ctx context.Context, nk runtime.NakamaModule, userID string) (*playerStateRecord, error) {
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{{
		Collection: collectionPlayer,
		Key:        keyState,
		UserID:     userID,
	}})
	if err != nil {
		return nil, err
	}
	if len(objects) == 0 {
		return &playerStateRecord{State: defaultPlayerState(), Version: ""}, nil
	}

	var state PlayerState
	if err := json.Unmarshal([]byte(objects[0].Value), &state); err != nil {
		return nil, err
	}
	normalizePlayerState(&state)
	return &playerStateRecord{
		State:   &state,
		Version: objects[0].Version,
	}, nil
}

func readPlayerState(ctx context.Context, nk runtime.NakamaModule, userID string) (*PlayerState, error) {
	record, err := readPlayerStateRecord(ctx, nk, userID)
	if err != nil {
		return nil, err
	}
	return record.State, nil
}

func writePlayerStateRecord(ctx context.Context, nk runtime.NakamaModule, userID string, record *playerStateRecord) error {
	bytes, err := json.Marshal(record.State)
	if err != nil {
		return err
	}
	acks, err := nk.StorageWrite(ctx, []*runtime.StorageWrite{{
		Collection:      collectionPlayer,
		Key:             keyState,
		UserID:          userID,
		Value:           string(bytes),
		Version:         record.Version,
		PermissionRead:  1,
		PermissionWrite: 0,
	}})
	if err != nil {
		return err
	}
	if len(acks) > 0 {
		record.Version = acks[0].Version
	}
	return nil
}

func writePlayerState(ctx context.Context, nk runtime.NakamaModule, userID string, state *PlayerState) error {
	record, err := readPlayerStateRecord(ctx, nk, userID)
	if err != nil {
		return err
	}
	record.State = state
	return writePlayerStateRecord(ctx, nk, userID, record)
}

func isVersionConflict(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "version") || strings.Contains(message, "conflict")
}

func modifyPlayerState(
	ctx context.Context,
	nk runtime.NakamaModule,
	userID string,
	fn func(*PlayerState) error,
) error {
	const maxAttempts = 3
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		record, err := readPlayerStateRecord(ctx, nk, userID)
		if err != nil {
			return err
		}
		if err := ensureWalletMigrated(ctx, nk, userID, record); err != nil {
			return err
		}
		if err := fn(record.State); err != nil {
			return err
		}
		if err := writePlayerStateRecord(ctx, nk, userID, record); err != nil {
			if isVersionConflict(err) {
				lastErr = err
				continue
			}
			return err
		}
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("state version conflict: %w", lastErr)
	}
	return fmt.Errorf("state version conflict")
}

func modifyPlayerStateWithWallet(
	ctx context.Context,
	nk runtime.NakamaModule,
	userID string,
	fn func(*PlayerState) (map[string]int64, map[string]interface{}, error),
) error {
	const maxAttempts = 3
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		record, err := readPlayerStateRecord(ctx, nk, userID)
		if err != nil {
			return err
		}
		if err := ensureWalletMigrated(ctx, nk, userID, record); err != nil {
			return err
		}
		changeset, metadata, err := fn(record.State)
		if err != nil {
			return err
		}
		if err := savePlayerStateWithWalletRecord(ctx, nk, userID, record, changeset, metadata); err != nil {
			if isVersionConflict(err) {
				lastErr = err
				continue
			}
			return err
		}
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("state version conflict: %w", lastErr)
	}
	return fmt.Errorf("state version conflict")
}

func ensurePlayerStateExists(ctx context.Context, nk runtime.NakamaModule, userID string) error {
	record, err := readPlayerStateRecord(ctx, nk, userID)
	if err != nil {
		return err
	}
	if record.Version != "" {
		return nil
	}
	return writePlayerStateRecord(ctx, nk, userID, record)
}

func firstDiscoverExists(ctx context.Context, nk runtime.NakamaModule, itemID string) (bool, error) {
	for _, ownerID := range []string{systemUserID, ""} {
		objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{{
			Collection: collectionFirstDiscover,
			Key:        itemID,
			UserID:     ownerID,
		}})
		if err != nil {
			return false, err
		}
		if len(objects) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func recordFirstDiscover(ctx context.Context, nk runtime.NakamaModule, itemID, userID, username string) (bool, error) {
	exists, err := firstDiscoverExists(ctx, nk, itemID)
	if err != nil {
		return false, err
	}
	if exists {
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
		UserID:          systemUserID,
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
