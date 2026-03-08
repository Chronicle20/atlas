package handler

import (
	as "atlas-login/account/session"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"
	"encoding/hex"
	"net"

	"github.com/Chronicle20/atlas-packet/login"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func LoginHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := login.Request{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		ipAddress := ""
		if addr := s.GetRemoteAddress(); addr != nil {
			if tcpAddr, ok := addr.(*net.TCPAddr); ok {
				ipAddress = tcpAddr.IP.String()
			} else {
				host, _, err := net.SplitHostPort(addr.String())
				if err == nil {
					ipAddress = host
				}
			}
		}
		hwid := hex.EncodeToString(p.HWID())

		err := as.NewProcessor(l, ctx).Create(s.SessionId(), s.AccountId(), p.Name(), p.Password(), ipAddress, hwid)
		if err != nil {
			authLoginFailedFunc := session.Announce(l)(ctx)(wp)(writer.AuthLoginFailed)
			err = authLoginFailedFunc(writer.AuthLoginFailedBody(writer.SystemError1))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to issue [%s].", writer.AuthLoginFailed)
			}
			return
		}
	}
}
