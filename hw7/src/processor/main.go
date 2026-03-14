package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
)

// ==================== Data Models (same as receiver) ====================

type Item struct {
	ProductID int     `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type Order struct {
	OrderID    string    `json:"order_id"`
	CustomerID int       `json:"customer_id"`
	Status     string    `json:"status"`
	Items      []Item    `json:"items"`
	CreatedAt  time.Time `json:"created_at"`
}

// ==================== SQS Client ====================

var sqsClient *sqs.Client
var sqsQueueURL string

// Counters for monitoring
var (
	totalProcessed int64
	totalFailed    int64
)

func initSQS() {
	sqsQueueURL = os.Getenv("SQS_QUEUE_URL")
	if sqsQueueURL == "" {
		log.Fatal("[ERROR] SQS_QUEUE_URL environment variable is required")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		log.Fatalf("[ERROR] Failed to load AWS config: %v", err)
	}

	sqsClient = sqs.NewFromConfig(cfg)
	log.Printf("[SQS] Initialized, queue URL: %s", sqsQueueURL)
}

// ==================== Payment Processing ====================

// simulatePayment simulates a 3-second payment verification
func simulatePayment(orderID string) error {
	log.Printf("[PAYMENT] Processing order %s (3s delay)...", orderID)
	time.Sleep(3 * time.Second)
	log.Printf("[PAYMENT] Order %s completed", orderID)
	return nil
}

// processMessage handles a single SQS message
func processMessage(msg string, receiptHandle string) {
	// SNS wraps the message in its own JSON envelope
	// We need to extract the actual order from the SNS "Message" field
	var snsEnvelope struct {
		Message string `json:"Message"`
	}

	var order Order

	// Try to parse as SNS envelope first
	if err := json.Unmarshal([]byte(msg), &snsEnvelope); err == nil && snsEnvelope.Message != "" {
		// It's an SNS envelope, parse the inner message
		if err := json.Unmarshal([]byte(snsEnvelope.Message), &order); err != nil {
			log.Printf("[ERROR] Failed to parse order from SNS message: %v", err)
			atomic.AddInt64(&totalFailed, 1)
			return
		}
	} else {
		// Try parsing as a direct order (for testing)
		if err := json.Unmarshal([]byte(msg), &order); err != nil {
			log.Printf("[ERROR] Failed to parse order: %v", err)
			atomic.AddInt64(&totalFailed, 1)
			return
		}
	}

	// Process the payment (3 second delay)
	err := simulatePayment(order.OrderID)
	if err != nil {
		log.Printf("[ERROR] Payment failed for order %s: %v", order.OrderID, err)
		atomic.AddInt64(&totalFailed, 1)
		return
	}

	// Delete message from SQS after successful processing
	_, err = sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
		QueueUrl:      &sqsQueueURL,
		ReceiptHandle: &receiptHandle,
	})
	if err != nil {
		log.Printf("[ERROR] Failed to delete message: %v", err)
	}

	atomic.AddInt64(&totalProcessed, 1)
}

// ==================== Worker Loop ====================

// pollAndProcess continuously polls SQS and processes messages
// Each worker goroutine runs this function independently
func pollAndProcess(workerID int) {
	log.Printf("[WORKER %d] Started polling SQS", workerID)

	for {
		// Long polling: wait up to 20 seconds for messages, receive up to 10
		result, err := sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
			QueueUrl:            &sqsQueueURL,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20, // Long polling
		})
		if err != nil {
			log.Printf("[WORKER %d] Error receiving messages: %v", workerID, err)
			time.Sleep(1 * time.Second) // Brief pause before retry
			continue
		}

		// Process each message in its own goroutine
		for _, msg := range result.Messages {
			processMessage(*msg.Body, *msg.ReceiptHandle)
		}
	}
}

// ==================== Main ====================

func main() {
	// Initialize SQS client
	initSQS()

	// Get number of worker goroutines from environment variable
	// Phase 3: start with 1, Phase 5: scale up to 5, 20, 100
	numWorkers := 1
	if w := os.Getenv("NUM_WORKERS"); w != "" {
		if n, err := strconv.Atoi(w); err == nil && n > 0 {
			numWorkers = n
		}
	}

	// Start worker goroutines
	log.Printf("[PROCESSOR] Starting %d worker goroutines", numWorkers)
	for i := 1; i <= numWorkers; i++ {
		go pollAndProcess(i)
	}

	// Health check endpoint (required for ECS health checks)
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":          "healthy",
			"service":         "order-processor",
			"num_workers":     numWorkers,
			"total_processed": atomic.LoadInt64(&totalProcessed),
			"total_failed":    atomic.LoadInt64(&totalFailed),
		})
	})

	log.Println("========================================")
	log.Println("Order Processor Service starting on :8080")
	log.Printf("  Workers: %d goroutines polling SQS", numWorkers)
	log.Println("  GET /health - Health check")
	log.Println("========================================")
	r.Run(":8080")
}
