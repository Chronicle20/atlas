package handler

import (
	"atlas-login/account"
	as "atlas-login/account/session"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const SetGenderHandle = "SetGenderHandle"

func SetGenderHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	setAccountResultFunc := session.Announce(l)(wp)(writer.SetAccountResult)
	return func(s session.Model, r *request.Reader) {
		confirmed := r.ReadBool()
		gender := r.ReadByte()
		l.Debugf("Reading [%s] message. body={confirmed=%t, gender=%d}", SetGenderHandle, confirmed, gender)

		var success = confirmed
		if confirmed {
			err := account.NewProcessor(l, ctx).UpdateGender(s.AccountId(), gender)
			if err != nil {
				l.WithError(err).Errorf("Unable to update the gender of account [%d].", s.AccountId())
				success = false
			}
		}

		if !success {
			l.Debugf("Logging account out, as they are still at login screen and need to issue a new request.")
			as.NewProcessor(l, ctx).Destroy(s.SessionId(), s.AccountId())
		}

		err := setAccountResultFunc(s, writer.SetAccountResultBody(gender, success))
		if err != nil {
			l.WithError(err).Errorf("Unable to issue set account result")
		}
	}
}
