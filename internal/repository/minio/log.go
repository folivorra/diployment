package minio

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/folivorra/diployment/internal/model"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

var ErrLogNotFound = errors.New("log not found")

type logMinioRepo struct {
	client *minio.Client
	bucket string
}

func NewLogMinioRepo(client *minio.Client, bucket string) *logMinioRepo {
	return &logMinioRepo{client: client, bucket: bucket}
}

func (r *logMinioRepo) Get(ctx context.Context, jobID uuid.UUID, phase model.Phase) (io.ReadCloser, error) {
	var key string
	switch phase {
	case model.PhaseBuild:
		key = fmt.Sprintf("%s.log", jobID.String())
	case model.PhaseDeploy:
		key = fmt.Sprintf("%s-deploy.log", jobID.String())
	}

	obj, err := r.client.GetObject(ctx, r.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		var resp minio.ErrorResponse
		if errors.As(err, &resp) && resp.Code == "NoSuchKey" {
			return nil, ErrLogNotFound
		}
		return nil, fmt.Errorf("get object: %w", err)
	}

	// вычитываем stat чтобы поймать NoSuchKey который возникает при первом чтении
	if _, err = obj.Stat(); err != nil {
		_ = obj.Close()
		var resp minio.ErrorResponse
		if errors.As(err, &resp) && resp.Code == "NoSuchKey" {
			return nil, ErrLogNotFound
		}
		return nil, fmt.Errorf("stat object: %w", err)
	}

	return obj, nil
}
