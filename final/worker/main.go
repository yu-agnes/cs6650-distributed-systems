package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"worker/scraper"
	"worker/services"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// SQSMessage matches what the API sends
type SQSMessage struct {
	JobID string   `json:"jobID"`
	URLs  []string `json:"urls"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGTERM (ECS sends this)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal, finishing current job...")
		cancel()
	}()

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	sqsClient := sqs.NewFromConfig(cfg)
	dynamoClient := dynamodb.NewFromConfig(cfg)

	sqsSvc := services.NewSQSService(sqsClient, os.Getenv("SQS_QUEUE_URL"))
	dynamoSvc := services.NewDynamoService(dynamoClient, os.Getenv("DYNAMODB_TABLE"))

	targetBaseURL := os.Getenv("TARGET_SERVICE_URL")
	if targetBaseURL == "" {
		log.Fatal("TARGET_SERVICE_URL is required")
	}

	log.Printf("Worker started. Polling SQS, target: %s\n", targetBaseURL)

	// Main poll loop
	for {
		if ctx.Err() != nil {
			log.Println("Context cancelled, exiting poll loop")
			break
		}

		messages, err := sqsSvc.PollMessages(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Printf("Error polling SQS: %v\n", err)
			continue
		}

		if len(messages) == 0 {
			continue
		}

		for _, msg := range messages {
			if msg.Body == nil || msg.ReceiptHandle == nil {
				continue
			}
			processJob(ctx, *msg.Body, *msg.ReceiptHandle, sqsSvc, dynamoSvc, targetBaseURL)
		}
	}

	log.Println("Worker shutdown complete")
}

func processJob(ctx context.Context, body string, receiptHandle string, sqsSvc *services.SQSService, dynamoSvc *services.DynamoService, targetBaseURL string) {
	var msg SQSMessage
	if err := json.Unmarshal([]byte(body), &msg); err != nil {
		log.Printf("Failed to unmarshal message: %v\n", err)
		return
	}

	log.Printf("Processing job %s with %d URLs\n", msg.JobID, len(msg.URLs))

	// Update status to "processing"
	if err := dynamoSvc.UpdateJobProcessing(ctx, msg.JobID); err != nil {
		log.Printf("Failed to update job status: %v\n", err)
		return
	}

	// Scrape each URL sequentially
	results := make([]scraper.Result, 0, len(msg.URLs))
	for _, urlPath := range msg.URLs {
		fullURL := targetBaseURL
		if !strings.HasSuffix(fullURL, "/") {
			fullURL += "/"
		}
		fullURL += strings.TrimPrefix(urlPath, "/")

		result := scraper.Scrape(fullURL)
		results = append(results, result)

		log.Printf("  Scraped %s: title=%q words=%d links=%d time=%.0fms\n",
			urlPath, result.Title, result.WordCount, result.LinkCount, result.ResponseTime)
	}

	// Write results to DynamoDB
	if err := dynamoSvc.UpdateJobCompleted(ctx, msg.JobID, results); err != nil {
		log.Printf("Failed to write results for job %s: %v\n", msg.JobID, err)
		return
	}

	// Delete message from SQS (acknowledge completion)
	if err := sqsSvc.DeleteMessage(ctx, receiptHandle); err != nil {
		log.Printf("Failed to delete message for job %s: %v\n", msg.JobID, err)
		return
	}

	log.Printf("Job %s completed: %d URLs scraped\n", msg.JobID, len(results))
}
