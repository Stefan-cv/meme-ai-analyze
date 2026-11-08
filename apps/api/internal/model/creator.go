package model

import "time"

// CreatorRecord represents a founder's profile stored in the database.
type CreatorRecord struct {
	Address          string    `db:"address"`
	TotalTokens      int       `db:"total_tokens"`
	RuggedTokens     int       `db:"rugged_tokens"`
	AvgLifetimeHours float64   `db:"avg_lifetime_hours"`
	FirstSeenAt      time.Time `db:"first_seen_at"`
	LastSeenAt       time.Time `db:"last_seen_at"`
	RiskLabel        string    `db:"risk_label"`
}

// CreatorToken represents a token created by a founder.
type CreatorToken struct {
	ID              int64      `db:"id"`
	CreatorAddress  string     `db:"creator_address"`
	TokenAddress    string     `db:"token_address"`
	TokenName       string     `db:"token_name"`
	TokenSymbol     string     `db:"token_symbol"`
	Status          string     `db:"status"` // "active" / "rugged" / "unknown"
	CreatedAt       time.Time  `db:"created_at"`
	RuggedAt        *time.Time `db:"rugged_at"`
}

// IndexerState represents the indexer's last processed block.
type IndexerState struct {
	ChainID   int       `db:"chain_id"`
	LastBlock int64     `db:"last_block"`
	UpdatedAt time.Time `db:"updated_at"`
}
