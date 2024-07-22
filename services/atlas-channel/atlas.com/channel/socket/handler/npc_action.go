package handler

import (
	"atlas-channel/character"
	"atlas-channel/npc"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const NPCActionHandle = "NPCActionHandle"

func NPCActionHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	npcActionFunc := session.Announce(l)(wp)(writer.NPCAction)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
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
			mp := model.Movement{}
			mp.Decode(l, s.Tenant(), readerOptions)(r)
			err = npcActionFunc(s, writer.NPCActionMoveBody(l, s.Tenant())(objectId, unk, unk2, mp))
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
