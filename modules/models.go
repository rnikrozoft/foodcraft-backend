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
	CraftCount          int      `json:"craft_count"`
	DiscoveryPoints     int      `json:"discovery_points"`
	Coins               int      `json:"coins"`
	Stars               int      `json:"stars"`
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
	UnlockedIngredients []string         `json:"unlocked_ingredients"`
	ShopCycleStart      int64            `json:"shop_cycle_start"`
	ShopManualResetCount int             `json:"shop_manual_reset_count"`
	ShopItemExpiresAt   map[string]int64 `json:"shop_item_expires_at"`
	ShopActiveIDs       []string         `json:"shop_active_ids"`
}

type DiscoverRequest struct {
	ItemID string   `json:"item_id"`
	From   []string `json:"from"`
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
	UnlockedIngredients []string `json:"unlocked_ingredients"`
}

type SyncRequest struct {
	Discoveries []DiscoverRequest `json:"discoveries"`
	CraftCount  int               `json:"craft_count"`
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

type ShopResetPrices struct {
	Mid      int    `json:"mid"`
	High     int    `json:"high"`
	Currency string `json:"currency"`
}

type ShopConfig struct {
	ResetCycleSec      int                `json:"reset_cycle_sec"`
	ResetMidCount      int                `json:"reset_mid_count"`
	ResetPrices        ShopResetPrices    `json:"reset_prices"`
	RarityCooldownDays map[string]int     `json:"rarity_cooldown_days"`
	RarityCosts        map[string]int     `json:"rarity_costs"`
	RarityLabels       map[string]string  `json:"rarity_labels"`
	RotationCount      int                `json:"rotation_count"`
}

type GameConfigResponse struct {
	Version int        `json:"version"`
	Shop    ShopConfig `json:"shop"`
}

type ShopResetInfo struct {
	PriceType        string `json:"price_type"`
	PriceLabel       string `json:"price_label"`
	CoinCost         int    `json:"coin_cost"`
	ResetsUsed       int    `json:"resets_used"`
	NextFreeResetSec int64  `json:"next_free_reset_sec"`
}

type ShopIngredientOffer struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	Emoji              string `json:"emoji"`
	Rarity             string `json:"rarity"`
	RarityLabel        string `json:"rarity_label"`
	Cost               int    `json:"cost"`
	Unlocked           bool   `json:"unlocked"`
	Buyable            bool   `json:"buyable"`
	CooldownSec        int64  `json:"cooldown_sec"`
	WindowRemainingSec int64  `json:"window_remaining_sec"`
}

type ShopStateResponse struct {
	Coins                int                   `json:"coins"`
	ShopCycleStart       int64                 `json:"shop_cycle_start"`
	ShopManualResetCount int                   `json:"shop_manual_reset_count"`
	ShopItemExpiresAt    map[string]int64      `json:"shop_item_expires_at"`
	ShopActiveIDs        []string              `json:"shop_active_ids"`
	UnlockedIngredients  []string              `json:"unlocked_ingredients"`
	ResetInfo            ShopResetInfo         `json:"reset_info"`
	Offers               []ShopIngredientOffer `json:"offers"`
}

type ResetShopRequest struct {
	PaymentType string `json:"payment_type"`
}

type PurchaseIngredientRequest struct {
	IngredientID string `json:"ingredient_id"`
}

type AdjustWalletRequest struct {
	CoinsDelta int `json:"coins_delta"`
	StarsDelta int `json:"stars_delta"`
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
