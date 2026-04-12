package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"chaosarena/internal/store"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

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

	// Read env vars
	albumsTable := os.Getenv("ALBUMS_TABLE")
	photosTable := os.Getenv("PHOTOS_TABLE")
	s3Bucket := os.Getenv("S3_BUCKET")
	sqsQueueURL := os.Getenv("SQS_QUEUE_URL")

	db := store.NewDynamoStore(dynamoClient, albumsTable, photosTable)
	s3Store := store.NewS3Store(s3Client, s3Bucket, region)

	// Semaphore to limit concurrent photo processing (avoid S3/DynamoDB throttling)
	sem := make(chan struct{}, 5)

	log.Printf("Worker started. Polling SQS: %s", sqsQueueURL)

	// Main polling loop
	for {
		select {
		case <-ctx.Done():
			log.Println("Worker shutting down.")
			return
		default:
		}

		// Long-poll SQS for messages (20s wait = efficient long polling)
		result, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &sqsQueueURL,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20,
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("ERROR receiving SQS messages: %v", err)
			continue
		}

		var wg sync.WaitGroup
		for _, msg := range result.Messages {
			sem <- struct{}{}
			wg.Add(1)
			go func(m sqstypes.Message) {
				defer wg.Done()
				defer func() { <-sem }()
				processMessage(ctx, db, s3Store, s3Client, s3Bucket, region, sqsClient, sqsQueueURL, m)
			}(msg)
		}
		wg.Wait()
	}
}

func processMessage(
	ctx context.Context,
	db *store.DynamoStore,
	s3Store *store.S3Store,
	s3Client *s3.Client,
	bucket, region string,
	sqsClient *sqs.Client,
	queueURL string,
	msg sqstypes.Message,
) {
	// Parse the SQS message
	var photoMsg store.PhotoMessage
	if err := json.Unmarshal([]byte(*msg.Body), &photoMsg); err != nil {
		log.Printf("ERROR: unmarshal SQS message: %v", err)
		deleteMsg(ctx, sqsClient, queueURL, msg)
		return
	}

	log.Printf("Processing photo %s for album %s", photoMsg.PhotoID, photoMsg.AlbumID)

	// Copy from temp to final location in S3
	finalKey := fmt.Sprintf("albums/%s/photos/%s", photoMsg.AlbumID, photoMsg.PhotoID)
	copySource := fmt.Sprintf("%s/%s", bucket, photoMsg.S3TempKey)

	_, err := s3Client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     &bucket,
		Key:        &finalKey,
		CopySource: &copySource,
	})
	if err != nil {
		log.Printf("ERROR: copy S3 object for photo %s: %v", photoMsg.PhotoID, err)
		_ = db.UpdatePhotoFailed(ctx, photoMsg.PhotoID)
		deleteMsg(ctx, sqsClient, queueURL, msg)
		return
	}

	// Delete the temp file (best effort)
	_ = s3Store.Delete(ctx, photoMsg.S3TempKey)

	// Update DynamoDB: status -> "completed", set url and s3_key
	publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, finalKey)
	if err := db.UpdatePhotoCompleted(ctx, photoMsg.PhotoID, publicURL, finalKey); err != nil {
		log.Printf("ERROR: update photo %s to completed: %v", photoMsg.PhotoID, err)
		deleteMsg(ctx, sqsClient, queueURL, msg)
		return
	}

	log.Printf("Photo %s completed: %s", photoMsg.PhotoID, publicURL)

	// Acknowledge: delete SQS message
	deleteMsg(ctx, sqsClient, queueURL, msg)
}

func deleteMsg(ctx context.Context, client *sqs.Client, queueURL string, msg sqstypes.Message) {
	_, err := client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &queueURL,
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		log.Printf("ERROR: delete SQS message: %v", err)
	}
}
