package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const WorldSelectHandle = "WorldSelectHandle"

func WorldSelectHandleFunc(l logrus.FieldLogger, _ context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	serverLoadFunc := session.Announce(l)(wp)(writer.ServerLoad)
	return func(s session.Model, r *request.Reader) {
		worldId := r.ReadByte()
		l.Debugf("Reading [%s] message. body={worldId=%d}", WorldSelectHandle, worldId)
		err := serverLoadFunc(s, writer.ServerLoadBody(l)(writer.ServerLoadCodeOk))
		if err != nil {
			l.WithError(err).Errorf("Unable to issue request server load")
		}
	}
}
