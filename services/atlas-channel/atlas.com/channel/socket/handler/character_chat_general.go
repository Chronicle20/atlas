package handler

import (
	"atlas-channel/message"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	chat "github.com/Chronicle20/atlas/libs/atlas-packet/chat/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterChatGeneralHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := chat.General{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		err := message.NewProcessor(l, ctx).GeneralChat(s.Field(), s.CharacterId(), p.Msg(), p.BOnlyBalloon())
		if err != nil {
			l.WithError(err).Errorf("Unable to process general chat message for character [%d].", s.CharacterId())
		}
	}
}
