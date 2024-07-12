package handler

import (
	"atlas-login/account"
	as "atlas-login/account/session"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const RegisterPinHandle = "RegisterPinHandle"

func RegisterPinHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	pinOperationFunc := session.Announce(wp)(writer.PinOperation)
	pinUpdateFunc := session.Announce(wp)(writer.PinUpdate)
	return func(s session.Model, r *request.Reader) {
		opt := r.ReadByte()
		if opt == 0 {
			l.Debugf("Account [%d] opted out of PIN registration. Terminating session.", s.AccountId())
			session.Destroy(l, span, session.GetRegistry())(s)
		}

		if opt == 1 {
			pin := r.ReadAsciiString()
			l.Debugf("Registering PIN [%s] for account [%d].", pin, s.AccountId())
			err := account.UpdatePin(l, span, s.Tenant())(s.AccountId(), pin)
			if err != nil {
				l.WithError(err).Errorf("Error updating PIN for account [%d].", s.AccountId())
				err = pinOperationFunc(s, writer.PinConnectionFailedBody(l))
				if err != nil {
					l.WithError(err).Errorf("Unable to write pin operation response due to error.")
					return
				}
				return
			}

			err = pinUpdateFunc(s, writer.PinUpdateBody(l)(writer.PinUpdateModeOk))
			if err != nil {
				l.WithError(err).Errorf("Unable to write pin update response due to error.")
				return
			}

			l.Debugf("Logging account out, as they are still at login screen and need to issue a new request.")
			as.Destroy(l, span, s.Tenant())(s.AccountId())
			return
		}
		l.Warnf("Unhandled opt [%d] for PIN registration. Terminating session.", opt)
		session.Destroy(l, span, session.GetRegistry())(s)
	}
}
