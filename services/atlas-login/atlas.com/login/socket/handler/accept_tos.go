package handler

import (
	"atlas-login/account"
	"atlas-login/configuration"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	account2 "github.com/Chronicle20/atlas-packet/account"
	loginpkt "github.com/Chronicle20/atlas-packet/login"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func AcceptTosHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := account2.AcceptTos{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if !p.Accepted() {
			l.Debugf("Account [%d] has chosen not to accept TOS. Terminating session.", s.AccountId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}

		err := account.NewProcessor(l, ctx).UpdateTos(s.AccountId(), p.Accepted())
		if err != nil {
			// TODO
		}
		account.NewProcessor(l, ctx).ForAccountById(s.AccountId(), issueSuccess(l)(ctx)(wp)(s))
	}
}

func issueSuccess(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[account.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[account.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(s session.Model) model.Operator[account.Model] {
			authSuccessFunc := session.Announce(l)(ctx)(wp)(loginpkt.AuthSuccessWriter)
			return func(s session.Model) model.Operator[account.Model] {
				return func(a account.Model) error {
					sc, err := configuration.GetTenantConfig(t.Id())
					if err != nil {
						l.WithError(err).Errorf("Unable to find server configuration.")
						return err
					}

					err = authSuccessFunc(writer.AuthSuccessBody(a.Id(), a.Name(), a.Gender(), sc.UsesPin, a.PIC()))(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to show successful authorization for account %d", a.Id())
					}
					return err
				}
			}
		}
	}
}
