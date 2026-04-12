package main

import (
	"context"
	"encoding/json"
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

		// Process all messages in the batch concurrently
		var wg sync.WaitGroup
		for _, msg := range result.Messages {
			wg.Add(1)
			go func(m sqstypes.Message) {
				defer wg.Done()
				processMessage(ctx, db, s3Store, sqsClient, sqsQueueURL, m)
			}(msg)
		}
		wg.Wait()
	}
}

func processMessage(
	ctx context.Context,
	db *store.DynamoStore,
	s3Store *store.S3Store,
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

	// Photo is already at the final S3 location (uploaded by API).
	// Worker only needs to update DynamoDB status to "completed".
	publicURL := s3Store.PublicURL(photoMsg.S3Key)

	if err := db.UpdatePhotoCompleted(ctx, photoMsg.PhotoID, publicURL, photoMsg.S3Key); err != nil {
		log.Printf("ERROR: update photo %s to completed: %v", photoMsg.PhotoID, err)
	}

	log.Printf("Photo %s completed: %s", photoMsg.PhotoID, publicURL)
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
