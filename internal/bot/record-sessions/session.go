package recordsessions

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/kvizyx/cycle"
	"github.com/kvizyx/voicelog/pkg/logger"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
)

type SessionID = snowflake.ID

const (
	sessionTTL = 1 * time.Hour
	recordTTL  = 7 * 24 * time.Hour // 1 week
)

type VoiceUploader interface {
	UploadVoice(
		ctx context.Context,
		ttl time.Duration,
		filePath string,
	) (uuid.UUID, error)
}

type Session struct {
	logger        logger.Logger
	voiceUploader VoiceUploader

	voiceManager voice.Manager
	discordAPI   rest.Rest

	voiceConn    voice.Conn
	recordWriter *oggwriter.OggWriter

	channelNotEmpty atomic.Bool   // does anyone ever joined current voice room
	channelMembers  atomic.Uint32 // current number of voice room members

	guildID   snowflake.ID
	channelID snowflake.ID

	cycle cycle.Cycle
}

func (s *Session) Start() error {
	sessionCtx, cancel := context.WithTimeout(context.Background(), sessionTTL)
	defer cancel()

	s.cycle = cycle.New(
		sessionCtx, cycle.Callbacks{
			OnStart: s.onStart,
			OnStop:  s.onStop,
			OnEvent: s.onEvent,
			Worker:  s.worker,
		},
	)

	startupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.cycle.Start(startupCtx); err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	return nil
}

func (s *Session) worker() {
	opusPacket, err := s.voiceConn.UDP().ReadPacket()
	if err != nil {
		s.logger.Debug("failed to read udp packet", slog.Any("error", err))
		return
	}

	rtpPacket := s.makeRTPPacket(opusPacket)

	if err = s.recordWriter.WriteRTP(rtpPacket); err != nil {
		s.logger.Debug("failed to write packet data to file", slog.Any("error", err))
		return
	}
}

func (s *Session) onStart(ctx context.Context) error {
	s.voiceConn = s.voiceManager.CreateConn(s.guildID)

	if err := s.voiceConn.Open(ctx, s.channelID, true, false); err != nil {
		return fmt.Errorf("failed to connect to voice channel: %w", err)
	}

	if err := s.voiceConn.SetSpeaking(ctx, voice.SpeakingFlagMicrophone); err != nil {
		return fmt.Errorf("failed to send speaking packet: %w", err)
	}

	recordPath := fmt.Sprintf(".tmp/channel%d.ogg", s.channelID)

	recordWriter, err := oggwriter.New(recordPath, 48000, 2)
	if err != nil {
		return fmt.Errorf("failed to create OGG writer: %w", err)
	}

	s.recordWriter = recordWriter

	s.logger.Info("voice recording session started")

	return nil
}

func (s *Session) onStop(ctx context.Context) error {
	recordPath := fmt.Sprintf(".tmp/channel%d.ogg", s.channelID)

	defer func() {
		s.voiceManager.Close(ctx)
		s.voiceManager.RemoveConn(s.guildID)

		_ = os.Remove(recordPath)
	}()

	recordID, err := s.voiceUploader.UploadVoice(ctx, recordTTL, recordPath)
	if err != nil {
		return fmt.Errorf("failed to upload voice record: %w", err)
	}

	s.logger.Info("voice record uploaded", slog.Any("record_id", recordID))

	_, _ = s.discordAPI.CreateMessage(
		s.channelID,
		discord.MessageCreate{
			Content: fmt.Sprintf(`Got it! Download link - http://localhost:8080/api/voices/%s`, recordID.String()),
		},
		rest.WithCtx(ctx),
	)

	if err = s.recordWriter.Close(); err != nil {
		return fmt.Errorf("failed to close record file: %w", err)
	}

	return nil
}

func (s *Session) onEvent(event cycle.Event) {
	switch event.Type() {
	case EventTypeMemberJoin:
		if members := s.channelMembers.Add(1); members == 1 {
			s.channelNotEmpty.Store(true)
		}

	case EventTypeMemberLeave:
		if (s.channelMembers.Load()-1 == 0) && s.channelNotEmpty.Load() {
			s.logger.Debug("channel is empty, stopping session")

			if err := s.cycle.Stop(context.TODO()); err != nil {
				s.logger.Error(
					"failed to stop session gracefully",
					slog.Any("error", err),
				)
			}
		}

		// decrease channel members count by 1.
		s.channelMembers.Add(^uint32(0))
	}
}

func (s *Session) makeRTPPacket(packet *voice.Packet) *rtp.Packet {
	return &rtp.Packet{
		Header: rtp.Header{
			// these values were taken from Discord API documentation
			Version:     2,
			PayloadType: 0x78,

			SequenceNumber: packet.Sequence,
			Timestamp:      packet.Timestamp,
			SSRC:           packet.SSRC,
		},
		Payload: packet.Opus,
	}
}
