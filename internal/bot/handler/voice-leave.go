package eventhandler

import (
	"log/slog"

	"github.com/disgoorg/disgo/events"
	recordsessions "github.com/kvizyx/voicelog/internal/bot/record-sessions"
)

type VoiceLeaveHandler func(event *events.GuildVoiceLeave)

func VoiceLeave(o HandlerOptions) VoiceLeaveHandler {
	return func(event *events.GuildVoiceLeave) {
		if event.Member.User.Bot {
			return
		}

		channelID := event.OldVoiceState.ChannelID
		if channelID == nil {
			o.Logger.Debug(
				"chanel id is empty, will not send event",
				slog.String("event", "guild_voice_leave"),
			)
			return
		}

		o.SessionsManager.SendEvent(*channelID, recordsessions.EventMemberLeave{})
	}
}
