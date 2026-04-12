package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"chaosarena/internal/handlers"
	"chaosarena/internal/store"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	// Load AWS config
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create AWS clients
	dynamoClient := dynamodb.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)
	sqsClient := sqs.NewFromConfig(cfg)

	// Create store layer
	albumsTable := os.Getenv("ALBUMS_TABLE")
	photosTable := os.Getenv("PHOTOS_TABLE")
	s3Bucket := os.Getenv("S3_BUCKET")
	sqsQueueURL := os.Getenv("SQS_QUEUE_URL")

	db := store.NewDynamoStore(dynamoClient, albumsTable, photosTable)
	s3Store := store.NewS3Store(s3Client, s3Bucket, region)
	sqsStore := store.NewSQSStore(sqsClient, sqsQueueURL)

	// Create handlers
	albumHandler := handlers.NewAlbumHandler(db)
	photoHandler := handlers.NewPhotoHandler(db, s3Store, sqsStore)

	// Setup Gin router
	r := gin.Default()

	// Health check
	r.GET("/health", handlers.GetHealth)

	// Album endpoints
	r.PUT("/albums/:album_id", albumHandler.PutAlbum)
	r.GET("/albums/:album_id", albumHandler.GetAlbum)
	r.GET("/albums", albumHandler.ListAlbums)

	// Photo endpoints
	r.POST("/albums/:album_id/photos", photoHandler.UploadPhoto)
	r.GET("/albums/:album_id/photos/:photo_id", photoHandler.GetPhoto)
	r.DELETE("/albums/:album_id/photos/:photo_id", photoHandler.DeletePhoto)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Album Store API listening on :%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
