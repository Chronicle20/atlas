package database

import (
	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

// EntityProvider returns a provider function for database entities
func EntityProvider[E any](db *gorm.DB, queryFunc func(db *gorm.DB) *gorm.DB) model.Provider[E] {
	return func() (E, error) {
		var entity E
		result := queryFunc(db).First(&entity)
		return entity, result.Error
	}
}

// SliceProvider returns a provider function for slices of database entities
func SliceProvider[E any](db *gorm.DB, queryFunc func(db *gorm.DB) *gorm.DB) model.Provider[[]E] {
	return func() ([]E, error) {
		var entities []E
		result := queryFunc(db).Find(&entities)
		return entities, result.Error
	}
}
