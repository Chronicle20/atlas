package templates

import (
	"context"
	"encoding/json"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byRegionAndVersionProvider(ctx context.Context) func(db *gorm.DB) func(region string, majorVersion uint16, minorVersion uint16) model.Provider[RestModel] {
	return func(db *gorm.DB) func(region string, majorVersion uint16, minorVersion uint16) model.Provider[RestModel] {
		return func(region string, majorVersion uint16, minorVersion uint16) model.Provider[RestModel] {
			return model.Map(Make)(byRegionVersionEntityProvider(ctx)(region, majorVersion, minorVersion)(db))
		}
	}
}

func allProvider(ctx context.Context) func(db *gorm.DB) func() model.Provider[[]RestModel] {
	return func(db *gorm.DB) func() model.Provider[[]RestModel] {
		return func() model.Provider[[]RestModel] {
			return model.SliceMap(Make)(allEntityProvider(ctx)(db))()
		}
	}
}

func Make(e Entity) (RestModel, error) {
	var rm RestModel
	err := json.Unmarshal(e.Data, &rm)
	if err != nil {
		return RestModel{}, err
	}
	rm.Id = e.Id.String()
	return rm, nil
}

func GetAll(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func() ([]RestModel, error) {
	return func(ctx context.Context) func(db *gorm.DB) func() ([]RestModel, error) {
		return func(db *gorm.DB) func() ([]RestModel, error) {
			return allProvider(ctx)(db)()
		}
	}
}

func GetByRegionAndVersion(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(region string, majorVersion uint16, minorVersion uint16) (RestModel, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(region string, majorVersion uint16, minorVersion uint16) (RestModel, error) {
		return func(db *gorm.DB) func(region string, majorVersion uint16, minorVersion uint16) (RestModel, error) {
			return func(region string, majorVersion uint16, minorVersion uint16) (RestModel, error) {
				return byRegionAndVersionProvider(ctx)(db)(region, majorVersion, minorVersion)()
			}
		}
	}
}
