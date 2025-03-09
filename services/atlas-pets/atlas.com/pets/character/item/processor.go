package item

import (
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetItemBySlot(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, inventoryId inventory.Type, slot int16) (Model, error) {
	return func(ctx context.Context) func(characterId uint32, inventoryId inventory.Type, slot int16) (Model, error) {
		return func(characterId uint32, inventoryId inventory.Type, slot int16) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestItemBySlot(characterId, byte(inventoryId), slot), Extract)()
		}
	}
}
