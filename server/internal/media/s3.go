package media

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/rxtech-lab/railway-wiki/internal/api"
)

// S3Config configures the S3 presigner.
type S3Config struct {
	Bucket    string
	Region    string
	Endpoint  string // optional; set for localstack / S3-compatible stores
	AccessKey string
	SecretKey string
	PublicURL string        // canonical base URL for stored objects; defaults to the request URL host
	UsePath   bool          // path-style addressing (required by localstack)
	TTL       time.Duration // presigned URL lifetime
}

// S3Presigner implements Presigner against an S3-compatible backend.
type S3Presigner struct {
	presign   *s3.PresignClient
	bucket    string
	publicURL string
	ttl       time.Duration
}

// NewS3Presigner builds an S3-backed presigner.
func NewS3Presigner(ctx context.Context, cfg S3Config) (*S3Presigner, error) {
	loadOpts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(cfg.Region)}
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePath
	})

	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	return &S3Presigner{
		presign:   s3.NewPresignClient(client),
		bucket:    cfg.Bucket,
		publicURL: cfg.PublicURL,
		ttl:       ttl,
	}, nil
}

// Presign returns a presigned PUT target for the given upload.
func (p *S3Presigner) Presign(ctx context.Context, in PresignInput) (*api.MediaUploadTarget, error) {
	key := objectKey(in.FileName)
	req, err := p.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(in.ContentType),
	}, s3.WithPresignExpires(p.ttl))
	if err != nil {
		return nil, fmt.Errorf("failed to presign upload: %w", err)
	}

	headers := map[string]string{}
	for k := range req.SignedHeader {
		headers[k] = req.SignedHeader.Get(k)
	}

	now := time.Now()
	target := &api.MediaUploadTarget{
		UploadUrl: req.URL,
		Method:    req.Method,
		Headers:   &headers,
		Url:       p.canonicalURL(key),
		Key:       &key,
		ExpiresAt: expiry(p.ttl, now),
	}
	return target, nil
}

func (p *S3Presigner) canonicalURL(key string) string {
	if p.publicURL != "" {
		return fmt.Sprintf("%s/%s", trimSlash(p.publicURL), key)
	}
	return fmt.Sprintf("s3://%s/%s", p.bucket, key)
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
