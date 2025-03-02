package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterChatWhisperHandle = "CharacterChatWhisperHandle"

func CharacterChatWhisperHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		mode := r.ReadByte()
		updateTime := r.ReadUint32()
		targetName := r.ReadAsciiString()
		message := r.ReadAsciiString()
		l.Debugf("Character [%d] using whipser mode [%d]. Target [%s], Message [%s], UpdateTime [%d]", s.CharacterId(), mode, targetName, message, updateTime)
	}
}
