package pet

import (
	"atlas-pets/character/item"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func Create(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, itemId uint32, slot int16) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, itemId uint32, slot int16) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, itemId uint32, slot int16) error {
			return func(characterId uint32, itemId uint32, slot int16) error {
				it, ok := inventory.TypeFromItemId(itemId)
				if !ok {
					return errors.New("invalid item id")
				}
				i, err := item.GetItemBySlot(l)(ctx)(characterId, it, slot)
				if err != nil {
					return err
				}

				var om Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					// TODO lookup name
					im := NewModelBuilder(0, i.Id(), itemId, "Great Pet")
					om, err = create(db)(t, characterId, im.Build())
					if err != nil {
						return err
					}
					return nil
				})
				if txErr != nil {
					return txErr
				}
				l.Debugf("Created pet [%d] for character [%d].", om.Id(), characterId)
				return nil
			}
		}
	}
}
