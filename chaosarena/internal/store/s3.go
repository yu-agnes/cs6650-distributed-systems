package store

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Store struct {
	client *s3.Client
	bucket string
	region string
}

func NewS3Store(client *s3.Client, bucket, region string) *S3Store {
	return &S3Store{
		client: client,
		bucket: bucket,
		region: region,
	}
}

// Upload streams a file to S3. size must be the exact byte count of data
// (from multipart.FileHeader.Size) so S3 gets an accurate Content-Length
// without buffering the entire payload into memory.
func (s *S3Store) Upload(ctx context.Context, key string, data io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &key,
		Body:          data,
		ContentLength: aws.Int64(size),
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
// Requires the bucket to have public read access or a bucket policy allowing GetObject.
func (s *S3Store) PublicURL(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
}
