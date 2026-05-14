package minio

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/folivorra/diployment/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinioClient(cfg config.MinIOConfig) (*minio.Client, error) {
	return minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
}

func InitBucket(ctx context.Context, m *minio.Client, bucketName string) error {
	exists, err := m.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("cannot check bucket: %v", err)
	}
	if !exists {
		if err := m.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("cannot create bucket: %v", err)
		}
		slog.Info("created minio bucket", slog.String("bucket", bucketName))
	}
	return nil
}
