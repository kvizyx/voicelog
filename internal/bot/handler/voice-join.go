package eventhandler

import (
	"log/slog"

	"github.com/disgoorg/disgo/events"
	recordsessions "github.com/kvizyx/voicelog/internal/bot/record-sessions"
)

type VoiceJoinHandler func(event *events.GuildVoiceJoin)

func VoiceJoin(o HandlerOptions) VoiceJoinHandler {
	return func(event *events.GuildVoiceJoin) {
		if event.Member.User.Bot {
			return
		}

		channelID := event.VoiceState.ChannelID
		if channelID == nil {
			o.Logger.Debug(
				"chanel id is empty, will not send event",
				slog.String("event", "guild_voice_join"),
			)
			return
		}

		o.SessionsManager.SendEvent(*channelID, recordsessions.EventMemberJoin{})
	}
}
