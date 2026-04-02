package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hw10/shared"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// handleSet processes a client write request.
// Only the Leader should receive these (the load tester sends writes to the Leader).
//
// Behavior depends on W:
//   - W=1: write locally, return immediately, replicate async
//   - W=5: write locally, replicate to ALL followers synchronously, then return
//   - W=3: write locally (counts as 1), replicate to followers concurrently,
//     return as soon as W total acks are collected
func handleSet(c *gin.Context) {
	var req shared.SetRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request, key is required"})
		return
	}

	// Step 1: Write locally and get the new version number.
	newVersion := store.Set(req.Key, req.Value)
	log.Printf("[Leader] SET key=%s value=%s version=%d", req.Key, req.Value, newVersion)

	// Step 2: Replicate based on W.
	if config.W <= 1 {
		// W=1: return to client immediately, replicate in background.
		go replicateToFollowers(req.Key, req.Value, newVersion)
		c.JSON(http.StatusCreated, gin.H{
			"status":  "created",
			"key":     req.Key,
			"version": newVersion,
		})
		return
	}

	// W>1: we need (W - 1) follower acks (leader itself counts as 1).
	acksNeeded := config.W - 1
	acks := replicateAndCollectAcks(req.Key, req.Value, newVersion, acksNeeded)

	if acks >= acksNeeded {
		c.JSON(http.StatusCreated, gin.H{
			"status":  "created",
			"key":     req.Key,
			"version": newVersion,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("only %d/%d followers acked", acks, acksNeeded),
		})
	}
}

// handleGet processes a client read request.
//
// Behavior depends on R:
//   - R=1: return local value immediately
//   - R>1: query followers concurrently, collect R responses total
//     (including local), return the one with the highest version
func handleGet(c *gin.Context) {
	key := c.Param("key")

	if config.R <= 1 {
		// R=1: just read locally.
		entry, found := store.Get(key)
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
			return
		}
		c.JSON(http.StatusOK, shared.GetResponse{
			Key: key, Value: entry.Value, Version: entry.Version,
		})
		return
	}

	// R>1: read from local + followers, pick the highest version.
	best := collectReads(key, config.R)
	if best == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}
	c.JSON(http.StatusOK, shared.GetResponse{
		Key: key, Value: best.Value, Version: best.Version,
	})
}

// ─── Replication helpers ────────────────────────────────────────────

// replicateToFollowers sends the write to all followers (fire-and-forget).
// Used when W=1 (async replication).
func replicateToFollowers(key, value string, version int) {
	for _, peer := range config.Peers {
		time.Sleep(200 * time.Millisecond) // assignment-required delay
		go sendInternalSet(peer, key, value, version)
	}
}

// replicateAndCollectAcks sends the write to all followers concurrently
// and waits until `needed` of them acknowledge, or all have responded.
// Used when W>1 (synchronous / quorum replication).
func replicateAndCollectAcks(key, value string, version int, needed int) int {
	ackCh := make(chan bool, len(config.Peers))

	for _, peer := range config.Peers {
		go func(p string) {
			time.Sleep(200 * time.Millisecond) // assignment-required delay
			ok := sendInternalSet(p, key, value, version)
			ackCh <- ok
		}(peer)
	}

	acks := 0
	total := 0
	for total < len(config.Peers) {
		if <-ackCh {
			acks++
		}
		total++
		if acks >= needed {
			return acks
		}
	}
	return acks
}

// sendInternalSet sends a replicated write to one follower.
// Returns true if the follower responded 200/201.
func sendInternalSet(peer, key, value string, version int) bool {
	url := fmt.Sprintf("http://%s/internal/set", peer)
	body, _ := json.Marshal(shared.InternalSetRequest{
		Key: key, Value: value, Version: version,
	})

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[Leader] Failed to replicate to %s: %v", peer, err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// collectReads queries local + followers for a key, collects up to `r` responses,
// and returns the entry with the highest version. Returns nil if not found anywhere.
func collectReads(key string, r int) *shared.Entry {
	type result struct {
		entry shared.Entry
		found bool
	}

	// Start with local read.
	localEntry, localFound := store.Get(key)

	// We need (r - 1) more reads from followers.
	resCh := make(chan result, len(config.Peers))
	var wg sync.WaitGroup

	for _, peer := range config.Peers {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			entry, found := sendInternalGet(p, key)
			resCh <- result{entry: entry, found: found}
		}(peer)
	}

	// Close channel when all goroutines finish.
	go func() {
		wg.Wait()
		close(resCh)
	}()

	// Collect results: start with local, then add follower responses.
	var best *shared.Entry
	collected := 0

	if localFound {
		best = &localEntry
		collected++
	} else {
		collected++ // local counts as a read even if not found
	}

	for res := range resCh {
		collected++
		if res.found {
			if best == nil || res.entry.Version > best.Version {
				e := res.entry // copy to avoid pointer reuse
				best = &e
			}
		}
		if collected >= r {
			break
		}
	}

	return best
}

// sendInternalGet queries one follower for a key's value.
func sendInternalGet(peer, key string) (shared.Entry, bool) {
	url := fmt.Sprintf("http://%s/internal/get/%s", peer, key)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[Leader] Failed to read from %s: %v", peer, err)
		return shared.Entry{}, false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return shared.Entry{}, false
	}

	var ir shared.InternalGetResponse
	if err := json.NewDecoder(resp.Body).Decode(&ir); err != nil {
		return shared.Entry{}, false
	}
	return shared.Entry{Value: ir.Value, Version: ir.Version}, ir.Found
}
