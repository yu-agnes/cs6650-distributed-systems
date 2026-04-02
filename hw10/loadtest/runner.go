package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// TestConfig holds all settings for a load test run.
type TestConfig struct {
	Mode         string        // "leader" or "leaderless"
	WritePercent int           // 0-100
	Duration     time.Duration // how long to run
	Concurrency  int           // number of goroutines
	NumKeys      int           // key space size (smaller = more overlap)
	LeaderURL    string        // where to send writes in leader mode
	Nodes        []string      // all node URLs
	OutPrefix    string        // output file prefix
}

// RequestResult records one request's outcome.
type RequestResult struct {
	Type      string        // "read" or "write"
	Key       string        // which key
	Version   int           // version returned
	Latency   time.Duration // how long the request took
	Stale     bool          // was this read stale?
	Timestamp time.Time     // when the request was made
}

// versionTracker keeps track of the latest known version per key.
// Used to detect stale reads: if a read returns a version lower
// than what we've already seen for that key, it's stale.
type versionTracker struct {
	mu       sync.RWMutex
	versions map[string]int
}

func newVersionTracker() *versionTracker {
	return &versionTracker{versions: make(map[string]int)}
}

func (vt *versionTracker) update(key string, version int) {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	if version > vt.versions[key] {
		vt.versions[key] = version
	}
}

func (vt *versionTracker) isStale(key string, version int) bool {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return version < vt.versions[key]
}

// runLoadTest executes the load test and returns all results.
func runLoadTest(cfg TestConfig) []RequestResult {
	var (
		allResults []RequestResult
		mu         sync.Mutex
		wg         sync.WaitGroup
		tracker    = newVersionTracker()
	)

	deadline := time.Now().Add(cfg.Duration)

	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))
			client := &http.Client{Timeout: 5 * time.Second}

			for time.Now().Before(deadline) {
				// Pick a random key from the small key space.
				key := fmt.Sprintf("key_%d", rng.Intn(cfg.NumKeys))

				// Decide: write or read?
				isWrite := rng.Intn(100) < cfg.WritePercent

				var result RequestResult
				if isWrite {
					result = doWrite(client, cfg, key, rng)
					if result.Version > 0 {
						tracker.update(key, result.Version)
					}
				} else {
					result = doRead(client, cfg, key, rng)
					if result.Version > 0 {
						result.Stale = tracker.isStale(key, result.Version)
					}
				}

				mu.Lock()
				allResults = append(allResults, result)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	return allResults
}

// doWrite sends a SET request and measures latency.
func doWrite(client *http.Client, cfg TestConfig, key string, rng *rand.Rand) RequestResult {
	value := fmt.Sprintf("v_%d", rng.Intn(1000000))

	// In leader mode, writes always go to the leader.
	// In leaderless mode, writes go to a random node (that node becomes coordinator).
	var targetURL string
	if cfg.Mode == "leader" {
		targetURL = cfg.LeaderURL
	} else {
		targetURL = cfg.Nodes[rng.Intn(len(cfg.Nodes))]
	}

	body, _ := json.Marshal(map[string]string{"key": key, "value": value})

	start := time.Now()
	resp, err := client.Post(targetURL+"/set", "application/json", bytes.NewReader(body))
	latency := time.Since(start)
	ts := time.Now()

	if err != nil {
		return RequestResult{Type: "write", Key: key, Latency: latency, Timestamp: ts}
	}
	defer resp.Body.Close()

	var sr struct {
		Version int `json:"version"`
	}
	json.NewDecoder(resp.Body).Decode(&sr)

	return RequestResult{
		Type:      "write",
		Key:       key,
		Version:   sr.Version,
		Latency:   latency,
		Timestamp: ts,
	}
}

// doRead sends a GET request and measures latency.
func doRead(client *http.Client, cfg TestConfig, key string, rng *rand.Rand) RequestResult {
	// In leader mode, reads can go to any node.
	// In leaderless mode, reads also go to any node (R=1, local read).
	targetURL := cfg.Nodes[rng.Intn(len(cfg.Nodes))]

	start := time.Now()
	resp, err := client.Get(targetURL + "/get/" + key)
	latency := time.Since(start)
	ts := time.Now()

	if err != nil {
		return RequestResult{Type: "read", Key: key, Latency: latency, Timestamp: ts}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return RequestResult{Type: "read", Key: key, Latency: latency, Timestamp: ts}
	}

	var gr struct {
		Value   string `json:"value"`
		Version int    `json:"version"`
	}
	json.NewDecoder(resp.Body).Decode(&gr)

	return RequestResult{
		Type:      "read",
		Key:       key,
		Version:   gr.Version,
		Latency:   latency,
		Timestamp: ts,
	}
}
