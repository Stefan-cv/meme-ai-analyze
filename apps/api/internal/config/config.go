package config

import (
	"fmt"

	"github.com/zeromicro/go-zero/rest"
)

// Config is the top-level application configuration
type Config struct {
	rest.RestConf
	CorsAllowedOrigins []string     `json:",optional"`
	Postgres           PostgresConf `json:",optional"`
	Redis              RedisConf    `json:",optional"`
	BSC                BSCConf      `json:",optional"`
	AI                 AIConf       `json:",optional"`
	Oracle             OracleConf   `json:",optional"`
	BscScan            BscScanConf  `json:",optional"`
}

// BscScanConf BscScan API configuration
type BscScanConf struct {
	APIKey  string
	BaseURL string `json:",default=https://api.bscscan.com/v2/api"`
}

// PostgresConf database configuration
type PostgresConf struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	MaxConns int `json:",default=20"`
	MinConns int `json:",default=5"`
}

// DSN returns the PostgreSQL connection string
func (p PostgresConf) DSN() string {
	return "postgres://" + p.User + ":" + p.Password +
		"@" + p.Host + ":" + itoa(p.Port) +
		"/" + p.Database + "?sslmode=disable"
}

// RedisConf cache configuration
type RedisConf struct {
	Host     string
	Port     int
	Password string `json:",optional"`
	DB       int    `json:",default=0"`
}

// Addr returns the Redis address
func (r RedisConf) Addr() string {
	return r.Host + ":" + itoa(r.Port)
}

// BSCConf BSC chain RPC configuration
type BSCConf struct {
	RPCUrl       string
	ChainID      int64
	BackupRPCUrl string `json:",optional"`
}

// AIConf AI service configuration
type AIConf struct {
	Provider string // "openai" or "gemini"
	APIKey   string
	Model    string `json:",default=gpt-4o"`
}

// OracleConf on-chain oracle contract configuration
type OracleConf struct {
	ContractAddress string
	PrivateKey      string
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
