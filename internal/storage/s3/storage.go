package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/kvizyx/voicelog/internal/config"
	"github.com/minio/minio-go/v7"
)

type Storage struct {
	client *minio.Client

	s3Config config.S3
}

func NewStorage(client *minio.Client, config config.S3) Storage {
	return Storage{
		client:   client,
		s3Config: config,
	}
}

func (s *Storage) UploadVoice(ctx context.Context, ttl time.Duration, filePath string) (uuid.UUID, error) {
	voiceID, err := uuid.NewRandom()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to generate id for voice record: %w", err)
	}

	uploadOpts := minio.PutObjectOptions{
		ContentType: "application/octet-stream",
		Expires:     time.Now().Add(ttl),
	}

	_, err = s.client.FPutObject(ctx, s.s3Config.Bucket, s.makeVoiceName(voiceID), filePath, uploadOpts)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to upload voice to s3: %w", err)
	}

	return voiceID, nil
}

func (s *Storage) DownloadVoice(ctx context.Context, voiceID uuid.UUID) (io.ReadCloser, error) {
	voiceRecord, err := s.client.GetObject(
		ctx, s.s3Config.Bucket,
		s.makeVoiceName(voiceID),
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to download voice from s3: %w", err)
	}

	return voiceRecord, nil
}

func (s *Storage) makeVoiceName(id uuid.UUID) string {
	return fmt.Sprintf("%s.ogg", id.String())
}
