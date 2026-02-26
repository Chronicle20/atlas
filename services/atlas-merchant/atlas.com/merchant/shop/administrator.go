package shop

import (
	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func create(entity *Entity) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		err := db.Create(entity).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(*entity)
	}
}

func update(entity *Entity) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		err := db.Save(entity).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(*entity)
	}
}
