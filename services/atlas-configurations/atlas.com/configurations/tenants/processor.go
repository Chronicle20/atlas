package tenants

import (
	"context"
	"encoding/json"
	"atlas-configurations/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byIdProvider(ctx context.Context) func(db *gorm.DB) func(id uuid.UUID) model.Provider[RestModel] {
	return func(db *gorm.DB) func(id uuid.UUID) model.Provider[RestModel] {
		return func(id uuid.UUID) model.Provider[RestModel] {
			return model.Map(Make)(byIdEntityProvider(ctx)(id)(db))
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

func GetById(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(id uuid.UUID) (RestModel, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(id uuid.UUID) (RestModel, error) {
		return func(db *gorm.DB) func(id uuid.UUID) (RestModel, error) {
			return func(id uuid.UUID) (RestModel, error) {
				return byIdProvider(ctx)(db)(id)()
			}
		}
	}
}

func UpdateById(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(tenantId uuid.UUID, input RestModel) error {
	return func(ctx context.Context) func(db *gorm.DB) func(tenantId uuid.UUID, input RestModel) error {
		return func(db *gorm.DB) func(tenantId uuid.UUID, input RestModel) error {
			return func(tenantId uuid.UUID, input RestModel) error {
				res, err := json.Marshal(input)
				if err != nil {
					return err
				}
				rm := &json.RawMessage{}
				err = rm.UnmarshalJSON(res)
				if err != nil {
					return err
				}

				return database.ExecuteTransaction(db, update(ctx, tenantId, input.Region, input.MajorVersion, input.MinorVersion, *rm))
			}
		}
	}
}
