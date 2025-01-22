package handler

import (
	"atlas-channel/npc"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const NPCStartConversationHandle = "NPCStartConversationHandle"

func NPCStartConversationHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		oid := r.ReadUint32()
		n, err := npc.GetInMapByObjectId(l)(ctx)(s.MapId(), oid)
		if err != nil {
			l.WithError(err).Errorf("Character [%d] is interacting with a map object [%d] that is not found in map [%d].", s.CharacterId(), oid, s.MapId())
			_ = session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}
		_ = npc.StartConversation(l)(ctx)(s.WorldId(), s.ChannelId(), s.MapId(), n.Template(), s.CharacterId())

	}
}
