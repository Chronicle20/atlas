package database

import (
	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

type EntityProvider[E any] func(db *gorm.DB) model.Provider[E]

func ModelProvider[M any, E any](db *gorm.DB) func(ep EntityProvider[E], t model.Transformer[E, M]) model.Provider[M] {
	return func(ep EntityProvider[E], t model.Transformer[E, M]) model.Provider[M] {
		return model.Map[E, M](ep(db), t)
	}
}

func FoldModelProvider[M any, N any](db *gorm.DB) func(ep EntityProvider[[]N], supplier model.Provider[M], folder model.Folder[N, M]) model.Provider[M] {
	return func(ep EntityProvider[[]N], supplier model.Provider[M], folder model.Folder[N, M]) model.Provider[M] {
		return model.Fold[N, M](ep(db), supplier, folder)
	}
}

func ModelSliceProvider[M any, E any](db *gorm.DB) func(ep EntityProvider[[]E], t model.Transformer[E, M]) model.Provider[[]M] {
	return func(ep EntityProvider[[]E], t model.Transformer[E, M]) model.Provider[[]M] {
		return model.SliceMap(ep(db), t)
	}
}

func Query[E any](db *gorm.DB, query interface{}) model.Provider[E] {
	var result E
	err := db.Where(query).First(&result).Error
	if err != nil {
		return model.ErrorProvider[E](err)
	}
	return model.FixedProvider[E](result)
}

func SliceQuery[E any](db *gorm.DB, query interface{}) model.Provider[[]E] {
	var results []E
	err := db.Where(query).Find(&results).Error
	if err != nil {
		return model.ErrorProvider[[]E](err)
	}
	return model.FixedProvider(results)
}
