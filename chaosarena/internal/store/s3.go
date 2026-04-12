package store

import (
	"context"
	"fmt"
	"io"

	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

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

// Upload streams a file directly to S3 using the transfer manager.
// No size parameter needed — s3manager handles unknown-size streams via multipart upload.
func (s *S3Store) Upload(ctx context.Context, key string, data io.Reader) error {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Body:   data,
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
