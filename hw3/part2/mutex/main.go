package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeMap wraps a map with a mutex
type SafeMap struct {
	mu sync.Mutex
	m  map[int]int
}

func main() {
	// Create SafeMap with initialized map
	sm := SafeMap{
		m: make(map[int]int),
	}

	var wg sync.WaitGroup

	start := time.Now()

	// Spawn 50 goroutines
	for g := 0; g < 50; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			// Each goroutine writes 1000 key-value pairs
			for i := 0; i < 1000; i++ {
				sm.mu.Lock()
				sm.m[g*1000+i] = i
				sm.mu.Unlock()
			}
		}(g)
	}

	wg.Wait()

	elapsed := time.Since(start)

	fmt.Println("Expected length:", 50*1000)
	fmt.Println("Actual length:", len(sm.m))
	fmt.Println("Time taken:", elapsed)
}
