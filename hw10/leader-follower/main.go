package main

import (
	"fmt"
	"hw10/shared"
	"log"

	"github.com/gin-gonic/gin"
)

// Global store and config accessible by all handlers.
var (
	store  *shared.Store
	config shared.Config
)

func main() {
	config = shared.LoadConfig()
	store = shared.NewStore()

	r := gin.Default()

	// --- Client-facing endpoints (both Leader and Follower expose these) ---
	r.POST("/set", handleSet)
	r.GET("/get/:key", handleGet)

	// --- Internal endpoints (called by Leader or other nodes, not by clients) ---
	r.POST("/internal/set", handleInternalSet)
	r.GET("/internal/get/:key", handleInternalGet)

	// --- Testing endpoint: returns local data without any replication logic ---
	r.GET("/local_read/:key", handleLocalRead)

	// --- Health check ---
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "role": config.Role})
	})

	addr := fmt.Sprintf(":%s", config.Port)
	log.Printf("[%s] Starting on %s | N=%d W=%d R=%d | Peers=%v",
		config.Role, addr, config.N, config.W, config.R, config.Peers)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
