package templates

import (
	"atlas-configurations/database"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func allEntityProvider(ctx context.Context) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.WithContext(ctx).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}

func byRegionVersionEntityProvider(ctx context.Context) func(region string, majorVersion uint16, minorVersion uint16) database.EntityProvider[Entity] {
	return func(region string, majorVersion uint16, minorVersion uint16) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			var result Entity
			err := db.WithContext(ctx).Where("region = ? AND major_version = ? AND minor_version = ?", region, majorVersion, minorVersion).First(&result).Error
			if err != nil {
				return model.ErrorProvider[Entity](err)
			}
			return model.FixedProvider[Entity](result)
		}
	}
}
