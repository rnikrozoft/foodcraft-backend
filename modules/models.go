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
	Discovered      []string `json:"discovered"`
	CraftCount      int      `json:"craft_count"`
	DiscoveryPoints int      `json:"discovery_points"`
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
	BoardID string             `json:"board_id"`
	Title   string             `json:"title"`
	Records []LeaderboardEntry `json:"records"`
	Owner   *LeaderboardEntry  `json:"owner,omitempty"`
}

type FirstDiscoverRecord struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	DiscoveredAt int64  `json:"discovered_at"`
}
