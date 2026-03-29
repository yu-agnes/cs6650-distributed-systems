package scraper

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Result holds extracted data from one page
type Result struct {
	URL          string  `json:"url"            dynamodbav:"url"`
	Title        string  `json:"title"          dynamodbav:"title"`
	WordCount    int     `json:"wordCount"      dynamodbav:"wordCount"`
	LinkCount    int     `json:"linkCount"      dynamodbav:"linkCount"`
	ResponseTime float64 `json:"responseTimeMs" dynamodbav:"responseTimeMs"`
	Error        string  `json:"error,omitempty" dynamodbav:"error,omitempty"`
}

var (
	titleRegex = regexp.MustCompile(`<title>(.*?)</title>`)
	linkRegex  = regexp.MustCompile(`<a\s+[^>]*href=`)
)

// Scrape fetches a URL and extracts page metadata
func Scrape(targetURL string) Result {
	start := time.Now()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(targetURL)
	if err != nil {
		return Result{
			URL:          targetURL,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("fetch error: %v", err),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{
			URL:          targetURL,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("read error: %v", err),
		}
	}

	elapsed := float64(time.Since(start).Milliseconds())
	html := string(body)

	// Extract title
	title := ""
	if matches := titleRegex.FindStringSubmatch(html); len(matches) > 1 {
		title = matches[1]
	}

	// Count words (simple: split by whitespace in body text)
	// Strip HTML tags first
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	textOnly := tagRegex.ReplaceAllString(html, " ")
	words := strings.Fields(textOnly)
	wordCount := len(words)

	// Count links
	linkCount := len(linkRegex.FindAllString(html, -1))

	return Result{
		URL:          targetURL,
		Title:        title,
		WordCount:    wordCount,
		LinkCount:    linkCount,
		ResponseTime: elapsed,
	}
}
