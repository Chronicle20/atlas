package services

import (
	"atlas-configurations/services/service"
	"context"
	"encoding/json"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ServiceType string

const (
	ServiceTypeLogin   = ServiceType("login-service")
	ServiceTypeChannel = ServiceType("channel-service")
	ServiceTypeDrops   = ServiceType("drops-service")
)

func byIdProvider(ctx context.Context) func(db *gorm.DB) func(id uuid.UUID) model.Provider[interface{}] {
	return func(db *gorm.DB) func(id uuid.UUID) model.Provider[interface{}] {
		return func(id uuid.UUID) model.Provider[interface{}] {
			return model.Map(Make)(byIdEntityProvider(ctx)(id)(db))
		}
	}
}

func allProvider(ctx context.Context) func(db *gorm.DB) func() model.Provider[[]interface{}] {
	return func(db *gorm.DB) func() model.Provider[[]interface{}] {
		return func() model.Provider[[]interface{}] {
			return model.SliceMap(Make)(allEntityProvider(ctx)(db))()
		}
	}
}

func Make(e Entity) (interface{}, error) {
	if e.Type == ServiceTypeLogin {
		var rm service.LoginRestModel
		err := json.Unmarshal(e.Data, &rm)
		if err != nil {
			return nil, err
		}
		rm.Id = e.Id.String()
		return rm, nil
	} else if e.Type == ServiceTypeChannel {
		var rm service.ChannelRestModel
		err := json.Unmarshal(e.Data, &rm)
		if err != nil {
			return nil, err
		}
		rm.Id = e.Id.String()
		return rm, nil
	} else if e.Type == ServiceTypeDrops {
		var rm service.GenericRestModel
		err := json.Unmarshal(e.Data, &rm)
		if err != nil {
			return nil, err
		}
		rm.Id = e.Id.String()
		return rm, nil
	}
	return nil, errors.New("invalid service type")
}

func GetAll(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func() ([]interface{}, error) {
	return func(ctx context.Context) func(db *gorm.DB) func() ([]interface{}, error) {
		return func(db *gorm.DB) func() ([]interface{}, error) {
			return allProvider(ctx)(db)()
		}
	}
}

func GetById(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(id uuid.UUID) (interface{}, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(id uuid.UUID) (interface{}, error) {
		return func(db *gorm.DB) func(id uuid.UUID) (interface{}, error) {
			return func(id uuid.UUID) (interface{}, error) {
				return byIdProvider(ctx)(db)(id)()
			}
		}
	}
}
