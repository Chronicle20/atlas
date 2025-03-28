package skill

import (
	skill2 "atlas-skills/kafka/message/skill"
	"atlas-skills/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"time"
)

func byCharacterIdProvider(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			mp := model.SliceMap(Make)(getByCharacterId(t.Id(), characterId)(db))()
			return model.SliceMap(model.Decorate(model.Decorators(CooldownDecorator(ctx)(characterId))))(mp)()
		}
	}
}

func GetByCharacterId(ctx context.Context) func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
	return func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
		return func(characterId uint32) ([]Model, error) {
			return byCharacterIdProvider(ctx)(db)(characterId)()
		}
	}
}

func byIdProvider(ctx context.Context) func(db *gorm.DB) func(characterId uint32, id uint32) model.Provider[Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(characterId uint32, id uint32) model.Provider[Model] {
		return func(characterId uint32, id uint32) model.Provider[Model] {
			mp := model.Map(Make)(getById(t.Id(), characterId, id)(db))
			return model.Map(model.Decorate(model.Decorators(CooldownDecorator(ctx)(characterId))))(mp)
		}
	}
}

func CooldownDecorator(ctx context.Context) func(characterId uint32) model.Decorator[Model] {
	t := tenant.MustFromContext(ctx)
	return func(characterId uint32) model.Decorator[Model] {
		return func(m Model) Model {
			ct, err := GetRegistry().Get(t, characterId, m.Id())
			if err != nil {
				return m
			}
			return m.SetCooldown(ct)
		}
	}
}

func GetById(ctx context.Context) func(db *gorm.DB) func(characterId uint32, id uint32) (Model, error) {
	return func(db *gorm.DB) func(characterId uint32, id uint32) (Model, error) {
		return func(characterId uint32, id uint32) (Model, error) {
			return byIdProvider(ctx)(db)(characterId, id)()
		}
	}
}

func RequestCreate(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
	return func(ctx context.Context) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
		return func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
			return producer.ProviderImpl(l)(ctx)(skill2.EnvCommandTopic)(createCommandProvider(characterId, id, level, masterLevel, expiration))
		}
	}
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
			return func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
				l.Debugf("Attempting to create skill [%d] for character [%d].", id, characterId)
				var s Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					var err error
					s, err = GetById(ctx)(tx)(characterId, id)
					if s.Id() != 0 {
						return errors.New("already exists")
					}
					s, err = create(tx, t.Id(), characterId, id, level, masterLevel, expiration)
					if err != nil {
						return err
					}
					return nil
				})
				if txErr != nil {
					return Model{}, txErr
				}
				l.Debugf("Created skill [%d] for character [%d].", id, characterId)
				_ = producer.ProviderImpl(l)(ctx)(skill2.EnvStatusEventTopic)(statusEventCreatedProvider(characterId, s.Id(), s.Level(), s.MasterLevel(), s.Expiration()))
				return s, nil
			}
		}
	}
}

func RequestUpdate(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
	return func(ctx context.Context) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
		return func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
			return producer.ProviderImpl(l)(ctx)(skill2.EnvCommandTopic)(updateCommandProvider(characterId, id, level, masterLevel, expiration))
		}
	}
}

func Update(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
			return func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
				l.Debugf("Attempting to update skill [%d] for character [%d].", id, characterId)
				var s Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					var err error
					s, err = GetById(ctx)(tx)(characterId, id)
					if err != nil {
						return errors.New("does not exist")
					}
					err = dynamicUpdate(tx)(SetLevel(level), SetMasterLevel(masterLevel), SetExpiration(expiration))(t.Id(), characterId)(s)
					if err != nil {
						return err
					}
					s, err = GetById(ctx)(tx)(characterId, id)
					if err != nil {
						return errors.New("does not exist")
					}
					return nil
				})
				if txErr != nil {
					return Model{}, txErr
				}
				l.Debugf("Update skill [%d] for character [%d].", id, characterId)
				_ = producer.ProviderImpl(l)(ctx)(skill2.EnvStatusEventTopic)(statusEventUpdatedProvider(characterId, s.Id(), s.Level(), s.MasterLevel(), s.Expiration()))
				return s, nil
			}
		}
	}
}

func SetCooldown(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, skillId uint32, cooldown uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, skillId uint32, cooldown uint32) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, skillId uint32, cooldown uint32) (Model, error) {
			return func(characterId uint32, skillId uint32, cooldown uint32) (Model, error) {
				l.Debugf("Applying cooldown of [%d] for character [%d] skill [%d].", cooldown, characterId, skillId)
				err := GetRegistry().Apply(t, characterId, skillId, cooldown)
				if err != nil {
					return Model{}, err
				}
				s, err := GetById(ctx)(db)(characterId, skillId)
				if err != nil {
					return Model{}, err
				}
				_ = producer.ProviderImpl(l)(ctx)(skill2.EnvStatusEventTopic)(statusEventCooldownAppliedProvider(characterId, s.Id(), s.CooldownExpiresAt()))
				return s, nil
			}
		}
	}
}

func ExpireCooldowns(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) {
	return func(ctx context.Context) func(db *gorm.DB) {
		return func(db *gorm.DB) {
			for _, s := range GetRegistry().GetAll() {
				if s.CooldownExpiresAt().Before(time.Now()) {
					_ = GetRegistry().Clear(s.Tenant(), s.CharacterId(), s.SkillId())
					_ = producer.ProviderImpl(l)(tenant.WithContext(ctx, s.Tenant()))(skill2.EnvStatusEventTopic)(statusEventCooldownExpiredProvider(s.CharacterId(), s.SkillId()))
				}
			}
		}
	}
}

func ClearAll(ctx context.Context) func(characterId uint32) error {
	t := tenant.MustFromContext(ctx)
	return func(characterId uint32) error {
		return GetRegistry().ClearAll(t, characterId)
	}
}
