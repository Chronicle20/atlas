package handler

import (
	"atlas-channel/npc"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const NPCContinueConversationHandle = "NPCContinueConversationHandle"

func NPCContinueConversationHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		lastMessageType := r.ReadByte()
		action := r.ReadByte()
		//returnText := ""
		selection := int32(-1)

		if lastMessageType == 2 {
			if action != 0 {
				_ = r.ReadAsciiString()
				// TODO handle quest in progress, continue quest

				//TODO set return text
				_ = npc.NewProcessor(l, ctx).ContinueConversation(s.CharacterId(), action, lastMessageType, selection)
				return
			}
			// TODO handle quest in progress, dispose
			_ = npc.NewProcessor(l, ctx).DisposeConversation(s.CharacterId())
			return
		} else {
			if len(r.GetRestAsBytes()) >= 4 {
				selection = r.ReadInt32()
			} else {
				selection = int32(r.ReadByte())
			}
			// TODO handle quest in progress, continue quest
			_ = npc.NewProcessor(l, ctx).ContinueConversation(s.CharacterId(), action, lastMessageType, selection)
		}
	}
}
