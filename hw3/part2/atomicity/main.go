package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

func main() {
	// Experiment 1: Normal integer (will have race condition)
	var normalCounter int64 = 0

	// Experiment 2: Atomic integer (thread-safe)
	var atomicCounter int64 = 0

	var wg sync.WaitGroup

	// Spawn 50 goroutines, each increments 1000 times
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				// Unsafe increment
				normalCounter++

				// Safe atomic increment
				atomic.AddInt64(&atomicCounter, 1)
			}
		}()
	}

	wg.Wait()

	fmt.Println("Expected value:", 50*1000)
	fmt.Println("Normal counter:", normalCounter)
	fmt.Println("Atomic counter:", atomicCounter)
}
