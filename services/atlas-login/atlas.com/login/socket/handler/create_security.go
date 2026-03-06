package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"
	"math/rand"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CreateSecurityHandle = "CreateSecurityHandle"

func CreateSecurityHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	loginAuthFunc := session.Announce(l)(ctx)(wp)(writer.LoginAuth)

	return func(s session.Model, _ *request.Reader) {
		loginScreen := [2]string{"MapLogin", "MapLogin1"}
		randomIndex := rand.Intn(len(loginScreen))

		err := loginAuthFunc(writer.LoginAuthBody(loginScreen[randomIndex]))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce login screen.")
		}
	}
}
