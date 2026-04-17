package pet

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const (
	Resource = "pets"
)

func getBaseRequest() string {
	return requests.RootUrl("PETS")
}

func requestCreate(i Model) requests.Request[RestModel] {
	rm, err := model.Map(Transform)(model.FixedProvider(i))()
	if err != nil {
		return func(l logrus.FieldLogger, ctx context.Context) (RestModel, error) {
			return RestModel{}, err
		}
	}
	return requests.PostRequest[RestModel](getBaseRequest()+Resource, rm)
}
