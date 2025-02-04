package handler

import (
	"atlas-login/account"
	"atlas-login/configuration"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const AcceptTosHandle = "AcceptTosHandle"

func AcceptTosHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	return func(s session.Model, r *request.Reader) {
		accepted := r.ReadBool()
		l.Debugf("Account [%d] responded to the TOS dialog with [%t].", s.AccountId(), accepted)
		if !accepted {
			l.Debugf("Account [%d] has chosen not to accept TOS. Terminating session.", s.AccountId())
			_ = session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}

		err := account.UpdateTos(l, ctx)(s.AccountId(), accepted)
		if err != nil {
			// TODO
		}
		account.ForAccountById(l, ctx)(s.AccountId(), issueSuccess(l, s, wp))
	}
}

func issueSuccess(l logrus.FieldLogger, s session.Model, wp writer.Producer) model.Operator[account.Model] {
	authSuccessFunc := session.Announce(l)(wp)(writer.AuthSuccess)
	return func(a account.Model) error {
		c, err := configuration.Get()
		if err != nil {
			l.WithError(err).Errorf("Unable to get configuration.")
			return err
		}
		t := s.Tenant()
		sc, err := c.FindServer(t.Id())
		if err != nil {
			l.WithError(err).Errorf("Unable to find server configuration.")
			return err
		}

		err = authSuccessFunc(s, writer.AuthSuccessBody(t)(a.Id(), a.Name(), a.Gender(), sc.UsesPIN, a.PIC()))
		if err != nil {
			l.WithError(err).Errorf("Unable to show successful authorization for account %d", a.Id())
		}
		return err
	}
}
