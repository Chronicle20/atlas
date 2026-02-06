package handler

import (
	"atlas-login/account"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const AfterLoginHandle = "AfterLoginHandle"

func AfterLoginHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	pinOperationFunc := session.Announce(l)(wp)(writer.PinOperation)
	return func(s session.Model, r *request.Reader) {
		opt1 := r.ReadByte()
		opt2 := byte(0)
		pin := ""
		if opt1 != 0 {
			opt2 = r.ReadByte()
			pin = r.ReadAsciiString()
		}
		l.Debugf("AfterLogin handling opt1 [%d] opt2 [%d].", opt1, opt2)
		if opt1 == 0 && opt2 == 0 {
			l.Debugf("Account [%d] has chosen not to input PIN. Terminating session.", s.AccountId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}

		a, err := account.NewProcessor(l, ctx).GetById(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get account [%d] being acted upon.", s.AccountId())
			return
		}

		if opt1 == 1 && opt2 == 1 {
			if a.PIN() == "" {
				l.Debugf("Requesting account [%d] to create PIN.", s.AccountId())
				err = pinOperationFunc(s, writer.RegisterPinBody(l))
				if err != nil {
					l.WithError(err).Errorf("Unable to write pin operation response due to error.")
					return
				}
				return
			}
			l.Debugf("Requesting account [%d] to input PIN.", s.AccountId())
			err = pinOperationFunc(s, writer.RequestPinBody(l))
			if err != nil {
				l.WithError(err).Errorf("Unable to write pin operation response due to error.")
				return
			}
			return
		}
		if opt1 == 1 && opt2 == 0 {
			if validPin(pin) {
				l.Debugf("Validated account [%d] to PIN.", s.AccountId())
				err = pinOperationFunc(s, writer.AcceptPinBody(l))
				if err != nil {
					l.WithError(err).Errorf("Unable to write pin operation response due to error.")
					return
				}
				return
			}
			l.Debugf("Account [%d] to PIN invalid.", s.AccountId())
			err = pinOperationFunc(s, writer.InvalidPinBody(l))
			if err != nil {
				l.WithError(err).Errorf("Unable to write pin operation response due to error.")
				return
			}
			return
		}
		if opt1 == 2 && opt2 == 0 {
			if validPin(pin) {
				l.Debugf("Requesting account [%d] to create PIN.", s.AccountId())
				err = pinOperationFunc(s, writer.RegisterPinBody(l))
				if err != nil {
					l.WithError(err).Errorf("Unable to write pin operation response due to error.")
					return
				}
				return
			}
			l.Debugf("Account [%d] to PIN invalid.", s.AccountId())
			err = pinOperationFunc(s, writer.InvalidPinBody(l))
			if err != nil {
				l.WithError(err).Errorf("Unable to write pin operation response due to error.")
				return
			}
			return
		}

		l.Warnf("Client should not have gotten here. Terminating session.")
		_ = session.NewProcessor(l, ctx).Destroy(s)
	}
}

func validPin(pin string) bool {
	// TODO terminate on too many attempts.
	return true
}
