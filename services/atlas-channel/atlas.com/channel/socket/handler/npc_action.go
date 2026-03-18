package handler

import (
	"atlas-channel/data/npc"
	"atlas-channel/movement"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	npcpacket "github.com/Chronicle20/atlas-packet/npc/clientbound"
	npcsb "github.com/Chronicle20/atlas-packet/npc/serverbound"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
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
		err = session.Announce(l)(ctx)(wp)(npcpacket.NpcActionWriter)(npcpacket.NewNpcActionAnimation(p.ObjectId(), p.Unk(), p.Unk2()).Encode)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to animate npc [%d] for character [%d].", n.Template(), s.CharacterId())
			return
		}
	}
}
