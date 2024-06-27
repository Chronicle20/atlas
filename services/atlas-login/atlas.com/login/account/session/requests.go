package session

import (
	"atlas-login/rest"
	"atlas-login/tenant"
	"fmt"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	LoginsResource = "accounts/%d/sessions/"
)

func getBaseRequest() string {
	return os.Getenv("ACCOUNT_SERVICE_URL")
}

func CreateLogin(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(sessionId uuid.UUID, accountId uint32, name string, password string, ipAddress string) (Model, error) {
	return func(sessionId uuid.UUID, accountId uint32, name string, password string, ipAddress string) (Model, error) {
		i := InputRestModel{
			Id:        0,
			SessionId: sessionId,
			Name:      name,
			Password:  password,
			IpAddress: ipAddress,
			State:     0,
		}
		resp, err := rest.MakePostRequest[OutputRestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+LoginsResource, accountId), i)(l)
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
