package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gin-gonic/gin"
)

func main() {
	// ==================== Read config from environment ====================
	tableName := getEnv("DYNAMODB_TABLE", "shopping-carts")
	awsRegion := getEnv("AWS_REGION", "us-east-1")
	dynamoEndpoint := getEnv("DYNAMODB_ENDPOINT", "") // empty = use real AWS

	// ==================== Connect to DynamoDB ====================
	var client *dynamodb.Client

	if dynamoEndpoint != "" {
		// LOCAL mode: use static fake credentials
		// This avoids AWS SDK picking up ~/.aws/credentials
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(awsRegion),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				"fakekey", "fakesecret", "",
			)),
		)
		if err != nil {
			log.Fatalf("Failed to load AWS config: %v", err)
		}
		client = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(dynamoEndpoint)
		})
		log.Printf("DynamoDB client initialized (local endpoint: %s)", dynamoEndpoint)
	} else {
		// AWS mode: use default credential chain (IAM role, env vars, etc.)
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(awsRegion),
		)
		if err != nil {
			log.Fatalf("Failed to load AWS config: %v", err)
		}
		client = dynamodb.NewFromConfig(cfg)
		log.Println("DynamoDB client initialized (AWS)")
	}

	// ==================== Set up Gin router ====================
	router := gin.Default()

	// Health check (ALB uses this)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Shopping Cart API endpoints (same routes as MySQL version)
	router.POST("/shopping-carts", createCartHandler(client, tableName))
	router.GET("/shopping-carts/:id", getCartHandler(client, tableName))
	router.POST("/shopping-carts/:id/items", addItemsHandler(client, tableName))

	// Start server on port 8080
	log.Println("Starting cart-api (DynamoDB) server on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// getEnv reads env variable with a fallback default
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
