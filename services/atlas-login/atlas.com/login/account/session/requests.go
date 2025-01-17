package session

import (
	"atlas-login/rest"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	LoginsResource = "accounts/%d/sessions/"
)

func getBaseRequest() string {
	return os.Getenv("ACCOUNT_SERVICE_URL")
}

func updateState(l logrus.FieldLogger, ctx context.Context) func(sessionId uuid.UUID, accountId uint32, state int) (Model, error) {
	return func(sessionId uuid.UUID, accountId uint32, state int) (Model, error) {
		i := InputRestModel{
			Id:        0,
			SessionId: sessionId,
			Issuer:    "LOGIN",
			State:     state,
		}
		resp, err := rest.MakePatchRequest[OutputRestModel](fmt.Sprintf(getBaseRequest()+LoginsResource, accountId), i)(l, ctx)
		if err != nil {
			return Model{}, err
		}
		return Model{
			Code:   resp.Code,
			Reason: resp.Reason,
			Until:  resp.Until,
		}, nil
	}
}
