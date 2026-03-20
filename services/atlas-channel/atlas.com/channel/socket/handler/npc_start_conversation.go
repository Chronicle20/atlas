package handler

import (
	npcData "atlas-channel/data/npc"
	"atlas-channel/npc"
	"atlas-channel/npc/shops"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	npc2 "github.com/Chronicle20/atlas-packet/npc/serverbound"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func NPCStartConversationHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := npc2.StartConversation{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		oid := p.Oid()

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
		err = npc.NewProcessor(l, ctx).StartConversation(s.Field(), n.Template(), s.CharacterId(), s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Failed to send conversation start command for character [%d] and NPC [%d].", s.CharacterId(), n.Template())
		}
		return
	}
}
