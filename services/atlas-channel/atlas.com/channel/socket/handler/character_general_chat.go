package handler

import (
	"atlas-channel/character"
	"atlas-channel/message"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const CharacterGeneralChatHandle = "CharacterGeneralChatHandle"

func CharacterGeneralChatHandleFunc(l logrus.FieldLogger, span opentracing.Span, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		var updateTime = uint32(0)
		if (s.Tenant().Region == "GMS" && s.Tenant().MajorVersion > 83) || s.Tenant().Region == "JMS" {
			updateTime = r.ReadUint32()
		}
		msg := r.ReadAsciiString()
		bOnlyBalloon := r.ReadBool()

		c, err := character.GetById(l, span, s.Tenant())(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character [%d] chatting.", s.CharacterId())
			return
		}

		l.Debugf("Character [%d] issued message [%s]. updateTime [%d]. bOnlyBalloon [%t].", s.CharacterId(), msg, updateTime, bOnlyBalloon)
		err = message.GeneralChat(l, span, s.Tenant())(s.WorldId(), s.ChannelId(), c.MapId(), s.CharacterId(), msg, bOnlyBalloon)
		if err != nil {
			l.WithError(err).Errorf("Unable to process general chat message for character [%d].", s.CharacterId())
		}
	}
}
