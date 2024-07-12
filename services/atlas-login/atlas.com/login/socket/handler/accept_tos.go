package handler

import (
	"atlas-login/account"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const AcceptTosHandle = "AcceptTosHandle"

func AcceptTosHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	return func(s session.Model, r *request.Reader) {
		accepted := r.ReadBool()
		l.Debugf("Account [%d] responded to the TOS dialog with [%t].", s.AccountId(), accepted)
		if !accepted {
			l.Debugf("Account [%d] has chosen not to accept TOS. Terminating session.", s.AccountId())
			session.Destroy(l, span, session.GetRegistry())(s)
			return
		}

		err := account.UpdateTos(l, span, s.Tenant())(s.AccountId(), accepted)
		if err != nil {

		}
		account.ForAccountById(l, span, s.Tenant())(s.AccountId(), issueSuccess(l, s, wp))
	}
}
