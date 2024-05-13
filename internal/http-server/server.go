package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kvizyx/voicelog/internal/bot"
	"github.com/kvizyx/voicelog/internal/config"
	"github.com/kvizyx/voicelog/internal/storage/s3"
	"github.com/kvizyx/voicelog/pkg/logger"
)

type Server struct {
	server *http.Server
	router *gin.Engine

	config          config.Config
	logger          logger.Logger
	voiceDownloader VoiceDownloader
}

type VoiceDownloader interface {
	DownloadVoice(ctx context.Context, voiceID uuid.UUID) (io.ReadCloser, error)
}

type Params struct {
	Config  config.Config
	Logger  logger.Logger
	Storage *s3.Storage
}

func NewServer(p Params) Server {
	router := gin.Default()

	server := &http.Server{
		Addr: fmt.Sprintf(
			"0.0.0.0:%d",
			p.Config.HTTP.Port,
		),
		IdleTimeout:  p.Config.HTTP.IdleTimeout,
		ReadTimeout:  p.Config.HTTP.ReadTimeout,
		WriteTimeout: p.Config.HTTP.WriteTimeout,
	}

	return Server{
		server: server,
		router: router,

		config:          p.Config,
		logger:          p.Logger,
		voiceDownloader: p.Storage,
	}
}

func (s *Server) Start(ctx context.Context) error {
	http.HandleFunc("GET /api/discord/invite-link", func(w http.ResponseWriter, r *http.Request) {
		inviteLink := fmt.Sprintf(
			"https://discord.com/oauth2/authorize?client_id=%s&permissions=%d&scope=bot",
			s.config.Discord.ClientID, bot.Permissions,
		)

		http.Redirect(w, r, inviteLink, http.StatusFound)
	})

	http.HandleFunc("GET /api/voices/{id}", func(w http.ResponseWriter, r *http.Request) {
		voiceID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid voice id", http.StatusBadRequest)
			return
		}

		voiceSrc, err := s.voiceDownloader.DownloadVoice(r.Context(), voiceID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to download voice: %s", err), http.StatusInternalServerError)
			return
		}
		defer voiceSrc.Close() // nolint: errcheck

		w.Header().Set("Content-Type", "audio/ogg")

		if _, err = io.Copy(w, voiceSrc); err != nil {
			http.Error(w, fmt.Sprintf("failed to transfer voice: %s", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	s.logger.Info("http server started")

	if err := s.server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	s.logger.Info("http server stopped")

	return nil
}
