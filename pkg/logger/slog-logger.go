package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	slogzerolog "github.com/samber/slog-zerolog/v2"
)

type loggerImpl struct {
	logger *slog.Logger
}

type Params struct {
	Env string

	LevelLocal slog.Level
	LevelProd  slog.Level
}

func MustNew(params Params) Logger {
	var (
		logLevel  slog.Level
		logWriter io.Writer
	)

	switch params.Env {
	case "production":
		logLevel = params.LevelLocal
		logWriter = os.Stdout
	case "local":
		logLevel = params.LevelProd
		logWriter = zerolog.ConsoleWriter{
			Out: os.Stdout,
		}
	default:
		panic(fmt.Sprintf("unknown app environment: %s", params.Env))
	}

	baseOption := slogzerolog.Option{
		Level: logLevel,
	}

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.ErrorStackFieldName = "stack"

	slogzerolog.SourceKey = "source"
	slogzerolog.ErrorKeys = []string{"helper"}

	baseLogger := zerolog.New(logWriter)

	baseOption.Logger = &baseLogger
	baseOption.AddSource = true

	slogLogger := slog.New(baseOption.NewZerologHandler())

	return &loggerImpl{
		logger: slogLogger,
	}
}

func (c *loggerImpl) handle(level slog.Level, input string, fields ...any) {
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	record := slog.NewRecord(time.Now(), level, input, pcs[0])

	for _, field := range fields {
		record.Add(field)
	}

	_ = c.logger.Handler().Handle(context.Background(), record)
}

func (c *loggerImpl) Info(input string, fields ...any) {
	c.handle(slog.LevelInfo, input, fields...)
}

func (c *loggerImpl) Warn(input string, fields ...any) {
	c.handle(slog.LevelWarn, input, fields...)
}

func (c *loggerImpl) Error(input string, fields ...any) {
	c.handle(slog.LevelError, input, fields...)
}

func (c *loggerImpl) Debug(input string, fields ...any) {
	c.handle(slog.LevelDebug, input, fields...)
}

func (c *loggerImpl) With(args ...any) Logger {
	return &loggerImpl{
		logger: c.logger.With(args...),
	}
}
