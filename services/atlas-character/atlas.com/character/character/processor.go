package character

import (
	"atlas-character/drop"
	"atlas-character/equipable"
	"atlas-character/equipment"
	"atlas-character/equipment/slot"
	"atlas-character/inventory"
	"atlas-character/kafka/producer"
	"atlas-character/portal"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
	"regexp"
	"strconv"
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
				willLevel := false
				current := uint32(0)
				txErr := db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(ctx)(tx)()(characterId)
					if err != nil {
						return err
					}

					if c.Experience()+amount >= GetExperienceNeededForLevel(c.Level()) {
						current = c.Experience() + amount - GetExperienceNeededForLevel(c.Level())
						willLevel = true
					} else {
						current = c.Experience() + amount
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
				if willLevel {
					_ = producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(awardLevelCommandProvider(characterId, worldId, channelId, 1))
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

func getMaxHpGrowth(_ logrus.FieldLogger) func(ctx context.Context) func(c Model) (uint16, error) {
	return func(_ context.Context) func(c Model) (uint16, error) {
		return func(c Model) (uint16, error) {
			if c.MaxHP() >= 30000 || c.HPMPUsed() > 9999 {
				return c.MaxHP(), errors.New("max ap to hp")
			}
			resMax := c.MaxHP()
			if job.IsA(job.Id(c.JobId()),
				job.WarriorId,
				job.FighterId, job.CrusaderId, job.HeroId,
				job.PageId, job.CrusaderId, job.WhiteKnightId,
				job.SpearmanId, job.DragonKnightId, job.DarkKnightId,
				job.DawnWarriorStage1Id, job.DawnWarriorStage2Id, job.DawnWarriorStage3Id, job.DawnWarriorStage4Id,
				job.AranStage1Id, job.AranStage2Id, job.AranStage3Id, job.AranStage4Id) {
				// TODO include MAX_HP_INCREASE skill
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
				// TODO include IMPROVE_MAX_HP
				resMax += 18
			} else {
				resMax += 8
			}
			return resMax, nil
		}
	}
}

func getMaxMpGrowth(_ logrus.FieldLogger) func(ctx context.Context) func(c Model) (uint16, error) {
	return func(_ context.Context) func(c Model) (uint16, error) {
		return func(c Model) (uint16, error) {
			if c.MaxMP() >= 30000 || c.HPMPUsed() > 9999 {
				return c.MaxMP(), errors.New("max ap to mp")
			}
			resMax := c.MaxMP()
			if job.IsA(job.Id(c.JobId()),
				job.WarriorId,
				job.FighterId, job.CrusaderId, job.HeroId,
				job.PageId, job.CrusaderId, job.WhiteKnightId,
				job.SpearmanId, job.DragonKnightId, job.DarkKnightId,
				job.DawnWarriorStage1Id, job.DawnWarriorStage2Id, job.DawnWarriorStage3Id, job.DawnWarriorStage4Id,
				job.AranStage1Id, job.AranStage2Id, job.AranStage3Id, job.AranStage4Id) {
				resMax += 2
			} else if job.IsA(job.Id(c.JobId()),
				job.MagicianId,
				job.FirePoisonWizardId, job.FirePoisonMagicianId, job.FirePoisonArchMagicianId,
				job.IceLightningWizardId, job.IceLightningMagicianId, job.IceLightningArchMagicianId,
				job.ClericId, job.PriestId, job.BishopId,
				job.BlazeWizardStage1Id, job.BlazeWizardStage2Id, job.BlazeWizardStage3Id, job.BlazeWizardStage4Id) {
				// TODO include INCREASING_MAX_MP skill
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
					c, err := GetById(ctx)(db)()(characterId)
					if err != nil {
						return err
					}
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
