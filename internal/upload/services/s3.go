package services

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"lead-crm/backend/config"
	"lead-crm/backend/internal/upload/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

type S3Service struct {
	cfg           *config.Config
	s3Client      *s3.Client
	presignClient *s3.PresignClient
}

var S3 = NewS3Service()

func NewS3Service() *S3Service {
	cfg := config.Get()

	var optFns []func(*awsconfig.LoadOptions) error
	if cfg.AWSRegion != "" {
		optFns = append(optFns, awsconfig.WithRegion(cfg.AWSRegion))
	}
	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		optFns = append(optFns, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), optFns...)
	if err != nil {
		awsCfg = aws.Config{}
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.AWSS3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.AWSS3Endpoint)
		}
		o.UsePathStyle = cfg.AWSS3ForcePathStyle
	})

	return &S3Service{
		cfg:           cfg,
		s3Client:      s3Client,
		presignClient: s3.NewPresignClient(s3Client),
	}
}

func (s *S3Service) GeneratePresignedPost(ctx context.Context, filename, contentType, folder string) (*models.PresignedURLResponse, error) {
	if s.cfg.AWSS3Bucket == "" {
		return nil, errors.New("AWS S3 bucket is not configured on the server")
	}

	cleanFolder := filepath.Clean(strings.Trim(folder, "/ "))
	if cleanFolder == "" || cleanFolder == "." || strings.HasPrefix(cleanFolder, "..") {
		cleanFolder = "uploads"
	}

	cleanName := filepath.Base(filename)
	cleanName = strings.ReplaceAll(cleanName, " ", "_")
	objectKey := fmt.Sprintf("%s/%d_%s_%s", cleanFolder, time.Now().Unix(), uuid.New().String()[:8], cleanName)

	postReq, err := s.presignClient.PresignPostObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.AWSS3Bucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}, func(opts *s3.PresignPostOptions) {
		opts.Expires = 15 * time.Minute
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned upload url: %w", err)
	}

	return &models.PresignedURLResponse{
		URL:    postReq.URL,
		Fields: postReq.Values,
		Key:    objectKey,
	}, nil
}

func (s *S3Service) VerifyUpload(ctx context.Context, key string) (*models.VerifyUploadResponse, error) {
	if s.cfg.AWSS3Bucket == "" {
		return nil, errors.New("AWS S3 bucket is not configured on the server")
	}

	_, err := s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.cfg.AWSS3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("file verification failed: %w", err)
	}

	return &models.VerifyUploadResponse{
		URL: s.GetObjectURL(key),
		Key: key,
	}, nil
}

func (s *S3Service) DeleteObjects(ctx context.Context, keys []string) error {
	if s.cfg.AWSS3Bucket == "" {
		return errors.New("AWS S3 bucket is not configured on the server")
	}

	var objectIDs []types.ObjectIdentifier
	for _, k := range keys {
		trimmed := strings.TrimSpace(k)
		if trimmed != "" {
			objectIDs = append(objectIDs, types.ObjectIdentifier{Key: aws.String(trimmed)})
		}
	}
	if len(objectIDs) == 0 {
		return nil
	}

	_, err := s.s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.cfg.AWSS3Bucket),
		Delete: &types.Delete{
			Objects: objectIDs,
			Quiet:   aws.Bool(true),
		},
	})
	return err
}

func (s *S3Service) GetObjectURL(key string) string {
	if s.cfg.AWSS3PublicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.cfg.AWSS3PublicURL, "/"), key)
	}
	if s.cfg.AWSS3Endpoint != "" {
		if s.cfg.AWSS3ForcePathStyle {
			return fmt.Sprintf("%s/%s/%s", strings.TrimRight(s.cfg.AWSS3Endpoint, "/"), s.cfg.AWSS3Bucket, key)
		}
		endpoint := strings.TrimPrefix(strings.TrimPrefix(s.cfg.AWSS3Endpoint, "http://"), "https://")
		return fmt.Sprintf("https://%s.%s/%s", s.cfg.AWSS3Bucket, endpoint, key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.cfg.AWSS3Bucket, s.cfg.AWSRegion, key)
}
