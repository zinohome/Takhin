// Copyright 2025 Takhin Data, Inc.

package tiered

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Client struct {
	client *s3.Client
	bucket string
	prefix string
}

type S3Config struct {
	Region          string
	Bucket          string
	Prefix          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

func NewS3Client(ctx context.Context, cfg S3Config) (*S3Client, error) {
	var opts []func(*config.LoadOptions) error
	
	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	}
	
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	s3Opts := func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	}

	client := s3.NewFromConfig(awsCfg, s3Opts)

	return &S3Client{
		client: client,
		bucket: cfg.Bucket,
		prefix: cfg.Prefix,
	}, nil
}

func (c *S3Client) UploadFile(ctx context.Context, localPath string, key string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	s3Key := filepath.Join(c.prefix, key)
	
	_, err = c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3Key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("upload to s3: %w", err)
	}

	return nil
}

func (c *S3Client) DownloadFile(ctx context.Context, key string, localPath string) error {
	s3Key := filepath.Join(c.prefix, key)
	
	result, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("get object from s3: %w", err)
	}
	defer result.Body.Close()

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, result.Body)
	if err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	return nil
}

func (c *S3Client) DeleteFile(ctx context.Context, key string) error {
	s3Key := filepath.Join(c.prefix, key)
	
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("delete from s3: %w", err)
	}

	return nil
}

func (c *S3Client) FileExists(ctx context.Context, key string) (bool, error) {
	s3Key := filepath.Join(c.prefix, key)
	
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, fmt.Errorf("head object: %w", err)
	}

	return true, nil
}

func (c *S3Client) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	s3Prefix := filepath.Join(c.prefix, prefix)
	
	result, err := c.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(s3Prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("list objects: %w", err)
	}

	keys := make([]string, 0, len(result.Contents))
	for _, obj := range result.Contents {
		if obj.Key != nil {
			keys = append(keys, *obj.Key)
		}
	}

	return keys, nil
}

func (c *S3Client) GetFileModTime(ctx context.Context, key string) (time.Time, error) {
	s3Key := filepath.Join(c.prefix, key)
	
	result, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("head object: %w", err)
	}

	if result.LastModified != nil {
		return *result.LastModified, nil
	}

	return time.Time{}, fmt.Errorf("no last modified time")
}
