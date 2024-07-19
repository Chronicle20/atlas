package handler

import (
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
		confirmed := r.ReadByte()
		gender := r.ReadByte()
		l.Debugf("Reading [%s] message. body={confirmed=%d, gender=%d}", SetGenderHandle, confirmed, gender)
		err := setAccountResultFunc(s, writer.SetAccountResultBody(l)(gender))
		if err != nil {
			l.WithError(err).Errorf("Unable to issue set account result")
		}
	}
}
