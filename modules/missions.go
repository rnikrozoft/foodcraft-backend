package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

// Mission counter types.
const (
	missionTypeCraft      = "craft"
	missionTypeDiscover   = "discover"
	missionTypeIngredient = "ingredient"
)

type MissionDef struct {
	ID          string
	Title       string
	Desc        string
	Type        string
	Target      int
	RewardCoins int
	RewardStars int
}

// Daily missions, reset at UTC midnight (same convention as daily reward / shop rotation).
var missionDefs = []MissionDef{
	{ID: "daily_craft_5", Title: "นักผสมขยัน", Desc: "ผสมวัตถุดิบ 5 ครั้ง", Type: missionTypeCraft, Target: 5, RewardCoins: 50},
	{ID: "daily_craft_15", Title: "มาราธอนตำรับ", Desc: "ผสมวัตถุดิบ 15 ครั้ง", Type: missionTypeCraft, Target: 15, RewardCoins: 120},
	{ID: "daily_discover_1", Title: "เชฟช่างค้น", Desc: "ค้นพบเมนูใหม่ 1 เมนู", Type: missionTypeDiscover, Target: 1, RewardCoins: 70},
	{ID: "daily_discover_3", Title: "นักสำรวจรสชาติ", Desc: "ค้นพบเมนูใหม่ 3 เมนู", Type: missionTypeDiscover, Target: 3, RewardCoins: 100, RewardStars: 1},
	{ID: "daily_ingredient_1", Title: "วัตถุดิบลับ", Desc: "ปลดล็อกวัตถุดิบใหม่ 1 ชนิด", Type: missionTypeIngredient, Target: 1, RewardCoins: 60},
}

type MissionView struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Desc        string `json:"desc"`
	Type        string `json:"type"`
	Target      int    `json:"target"`
	Progress    int    `json:"progress"`
	RewardCoins int    `json:"reward_coins"`
	RewardStars int    `json:"reward_stars"`
	Claimed     bool   `json:"claimed"`
	CanClaim    bool   `json:"can_claim"`
}

type MissionsResponse struct {
	Day            string        `json:"day"`
	NextResetSec   int64         `json:"next_reset_sec"`
	ClaimableCount int           `json:"claimable_count"`
	Missions       []MissionView `json:"missions"`
}

type ClaimMissionRequest struct {
	MissionID string `json:"mission_id"`
}

type ClaimMissionResponse struct {
	MissionID      string        `json:"mission_id"`
	CoinsReceived  int           `json:"coins_received"`
	StarsReceived  int           `json:"stars_received"`
	Coins          int           `json:"coins"`
	Stars          int           `json:"stars"`
	Day            string        `json:"day"`
	NextResetSec   int64         `json:"next_reset_sec"`
	ClaimableCount int           `json:"claimable_count"`
	Missions       []MissionView `json:"missions"`
}

func missionDayKey() string {
	return time.Now().UTC().Format("2006-01-02")
}

func nextMissionResetSec() int64 {
	nextMidnight := todayMidnightUTC().Add(24 * time.Hour)
	remaining := nextMidnight.Unix() - time.Now().UTC().Unix()
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ensureMissionDay rolls daily counters over when the UTC day changes.
func ensureMissionDay(state *PlayerState) {
	day := missionDayKey()
	if state.MissionsDay == day {
		return
	}
	state.MissionsDay = day
	state.MissionCraftCount = 0
	state.MissionDiscoverCount = 0
	state.MissionIngredientCount = 0
	state.MissionsClaimed = nil
}

// recordMissionProgress is called from the craft flow (inside the player-state
// transaction) so counters stay atomic with the craft itself.
func recordMissionProgress(state *PlayerState, resultKind string, isNew bool) {
	ensureMissionDay(state)
	state.MissionCraftCount++
	if !isNew {
		return
	}
	switch resultKind {
	case "menu":
		state.MissionDiscoverCount++
	case "ingredient":
		state.MissionIngredientCount++
	}
}

func missionDefByID(id string) (MissionDef, bool) {
	for _, def := range missionDefs {
		if def.ID == id {
			return def, true
		}
	}
	return MissionDef{}, false
}

func missionProgress(state *PlayerState, def MissionDef) int {
	switch def.Type {
	case missionTypeCraft:
		return state.MissionCraftCount
	case missionTypeDiscover:
		return state.MissionDiscoverCount
	case missionTypeIngredient:
		return state.MissionIngredientCount
	}
	return 0
}

func missionClaimed(state *PlayerState, id string) bool {
	for _, claimed := range state.MissionsClaimed {
		if claimed == id {
			return true
		}
	}
	return false
}

func buildMissionViews(state *PlayerState) ([]MissionView, int) {
	views := make([]MissionView, 0, len(missionDefs))
	claimable := 0
	for _, def := range missionDefs {
		progress := missionProgress(state, def)
		if progress > def.Target {
			progress = def.Target
		}
		claimed := missionClaimed(state, def.ID)
		canClaim := !claimed && progress >= def.Target
		if canClaim {
			claimable++
		}
		views = append(views, MissionView{
			ID:          def.ID,
			Title:       def.Title,
			Desc:        def.Desc,
			Type:        def.Type,
			Target:      def.Target,
			Progress:    progress,
			RewardCoins: def.RewardCoins,
			RewardStars: def.RewardStars,
			Claimed:     claimed,
			CanClaim:    canClaim,
		})
	}
	return views, claimable
}

// claimableMissionCount powers the footer badge; it assumes the state's
// mission day is current (call ensureMissionDay on a copy first for reads).
func claimableMissionCount(state *PlayerState) int {
	_, claimable := buildMissionViews(state)
	return claimable
}

func buildMissionsResponse(state *PlayerState) MissionsResponse {
	views, claimable := buildMissionViews(state)
	return MissionsResponse{
		Day:            missionDayKey(),
		NextResetSec:   nextMissionResetSec(),
		ClaimableCount: claimable,
		Missions:       views,
	}
}

func rpcGetMissions(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}
	state, err := readPlayerState(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}
	// In-memory rollover only: reads never persist; the next craft/claim does.
	ensureMissionDay(state)
	resp := buildMissionsResponse(state)
	bytes, _ := json.Marshal(resp)
	return string(bytes), nil
}

func rpcClaimMission(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}
	if err := checkRateLimit(userID, "mission_claim", gameBalance.RateLimitDaily); err != nil {
		return "", err
	}

	var req ClaimMissionRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		return "", rpcError("invalid payload", 3)
	}
	def, ok := missionDefByID(req.MissionID)
	if !ok {
		return "", rpcError("unknown mission", 3)
	}

	err = modifyPlayerStateWithWallet(ctx, nk, userID, func(state *PlayerState) (map[string]int64, map[string]interface{}, error) {
		ensureMissionDay(state)
		if missionClaimed(state, def.ID) {
			return nil, nil, rpcError("mission already claimed", 9)
		}
		if missionProgress(state, def) < def.Target {
			return nil, nil, rpcError("mission not complete", 9)
		}
		state.MissionsClaimed = append(state.MissionsClaimed, def.ID)

		changeset := map[string]int64{}
		if def.RewardCoins > 0 {
			changeset[walletKeyCoins] = int64(def.RewardCoins)
		}
		if def.RewardStars > 0 {
			changeset[walletKeyStars] = int64(def.RewardStars)
		}
		return changeset, map[string]interface{}{
			"source":     "mission_claim",
			"mission_id": def.ID,
		}, nil
	})
	if err != nil {
		return "", err
	}

	state, err := readPlayerState(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}
	ensureMissionDay(state)
	wallet, err := walletResponse(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read wallet", 13)
	}

	views, claimable := buildMissionViews(state)
	resp := ClaimMissionResponse{
		MissionID:      def.ID,
		CoinsReceived:  def.RewardCoins,
		StarsReceived:  def.RewardStars,
		Coins:          wallet.Coins,
		Stars:          wallet.Stars,
		Day:            missionDayKey(),
		NextResetSec:   nextMissionResetSec(),
		ClaimableCount: claimable,
		Missions:       views,
	}
	out, _ := json.Marshal(resp)
	return string(out), nil
}
