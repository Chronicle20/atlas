package handler

import (
	"atlas-channel/data/npc"
	_map "atlas-channel/map"
	"atlas-channel/movement"
	controllernpc "atlas-channel/npc/controller"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	npcpacket "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	npcsb "github.com/Chronicle20/atlas/libs/atlas-packet/npc/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func NPCActionHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := npcsb.ActionRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if p.HasMovement() {
			_ = movement.NewProcessor(l, ctx, wp).ForNPC(s.Field(), s.CharacterId(), p.ObjectId(), p.Unk(), p.Unk2(), p.MovementData())
			return
		}

		n, err := npc.NewProcessor(l, ctx).GetInMapByObjectId(s.MapId(), p.ObjectId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve npc moving.")
			return
		}
		if !controllernpc.IsController(ctx, tenant.MustFromContext(ctx), s.Field(), s.CharacterId(), p.ObjectId()) {
			l.Debugf("Dropping NPC [%d] animation from non-controller [%d].", p.ObjectId(), s.CharacterId())
			return
		}
		op := session.Announce(l)(ctx)(wp)(npcpacket.NpcActionWriter)(npcpacket.NewNpcActionAnimation(p.ObjectId(), p.Unk(), p.Unk2()).Encode)
		if err = op(s); err != nil {
			l.WithError(err).Errorf("Unable to animate npc [%d] for character [%d].", n.Template(), s.CharacterId())
			return
		}
		if rerr := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), op); rerr != nil {
			l.WithError(rerr).Errorf("Unable to relay npc [%d] animation to field.", p.ObjectId())
		}
	}
}
