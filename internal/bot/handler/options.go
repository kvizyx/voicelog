package eventhandler

import (
	recordsessions "github.com/kvizyx/voicelog/internal/bot/record-sessions"
	"github.com/kvizyx/voicelog/pkg/logger"
)

type HandlerOptions struct {
	Logger          logger.Logger
	SessionsManager *recordsessions.SessionsManager
}
