package handler

import (
	"atlas-login/account"
	as "atlas-login/account/session"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	account2 "github.com/Chronicle20/atlas-packet/account"
	loginpkt "github.com/Chronicle20/atlas-packet/login"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func RegisterPinHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := account2.RegisterPin{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if !p.PinInput() {
			l.Debugf("Account [%d] opted out of PIN registration. Terminating session.", s.AccountId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
		}

		if len(p.Pin()) < 4 {
			l.Warnf("Read an invalid length pin. Possibly just the bug with inputting pins with leading zeros")
			err := session.Announce(l)(ctx)(wp)(loginpkt.PinOperationWriter)(writer.PinConnectionFailedBody())(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write pin operation response due to error.")
				return
			}
			return
		}

		if len(p.Pin()) > 4 {
			l.Warnf("Read an invalid length pin. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}

		l.Debugf("Registering PIN for account [%d].", s.AccountId())
		err := account.NewProcessor(l, ctx).UpdatePin(s.AccountId(), p.Pin())
		if err != nil {
			l.WithError(err).Errorf("Error updating PIN for account [%d].", s.AccountId())
			err = session.Announce(l)(ctx)(wp)(loginpkt.PinOperationWriter)(writer.PinConnectionFailedBody())(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write pin operation response due to error.")
				return
			}
			return
		}

		err = session.Announce(l)(ctx)(wp)(loginpkt.PinUpdateWriter)(writer.PinUpdateBody(writer.PinUpdateModeOk))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to write pin update response due to error.")
			return
		}

		l.Debugf("Logging account out, as they are still at login screen and need to issue a new request.")
		as.NewProcessor(l, ctx).Destroy(s.SessionId(), s.AccountId())
	}
}
