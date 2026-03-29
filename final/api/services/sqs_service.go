package services

import (
	"context"
	"encoding/json"
	"fmt"

	"api/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
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

// SendJob sends the job to SQS for workers to pick up
func (s *SQSService) SendJob(ctx context.Context, jobID string, urls []string) error {
	msg := models.SQSMessage{
		JobID: jobID,
		URLs:  urls,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	_, err = s.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}
