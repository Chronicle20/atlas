package session

import (
	"atlas-channel/rest"
	"atlas-channel/tenant"
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

func updateState(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(sessionId uuid.UUID, accountId uint32, state int) (Model, error) {
	return func(sessionId uuid.UUID, accountId uint32, state int) (Model, error) {
		i := InputRestModel{
			Id:        0,
			Issuer:    "CHANNEL",
			SessionId: sessionId,
			State:     state,
		}
		resp, err := rest.MakePatchRequest[OutputRestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+LoginsResource, accountId), i)(l)
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
