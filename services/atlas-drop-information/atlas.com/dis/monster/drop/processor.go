package drop

import (
	"atlas-dis/database"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func GetAll(l logrus.FieldLogger, db *gorm.DB) []Model {
	ms, err := database.ModelSliceProvider[Model, entity](db)(getAll(), makeDrop)()
	if err != nil {
		l.WithError(err).Errorf("There was an error retrieving drops")
		return make([]Model, 0)
	}
	return ms
}

func GetForMonster(l logrus.FieldLogger, db *gorm.DB) func(monsterId uint32) []Model {
	return func(monsterId uint32) []Model {
		ms, err := database.ModelSliceProvider[Model, entity](db)(getByMonsterId(monsterId), makeDrop)()
		if err != nil {
			l.WithError(err).Errorf("There was an error retrieving drops for monster [%d]", monsterId)
			return make([]Model, 0)
		}
		return ms
	}
}
