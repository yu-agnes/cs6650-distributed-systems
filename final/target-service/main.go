package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	// Any path returns a fake HTML page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Simulate variable response time (50-500ms)
		delay := time.Duration(50+rand.Intn(450)) * time.Millisecond
		time.Sleep(delay)

		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "home"
		}

		// Generate fake HTML with random content
		wordCount := 100 + rand.Intn(400)
		linkCount := 5 + rand.Intn(15)
		title := fmt.Sprintf("Page: %s", path)

		words := generateWords(wordCount)
		links := generateLinks(linkCount)

		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>%s</title></head>
<body>
<h1>%s</h1>
<p>%s</p>
%s
</body>
</html>`, title, title, words, links)

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	})

	fmt.Printf("Target service listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}

var sampleWords = []string{
	"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
	"distributed", "system", "scalable", "cloud", "computing", "data",
	"network", "server", "client", "request", "response", "pipeline",
	"worker", "queue", "message", "process", "thread", "concurrent",
}

func generateWords(count int) string {
	words := make([]string, count)
	for i := range words {
		words[i] = sampleWords[rand.Intn(len(sampleWords))]
	}
	return strings.Join(words, " ")
}

func generateLinks(count int) string {
	var sb strings.Builder
	for i := 0; i < count; i++ {
		sb.WriteString(fmt.Sprintf(`<a href="/page-%d">Link %d</a>`+"\n", i, i))
	}
	return sb.String()
}
