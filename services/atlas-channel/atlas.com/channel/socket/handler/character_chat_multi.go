package handler

import (
	"atlas-channel/message"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
	"strconv"
)

const CharacterChatMultiHandle = "CharacterChatMultiHandle"

func CharacterChatMultiHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		chatType := r.ReadByte()
		recipientCount := r.ReadByte()
		recipients := make([]uint32, recipientCount)
		recipientStr := ""
		for i := 0; i < int(recipientCount); i++ {
			recipients[i] = r.ReadUint32()
			recipientStr += strconv.Itoa(int(recipients[i]))
		}
		chatText := r.ReadAsciiString()

		l.Debugf("Character [%d] issued message [%s]. type [%d], recipientCount [%d]. recipients [%s].", s.CharacterId(), chatText, chatType, recipientCount, recipientStr)
		if chatType == 0 {
			_ = message.BuddyChat(l)(ctx)(s.Map(), s.CharacterId(), chatText, recipients)
			return
		}
		if chatType == 1 {
			_ = message.PartyChat(l)(ctx)(s.Map(), s.CharacterId(), chatText, recipients)
			return
		}
		if chatType == 2 {
			_ = message.GuildChat(l)(ctx)(s.Map(), s.CharacterId(), chatText, recipients)
			return
		}
		if chatType == 3 {
			_ = message.AllianceChat(l)(ctx)(s.Map(), s.CharacterId(), chatText, recipients)
			return
		}
	}
}
