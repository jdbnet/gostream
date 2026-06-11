package s3

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "gostream/internal/config"
)

type Client struct {
	client *s3.Client
	bucket string
}

func NewClient(cfg *appconfig.Config) (*Client, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if cfg.S3.Endpoint != "" {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           cfg.S3.Endpoint,
				SigningRegion: cfg.S3.Region,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.S3.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3.AccessKey, cfg.S3.SecretKey, "")),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.S3.PathStyle
	})

	return &Client{
		client: client,
		bucket: cfg.S3.Bucket,
	}, nil
}

func (c *Client) UploadFile(key string, file multipart.File, contentType string) error {
	_, err := c.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	return err
}

func (c *Client) UploadLocalFile(key string, file io.Reader, contentType string) error {
	_, err := c.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	return err
}

func (c *Client) DeleteFile(key string) error {
	_, err := c.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (c *Client) GetStream(key string) (io.ReadCloser, error) {
	out, err := c.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}
