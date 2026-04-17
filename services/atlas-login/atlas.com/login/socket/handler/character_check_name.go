package handler

import (
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterCheckNameHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.CheckName{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		ok, err := character.NewProcessor(l, ctx).IsValidName(p.Name())
		if err != nil {
			l.Debugf("Error determining if name [%s] is valid.", p.Name())
			err = session.Announce(l)(ctx)(wp)(charcb.CharacterNameResponseWriter)(writer.CharacterNameResponseBody(p.Name(), writer.CharacterNameResponseCodeSystemError))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write character name response due to error.")
				return
			}
			return
		}

		if !ok {
			l.Debugf("Name [%s] is not allowed.", p.Name())
			err = session.Announce(l)(ctx)(wp)(charcb.CharacterNameResponseWriter)(writer.CharacterNameResponseBody(p.Name(), writer.CharacterNameResponseCodeNotAllowed))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write character name response due to error.")
				return
			}
			return
		}

		l.Debugf("Allowing character creation with the name of [%s].", p.Name())
		err = session.Announce(l)(ctx)(wp)(charcb.CharacterNameResponseWriter)(writer.CharacterNameResponseBody(p.Name(), writer.CharacterNameResponseCodeOk))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to write character name response due to error.")
			return
		}
	}
}
