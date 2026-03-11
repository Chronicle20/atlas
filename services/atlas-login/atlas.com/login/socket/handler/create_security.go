package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"
	"math/rand"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CreateSecurityHandle = "CreateSecurityHandle"

func CreateSecurityHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	loginAuthFunc := session.Announce(l)(ctx)(wp)(loginpkt.LoginAuthWriter)

	return func(s session.Model, _ *request.Reader, readerOptions map[string]interface{}) {
		loginScreen := [2]string{"MapLogin", "MapLogin1"}
		randomIndex := rand.Intn(len(loginScreen))

		err := loginAuthFunc(loginpkt.NewLoginAuth(loginScreen[randomIndex]).Encode)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce login screen.")
		}
	}
}
