package handler

import (
	"atlas-channel/character"
	"atlas-channel/portal"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const MapChangeHandle = "MapChangeHandle"

func MapChangeHandleFunc(l logrus.FieldLogger, span opentracing.Span, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		cs := r.Available() == 0
		var fieldKey byte
		var targetId uint32
		var portalName string
		var x int16
		var y int16
		var unused byte
		var premium byte
		var chase bool
		var targetX int32
		var targetY int32

		if !cs {
			fieldKey = r.ReadByte()
			targetId = r.ReadUint32()
			portalName = r.ReadAsciiString()
			x = r.ReadInt16()
			y = r.ReadInt16()
			unused = r.ReadByte()
			premium = r.ReadByte()
			if s.Tenant().Region == "GMS" && s.Tenant().MajorVersion >= 83 {
				chase = r.ReadBool()
			}
			if chase {
				targetX = r.ReadInt32()
				targetY = r.ReadInt32()
			}
		}

		l.Debugf("Character [%d] attempting to enter portal [%s] at [%d,%d] heading to [%d]. FieldKey [%d].", s.CharacterId(), portalName, x, y, targetId, fieldKey)
		l.Debugf("Unused [%d], Premium [%d], Chase [%t], TargetX [%d], TargetY [%d]", unused, premium, chase, targetX, targetY)
		c, err := character.GetById(l, span, s.Tenant())(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character [%d].", s.CharacterId())
			return
		}
		_ = portal.Enter(l, span, s.Tenant())(s.WorldId(), s.ChannelId(), c.MapId(), portalName, s.CharacterId())
	}
}
