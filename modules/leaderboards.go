package main

import (
	"context"
	"fmt"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	leaderboardFame             = "culinary_fame"
	leaderboardExplorer         = "explorer"
	leaderboardFirstDiscoverer  = "first_discoverer"
	leaderboardEfficiency       = "efficiency"
	leaderboardSpeed100         = "speed_runner_100"
	leaderboardSpeed500         = "speed_runner_500"
	leaderboardCombo            = "combo_master"
	leaderboardRareHunter       = "rare_hunter"
	leaderboardCatThai          = "category_thai"
	leaderboardCatJapanese      = "category_japanese"
	leaderboardCatChinese         = "category_chinese"
	leaderboardCatWestern       = "category_western"
	leaderboardCatDessert       = "category_dessert"
	leaderboardSeasonExplorer   = "season_explorer"
	leaderboardSeasonEfficiency = "season_efficiency"
)

var leaderboardDefs []boardDef

type boardDef struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Tier        int    `json:"tier"`
	Description string `json:"description"`
	ScoreUnit   string `json:"score_unit"`
}

func applyLeaderboardBalanceConfig() {
	m1 := gameBalance.LeaderboardMilestoneMenu1
	m2 := gameBalance.LeaderboardMilestoneMenu2
	rareMin := gameBalance.LeaderboardRareTierMin
	leaderboardDefs = []boardDef{
		{leaderboardFame, "ชื่อเสียง", 1, "แต้มค้นพบรวม — เมนูหายากได้แต้มมากกว่า", "แต้ม"},
		{leaderboardExplorer, "นักสำรวจ", 1, "จำนวนเมนูที่ค้นพบทั้งหมด", "เมนู"},
		{leaderboardFirstDiscoverer, "ผู้ค้นพบคนแรก", 1, "จำนวนเมนูที่เป็นคนแรกของเซิร์ฟเวอร์", "ครั้ง"},
		{leaderboardEfficiency, "ประสิทธิภาพ", 2, "เมนูที่ค้นพบ ÷ จำนวนครั้งที่ผสม (ยิ่งสูงยิ่งเก่ง)", "%"},
		{leaderboardSpeed100, fmt.Sprintf("ความเร็ว %d เมนู", m1), 2, fmt.Sprintf("ใครค้นพบครบ %d เมนูเร็วที่สุด", m1), "คะแนน"},
		{leaderboardSpeed500, fmt.Sprintf("ความเร็ว %d เมนู", m2), 2, fmt.Sprintf("ใครค้นพบครบ %d เมนูเร็วที่สุด", m2), "คะแนน"},
		{leaderboardCombo, "คอมโบมาสเตอร์", 2, "ค้นพบเมนูใหม่ติดต่อกันสูงสุด", "ครั้ง"},
		{leaderboardRareHunter, "นักล่าของหายาก", 2, fmt.Sprintf("จำนวนเมนูระดับหายาก (tier %d+) ที่ค้นพบ", rareMin), "เมนู"},
		{leaderboardCatThai, "อาหารไทย", 3, "จำนวนเมนูอาหารไทยที่ค้นพบ", "เมนู"},
		{leaderboardCatJapanese, "อาหารญี่ปุ่น", 3, "จำนวนเมนูอาหารญี่ปุ่นที่ค้นพบ", "เมนู"},
		{leaderboardCatChinese, "อาหารจีน", 3, "จำนวนเมนูอาหารจีนที่ค้นพบ", "เมนู"},
		{leaderboardCatWestern, "อาหารตะวันตก", 3, "จำนวนเมนูอาหารตะวันตกที่ค้นพบ", "เมนู"},
		{leaderboardCatDessert, "ของหวาน", 3, "จำนวนเมนูของหวานที่ค้นพบ", "เมนู"},
		{leaderboardSeasonExplorer, "นักสำรวจประจำเดือน", 5, "เมนูที่ค้นพบใหม่ในเดือนนี้", "เมนู"},
		{leaderboardSeasonEfficiency, "ประสิทธิภาพประจำเดือน", 5, "ประสิทธิภาพการผสมในเดือนนี้", "%"},
	}
}

func boardDefFor(id string) (boardDef, bool) {
	for _, def := range leaderboardDefs {
		if def.ID == id {
			return def, true
		}
	}
	return boardDef{}, false
}

func ensurePlayerTimestamps(state *PlayerState) {
	now := time.Now().UTC().Unix()
	if state.StartedAt == 0 {
		state.StartedAt = now
	}
	refreshSeason(state)
}

func refreshSeason(state *PlayerState) {
	month := time.Now().UTC().Format("2006-01")
	if state.SeasonMonth != month {
		state.SeasonMonth = month
		state.SeasonDiscoveries = 0
		state.SeasonCraftCount = 0
	}
}

func onNewDiscovery(state *PlayerState, itemID string, wasFirst bool) {
	ensurePlayerTimestamps(state)
	state.CurrentComboStreak++
	if state.CurrentComboStreak > state.MaxComboStreak {
		state.MaxComboStreak = state.CurrentComboStreak
	}
	if wasFirst {
		state.FirstDiscoverCount++
	}
	if catalog.TierFor(itemID) >= gameBalance.LeaderboardRareTierMin {
		state.RareDiscoverCount++
	}
	state.SeasonDiscoveries++
	count := discoveryMenuCount(state)
	now := time.Now().UTC().Unix()
	if count >= gameBalance.LeaderboardMilestoneMenu1 && state.Milestone100At == 0 {
		state.Milestone100At = now
	}
	if count >= gameBalance.LeaderboardMilestoneMenu2 && state.Milestone500At == 0 {
		state.Milestone500At = now
	}
}

func onCraftRecorded(state *PlayerState, isNewDiscovery bool) {
	ensurePlayerTimestamps(state)
	state.SeasonCraftCount++
	if !isNewDiscovery {
		state.CurrentComboStreak = 0
	}
}

func discoveryMenuCount(state *PlayerState) int {
	count := 0
	for _, id := range state.Discovered {
		item, ok := catalog.Items[id]
		if !ok || item.Tier < 1 {
			continue
		}
		count++
	}
	return count
}

func categoryCount(state *PlayerState, category string) int {
	count := 0
	for _, id := range state.Discovered {
		item, ok := catalog.Items[id]
		if !ok || item.Tier < 1 || item.Category != category {
			continue
		}
		count++
	}
	return count
}

func efficiencyScore(discoveredMenus, craftCount int) int64 {
	if craftCount <= 0 {
		return 0
	}
	return int64(discoveredMenus) * 10000 / int64(craftCount)
}

func speedScore(reachedAt int64) int64 {
	if reachedAt <= 0 {
		return 0
	}
	score := gameBalance.LeaderboardSpeedScoreBase - reachedAt
	if score < 0 {
		return 0
	}
	return score
}

func updateLeaderboards(ctx context.Context, nk runtime.NakamaModule, userID, username string, state *PlayerState) error {
	ensurePlayerTimestamps(state)
	menuCount := discoveryMenuCount(state)
	eff := efficiencyScore(menuCount, state.CraftCount)
	seasonEff := efficiencyScore(state.SeasonDiscoveries, state.SeasonCraftCount)

	writes := []struct {
		id    string
		score int64
	}{
		{leaderboardFame, int64(state.DiscoveryPoints)},
		{leaderboardExplorer, int64(menuCount)},
		{leaderboardFirstDiscoverer, int64(state.FirstDiscoverCount)},
		{leaderboardEfficiency, eff},
		{leaderboardSpeed100, speedScore(state.Milestone100At)},
		{leaderboardSpeed500, speedScore(state.Milestone500At)},
		{leaderboardCombo, int64(state.MaxComboStreak)},
		{leaderboardRareHunter, int64(state.RareDiscoverCount)},
		{leaderboardCatThai, int64(categoryCount(state, "thai"))},
		{leaderboardCatJapanese, int64(categoryCount(state, "japanese"))},
		{leaderboardCatChinese, int64(categoryCount(state, "chinese"))},
		{leaderboardCatWestern, int64(categoryCount(state, "western"))},
		{leaderboardCatDessert, int64(categoryCount(state, "dessert"))},
		{leaderboardSeasonExplorer, int64(state.SeasonDiscoveries)},
		{leaderboardSeasonEfficiency, seasonEff},
	}

	for _, entry := range writes {
		if _, err := nk.LeaderboardRecordWrite(ctx, entry.id, userID, username, entry.score, 0, map[string]interface{}{}, nil); err != nil {
			return fmt.Errorf("%s: %w", entry.id, err)
		}
	}
	return nil
}

func allBoardRanks(ctx context.Context, nk runtime.NakamaModule, userID string) map[string]int {
	ranks := make(map[string]int, len(leaderboardDefs))
	for _, def := range leaderboardDefs {
		rank, _ := getOwnerRank(ctx, nk, def.ID, userID)
		ranks[def.ID] = rank
	}
	return ranks
}
