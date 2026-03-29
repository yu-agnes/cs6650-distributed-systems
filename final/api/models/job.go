package models

import "time"

// Job represents a scraping job stored in DynamoDB
type Job struct {
	JobID       string         `json:"jobID" dynamodbav:"jobID"`
	Status      string         `json:"status" dynamodbav:"status"`           // pending, processing, completed, failed
	URLs        []string       `json:"urls" dynamodbav:"urls"`
	Results     []ScrapeResult `json:"results,omitempty" dynamodbav:"results,omitempty"`
	TotalURLs   int            `json:"totalUrls" dynamodbav:"totalUrls"`
	CompletedAt string         `json:"completedAt,omitempty" dynamodbav:"completedAt,omitempty"`
	CreatedAt   string         `json:"createdAt" dynamodbav:"createdAt"`
}

// ScrapeResult holds the extracted data from one URL
type ScrapeResult struct {
	URL          string  `json:"url" dynamodbav:"url"`
	Title        string  `json:"title" dynamodbav:"title"`
	WordCount    int     `json:"wordCount" dynamodbav:"wordCount"`
	LinkCount    int     `json:"linkCount" dynamodbav:"linkCount"`
	ResponseTime float64 `json:"responseTimeMs" dynamodbav:"responseTimeMs"`
	Error        string  `json:"error,omitempty" dynamodbav:"error,omitempty"`
}

// SQSMessage is what gets sent to SQS
type SQSMessage struct {
	JobID string   `json:"jobID"`
	URLs  []string `json:"urls"`
}

func NewJob(jobID string, urls []string) *Job {
	return &Job{
		JobID:     jobID,
		Status:    "pending",
		URLs:      urls,
		TotalURLs: len(urls),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}
