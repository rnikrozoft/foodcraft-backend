package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/heroiclabs/nakama-common/runtime"
)

// hintCostStars is read from the environment with a safe default so the server
// still boots if the key is missing. Stars are the premium currency (เพชร).
func hintCostStars() int {
	if v := strings.TrimSpace(os.Getenv("HINT_COST_STARS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 2
}

type HintResponse struct {
	Available bool   `json:"available"`
	AID       string `json:"a_id,omitempty"`
	BID       string `json:"b_id,omitempty"`
	AName     string `json:"a_name,omitempty"`
	BName     string `json:"b_name,omitempty"`
	AEmoji    string `json:"a_emoji,omitempty"`
	BEmoji    string `json:"b_emoji,omitempty"`
	Cost      int    `json:"cost"`
	Coins     int    `json:"coins"`
	Stars     int    `json:"stars"`
	Message   string `json:"message,omitempty"`
}

// playerOwnedSet = everything the player can currently place in the craft slots:
// starter items + shop-unlocked ingredients + discovered menus/ingredients.
func playerOwnedSet(state *PlayerState) map[string]bool {
	owned := make(map[string]bool, len(state.Discovered)+len(state.UnlockedIngredients)+len(catalog.StarterIDs))
	for id := range catalog.StarterIDs {
		owned[id] = true
	}
	for _, id := range state.UnlockedIngredients {
		owned[id] = true
	}
	for _, id := range state.Discovered {
		owned[id] = true
	}
	return owned
}

// findHintPair returns a recipe whose two inputs the player already owns but
// whose result they have NOT discovered yet. It prefers the lowest-tier result
// (the most approachable next discovery) and picks randomly among ties.
func findHintPair(state *PlayerState) (a, b, result string, ok bool) {
	owned := playerOwnedSet(state)
	discovered := make(map[string]bool, len(state.Discovered))
	for _, id := range state.Discovered {
		discovered[id] = true
	}

	type cand struct{ a, b, res string }
	var cands []cand
	bestTier := int(^uint(0) >> 1) // max int

	for key, res := range catalog.RecipeByKey {
		if discovered[res] {
			continue
		}
		parts := strings.SplitN(key, "|", 2)
		if len(parts) != 2 {
			continue
		}
		if !owned[parts[0]] || !owned[parts[1]] {
			continue
		}
		t := catalog.Items[res].Tier
		if t < bestTier {
			bestTier = t
			cands = cands[:0]
		}
		if t == bestTier {
			cands = append(cands, cand{parts[0], parts[1], res})
		}
	}

	if len(cands) == 0 {
		return "", "", "", false
	}
	pick := cands[secureRandInt(len(cands))]
	return pick.a, pick.b, pick.res, true
}

// rpcBuyHint spends HINT_COST_STARS stars to reveal one craftable-but-undiscovered
// recipe pair. If no such recipe exists, it returns available:false WITHOUT charging.
func rpcBuyHint(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	if err := ensureCatalog(logger); err != nil {
		return "", rpcError("catalog unavailable", 13)
	}
	userID, err := mustUserID(ctx)
	if err != nil {
		return "", err
	}
	if err := checkRateLimit(userID, "buy_hint", gameBalance.RateLimitDaily); err != nil {
		return "", err
	}

	state, err := readPlayerState(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read state", 13)
	}
	wallet0, err := walletResponse(ctx, nk, userID)
	if err != nil {
		return "", rpcError("failed to read wallet", 13)
	}
	cost := hintCostStars()

	// No charge if there's nothing left to hint at.
	if _, _, _, ok := findHintPair(state); !ok {
		resp := HintResponse{Available: false, Cost: cost, Coins: wallet0.Coins, Stars: wallet0.Stars, Message: "no_hint"}
		out, _ := json.Marshal(resp)
		return string(out), nil
	}
	if wallet0.Stars < cost {
		return "", rpcError("insufficient stars", 9)
	}

	var resp HintResponse
	err = modifyPlayerStateWithWallet(ctx, nk, userID, func(state *PlayerState) (map[string]int64, map[string]interface{}, error) {
		wallet, werr := walletResponse(ctx, nk, userID)
		if werr != nil {
			return nil, nil, werr
		}
		if wallet.Stars < cost {
			return nil, nil, rpcError("insufficient stars", 9)
		}
		na, nb, _, nok := findHintPair(state)
		if !nok {
			return nil, nil, rpcError("no hint available", 9)
		}
		ai := catalog.Items[na]
		bi := catalog.Items[nb]
		resp = HintResponse{
			Available: true,
			AID:       na, BID: nb,
			AName: ai.NameTH, BName: bi.NameTH,
			AEmoji: ai.Emoji, BEmoji: bi.Emoji,
			Cost:  cost,
			Coins: wallet.Coins,
			Stars: wallet.Stars - cost,
		}
		return map[string]int64{walletKeyStars: -int64(cost)}, map[string]interface{}{"source": "buy_hint"}, nil
	})
	if err != nil {
		return "", err
	}

	out, _ := json.Marshal(resp)
	return string(out), nil
}
