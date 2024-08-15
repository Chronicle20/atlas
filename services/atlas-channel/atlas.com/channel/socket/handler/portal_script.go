package handler

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/portal"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const PortalScriptHandle = "PortalScriptHandle"

func PortalScriptHandleFunc(l logrus.FieldLogger, span opentracing.Span, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		fieldKey := r.ReadByte()
		portalName := r.ReadAsciiString()
		x := r.ReadInt16()
		y := r.ReadInt16()
		l.Debugf("Character [%d] attempting to execute portal script for [%s] at [%d,%d]. FieldKey [%d].", s.CharacterId(), portalName, x, y, fieldKey)

		_ = portal.Enter(l, span, producer.ProviderImpl(l)(span))(s.Tenant(), s.WorldId(), s.ChannelId(), s.MapId(), portalName, s.CharacterId())
	}
}
