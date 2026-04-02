package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// --- Config: adjust these if your ports are different ---
const (
	leaderURL    = "http://localhost:8081"
	follower1URL = "http://localhost:8082"
	follower2URL = "http://localhost:8083"
	follower3URL = "http://localhost:8084"
	follower4URL = "http://localhost:8085"
)

var followerURLs = []string{follower1URL, follower2URL, follower3URL, follower4URL}

// --- Response types ---
type SetResponse struct {
	Status  string `json:"status"`
	Key     string `json:"key"`
	Version int    `json:"version"`
}

type GetResponse struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int    `json:"version"`
}

// ──────────────────────────────────────────────────────────
// Test 1: Write to Leader, then read from Leader → consistent
// ──────────────────────────────────────────────────────────
func testLeaderReadConsistency() {
	fmt.Println("\n=== Test 1: Write → Read from Leader ===")

	version := doSet(leaderURL, "test1", "hello")
	fmt.Printf("  SET key=test1 value=hello → version=%d\n", version)

	val, ver := doGet(leaderURL, "test1")
	if val == "hello" && ver == version {
		fmt.Println("  ✅ PASS: Leader returned consistent data")
	} else {
		fmt.Printf("  ❌ FAIL: Expected hello/v%d, got %s/v%d\n", version, val, ver)
	}
}

// ──────────────────────────────────────────────────────────
// Test 2: Write to Leader, then read from Follower → consistent
// (Works because W=5 means all followers are updated before
//  the set returns)
// ──────────────────────────────────────────────────────────
func testFollowerReadConsistency() {
	fmt.Println("\n=== Test 2: Write → Read from Followers ===")

	version := doSet(leaderURL, "test2", "world")
	fmt.Printf("  SET key=test2 value=world → version=%d\n", version)

	allPass := true
	for i, fURL := range followerURLs {
		val, ver := doGet(fURL, "test2")
		if val == "world" && ver == version {
			fmt.Printf("  ✅ Follower %d: consistent (value=%s, version=%d)\n", i+1, val, ver)
		} else {
			fmt.Printf("  ❌ Follower %d: INCONSISTENT (value=%s, version=%d)\n", i+1, val, ver)
			allPass = false
		}
	}
	if allPass {
		fmt.Println("  ✅ PASS: All followers returned consistent data")
	}
}

// ──────────────────────────────────────────────────────────
// Test 3: Expose inconsistency window using local_read
//
// Strategy: send a SET to the Leader, and IMMEDIATELY (in parallel)
// send local_read requests to all Followers. Because the Leader
// sleeps 200ms between each follower notification, and each
// follower sleeps 100ms before writing, there is a window where
// some followers haven't been updated yet.
//
// We run this many times to increase the chance of catching it.
// ──────────────────────────────────────────────────────────
func testLocalReadInconsistencyWindow() {
	fmt.Println("\n=== Test 3: Expose inconsistency window via local_read ===")

	inconsistenciesSeen := 0
	rounds := 20

	for round := 0; round < rounds; round++ {
		key := fmt.Sprintf("window_%d", round)

		// First, write an initial value and wait for it to propagate.
		doSet(leaderURL, key, "old")
		time.Sleep(500 * time.Millisecond) // wait for full propagation

		// Now fire a new write AND local_reads at the same time.
		var wg sync.WaitGroup
		staleResults := make([]bool, len(followerURLs))

		// Start the write (don't wait for it to finish).
		wg.Add(1)
		go func() {
			defer wg.Done()
			doSet(leaderURL, key, "new")
		}()

		// Immediately blast local_read to all followers.
		// Small delay to let the Leader start processing but not finish.
		time.Sleep(50 * time.Millisecond)

		for i, fURL := range followerURLs {
			wg.Add(1)
			go func(idx int, url string) {
				defer wg.Done()
				val, _ := doLocalRead(url, key)
				if val == "old" {
					staleResults[idx] = true
				}
			}(i, fURL)
		}

		wg.Wait()

		for i, stale := range staleResults {
			if stale {
				inconsistenciesSeen++
				if inconsistenciesSeen <= 5 { // only print first 5 to keep output clean
					fmt.Printf("  ⚡ Round %d: Follower %d returned STALE data (old instead of new)\n",
						round, i+1)
				}
			}
		}
	}

	if inconsistenciesSeen > 0 {
		fmt.Printf("  ✅ PASS: Caught %d stale reads across %d rounds — inconsistency window confirmed!\n",
			inconsistenciesSeen, rounds)
	} else {
		fmt.Printf("  ⚠️  No stale reads caught in %d rounds (try increasing rounds or reducing delays)\n", rounds)
	}
}

// ──────────────────────────────────────────────────────────
// HTTP helper functions
// ──────────────────────────────────────────────────────────

func doSet(baseURL, key, value string) int {
	body, _ := json.Marshal(map[string]string{"key": key, "value": value})
	resp, err := http.Post(baseURL+"/set", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Printf("  ERROR: SET failed: %v\n", err)
		return -1
	}
	defer resp.Body.Close()

	var sr SetResponse
	json.NewDecoder(resp.Body).Decode(&sr)
	return sr.Version
}

func doGet(baseURL, key string) (string, int) {
	resp, err := http.Get(baseURL + "/get/" + key)
	if err != nil {
		fmt.Printf("  ERROR: GET failed: %v\n", err)
		return "", -1
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", 0
	}

	var gr GetResponse
	json.NewDecoder(resp.Body).Decode(&gr)
	return gr.Value, gr.Version
}

func doLocalRead(baseURL, key string) (string, int) {
	resp, err := http.Get(baseURL + "/local_read/" + key)
	if err != nil {
		return "", -1
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", 0
	}

	var gr GetResponse
	json.NewDecoder(resp.Body).Decode(&gr)
	return gr.Value, gr.Version
}

// ──────────────────────────────────────────────────────────
// Main
// ──────────────────────────────────────────────────────────

func main() {
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║   Leader-Follower Consistency Tests      ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println("Make sure the leader-follower cluster is running (W=5 R=1).")

	testLeaderReadConsistency()
	testFollowerReadConsistency()
	testLocalReadInconsistencyWindow()

	fmt.Println("\n✅ All tests completed.")
}
