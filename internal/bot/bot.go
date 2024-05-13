package bot

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	eventhandler "github.com/kvizyx/voicelog/internal/bot/handler"
	recordsessions "github.com/kvizyx/voicelog/internal/bot/record-sessions"
	"github.com/kvizyx/voicelog/internal/config"
	"github.com/kvizyx/voicelog/internal/storage/s3"
	"github.com/kvizyx/voicelog/pkg/logger"
)

type Bot struct {
	config          config.Config
	logger          logger.Logger
	sessionsManager *recordsessions.SessionsManager

	s3storage *s3.Storage
	botClient bot.Client
}

type Params struct {
	Config    config.Config
	Logger    logger.Logger
	S3Storage *s3.Storage
}

func NewDiscordBot(params Params) Bot {
	return Bot{
		config:    params.Config,
		logger:    params.Logger,
		s3storage: params.S3Storage,
	}
}

func (b *Bot) Start(ctx context.Context) error {
	botClient, err := disgo.New(
		b.config.BotToken,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(Permissions),
			gateway.WithAutoReconnect(true),
		),
		bot.WithCacheConfigOpts(
			// enables voice states caches to get access to old voice states
			cache.WithCaches(cache.FlagVoiceStates),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	b.botClient = botClient

	b.sessionsManager = recordsessions.NewManager(recordsessions.Params{
		Logger:       b.logger,
		S3Storage:    b.s3storage,
		VoiceManager: b.botClient.VoiceManager(),
		DiscordAPI:   b.botClient.Rest(),
	})

	handlerOpts := eventhandler.HandlerOptions{
		Logger:          b.logger,
		SessionsManager: b.sessionsManager,
	}

	botClient.EventManager().AddEventListeners(&events.ListenerAdapter{
		OnGuildChannelCreate: eventhandler.ChannelCreate(handlerOpts),
		OnGuildVoiceJoin:     eventhandler.VoiceJoin(handlerOpts),
		OnGuildVoiceLeave:    eventhandler.VoiceLeave(handlerOpts),
	})

	if err = b.botClient.OpenGateway(ctx); err != nil {
		return fmt.Errorf("failed to connect to discord gateway: %w", err)
	}

	b.logger.Info("discord bot started")

	return nil
}

func (b *Bot) Stop(ctx context.Context) error {
	b.sessionsManager.StopAll(ctx)
	b.botClient.Close(ctx)

	b.logger.Info("discord bot stopped")

	return nil
}
