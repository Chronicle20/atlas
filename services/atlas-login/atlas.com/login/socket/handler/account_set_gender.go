package handler

import (
	"atlas-login/account"
	as "atlas-login/account/session"
	"atlas-login/kafka/producer"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const SetGenderHandle = "SetGenderHandle"

func SetGenderHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	setAccountResultFunc := session.Announce(l)(wp)(writer.SetAccountResult)
	return func(s session.Model, r *request.Reader) {
		confirmed := r.ReadBool()
		gender := r.ReadByte()
		l.Debugf("Reading [%s] message. body={confirmed=%t, gender=%d}", SetGenderHandle, confirmed, gender)

		var success = confirmed
		if confirmed {
			err := account.UpdateGender(l, span, s.Tenant())(s.AccountId(), gender)
			if err != nil {
				l.WithError(err).Errorf("Unable to update the gender of account [%d].", s.AccountId())
				success = false
			}
		}

		if !success {
			l.Debugf("Logging account out, as they are still at login screen and need to issue a new request.")
			as.Destroy(l, producer.ProviderImpl(l)(span))(s.Tenant(), s.SessionId(), s.AccountId())
		}

		err := setAccountResultFunc(s, writer.SetAccountResultBody(l)(gender, success))
		if err != nil {
			l.WithError(err).Errorf("Unable to issue set account result")
		}
	}
}
