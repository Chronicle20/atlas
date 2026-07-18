package services

import (
	"context"

	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func getAll(ctx context.Context, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.WithContext(ctx), page)
	}
}

func byIdEntityProvider(ctx context.Context) func(id uuid.UUID) database.EntityProvider[Entity] {
	return func(id uuid.UUID) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			var result Entity
			err := db.WithContext(ctx).Where("id = ?", id).First(&result).Error
			if err != nil {
				return model.ErrorProvider[Entity](err)
			}
			return model.FixedProvider[Entity](result)
		}
	}
}
