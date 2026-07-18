package main

type ItemDef struct {
	ID       string `json:"id"`
	NameTH   string `json:"name_th"`
	Emoji    string `json:"emoji"`
	Category string `json:"category"`
	Tier     int    `json:"tier"`
	Starter  bool   `json:"starter"`
}

type RecipeDef struct {
	A      string `json:"a"`
	B      string `json:"b"`
	Result string `json:"result"`
}

type PlayerState struct {
	Discovered          []string `json:"discovered"`
	// DiscoveredAt maps item id -> unix timestamp of when the player first
	// discovered/unlocked it (covers both menu discoveries and ingredient
	// unlocks). Items discovered before this field existed are backfilled to
	// StartedAt (see backfillDiscoveredAt) so sorting stays stable even
	// though their true historical order isn't recoverable.
	DiscoveredAt        map[string]int64 `json:"discovered_at,omitempty"`
	CraftCount          int      `json:"craft_count"`
	DiscoveryPoints     int      `json:"discovery_points"`
	LegacyCoins         int      `json:"coins,omitempty"`
	LegacyStars         int      `json:"stars,omitempty"`
	FirstDiscoverCount  int      `json:"first_discover_count"`
	MaxComboStreak      int      `json:"max_combo_streak"`
	CurrentComboStreak  int      `json:"current_combo_streak"`
	RareDiscoverCount   int      `json:"rare_discover_count"`
	StartedAt           int64    `json:"started_at"`
	Milestone100At      int64    `json:"milestone_100_at"`
	Milestone500At      int64    `json:"milestone_500_at"`
	SeasonMonth         string           `json:"season_month"`
	SeasonDiscoveries   int              `json:"season_discoveries"`
	SeasonCraftCount    int              `json:"season_craft_count"`
	UnlockedIngredients     []string `json:"unlocked_ingredients"`
	ShopLastAutoResetUnix   int64    `json:"shop_last_auto_reset_unix"`
	ShopActiveIDs           []string `json:"shop_active_ids"`
	MissionsDay             string   `json:"missions_day,omitempty"`
	MissionCraftCount       int      `json:"mission_craft_count,omitempty"`
	MissionDiscoverCount    int      `json:"mission_discover_count,omitempty"`
	MissionIngredientCount  int      `json:"mission_ingredient_count,omitempty"`
	MissionsClaimed         []string `json:"missions_claimed,omitempty"`
	// Legacy fields — ignored after auto-only shop migration.
	ShopCycleStart       int64            `json:"shop_cycle_start,omitempty"`
	ShopManualResetCount int              `json:"shop_manual_reset_count,omitempty"`
	ShopItemExpiresAt    map[string]int64 `json:"shop_item_expires_at,omitempty"`
	ShopRarityExpiresAt  map[string]int64 `json:"shop_rarity_expires_at,omitempty"`
}

type CraftRequest struct {
	From []string `json:"from"`
}

type DiscoverResponse struct {
	IsNew           bool     `json:"is_new"`
	ItemID          string   `json:"item_id"`
	DiscoveryPoints int      `json:"discovery_points"`
	DiscoveryCount  int      `json:"discovery_count"`
	CraftCount      int      `json:"craft_count"`
	Coins           int      `json:"coins"`
	Stars           int      `json:"stars"`
	RewardType      string   `json:"reward_type,omitempty"`
	RewardAmount    int      `json:"reward_amount,omitempty"`
	FirstDiscoverer string   `json:"first_discoverer,omitempty"`
	Discovered      []string `json:"discovered"`
	DiscoveredAt    map[string]int64 `json:"discovered_at,omitempty"`
	UnlockedIngredients []string `json:"unlocked_ingredients"`
	CraftableCounts     map[string]int `json:"craftable_counts,omitempty"`
	MissionClaimable    int      `json:"mission_claimable"`
}

type ProfileResponse struct {
	Username        string         `json:"username"`
	DiscoveryCount  int            `json:"discovery_count"`
	DiscoveryPoints int            `json:"discovery_points"`
	CraftCount      int            `json:"craft_count"`
	Efficiency      float64        `json:"efficiency"`
	Discovered      []string       `json:"discovered"`
	Ranks           map[string]int `json:"ranks"`
}

type LeaderboardEntry struct {
	Rank     int64  `json:"rank"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Score    int64  `json:"score"`
}

type LeaderboardResponse struct {
	BoardID     string             `json:"board_id"`
	Title       string             `json:"title"`
	Description string             `json:"description,omitempty"`
	ScoreUnit   string             `json:"score_unit,omitempty"`
	Tier        int                `json:"tier,omitempty"`
	Records     []LeaderboardEntry `json:"records"`
	Owner       *LeaderboardEntry  `json:"owner,omitempty"`
}

type LeaderboardListResponse struct {
	Boards []boardDef `json:"boards"`
}

type HallOfFameEntry struct {
	ItemID       string `json:"item_id"`
	ItemName     string `json:"item_name"`
	Emoji        string `json:"emoji"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	DiscoveredAt int64  `json:"discovered_at"`
}

type HallOfFameResponse struct {
	Entries []HallOfFameEntry `json:"entries"`
}

type ShopConfig struct {
	RarityRotationWeights map[string]int    `json:"rarity_rotation_weights"`
	RarityCosts           map[string]int    `json:"rarity_costs"`
	RarityLabels          map[string]string `json:"rarity_labels"`
	RotationCount         int               `json:"rotation_count"`
}

type GameConfigResponse struct {
	Version           int                 `json:"version"`
	Items             []ItemDef           `json:"items"`
	StarterItems      []string            `json:"starter_items"`
	DiscoverableTotal int                 `json:"discoverable_total"`
	DailyRewardCoins  int                 `json:"daily_reward_coins"`
	Monetization      MonetizationConfig  `json:"monetization"`
	Shop              ShopConfig          `json:"shop"`
}

type MonetizationConfig struct {
	AdRewardCoins    int `json:"ad_reward_coins"`
	StarterPackCoins int `json:"starter_pack_coins"`
}

type ShopIngredientOffer struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Emoji       string `json:"emoji"`
	Rarity      string `json:"rarity"`
	RarityLabel string `json:"rarity_label"`
	Cost        int    `json:"cost"`
	Unlocked    bool   `json:"unlocked"`
	Buyable     bool   `json:"buyable"`
}

type ShopStateResponse struct {
	Coins                 int                   `json:"coins"`
	Stars                 int                   `json:"stars"`
	ShopLastAutoResetUnix int64                 `json:"shop_last_auto_reset_unix"`
	NextShopResetSec      int64                 `json:"next_shop_reset_sec"`
	ShopActiveIDs         []string              `json:"shop_active_ids"`
	UnlockedIngredients   []string              `json:"unlocked_ingredients"`
	DiscoveredAt          map[string]int64      `json:"discovered_at,omitempty"`
	CraftableCounts       map[string]int        `json:"craftable_counts,omitempty"`
	Offers                []ShopIngredientOffer `json:"offers"`
}

type PurchaseIngredientRequest struct {
	IngredientID string `json:"ingredient_id"`
}

type WalletResponse struct {
	Coins int `json:"coins"`
	Stars int `json:"stars"`
}

type FirstDiscoverRecord struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	DiscoveredAt int64  `json:"discovered_at"`
}
