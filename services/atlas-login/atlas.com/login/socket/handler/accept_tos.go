package handler

import (
	"atlas-login/account"
	"atlas-login/configuration"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const AcceptTosHandle = "AcceptTosHandle"

func AcceptTosHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	ap := account.NewProcessor(l, ctx)
	return func(s session.Model, r *request.Reader) {
		accepted := r.ReadBool()
		l.Debugf("Account [%d] responded to the TOS dialog with [%t].", s.AccountId(), accepted)
		if !accepted {
			l.Debugf("Account [%d] has chosen not to accept TOS. Terminating session.", s.AccountId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}

		err := ap.UpdateTos(s.AccountId(), accepted)
		if err != nil {
			// TODO
		}
		ap.ForAccountById(s.AccountId(), issueSuccess(l)(ctx)(wp)(s))
	}
}

func issueSuccess(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[account.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[account.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(s session.Model) model.Operator[account.Model] {
			authSuccessFunc := session.Announce(l)(wp)(writer.AuthSuccess)
			return func(s session.Model) model.Operator[account.Model] {
				return func(a account.Model) error {
					sc, err := configuration.GetTenantConfig(t.Id())
					if err != nil {
						l.WithError(err).Errorf("Unable to find server configuration.")
						return err
					}

					err = authSuccessFunc(s, writer.AuthSuccessBody(t)(a.Id(), a.Name(), a.Gender(), sc.UsesPin, a.PIC()))
					if err != nil {
						l.WithError(err).Errorf("Unable to show successful authorization for account %d", a.Id())
					}
					return err
				}
			}
		}
	}
}
