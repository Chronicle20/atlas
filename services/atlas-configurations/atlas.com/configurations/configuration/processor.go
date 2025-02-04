package configuration

import (
	"atlas-configurations/configuration/service/channel"
	"atlas-configurations/configuration/service/characterfactory"
	"atlas-configurations/configuration/service/login"
	"atlas-configurations/database"
	"context"
	"encoding/json"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func byServiceIdAndTypeEntityProvider(ctx context.Context) func(serviceId uuid.UUID, serviceType string) database.EntityProvider[Entity] {
	return func(serviceId uuid.UUID, serviceType string) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			var result Entity
			err := db.WithContext(ctx).Where("service_id = ? AND type = ?", serviceId, serviceType).First(&result).Error
			if err != nil {
				return model.ErrorProvider[Entity](err)
			}
			return model.FixedProvider[Entity](result)
		}
	}
}

func configurationProvider[M any](ctx context.Context) func(db *gorm.DB) func(serviceId uuid.UUID, serviceType string) func(t model.Transformer[Entity, M]) model.Provider[M] {
	return func(db *gorm.DB) func(serviceId uuid.UUID, serviceType string) func(t model.Transformer[Entity, M]) model.Provider[M] {
		return func(serviceId uuid.UUID, serviceType string) func(t model.Transformer[Entity, M]) model.Provider[M] {
			return func(t model.Transformer[Entity, M]) model.Provider[M] {
				return model.Map(t)(byServiceIdAndTypeEntityProvider(ctx)(serviceId, serviceType)(db))
			}
		}
	}
}

func GetChannelServiceConfiguration(ctx context.Context) func(db *gorm.DB) func(serviceId uuid.UUID) (channel.RestModel, error) {
	return func(db *gorm.DB) func(serviceId uuid.UUID) (channel.RestModel, error) {
		return func(serviceId uuid.UUID) (channel.RestModel, error) {
			return configurationProvider[channel.RestModel](ctx)(db)(serviceId, TypeChannelService)(MakeChannelServiceModel)()
		}
	}
}

func MakeChannelServiceModel(e Entity) (channel.RestModel, error) {
	var rm channel.RestModel
	err := json.Unmarshal(e.Data, &rm)
	if err != nil {
		return channel.RestModel{}, err
	}
	rm.Id = e.ServiceId
	return rm, nil
}

func GetCharacterFactoryConfiguration(ctx context.Context) func(db *gorm.DB) func(serviceId uuid.UUID) (characterfactory.RestModel, error) {
	return func(db *gorm.DB) func(serviceId uuid.UUID) (characterfactory.RestModel, error) {
		return func(serviceId uuid.UUID) (characterfactory.RestModel, error) {
			return configurationProvider[characterfactory.RestModel](ctx)(db)(serviceId, TypeCharacterFactory)(MakeCharacterFactoryModel)()
		}
	}
}

func MakeCharacterFactoryModel(e Entity) (characterfactory.RestModel, error) {
	var rm characterfactory.RestModel
	err := json.Unmarshal(e.Data, &rm)
	if err != nil {
		return characterfactory.RestModel{}, err
	}
	rm.Id = e.ServiceId
	return rm, nil
}

func GetLoginServiceConfiguration(ctx context.Context) func(db *gorm.DB) func(serviceId uuid.UUID) (login.RestModel, error) {
	return func(db *gorm.DB) func(serviceId uuid.UUID) (login.RestModel, error) {
		return func(serviceId uuid.UUID) (login.RestModel, error) {
			return configurationProvider[login.RestModel](ctx)(db)(serviceId, TypeLoginService)(MakeLoginServiceModel)()
		}
	}
}

func MakeLoginServiceModel(e Entity) (login.RestModel, error) {
	var rm login.RestModel
	err := json.Unmarshal(e.Data, &rm)
	if err != nil {
		return login.RestModel{}, err
	}
	rm.Id = e.ServiceId
	return rm, nil
}
