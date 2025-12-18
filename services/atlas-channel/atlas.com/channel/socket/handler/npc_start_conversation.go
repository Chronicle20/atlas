package handler

import (
	npcData "atlas-channel/data/npc"
	"atlas-channel/npc"
	"atlas-channel/npc/shops"
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
		n, err := npcData.NewProcessor(l, ctx).GetInMapByObjectId(s.MapId(), oid)
		if err != nil {
			l.WithError(err).Errorf("Character [%d] is interacting with a map object [%d] that is not found in map [%d].", s.CharacterId(), oid, s.MapId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}
		sp := shops.NewProcessor(l, ctx)
		_, err = sp.GetShop(n.Template())
		if err == nil {
			err = sp.EnterShop(s.CharacterId(), n.Template())
			if err != nil {
				l.WithError(err).Errorf("Failed to send shop enter command for character [%d] and NPC [%d].", s.CharacterId(), n.Template())
			}
			return
		}
		err = npc.NewProcessor(l, ctx).StartConversation(s.Map(), n.Template(), s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Failed to send conversation start command for character [%d] and NPC [%d].", s.CharacterId(), n.Template())
		}
		return
	}
}
