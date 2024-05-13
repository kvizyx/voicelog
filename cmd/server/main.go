package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/kvizyx/voicelog/internal/app"
	"github.com/kvizyx/voicelog/internal/config"
	"github.com/kvizyx/voicelog/internal/storage/s3"
	loglib "github.com/kvizyx/voicelog/pkg/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/sync/errgroup"
)

var configPath = flag.String(
	"config",
	".env",
	"path to config",
)

func main() {
	cfg, err := config.New(*configPath)
	if err != nil {
		panic(err)
	}

	logger := loglib.MustNew(loglib.Params{
		Env:        cfg.Env,
		LevelLocal: slog.LevelDebug,
		LevelProd:  slog.LevelInfo,
	})

	minioClient, err := initMinioClient(context.TODO(), cfg)
	if err != nil {
		panic(err)
	}

	s3Storage := s3.NewStorage(minioClient, cfg.S3)

	a := app.New(app.Params{
		Logger:    logger,
		Config:    cfg,
		S3Storage: &s3Storage,
	})

	parentCtx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL,
	)
	defer cancel()

	eg, lifeCtx := errgroup.WithContext(parentCtx)

	eg.Go(func() error {
		if err = a.Start(lifeCtx); err != nil {
			return fmt.Errorf("start service: %w", err)
		}

		return nil
	})

	eg.Go(func() error {
		<-lifeCtx.Done()

		logger.Info("stopping service...")

		if err = a.Stop(context.TODO()); err != nil {
			return fmt.Errorf("stop service: %w", err)
		}

		return nil
	})

	if err = eg.Wait(); err != nil {
		logger.Error("failed to shutdown gracefully", slog.Any("error", err))
	}
}

// initMinioClient creates minio client and ensures that bucket is created, otherwise create new one.
func initMinioClient(ctx context.Context, config config.Config) (*minio.Client, error) {
	minioClient, err := minio.New(config.S3.Addr, &minio.Options{
		Creds:  credentials.NewStaticV4(config.S3.AccessKey, config.S3.SecretKey, ""),
		Secure: config.Env == "production",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	bucketExists, err := minioClient.BucketExists(ctx, config.S3.Bucket)
	if err != nil && !bucketExists {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if bucketExists {
		return minioClient, nil
	}

	err = minioClient.MakeBucket(ctx, config.S3.Bucket, minio.MakeBucketOptions{
		Region: config.S3.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return minioClient, nil
}
