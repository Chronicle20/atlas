package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const CharacterMoveHandle = "CharacterMoveHandle"

func CharacterMoveHandleFunc(l logrus.FieldLogger, span opentracing.Span, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		var dr0 uint32
		var dr1 uint32
		if (s.Tenant().Region == "GMS" && s.Tenant().MajorVersion > 83) || s.Tenant().Region == "JMS" {
			dr0 = r.ReadUint32()
			dr1 = r.ReadUint32()
		}
		fieldKey := r.ReadByte()

		var dr2 uint32
		var dr3 uint32
		if (s.Tenant().Region == "GMS" && s.Tenant().MajorVersion > 83) || s.Tenant().Region == "JMS" {
			dr2 = r.ReadUint32()
			dr3 = r.ReadUint32()
		}

		var crc uint32
		if (s.Tenant().Region == "GMS" && s.Tenant().MajorVersion > 28) || s.Tenant().Region == "JMS" {
			crc = r.ReadUint32()
		}

		var dwKey uint32
		var crc32 uint32
		if (s.Tenant().Region == "GMS" && s.Tenant().MajorVersion > 83) || s.Tenant().Region == "JMS" {
			dwKey = r.ReadUint32()
			crc32 = r.ReadUint32()
		}

		l.Debugf("Character [%d] has moved. dr0 [%d], dr1 [%d], fieldKey [%d], dr2 [%d], dr3 [%d], crc [%d], dwKey [%d], crc32 [%d].", s.CharacterId(), dr0, dr1, fieldKey, dr2, dr3, crc, dwKey, crc32)

		mp := model.Movement{}
		mp.Decode(l, s.Tenant(), readerOptions)(r)
		character.Move(l, span, s.Tenant())(s.WorldId(), s.ChannelId(), s.MapId(), s.CharacterId(), mp)
	}
}
