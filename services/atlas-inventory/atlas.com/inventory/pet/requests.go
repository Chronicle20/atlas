package pet

import (
	"atlas-inventory/rest"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const (
	Resource = "pets"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("PETS")
}

func requestById(petId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, petId))
}

func requestCreate(i Model) requests.Request[RestModel] {
	rm, err := model.Map(Transform)(model.FixedProvider(i))()
	if err != nil {
		return func(l logrus.FieldLogger, ctx context.Context) (RestModel, error) {
			return RestModel{}, err
		}
	}
	return rest.MakePostRequest[RestModel](getBaseRequest()+Resource, rm)
}
