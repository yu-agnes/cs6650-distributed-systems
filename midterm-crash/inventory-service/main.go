package main

import (
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Controls whether the service is in "failure mode"
var (
	failureMode bool
	modeMu      sync.RWMutex
)

func main() {
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Toggle failure mode: POST /fault/on or /fault/off
	r.POST("/fault/on", func(c *gin.Context) {
		modeMu.Lock()
		failureMode = true
		modeMu.Unlock()
		log.Println("[FAULT INJECTION] Failure mode ON - responses will be delayed 5s")
		c.JSON(200, gin.H{"failure_mode": true})
	})

	r.POST("/fault/off", func(c *gin.Context) {
		modeMu.Lock()
		failureMode = false
		modeMu.Unlock()
		log.Println("[FAULT INJECTION] Failure mode OFF - normal operation")
		c.JSON(200, gin.H{"failure_mode": false})
	})

	// Inventory check endpoint
	r.GET("/inventory/:productId", func(c *gin.Context) {
		modeMu.RLock()
		isFailing := failureMode
		modeMu.RUnlock()

		if isFailing {
			// Simulate a slow/degraded service: sleep 5 seconds
			time.Sleep(5 * time.Second)
			c.JSON(503, gin.H{"error": "service overloaded"})
			return
		}

		// Normal response: return stock info in ~10ms
		time.Sleep(10 * time.Millisecond)
		c.JSON(200, gin.H{
			"product_id": c.Param("productId"),
			"in_stock":   true,
			"quantity":   42,
		})
	})

	log.Println("Inventory Service starting on port 8081...")
	r.Run(":8081")
}
