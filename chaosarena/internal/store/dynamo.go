package store

import (
	"context"
	"errors"
	"fmt"

	"chaosarena/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoStore struct {
	client      *dynamodb.Client
	albumsTable string
	photosTable string
}

func NewDynamoStore(client *dynamodb.Client, albumsTable, photosTable string) *DynamoStore {
	return &DynamoStore{
		client:      client,
		albumsTable: albumsTable,
		photosTable: photosTable,
	}
}

// ==================== Album Operations ====================

// PutAlbum creates or updates an album. Returns true if newly created.
// Uses a conditional PutItem for creates (so we can detect new vs existing)
// and an UpdateItem for updates that never touches photo_seq_counter, avoiding
// the read-modify-write race with IncrementPhotoSeq.
func (d *DynamoStore) PutAlbum(ctx context.Context, album models.Album) (bool, error) {
	// Attempt a conditional create first (fails if already exists).
	item, err := attributevalue.MarshalMap(album)
	if err != nil {
		return false, fmt.Errorf("marshal album: %w", err)
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           &d.albumsTable,
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(album_id)"),
	})
	if err == nil {
		// Successfully created a new album.
		return true, nil
	}

	// If condition failed, the album already exists — do a targeted update
	// that never touches photo_seq_counter.
	var condErr *types.ConditionalCheckFailedException
	if !errors.As(err, &condErr) {
		return false, fmt.Errorf("put album: %w", err)
	}

	_, err = d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &d.albumsTable,
		Key: map[string]types.AttributeValue{
			"album_id": &types.AttributeValueMemberS{Value: album.AlbumID},
		},
		UpdateExpression: aws.String("SET title = :title, description = :desc, #owner = :owner"),
		ExpressionAttributeNames: map[string]string{
			"#owner": "owner", // "owner" is not a reserved word but alias for safety
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":title": &types.AttributeValueMemberS{Value: album.Title},
			":desc":  &types.AttributeValueMemberS{Value: album.Description},
			":owner": &types.AttributeValueMemberS{Value: album.Owner},
		},
	})
	if err != nil {
		return false, fmt.Errorf("update album: %w", err)
	}

	return false, nil
}

// GetAlbum retrieves an album by ID. Returns nil if not found.
func (d *DynamoStore) GetAlbum(ctx context.Context, albumID string) (*models.Album, error) {
	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      &d.albumsTable,
		ConsistentRead: aws.Bool(true),
		Key: map[string]types.AttributeValue{
			"album_id": &types.AttributeValueMemberS{Value: albumID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get album: %w", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	var album models.Album
	if err := attributevalue.UnmarshalMap(result.Item, &album); err != nil {
		return nil, fmt.Errorf("unmarshal album: %w", err)
	}
	return &album, nil
}

// ListAlbums returns all albums. Handles DynamoDB pagination internally.
func (d *DynamoStore) ListAlbums(ctx context.Context) ([]models.Album, error) {
	var albums []models.Album
	var lastKey map[string]types.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName:         &d.albumsTable,
			ExclusiveStartKey: lastKey,
		}

		result, err := d.client.Scan(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("scan albums: %w", err)
		}

		var page []models.Album
		if err := attributevalue.UnmarshalListOfMaps(result.Items, &page); err != nil {
			return nil, fmt.Errorf("unmarshal albums page: %w", err)
		}
		albums = append(albums, page...)

		if result.LastEvaluatedKey == nil {
			break
		}
		lastKey = result.LastEvaluatedKey
	}

	return albums, nil
}

// IncrementPhotoSeq atomically increments the photo sequence counter for an album.
// Returns the new seq value. This is the key to concurrent-safe seq assignment.
func (d *DynamoStore) IncrementPhotoSeq(ctx context.Context, albumID string) (int, error) {
	result, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &d.albumsTable,
		Key: map[string]types.AttributeValue{
			"album_id": &types.AttributeValueMemberS{Value: albumID},
		},
		UpdateExpression: aws.String("ADD photo_seq_counter :inc"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inc": &types.AttributeValueMemberN{Value: "1"},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return 0, fmt.Errorf("increment photo seq: %w", err)
	}

	// Extract the new counter value
	counterAttr, ok := result.Attributes["photo_seq_counter"]
	if !ok {
		return 0, fmt.Errorf("photo_seq_counter not in response")
	}
	var counter int
	if err := attributevalue.Unmarshal(counterAttr, &counter); err != nil {
		return 0, fmt.Errorf("unmarshal counter: %w", err)
	}
	return counter, nil
}

// ==================== Photo Operations ====================

// PutPhoto creates a photo record with status "processing".
func (d *DynamoStore) PutPhoto(ctx context.Context, photo models.Photo) error {
	item, err := attributevalue.MarshalMap(photo)
	if err != nil {
		return fmt.Errorf("marshal photo: %w", err)
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &d.photosTable,
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put photo: %w", err)
	}
	return nil
}

// GetPhoto retrieves a photo by ID. Returns nil if not found.
func (d *DynamoStore) GetPhoto(ctx context.Context, photoID string) (*models.Photo, error) {
	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      &d.photosTable,
		ConsistentRead: aws.Bool(true),
		Key: map[string]types.AttributeValue{
			"photo_id": &types.AttributeValueMemberS{Value: photoID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get photo: %w", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	var photo models.Photo
	if err := attributevalue.UnmarshalMap(result.Item, &photo); err != nil {
		return nil, fmt.Errorf("unmarshal photo: %w", err)
	}
	return &photo, nil
}

// UpdatePhotoCompleted sets a photo's status to "completed" and adds the URL.
// Uses ConditionExpression to prevent resurrecting a photo that was deleted
// while processing (delete-then-complete race condition).
func (d *DynamoStore) UpdatePhotoCompleted(ctx context.Context, photoID, url, s3Key string) error {
	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &d.photosTable,
		Key: map[string]types.AttributeValue{
			"photo_id": &types.AttributeValueMemberS{Value: photoID},
		},
		ConditionExpression: aws.String("attribute_exists(photo_id)"),
		UpdateExpression:    aws.String("SET #status = :status, #url = :url, s3_key = :s3key"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
			"#url":    "url",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: "completed"},
			":url":    &types.AttributeValueMemberS{Value: url},
			":s3key":  &types.AttributeValueMemberS{Value: s3Key},
		},
	})
	if err != nil {
		// ConditionalCheckFailedException means the photo was deleted — that's fine
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return nil
		}
		return fmt.Errorf("update photo completed: %w", err)
	}
	return nil
}

// UpdatePhotoFailed sets a photo's status to "failed".
// Uses ConditionExpression to prevent resurrecting a deleted photo.
func (d *DynamoStore) UpdatePhotoFailed(ctx context.Context, photoID string) error {
	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &d.photosTable,
		Key: map[string]types.AttributeValue{
			"photo_id": &types.AttributeValueMemberS{Value: photoID},
		},
		ConditionExpression: aws.String("attribute_exists(photo_id)"),
		UpdateExpression:    aws.String("SET #status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: "failed"},
		},
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return nil
		}
		return fmt.Errorf("update photo failed: %w", err)
	}
	return nil
}

// DeletePhoto removes a photo record from DynamoDB.
func (d *DynamoStore) DeletePhoto(ctx context.Context, photoID string) error {
	_, err := d.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &d.photosTable,
		Key: map[string]types.AttributeValue{
			"photo_id": &types.AttributeValueMemberS{Value: photoID},
		},
	})
	if err != nil {
		return fmt.Errorf("delete photo: %w", err)
	}
	return nil
}
