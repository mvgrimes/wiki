package server

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3PutClientStub struct {
	input *s3.PutObjectInput
	err   error
}

func (s *s3PutClientStub) PutObject(_ context.Context, params *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	s.input = params
	if s.err != nil {
		return nil, s.err
	}
	return &s3.PutObjectOutput{}, nil
}

func TestS3ObjectStorePutObject(t *testing.T) {
	client := &s3PutClientStub{}
	store := &S3ObjectStore{client: client}

	err := store.PutObject(context.Background(), "bucket-1", "attachments/a/file.txt", "text/plain", bytes.NewReader([]byte("hello")), 5)
	if err != nil {
		t.Fatalf("PutObject() error = %v", err)
	}
	if client.input == nil || *client.input.Bucket != "bucket-1" || *client.input.Key != "attachments/a/file.txt" {
		t.Fatalf("unexpected put object input: %#v", client.input)
	}
}

func TestS3ObjectStorePutObjectErrorWrap(t *testing.T) {
	store := &S3ObjectStore{client: &s3PutClientStub{err: errors.New("boom")}}
	err := store.PutObject(context.Background(), "bucket-1", "attachments/a/file.txt", "text/plain", bytes.NewReader([]byte("hello")), 5)
	if err == nil || !strings.Contains(err.Error(), "s3 put object") {
		t.Fatalf("PutObject() error = %v, want wrapped s3 put object error", err)
	}
}

func TestNewS3ObjectStoreRequiresRegion(t *testing.T) {
	_, err := NewS3ObjectStore("", "", "", "", true)
	if err == nil || !strings.Contains(err.Error(), "s3 region is required") {
		t.Fatalf("error = %v, want region required", err)
	}
}
