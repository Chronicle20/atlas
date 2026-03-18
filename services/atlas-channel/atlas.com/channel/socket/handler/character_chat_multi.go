package handler

import (
	"atlas-channel/message"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	chat "github.com/Chronicle20/atlas-packet/chat/serverbound"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterChatMultiHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := chat.Multi{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		mp := message.NewProcessor(l, ctx)
		if p.ChatType() == 0 {
			_ = mp.BuddyChat(s.Field(), s.CharacterId(), p.ChatText(), p.Recipients())
			return
		}
		if p.ChatType() == 1 {
			_ = mp.PartyChat(s.Field(), s.CharacterId(), p.ChatText(), p.Recipients())
			return
		}
		if p.ChatType() == 2 {
			_ = mp.GuildChat(s.Field(), s.CharacterId(), p.ChatText(), p.Recipients())
			return
		}
		if p.ChatType() == 3 {
			_ = mp.AllianceChat(s.Field(), s.CharacterId(), p.ChatText(), p.Recipients())
			return
		}
	}
}
