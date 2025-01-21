package guild

import (
	"atlas-guilds/character"
	"atlas-guilds/guild/member"
	"atlas-guilds/guild/title"
	"atlas-guilds/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byNameProvider(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, name string) model.Provider[Model] {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, name string) model.Provider[Model] {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, name string) model.Provider[Model] {
			return func(worldId byte, name string) model.Provider[Model] {
				ep := model.SliceMap[Entity, Model](Make)(getForName(t.Id(), worldId, name)(db))(model.ParallelMap())
				return model.FirstProvider(ep, model.Filters[Model]())
			}
		}
	}
}

func GetByName(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, name string) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, name string) (Model, error) {
		return func(db *gorm.DB) func(worldId byte, name string) (Model, error) {
			return func(worldId byte, name string) (Model, error) {
				return byNameProvider(l)(ctx)(db)(worldId, name)()
			}
		}
	}
}

func RequestCreate(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) error {
		return func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) error {
			return func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) error {
				_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventRequestAgreementProvider(worldId, characterId, name))
				return nil
			}
		}
	}

}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, leaderId uint32, name string) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, leaderId uint32, name string) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, leaderId uint32, name string) (Model, error) {
			return func(worldId byte, leaderId uint32, name string) (Model, error) {
				var err error
				var lc character.Model
				lc, err = character.GetById(l)(ctx)(leaderId)
				if err != nil {
					l.WithError(err).Errorf("Unable to locate character [%d] creating guild.", leaderId)
					return Model{}, err
				}

				var g Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					_, err = GetByName(l)(ctx)(tx)(worldId, name)
					if !errors.Is(err, gorm.ErrRecordNotFound) {
						l.WithError(err).Errorf("Attempting to create a guild [%s] by name which already exists.", name)
						return errors.New("already exists")
					}

					// TODO ensure leader is not already in a guild.

					g, err = create(tx, t, worldId, leaderId, name)
					if err != nil {
						l.WithError(err).Errorf("Unable to create guild [%s].", name)
						return err
					}

					_, err = member.AddMember(l)(ctx)(tx)(g.Id(), leaderId, lc.Name(), lc.JobId(), lc.Level(), 1)
					if err != nil {
						l.WithError(err).Errorf("Unable to add leader [%d] as guild [%d] member.", leaderId, g.Id())
						return err
					}

					_, err = title.CreateDefaults(l)(ctx)(tx)(g.Id())
					if err != nil {
						l.WithError(err).Errorf("Unable to create default ranks for guild [%d].", g.Id())
						return err
					}

					return nil
				})
				return g, txErr
			}
		}
	}
}
