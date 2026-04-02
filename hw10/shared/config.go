package shared

import (
	"os"
	"strconv"
	"strings"
)

// Config holds everything a node needs to know at startup:
// who it is, who its peers are, and what R/W strategy to use.
type Config struct {
	// Role is "leader" or "follower" (leader-follower mode),
	// or "node" (leaderless mode).
	Role string

	// Port this node listens on.
	Port string

	// Peers is the list of other node addresses, e.g. ["node2:8080", "node3:8080"].
	// For Leader: these are the Follower addresses.
	// For Leaderless: these are all other node addresses.
	Peers []string

	// N is the total number of nodes (always 5 in this assignment).
	N int

	// W is how many nodes must acknowledge a write before returning to the client.
	W int

	// R is how many nodes must be read before returning to the client.
	R int
}

// LoadConfig reads configuration from environment variables.
// This lets the same binary behave differently depending on
// how it's launched in Docker Compose.
//
// Expected env vars:
//
//	ROLE       = "leader" | "follower" | "node"
//	PORT       = "8080"  (default)
//	PEERS      = "node2:8080,node3:8080,node4:8080,node5:8080"
//	N          = "5"
//	W          = "5"     (how many acks needed for write)
//	R          = "1"     (how many reads needed)
func LoadConfig() Config {
	cfg := Config{
		Role: getEnv("ROLE", "leader"),
		Port: getEnv("PORT", "8080"),
		N:    getEnvInt("N", 5),
		W:    getEnvInt("W", 5),
		R:    getEnvInt("R", 1),
	}

	peersStr := os.Getenv("PEERS")
	if peersStr != "" {
		cfg.Peers = strings.Split(peersStr, ",")
	}

	return cfg
}

// --- helper functions ---

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
