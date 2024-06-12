package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const ServerStatusHandle = "ServerStatusHandle"

func ServerStatusHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	serverStatusFunc := session.Announce(wp)(writer.ServerStatus)
	return func(s session.Model, r *request.Reader) {
		worldId := byte(r.ReadUint16())

		cs := world.GetCapacityStatus(l, span)(worldId)
		err := serverStatusFunc(s, writer.ServerStatusBody(l)(cs))
		if err != nil {
			l.WithError(err).Errorf("Unable to issue world capacity status information")
		}
	}
}
