// Package config parses flags and environment for the server.
package config

import (
	"flag"
	"os"
	"time"
)

type Config struct {
	Addr    string
	DB      string
	Refresh time.Duration
	Token   string
}

// Parse reads -addr/-db/-refresh/-token flags plus the TINYRSS_TOKEN env var.
// Env wins over the flag for the token (a flag is visible via ps).
func Parse() *Config {
	var cfg Config
	flag.StringVar(&cfg.Addr, "addr", "127.0.0.1:8087", "listen address")
	flag.StringVar(&cfg.DB, "db", "data/tinyrss.db", "SQLite database path")
	flag.DurationVar(&cfg.Refresh, "refresh", 15*time.Minute, "poll interval for due feeds")
	flag.StringVar(&cfg.Token, "token", "", "access token (TINYRSS_TOKEN env takes precedence)")
	flag.Parse()
	if env := os.Getenv("TINYRSS_TOKEN"); env != "" {
		cfg.Token = env
	}
	return &cfg
}
