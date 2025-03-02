package handler

import (
	"atlas-channel/message"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterChatWhisperHandle = "CharacterChatWhisperHandle"

type WhisperMode byte

const (
	WhisperModeChat = WhisperMode(6)
)

func CharacterChatWhisperHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		mode := WhisperMode(r.ReadByte())
		updateTime := uint32(0)
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			updateTime = r.ReadUint32()
		}
		targetName := r.ReadAsciiString()
		msg := ""

		if mode == WhisperModeChat {
			msg = r.ReadAsciiString()
			err := message.WhisperChat(l)(ctx)(s.Map(), s.CharacterId(), msg, targetName)
			if err != nil {
				// TODO whisper error response.
				return
			}
			return
		}
		l.Debugf("Character [%d] using whipser mode [%d]. Target [%s], Message [%s], UpdateTime [%d]", s.CharacterId(), mode, targetName, msg, updateTime)
	}
}
