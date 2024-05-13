package recordsessions

import (
	"context"
	"log/slog"
	"sync"

	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
	"github.com/kvizyx/cycle"
	"github.com/kvizyx/voicelog/internal/storage/s3"
	"github.com/kvizyx/voicelog/pkg/logger"
)

type SessionsManager struct {
	Params

	sessions map[SessionID]*Session
	mu       *sync.RWMutex
}

type Params struct {
	Logger       logger.Logger
	S3Storage    *s3.Storage
	VoiceManager voice.Manager
	DiscordAPI   rest.Rest
}

func NewManager(params Params) *SessionsManager {
	return &SessionsManager{
		Params: params,

		sessions: make(map[SessionID]*Session),
		mu:       &sync.RWMutex{},
	}
}

func (sm *SessionsManager) Spawn(guildID, channelID snowflake.ID) error {
	sessionLogger := sm.Logger.With(
		slog.Any("guild_id", guildID),
		slog.Any("channel_id", channelID),
	)

	session := &Session{
		logger:        sessionLogger,
		voiceUploader: sm.S3Storage,
		voiceManager:  sm.VoiceManager,
		discordAPI:    sm.DiscordAPI,

		guildID:   guildID,
		channelID: channelID,
	}

	sm.mu.Lock()
	sm.sessions[channelID] = session
	sm.mu.Unlock()

	go func() {
		defer func() {
			sm.mu.Lock()
			delete(sm.sessions, channelID)
			sm.mu.Unlock()
		}()

		if err := session.Start(); err != nil {
			sm.Logger.Error("failed to start recording session", slog.Any("error", err))
		}
	}()

	return nil
}

// SendEvent send event to session with given id.
func (sm *SessionsManager) SendEvent(channelID SessionID, event cycle.Event) {
	sm.mu.RLock()
	session, found := sm.sessions[channelID]
	sm.mu.RUnlock()

	if !found {
		return
	}

	session.cycle.SendEvent(event)
}

// StopAll stop all voice recording sessions gracefully.
func (sm *SessionsManager) StopAll(ctx context.Context) {
	wg := &sync.WaitGroup{}

	for _, session := range sm.sessions {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := session.cycle.Stop(ctx); err != nil {
				sm.Logger.Error(
					"failed to stop recording session gracefully",
					slog.Any("error", err),
				)
			}
		}()
	}

	wg.Wait()
}
