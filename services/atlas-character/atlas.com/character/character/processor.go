package character

import (
	"atlas-character/database"
	"atlas-character/equipable"
	"atlas-character/equipment"
	"atlas-character/equipment/slot"
	"atlas-character/inventory"
	"atlas-character/kafka/producer"
	"atlas-character/portal"
	"atlas-character/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"regexp"
)

var blockedNameErr = errors.New("blocked name")
var invalidLevelErr = errors.New("invalid level")

func byIdProvider(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32) model.Provider[Model] {
	return func(characterId uint32) model.Provider[Model] {
		return database.ModelProvider[Model, entity](db)(getById(tenant.Id, characterId), makeCharacter)
	}
}

func GetById(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, decorators ...model.Decorator[Model]) (Model, error) {
	return func(characterId uint32, decorators ...model.Decorator[Model]) (Model, error) {
		return model.Map(byIdProvider(l, db, tenant)(characterId), model.Decorate(decorators...))()
	}
}

func byAccountInWorldProvider(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(accountId uint32, worldId byte) model.Provider[[]Model] {
	return func(accountId uint32, worldId byte) model.Provider[[]Model] {
		return database.ModelSliceProvider[Model, entity](db)(getForAccountInWorld(tenant.Id, accountId, worldId), makeCharacter)
	}
}

func GetForAccountInWorld(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(accountId uint32, worldId byte, decorators ...model.Decorator[Model]) ([]Model, error) {
	return func(accountId uint32, worldId byte, decorators ...model.Decorator[Model]) ([]Model, error) {
		return model.SliceMap(byAccountInWorldProvider(l, db, tenant)(accountId, worldId), model.Decorate(decorators...))()
	}
}

func byMapInWorld(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(worldId byte, mapId uint32) model.Provider[[]Model] {
	return func(worldId byte, mapId uint32) model.Provider[[]Model] {
		return database.ModelSliceProvider[Model, entity](db)(getForMapInWorld(tenant.Id, worldId, mapId), makeCharacter)
	}
}

func GetForMapInWorld(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(worldId byte, mapId uint32, decorators ...model.Decorator[Model]) ([]Model, error) {
	return func(worldId byte, mapId uint32, decorators ...model.Decorator[Model]) ([]Model, error) {
		return model.SliceMap(byMapInWorld(l, db, tenant)(worldId, mapId), model.Decorate(decorators...))()
	}
}

func byName(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(name string) model.Provider[[]Model] {
	return func(name string) model.Provider[[]Model] {
		return database.ModelSliceProvider[Model, entity](db)(getForName(tenant.Id, name), makeCharacter)
	}
}

func GetForName(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(name string, decorators ...model.Decorator[Model]) ([]Model, error) {
	return func(name string, decorators ...model.Decorator[Model]) ([]Model, error) {
		return model.SliceMap(byName(l, db, tenant)(name), model.Decorate(decorators...))()
	}
}

func InventoryModelDecorator(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) model.Decorator[Model] {
	return func(m Model) Model {
		i, err := inventory.GetInventories(l, db, span, tenant)(m.Id())
		if err != nil {
			return m
		}

		es, err := model.Fold(equipable.EquipmentProvider(l, db, span, tenant)(i.Equipable().Id()), model.FixedProvider(m.GetEquipment()), FoldEquipable)()
		if err != nil {
			return CloneModel(m).SetInventory(i).Build()
		}

		return CloneModel(m).SetEquipment(es).SetInventory(i).Build()
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

func IsValidName(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(name string) (bool, error) {
	return func(name string) (bool, error) {
		m, err := regexp.MatchString("[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}", name)
		if err != nil {
			return false, err
		}
		if !m {
			return false, nil
		}

		cs, err := GetForName(l, db, tenant)(name)
		if len(cs) != 0 || err != nil {
			return false, nil
		}

		//TODO
		//bn, err := blocked_name.IsBlockedName(l, span)(name)
		//if bn {
		//	return false, err
		//}

		return true, nil
	}
}

func Create(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, eventProducer producer.Provider) func(tenant tenant.Model, input Model) (Model, error) {
	return func(tenant tenant.Model, input Model) (Model, error) {
		ok, err := IsValidName(l, db, tenant)(input.Name())
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

		var res Model
		err = db.Transaction(func(tx *gorm.DB) error {
			res, err = create(tx, tenant.Id, input.accountId, input.worldId, input.name, input.level, input.strength, input.dexterity, input.intelligence, input.luck, input.maxHp, input.maxMp, input.jobId, input.gender, input.hair, input.face, input.skinColor, input.mapId)
			if err != nil {
				l.WithError(err).Errorf("Error persisting character in database.")
				tx.Rollback()
				return err
			}

			inv, err := inventory.Create(l, tx, span, tenant)(res.id, 24)
			if err != nil {
				l.WithError(err).Errorf("Unable to create inventory for character during character creation.")
				tx.Rollback()
				return err
			}
			res = CloneModel(res).SetInventory(inv).Build()
			return nil
		})

		if err == nil {
			err = eventProducer(EnvEventTopicCharacterStatus)(createdEventProvider(tenant, res.Id(), res.WorldId(), res.Name()))
		}
		return res, err
	}
}

func Delete(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32) error {
	return func(characterId uint32) error {
		err := db.Transaction(func(tx *gorm.DB) error {
			c, err := GetById(l, tx, tenant)(characterId, InventoryModelDecorator(l, tx, span, tenant))
			if err != nil {
				return err
			}

			// delete equipment.
			err = equipment.Delete(l, tx, span, tenant)(c.equipment)
			if err != nil {
				l.WithError(err).Errorf("Unable to delete equipment for character with id [%d].", characterId)
				return err
			}

			// delete inventories.
			err = inventory.DeleteEquipableInventory(l, tx, span, tenant)(characterId, c.inventory.Equipable())
			if err != nil {
				l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
				return err
			}
			err = inventory.DeleteItemInventory(l, tx, span, tenant)(characterId, c.inventory.Useable())
			if err != nil {
				l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
				return err
			}
			err = inventory.DeleteItemInventory(l, tx, span, tenant)(characterId, c.inventory.Setup())
			if err != nil {
				l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
				return err
			}
			err = inventory.DeleteItemInventory(l, tx, span, tenant)(characterId, c.inventory.Etc())
			if err != nil {
				l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
				return err
			}
			err = inventory.DeleteItemInventory(l, tx, span, tenant)(characterId, c.inventory.Cash())
			if err != nil {
				l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
				return err
			}

			_ = inventory.GetLockRegistry().DeleteForCharacter(characterId)

			err = delete(tx, tenant.Id, characterId)
			if err != nil {
				return err
			}

			return nil
		})
		return err
	}
}

func Login(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte) error {
	return func(characterId uint32, worldId byte, channelId byte) error {
		alf := announceLogin(producer.ProviderImpl(l)(span))(tenant)(worldId, channelId)
		return model.For(byIdProvider(l, db, tenant)(characterId), alf)
	}
}

func announceLogin(provider producer.Provider) func(tenant tenant.Model) func(worldId byte, channelId byte) model.Operator[Model] {
	return func(tenant tenant.Model) func(worldId byte, channelId byte) model.Operator[Model] {
		return func(worldId byte, channelId byte) model.Operator[Model] {
			return func(c Model) error {
				return provider(EnvEventTopicCharacterStatus)(loginEventProvider(tenant, c.Id(), worldId, channelId, c.MapId()))
			}
		}
	}
}

func Logout(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte) error {
	return func(characterId uint32, worldId byte, channelId byte) error {
		alf := announceLogout(producer.ProviderImpl(l)(span))(tenant)(worldId, channelId)
		return model.For(byIdProvider(l, db, tenant)(characterId), alf)
	}
}

func announceLogout(provider producer.Provider) func(tenant tenant.Model) func(worldId byte, channelId byte) model.Operator[Model] {
	return func(tenant tenant.Model) func(worldId byte, channelId byte) model.Operator[Model] {
		return func(worldId byte, channelId byte) model.Operator[Model] {
			return func(c Model) error {
				return provider(EnvEventTopicCharacterStatus)(logoutEventProvider(tenant, c.Id(), worldId, channelId, c.MapId()))
			}
		}
	}
}

func ChangeMap(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte, mapId uint32, portalId uint32) error {
	return func(characterId uint32, worldId byte, channelId byte, mapId uint32, portalId uint32) error {
		cmf := changeMap(db)(tenant)(mapId)
		papf := positionAtPortal(l)(span)(tenant)(mapId, portalId)
		amcf := announceMapChanged(producer.ProviderImpl(l)(span))(tenant)(worldId, channelId, mapId, portalId)
		return model.For(byIdProvider(l, db, tenant)(characterId), model.ThenOperator(cmf, papf, amcf))
	}
}

func changeMap(db *gorm.DB) func(tenant tenant.Model) func(mapId uint32) model.Operator[Model] {
	return func(tenant tenant.Model) func(mapId uint32) model.Operator[Model] {
		return func(mapId uint32) model.Operator[Model] {
			return func(c Model) error {
				return dynamicUpdate(db, tenant)(SetMapId(mapId))(c)
			}
		}
	}
}

func positionAtPortal(l logrus.FieldLogger) func(span opentracing.Span) func(tenant tenant.Model) func(mapId uint32, portalId uint32) model.Operator[Model] {
	return func(span opentracing.Span) func(tenant tenant.Model) func(mapId uint32, portalId uint32) model.Operator[Model] {
		return func(tenant tenant.Model) func(mapId uint32, portalId uint32) model.Operator[Model] {
			return func(mapId uint32, portalId uint32) model.Operator[Model] {
				return func(c Model) error {
					por, err := portal.GetInMapById(l, span, tenant)(mapId, portalId)
					if err != nil {
						return err
					}
					GetTemporalRegistry().UpdatePosition(c.Id(), por.X(), por.Y())
					return nil
				}
			}
		}
	}
}

func announceMapChanged(provider producer.Provider) func(tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, portalId uint32) model.Operator[Model] {
	return func(tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, portalId uint32) model.Operator[Model] {
		return func(worldId byte, channelId byte, mapId uint32, portalId uint32) model.Operator[Model] {
			return func(c Model) error {
				return provider(EnvEventTopicCharacterStatus)(mapChangedEventProvider(tenant, c.Id(), worldId, channelId, c.MapId(), mapId, portalId))
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

func FoldMovementSummary(summary MovementSummary, e element) (MovementSummary, error) {
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

func Move(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte, mapId uint32, movement movement) error {
	return func(characterId uint32, worldId byte, channelId byte, mapId uint32, movement movement) error {
		msp := model.Fold(model.FixedProvider(movement.Elements), MovementSummaryProvider(movement.StartX, movement.StartY, GetTemporalRegistry().GetById(characterId).Stance()), FoldMovementSummary)
		err := model.For(msp, updateTemporal(characterId))
		if err != nil {
			return err
		}
		return producer.ProviderImpl(l)(span)(EnvEventTopicMovement)(move(tenant, worldId, channelId, mapId, characterId, movement))
	}
}

func updateTemporal(characterId uint32) model.Operator[MovementSummary] {
	return func(ms MovementSummary) error {
		GetTemporalRegistry().Update(characterId, ms.X, ms.Y, ms.Stance)
		return nil
	}
}
