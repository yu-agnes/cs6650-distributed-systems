package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func main() {
	const iterations = 100000
	line := []byte("Hello, this is a test line for file writing.\n")

	// ========== Unbuffered Write ==========
	f1, err := os.Create("unbuffered.txt")
	if err != nil {
		panic(err)
	}

	start1 := time.Now()
	for i := 0; i < iterations; i++ {
		f1.Write(line)
	}
	f1.Close()
	unbufferedTime := time.Since(start1)

	// ========== Buffered Write ==========
	f2, err := os.Create("buffered.txt")
	if err != nil {
		panic(err)
	}

	w := bufio.NewWriter(f2)

	start2 := time.Now()
	for i := 0; i < iterations; i++ {
		w.WriteString(string(line))
	}
	w.Flush()
	f2.Close()
	bufferedTime := time.Since(start2)

	// ========== Results ==========
	fmt.Println("Iterations:", iterations)
	fmt.Println("Unbuffered time:", unbufferedTime)
	fmt.Println("Buffered time:  ", bufferedTime)
}
