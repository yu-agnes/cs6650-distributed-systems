package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSStore struct {
	client   *sqs.Client
	queueURL string
}

func NewSQSStore(client *sqs.Client, queueURL string) *SQSStore {
	return &SQSStore{
		client:   client,
		queueURL: queueURL,
	}
}

// PhotoMessage is the message sent to SQS when a photo upload is accepted.
type PhotoMessage struct {
	PhotoID  string `json:"photo_id"`
	AlbumID  string `json:"album_id"`
	S3TempKey string `json:"s3_temp_key"`
}

// SendPhotoMessage sends a message to the SQS queue to notify the worker.
func (s *SQSStore) SendPhotoMessage(ctx context.Context, msg PhotoMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal sqs message: %w", err)
	}

	_, err = s.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &s.queueURL,
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return fmt.Errorf("send sqs message: %w", err)
	}
	return nil
}

// DeleteMessage removes a message from the queue after successful processing.
func (s *SQSStore) DeleteMessage(ctx context.Context, receiptHandle string) error {
	_, err := s.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &s.queueURL,
		ReceiptHandle: &receiptHandle,
	})
	if err != nil {
		return fmt.Errorf("delete sqs message: %w", err)
	}
	return nil
}
