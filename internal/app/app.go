package app

import (
	"context"
	"fmt"

	"github.com/kvizyx/voicelog/internal/bot"
	"github.com/kvizyx/voicelog/internal/config"
	httpserver "github.com/kvizyx/voicelog/internal/http-server"
	"github.com/kvizyx/voicelog/internal/storage/s3"
	"github.com/kvizyx/voicelog/pkg/logger"
	"golang.org/x/sync/errgroup"
)

type App struct {
	Params

	discordBot bot.Bot
	httpServer httpserver.Server
}

type Params struct {
	Logger    logger.Logger
	Config    config.Config
	S3Storage *s3.Storage
}

func New(params Params) App {
	return App{Params: params}
}

func (a *App) Start(ctx context.Context) error {
	a.discordBot = bot.NewDiscordBot(bot.Params{
		Config:    a.Config,
		Logger:    a.Logger,
		S3Storage: a.S3Storage,
	})

	a.httpServer = httpserver.NewServer(httpserver.Params{
		Config:  a.Config,
		Logger:  a.Logger,
		Storage: a.S3Storage,
	})

	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		if err := a.discordBot.Start(groupCtx); err != nil {
			return fmt.Errorf("start discord bot: %w", err)
		}

		return nil
	})

	group.Go(func() error {
		if err := a.httpServer.Start(groupCtx); err != nil {
			return fmt.Errorf("start http server: %w", err)
		}

		return nil
	})

	return group.Wait()
}

func (a *App) Stop(ctx context.Context) error {
	if err := a.discordBot.Stop(ctx); err != nil {
		return fmt.Errorf("stop discord bot: %w", err)
	}

	if err := a.httpServer.Stop(ctx); err != nil {
		return fmt.Errorf("stop http server: %w", err)
	}

	return nil
}
