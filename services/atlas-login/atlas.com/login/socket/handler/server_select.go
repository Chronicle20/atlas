package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const WorldSelectHandle = "WorldSelectHandle"

func WorldSelectHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		worldId := r.ReadByte()
		l.Debugf("Reading [%s] message. body={worldId=%d}", WorldSelectHandle, worldId)
		err := session.Announce(l)(ctx)(wp)(writer.ServerLoad)(writer.ServerLoadBody(writer.ServerLoadCodeOk))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to issue request server load")
		}
	}
}
