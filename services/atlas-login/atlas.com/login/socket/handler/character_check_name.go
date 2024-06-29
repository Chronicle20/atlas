package handler

import (
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const CharacterCheckNameHandle = "CharacterCheckNameHandle"

func CharacterCheckNameHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	characterNameResponseFunc := session.Announce(wp)(writer.CharacterNameResponse)
	return func(s session.Model, r *request.Reader) {
		name := r.ReadAsciiString()
		ok, err := character.IsValidName(l, span, s.Tenant())(name)
		if err != nil {
			l.Debugf("Error determining if name [%s] is valid.", name)
			err = characterNameResponseFunc(s, writer.CharacterNameResponseBody(l)(name, writer.CharacterNameResponseCodeSystemError))
			if err != nil {
				l.WithError(err).Errorf("Unable to write character name response due to error.")
				return
			}
			return
		}

		if !ok {
			l.Debugf("Name [%s] is not allowed.", name)
			err = characterNameResponseFunc(s, writer.CharacterNameResponseBody(l)(name, writer.CharacterNameResponseCodeNotAllowed))
			if err != nil {
				l.WithError(err).Errorf("Unable to write character name response due to error.")
				return
			}
			return
		}

		l.Debugf("Allowing character creation with the name of [%s].", name)
		err = characterNameResponseFunc(s, writer.CharacterNameResponseBody(l)(name, writer.CharacterNameResponseCodeOk))
		if err != nil {
			l.WithError(err).Errorf("Unable to write character name response due to error.")
			return
		}
	}
}
