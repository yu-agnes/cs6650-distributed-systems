package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/gin-gonic/gin"
)

// ==================== Data Models ====================

type Item struct {
	ProductID int     `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type Order struct {
	OrderID    string    `json:"order_id"`
	CustomerID int       `json:"customer_id"`
	Status     string    `json:"status"` // pending, processing, completed
	Items      []Item    `json:"items"`
	CreatedAt  time.Time `json:"created_at"`
}

type OrderRequest struct {
	CustomerID int    `json:"customer_id"`
	Items      []Item `json:"items"`
}

// ==================== Payment Processor Simulation ====================
// Buffered channel with capacity 1: only ONE payment processed at a time

var paymentSemaphore = make(chan struct{}, 1)

// Counters for monitoring
var (
	syncReceived  int64
	syncCompleted int64
	asyncReceived int64
)

// simulatePayment acquires the semaphore, sleeps 3s, then releases.
func simulatePayment(orderID string) error {
	paymentSemaphore <- struct{}{}
	defer func() { <-paymentSemaphore }()

	log.Printf("[PAYMENT] Processing order %s (3s delay)...", orderID)
	time.Sleep(3 * time.Second)
	log.Printf("[PAYMENT] Order %s completed", orderID)
	return nil
}

func generateOrderID() string {
	return fmt.Sprintf("ORD-%d", time.Now().UnixNano())
}

// ==================== SNS Client ====================

var snsClient *sns.Client
var snsTopicARN string

func initSNS() {
	// Get SNS topic ARN from environment variable
	snsTopicARN = os.Getenv("SNS_TOPIC_ARN")
	if snsTopicARN == "" {
		log.Println("[WARNING] SNS_TOPIC_ARN not set - async endpoint will not work")
		return
	}

	// Load AWS config (uses ECS task role credentials automatically)
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		log.Printf("[ERROR] Failed to load AWS config: %v", err)
		return
	}

	snsClient = sns.NewFromConfig(cfg)
	log.Printf("[SNS] Initialized, topic ARN: %s", snsTopicARN)
}

// ==================== Main ====================

func main() {
	// Initialize SNS client
	initSNS()

	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":         "healthy",
			"service":        "order-receiver",
			"sync_received":  atomic.LoadInt64(&syncReceived),
			"sync_completed": atomic.LoadInt64(&syncCompleted),
			"async_received": atomic.LoadInt64(&asyncReceived),
			"sns_configured": snsClient != nil,
		})
	})

	// ==================== Phase 1: Synchronous Endpoint ====================
	// Customer waits for payment processing (3s) before getting response
	r.POST("/orders/sync", func(c *gin.Context) {
		atomic.AddInt64(&syncReceived, 1)

		var req OrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		order := Order{
			OrderID:    generateOrderID(),
			CustomerID: req.CustomerID,
			Status:     "processing",
			Items:      req.Items,
			CreatedAt:  time.Now(),
		}

		// Synchronous: customer waits here
		err := simulatePayment(order.OrderID)
		if err != nil {
			order.Status = "failed"
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "payment processing failed",
				"order": order,
			})
			return
		}

		atomic.AddInt64(&syncCompleted, 1)
		order.Status = "completed"
		c.JSON(http.StatusOK, gin.H{
			"message": "order processed successfully",
			"order":   order,
		})
	})

	// ==================== Phase 3: Asynchronous Endpoint ====================
	// Publish order to SNS and return 202 immediately - customer doesn't wait
	r.POST("/orders/async", func(c *gin.Context) {
		atomic.AddInt64(&asyncReceived, 1)

		var req OrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		// Check if SNS is configured
		if snsClient == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "async processing not available - SNS not configured",
			})
			return
		}

		order := Order{
			OrderID:    generateOrderID(),
			CustomerID: req.CustomerID,
			Status:     "pending",
			Items:      req.Items,
			CreatedAt:  time.Now(),
		}

		// Convert order to JSON for SNS message
		orderJSON, err := json.Marshal(order)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize order"})
			return
		}

		// Publish to SNS - this is fast, just sends a message
		_, err = snsClient.Publish(context.TODO(), &sns.PublishInput{
			TopicArn: aws.String(snsTopicARN),
			Message:  aws.String(string(orderJSON)),
		})
		if err != nil {
			log.Printf("[ERROR] Failed to publish to SNS: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to queue order",
				"order": order,
			})
			return
		}

		// Return 202 Accepted immediately - customer doesn't wait for payment
		log.Printf("[ASYNC] Order %s published to SNS", order.OrderID)
		c.JSON(http.StatusAccepted, gin.H{
			"message": "order accepted for processing",
			"order":   order,
		})
	})

	log.Println("========================================")
	log.Println("Order Receiver Service starting on :8080")
	log.Println("  POST /orders/sync  - Synchronous (3s wait)")
	log.Println("  POST /orders/async - Asynchronous (instant)")
	log.Println("  GET  /health       - Health check")
	log.Println("========================================")
	r.Run(":8080")
}
