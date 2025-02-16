package character

import (
	"atlas-character/drop"
	"atlas-character/equipable"
	"atlas-character/equipment"
	"atlas-character/equipment/slot"
	"atlas-character/inventory"
	"atlas-character/kafka/producer"
	"atlas-character/portal"
	skill2 "atlas-character/skill"
	skill3 "atlas-character/skill/data"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
	"math/rand"
	"regexp"
	"time"
)

var blockedNameErr = errors.New("blocked name")
var invalidLevelErr = errors.New("invalid level")

const (
	CommandDistributeApAbilityStrength     = "STRENGTH"
	CommandDistributeApAbilityDexterity    = "DEXTERITY"
	CommandDistributeApAbilityIntelligence = "INTELLIGENCE"
	CommandDistributeApAbilityLuck         = "LUCK"
	CommandDistributeApAbilityHp           = "HP"
	CommandDistributeApAbilityMp           = "MP"
)

// entityModelMapper A function which maps an entity provider to a Model provider
type entityModelMapper = func(provider model.Provider[entity]) model.Provider[Model]

// entitySliceModelMapper A function which maps an entity slice provider to a Model slice provider
type entitySliceModelMapper = func(provider model.Provider[[]entity]) model.Provider[[]Model]

var entityModelMapperFunc = model.Map[entity, Model](modelFromEntity)

var entitySliceModelMapperFunc = model.SliceMap[entity, Model](modelFromEntity)

type IdProvider = func(uint32) model.Provider[Model]

type IdRetriever = func(uint32) (Model, error)

// ByIdProvider Retrieves a singular character by id.
var ByIdProvider = model.Flip(model.Compose(model.Curry(model.Compose[context.Context, tenant.Model, IdProvider]), byIdProvider))(tenant.MustFromContext)

func byIdProvider(db *gorm.DB) func(t tenant.Model) func(characterId uint32) model.Provider[Model] {
	return func(t tenant.Model) func(characterId uint32) model.Provider[Model] {
		return func(characterId uint32) model.Provider[Model] {
			return entityModelMapperFunc(getById(t.Id(), characterId)(db))
		}
	}
}

// GetById Retrieves a singular character by id.
func GetById(ctx context.Context) func(db *gorm.DB) func(decorators ...model.Decorator[Model]) IdRetriever {
	return func(db *gorm.DB) func(decorators ...model.Decorator[Model]) IdRetriever {
		return func(decorators ...model.Decorator[Model]) IdRetriever {
			return func(id uint32) (Model, error) {
				return model.Map(model.Decorate[Model](decorators))(ByIdProvider(db)(ctx)(id))()
			}
		}
	}
}

type AccountsInWorldProvider = func(accountId uint32) func(worldId byte) model.Provider[[]Model]

type AccountsInWorldRetriever = func(accountId uint32) func(worldId byte) ([]Model, error)

func byAccountInWorldProvider(db *gorm.DB) func(tenant tenant.Model) AccountsInWorldProvider {
	return func(tenant tenant.Model) AccountsInWorldProvider {
		return func(accountId uint32) func(worldId byte) model.Provider[[]Model] {
			return func(worldId byte) model.Provider[[]Model] {
				return entitySliceModelMapperFunc(getForAccountInWorld(tenant.Id(), accountId, worldId)(db))(model.ParallelMap())
			}
		}
	}
}

func GetForAccountInWorld(db *gorm.DB) func(ctx context.Context) func(accountId uint32, worldId byte, decorators ...model.Decorator[Model]) ([]Model, error) {
	return func(ctx context.Context) func(accountId uint32, worldId byte, decorators ...model.Decorator[Model]) ([]Model, error) {
		return func(accountId uint32, worldId byte, decorators ...model.Decorator[Model]) ([]Model, error) {
			t := tenant.MustFromContext(ctx)
			return model.SliceMap(model.Decorate(decorators))(byAccountInWorldProvider(db)(t)(accountId)(worldId))(model.ParallelMap())()
		}
	}
}

func byMapInWorld(db *gorm.DB, tenant tenant.Model) func(worldId byte, mapId uint32) model.Provider[[]Model] {
	return func(worldId byte, mapId uint32) model.Provider[[]Model] {
		return entitySliceModelMapperFunc(getForMapInWorld(tenant.Id(), worldId, mapId)(db))(model.ParallelMap())
	}
}

func GetForMapInWorld(db *gorm.DB) func(ctx context.Context) func(worldId byte, mapId uint32, decorators ...model.Decorator[Model]) ([]Model, error) {
	return func(ctx context.Context) func(worldId byte, mapId uint32, decorators ...model.Decorator[Model]) ([]Model, error) {
		return func(worldId byte, mapId uint32, decorators ...model.Decorator[Model]) ([]Model, error) {
			t := tenant.MustFromContext(ctx)
			return model.SliceMap(model.Decorate[Model](decorators))(byMapInWorld(db, t)(worldId, mapId))(model.ParallelMap())()
		}
	}
}

type NameProvider = func(string) model.Provider[[]Model]

type NameRetriever = func(string) ([]Model, error)

// ByNameProvider Retrieves a singular account by name.
var ByNameProvider = model.Flip(model.Compose(model.Curry(model.Compose[context.Context, tenant.Model, NameProvider]), byNameProvider))(tenant.MustFromContext)

func byNameProvider(db *gorm.DB) func(t tenant.Model) func(string) model.Provider[[]Model] {
	return func(t tenant.Model) func(string) model.Provider[[]Model] {
		return func(name string) model.Provider[[]Model] {
			return model.SliceMap[entity, Model](modelFromEntity)(getForName(t.Id(), name)(db))(model.ParallelMap())
		}
	}
}

func GetForName(db *gorm.DB) func(ctx context.Context) func(name string, decorators ...model.Decorator[Model]) ([]Model, error) {
	return func(ctx context.Context) func(name string, decorators ...model.Decorator[Model]) ([]Model, error) {
		return func(name string, decorators ...model.Decorator[Model]) ([]Model, error) {
			return model.SliceMap(model.Decorate[Model](decorators))(ByNameProvider(db)(ctx)(name))(model.ParallelMap())()
		}
	}
}

func InventoryModelDecorator(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) model.Decorator[Model] {
	return func(db *gorm.DB) func(ctx context.Context) model.Decorator[Model] {
		return func(ctx context.Context) model.Decorator[Model] {
			return func(m Model) Model {
				i, err := inventory.GetInventories(l)(db)(ctx)(m.Id())
				if err != nil {
					return m
				}

				es, err := model.Fold(equipable.EquipmentProvider(l)(db)(ctx)(i.Equipable().Id()), model.FixedProvider(m.GetEquipment()), FoldEquipable)()
				if err != nil {
					return CloneModel(m).SetInventory(i).Build()
				}
				return CloneModel(m).SetEquipment(es).SetInventory(i).Build()
			}
		}
	}
}

func SkillModelDecorator(l logrus.FieldLogger) func(ctx context.Context) model.Decorator[Model] {
	return func(ctx context.Context) model.Decorator[Model] {
		return func(m Model) Model {
			ms, err := skill2.GetByCharacterId(l)(ctx)(m.Id())
			if err != nil {
				return m
			}
			return CloneModel(m).SetSkills(ms).Build()
		}
	}
}

func FoldEquipable(m equipment.Model, e equipable.Model) (equipment.Model, error) {
	var setter equipment.SlotSetter
	if e.Slot() > -100 {
		switch slot.Position(e.Slot()) {
		case slot.PositionHat:
			setter = m.SetHat
		case slot.PositionMedal:
			setter = m.SetMedal
		case slot.PositionForehead:
			setter = m.SetForehead
		case slot.PositionRing1:
			setter = m.SetRing1
		case slot.PositionRing2:
			setter = m.SetRing2
		case slot.PositionEye:
			setter = m.SetEye
		case slot.PositionEarring:
			setter = m.SetEarring
		case slot.PositionShoulder:
			setter = m.SetShoulder
		case slot.PositionCape:
			setter = m.SetCape
		case slot.PositionTop:
			setter = m.SetTop
		case slot.PositionPendant:
			setter = m.SetPendant
		case slot.PositionWeapon:
			setter = m.SetWeapon
		case slot.PositionShield:
			setter = m.SetShield
		case slot.PositionGloves:
			setter = m.SetGloves
		case slot.PositionBottom:
			setter = m.SetBottom
		case slot.PositionBelt:
			setter = m.SetBelt
		case slot.PositionRing3:
			setter = m.SetRing3
		case slot.PositionRing4:
			setter = m.SetRing4
		case slot.PositionShoes:
			setter = m.SetShoes
		}
	} else {
		switch slot.Position(e.Slot() + 100) {
		case slot.PositionHat:
			setter = m.SetCashHat
		case slot.PositionMedal:
			setter = m.SetCashMedal
		case slot.PositionForehead:
			setter = m.SetCashForehead
		case slot.PositionRing1:
			setter = m.SetCashRing1
		case slot.PositionRing2:
			setter = m.SetCashRing2
		case slot.PositionEye:
			setter = m.SetCashEye
		case slot.PositionEarring:
			setter = m.SetCashEarring
		case slot.PositionShoulder:
			setter = m.SetCashShoulder
		case slot.PositionCape:
			setter = m.SetCashCape
		case slot.PositionTop:
			setter = m.SetCashTop
		case slot.PositionPendant:
			setter = m.SetCashPendant
		case slot.PositionWeapon:
			setter = m.SetCashWeapon
		case slot.PositionShield:
			setter = m.SetCashShield
		case slot.PositionGloves:
			setter = m.SetCashGloves
		case slot.PositionBottom:
			setter = m.SetCashBottom
		case slot.PositionBelt:
			setter = m.SetCashBelt
		case slot.PositionRing3:
			setter = m.SetCashRing3
		case slot.PositionRing4:
			setter = m.SetCashRing4
		case slot.PositionShoes:
			setter = m.SetCashShoes
		}
	}
	return setter(&e), nil
}

func IsValidName(_ logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(name string) (bool, error) {
	return func(db *gorm.DB) func(ctx context.Context) func(name string) (bool, error) {
		return func(ctx context.Context) func(name string) (bool, error) {
			return func(name string) (bool, error) {
				m, err := regexp.MatchString("[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}", name)
				if err != nil {
					return false, err
				}
				if !m {
					return false, nil
				}

				cs, err := GetForName(db)(ctx)(name)
				if len(cs) != 0 || err != nil {
					return false, nil
				}

				//TODO
				//bn, err := blocked_name.IsBlockedName(l, ctx)(name)
				//if bn {
				//	return false, err
				//}

				return true, nil

			}
		}
	}
}

func Create(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(input Model) (Model, error) {
	return func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(input Model) (Model, error) {
		return func(ctx context.Context) func(eventProducer producer.Provider) func(input Model) (Model, error) {
			return func(eventProducer producer.Provider) func(input Model) (Model, error) {
				return func(input Model) (Model, error) {

					ok, err := IsValidName(l)(db)(ctx)(input.Name())
					if err != nil {
						l.WithError(err).Errorf("Error validating name [%s] during character creation.", input.Name())
						return Model{}, err
					}
					if !ok {
						l.Infof("Attempting to create a character with an invalid name [%s].", input.Name())
						return Model{}, blockedNameErr
					}
					if input.Level() < 1 || input.Level() > 200 {
						l.Infof("Attempting to create character with an invalid level [%d].", input.Level())
						return Model{}, invalidLevelErr
					}

					t := tenant.MustFromContext(ctx)
					var res Model
					err = db.Transaction(func(tx *gorm.DB) error {
						res, err = create(tx, t.Id(), input.accountId, input.worldId, input.name, input.level, input.strength, input.dexterity, input.intelligence, input.luck, input.maxHp, input.maxMp, input.jobId, input.gender, input.hair, input.face, input.skinColor, input.mapId)
						if err != nil {
							l.WithError(err).Errorf("Error persisting character in database.")
							tx.Rollback()
							return err
						}

						inv, err := inventory.Create(l)(tx)(ctx)(res.id, 24)
						if err != nil {
							l.WithError(err).Errorf("Unable to create inventory for character during character creation.")
							tx.Rollback()
							return err
						}
						res = CloneModel(res).SetInventory(inv).Build()
						return nil
					})

					if err == nil {
						err = eventProducer(EnvEventTopicCharacterStatus)(createdEventProvider(res.Id(), res.WorldId(), res.Name()))
					}
					return res, err
				}
			}
		}
	}
}

func Delete(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(characterId uint32) error {
	return func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(characterId uint32) error {
		return func(ctx context.Context) func(eventProducer producer.Provider) func(characterId uint32) error {
			return func(eventProducer producer.Provider) func(characterId uint32) error {
				return func(characterId uint32) error {
					err := db.Transaction(func(tx *gorm.DB) error {
						c, err := GetById(ctx)(tx)(InventoryModelDecorator(l)(tx)(ctx))(characterId)
						if err != nil {
							return err
						}

						// delete equipment.
						err = equipment.Delete(l)(tx)(ctx)(c.equipment)
						if err != nil {
							l.WithError(err).Errorf("Unable to delete equipment for character with id [%d].", characterId)
							return err
						}

						// delete inventories.
						err = inventory.DeleteEquipableInventory(l)(tx)(ctx)(characterId, c.inventory.Equipable())
						if err != nil {
							l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
							return err
						}
						err = inventory.DeleteItemInventory(l)(tx)(ctx)(characterId, c.inventory.Useable())
						if err != nil {
							l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
							return err
						}
						err = inventory.DeleteItemInventory(l)(tx)(ctx)(characterId, c.inventory.Setup())
						if err != nil {
							l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
							return err
						}
						err = inventory.DeleteItemInventory(l)(tx)(ctx)(characterId, c.inventory.Etc())
						if err != nil {
							l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
							return err
						}
						err = inventory.DeleteItemInventory(l)(tx)(ctx)(characterId, c.inventory.Cash())
						if err != nil {
							l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
							return err
						}

						_ = inventory.GetLockRegistry().DeleteForCharacter(characterId)

						tenant := tenant.MustFromContext(ctx)
						err = delete(tx, tenant.Id(), characterId)
						if err != nil {
							return err
						}

						err = eventProducer(EnvEventTopicCharacterStatus)(deletedEventProvider(characterId, c.WorldId()))
						if err != nil {
							l.WithError(err).Errorf("Unable to notify to other services a character [%d] has been deleted.", characterId)
						}
						return nil
					})
					return err
				}
			}
		}
	}
}

func Login(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32) func(worldId byte) func(channelId byte) error {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32) func(worldId byte) func(channelId byte) error {
		return func(ctx context.Context) func(characterId uint32) func(worldId byte) func(channelId byte) error {
			return func(characterId uint32) func(worldId byte) func(channelId byte) error {
				return func(worldId byte) func(channelId byte) error {
					return func(channelId byte) error {
						return model.For(byIdProvider(db)(tenant.MustFromContext(ctx))(characterId), announceLogin(producer.ProviderImpl(l)(ctx))(worldId)(channelId))
					}
				}
			}
		}
	}
}

func announceLogin(provider producer.Provider) func(worldId byte) func(channelId byte) model.Operator[Model] {
	return func(worldId byte) func(channelId byte) model.Operator[Model] {
		return func(channelId byte) model.Operator[Model] {
			return func(c Model) error {
				return provider(EnvEventTopicCharacterStatus)(loginEventProvider(c.Id(), worldId, channelId, c.MapId()))
			}
		}
	}
}

func Logout(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32) func(worldId byte) func(channelId byte) error {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32) func(worldId byte) func(channelId byte) error {
		return func(ctx context.Context) func(characterId uint32) func(worldId byte) func(channelId byte) error {
			return func(characterId uint32) func(worldId byte) func(channelId byte) error {
				return func(worldId byte) func(channelId byte) error {
					return func(channelId byte) error {
						return model.For(byIdProvider(db)(tenant.MustFromContext(ctx))(characterId), announceLogout(producer.ProviderImpl(l)(ctx))(worldId)(channelId))
					}
				}
			}
		}
	}
}

func announceLogout(provider producer.Provider) func(worldId byte) func(channelId byte) model.Operator[Model] {
	return func(worldId byte) func(channelId byte) model.Operator[Model] {
		return func(channelId byte) model.Operator[Model] {
			return func(c Model) error {
				return provider(EnvEventTopicCharacterStatus)(logoutEventProvider(c.Id(), worldId, channelId, c.MapId()))
			}
		}
	}
}

func ChangeChannel(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32) func(worldId byte) func(channelId byte) func(oldChannelId byte) error {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32) func(worldId byte) func(channelId byte) func(oldChannelId byte) error {
		return func(ctx context.Context) func(characterId uint32) func(worldId byte) func(channelId byte) func(oldChannelId byte) error {
			return func(characterId uint32) func(worldId byte) func(channelId byte) func(oldChannelId byte) error {
				return func(worldId byte) func(channelId byte) func(oldChannelId byte) error {
					return func(channelId byte) func(oldChannelId byte) error {
						return func(oldChannelId byte) error {
							return model.For(byIdProvider(db)(tenant.MustFromContext(ctx))(characterId), announceChangeChannel(producer.ProviderImpl(l)(ctx))(worldId)(channelId)(oldChannelId))
						}
					}
				}
			}
		}
	}
}

func announceChangeChannel(provider producer.Provider) func(worldId byte) func(channelId byte) func(oldChannelId byte) model.Operator[Model] {
	return func(worldId byte) func(channelId byte) func(oldChannelId byte) model.Operator[Model] {
		return func(channelId byte) func(oldChannelId byte) model.Operator[Model] {
			return func(oldChannelId byte) model.Operator[Model] {
				return func(c Model) error {
					return provider(EnvEventTopicCharacterStatus)(changeChannelEventProvider(c.Id(), worldId, channelId, oldChannelId, c.MapId()))
				}
			}
		}
	}
}

func ChangeMap(l logrus.FieldLogger, db *gorm.DB, ctx context.Context) func(characterId uint32, worldId byte, channelId byte, mapId uint32, portalId uint32) error {
	return func(characterId uint32, worldId byte, channelId byte, mapId uint32, portalId uint32) error {
		cmf := changeMap(db)(ctx)(mapId)
		papf := positionAtPortal(l)(ctx)(mapId, portalId)
		amcf := announceMapChanged(producer.ProviderImpl(l)(ctx))(worldId, channelId, mapId, portalId)
		return model.For(byIdProvider(db)(tenant.MustFromContext(ctx))(characterId), model.ThenOperator(cmf, model.Operators(papf, amcf)))
	}
}

func changeMap(db *gorm.DB) func(ctx context.Context) func(mapId uint32) model.Operator[Model] {
	return func(ctx context.Context) func(mapId uint32) model.Operator[Model] {
		return func(mapId uint32) model.Operator[Model] {
			return func(c Model) error {
				tenant := tenant.MustFromContext(ctx)
				return dynamicUpdate(db)(SetMapId(mapId))(tenant.Id())(c)
			}
		}
	}
}

func positionAtPortal(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32, portalId uint32) model.Operator[Model] {
	return func(ctx context.Context) func(mapId uint32, portalId uint32) model.Operator[Model] {
		return func(mapId uint32, portalId uint32) model.Operator[Model] {
			return func(c Model) error {
				por, err := portal.GetInMapById(l, ctx)(mapId, portalId)
				if err != nil {
					return err
				}
				GetTemporalRegistry().UpdatePosition(c.Id(), por.X(), por.Y())
				return nil
			}
		}
	}
}

func announceMapChanged(provider producer.Provider) func(worldId byte, channelId byte, mapId uint32, portalId uint32) model.Operator[Model] {
	return func(worldId byte, channelId byte, mapId uint32, portalId uint32) model.Operator[Model] {
		return func(c Model) error {
			return provider(EnvEventTopicCharacterStatus)(mapChangedEventProvider(c.Id(), worldId, channelId, c.MapId(), mapId, portalId))
		}
	}
}

func ChangeJob(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, jobId uint16) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, jobId uint16) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, jobId uint16) error {
			return func(characterId uint32, worldId byte, channelId byte, jobId uint16) error {
				l.Debugf("Attempting to set character [%d] job to [%d].", characterId, jobId)
				txErr := db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(tx)()(characterId)
					if err != nil {
						return err
					}
					err = dynamicUpdate(tx)(SetJob(jobId))(t.Id())(c)
					if err != nil {
						return err
					}
					return nil
				})
				if txErr != nil {
					l.WithError(txErr).Errorf("Could not set character [%d] job to [%d].", characterId, jobId)
					return txErr
				}
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(jobChangedEventProvider(characterId, worldId, channelId, jobId))
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(worldId, channelId, characterId, []string{"JOB"}))
				return nil
			}
		}
	}
}

type ExperienceModel struct {
	experienceType string
	amount         uint32
	attr1          uint32
}

func NewExperienceModel(experienceType string, amount uint32, attr1 uint32) ExperienceModel {
	return ExperienceModel{experienceType, amount, attr1}
}

func AwardExperience(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, experience []ExperienceModel) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, experience []ExperienceModel) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, experience []ExperienceModel) error {
			return func(characterId uint32, worldId byte, channelId byte, experience []ExperienceModel) error {
				amount := uint32(0)
				for _, e := range experience {
					amount += e.amount
				}

				l.Debugf("Attempting to award character [%d] [%d] experience.", characterId, amount)
				awardedLevels := byte(0)
				current := uint32(0)
				txErr := db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(tx)()(characterId)
					if err != nil {
						return err
					}

					curLevel := c.Level()
					current = c.Experience() + amount
					for current > GetExperienceNeededForLevel(curLevel) {
						current -= GetExperienceNeededForLevel(curLevel)
						curLevel += 1
						awardedLevels += 1
					}

					err = dynamicUpdate(tx)(SetExperience(current))(t.Id())(c)
					if err != nil {
						return err
					}
					return nil
				})
				if txErr != nil {
					l.WithError(txErr).Errorf("Could not award character [%d] [%d] experience.", characterId, amount)
					return txErr
				}

				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(experienceChangedEventProvider(characterId, worldId, channelId, experience, current))
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(worldId, channelId, characterId, []string{"EXPERIENCE"}))
				if awardedLevels > 0 {
					_ = producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(awardLevelCommandProvider(characterId, worldId, channelId, awardedLevels))
				}
				return nil
			}
		}
	}
}

func AwardLevel(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, amount byte) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, amount byte) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, amount byte) error {
			return func(characterId uint32, worldId byte, channelId byte, amount byte) error {
				l.Debugf("Attempting to award character [%d] [%d] level(s).", characterId, amount)
				actual := amount
				current := byte(0)
				txErr := db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(tx)()(characterId)
					if err != nil {
						return err
					}

					if c.Level()+amount > MaxLevel {
						l.Debugf("Awarding [%d] level(s) would cause character [%d] to go over cap [%d]. Setting change to [%d].", amount, characterId, MaxLevel, actual)
						actual = MaxLevel - c.Level()
					}
					current = c.Level() + actual

					err = dynamicUpdate(tx)(SetLevel(current))(t.Id())(c)
					if err != nil {
						return err
					}
					return nil
				})
				if txErr != nil {
					l.WithError(txErr).Errorf("Could not award character [%d] [%d] level(s).", characterId, actual)
					return txErr
				}
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(levelChangedEventProvider(characterId, worldId, channelId, actual, current))
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(worldId, channelId, characterId, []string{"LEVEL"}))
				return nil
			}
		}
	}
}

type MovementSummary struct {
	X      int16
	Y      int16
	Stance byte
}

func MovementSummaryProvider(x int16, y int16, stance byte) model.Provider[MovementSummary] {
	return func() (MovementSummary, error) {
		return MovementSummary{
			X:      x,
			Y:      y,
			Stance: stance,
		}, nil
	}
}

func FoldMovementSummary(summary MovementSummary, e Element) (MovementSummary, error) {
	ms := MovementSummary{X: summary.X, Y: summary.Y, Stance: summary.Stance}
	if e.TypeStr == MovementTypeNormal {
		ms.X = e.X
		ms.Y = e.Y
		ms.Stance = e.MoveAction
	} else if e.TypeStr == MovementTypeJump || e.TypeStr == MovementTypeTeleport || e.TypeStr == MovementTypeStartFallDown {
		ms.Stance = e.MoveAction
	}
	return ms, nil
}

func Move(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) func(worldId byte) func(channelId byte) func(mapId uint32) func(movement Movement) error {
	return func(ctx context.Context) func(characterId uint32) func(worldId byte) func(channelId byte) func(mapId uint32) func(movement Movement) error {
		return func(characterId uint32) func(worldId byte) func(channelId byte) func(mapId uint32) func(movement Movement) error {
			return func(worldId byte) func(channelId byte) func(mapId uint32) func(movement Movement) error {
				return func(channelId byte) func(mapId uint32) func(movement Movement) error {
					return func(mapId uint32) func(movement Movement) error {
						return func(movement Movement) error {
							msp := model.Fold(model.FixedProvider(movement.Elements), MovementSummaryProvider(movement.StartX, movement.StartY, GetTemporalRegistry().GetById(characterId).Stance()), FoldMovementSummary)
							err := model.For(msp, updateTemporal(characterId))
							if err != nil {
								return err
							}
							return producer.ProviderImpl(l)(ctx)(EnvEventTopicMovement)(move(worldId, channelId, mapId, characterId, movement))
						}
					}
				}
			}
		}
	}
}

func updateTemporal(characterId uint32) model.Operator[MovementSummary] {
	return func(ms MovementSummary) error {
		GetTemporalRegistry().Update(characterId, ms.X, ms.Y, ms.Stance)
		return nil
	}
}

func RequestChangeMeso(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, amount int32, actorId uint32, actorType string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, amount int32, actorId uint32, actorType string) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, amount int32, actorId uint32, actorType string) error {
			return func(characterId uint32, amount int32, actorId uint32, actorType string) error {
				return db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(tx)()(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
						return err
					}
					if int64(c.Meso())+int64(amount) < 0 {
						l.Debugf("Request for character [%d] would leave their meso negative. Amount [%d]. Existing [%d].", characterId, amount, c.Meso())
						return producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(notEnoughMesoErrorStatusEventProvider(characterId, c.WorldId(), amount))
					}
					if amount > 0 && uint32(amount) > (math.MaxUint32-c.Meso()) {
						l.Errorf("Transaction for character [%d] would result in a uint32 overflow. Rejecting transaction.", characterId)
						return err
					}

					err = dynamicUpdate(tx)(SetMeso(uint32(int64(c.Meso()) + int64(amount))))(t.Id())(c)
					_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(mesoChangedStatusEventProvider(characterId, c.WorldId(), amount, actorId, actorType))
					return producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(c.WorldId(), 0, characterId, []string{"MESO"}))
				})
			}
		}
	}
}

func AttemptMesoPickUp(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, dropId uint32, meso uint32) error {
	return func(db *gorm.DB) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, dropId uint32, meso uint32) error {
		return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, dropId uint32, meso uint32) error {
			t := tenant.MustFromContext(ctx)
			return func(worldId byte, channelId byte, mapId uint32, characterId uint32, dropId uint32, meso uint32) error {
				txErr := db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(tx)()(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
						return err
					}
					if meso > (math.MaxUint32 - c.Meso()) {
						l.Errorf("Transaction for character [%d] would result in a uint32 overflow. Rejecting transaction.", characterId)
						return err
					}

					err = dynamicUpdate(tx)(SetMeso(uint32(int64(c.Meso()) + int64(meso))))(t.Id())(c)
					return producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(c.WorldId(), 0, characterId, []string{"MESO"}))
				})
				if txErr != nil {
					return txErr
				}
				return drop.RequestPickUp(l)(ctx)(worldId, channelId, mapId, dropId, characterId)
			}
		}
	}
}

func RequestDropMeso(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, channelId byte, mapId uint32, characterId uint32, amount uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, channelId byte, mapId uint32, characterId uint32, amount uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, channelId byte, mapId uint32, characterId uint32, amount uint32) error {
			return func(worldId byte, channelId byte, mapId uint32, characterId uint32, amount uint32) error {
				txErr := db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(tx)()(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
						return err
					}
					if int64(c.Meso())-int64(amount) < 0 {
						l.Debugf("Request for character [%d] would leave their meso negative. Amount [%d]. Existing [%d].", characterId, amount, c.Meso())
						return producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(notEnoughMesoErrorStatusEventProvider(characterId, c.WorldId(), int32(amount)))
					}

					return dynamicUpdate(tx)(SetMeso(c.Meso() - amount))(t.Id())(c)
				})
				if txErr != nil {
					return txErr
				}

				tc := GetTemporalRegistry().GetById(characterId)

				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(worldId, channelId, characterId, []string{"MESO"}))
				// TODO determine appropriate drop type and mod
				_ = drop.CreateForMesos(l)(ctx)(worldId, channelId, mapId, amount, 2, tc.X(), tc.Y(), characterId)
				return nil
			}
		}
	}
}

func RequestChangeFame(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, amount int8, actorId uint32, actorType string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, amount int8, actorId uint32, actorType string) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, amount int8, actorId uint32, actorType string) error {
			return func(characterId uint32, amount int8, actorId uint32, actorType string) error {
				return db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(tx)()(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their fame adjusted.", characterId)
						return err
					}

					total := c.Fame() + int16(amount)
					err = dynamicUpdate(tx)(SetFame(total))(t.Id())(c)
					_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(fameChangedStatusEventProvider(characterId, c.WorldId(), amount, actorId, actorType))
					return producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(c.WorldId(), 0, characterId, []string{"FAME"}))
				})
			}
		}
	}
}

type Distribution struct {
	Ability string
	Amount  int8
}

func RequestDistributeAp(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, distributions []Distribution) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, distributions []Distribution) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, distributions []Distribution) error {
			return func(characterId uint32, distributions []Distribution) error {
				return db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(tx)()(characterId)
					if err != nil {
						_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(c.WorldId(), 0, characterId, []string{}))
						return err
					}
					if c.AP() < uint16(len(distributions)) {
						_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(c.WorldId(), 0, characterId, []string{}))
						return errors.New("not enough ap")
					}

					var eufs = make([]EntityUpdateFunction, 0)
					var stat = make([]string, 0)

					spent := uint16(0)
					for _, d := range distributions {
						switch d.Ability {
						case CommandDistributeApAbilityStrength:
							eufs = append(eufs, SetStrength(uint16(int16(c.Strength())+int16(d.Amount))))
							stat = append(stat, "STRENGTH")
							break
						case CommandDistributeApAbilityDexterity:
							eufs = append(eufs, SetDexterity(uint16(int16(c.Dexterity())+int16(d.Amount))))
							stat = append(stat, "DEXTERITY")
							break
						case CommandDistributeApAbilityIntelligence:
							eufs = append(eufs, SetIntelligence(uint16(int16(c.Intelligence())+int16(d.Amount))))
							stat = append(stat, "INTELLIGENCE")
							break
						case CommandDistributeApAbilityLuck:
							eufs = append(eufs, SetLuck(uint16(int16(c.Luck())+int16(d.Amount))))
							stat = append(stat, "LUCK")
							break
						case CommandDistributeApAbilityHp:
							hpGrowth, err := getMaxHpGrowth(l)(ctx)(c)
							if err != nil {
								return err
							}
							eufs = append(eufs, SetMaxHP(uint16(int16(hpGrowth)*int16(d.Amount))))
							eufs = append(eufs, SetHPMPUsed(c.HPMPUsed()+int(d.Amount)))
							stat = append(stat, "MAX_HP")
							break
						case CommandDistributeApAbilityMp:
							mpGrowth, err := getMaxMpGrowth(l)(ctx)(c)
							if err != nil {
								return err
							}
							eufs = append(eufs, SetMaxMP(uint16(int16(mpGrowth)*int16(d.Amount))))
							eufs = append(eufs, SetHPMPUsed(c.HPMPUsed()+int(d.Amount)))
							stat = append(stat, "MAX_MP")
							break
						}
						spent = uint16(int16(spent) + int16(d.Amount))
					}

					if len(eufs) == 0 {
						_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(c.WorldId(), 0, characterId, []string{}))
						return errors.New("invalid ability")
					}

					eufs = append(eufs, SetAP(c.AP()-spent))
					stat = append(stat, "AVAILABLE_AP")

					err = dynamicUpdate(tx)(eufs...)(t.Id())(c)
					if err != nil {
						_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(c.WorldId(), 0, characterId, []string{"AVAILABLE_AP"}))
						return err
					}

					_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(c.WorldId(), 0, characterId, stat))
					return nil
				})
			}
		}
	}
}

func RequestDistributeSp(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, skillId uint32, amount int8) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, skillId uint32, amount int8) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, skillId uint32, amount int8) error {
			var c Model
			return func(characterId uint32, skillId uint32, amount int8) error {
				txErr := db.Transaction(func(tx *gorm.DB) error {
					var err error
					c, err = GetById(ctx)(db)(SkillModelDecorator(l)(ctx))(characterId)
					if err != nil {
						return err
					}
					sjid, ok := job.FromSkillId(skill.Id(skillId))
					if !ok {
						return errors.New("unable to locate job from skill")
					}
					sb := getSkillBook(sjid.Id())
					if c.SP(sb) <= uint32(amount) {
						return errors.New("not enough sp")
					}
					return dynamicUpdate(tx)(SetSP(c.SP(sb)-uint32(amount), uint32(sb)))(t.Id())(c)
				})
				if txErr != nil {
					return txErr
				}
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(c.WorldId(), 0, characterId, []string{"AVAILABLE_SP"}))

				if val := c.GetSkill(skillId); val.Id() != skillId {
					_ = skill2.RequestCreate(l)(ctx)(characterId, skillId, byte(amount), 0, time.Time{})
				} else {
					_ = skill2.RequestUpdate(l)(ctx)(characterId, skillId, val.Level()+byte(amount), val.MasterLevel(), val.Expiration())
				}
				return nil
			}
		}
	}
}

func getMaxHpGrowth(l logrus.FieldLogger) func(ctx context.Context) func(c Model) (uint16, error) {
	return func(ctx context.Context) func(c Model) (uint16, error) {
		return func(c Model) (uint16, error) {
			if c.MaxHP() >= 30000 || c.HPMPUsed() > 9999 {
				return c.MaxHP(), errors.New("max ap to hp")
			}
			var improvingHPSkillId skill.Id
			resMax := c.MaxHP()
			if job.IsA(job.Id(c.JobId()),
				job.WarriorId,
				job.FighterId, job.CrusaderId, job.HeroId,
				job.PageId, job.CrusaderId, job.WhiteKnightId,
				job.SpearmanId, job.DragonKnightId, job.DarkKnightId,
				job.DawnWarriorStage1Id, job.DawnWarriorStage2Id, job.DawnWarriorStage3Id, job.DawnWarriorStage4Id,
				job.AranStage1Id, job.AranStage2Id, job.AranStage3Id, job.AranStage4Id) {
				if job.IsCygnus(job.Id(c.JobId())) {
					improvingHPSkillId = skill.DawnWarriorStage1ImprovedMaxHpIncreaseId
				} else {
					improvingHPSkillId = skill.WarriorImprovedMaxHpIncreaseId
				}
				resMax += 20
			} else if job.IsA(job.Id(c.JobId()),
				job.MagicianId,
				job.FirePoisonWizardId, job.FirePoisonMagicianId, job.FirePoisonArchMagicianId,
				job.IceLightningWizardId, job.IceLightningMagicianId, job.IceLightningArchMagicianId,
				job.ClericId, job.PriestId, job.BishopId,
				job.BlazeWizardStage1Id, job.BlazeWizardStage2Id, job.BlazeWizardStage3Id, job.BlazeWizardStage4Id) {
				resMax += 6
			} else if job.IsA(job.Id(c.JobId()),
				job.BowmanId,
				job.HunterId, job.RangerId, job.BowmasterId,
				job.CrossbowmanId, job.SniperId, job.MarksmanId,
				job.WindArcherStage1Id, job.WindArcherStage2Id, job.WindArcherStage3Id, job.WindArcherStage4Id,
				job.RogueId,
				job.AssassinId, job.HermitId, job.NightLordId,
				job.BanditId, job.ChiefBanditId, job.ShadowerId,
				job.NightWalkerStage1Id, job.NightWalkerStage2Id, job.NightWalkerStage3Id, job.NightWalkerStage4Id) {
				resMax += 16
			} else if job.IsA(job.Id(c.JobId()),
				job.PirateId,
				job.BrawlerId, job.MarauderId, job.BuccaneerId,
				job.GunslingerId, job.OutlawId, job.CorsairId,
				job.ThunderBreakerStage1Id, job.ThunderBreakerStage2Id, job.ThunderBreakerStage3Id, job.ThunderBreakerStage4Id) {
				if job.IsCygnus(job.Id(c.JobId())) {
					improvingHPSkillId = skill.ThunderBreakerStage2ImprovedMaxHpIncreaseId
				} else {
					improvingHPSkillId = skill.BrawlerImproveMaxHpId
				}
				resMax += 18
			} else {
				resMax += 8
			}

			if improvingHPSkillId > 0 {
				var improvingHPSkillLevel = c.GetSkillLevel(uint32(improvingHPSkillId))
				se, err := skill3.GetEffect(l)(ctx)(uint32(improvingHPSkillId), improvingHPSkillLevel)
				if err == nil {
					resMax = uint16(int16(resMax) + se.Y())
				}
			}
			return resMax, nil
		}
	}
}

func getMaxMpGrowth(l logrus.FieldLogger) func(ctx context.Context) func(c Model) (uint16, error) {
	return func(ctx context.Context) func(c Model) (uint16, error) {
		return func(c Model) (uint16, error) {
			if c.MaxMP() >= 30000 || c.HPMPUsed() > 9999 {
				return c.MaxMP(), errors.New("max ap to mp")
			}
			var improvingMPSkillId skill.Id
			resMax := c.MaxMP()
			if job.IsA(job.Id(c.JobId()),
				job.WarriorId,
				job.FighterId, job.CrusaderId, job.HeroId,
				job.PageId, job.CrusaderId, job.WhiteKnightId,
				job.SpearmanId, job.DragonKnightId, job.DarkKnightId,
				job.DawnWarriorStage1Id, job.DawnWarriorStage2Id, job.DawnWarriorStage3Id, job.DawnWarriorStage4Id,
				job.AranStage1Id, job.AranStage2Id, job.AranStage3Id, job.AranStage4Id) {
				if job.IsA(job.Id(c.JobId()), job.CrusaderId, job.WhiteKnightId) {
					improvingMPSkillId = skill.WhiteKnightImprovingMpRecoveryId
				} else if job.IsA(job.Id(c.JobId()), job.DawnWarriorStage3Id, job.DawnWarriorStage4Id) {
					improvingMPSkillId = skill.DawnWarriorStage3ImprovedMpRecoveryId
				}
				resMax += 2
			} else if job.IsA(job.Id(c.JobId()),
				job.MagicianId,
				job.FirePoisonWizardId, job.FirePoisonMagicianId, job.FirePoisonArchMagicianId,
				job.IceLightningWizardId, job.IceLightningMagicianId, job.IceLightningArchMagicianId,
				job.ClericId, job.PriestId, job.BishopId,
				job.BlazeWizardStage1Id, job.BlazeWizardStage2Id, job.BlazeWizardStage3Id, job.BlazeWizardStage4Id) {
				if job.IsCygnus(job.Id(c.JobId())) {
					improvingMPSkillId = skill.BlazeWizardStage1ImprovedMaxMpIncreaseId
				} else {
					improvingMPSkillId = skill.MagicianImprovedMaxMpIncreaseId
				}
				resMax += 18
			} else if job.IsA(job.Id(c.JobId()),
				job.BowmanId,
				job.HunterId, job.RangerId, job.BowmasterId,
				job.CrossbowmanId, job.SniperId, job.MarksmanId,
				job.WindArcherStage1Id, job.WindArcherStage2Id, job.WindArcherStage3Id, job.WindArcherStage4Id,
				job.RogueId,
				job.AssassinId, job.HermitId, job.NightLordId,
				job.BanditId, job.ChiefBanditId, job.ShadowerId,
				job.NightWalkerStage1Id, job.NightWalkerStage2Id, job.NightWalkerStage3Id, job.NightWalkerStage4Id) {
				resMax += 10
			} else if job.IsA(
				job.PirateId,
				job.BrawlerId, job.MarauderId, job.BuccaneerId,
				job.GunslingerId, job.OutlawId, job.CorsairId,
				job.ThunderBreakerStage1Id, job.ThunderBreakerStage2Id, job.ThunderBreakerStage3Id, job.ThunderBreakerStage4Id) {
				resMax += 14
			} else {
				resMax += 6
			}
			// TODO this needs to incorporate computed total intelligence (buffs, weapons, etc)
			resMax += uint16(math.Ceil(float64(c.Intelligence()) / 10))

			if improvingMPSkillId > 0 {
				var improvingMPSkillLevel = c.GetSkillLevel(uint32(improvingMPSkillId))
				se, err := skill3.GetEffect(l)(ctx)(uint32(improvingMPSkillId), improvingMPSkillLevel)
				if err == nil {
					resMax = uint16(int16(resMax) + se.X())
				}
			}

			return resMax, nil
		}
	}
}

func enforceBounds(change int16, current uint16, upperBound uint16, lowerBound uint16) uint16 {
	var adjusted = int16(current) + change
	return uint16(math.Min(math.Max(float64(adjusted), float64(lowerBound)), float64(upperBound)))
}

func ChangeHP(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, amount int16) error {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, amount int16) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, amount int16) error {
			return func(worldId byte, channelId byte, characterId uint32, amount int16) error {
				txErr := db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(db)()(characterId)
					if err != nil {
						return err
					}
					// TODO consider effective (temporary) Max HP.
					adjusted := enforceBounds(amount, c.HP(), c.MaxHP(), 0)
					l.Debugf("Attempting to adjust character [%d] health by [%d] to [%d].", characterId, amount, adjusted)
					return dynamicUpdate(tx)(SetHealth(adjusted))(t.Id())(c)
				})
				if txErr != nil {
					return txErr
				}
				// TODO need to emit event when character dies.
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(worldId, channelId, characterId, []string{"HP"}))
				return nil
			}
		}
	}
}

func ChangeMP(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, amount int16) error {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, amount int16) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, amount int16) error {
			return func(worldId byte, channelId byte, characterId uint32, amount int16) error {
				txErr := db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(tx)()(characterId)
					if err != nil {
						return err
					}
					// TODO consider effective (temporary) Max MP.
					adjusted := enforceBounds(amount, c.MP(), c.MaxMP(), 0)
					l.Debugf("Attempting to adjust character [%d] mana by [%d] to [%d].", characterId, amount, adjusted)
					return dynamicUpdate(tx)(SetMana(adjusted))(t.Id())(c)
				})
				if txErr != nil {
					return txErr
				}
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(worldId, channelId, characterId, []string{"MP"}))
				return nil
			}
		}
	}
}

func ProcessLevelChange(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, amount byte) error {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, amount byte) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, amount byte) error {
			return func(worldId byte, channelId byte, characterId uint32, amount byte) error {
				var addedAP uint16
				var addedSP uint32
				var addedHP uint16
				var addedMP uint16
				var addedStr uint16
				var addedDex uint16
				var sus = []string{"AVAILABLE_AP", "AVAILABLE_SP", "HP", "MAX_HP", "MP", "MAX_MP"}

				txErr := db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(tx)(SkillModelDecorator(l)(ctx))(characterId)
					if err != nil {
						return err
					}

					effectiveLevel := c.Level() - amount

					for i := range amount {
						effectiveLevel = effectiveLevel + i + 1

						if t.Region() == "GMS" && t.MajorVersion() == 83 {
							// TODO properly define this range. For these versions, Beginner, Noblesse, and Legend AP are auto assigned.
							if job.IsBeginner(job.Id(c.JobId())) && effectiveLevel < 11 {
								if effectiveLevel < 6 {
									addedStr += 5
								} else {
									addedStr += 4
									addedDex += 1
								}
							} else {
								addedAP += computeOnLevelAddedAP(job.Id(c.JobId()), effectiveLevel)
							}
						} else {
							addedAP += computeOnLevelAddedAP(job.Id(c.JobId()), effectiveLevel)
						}

						addedSP += computeOnLevelAddedSP(job.Id(c.JobId()))
						// TODO could potentially pre-compute HP and MP so you don't incur loop cost
						aHP, aMP := computeOnLevelAddedHPandMP(l)(ctx)(c)
						addedHP += aHP
						addedMP += aMP
					}

					l.Debugf("As a result of processing a level change of [%d]. Character [%d] will gain [%d] AP, [%d] SP, [%d] HP, and [%d] MP.", amount, characterId, addedAP, addedSP, addedHP, addedMP)
					sb := getSkillBook(job.Id(c.JobId()))

					var eufs = []EntityUpdateFunction{
						SetAP(c.AP() + addedAP),
						SetSP(c.SP(sb)+addedSP, uint32(sb)),
						SetHealth(c.MaxHP() + addedHP),
						SetMaxHP(c.MaxHP() + addedHP),
						SetMana(c.MaxMP() + addedMP),
						SetMaxMP(c.MaxMP() + addedMP),
					}

					if addedStr > 0 {
						eufs = append(eufs, SetStrength(c.Strength()+addedStr))
						sus = append(sus, "STRENGTH")
					}
					if addedDex > 0 {
						eufs = append(eufs, SetDexterity(c.Dexterity()+addedDex))
						sus = append(sus, "DEXTERITY")
					}

					return dynamicUpdate(tx)(eufs...)(t.Id())(c)
				})
				if txErr != nil {
					return txErr
				}
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(worldId, channelId, characterId, sus))
				return nil
			}
		}
	}
}

func computeOnLevelAddedAP(jobId job.Id, level byte) uint16 {
	toGain := uint16(5)
	if job.IsCygnus(jobId) {
		if level > 10 {
			if level <= 17 {
				toGain += 2
			} else if level < 77 {
				toGain += 1
			}
		}
	}
	return toGain
}

func computeOnLevelAddedSP(jobId job.Id) uint32 {
	// TODO need to account for 6 beginner skill levels
	if job.IsBeginner(jobId) {
		return 0
	}
	return 3
}

func computeOnLevelAddedHPandMP(l logrus.FieldLogger) func(ctx context.Context) func(c Model) (uint16, uint16) {
	return func(ctx context.Context) func(c Model) (uint16, uint16) {
		return func(c Model) (uint16, uint16) {
			var addedHP uint16
			var addedMP uint16
			var improvingHPSkillId skill.Id
			var improvingMPSkillId skill.Id

			randBoundFunc := func(lower uint16, upper uint16) uint16 {
				return uint16(rand.Float32()*float32(upper-lower+1)) + lower
			}

			if job.IsBeginner(job.Id(c.JobId())) {
				addedHP = randBoundFunc(12, 16)
				addedMP = randBoundFunc(10, 12)
			} else if job.IsA(job.Id(c.JobId()),
				job.WarriorId,
				job.FighterId, job.CrusaderId, job.HeroId,
				job.PageId, job.CrusaderId, job.WhiteKnightId,
				job.SpearmanId, job.DragonKnightId, job.DarkKnightId,
				job.DawnWarriorStage1Id, job.DawnWarriorStage2Id, job.DawnWarriorStage3Id, job.DawnWarriorStage4Id) {
				if job.IsCygnus(job.Id(c.JobId())) {
					improvingHPSkillId = skill.DawnWarriorStage1ImprovedMaxHpIncreaseId
				} else {
					improvingHPSkillId = skill.WarriorImprovedMaxHpIncreaseId
				}
				if job.IsA(job.Id(c.JobId()), job.CrusaderId, job.WhiteKnightId) {
					improvingMPSkillId = skill.WhiteKnightImprovingMpRecoveryId
				} else if job.IsA(job.Id(c.JobId()), job.DawnWarriorStage3Id, job.DawnWarriorStage4Id) {
					improvingMPSkillId = skill.DawnWarriorStage3ImprovedMpRecoveryId
				}
				addedHP = randBoundFunc(24, 28)
				addedMP = randBoundFunc(4, 6)
			} else if job.IsA(job.Id(c.JobId()),
				job.MagicianId,
				job.FirePoisonWizardId, job.FirePoisonMagicianId, job.FirePoisonArchMagicianId,
				job.IceLightningWizardId, job.IceLightningMagicianId, job.IceLightningArchMagicianId,
				job.ClericId, job.PriestId, job.BishopId,
				job.BlazeWizardStage1Id, job.BlazeWizardStage2Id, job.BlazeWizardStage3Id, job.BlazeWizardStage4Id) {
				if job.IsCygnus(job.Id(c.JobId())) {
					improvingMPSkillId = skill.BlazeWizardStage1ImprovedMaxMpIncreaseId
				} else {
					improvingMPSkillId = skill.MagicianImprovedMaxMpIncreaseId
				}
				addedHP = randBoundFunc(10, 14)
				addedMP = randBoundFunc(22, 24)
			} else if job.IsA(job.Id(c.JobId()),
				job.BowmanId,
				job.HunterId, job.RangerId, job.BowmasterId,
				job.CrossbowmanId, job.SniperId, job.MarksmanId,
				job.WindArcherStage1Id, job.WindArcherStage2Id, job.WindArcherStage3Id, job.WindArcherStage4Id,
				job.RogueId,
				job.AssassinId, job.HermitId, job.NightLordId,
				job.BanditId, job.ChiefBanditId, job.ShadowerId,
				job.NightWalkerStage1Id, job.NightWalkerStage2Id, job.NightWalkerStage3Id, job.NightWalkerStage4Id) {
				addedHP = randBoundFunc(20, 24)
				addedMP = randBoundFunc(14, 16)
			} else if job.IsA(job.Id(c.JobId()), job.GmId, job.SuperGmId) {
				addedHP = 30000
				addedMP = 30000
			} else if job.IsA(
				job.PirateId,
				job.BrawlerId, job.MarauderId, job.BuccaneerId,
				job.GunslingerId, job.OutlawId, job.CorsairId,
				job.ThunderBreakerStage1Id, job.ThunderBreakerStage2Id, job.ThunderBreakerStage3Id, job.ThunderBreakerStage4Id) {
				if job.IsCygnus(job.Id(c.JobId())) {
					improvingHPSkillId = skill.ThunderBreakerStage2ImprovedMaxHpIncreaseId
				} else {
					improvingHPSkillId = skill.BrawlerImproveMaxHpId
				}
				addedHP = randBoundFunc(22, 28)
				addedMP = randBoundFunc(18, 23)
			} else if job.IsA(job.Id(c.JobId()), job.AranStage1Id, job.AranStage2Id, job.AranStage3Id, job.AranStage4Id) {
				addedHP = randBoundFunc(44, 48)
				addedMP = randBoundFunc(4, 8)
			}

			if improvingHPSkillId > 0 {
				var improvingHPSkillLevel = c.GetSkillLevel(uint32(improvingHPSkillId))
				se, err := skill3.GetEffect(l)(ctx)(uint32(improvingHPSkillId), improvingHPSkillLevel)
				if err == nil {
					addedHP = uint16(int16(addedHP) + se.X())
				}
			}
			if improvingMPSkillId > 0 {
				var improvingMPSkillLevel = c.GetSkillLevel(uint32(improvingMPSkillId))
				se, err := skill3.GetEffect(l)(ctx)(uint32(improvingMPSkillId), improvingMPSkillLevel)
				if err == nil {
					addedMP = uint16(int16(addedMP) + se.X())
				}
			}
			return addedHP, addedMP
		}
	}
}

func ProcessJobChange(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, jobId uint16) error {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, jobId uint16) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, jobId uint16) error {
			return func(worldId byte, channelId byte, characterId uint32, jobId uint16) error {
				var addedAP uint16
				var addedSP uint32
				var addedHP uint16
				var addedMP uint16

				randBoundFunc := func(lower uint16, upper uint16) uint16 {
					return uint16(rand.Float32()*float32(upper-lower+1)) + lower
				}

				txErr := db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(tx)(SkillModelDecorator(l)(ctx))(characterId)
					if err != nil {
						return err
					}

					// TODO award job change AP is this only Cygnus?
					if job.IsCygnus(job.Id(jobId)) {
						addedAP = 7
					}

					addedSP = 1
					if job.IsA(job.Id(jobId), job.EvanId, job.EvanStage1Id, job.EvanStage2Id, job.EvanStage3Id, job.EvanStage4Id, job.EvanStage5Id, job.EvanStage6Id, job.EvanStage7Id, job.EvanStage8Id, job.EvanStage9Id, job.EvanStage10Id) {
						addedAP += 2
					} else if job.IsFourthJob(job.Id(jobId)) {
						addedSP += 2
					}

					if job.IsA(job.Id(jobId), job.WarriorId, job.DawnWarriorStage1Id, job.AranStage1Id) {
						addedHP = randBoundFunc(200, 250)
					} else if job.IsA(job.Id(jobId), job.MagicianId, job.BlazeWizardStage1Id, job.EvanStage1Id) {
						addedMP = randBoundFunc(100, 150)
					} else if job.IsA(job.Id(jobId), job.BowmanId, job.RogueId, job.PirateId, job.WindArcherStage1Id, job.NightWalkerStage1Id, job.ThunderBreakerStage1Id) {
						addedHP = randBoundFunc(100, 150)
						addedMP = randBoundFunc(25, 50)
					} else if job.IsA(job.Id(jobId),
						job.FighterId, job.CrusaderId, job.HeroId,
						job.PageId, job.CrusaderId, job.WhiteKnightId,
						job.SpearmanId, job.DragonKnightId, job.DarkKnightId,
						job.DawnWarriorStage2Id, job.DawnWarriorStage3Id, job.DawnWarriorStage4Id,
						job.AranStage2Id, job.AranStage3Id, job.AranStage4Id) {
						addedHP = randBoundFunc(300, 350)
					} else if job.IsA(job.Id(jobId),
						job.FirePoisonWizardId, job.FirePoisonMagicianId, job.FirePoisonArchMagicianId,
						job.IceLightningWizardId, job.IceLightningMagicianId, job.IceLightningArchMagicianId,
						job.ClericId, job.PriestId, job.BishopId,
						job.BlazeWizardStage2Id, job.BlazeWizardStage3Id, job.BlazeWizardStage4Id,
						job.EvanStage2Id, job.EvanStage3Id, job.EvanStage4Id, job.EvanStage5Id, job.EvanStage6Id, job.EvanStage7Id, job.EvanStage8Id, job.EvanStage9Id, job.EvanStage10Id) {
						addedMP = randBoundFunc(450, 500)
					} else if !job.IsBeginner(job.Id(jobId)) {
						addedHP = randBoundFunc(300, 350)
						addedMP = randBoundFunc(150, 200)
					}

					l.Debugf("As a result of processing a job change to [%d]. Character [%d] will gain [%d] AP, [%d] SP, [%d] HP, and [%d] MP.", jobId, characterId, addedAP, addedSP, addedHP, addedMP)
					sb := getSkillBook(job.Id(c.JobId()))
					return dynamicUpdate(tx)(SetAP(c.AP()+addedAP), SetSP(c.SP(sb)+addedSP, uint32(sb)), SetHealth(c.MaxHP()+addedHP), SetMaxHP(c.MaxHP()+addedHP), SetMana(c.MaxMP()+addedMP), SetMaxMP(c.MaxMP()+addedMP))(t.Id())(c)
				})
				if txErr != nil {
					return txErr
				}
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicCharacterStatus)(statChangedProvider(worldId, channelId, characterId, []string{"AVAILABLE_AP", "AVAILABLE_SP", "HP", "MAX_HP", "MP", "MAX_MP"}))
				return nil
			}
		}
	}
}

func getSkillBook(jobId job.Id) int {
	if jobId >= job.EvanStage2Id && jobId <= job.EvanStage10Id {
		return int(jobId - 2209)
	}
	return 0
}
