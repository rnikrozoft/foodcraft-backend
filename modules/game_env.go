package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// GameBalance holds tunables loaded from .env (see .env.example). All keys are required.
var gameBalance GameBalance

type GameBalance struct {
	DailyRewardCoins int

	RewardStarPercent       int
	RewardCoinBase          int
	RewardCoinPerTier       int
	RewardCoinRandomPerTier int
	RewardStarAmountLow     int
	RewardStarAmountHigh    int
	RewardStarTierThreshold int

	RateLimitCraft    time.Duration
	RateLimitPurchase time.Duration
	RateLimitDaily    time.Duration

	DiscoveryPointsPerTier  int
	DiscoveryBranchBonus    int
	DiscoveryBranchBonusMax int

	LeaderboardMilestoneMenu1 int
	LeaderboardMilestoneMenu2 int
	LeaderboardSpeedScoreBase int64
	LeaderboardRareTierMin    int

	AdRewardCoins    int
	StarterPackCoins int
}

func loadGameBalance() error {
	var err error
	if gameBalance.DailyRewardCoins, err = envRequiredIntPositive("DAILY_REWARD_COINS"); err != nil {
		return err
	}
	if gameBalance.RewardStarPercent, err = envRequiredIntRange("REWARD_STAR_PERCENT", 0, 100); err != nil {
		return err
	}
	if gameBalance.RewardCoinBase, err = envRequiredInt("REWARD_COIN_BASE"); err != nil {
		return err
	}
	if gameBalance.RewardCoinPerTier, err = envRequiredInt("REWARD_COIN_PER_TIER"); err != nil {
		return err
	}
	if gameBalance.RewardCoinRandomPerTier, err = envRequiredInt("REWARD_COIN_RANDOM_PER_TIER"); err != nil {
		return err
	}
	if gameBalance.RewardStarAmountLow, err = envRequiredIntPositive("REWARD_STAR_AMOUNT_LOW"); err != nil {
		return err
	}
	if gameBalance.RewardStarAmountHigh, err = envRequiredIntPositive("REWARD_STAR_AMOUNT_HIGH"); err != nil {
		return err
	}
	if gameBalance.RewardStarTierThreshold, err = envRequiredInt("REWARD_STAR_TIER_THRESHOLD"); err != nil {
		return err
	}
	if gameBalance.RateLimitCraft, err = envRequiredDurationMs("RATE_LIMIT_CRAFT_MS"); err != nil {
		return err
	}
	if gameBalance.RateLimitPurchase, err = envRequiredDurationMs("RATE_LIMIT_PURCHASE_MS"); err != nil {
		return err
	}
	if gameBalance.RateLimitDaily, err = envRequiredDurationSec("RATE_LIMIT_DAILY_SEC"); err != nil {
		return err
	}
	if gameBalance.DiscoveryPointsPerTier, err = envRequiredIntPositive("DISCOVERY_POINTS_PER_TIER"); err != nil {
		return err
	}
	if gameBalance.DiscoveryBranchBonus, err = envRequiredIntPositive("DISCOVERY_BRANCH_BONUS"); err != nil {
		return err
	}
	if gameBalance.DiscoveryBranchBonusMax, err = envRequiredIntPositive("DISCOVERY_BRANCH_BONUS_MAX"); err != nil {
		return err
	}
	if gameBalance.LeaderboardMilestoneMenu1, err = envRequiredIntPositive("LEADERBOARD_MILESTONE_MENU_1"); err != nil {
		return err
	}
	if gameBalance.LeaderboardMilestoneMenu2, err = envRequiredIntPositive("LEADERBOARD_MILESTONE_MENU_2"); err != nil {
		return err
	}
	if gameBalance.LeaderboardSpeedScoreBase, err = envRequiredInt64Positive("LEADERBOARD_SPEED_SCORE_BASE"); err != nil {
		return err
	}
	if gameBalance.LeaderboardRareTierMin, err = envRequiredIntPositive("LEADERBOARD_RARE_TIER_MIN"); err != nil {
		return err
	}
	if gameBalance.AdRewardCoins, err = envRequiredIntPositive("AD_REWARD_COINS"); err != nil {
		return err
	}
	if gameBalance.StarterPackCoins, err = envRequiredIntPositive("STARTER_PACK_COINS"); err != nil {
		return err
	}

	applyLeaderboardBalanceConfig()
	return nil
}

func envRequiredInt(key string) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0, fmt.Errorf("missing required env: %s (see .env.example)", key)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid env %s=%q: %w", key, v, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("invalid env %s=%q: must be >= 0", key, v)
	}
	return n, nil
}

func envRequiredIntPositive(key string) (int, error) {
	n, err := envRequiredInt(key)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("invalid env %s: must be > 0", key)
	}
	return n, nil
}

func envRequiredIntRange(key string, min, max int) (int, error) {
	n, err := envRequiredInt(key)
	if err != nil {
		return 0, err
	}
	if n < min || n > max {
		return 0, fmt.Errorf("invalid env %s=%d: must be between %d and %d", key, n, min, max)
	}
	return n, nil
}

func envRequiredInt64Positive(key string) (int64, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0, fmt.Errorf("missing required env: %s (see .env.example)", key)
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid env %s=%q: %w", key, v, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("invalid env %s: must be > 0", key)
	}
	return n, nil
}

func envRequiredDurationMs(key string) (time.Duration, error) {
	ms, err := envRequiredIntPositive(key)
	if err != nil {
		return 0, err
	}
	return time.Duration(ms) * time.Millisecond, nil
}

func envRequiredDurationSec(key string) (time.Duration, error) {
	sec, err := envRequiredIntPositive(key)
	if err != nil {
		return 0, err
	}
	return time.Duration(sec) * time.Second, nil
}
