package main

import (
	"fmt"
	"sync"
)

func main() {
	// Create a plain map
	m := make(map[int]int)

	var wg sync.WaitGroup

	// Spawn 50 goroutines
	for g := 0; g < 50; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			// Each goroutine writes 1000 key-value pairs
			for i := 0; i < 1000; i++ {
				m[g*1000+i] = i
			}
		}(g)
	}

	wg.Wait()

	fmt.Println("Expected length:", 50*1000)
	fmt.Println("Actual length:", len(m))
}
