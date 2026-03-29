package services

import (
	"context"
	"fmt"

	"api/models"

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

// CreateJob writes the initial job record with status "pending"
func (s *DynamoService) CreateJob(ctx context.Context, job *models.Job) error {
	item, err := attributevalue.MarshalMap(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put item: %w", err)
	}
	return nil
}

// GetJob reads the job record - uses strongly consistent read
func (s *DynamoService) GetJob(ctx context.Context, jobID string) (*models.Job, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"jobID": &types.AttributeValueMemberS{Value: jobID},
		},
		ConsistentRead: aws.Bool(true), // avoid eventual consistency issues
	})
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}

	if result.Item == nil {
		return nil, nil // job not found
	}

	var job models.Job
	if err := attributevalue.UnmarshalMap(result.Item, &job); err != nil {
		return nil, fmt.Errorf("unmarshal job: %w", err)
	}
	return &job, nil
}
