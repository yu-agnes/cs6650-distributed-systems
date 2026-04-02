package main

import (
	"fmt"
	"hw10/shared"
	"log"

	"github.com/gin-gonic/gin"
)

var (
	store  *shared.Store
	config shared.Config
)

func main() {
	config = shared.LoadConfig()
	store = shared.NewStore()

	r := gin.Default()

	// --- Client-facing endpoints ---
	// Any node can receive these. The node that receives a write
	// becomes the Write Coordinator for that request.
	r.POST("/set", handleSet)
	r.GET("/get/:key", handleGet)

	// --- Internal endpoint: called by the Write Coordinator ---
	r.POST("/internal/set", handleInternalSet)

	// --- Testing endpoint: local read without coordination ---
	r.GET("/local_read/:key", handleLocalRead)

	// --- Health check ---
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "role": "node"})
	})

	addr := fmt.Sprintf(":%s", config.Port)
	log.Printf("[node] Starting on %s | N=%d W=%d R=%d | Peers=%v",
		addr, config.N, config.W, config.R, config.Peers)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
