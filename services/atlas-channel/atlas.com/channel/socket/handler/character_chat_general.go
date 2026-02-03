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

const CharacterChatGeneralHandle = "CharacterChatGeneralHandle"

func CharacterChatGeneralHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		var updateTime = uint32(0)

		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			updateTime = r.ReadUint32()
		}
		msg := r.ReadAsciiString()
		bOnlyBalloon := r.ReadBool()

		l.Debugf("Character [%d] issued message [%s]. updateTime [%d]. bOnlyBalloon [%t].", s.CharacterId(), msg, updateTime, bOnlyBalloon)
		err := message.NewProcessor(l, ctx).GeneralChat(s.Field(), s.CharacterId(), msg, bOnlyBalloon)
		if err != nil {
			l.WithError(err).Errorf("Unable to process general chat message for character [%d].", s.CharacterId())
		}
	}
}
