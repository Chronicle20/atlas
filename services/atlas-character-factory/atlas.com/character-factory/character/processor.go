package character

import (
	"atlas-character-factory/data"
	"atlas-character-factory/inventory/item"
	"atlas-character-factory/job"
	"atlas-character-factory/tenant"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func byIdProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(characterId uint32) model.Provider[Model] {
	return func(characterId uint32) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestById(ctx, tenant)(characterId), Extract)
	}
}

func GetById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return byIdProvider(l, ctx, tenant)(characterId)()
	}
}

func Create(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(accountId uint32, worldId byte, name string, gender byte, mapId uint32, jobIndex uint32, subJobIndex uint32, face uint32, hair uint32, hairColor uint32, skinColor byte) (Model, error) {
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

		rm, err := requestCreate(ctx, tenant)(accountId, worldId, name, gender, mapId, jobId, face, hair, hairColor, skinColor)(l)
		if err != nil {
			return Model{}, err
		}
		return Extract(rm)
	}
}

func CreateItem(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(characterId uint32, itemId uint32) (item.Model, error) {
	return func(characterId uint32, itemId uint32) (item.Model, error) {
		rm, err := requestCreateItem(ctx, tenant)(characterId, itemId)(l)
		if err != nil {
			return item.Model{}, err
		}
		return item.Extract(rm)
	}
}

func EquipItem(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(characterId uint32, itemId uint32, slot int16) error {
	return func(characterId uint32, itemId uint32, slot int16) error {
		slots, err := data.GetById(l, ctx, tenant)(itemId)
		if err != nil || slots == nil || len(slots) == 0 {
			return err
		}
		// TODO special handling for rings, or legit multiple possible slots.
		destinationSlot := slots[0]

		e, err := requestEquipableItemBySlot(ctx, tenant)(characterId, slot)(l)
		if err != nil {
			return err
		}

		_, err = requestEquipItem(ctx, tenant)(characterId, destinationSlot.Name(), itemId, e.Slot)(l)
		return err
	}
}
