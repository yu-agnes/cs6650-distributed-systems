package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
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
	Status     string    `json:"status"`
	Items      []Item    `json:"items"`
	CreatedAt  time.Time `json:"created_at"`
}

// ==================== Handler ====================

func handler(ctx context.Context, snsEvent events.SNSEvent) error {
	for _, record := range snsEvent.Records {
		// SNS message body is directly available - no envelope parsing needed
		message := record.SNS.Message

		var order Order
		if err := json.Unmarshal([]byte(message), &order); err != nil {
			log.Printf("[ERROR] Failed to parse order: %v", err)
			continue
		}

		log.Printf("[LAMBDA] Processing order %s for customer %d", order.OrderID, order.CustomerID)

		// Simulate 3-second payment processing (same as ECS processor)
		time.Sleep(3 * time.Second)

		log.Printf("[LAMBDA] Order %s completed", order.OrderID)
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
