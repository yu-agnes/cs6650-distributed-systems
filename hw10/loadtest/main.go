package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	// --- CLI flags ---
	mode := flag.String("mode", "leader", "Database mode: leader or leaderless")
	writePercent := flag.Int("write-pct", 50, "Write percentage (e.g. 50 means 50% writes, 50% reads)")
	duration := flag.Int("duration", 30, "Test duration in seconds")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent workers")
	numKeys := flag.Int("keys", 20, "Number of distinct keys (smaller = more read/write overlap)")
	leaderURL := flag.String("leader", "http://localhost:8081", "Leader URL (for leader mode writes)")
	nodesStr := flag.String("nodes", "http://localhost:8081,http://localhost:8082,http://localhost:8083,http://localhost:8084,http://localhost:8085",
		"Comma-separated node URLs")
	outPrefix := flag.String("out", "results", "Output CSV file prefix")
	flag.Parse()

	nodes := strings.Split(*nodesStr, ",")

	cfg := TestConfig{
		Mode:         *mode,
		WritePercent: *writePercent,
		Duration:     time.Duration(*duration) * time.Second,
		Concurrency:  *concurrency,
		NumKeys:      *numKeys,
		LeaderURL:    *leaderURL,
		Nodes:        nodes,
		OutPrefix:    *outPrefix,
	}

	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║          KV Database Load Tester         ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Printf("  Mode:        %s\n", cfg.Mode)
	fmt.Printf("  Write%%:      %d%% writes / %d%% reads\n", cfg.WritePercent, 100-cfg.WritePercent)
	fmt.Printf("  Duration:    %s\n", cfg.Duration)
	fmt.Printf("  Concurrency: %d workers\n", cfg.Concurrency)
	fmt.Printf("  Keys:        %d distinct keys\n", cfg.NumKeys)
	fmt.Printf("  Leader:      %s\n", cfg.LeaderURL)
	fmt.Printf("  Nodes:       %v\n", cfg.Nodes)
	fmt.Println()

	results := runLoadTest(cfg)

	// Write results to CSV files.
	writeLatencyCSV(cfg.OutPrefix+"_latency.csv", results)
	writeStaleCSV(cfg.OutPrefix+"_stale.csv", results)
	writeRWIntervalCSV(cfg.OutPrefix+"_rw_interval.csv", results)

	// Print summary.
	printSummary(results)

	log.Printf("Results written to %s_latency.csv, %s_stale.csv, %s_rw_interval.csv",
		cfg.OutPrefix, cfg.OutPrefix, cfg.OutPrefix)
}

// writeLatencyCSV writes one row per request: type (read/write), latency_ms.
func writeLatencyCSV(filename string, results []RequestResult) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Cannot create %s: %v", filename, err)
	}
	defer f.Close()

	fmt.Fprintln(f, "type,latency_ms,key,version,stale")
	for _, r := range results {
		fmt.Fprintf(f, "%s,%.2f,%s,%d,%t\n",
			r.Type, float64(r.Latency.Microseconds())/1000.0,
			r.Key, r.Version, r.Stale)
	}
}

// writeStaleCSV writes summary of stale reads.
func writeStaleCSV(filename string, results []RequestResult) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Cannot create %s: %v", filename, err)
	}
	defer f.Close()

	totalReads := 0
	staleReads := 0
	for _, r := range results {
		if r.Type == "read" {
			totalReads++
			if r.Stale {
				staleReads++
			}
		}
	}

	fmt.Fprintln(f, "total_reads,stale_reads,stale_pct")
	stalePct := 0.0
	if totalReads > 0 {
		stalePct = float64(staleReads) / float64(totalReads) * 100
	}
	fmt.Fprintf(f, "%d,%d,%.2f\n", totalReads, staleReads, stalePct)
}

// writeRWIntervalCSV writes the time gap between a write and the
// next read on the same key. This shows the "locality in time" of
// the load generator and helps explain stale read frequency.
func writeRWIntervalCSV(filename string, results []RequestResult) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Cannot create %s: %v", filename, err)
	}
	defer f.Close()

	// Track the last write timestamp per key.
	lastWrite := make(map[string]time.Time)
	fmt.Fprintln(f, "key,interval_ms")

	for _, r := range results {
		if r.Type == "write" {
			lastWrite[r.Key] = r.Timestamp
		} else if r.Type == "read" {
			if wt, ok := lastWrite[r.Key]; ok {
				interval := r.Timestamp.Sub(wt)
				fmt.Fprintf(f, "%s,%.2f\n", r.Key, float64(interval.Microseconds())/1000.0)
			}
		}
	}
}

func printSummary(results []RequestResult) {
	var readCount, writeCount, staleCount int
	var readLatencySum, writeLatencySum time.Duration

	for _, r := range results {
		switch r.Type {
		case "read":
			readCount++
			readLatencySum += r.Latency
			if r.Stale {
				staleCount++
			}
		case "write":
			writeCount++
			writeLatencySum += r.Latency
		}
	}

	fmt.Println("─── Summary ───────────────────────────────")
	fmt.Printf("  Total requests:  %d\n", len(results))
	fmt.Printf("  Writes:          %d", writeCount)
	if writeCount > 0 {
		fmt.Printf(" (avg %.1fms)", float64(writeLatencySum.Microseconds())/float64(writeCount)/1000.0)
	}
	fmt.Println()
	fmt.Printf("  Reads:           %d", readCount)
	if readCount > 0 {
		fmt.Printf(" (avg %.1fms)", float64(readLatencySum.Microseconds())/float64(readCount)/1000.0)
	}
	fmt.Println()
	fmt.Printf("  Stale reads:     %d", staleCount)
	if readCount > 0 {
		fmt.Printf(" (%.1f%%)", float64(staleCount)/float64(readCount)*100)
	}
	fmt.Println()
	fmt.Println("────────────────────────────────────────────")
}
