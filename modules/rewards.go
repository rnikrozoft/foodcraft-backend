package main

import (
	"crypto/rand"
	"encoding/binary"
)

const starDropPercent = 28

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
	if secureRandInt(100) < starDropPercent {
		return "star", starAmount(tier)
	}
	return "coin", coinAmount(tier)
}

func coinAmount(tier int) int {
	base := 10 + tier*12
	return base + secureRandInt(tier*8+1)
}

func starAmount(tier int) int {
	if tier <= 2 {
		return 1
	}
	return 2
}
