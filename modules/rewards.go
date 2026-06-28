package main

import (
	"crypto/rand"
	"encoding/binary"
)

func secureRandInt(max int) int {
	if max <= 0 {
		return 0
	}
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0
	}
	n := binary.LittleEndian.Uint64(buf[:])
	return int(n % uint64(max))
}

func rollDiscoveryReward(tier int) (rewardType string, amount int) {
	if tier < 1 {
		tier = 1
	}
	if secureRandInt(100) < gameBalance.RewardStarPercent {
		return "star", starAmount(tier)
	}
	return "coin", coinAmount(tier)
}

func coinAmount(tier int) int {
	base := gameBalance.RewardCoinBase + tier*gameBalance.RewardCoinPerTier
	randomSpan := tier*gameBalance.RewardCoinRandomPerTier + 1
	return base + secureRandInt(randomSpan)
}

func starAmount(tier int) int {
	if tier <= gameBalance.RewardStarTierThreshold {
		return gameBalance.RewardStarAmountLow
	}
	return gameBalance.RewardStarAmountHigh
}
