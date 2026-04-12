package store

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const smallFileThreshold = 5 * 1024 * 1024 // 5MB

type S3Store struct {
	client   *s3.Client
	uploader *s3manager.Uploader
	bucket   string
	region   string
}

func NewS3Store(client *s3.Client, bucket, region string) *S3Store {
	return &S3Store{
		client:   client,
		uploader: s3manager.NewUploader(client),
		bucket:   bucket,
		region:   region,
	}
}

// Upload streams a file to S3.
// Files <= 5MB use a single PutObject with exact Content-Length (faster).
// Files > 5MB use s3manager multipart upload (handles unknown total size).
func (s *S3Store) Upload(ctx context.Context, key string, data io.Reader) error {
	// Read up to 5MB+1 to decide which upload path to take
	peek := make([]byte, smallFileThreshold+1)
	n, err := io.ReadFull(data, peek)
	if err != nil && err != io.ErrUnexpectedEOF {
		return fmt.Errorf("s3 read: %w", err)
	}

	if n <= smallFileThreshold {
		// Small file: single PutObject with known Content-Length
		_, putErr := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:        &s.bucket,
			Key:           &key,
			Body:          bytes.NewReader(peek[:n]),
			ContentLength: aws.Int64(int64(n)),
		})
		if putErr != nil {
			return fmt.Errorf("s3 upload: %w", putErr)
		}
		return nil
	}

	// Large file: s3manager multipart with buffered prefix + remaining stream
	combined := io.MultiReader(bytes.NewReader(peek), data)
	_, uploadErr := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Body:   combined,
	})
	if uploadErr != nil {
		return fmt.Errorf("s3 upload: %w", uploadErr)
	}
	return nil
}

// UploadBytes uploads a byte slice to S3 with exact Content-Length (fastest path).
func (s *S3Store) UploadBytes(ctx context.Context, key string, data []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &key,
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
	})
	if err != nil {
		return fmt.Errorf("s3 upload: %w", err)
	}
	return nil
}

// Delete removes a file from S3.
func (s *S3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("s3 delete: %w", err)
	}
	return nil
}

// PublicURL returns the public URL for an S3 object.
func (s *S3Store) PublicURL(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
}
