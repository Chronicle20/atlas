package handler

import (
	"atlas-channel/npc"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const NPCStartConversationHandle = "NPCStartConversationHandle"

func NPCStartConversationHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		oid := r.ReadUint32()
		p := npc.NewProcessor(l, ctx)
		n, err := p.GetInMapByObjectId(s.MapId(), oid)
		if err != nil {
			l.WithError(err).Errorf("Character [%d] is interacting with a map object [%d] that is not found in map [%d].", s.CharacterId(), oid, s.MapId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}
		nsm, err := p.GetShop(n.Template())
		if err == nil {
			bp := writer.NPCShopBody(l, tenant.MustFromContext(ctx))(n.Template(), nsm.Commodities())
			_ = session.Announce(l)(ctx)(wp)(writer.NPCShop)(bp)(s)
		}
		_ = p.StartConversation(s.Map(), n.Template(), s.CharacterId())
	}
}
