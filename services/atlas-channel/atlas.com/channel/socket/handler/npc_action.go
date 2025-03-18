package handler

import (
	"atlas-channel/npc"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const NPCActionHandle = "NPCActionHandle"

func NPCActionHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		objectId := r.ReadUint32()
		unk := r.ReadByte()
		unk2 := r.ReadByte()

		n, err := npc.GetInMapByObjectId(l)(ctx)(s.MapId(), objectId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve npc moving.")
			return
		}
		// TODO could validate NPC has ability to move
		rest := r.GetRestAsBytes()
		if len(rest) > 0 {
			mp := model.Movement{}
			mp.Decode(l, t, readerOptions)(r)
			err = session.Announce(l)(ctx)(wp)(writer.NPCAction)(writer.NPCActionMoveBody(l, t)(objectId, unk, unk2, mp))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to move npc [%d] for character [%d].", n.Template(), s.CharacterId())
			}
			return
		}
		err = session.Announce(l)(ctx)(wp)(writer.NPCAction)(writer.NPCActionAnimationBody(l)(objectId, unk, unk2))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to animate npc [%d] for character [%d].", n.Template(), s.CharacterId())
			return
		}
	}
}
