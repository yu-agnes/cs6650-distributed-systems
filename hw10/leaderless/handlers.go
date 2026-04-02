package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hw10/shared"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// handleSet processes a client write request.
// In leaderless mode, whichever node receives this becomes the
// Write Coordinator for this request.
//
// Assignment specifies W=N (all nodes must ack), R=1.
// So the coordinator writes locally, then replicates to ALL peers
// and waits for every one to confirm before returning 201.
func handleSet(c *gin.Context) {
	var req shared.SetRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request, key is required"})
		return
	}

	// Step 1: Write locally, get version.
	newVersion := store.Set(req.Key, req.Value)
	log.Printf("[Coordinator] SET key=%s value=%s version=%d", req.Key, req.Value, newVersion)

	// Step 2: Replicate to all peers (W=N means we need all of them).
	acksNeeded := len(config.Peers) // all other nodes
	ackCh := make(chan bool, acksNeeded)

	for _, peer := range config.Peers {
		go func(p string) {
			time.Sleep(200 * time.Millisecond) // assignment-required delay
			ok := sendInternalSet(p, req.Key, req.Value, newVersion)
			ackCh <- ok
		}(peer)
	}

	// Wait for all peers to respond.
	acks := 0
	for i := 0; i < acksNeeded; i++ {
		if <-ackCh {
			acks++
		}
	}

	if acks == acksNeeded {
		c.JSON(http.StatusCreated, gin.H{
			"status":  "created",
			"key":     req.Key,
			"version": newVersion,
		})
	} else {
		// Some peers failed, but we still created locally.
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("only %d/%d peers acked", acks, acksNeeded),
		})
	}
}

// handleGet processes a client read request.
// R=1 in leaderless mode: just return this node's local value.
// This is where the inconsistency window can be observed —
// if a write is in progress on another node (coordinator hasn't
// finished replicating), this node may return stale data.
func handleGet(c *gin.Context) {
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

// handleInternalSet is called by the Write Coordinator to replicate
// a write to this node. Same as leader-follower's internal set.
func handleInternalSet(c *gin.Context) {
	var req shared.InternalSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Assignment-required delay.
	time.Sleep(100 * time.Millisecond)

	store.SetWithVersion(req.Key, req.Value, req.Version)
	log.Printf("[node] INTERNAL SET key=%s version=%d", req.Key, req.Version)

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleLocalRead returns this node's local value without any
// coordination. Used by unit tests to observe the inconsistency window.
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

// sendInternalSet sends a replicated write to one peer.
func sendInternalSet(peer, key, value string, version int) bool {
	url := fmt.Sprintf("http://%s/internal/set", peer)
	body, _ := json.Marshal(shared.InternalSetRequest{
		Key: key, Value: value, Version: version,
	})

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[Coordinator] Failed to replicate to %s: %v", peer, err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
