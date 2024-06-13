package login

import (
	"atlas-login/rest"
	"atlas-login/tenant"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	LoginsResource = "logins/"
)

func getBaseRequest() string {
	return os.Getenv("ACCOUNT_SERVICE_URL")
}

func CreateLogin(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(sessionId uuid.UUID, name string, password string, ipAddress string) (Model, error) {
	return func(sessionId uuid.UUID, name string, password string, ipAddress string) (Model, error) {
		i := InputRestModel{
			Id:        0,
			SessionId: sessionId,
			Name:      name,
			Password:  password,
			IpAddress: ipAddress,
			State:     0,
		}
		resp, err := rest.MakePostRequest[OutputRestModel](l, span, tenant)(getBaseRequest()+LoginsResource, i)(l)
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
