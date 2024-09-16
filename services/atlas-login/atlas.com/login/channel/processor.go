package channel

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func ByWorldModelProvider(l logrus.FieldLogger, ctx context.Context) func(worldId byte) model.Provider[[]Model] {
	return func(worldId byte) model.Provider[[]Model] {
		return requests.SliceProvider[RestModel, Model](l, ctx)(requestChannelsForWorld(worldId), Extract)
	}
}
