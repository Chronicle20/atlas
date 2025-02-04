package configuration

import (
	"atlas-configurations/configuration/service/channel"
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

func channelServiceConfigurationProvider(ctx context.Context) func(db *gorm.DB) func(serviceId uuid.UUID) model.Provider[channel.RestModel] {
	return func(db *gorm.DB) func(serviceId uuid.UUID) model.Provider[channel.RestModel] {
		return func(serviceId uuid.UUID) model.Provider[channel.RestModel] {
			return configurationProvider[channel.RestModel](ctx)(db)(serviceId, TypeChannelService)(MakeChannelServiceModel)
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

func GetChannelServiceConfiguration(ctx context.Context) func(db *gorm.DB) func(serviceId uuid.UUID) (channel.RestModel, error) {
	return func(db *gorm.DB) func(serviceId uuid.UUID) (channel.RestModel, error) {
		return func(serviceId uuid.UUID) (channel.RestModel, error) {
			return channelServiceConfigurationProvider(ctx)(db)(serviceId)()
		}
	}
}
