package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type s3Client interface {
	PutObject(
		ctx context.Context,
		params *s3.PutObjectInput,
		optFns ...func(*s3.Options),
	) (*s3.PutObjectOutput, error)
	GetObject(
		ctx context.Context,
		params *s3.GetObjectInput,
		optFns ...func(*s3.Options),
	) (*s3.GetObjectOutput, error)
}

type S3ObjectStore struct {
	client s3Client
}

func NewS3ObjectStore(
	region, endpoint, accessKeyID, secretAccessKey string,
	forcePathStyle bool,
) (*S3ObjectStore, error) {
	if strings.TrimSpace(region) == "" {
		return nil, fmt.Errorf("s3 region is required")
	}

	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = forcePathStyle
		if strings.TrimSpace(endpoint) != "" {
			o.BaseEndpoint = aws.String(strings.TrimSpace(endpoint))
		}
	})

	return &S3ObjectStore{client: client}, nil
}

func (s *S3ObjectStore) PutObject(
	ctx context.Context,
	bucket, key, contentType string,
	body io.Reader,
	size int64,
) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return fmt.Errorf("s3 put object: %w", err)
	}
	return nil
}

func (s *S3ObjectStore) GetObject(
	ctx context.Context,
	bucket, key string,
) (AttachmentObject, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return AttachmentObject{}, errAttachmentNotFound
		}
		return AttachmentObject{}, fmt.Errorf("s3 get object: %w", err)
	}

	return AttachmentObject{Body: out.Body, ContentType: aws.ToString(out.ContentType)}, nil
}
