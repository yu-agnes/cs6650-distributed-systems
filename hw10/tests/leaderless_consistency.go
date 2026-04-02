package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// --- Config: same ports, but all nodes are equal ---
const (
	node1URL = "http://localhost:8081"
	node2URL = "http://localhost:8082"
	node3URL = "http://localhost:8083"
	node4URL = "http://localhost:8084"
	node5URL = "http://localhost:8085"
)

var allNodeURLs = []string{node1URL, node2URL, node3URL, node4URL, node5URL}

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
// Test 1: Expose inconsistency window
//
// Write to node1 (it becomes Coordinator). DURING the write,
// immediately read from other nodes via get. Since R=1 returns
// local data, and the Coordinator hasn't finished replicating
// yet, other nodes should return stale or missing data.
// ──────────────────────────────────────────────────────────
func testLeaderlessInconsistencyWindow() {
	fmt.Println("\n=== Test 1: Expose inconsistency window during write ===")

	inconsistenciesSeen := 0
	rounds := 20

	for round := 0; round < rounds; round++ {
		key := fmt.Sprintf("ll_window_%d", round)
		coordinatorURL := allNodeURLs[round%len(allNodeURLs)] // rotate coordinator

		// Write initial value, wait for full propagation.
		doSet(coordinatorURL, key, "old")
		time.Sleep(500 * time.Millisecond)

		// Fire new write AND reads simultaneously.
		var wg sync.WaitGroup
		staleResults := make([]bool, len(allNodeURLs))

		// Start write on coordinator (don't wait).
		wg.Add(1)
		go func() {
			defer wg.Done()
			doSet(coordinatorURL, key, "new")
		}()

		// Small delay to let coordinator start but not finish.
		time.Sleep(50 * time.Millisecond)

		// Read from ALL nodes (including coordinator).
		for i, nURL := range allNodeURLs {
			wg.Add(1)
			go func(idx int, url string) {
				defer wg.Done()
				val, _ := doGet(url, key)
				if val == "old" {
					staleResults[idx] = true
				}
			}(i, nURL)
		}

		wg.Wait()

		for i, stale := range staleResults {
			if stale {
				inconsistenciesSeen++
				if inconsistenciesSeen <= 5 {
					fmt.Printf("  ⚡ Round %d: Node %d returned STALE data\n", round, i+1)
				}
			}
		}
	}

	if inconsistenciesSeen > 0 {
		fmt.Printf("  ✅ PASS: Caught %d stale reads — inconsistency window confirmed!\n",
			inconsistenciesSeen)
	} else {
		fmt.Printf("  ⚠️  No stale reads caught in %d rounds\n", rounds)
	}
}

// ──────────────────────────────────────────────────────────
// Test 2: After Coordinator confirms write, read from
// Coordinator → should be consistent
// ──────────────────────────────────────────────────────────
func testCoordinatorReadAfterWrite() {
	fmt.Println("\n=== Test 2: Write → Read from Coordinator (after ack) ===")

	version := doSet(node1URL, "ll_test2", "consistent_value")
	fmt.Printf("  SET key=ll_test2 → version=%d\n", version)

	val, ver := doGet(node1URL, "ll_test2")
	if val == "consistent_value" && ver == version {
		fmt.Println("  ✅ PASS: Coordinator returned consistent data")
	} else {
		fmt.Printf("  ❌ FAIL: Expected consistent_value/v%d, got %s/v%d\n", version, val, ver)
	}
}

// ──────────────────────────────────────────────────────────
// Test 3: After Coordinator confirms write (W=N), read from
// another node → should be consistent (all nodes were updated)
// ──────────────────────────────────────────────────────────
func testOtherNodeReadAfterWrite() {
	fmt.Println("\n=== Test 3: Write → Read from other node (after ack) ===")

	version := doSet(node1URL, "ll_test3", "all_synced")
	fmt.Printf("  SET key=ll_test3 → version=%d\n", version)

	allPass := true
	for i := 1; i < len(allNodeURLs); i++ {
		val, ver := doGet(allNodeURLs[i], "ll_test3")
		if val == "all_synced" && ver == version {
			fmt.Printf("  ✅ Node %d: consistent (value=%s, version=%d)\n", i+1, val, ver)
		} else {
			fmt.Printf("  ❌ Node %d: INCONSISTENT (value=%s, version=%d)\n", i+1, val, ver)
			allPass = false
		}
	}
	if allPass {
		fmt.Println("  ✅ PASS: All nodes consistent after Coordinator ack")
	}
}

// ──────────────────────────────────────────────────────────
// HTTP helpers
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
	fmt.Println("║   Leaderless Consistency Tests           ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println("Make sure the leaderless cluster is running.")

	testLeaderlessInconsistencyWindow()
	testCoordinatorReadAfterWrite()
	testOtherNodeReadAfterWrite()

	fmt.Println("\n✅ All tests completed.")
}
