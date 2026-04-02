package main

import (
	"hw10/shared"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// handleInternalSet is called by the Leader to replicate a write to this node.
// The version is already determined by the Leader, so we use SetWithVersion.
// Assignment requires: Follower sleeps 100ms before responding.
func handleInternalSet(c *gin.Context) {
	var req shared.InternalSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Assignment-required delay: simulate real-world storage latency.
	time.Sleep(100 * time.Millisecond)

	store.SetWithVersion(req.Key, req.Value, req.Version)
	log.Printf("[%s] INTERNAL SET key=%s version=%d", config.Role, req.Key, req.Version)

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleInternalGet is called by the Leader when R > 1.
// It reads the local value and returns it (with version).
// Assignment requires: Follower sleeps 50ms before responding.
func handleInternalGet(c *gin.Context) {
	key := c.Param("key")

	// Assignment-required delay: simulate read latency.
	time.Sleep(50 * time.Millisecond)

	entry, found := store.Get(key)
	if !found {
		c.JSON(http.StatusNotFound, shared.InternalGetResponse{
			Key: key, Found: false,
		})
		return
	}

	c.JSON(http.StatusOK, shared.InternalGetResponse{
		Key:     key,
		Value:   entry.Value,
		Version: entry.Version,
		Found:   true,
	})
}

// handleLocalRead returns this node's local value WITHOUT any
// replication or coordination logic. This is the "sneaky" testing
// endpoint used to expose the inconsistency window.
// No artificial delay is added here — we want to read as fast as
// possible to catch the window where a follower hasn't been updated yet.
func handleLocalRead(c *gin.Context) {
	key := c.Param("key")

	entry, found := store.Get(key)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}

	c.JSON(http.StatusOK, shared.GetResponse{
		Key:     key,
		Value:   entry.Value,
		Version: entry.Version,
	})
}
