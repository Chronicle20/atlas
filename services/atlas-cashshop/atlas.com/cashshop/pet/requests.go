package pet

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "pets"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "PETS")
}

func requestCreate(ctx context.Context, i Model) requests.Request[RestModel] {
	rm, err := model.Map(Transform)(model.FixedProvider(i))()
	if err != nil {
		return func(l logrus.FieldLogger, ctx context.Context) (RestModel, error) {
			return RestModel{}, err
		}
	}
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.PostRequest[RestModel](root+Resource, rm)
}
