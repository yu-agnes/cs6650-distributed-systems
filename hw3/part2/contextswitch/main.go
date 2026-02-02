package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	const iterations = 1000000

	// ========== Single Thread Mode ==========
	runtime.GOMAXPROCS(1)

	ch1 := make(chan struct{})
	ch2 := make(chan struct{})
	done := make(chan struct{})

	// Goroutine A
	go func() {
		for i := 0; i < iterations; i++ {
			ch1 <- struct{}{}
			<-ch2
		}
		done <- struct{}{}
	}()

	// Goroutine B
	go func() {
		for i := 0; i < iterations; i++ {
			<-ch1
			ch2 <- struct{}{}
		}
	}()

	start1 := time.Now()
	<-done
	singleThreadTime := time.Since(start1)

	// ========== Multi Thread Mode ==========
	runtime.GOMAXPROCS(runtime.NumCPU())

	ch3 := make(chan struct{})
	ch4 := make(chan struct{})
	done2 := make(chan struct{})

	// Goroutine A
	go func() {
		for i := 0; i < iterations; i++ {
			ch3 <- struct{}{}
			<-ch4
		}
		done2 <- struct{}{}
	}()

	// Goroutine B
	go func() {
		for i := 0; i < iterations; i++ {
			<-ch3
			ch4 <- struct{}{}
		}
	}()

	start2 := time.Now()
	<-done2
	multiThreadTime := time.Since(start2)

	// ========== Results ==========
	fmt.Println("Iterations:", iterations)
	fmt.Println("Single thread (GOMAXPROCS=1):", singleThreadTime)
	fmt.Printf("Average switch time: %v\n", singleThreadTime/(2*iterations))
	fmt.Println()
	fmt.Println("Multi thread (GOMAXPROCS=CPU):", multiThreadTime)
	fmt.Printf("Average switch time: %v\n", multiThreadTime/(2*iterations))
}
