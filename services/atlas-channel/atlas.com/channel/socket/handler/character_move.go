package handler

import (
	"atlas-channel/movement"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterMoveHandle = "CharacterMoveHandle"

func CharacterMoveHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		var dr0 uint32
		var dr1 uint32

		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			dr0 = r.ReadUint32()
			dr1 = r.ReadUint32()
		}
		fieldKey := r.ReadByte()

		var dr2 uint32
		var dr3 uint32
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			dr2 = r.ReadUint32()
			dr3 = r.ReadUint32()
		}

		var crc uint32
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			crc = r.ReadUint32()
		}

		var dwKey uint32
		var crc32 uint32
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			dwKey = r.ReadUint32()
			crc32 = r.ReadUint32()
		}

		l.Debugf("Character [%d] has moved. dr0 [%d], dr1 [%d], fieldKey [%d], dr2 [%d], dr3 [%d], crc [%d], dwKey [%d], crc32 [%d].", s.CharacterId(), dr0, dr1, fieldKey, dr2, dr3, crc, dwKey, crc32)

		mp := model.Movement{}
		mp.Decode(l, t, readerOptions)(r)
		_ = movement.NewProcessor(l, ctx, wp).ForCharacter(s.Field(), s.CharacterId(), mp)
	}
}
