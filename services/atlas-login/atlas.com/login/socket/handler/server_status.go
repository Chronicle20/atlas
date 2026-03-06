package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"context"

	world2 "github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const ServerStatusHandle = "ServerStatusHandle"

func ServerStatusHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		worldId := world2.Id(r.ReadUint16())

		cs := world.NewProcessor(l, ctx).GetCapacityStatus(worldId)
		err := session.Announce(l)(ctx)(wp)(writer.ServerStatus)(writer.ServerStatusBody(cs))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to issue world capacity status information")
		}
	}
}
