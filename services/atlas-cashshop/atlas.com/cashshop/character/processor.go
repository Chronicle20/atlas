package character

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetByIdWithInventory(l logrus.FieldLogger) func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
		return func(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				p := requests.Provider[RestModel, Model](l, ctx)(requestByIdWithInventory(characterId), Extract)
				return model.Map(model.Decorate(decorators))(p)()
			}
		}
	}
}
