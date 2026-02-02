package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var m sync.Map

	var wg sync.WaitGroup

	start := time.Now()

	// Spawn 50 goroutines
	for g := 0; g < 50; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			// Each goroutine writes 1000 key-value pairs
			for i := 0; i < 1000; i++ {
				m.Store(g*1000+i, i)
			}
		}(g)
	}

	wg.Wait()

	elapsed := time.Since(start)

	// Count entries using Range
	count := 0
	m.Range(func(key, value any) bool {
		count++
		return true // continue iteration
	})

	fmt.Println("Expected length:", 50*1000)
	fmt.Println("Actual length:", count)
	fmt.Println("Time taken:", elapsed)
}