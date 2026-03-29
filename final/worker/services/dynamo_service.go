package services

import (
	"context"
	"fmt"
	"time"

	"worker/scraper"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoService struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoService(client *dynamodb.Client, tableName string) *DynamoService {
	return &DynamoService{
		client:    client,
		tableName: tableName,
	}
}

// UpdateJobProcessing sets the job status to "processing"
func (s *DynamoService) UpdateJobProcessing(ctx context.Context, jobID string) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"jobID": &types.AttributeValueMemberS{Value: jobID},
		},
		UpdateExpression: aws.String("SET #status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: "processing"},
		},
	})
	if err != nil {
		return fmt.Errorf("update job processing: %w", err)
	}
	return nil
}

// UpdateJobCompleted writes results and sets status to "completed"
func (s *DynamoService) UpdateJobCompleted(ctx context.Context, jobID string, results []scraper.Result) error {
	resultsAV, err := attributevalue.MarshalList(results)
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"jobID": &types.AttributeValueMemberS{Value: jobID},
		},
		UpdateExpression: aws.String("SET #status = :status, results = :results, completedAt = :completedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":      &types.AttributeValueMemberS{Value: "completed"},
			":results":     &types.AttributeValueMemberL{Value: resultsAV},
			":completedAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		return fmt.Errorf("update job completed: %w", err)
	}
	return nil
}
