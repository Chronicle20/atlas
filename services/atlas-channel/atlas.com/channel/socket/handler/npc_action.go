package handler

import (
	"atlas-channel/character"
	"atlas-channel/npc"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const NPCActionHandle = "NPCActionHandle"

func NPCActionHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	npcActionFunc := session.Announce(wp)(writer.NPCAction)
	return func(s session.Model, r *request.Reader) {
		objectId := r.ReadUint32()
		unk := r.ReadByte()
		unk2 := r.ReadByte()

		c, err := character.GetById(l, span, s.Tenant())(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character seeing npc movement.")
			return
		}
		n, err := npc.GetInMapByObjectId(l, span, s.Tenant())(c.MapId(), objectId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve npc moving.")
			return
		}
		// TODO could validate NPC has ability to move
		rest := r.GetRestAsBytes()
		if len(rest) > 0 {
			err = npcActionFunc(s, writer.NPCActionMoveBody(l)(objectId, unk, unk2, rest))
			if err != nil {
				l.WithError(err).Errorf("Unable to move npc [%d] for character [%d].", n.Template(), s.CharacterId())
				return
			}
			return
		}
		err = npcActionFunc(s, writer.NPCActionAnimationBody(l)(objectId, unk, unk2))
		if err != nil {
			l.WithError(err).Errorf("Unable to animate npc [%d] for character [%d].", n.Template(), s.CharacterId())
			return
		}
	}
}
