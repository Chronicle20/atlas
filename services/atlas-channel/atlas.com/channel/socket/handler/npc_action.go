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
	npcActionFunc := session.Announce(wp)(writer.NPCAction)
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

func readElementGMS87(r *request.Reader, m *model.Element) {
	m.ElemType = r.ReadByte()
	switch m.ElemType {
	case 0:
	case 5:
	case 0xF:
	case 0x11:
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.Fh = r.ReadInt16()
		if m.ElemType == 15 {
			m.FhFallStart = r.ReadInt16()
		}
		m.XOffset = r.ReadInt16()
		m.YOffset = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	case 3:
	case 4:
	case 7:
	case 8:
	case 9:
	case 0xB:
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Fh = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	case 0xE:
		m.X = m.StartX
		m.Y = m.StartY
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.FhFallStart = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	case 0x17:
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	case 1:
	case 2:
	case 6:
	case 0xC:
	case 0xD:
	case 0x10:
	case 0x12:
	case 0x13:
	case 0x14:
	case 0x16:
	case 0x18:
		m.X = m.StartX
		m.Y = m.StartY
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	case 0xA:
		m.BStat = r.ReadByte()
	default:
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	}
}

func readElementJMS185(r *request.Reader, m *model.Element) {
	m.ElemType = r.ReadByte()
	switch m.ElemType {
	case 0:
	case 5:
	case 0xF:
	case 0x11:
	case 0x1F:
	case 0x20:
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.Fh = r.ReadInt16()
		if m.ElemType == 15 {
			m.FhFallStart = r.ReadInt16()
		}
		m.XOffset = r.ReadInt16()
		m.YOffset = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	case 3:
	case 4:
	case 7:
	case 8:
	case 9:
	case 0xB:
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Fh = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	case 0xE:
		m.X = m.StartX
		m.Y = m.StartY
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.FhFallStart = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	case 1:
	case 2:
	case 6:
	case 0xC:
	case 0xD:
	case 0x10:
	case 0x12:
	case 0x13:
	case 0x14:
	case 0x17:
	case 0x19:
	case 0x1B:
	case 0x1C:
	case 0x1D:
	case 0x1E:
		m.X = m.StartX
		m.Y = m.StartY
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	case 0xA:
		m.BStat = r.ReadByte()
	default:
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	}
}
