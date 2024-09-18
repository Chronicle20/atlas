package character

import (
	"atlas-character-factory/data"
	"atlas-character-factory/inventory/item"
	"atlas-character-factory/job"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func byIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
		return func(characterId uint32) model.Provider[Model] {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(characterId), Extract)
		}
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return byIdProvider(l)(ctx)(characterId)()
		}
	}
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(accountId uint32, worldId byte, name string, gender byte, mapId uint32, jobIndex uint32, subJobIndex uint32, face uint32, hair uint32, hairColor uint32, skinColor byte) (Model, error) {
	return func(ctx context.Context) func(accountId uint32, worldId byte, name string, gender byte, mapId uint32, jobIndex uint32, subJobIndex uint32, face uint32, hair uint32, hairColor uint32, skinColor byte) (Model, error) {
		return func(accountId uint32, worldId byte, name string, gender byte, mapId uint32, jobIndex uint32, subJobIndex uint32, face uint32, hair uint32, hairColor uint32, skinColor byte) (Model, error) {
			jobId := job.Beginner
			if jobIndex == 0 {
				jobId = job.Noblesse
			} else if jobIndex == 1 {
				if subJobIndex == 0 {
					jobId = job.Beginner
				} else if subJobIndex == 1 {
					jobId = job.BladeRecruit
				}
			} else if jobIndex == 2 {
				jobId = job.Aran
			} else if jobIndex == 3 {
				jobId = job.Evan
			}

			rm, err := requestCreate(accountId, worldId, name, gender, mapId, jobId, face, hair, hairColor, skinColor)(l, ctx)
			if err != nil {
				return Model{}, err
			}
			return Extract(rm)
		}
	}
}

func CreateItem(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, itemId uint32) (item.Model, error) {
	return func(ctx context.Context) func(characterId uint32, itemId uint32) (item.Model, error) {
		return func(characterId uint32, itemId uint32) (item.Model, error) {
			rm, err := requestCreateItem(characterId, itemId)(l, ctx)
			if err != nil {
				return item.Model{}, err
			}
			return item.Extract(rm)
		}
	}
}

func EquipItem(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, itemId uint32, slot int16) error {
	return func(ctx context.Context) func(characterId uint32, itemId uint32, slot int16) error {
		return func(characterId uint32, itemId uint32, slot int16) error {
			slots, err := data.GetById(l)(ctx)(itemId)
			if err != nil || slots == nil || len(slots) == 0 {
				return err
			}
			// TODO special handling for rings, or legit multiple possible slots.
			destinationSlot := slots[0]

			e, err := requestEquipableItemBySlot(characterId, slot)(l, ctx)
			if err != nil {
				return err
			}

			_, err = requestEquipItem(characterId, destinationSlot.Name(), itemId, e.Slot)(l, ctx)
			return err
		}
	}
}
