package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const ServerStatusHandle = "ServerStatusHandle"

func ServerStatusHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	serverStatusFunc := session.Announce(l)(wp)(writer.ServerStatus)
	return func(s session.Model, r *request.Reader) {
		worldId := byte(r.ReadUint16())

		cs := world.NewProcessor(l, ctx).GetCapacityStatus(worldId)
		err := serverStatusFunc(s, writer.ServerStatusBody(cs))
		if err != nil {
			l.WithError(err).Errorf("Unable to issue world capacity status information")
		}
	}
}
