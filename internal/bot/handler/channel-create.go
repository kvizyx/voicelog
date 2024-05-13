package eventhandler

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

type ChannelCreateHandler func(event *events.GuildChannelCreate)

func ChannelCreate(o HandlerOptions) ChannelCreateHandler {
	return func(event *events.GuildChannelCreate) {
		if event.Channel.Type() != discord.ChannelTypeGuildVoice {
			return
		}

		if err := o.SessionsManager.Spawn(event.GuildID, event.ChannelID); err != nil {
			o.Logger.Error("failed to spawn recording session", slog.Any("error", err))
		}
	}
}
