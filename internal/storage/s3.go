package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "gostream/internal/config"
)

type S3Backend struct {
	client *s3.Client
	bucket string
}

func NewS3(cfg *appconfig.Config) (*S3Backend, error) {
	if cfg.Storage.S3.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket not configured")
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if cfg.Storage.S3.Endpoint != "" {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           cfg.Storage.S3.Endpoint,
				SigningRegion: cfg.Storage.S3.Region,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.Storage.S3.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.Storage.S3.AccessKey, cfg.Storage.S3.SecretKey, "")),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.Storage.S3.PathStyle
		o.DisableLogOutputChecksumValidationSkipped = true
	})

	return &S3Backend{
		client: client,
		bucket: cfg.Storage.S3.Bucket,
	}, nil
}

func (c *S3Backend) UploadFile(key string, file multipart.File, contentType string) error {
	_, err := c.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	return err
}

func (c *S3Backend) Upload(key string, file io.Reader, contentType string) error {
	_, err := c.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	return err
}

func (c *S3Backend) Delete(key string) error {
	_, err := c.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (c *S3Backend) GetStream(key string) (io.ReadCloser, error) {
	out, err := c.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (c *S3Backend) Writable() bool {
	return true
}

func (c *S3Backend) BasePath() string {
	return ""
}
