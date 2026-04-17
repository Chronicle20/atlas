package handler

import (
	"atlas-login/account"
	as "atlas-login/account/session"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	account2 "github.com/Chronicle20/atlas/libs/atlas-packet/account/serverbound"
	loginpkt "github.com/Chronicle20/atlas/libs/atlas-packet/login/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func SetGenderHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := account2.SetGender{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		success := p.Set()
		if p.Set() {
			err := account.NewProcessor(l, ctx).UpdateGender(s.AccountId(), p.Gender())
			if err != nil {
				l.WithError(err).Errorf("Unable to update the gender of account [%d].", s.AccountId())
				success = false
			}
		}

		if !success {
			l.Debugf("Logging account out, as they are still at login screen and need to issue a new request.")
			as.NewProcessor(l, ctx).Destroy(s.SessionId(), s.AccountId())
		}

		err := session.Announce(l)(ctx)(wp)(loginpkt.SetAccountResultWriter)(loginpkt.NewSetAccountResult(p.Gender(), success).Encode)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to issue set account result")
		}
	}
}
