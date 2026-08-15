package environments

import (
	"context"

	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func getAll(ctx context.Context, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.WithContext(ctx), page)
	}
}

// byNameEntityProvider looks up an environment by its wire identity (Name),
// not by primary key: name is what the outbox key, the heartbeat, and the
// bootstrap Job all address it by.
func byNameEntityProvider(ctx context.Context) func(name string) database.EntityProvider[Entity] {
	return func(name string) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			var result Entity
			err := db.WithContext(ctx).Where("name = ?", name).First(&result).Error
			if err != nil {
				return model.ErrorProvider[Entity](err)
			}
			return model.FixedProvider[Entity](result)
		}
	}
}
