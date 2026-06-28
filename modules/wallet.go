package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	walletKeyCoins = "coins"
	walletKeyStars = "stars"
)

func readWalletBalances(ctx context.Context, nk runtime.NakamaModule, userID string) (coins int, stars int, err error) {
	account, err := nk.AccountGetId(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	coins, stars = parseWalletJSON(account.GetWallet())
	return coins, stars, nil
}

func parseWalletJSON(walletJSON string) (coins int, stars int) {
	if walletJSON == "" {
		return 0, 0
	}
	var wallet map[string]interface{}
	if err := json.Unmarshal([]byte(walletJSON), &wallet); err != nil {
		return 0, 0
	}
	return walletAmount(wallet, walletKeyCoins), walletAmount(wallet, walletKeyStars)
}

func walletAmount(wallet map[string]interface{}, key string) int {
	raw, ok := wallet[key]
	if !ok || raw == nil {
		return 0
	}
	switch value := raw.(type) {
	case float64:
		return int(value)
	case int64:
		return int(value)
	case int:
		return value
	case json.Number:
		parsed, err := value.Int64()
		if err != nil {
			return 0
		}
		return int(parsed)
	default:
		return 0
	}
}

func walletResponse(ctx context.Context, nk runtime.NakamaModule, userID string) (WalletResponse, error) {
	coins, stars, err := readWalletBalances(ctx, nk, userID)
	if err != nil {
		return WalletResponse{}, err
	}
	return WalletResponse{Coins: coins, Stars: stars}, nil
}

func rewardChangeset(rewardType string, amount int) map[string]int64 {
	if amount <= 0 {
		return nil
	}
	changeset := map[string]int64{}
	switch rewardType {
	case "coin":
		changeset[walletKeyCoins] = int64(amount)
	case "star":
		changeset[walletKeyStars] = int64(amount)
	default:
		return nil
	}
	return changeset
}

func mergeChangeset(target map[string]int64, delta map[string]int64) {
	for key, amount := range delta {
		if amount == 0 {
			continue
		}
		target[key] += amount
	}
}

func walletUpdate(
	ctx context.Context,
	nk runtime.NakamaModule,
	userID string,
	changeset map[string]int64,
	metadata map[string]interface{},
) error {
	if len(changeset) == 0 {
		return nil
	}
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	_, _, err := nk.WalletUpdate(ctx, userID, changeset, metadata, true)
	return err
}

func savePlayerStateWithWalletRecord(
	ctx context.Context,
	nk runtime.NakamaModule,
	userID string,
	record *playerStateRecord,
	changeset map[string]int64,
	metadata map[string]interface{},
) error {
	bytes, err := json.Marshal(record.State)
	if err != nil {
		return err
	}

	writes := []*runtime.StorageWrite{{
		Collection:      collectionPlayer,
		Key:             keyState,
		UserID:          userID,
		Value:           string(bytes),
		Version:         record.Version,
		PermissionRead:  1,
		PermissionWrite: 0,
	}}

	var walletUpdates []*runtime.WalletUpdate
	if len(changeset) > 0 {
		if metadata == nil {
			metadata = map[string]interface{}{}
		}
		walletUpdates = append(walletUpdates, &runtime.WalletUpdate{
			UserID:    userID,
			Changeset: changeset,
			Metadata:  metadata,
		})
	}

	acks, _, err := nk.MultiUpdate(ctx, nil, writes, nil, walletUpdates, true)
	if err != nil {
		return err
	}
	if len(acks) > 0 {
		record.Version = acks[0].Version
	}
	return nil
}

func savePlayerStateWithWallet(
	ctx context.Context,
	nk runtime.NakamaModule,
	userID string,
	state *PlayerState,
	changeset map[string]int64,
	metadata map[string]interface{},
) error {
	record, err := readPlayerStateRecord(ctx, nk, userID)
	if err != nil {
		return err
	}
	record.State = state
	return savePlayerStateWithWalletRecord(ctx, nk, userID, record, changeset, metadata)
}

func ensureWalletMigrated(ctx context.Context, nk runtime.NakamaModule, userID string, record *playerStateRecord) error {
	state := record.State
	legacyCoins := state.LegacyCoins
	legacyStars := state.LegacyStars
	if legacyCoins <= 0 && legacyStars <= 0 {
		return nil
	}

	changeset := map[string]int64{}
	if legacyCoins > 0 {
		changeset[walletKeyCoins] = int64(legacyCoins)
	}
	if legacyStars > 0 {
		changeset[walletKeyStars] = int64(legacyStars)
	}

	if err := walletUpdate(ctx, nk, userID, changeset, map[string]interface{}{
		"source": "legacy_storage_migration",
	}); err != nil {
		return fmt.Errorf("wallet migration failed: %w", err)
	}

	state.LegacyCoins = 0
	state.LegacyStars = 0
	return writePlayerStateRecord(ctx, nk, userID, record)
}

func loadPlayerStateAndWallet(ctx context.Context, nk runtime.NakamaModule, userID string) (*PlayerState, WalletResponse, error) {
	record, err := readPlayerStateRecord(ctx, nk, userID)
	if err != nil {
		return nil, WalletResponse{}, err
	}
	if err := ensureWalletMigrated(ctx, nk, userID, record); err != nil {
		return nil, WalletResponse{}, err
	}
	wallet, err := walletResponse(ctx, nk, userID)
	if err != nil {
		return nil, WalletResponse{}, err
	}
	return record.State, wallet, nil
}
