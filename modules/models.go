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
	SeasonMonth         string   `json:"season_month"`
	SeasonDiscoveries   int      `json:"season_discoveries"`
	SeasonCraftCount    int      `json:"season_craft_count"`
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

type FirstDiscoverRecord struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	DiscoveredAt int64  `json:"discovered_at"`
}
