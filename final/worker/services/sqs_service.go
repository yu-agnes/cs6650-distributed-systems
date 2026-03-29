package services

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSService struct {
	client   *sqs.Client
	queueURL string
}

func NewSQSService(client *sqs.Client, queueURL string) *SQSService {
	return &SQSService{
		client:   client,
		queueURL: queueURL,
	}
}

// PollMessages receives up to 1 message with long polling (10s wait)
func (s *SQSService) PollMessages(ctx context.Context) ([]types.Message, error) {
	result, err := s.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(s.queueURL),
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     10, // long polling
	})
	if err != nil {
		return nil, fmt.Errorf("receive message: %w", err)
	}
	return result.Messages, nil
}

// DeleteMessage removes a processed message from the queue
func (s *SQSService) DeleteMessage(ctx context.Context, receiptHandle string) error {
	_, err := s.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(s.queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	return nil
}
