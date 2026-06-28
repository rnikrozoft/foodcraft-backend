package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	collectionReward   = "reward"
	keyDailyReward     = "daily"
	dailyRewardCoins   = 120
)

type DailyRewardRecord struct {
	LastClaimUnix int64 `json:"last_claim_unix"`
}

type DailyRewardStatusResponse struct {
	CanClaim      bool `json:"can_claim"`
	Coins         int  `json:"coins"`
	NextClaimSec  int64 `json:"next_claim_sec"`
	LastClaimUnix int64 `json:"last_claim_unix"`
}

type DailyRewardClaimResponse struct {
	CoinsReceived int   `json:"coins_received"`
	Coins         int   `json:"coins"`
	Stars         int   `json:"stars"`
	CanClaim      bool  `json:"can_claim"`
	NextClaimSec  int64 `json:"next_claim_sec"`
	LastClaimUnix int64 `json:"last_claim_unix"`
}

func todayMidnightUTC() time.Time {
	t := time.Now().UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func canClaimDailyReward(lastClaimUnix int64) bool {
	if lastClaimUnix <= 0 {
		return true
	}
	return time.Unix(lastClaimUnix, 0).UTC().Before(todayMidnightUTC())
}

func nextDailyClaimSec(lastClaimUnix int64) int64 {
	if canClaimDailyReward(lastClaimUnix) {
		return 0
	}
	nextMidnight := todayMidnightUTC().Add(24 * time.Hour)
	remaining := nextMidnight.Unix() - time.Now().UTC().Unix()
	if remaining < 0 {
		return 0
	}
	return remaining
}

func buildDailyRewardStatus(record DailyRewardRecord) DailyRewardStatusResponse {
	return DailyRewardStatusResponse{
		CanClaim:      canClaimDailyReward(record.LastClaimUnix),
		Coins:         dailyRewardCoins,
		NextClaimSec:  nextDailyClaimSec(record.LastClaimUnix),
		LastClaimUnix: record.LastClaimUnix,
	}
}

func readDailyRewardRecord(ctx context.Context, nk runtime.NakamaModule, userID string) (DailyRewardRecord, string, error) {
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{{
		Collection: collectionReward,
		Key:        keyDailyReward,
		UserID:     userID,
	}})
	if err != nil {
		return DailyRewardRecord{}, "", err
	}
	if len(objects) == 0 {
		return DailyRewardRecord{}, "", nil
	}
	var record DailyRewardRecord
	if err := json.Unmarshal([]byte(objects[0].Value), &record); err != nil {
		return DailyRewardRecord{}, "", err
	}
	return record, objects[0].Version, nil
}

func rpcGetDailyReward(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}
	record, _, err := readDailyRewardRecord(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read daily reward", 13)
	}
	resp := buildDailyRewardStatus(record)
	bytes, _ := json.Marshal(resp)
	return string(bytes), nil
}

func rpcClaimDailyReward(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}
	if err := checkRateLimit(userID, "daily_claim", dailyRateLimit); err != nil {
		return "", err
	}

	const maxAttempts = 3
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		record, version, err := readDailyRewardRecord(ctx, nk, userID)
		if err != nil {
			return "", rpcError("failed to read daily reward", 13)
		}
		if !canClaimDailyReward(record.LastClaimUnix) {
			return "", rpcError("daily reward already claimed", 9)
		}

		record.LastClaimUnix = time.Now().UTC().Unix()
		bytes, err := json.Marshal(record)
		if err != nil {
			return "", rpcError("failed to encode daily reward", 13)
		}

		changeset := map[string]int64{walletKeyCoins: int64(dailyRewardCoins)}
		writes := []*runtime.StorageWrite{{
			Collection:      collectionReward,
			Key:             keyDailyReward,
			UserID:          userID,
			Value:           string(bytes),
			Version:         version,
			PermissionRead:  1,
			PermissionWrite: 0,
		}}
		walletUpdates := []*runtime.WalletUpdate{{
			UserID:    userID,
			Changeset: changeset,
			Metadata: map[string]interface{}{
				"source": "daily_reward",
			},
		}}

		_, _, err = nk.MultiUpdate(ctx, nil, writes, nil, walletUpdates, true)
		if err != nil {
			if isVersionConflict(err) {
				lastErr = err
				continue
			}
			return "", rpcError("failed to claim daily reward", 13)
		}

		wallet, err := walletResponse(ctx, nk, userID)
		if err != nil {
			return "", rpcError("failed to read wallet", 13)
		}
		status := buildDailyRewardStatus(record)
		resp := DailyRewardClaimResponse{
			CoinsReceived: dailyRewardCoins,
			Coins:         wallet.Coins,
			Stars:         wallet.Stars,
			CanClaim:      status.CanClaim,
			NextClaimSec:  status.NextClaimSec,
			LastClaimUnix: record.LastClaimUnix,
		}
		out, _ := json.Marshal(resp)
		return string(out), nil
	}
	if lastErr != nil {
		logger.Warn("daily reward version conflict: %v", lastErr)
	}
	return "", rpcError("daily reward busy, try again", 13)
}
