package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"api/handlers"
	"api/services"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load AWS config
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create AWS clients
	sqsClient := sqs.NewFromConfig(cfg)
	dynamoClient := dynamodb.NewFromConfig(cfg)

	// Create services
	sqsSvc := services.NewSQSService(sqsClient, os.Getenv("SQS_QUEUE_URL"))
	dynamoSvc := services.NewDynamoService(dynamoClient, os.Getenv("DYNAMODB_TABLE"))

	// Create handler
	jobHandler := handlers.NewJobHandler(dynamoSvc, sqsSvc)

	// Setup Gin router
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.POST("/jobs", jobHandler.CreateJob)
	r.GET("/jobs/:id", jobHandler.GetJob)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("API service listening on :%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
