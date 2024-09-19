package handler

import (
	"atlas-login/account"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"
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
			session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}

		err := account.UpdateTos(l, ctx)(s.AccountId(), accepted)
		if err != nil {
			// TODO
		}
		account.ForAccountById(l, ctx)(s.AccountId(), issueSuccess(l, s, wp))
	}
}
